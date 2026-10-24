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
	// An instance-local diagnostic can abort before the mandatory real sync.
	beforeTreeSync func(string) error
}

func NewPNPMVerifiedCache(intake, evidence, cache string, frozen artifactnpm.PNPMInspectionContext, reader ProjectEvidenceReader, layout PNPMLayoutProvider) (*PNPMVerifiedCache, error) {
	roots := []string{intake, evidence, cache}
	if !frozen.Valid() || reader == nil || layout == nil || !separateRoots(append(roots, frozen.Snapshot().Context().Target().String())) {
		return nil, errors.New("pnpm cache requires frozen context and separate trusted roots")
	}
	for _, root := range roots {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
			return nil, errors.New("pnpm cache roots must be canonical absolute paths")
		}
	}
	return &PNPMVerifiedCache{intakeRoot: intake, evidenceRoot: evidence, cacheRoot: cache, frozen: frozen, evidence: reader, layout: layout}, nil
}

func pnpmCacheHash(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

func (c *PNPMVerifiedCache) verifySources(ctx context.Context, input artifactnpm.PNPMInspectionInput) (resultErr error) {
	if len(input.Artifacts()) == 0 {
		return nil
	}
	if err := trustedExistingDirectory(c.intakeRoot); err != nil {
		return err
	}
	root, err := os.OpenRoot(c.intakeRoot)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	for _, subject := range input.Artifacts() {
		if _, err := artifactnpm.ReadPNPMIntakeArchive(ctx, root, subject); err != nil {
			return err
		}
	}
	return nil
}

func renderPNPMCacheLayout(ctx context.Context, layout artifactnpm.PNPMInspectionLayout) ([]byte, error) {
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
	if err := ctx.Err(); err != nil {
		return staged, errors.Join(err, context.Cause(ctx))
	}
	if err := ensureTrustedRoot(c.cacheRoot); err != nil {
		return staged, err
	}
	run, err := domain.NewRunID()
	if err != nil {
		return staged, err
	}
	temporary, err := os.MkdirTemp(c.cacheRoot, ".pnpm-stage-")
	if err != nil {
		return staged, err
	}
	final := filepath.Join(c.cacheRoot, run.String())
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
	for _, control := range c.frozen.Controls() {
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
	if c.beforeTreeSync != nil {
		if err := c.beforeTreeSync(temporary); err != nil {
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
	if err := syncDirectory(c.cacheRoot); err != nil {
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
	id, ok := strings.CutPrefix(staged.ContentHandle(), "project-cache:")
	if !ok {
		return "", errors.New("pnpm cache handle is invalid")
	}
	if _, err := domain.ParseRunID(id); err != nil {
		return "", err
	}
	rootName := filepath.Join(c.cacheRoot, id)
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
	if err != nil || info == nil || !pypiSingleLink(info) || info.Mode().Perm() != 0o444 || len(receipt) > pnpmCacheReceiptLimit || pnpmCacheHash(receipt) != staged.Digest().String() {
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
	want, err := json.Marshal(expected)
	if err != nil || !bytes.Equal(want, receipt) {
		return "", errors.Join(errors.New("pnpm cache data or recorded approval changed"), err)
	}
	entries, err := os.ReadDir(rootName)
	if err != nil || len(entries) != 2 || entries[0].Name() != "data" || entries[1].Name() != "receipt.json" {
		return "", errors.Join(errors.New("pnpm cache contains a foreign member"), err)
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(err, context.Cause(ctx))
	}
	return filepath.Join(rootName, "data"), nil
}
