package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type terraformCacheEntry struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Envelope         string            `json:"envelope"`
	Integrity        string            `json:"integrity"`
	Run              string            `json:"run"`
	H1               string            `json:"h1"`
	ZH               string            `json:"zh"`
	Executable       string            `json:"executable"`
	ExecutableDigest string            `json:"executable_digest"`
	Evidence         []goCacheEvidence `json:"evidence"`
	Policy           string            `json:"policy"`
	PolicyVersion    uint64            `json:"policy_version"`
}
type terraformFileRecord struct {
	cacheFileRecord
	Mode uint32 `json:"mode"`
}
type terraformCacheDocument struct {
	Schema        int                   `json:"schema"`
	Graph         string                `json:"graph"`
	Project       string                `json:"project"`
	Policy        string                `json:"policy"`
	PolicyVersion uint64                `json:"policy_version"`
	Controls      []cacheFileRecord     `json:"controls"`
	Entries       []terraformCacheEntry `json:"entries"`
	Files         []terraformFileRecord `json:"files"`
}

// TerraformVerifiedCache stages exact rehashed packages only after complete
// entry/set ALLOW and independent required Evidence records. It has no runtime
// capability and copies neither an ambient Terraform cache nor project markers.
type TerraformVerifiedCache struct {
	intakeRoot, evidenceRoot, cacheRoot string
	evidence                            ProjectEvidenceReader
}

func NewTerraformVerifiedCache(intake, evidence, cache string, reader ProjectEvidenceReader) (*TerraformVerifiedCache, error) {
	if reader == nil || !separateRoots([]string{intake, evidence, cache}) {
		return nil, errors.New("terraform cache requires separate intake/Evidence/cache roots")
	}
	for _, r := range []string{intake, evidence, cache} {
		if !filepath.IsAbs(r) || filepath.Clean(r) != r || r == "/" {
			return nil, errors.New("terraform cache root is invalid")
		}
	}
	return &TerraformVerifiedCache{intake, evidence, cache, reader}, nil
}

func (c *TerraformVerifiedCache) approval(ctx context.Context, set domain.ProjectVerifiedSet) (terraformCacheDocument, error) {
	if c == nil || ctx == nil || !set.Valid() || set.Inspected().Snapshot().Source() != artifactterraform.Source() || len(set.Inspected().Inspections()) == 0 || len(set.Inspected().Inspections()) > 32 {
		return terraformCacheDocument{}, errors.New("terraform cache requires complete provider ALLOW")
	}
	snapshot := set.Inspected().Snapshot()
	hash := sha256.Sum256([]byte(snapshot.Context().Target().String()))
	doc := terraformCacheDocument{Schema: 1, Graph: snapshot.GraphDigest().String(), Project: hex.EncodeToString(hash[:]), Policy: set.Decision().PolicyID(), PolicyVersion: set.Decision().Version()}
	for _, control := range snapshot.ControlDigests() {
		doc.Controls = append(doc.Controls, cacheFileRecord{Path: control.Name(), SHA256: control.Digest().String()})
	}
	for _, inspection := range set.Inspected().Inspections() {
		a := inspection.Artifact()
		reference, e := artifactterraform.IdentityReference(a.Identity())
		if e != nil {
			return terraformCacheDocument{}, e
		}
		integrity, _ := a.DeclaredIntegrity()
		registryDigest, archiveDigest, e := artifactterraform.ParseIntegrity(integrity)
		if e != nil {
			return terraformCacheDocument{}, e
		}
		records, e := projectCacheEvidence(ctx, c.evidence, inspection)
		if e != nil {
			return terraformCacheDocument{}, e
		}
		required := map[string]bool{}
		for _, check := range inspection.Checks() {
			if check.Required() && check.Status() == domain.ExecutionCompleted {
				required[check.ID().String()] = true
			}
		}
		if !required["terraform-signed-package"] || !required["terraform-provider-static"] || !required["terraform-provider-dynamic"] {
			return terraformCacheDocument{}, errors.New("terraform approval omits required source/static/dynamic coverage")
		}
		bundle, e := artifactterraform.ReadIntake(c.intakeRoot, a)
		if e != nil {
			return terraformCacheDocument{}, e
		}
		if bundle.RegistryDigest() != registryDigest || bundle.ArchiveDigest() != archiveDigest {
			return terraformCacheDocument{}, errors.New("terraform cache package differs from exact selected integrity")
		}
		contents, e := artifactterraform.InspectPackage(ctx, bundle, reference.Locator())
		if e != nil {
			return terraformCacheDocument{}, e
		}
		doc.Entries = append(doc.Entries, terraformCacheEntry{a.Identity().Name(), a.Identity().Version(), a.Digest().String(), integrity, inspection.RunID().String(), contents.H1, contents.ZH, contents.Executable, contents.ExecutableDigest, records, inspection.PolicyDecision().PolicyID(), inspection.PolicyDecision().Version()})
	}
	return doc, nil
}

func terraformProviderPath(name, version string) (string, error) {
	identity, e := domain.NewResolvedArtifactIdentity(artifactterraform.Source(), name, version, "linux/amd64")
	if e != nil {
		return "", e
	}
	reference, e := artifactterraform.IdentityReference(identity)
	if e != nil {
		return "", e
	}
	parts := strings.Split(reference.Locator(), "@")
	return filepath.Join("registry.terraform.io", filepath.FromSlash(parts[0]), parts[1], "linux_amd64"), nil
}

func (c *TerraformVerifiedCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (staged domain.StagedProjectSet, resultErr error) {
	doc, e := c.approval(ctx, set)
	if e != nil {
		return staged, e
	}
	if e := ensureTrustedRoot(c.cacheRoot); e != nil {
		return staged, e
	}
	run, e := domain.NewRunID()
	if e != nil {
		return staged, e
	}
	temporary, e := os.MkdirTemp(c.cacheRoot, ".terraform-stage-")
	if e != nil {
		return staged, e
	}
	published := false
	defer func() {
		if !published {
			resultErr = errors.Join(resultErr, os.RemoveAll(temporary))
		}
		if resultErr != nil {
			staged = domain.StagedProjectSet{}
		}
	}()
	tree := filepath.Join(temporary, "providers")
	if e := os.Mkdir(tree, 0o755); e != nil {
		return staged, e
	}
	var total int64
	var files int
	for _, inspection := range set.Inspected().Inspections() {
		a := inspection.Artifact()
		if a.ContentHandle() != "intake:"+inspection.RunID().String()+":linux/amd64" {
			return staged, errors.New("terraform cache intake Run binding differs")
		}
		bundle, e := artifactterraform.ReadIntake(c.intakeRoot, a)
		if e != nil {
			return staged, e
		}
		ref, e := artifactterraform.IdentityReference(a.Identity())
		if e != nil {
			return staged, e
		}
		contents, e := artifactterraform.InspectPackage(ctx, bundle, ref.Locator())
		if e != nil {
			return staged, e
		}
		relative, e := terraformProviderPath(a.Identity().Name(), a.Identity().Version())
		if e != nil {
			return staged, e
		}
		dir := filepath.Join(tree, relative)
		if e := os.MkdirAll(dir, 0o755); e != nil {
			return staged, e
		}
		var names []string
		for name := range contents.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			body := contents.Files[name]
			total += int64(len(body))
			files++
			if total > 200<<20 || files > 10000 {
				return staged, errors.New("terraform provider set exceeds installation bounds")
			}
			if e := writeRecord(dir, name, body); e != nil {
				return staged, e
			}
			mode := os.FileMode(0o444)
			if name == contents.Executable {
				mode = 0o555
			}
			if e := os.Chmod(filepath.Join(dir, name), mode); e != nil {
				return staged, e
			}
		}
	}
	doc.Files, _, e = terraformTreeInventory(ctx, tree)
	if e != nil {
		return staged, e
	}
	body, e := json.Marshal(doc)
	if e != nil || len(body) > 4<<20 {
		return staged, errors.New("terraform cache receipt exceeds bounds")
	}
	digest, _ := domain.NewSHA256Digest(terraformHash(body))
	staged, e = domain.NewStagedProjectSet(set, "project-cache:"+run.String(), digest)
	if e != nil {
		return staged, e
	}
	if e := writeRecord(temporary, "receipt.json", body); e != nil {
		return staged, e
	}
	if e := syncTree(temporary); e != nil {
		return staged, e
	}
	final := filepath.Join(c.cacheRoot, run.String())
	if e := renameNoReplace(temporary, final); e != nil {
		return staged, e
	}
	published = true
	if e := syncDirectory(c.cacheRoot); e != nil {
		return staged, e
	}
	return staged, nil
}

func terraformHash(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

// Inventory reads via an anchored root, rejects all aliases and returns exact
// file/directory identities in addition to hashes and installation modes.
func terraformTreeInventory(ctx context.Context, directory string) ([]terraformFileRecord, map[string]os.FileInfo, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, nil, errors.New("terraform tree inventory request is invalid")
	}
	info, e := os.Lstat(directory)
	if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, errors.New("terraform provider tree is untrusted")
	}
	root, e := os.OpenRoot(directory)
	if e != nil {
		return nil, nil, e
	}
	defer root.Close()
	opened, e := root.Stat(".")
	if e != nil || !os.SameFile(info, opened) {
		return nil, nil, errors.New("terraform provider root changed")
	}
	members := map[string]os.FileInfo{}
	var records []terraformFileRecord
	var total int64
	files := 0
	e = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil || len(records) >= 20000 {
			return errors.New("terraform provider inventory exceeds entry bounds")
		}
		before, e := root.Lstat(name)
		if e != nil || before.Mode()&os.ModeSymlink != 0 || before.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return errors.New("terraform provider inventory contains an alias or special mode")
		}
		members[name] = before
		if before.IsDir() {
			if before.Mode().Perm() != 0o755 {
				return errors.New("terraform provider directory mode drifted")
			}
			records = append(records, terraformFileRecord{cacheFileRecord{Path: name, Kind: "directory"}, 0o755})
			return nil
		}
		if !before.Mode().IsRegular() || !pypiSingleLink(before) || (before.Mode().Perm() != 0o444 && before.Mode().Perm() != 0o555) || before.Size() < 0 || before.Size() > artifactterraform.MaxProviderArchiveBytes {
			return errors.New("terraform provider member is invalid")
		}
		files++
		total += before.Size()
		if files > 10000 || total > 200<<20 {
			return errors.New("terraform provider tree exceeds content bounds")
		}
		f, e := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if e != nil {
			return e
		}
		opened, e := f.Stat()
		if e != nil || !os.SameFile(before, opened) {
			_ = f.Close()
			return errors.New("terraform provider file changed")
		}
		hash := sha256.New()
		n, re := io.Copy(hash, io.LimitReader(f, before.Size()+1))
		after, ae := f.Stat()
		ce := f.Close()
		current, pe := root.Lstat(name)
		if re != nil || ae != nil || ce != nil || pe != nil || n != before.Size() || !os.SameFile(before, after) || !os.SameFile(before, current) || after.Mode() != before.Mode() || after.Size() != before.Size() || after.ModTime() != before.ModTime() || !pypiSingleLink(after) {
			return errors.New("terraform provider file changed during hash")
		}
		records = append(records, terraformFileRecord{cacheFileRecord{Path: name, Kind: "file", Size: n, SHA256: hex.EncodeToString(hash.Sum(nil))}, uint32(before.Mode().Perm())})
		return nil
	})
	if e != nil {
		return nil, nil, e
	}
	for name, before := range members {
		current, e := root.Lstat(name)
		if e != nil || !os.SameFile(before, current) || before.Mode() != current.Mode() {
			return nil, nil, errors.New("terraform provider tree identity drifted")
		}
	}
	current, e := os.Lstat(directory)
	if e != nil || !os.SameFile(info, current) {
		return nil, nil, errors.New("terraform provider root identity drifted")
	}
	return records, members, nil
}

func (c *TerraformVerifiedCache) verifyRecordedApproval(ctx context.Context, doc terraformCacheDocument) error {
	if doc.Schema != 1 || doc.Policy == "" || doc.PolicyVersion == 0 || len(doc.Entries) == 0 || len(doc.Entries) > 32 || len(doc.Controls) != 2 {
		return errors.New("terraform retained approval coverage is invalid")
	}
	seen := map[string]bool{}
	for _, entry := range doc.Entries {
		identity, e := domain.NewResolvedArtifactIdentity(artifactterraform.Source(), entry.Name, entry.Version, "linux/amd64")
		if e != nil {
			return e
		}
		if _, e := artifactterraform.IdentityReference(identity); e != nil {
			return e
		}
		if seen[entry.Name] || entry.Policy == "" || entry.PolicyVersion == 0 || len(entry.Evidence) != 3 {
			return errors.New("terraform retained entry approval is incomplete or ambiguous")
		}
		seen[entry.Name] = true
		if _, _, e := artifactterraform.ParseIntegrity(entry.Integrity); e != nil {
			return e
		}
		run, e := domain.ParseRunID(entry.Run)
		if e != nil {
			return e
		}
		digest, e := domain.NewSHA256Digest(entry.Envelope)
		if e != nil {
			return e
		}
		covered := map[string]bool{}
		for _, record := range entry.Evidence {
			id, e := domain.NewEvidenceID(record.ID)
			if e != nil {
				return e
			}
			ref, e := domain.NewEvidenceReference(id, "evidence:"+run.String()+":"+id.String())
			if e != nil {
				return e
			}
			item, hash, e := c.evidence.ReadReference(ctx, run, ref, identity, digest)
			if e != nil || hash.String() != record.SHA256 || covered[item.CheckID().String()] {
				return errors.New("terraform retained Evidence changed, missing or duplicated")
			}
			covered[item.CheckID().String()] = true
		}
		if !covered["terraform-signed-package"] || !covered["terraform-provider-static"] || !covered["terraform-provider-dynamic"] {
			return errors.New("terraform retained Evidence omits required coverage")
		}
	}
	return nil
}

func (c *TerraformVerifiedCache) openDocument(ctx context.Context, handle string, digest domain.ContentDigest, expected terraformCacheDocument) (string, error) {
	if e := c.verifyRecordedApproval(ctx, expected); e != nil {
		return "", e
	}
	id, ok := strings.CutPrefix(handle, "project-cache:")
	if !ok {
		return "", errors.New("terraform cache handle is invalid")
	}
	if _, e := domain.ParseRunID(id); e != nil {
		return "", e
	}
	directory := filepath.Join(c.cacheRoot, id)
	if e := trustedExistingDirectory(directory); e != nil {
		return "", e
	}
	root, e := os.OpenRoot(directory)
	if e != nil {
		return "", e
	}
	defer root.Close()
	body, info, e := readGoTransactionControl(root, "receipt.json")
	if e != nil || !pypiSingleLink(info) || terraformHash(body) != digest.String() {
		return "", errors.New("terraform cache receipt changed or unavailable")
	}
	tree := filepath.Join(directory, "providers")
	expected.Files, _, e = terraformTreeInventory(ctx, tree)
	if e != nil {
		return "", e
	}
	want, e := json.Marshal(expected)
	if e != nil || !bytes.Equal(want, body) {
		return "", errors.New("terraform provider cache differs from retained approval")
	}
	return tree, nil
}

func (c *TerraformVerifiedCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (string, error) {
	if !staged.Valid() {
		return "", errors.New("terraform staged set is invalid")
	}
	doc, e := c.approval(ctx, staged.Set())
	if e != nil {
		return "", e
	}
	return c.openDocument(ctx, staged.ContentHandle(), staged.Digest(), doc)
}
