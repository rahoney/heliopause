package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	maxCargoProjectCacheFiles = 10000
	maxCargoProjectCacheBytes = int64(200 << 20)
	cargoCacheReceipt         = "receipt.json"
	maxCargoCacheReceiptBytes = 4 << 20
)

// CargoVerifiedCache materializes only independently approved crate bytes.
// Neither Cargo's resolver cache nor project-controlled configuration is copied.
type CargoVerifiedCache struct {
	intakeRoot, evidenceRoot, cacheRoot string
	evidence                            ProjectEvidenceReader
}

func NewCargoVerifiedCache(intake, evidence, cache string, reader ProjectEvidenceReader) (*CargoVerifiedCache, error) {
	roots := []string{intake, evidence, cache}
	if reader == nil || !separateRoots(roots) {
		return nil, errors.New("cargo cache requires separate intake, Evidence and cache roots")
	}
	for _, root := range roots {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
			return nil, errors.New("cargo cache roots must be canonical absolute paths")
		}
	}
	return &CargoVerifiedCache{intake, evidence, cache, reader}, nil
}

type cargoCacheEntry struct {
	Source        string            `json:"source"`
	Crate         string            `json:"crate"`
	Version       string            `json:"version"`
	Digest        string            `json:"digest"`
	Integrity     string            `json:"integrity"`
	Run           string            `json:"run"`
	Evidence      []goCacheEvidence `json:"evidence"`
	Policy        string            `json:"policy"`
	PolicyVersion uint64            `json:"policy_version"`
}

type cargoCacheDocument struct {
	Schema         int               `json:"schema"`
	Graph          string            `json:"graph"`
	Project        string            `json:"project"`
	Policy         string            `json:"policy"`
	PolicyVersion  uint64            `json:"policy_version"`
	Controls       []cacheFileRecord `json:"controls"`
	Entries        []cargoCacheEntry `json:"entries"`
	Files          []cacheFileRecord `json:"files"`
	DependencyFree bool              `json:"dependency_free,omitempty"`
}

func (c *CargoVerifiedCache) approval(ctx context.Context, set domain.ProjectVerifiedSet) (cargoCacheDocument, error) {
	if c == nil || ctx == nil || !set.Valid() || set.Inspected().Snapshot().Source() != artifactcargo.Source() {
		return cargoCacheDocument{}, errors.New("cargo cache requires complete project ALLOW")
	}
	snapshot := set.Inspected().Snapshot()
	if len(snapshot.Dependencies()) > 4096 {
		return cargoCacheDocument{}, errors.New("cargo cache graph exceeds subject bound")
	}
	project := sha256.Sum256([]byte(snapshot.Context().Target().String()))
	doc := cargoCacheDocument{Schema: 1, Graph: snapshot.GraphDigest().String(), Project: hex.EncodeToString(project[:]), Policy: set.Decision().PolicyID(), PolicyVersion: set.Decision().Version(), DependencyFree: snapshot.DependencyFree()}
	for _, control := range snapshot.ControlDigests() {
		doc.Controls = append(doc.Controls, cacheFileRecord{Path: control.Name(), SHA256: control.Digest().String()})
	}
	for _, inspection := range set.Inspected().Inspections() {
		a := inspection.Artifact()
		integrity, _ := a.DeclaredIntegrity()
		checksum, err := artifactcargo.ParseIntegrity(integrity)
		if err != nil || checksum != a.Digest().String() || a.Identity().Source() != artifactcargo.Source() || a.Identity().Variant() != "crate" {
			return cargoCacheDocument{}, errors.New("cargo cache approval subject is invalid")
		}
		records, err := projectCacheEvidence(ctx, c.evidence, inspection)
		if err != nil {
			return cargoCacheDocument{}, err
		}
		doc.Entries = append(doc.Entries, cargoCacheEntry{a.Identity().Source().String(), a.Identity().Name(), a.Identity().Version(), a.Digest().String(), integrity, inspection.RunID().String(), records, inspection.PolicyDecision().PolicyID(), inspection.PolicyDecision().Version()})
	}
	return doc, nil
}

// The checksum document is controller-generated from rehashed approved files.
// An archive cannot supply this reserved directory-source authority.
func cargoVendorChecksum(files map[string][]byte, integrity string) ([]byte, error) {
	checksum, err := artifactcargo.ParseIntegrity(integrity)
	if err != nil {
		return nil, err
	}
	hashes := map[string]string{}
	for name, body := range files {
		folded := strings.ToLower(name)
		if folded == ".cargo-checksum.json" || strings.HasPrefix(folded, ".cargo-checksum.json/") {
			return nil, errors.New("cargo archive collides with controller checksum metadata")
		}
		hash := sha256.Sum256(body)
		hashes[name] = hex.EncodeToString(hash[:])
	}
	body, err := json.Marshal(struct {
		Files   map[string]string `json:"files"`
		Package string            `json:"package"`
	}{hashes, checksum})
	if err != nil || len(body) > maxCargoCacheReceiptBytes {
		return nil, errors.New("cargo vendor checksum exceeds bound")
	}
	return body, nil
}

func (c *CargoVerifiedCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (staged domain.StagedProjectSet, resultErr error) {
	doc, err := c.approval(ctx, set)
	if err != nil {
		return staged, err
	}
	if err := ctx.Err(); err != nil {
		return staged, err
	}
	if err := ensureTrustedRoot(c.cacheRoot); err != nil {
		return staged, err
	}
	run, err := domain.NewRunID()
	if err != nil {
		return staged, err
	}
	temporary, err := os.MkdirTemp(c.cacheRoot, ".cargo-stage-")
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
	vendor := filepath.Join(temporary, "vendor")
	if err := os.Mkdir(vendor, 0o755); err != nil {
		return staged, err
	}
	var totalBytes int64
	totalFiles := 0
	for _, inspection := range set.Inspected().Inspections() {
		if err := ctx.Err(); err != nil {
			return staged, err
		}
		a := inspection.Artifact()
		if a.ContentHandle() != "intake:"+inspection.RunID().String()+":crate" {
			return staged, errors.New("cargo cache intake Run binding differs")
		}
		body, err := artifactcargo.ReadIntake(c.intakeRoot, a)
		if err != nil {
			return staged, err
		}
		files, err := artifactcargo.ArchiveFiles(body, a.Identity())
		if err != nil {
			return staged, err
		}
		integrity, _ := a.DeclaredIntegrity()
		checksum, err := cargoVendorChecksum(files, integrity)
		if err != nil {
			return staged, err
		}
		totalFiles += len(files) + 1
		totalBytes += int64(len(checksum))
		for _, body := range files {
			totalBytes += int64(len(body))
		}
		if totalFiles > maxCargoProjectCacheFiles || totalBytes > maxCargoProjectCacheBytes {
			return staged, fmt.Errorf("cargo cache exceeds content limits: files=%d file_limit=%d bytes=%d byte_limit=%d", totalFiles, maxCargoProjectCacheFiles, totalBytes, maxCargoProjectCacheBytes)
		}
		crate := filepath.Join(vendor, a.Identity().Name()+"-"+a.Identity().Version())
		if err := os.Mkdir(crate, 0o755); err != nil {
			return staged, errors.New("create exact Cargo vendor directory")
		}
		var names []string
		for name := range files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := ctx.Err(); err != nil {
				return staged, err
			}
			file := filepath.Join(crate, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				return staged, err
			}
			if err := writeRecord(filepath.Dir(file), filepath.Base(file), files[name]); err != nil {
				return staged, err
			}
		}
		if err := writeRecord(crate, ".cargo-checksum.json", checksum); err != nil {
			return staged, err
		}
	}
	if err := sealGoCacheTree(vendor); err != nil {
		return staged, err
	}
	doc.Files, err = projectCacheInventory(ctx, vendor, maxCargoProjectCacheFiles, maxCargoProjectCacheBytes, artifactcargo.MaxCrateFileBytes, true)
	if err != nil {
		return staged, err
	}
	receipt, err := json.Marshal(doc)
	if err != nil || len(receipt) > maxCargoCacheReceiptBytes {
		return staged, errors.New("cargo cache receipt exceeds bound")
	}
	hash := sha256.Sum256(receipt)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	staged, err = domain.NewStagedProjectSet(set, "project-cache:"+run.String(), digest)
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	if err := writeRecord(temporary, cargoCacheReceipt, receipt); err != nil {
		return staged, err
	}
	if err := syncTree(temporary); err != nil {
		return staged, err
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

func (c *CargoVerifiedCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (string, error) {
	if c == nil || ctx == nil || !staged.Valid() {
		return "", errors.New("valid approved Cargo cache is required")
	}
	doc, err := c.approval(ctx, staged.Set())
	if err != nil {
		return "", err
	}
	return c.openDocument(ctx, staged.ContentHandle(), staged.Digest(), doc)
}

func (c *CargoVerifiedCache) openDocument(ctx context.Context, handle string, digest domain.ContentDigest, expected cargoCacheDocument) (string, error) {
	if err := c.verifyRecordedApproval(ctx, expected); err != nil {
		return "", err
	}
	id, ok := strings.CutPrefix(handle, "project-cache:")
	if !ok {
		return "", errors.New("cargo cache handle is invalid")
	}
	if _, err := domain.ParseRunID(id); err != nil {
		return "", errors.New("cargo cache handle is invalid")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root := filepath.Join(c.cacheRoot, id)
	if err := trustedExistingDirectory(root); err != nil {
		return "", err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return "", err
	}
	defer r.Close()
	info, err := r.Lstat(cargoCacheReceipt)
	if err != nil || !info.Mode().IsRegular() || !pypiSingleLink(info) || info.Size() <= 0 || info.Size() > maxCargoCacheReceiptBytes {
		return "", errors.New("cargo cache receipt is invalid")
	}
	f, err := r.OpenFile(cargoCacheReceipt, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	opened, statErr := f.Stat()
	if statErr != nil || !os.SameFile(info, opened) {
		_ = f.Close()
		return "", errors.New("cargo cache receipt identity changed")
	}
	body, readErr := io.ReadAll(io.LimitReader(f, info.Size()+1))
	after, afterErr := f.Stat()
	closeErr := f.Close()
	hash := sha256.Sum256(body)
	if readErr != nil || closeErr != nil || afterErr != nil || !os.SameFile(info, after) || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) || int64(len(body)) != info.Size() || hex.EncodeToString(hash[:]) != digest.String() {
		return "", errors.New("cargo cache receipt changed")
	}
	var actual cargoCacheDocument
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&actual) != nil {
		return "", errors.New("cargo cache receipt grammar is invalid")
	}
	expected.Files, err = projectCacheInventory(ctx, filepath.Join(root, "vendor"), maxCargoProjectCacheFiles, maxCargoProjectCacheBytes, artifactcargo.MaxCrateFileBytes, true)
	if err != nil {
		return "", err
	}
	if expected.DependencyFree && (len(expected.Files) != 1 || expected.Files[0].Path != "." || expected.Files[0].Kind != "directory") {
		return "", errors.New("dependency-free cargo cache contains extra content")
	}
	want, err := json.Marshal(expected)
	if err != nil || !bytes.Equal(want, body) {
		return "", errors.New("cargo cache content or retained approval changed")
	}
	return filepath.Join(root, "vendor"), nil
}

func (c *CargoVerifiedCache) verifyRecordedApproval(ctx context.Context, doc cargoCacheDocument) error {
	if (len(doc.Entries) == 0) != doc.DependencyFree || len(doc.Entries) > 4096 {
		return errors.New("cargo retained approval coverage is invalid")
	}
	seen := map[domain.ResolvedArtifactIdentity]bool{}
	for _, entry := range doc.Entries {
		identity, err := domain.NewResolvedArtifactIdentity(artifactcargo.Source(), entry.Crate, entry.Version, "crate")
		if _, parseErr := artifactcargo.ParseReference(entry.Crate + "@" + entry.Version); err != nil || parseErr != nil || entry.Source != artifactcargo.Source().String() || seen[identity] || entry.Policy == "" || entry.PolicyVersion == 0 || len(entry.Evidence) == 0 || len(entry.Evidence) > 4096 {
			return errors.New("cargo retained entry approval is invalid")
		}
		seen[identity] = true
		checksum, err := artifactcargo.ParseIntegrity(entry.Integrity)
		if err != nil || checksum != entry.Digest {
			return errors.New("cargo retained entry checksum is invalid")
		}
		run, err := domain.ParseRunID(entry.Run)
		if err != nil {
			return err
		}
		digest, err := domain.NewSHA256Digest(entry.Digest)
		if err != nil {
			return err
		}
		for _, record := range entry.Evidence {
			id, err := domain.NewEvidenceID(record.ID)
			if err != nil {
				return err
			}
			ref, err := domain.NewEvidenceReference(id, "evidence:"+run.String()+":"+id.String())
			if err != nil {
				return err
			}
			_, hash, err := c.evidence.ReadReference(ctx, run, ref, identity, digest)
			if err != nil || hash.String() != record.SHA256 {
				return errors.New("cargo retained Evidence is changed or unavailable")
			}
		}
	}
	return nil
}
