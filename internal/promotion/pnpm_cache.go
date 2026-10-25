package promotion

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const pnpmCacheReceiptLimit = 4 << 20
const pnpmCacheArchiveName = "node_modules.tar"
const pnpmCacheDataLimit = artifactnpm.PNPMLayoutArchiveLimit + 2*artifactnpm.ProjectControlLimit

// PNPMLayoutProvider supplies an immutable completed layout for exact acquired
// runs. It confers no Policy, recorded Evidence or publication authority.
type PNPMLayoutProvider interface {
	PreparedLayoutFor(domain.ProjectDependencySnapshot, map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.PNPMInspectionLayout, error)
}

// PNPMVerifiedCache stages data files only. A native tree remains inside one
// bounded tar; this component never extracts links or executes an artifact.
type PNPMVerifiedCache struct {
	intakeRoot, evidenceRoot, cacheRoot string
	frozen                              artifactnpm.PNPMInspectionContext
	evidence                            ProjectEvidenceReader
	layout                              PNPMLayoutProvider
	recordBuilder                       ProjectRecordBuilder
	operation                           domain.OperationID
	// An instance-local diagnostic can abort before the mandatory real sync.
	beforeTreeSync func(string) error
	// Diagnostic abort only; mandatory materialization readback is never skipped.
	beforePrepareSync func(*PNPMPreparedProject) error
}

func NewPNPMVerifiedCache(intake, evidence, cache string, frozen artifactnpm.PNPMInspectionContext, reader ProjectEvidenceReader, layout PNPMLayoutProvider, records ProjectRecordBuilder, operation domain.OperationID) (*PNPMVerifiedCache, error) {
	roots := []string{intake, evidence, cache}
	if !frozen.Valid() || reader == nil || layout == nil || records == nil || operation.String() == "" || !separateRoots(append(roots, frozen.Snapshot().Context().Target().String())) {
		return nil, errors.New("pnpm cache requires frozen context, actual operation, records and separate trusted roots")
	}
	for _, root := range roots {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
			return nil, errors.New("pnpm cache roots must be canonical absolute paths")
		}
	}
	return &PNPMVerifiedCache{intakeRoot: intake, evidenceRoot: evidence, cacheRoot: cache, frozen: frozen, evidence: reader, layout: layout, recordBuilder: records, operation: operation}, nil
}

func pnpmCacheHash(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

func (c *PNPMVerifiedCache) verifySources(ctx context.Context, input artifactnpm.PNPMInspectionInput) (resultErr error) {
	return verifyNPMProjectCacheSources(ctx, c.intakeRoot, input.Artifacts())
}

func verifyNPMProjectCacheSources(ctx context.Context, intakeRoot string, subjects []domain.AcquiredArtifact) (resultErr error) {
	if len(subjects) == 0 {
		return nil
	}
	if err := trustedExistingDirectory(intakeRoot); err != nil {
		return err
	}
	root, err := os.OpenRoot(intakeRoot)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	for _, subject := range subjects {
		if _, err := artifactnpm.ReadPNPMIntakeArchive(ctx, root, subject); err != nil {
			return err
		}
	}
	return nil
}

func renderPNPMCacheLayout(ctx context.Context, layout artifactnpm.PNPMInspectionLayout) ([]byte, error) {
	return renderNPMProjectCacheLayout(ctx, layout)
}

type npmProjectCacheLayout interface {
	Digest() domain.ContentDigest
	Entries() []artifactnpm.PNPMLayoutEntry
	ReadFile(string) ([]byte, error)
}

func renderNPMProjectCacheLayout(ctx context.Context, layout npmProjectCacheLayout) ([]byte, error) {
	if ctx == nil || layout.Digest().String() == "" {
		return nil, errors.New("pnpm cache layout is unavailable")
	}
	entries := layout.Entries()
	slices.SortFunc(entries, func(a, b artifactnpm.PNPMLayoutEntry) int { return strings.Compare(a.Name, b.Name) })
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, errors.Join(err, context.Cause(ctx))
		}
		var body []byte
		if entry.Kind == tar.TypeReg {
			var err error
			body, err = layout.ReadFile(entry.Name)
			if err != nil || int64(len(body)) != entry.Size || pnpmCacheHash(body) != entry.Digest.String() {
				return nil, errors.Join(errors.New("pnpm bound layout bytes changed"), err)
			}
		}
		if buffer.Len() > artifactnpm.PNPMLayoutArchiveLimit-len(body) {
			return nil, errors.New("pnpm cache tar exceeds its fixed bound")
		}
		header := &tar.Header{Name: entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Uid: entry.UID, Gid: entry.GID, Size: entry.Size, Linkname: entry.Link}
		if err := writer.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := writer.Write(body); err != nil {
			return nil, err
		}
		if buffer.Len() > artifactnpm.PNPMLayoutArchiveLimit {
			return nil, errors.New("pnpm cache tar exceeds its fixed bound")
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if buffer.Len() > artifactnpm.PNPMLayoutArchiveLimit {
		return nil, errors.New("pnpm cache tar exceeds its fixed bound")
	}
	// The existing parser independently checks the emitted envelope and links.
	if _, err := artifactnpm.ParsePNPMLayoutArchive(ctx, buffer.Bytes()); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (c *PNPMVerifiedCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (staged domain.StagedProjectSet, resultErr error) {
	doc, input, err := c.approval(ctx, set)
	if err != nil {
		return staged, err
	}
	var archive []byte
	if !doc.DependencyFree {
		acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
		for _, inspection := range set.Inspected().Inspections() {
			acquired[inspection.Node()] = inspection.Artifact()
		}
		layout, err := c.layout.PreparedLayoutFor(set.Inspected().Snapshot(), acquired)
		if err != nil || !layout.MatchesInput(input) || layout.Digest().String() != doc.Layout {
			return staged, errors.Join(errors.New("pnpm completed layout differs from recorded approval"), err)
		}
		archive, err = renderPNPMCacheLayout(ctx, layout)
		if err != nil {
			return staged, err
		}
	}
	if err := c.verifySources(ctx, input); err != nil {
		return staged, err
	}
	material, err := c.projectRecords(ctx, set, doc)
	if err != nil {
		return staged, err
	}
	doc.Manifest = material.manifest.String()
	if err := ctx.Err(); err != nil {
		return staged, errors.Join(err, context.Cause(ctx))
	}
	return publishNPMProjectCache(ctx, set, doc, c.cacheRoot, c.frozen.Controls(), archive, material, c.beforeTreeSync, npmProjectCachePNPM)
}

type npmProjectCacheClient uint8

const (
	npmProjectCachePNPM npmProjectCacheClient = iota + 1
	npmProjectCacheBun
)

type npmProjectCacheSpec struct{ prefix, stagePrefix, provenanceName string }

func npmProjectCacheIdentity(client npmProjectCacheClient) (npmProjectCacheSpec, bool) {
	switch client {
	case npmProjectCachePNPM:
		return npmProjectCacheSpec{"pnpm", ".pnpm-stage-", "pnpm-provenance.json"}, true
	case npmProjectCacheBun:
		return npmProjectCacheSpec{"bun", ".bun-stage-", "bun-provenance.json"}, true
	default:
		return npmProjectCacheSpec{}, false
	}
}

func publishNPMProjectCache(ctx context.Context, set domain.ProjectVerifiedSet, doc pnpmCacheDocument, cacheRoot string, controls []domain.ProjectControlFile, archive []byte, material pnpmProjectRecords, beforeTreeSync func(string) error, client npmProjectCacheClient) (staged domain.StagedProjectSet, resultErr error) {
	spec, known := npmProjectCacheIdentity(client)
	if !known {
		return staged, errors.New("unknown project cache client")
	}
	var err error
	if err := ensureTrustedRoot(cacheRoot); err != nil {
		return staged, err
	}
	run, err := domain.NewRunID()
	if err != nil {
		return staged, err
	}
	temporary, err := os.MkdirTemp(cacheRoot, spec.stagePrefix)
	if err != nil {
		return staged, err
	}
	final := filepath.Join(cacheRoot, run.String())
	published := false
	defer func() {
		if !published {
			resultErr = errors.Join(resultErr, os.RemoveAll(temporary))
		}
		if resultErr != nil {
			staged = domain.StagedProjectSet{}
		}
	}()
	data := filepath.Join(temporary, "data")
	if err := os.Mkdir(data, 0o700); err != nil {
		return staged, err
	}
	for _, control := range controls {
		if err := writePrivateRecord(data, control.Name(), control.Body()); err != nil {
			return staged, err
		}
	}
	if !doc.DependencyFree {
		if err := writePrivateRecord(data, pnpmCacheArchiveName, archive); err != nil {
			return staged, err
		}
	}
	if err := sealGoCacheTree(data); err != nil {
		return staged, err
	}
	records := filepath.Join(temporary, "records")
	if err := os.Mkdir(records, 0o700); err != nil {
		return staged, err
	}
	for _, record := range []struct {
		name string
		body []byte
	}{
		{"manifest.json", material.document}, {"sbom.cdx.json", material.sbom}, {spec.provenanceName, material.provenance},
	} {
		if err := writePrivateRecord(records, record.name, record.body); err != nil {
			return staged, err
		}
	}
	if err := sealGoCacheTree(records); err != nil {
		return staged, err
	}
	doc.Records, err = projectCacheInventory(ctx, records, 4, pnpmProjectRecordsLimit, pnpmProjectRecordLimit, true)
	if err != nil {
		return staged, err
	}
	doc.Files, err = projectCacheInventory(ctx, data, 4, pnpmCacheDataLimit, artifactnpm.PNPMLayoutArchiveLimit, true)
	if err != nil {
		return staged, err
	}
	receipt, err := json.Marshal(doc)
	if err != nil || len(receipt) > pnpmCacheReceiptLimit {
		return staged, errors.Join(errors.New("pnpm cache receipt exceeds its fixed bound"), err)
	}
	digest, err := domain.NewSHA256Digest(pnpmCacheHash(receipt))
	if err != nil {
		return staged, err
	}
	staged, err = domain.NewStagedProjectSet(set, "project-cache:"+run.String(), digest)
	if err != nil {
		return staged, err
	}
	if err := writePrivateRecord(temporary, "receipt.json", receipt); err != nil {
		return staged, err
	}
	if err := os.Chmod(filepath.Join(temporary, "receipt.json"), 0o444); err != nil {
		return staged, err
	}
	if beforeTreeSync != nil {
		if err := beforeTreeSync(temporary); err != nil {
			return staged, err
		}
	}
	if err := syncTree(temporary); err != nil {
		return staged, err
	}
	if err := ctx.Err(); err != nil {
		return staged, errors.Join(err, context.Cause(ctx))
	}
	if err := renameNoReplace(temporary, final); err != nil {
		return staged, err
	}
	published = true
	if err := syncDirectory(cacheRoot); err != nil {
		return staged, errors.Join(err, os.RemoveAll(final))
	}
	return staged, nil

}

// OpenProjectCache rechecks current recorded approval and every sealed data
// byte. The returned directory contains data, not a Host installation.
func (c *PNPMVerifiedCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (directory string, resultErr error) {
	if !staged.Valid() {
		return "", errors.New("pnpm staged project is required")
	}
	expected, _, err := c.approval(ctx, staged.Set())
	if err != nil {
		return "", err
	}
	material, err := c.projectRecords(ctx, staged.Set(), expected)
	if err != nil {
		return "", err
	}
	expected.Manifest = material.manifest.String()
	return openPNPMApprovedCache(ctx, c.cacheRoot, staged.ContentHandle(), staged.Digest(), expected, &material)
}

// Retained callers supply controller-owned approval; current callers additionally
// regenerate their records from the exact in-memory approved operation.
func openPNPMApprovedCache(ctx context.Context, cacheRoot, handle string, digest domain.ContentDigest, expected pnpmCacheDocument, material *pnpmProjectRecords) (directory string, resultErr error) {
	return openNPMApprovedProjectCache(ctx, cacheRoot, handle, digest, expected, material, npmProjectCachePNPM)
}

func openNPMApprovedProjectCache(ctx context.Context, cacheRoot, handle string, digest domain.ContentDigest, expected pnpmCacheDocument, material *pnpmProjectRecords, client npmProjectCacheClient) (directory string, resultErr error) {
	spec, known := npmProjectCacheIdentity(client)
	if !known {
		return "", errors.New("unknown project cache client")
	}

	id, ok := strings.CutPrefix(handle, "project-cache:")
	if !ok {
		return "", errors.New("pnpm cache handle is invalid")
	}
	if _, err := domain.ParseRunID(id); err != nil {
		return "", err
	}
	rootName := filepath.Join(cacheRoot, id)
	if err := trustedExistingDirectory(rootName); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(rootName)
	if err != nil {
		return "", err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil {
			directory = ""
		}
	}()
	receipt, info, err := readGoTransactionControl(root, "receipt.json")
	if err != nil || info == nil || !pypiSingleLink(info) || info.Mode().Perm() != 0o444 || len(receipt) > pnpmCacheReceiptLimit || pnpmCacheHash(receipt) != digest.String() {
		return "", errors.Join(errors.New("pnpm cache receipt changed or is unavailable"), err)
	}
	var actual pnpmCacheDocument
	decoder := json.NewDecoder(bytes.NewReader(receipt))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&actual); err != nil {
		return "", err
	}
	expected.Files, err = projectCacheInventory(ctx, filepath.Join(rootName, "data"), 4, pnpmCacheDataLimit, artifactnpm.PNPMLayoutArchiveLimit, true)
	if err != nil {
		return "", err
	}
	expected.Records, err = projectCacheInventory(ctx, filepath.Join(rootName, "records"), 4, pnpmProjectRecordsLimit, pnpmProjectRecordLimit, true)
	if err != nil {
		return "", err
	}
	if material != nil {
		for _, record := range []struct {
			name string
			body []byte
		}{
			{"manifest.json", material.document}, {"sbom.cdx.json", material.sbom}, {spec.provenanceName, material.provenance},
		} {
			found := false
			for _, file := range expected.Records {
				if file.Path == record.name && file.Kind == "file" && file.SHA256 == pnpmCacheHash(record.body) && file.Size == int64(len(record.body)) {
					found = true
				}
			}
			if !found {
				return "", errors.New("pnpm cached records differ from current operation approval")
			}
		}
	}

	want, err := json.Marshal(expected)
	if err != nil || !bytes.Equal(want, receipt) {
		return "", errors.Join(errors.New("pnpm cache data or recorded approval changed"), err)
	}
	entries, err := os.ReadDir(rootName)
	if err != nil || len(entries) != 3 || entries[0].Name() != "data" || entries[1].Name() != "receipt.json" || entries[2].Name() != "records" {
		return "", errors.Join(errors.New("pnpm cache contains a foreign member"), err)
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(err, context.Cause(ctx))
	}
	return filepath.Join(rootName, "data"), nil
}
