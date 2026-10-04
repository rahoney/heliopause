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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	maxGoProjectCacheBytes int64 = 512 << 20
	maxGoProjectCacheFiles       = 20000
	goCacheReceipt               = "receipt.json"
)

// GoVerifiedCache has separate intake, Evidence and cache roots. Materialized
// files come only from rehashed ALLOW subjects, never an ambient/resolver cache.
type GoEvidenceReader interface {
	ReadReference(context.Context, domain.RunID, domain.EvidenceReference, domain.ResolvedArtifactIdentity, domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error)
}

type GoVerifiedCache struct {
	intakeRoot, evidenceRoot, cacheRoot string
	evidence                            GoEvidenceReader
}

func NewGoVerifiedCache(intakeRoot, evidenceRoot, cacheRoot string, evidence GoEvidenceReader) (*GoVerifiedCache, error) {
	if evidence == nil {
		return nil, errors.New("go cache requires recorded Evidence reader")
	}
	roots := []string{intakeRoot, evidenceRoot, cacheRoot}
	for _, root := range roots {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
			return nil, errors.New("go verified cache requires canonical absolute roots")
		}
	}
	if !separateRoots(roots) {
		return nil, errors.New("go verified cache roots overlap")
	}
	return &GoVerifiedCache{intakeRoot, evidenceRoot, cacheRoot, evidence}, nil
}

type goCacheFile struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type goCacheEntry struct {
	Source        string            `json:"source"`
	Module        string            `json:"module"`
	Version       string            `json:"version"`
	Digest        string            `json:"digest"`
	Integrity     string            `json:"integrity"`
	Run           string            `json:"run"`
	Evidence      []goCacheEvidence `json:"evidence"`
	Policy        string            `json:"policy"`
	PolicyVersion uint64            `json:"policy_version"`
}
type goCacheEvidence struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}
type goCacheDocument struct {
	Schema         int            `json:"schema"`
	Graph          string         `json:"graph"`
	Project        string         `json:"project"`
	Policy         string         `json:"policy"`
	PolicyVersion  uint64         `json:"policy_version"`
	Controls       []goCacheFile  `json:"controls"`
	Entries        []goCacheEntry `json:"entries"`
	Files          []goCacheFile  `json:"files"`
	DependencyFree bool           `json:"dependency_free,omitempty"`
}

func (c *GoVerifiedCache) goCacheApproval(ctx context.Context, set domain.ProjectVerifiedSet) (goCacheDocument, error) {
	project := sha256.Sum256([]byte(set.Inspected().Snapshot().Context().Target().String()))
	doc := goCacheDocument{Schema: 1, Graph: set.Inspected().Snapshot().GraphDigest().String(), Project: hex.EncodeToString(project[:]), Policy: set.Decision().PolicyID(), PolicyVersion: set.Decision().Version()}
	doc.DependencyFree = set.Inspected().Snapshot().DependencyFree()
	for _, control := range set.Inspected().Snapshot().ControlDigests() {
		doc.Controls = append(doc.Controls, goCacheFile{Path: control.Name(), SHA256: control.Digest().String()})
	}
	for _, inspection := range set.Inspected().Inspections() {
		a := inspection.Artifact()
		p := inspection.PolicyDecision()
		integrity, _ := a.DeclaredIntegrity()
		e := goCacheEntry{Source: a.Identity().Source().String(), Module: a.Identity().Name(), Version: a.Identity().Version(), Digest: a.Digest().String(), Integrity: integrity, Run: inspection.RunID().String(), Policy: p.PolicyID(), PolicyVersion: p.Version()}
		covered := map[domain.CheckID]bool{}
		for _, ref := range inspection.Evidence() {
			item, digest, err := c.evidence.ReadReference(ctx, inspection.RunID(), ref, a.Identity(), a.Digest())
			if err != nil {
				return goCacheDocument{}, err
			}
			covered[item.CheckID()] = true
			e.Evidence = append(e.Evidence, goCacheEvidence{ref.ID().String(), digest.String()})
		}
		for _, check := range inspection.Checks() {
			if check.Required() && !covered[check.ID()] {
				return goCacheDocument{}, errors.New("go cache approval has missing required Evidence")
			}
		}
		doc.Entries = append(doc.Entries, e)
	}
	return doc, nil
}

func (c *GoVerifiedCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (staged domain.StagedProjectSet, resultErr error) {
	if ctx == nil || c == nil || !set.Valid() || set.Inspected().Snapshot().Source() != artifactgo.Source() {
		return staged, errors.New("go cache requires complete approved project")
	}
	if err := ctx.Err(); err != nil {
		return staged, err
	}
	if err := ensureTrustedRoot(c.cacheRoot); err != nil {
		return staged, err
	}
	id, err := domain.NewRunID()
	if err != nil {
		return staged, err
	}
	final := filepath.Join(c.cacheRoot, id.String())
	temporary, err := os.MkdirTemp(c.cacheRoot, ".go-stage-")
	if err != nil {
		return staged, errors.New("create private Go verified cache tree")
	}
	published := false
	defer func() {
		if !published {
			if err := os.RemoveAll(temporary); err != nil {
				resultErr = errors.Join(resultErr, errors.New("dispose incomplete Go cache"))
			}
		}
		if resultErr != nil {
			staged = domain.StagedProjectSet{}
		}
	}()
	moduleRoot := filepath.Join(temporary, "modcache")
	if err := os.Mkdir(moduleRoot, 0o755); err != nil {
		return staged, err
	}
	var totalBytes uint64
	var totalFiles int
	for _, inspection := range set.Inspected().Inspections() {
		if err := ctx.Err(); err != nil {
			return staged, err
		}
		a := inspection.Artifact()
		if a.ContentHandle() != "intake:"+inspection.RunID().String()+":module" {
			return staged, errors.New("go cache intake Run binding differs")
		}
		bundle, err := artifactgo.ReadIntake(c.intakeRoot, a)
		if err != nil {
			return staged, err
		}
		reader, err := bundle.ZipReader()
		if err != nil {
			return staged, err
		}
		entries, err := artifactgo.CheckedFiles(reader, a.Identity())
		if err != nil {
			return staged, err
		}
		// Bound the complete next expansion before writing or extracting it.
		totalBytes += uint64(len(bundle.Mod()) + len(bundle.Zip()) + 512)
		totalFiles += len(entries) + 4
		for _, entry := range entries {
			totalBytes += entry.UncompressedSize64
		}
		if totalBytes > uint64(maxGoProjectCacheBytes) || totalFiles > maxGoProjectCacheFiles {
			return staged, fmt.Errorf("go project cache exceeds bounded content limits: bytes=%d byte_limit=%d files=%d file_limit=%d", totalBytes, maxGoProjectCacheBytes, totalFiles, maxGoProjectCacheFiles)
		}
		integrity, ok := a.DeclaredIntegrity()
		if !ok {
			return staged, errors.New("go cache subject lacks frozen integrity")
		}
		if err := artifactgo.MaterializeModule(moduleRoot, bundle, a.Identity(), integrity); err != nil {
			return staged, err
		}
	}
	files, err := goCacheInventory(ctx, moduleRoot)
	if err != nil {
		return staged, err
	}
	doc, err := c.goCacheApproval(ctx, set)
	if err != nil {
		return staged, err
	}
	doc.Files = files
	body, err := json.Marshal(doc)
	if err != nil {
		return staged, err
	}
	if len(body) > 4<<20 {
		return staged, errors.New("go cache receipt exceeds bound")
	}
	if err := writeRecord(temporary, goCacheReceipt, body); err != nil {
		return staged, err
	}
	if err := sealGoCacheTree(moduleRoot); err != nil {
		return staged, err
	}
	if err := syncTree(temporary); err != nil {
		return staged, err
	}
	if err := renameNoReplace(temporary, final); err != nil {
		return staged, errors.New("publish Go verified cache")
	}
	published = true
	if err := syncDirectory(c.cacheRoot); err != nil {
		return staged, errors.Join(errors.New("go cache publication sync failed"), os.RemoveAll(final))
	}
	hash := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		return staged, err
	}
	return domain.NewStagedProjectSet(set, "project-cache:"+id.String(), digest)
}

// OpenProjectCache rechecks the trusted receipt binding and every sealed file
// before returning the infrastructure-selected read-only mount source.
func (c *GoVerifiedCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (string, error) {
	if ctx == nil || c == nil || !staged.Valid() {
		return "", errors.New("valid staged Go project cache is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	expected, err := c.goCacheApproval(ctx, staged.Set())
	if err != nil {
		return "", err
	}
	return c.openProjectCacheDocument(ctx, staged.ContentHandle(), staged.Digest(), expected)
}

// The expected document is either derived from a live typed approval or read
// from the separate controller-owned project receipt, never the project marker.
func (c *GoVerifiedCache) openProjectCacheDocument(ctx context.Context, handle string, digest domain.ContentDigest, expected goCacheDocument) (string, error) {
	id, ok := strings.CutPrefix(handle, "project-cache:")
	if !ok {
		return "", errors.New("invalid Go cache handle")
	}
	if _, err := domain.ParseRunID(id); err != nil {
		return "", errors.New("invalid Go cache handle")
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
	info, err := r.Lstat(goCacheReceipt)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 4<<20 {
		return "", errors.New("go cache receipt is not bounded regular content")
	}
	f, err := r.OpenFile(goCacheReceipt, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	opened, statErr := f.Stat()
	if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		_ = f.Close()
		return "", errors.New("go cache receipt identity changed")
	}
	body, readErr := io.ReadAll(io.LimitReader(f, info.Size()+1))
	closeErr := f.Close()
	hash := sha256.Sum256(body)
	if readErr != nil || closeErr != nil || int64(len(body)) != info.Size() || hex.EncodeToString(hash[:]) != digest.String() {
		return "", errors.New("go cache receipt changed")
	}
	var doc goCacheDocument
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return "", errors.New("go cache receipt is invalid")
	}
	files, err := goCacheInventory(ctx, filepath.Join(root, "modcache"))
	if err != nil {
		return "", err
	}
	if expected.DependencyFree && (len(files) != 1 || files[0].Path != "." || files[0].Kind != "directory") {
		return "", errors.New("dependency-free Go cache contains undeclared content")
	}
	expected.Files = files
	want, err := json.Marshal(expected)
	if err != nil || !bytes.Equal(want, body) {
		return "", errors.New("go cache content or approval binding changed")
	}
	return filepath.Join(root, "modcache"), nil
}

func (c *GoVerifiedCache) verifyRecordedApproval(ctx context.Context, doc goCacheDocument) error {
	if (len(doc.Entries) == 0) != doc.DependencyFree || len(doc.Entries) > 4096 {
		return errors.New("go retained approval coverage is invalid")
	}
	seen := map[domain.ResolvedArtifactIdentity]bool{}
	for _, entry := range doc.Entries {
		identity, err := domain.NewResolvedArtifactIdentity(artifactgo.Source(), entry.Module, entry.Version, "module")
		if err != nil || entry.Source != artifactgo.Source().String() || seen[identity] || entry.Policy == "" || entry.PolicyVersion == 0 || len(entry.Evidence) == 0 || len(entry.Evidence) > 4096 {
			return errors.New("go retained entry approval is invalid")
		}
		seen[identity] = true
		run, err := domain.ParseRunID(entry.Run)
		if err != nil {
			return errors.New("go retained entry Run is invalid")
		}
		digest, err := domain.NewSHA256Digest(entry.Digest)
		if err != nil {
			return errors.New("go retained entry digest is invalid")
		}
		for _, recorded := range entry.Evidence {
			id, err := domain.NewEvidenceID(recorded.ID)
			if err != nil {
				return errors.New("go retained Evidence identity is invalid")
			}
			ref, err := domain.NewEvidenceReference(id, "evidence:"+run.String()+":"+id.String())
			if err != nil {
				return err
			}
			_, hash, err := c.evidence.ReadReference(ctx, run, ref, identity, digest)
			if err != nil || hash.String() != recorded.SHA256 {
				return errors.New("go retained Evidence changed or is unavailable")
			}
		}
	}
	return nil
}

func goCacheInventory(ctx context.Context, root string) ([]goCacheFile, error) {
	var files []goCacheFile
	var total int64
	var visited int
	var fileCount int
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("go cache tree contains an invalid path")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		visited++
		if visited > 2*maxGoProjectCacheFiles {
			return errors.New("go cache tree exceeds bounded entry limits")
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		if info.IsDir() {
			files = append(files, goCacheFile{Path: filepath.ToSlash(rel), Kind: "directory"})
			return nil
		}
		total += info.Size()
		fileCount++
		if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > artifactgo.MaxZipBytes || total > maxGoProjectCacheBytes || fileCount > maxGoProjectCacheFiles {
			return errors.New("go project cache exceeds bounded content limits")
		}
		f, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		opened, statErr := f.Stat()
		if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
			_ = f.Close()
			return errors.New("go cache member changed")
		}
		hash := sha256.New()
		n, copyErr := io.Copy(hash, io.LimitReader(f, info.Size()+1))
		closeErr := f.Close()
		if copyErr != nil || closeErr != nil || n != info.Size() {
			return errors.New("go cache member read is incomplete")
		}
		files = append(files, goCacheFile{Path: filepath.ToSlash(rel), Kind: "file", Size: n, SHA256: hex.EncodeToString(hash.Sum(nil))})
		return nil
	})
	return files, err
}

func sealGoCacheTree(root string) error {
	return filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("go cache sealing found an invalid path")
		}
		if entry.IsDir() {
			return os.Chmod(name, 0o755)
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("go cache sealing requires regular files")
		}
		return os.Chmod(name, 0o444)
	})
}
