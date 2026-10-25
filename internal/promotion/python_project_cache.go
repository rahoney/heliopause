package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// ProjectRecordBuilder is implemented by the trusted record adapter and wired
// by composition. Promotion does not import Evidence serialization machinery.
type ProjectRecordBuilder interface {
	BuildProject(context.Context, domain.OperationID, domain.ProjectVerifiedSet, []domain.LockedDependency, []domain.DependencyEdge, []domain.ProjectControlFile, string) (domain.ContentDigest, []byte, []byte, error)
}

type PythonProjectCache struct {
	storage   *LocalStaging
	evidence  ProjectEvidenceReader
	records   ProjectRecordBuilder
	operation domain.OperationID
}

func NewPythonProjectCache(intakeRoot, evidenceRoot, cacheRoot string, evidence ProjectEvidenceReader, records ProjectRecordBuilder, operation domain.OperationID) (*PythonProjectCache, error) {
	if evidence == nil || records == nil || operation.String() == "" {
		return nil, errors.New("python project cache requires recorded Evidence, records and the actual operation")
	}
	for _, root := range []string{intakeRoot, evidenceRoot, cacheRoot} {
		if root == "/" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return nil, errors.New("python cache roots must be canonical")
		}
	}
	storage, err := NewLocalStaging(intakeRoot, evidenceRoot, cacheRoot)
	if err != nil {
		return nil, err
	}
	return &PythonProjectCache{storage, evidence, records, operation}, nil
}

type pythonProjectCacheEntry struct {
	Source, Name, Version, Digest, Integrity, Run, Policy string
	PolicyVersion                                         uint64
	Reasons                                               []string
	Evidence                                              []goCacheEvidence
}
type pythonProjectReceipt struct {
	Schema                                          int
	Operation, Graph, Target, Mode, Runtime, Policy string
	PolicyVersion                                   uint64
	Reasons                                         []string
	DependencyFree                                  bool
	Controls                                        []cacheFileRecord
	Entries                                         []pythonProjectCacheEntry
	ManifestID                                      string
	Files                                           []cacheFileRecord
}
type pythonProjectMaterial struct {
	nodes          []domain.LockedDependency
	manifest       domain.ContentDigest
	document, sbom []byte
}

func (c *PythonProjectCache) approval(ctx context.Context, set domain.ProjectVerifiedSet, update domain.ProjectDependencyUpdate) (pythonProjectReceipt, error) {
	if ctx == nil || c == nil || !set.Valid() || !update.Valid() || update.HasRequestedResolution() || !reflect.DeepEqual(set.Inspected().Snapshot(), update.Snapshot()) || update.Snapshot().Source() != artifactpypi.PublicPyPIProfile().Source() || (update.Snapshot().Context().Mode() != domain.InstallPythonVenv && update.Snapshot().Context().Mode() != domain.InstallPythonProjectControls) {
		return pythonProjectReceipt{}, errors.New("python cache requires the exact approved complete UV selection")
	}
	if err := ctx.Err(); err != nil {
		return pythonProjectReceipt{}, err
	}
	s := update.Snapshot()
	reasons := set.Decision().Reasons()
	sort.Strings(reasons)
	doc := pythonProjectReceipt{Schema: 1, Operation: c.operation.String(), Graph: s.GraphDigest().String(), Target: s.Context().Target().String(), Mode: string(s.Context().Mode()), Runtime: sandbox.UVResolverRuntimeIdentity(), Policy: set.Decision().PolicyID(), PolicyVersion: set.Decision().Version(), Reasons: reasons, DependencyFree: s.DependencyFree(), Controls: []cacheFileRecord{}, Entries: []pythonProjectCacheEntry{}}
	for _, control := range s.ControlDigests() {
		doc.Controls = append(doc.Controls, cacheFileRecord{Path: control.Name(), SHA256: control.Digest().String()})
	}
	for _, inspection := range set.Inspected().Inspections() {
		a := inspection.Artifact()
		id := a.Identity()
		integrity, ok := a.DeclaredIntegrity()
		if !ok || id.Variant() != "wheel" || integrity != "sha256:"+a.Digest().String() {
			return pythonProjectReceipt{}, errors.New("python project entry lacks exact wheel verification")
		}
		evidence, err := projectCacheEvidence(ctx, c.evidence, inspection)
		if err != nil {
			return pythonProjectReceipt{}, err
		}
		r := inspection.PolicyDecision().Reasons()
		sort.Strings(r)
		doc.Entries = append(doc.Entries, pythonProjectCacheEntry{id.Source().String(), id.Name(), id.Version(), a.Digest().String(), integrity, inspection.RunID().String(), inspection.PolicyDecision().PolicyID(), inspection.PolicyDecision().Version(), r, evidence})
	}
	return doc, nil
}

func uvSyncSelectedLock(update domain.ProjectDependencyUpdate) (artifactpypi.UVProjectLock, error) {
	for _, control := range update.SelectedControls() {
		if control.Name() == artifactpypi.UVSelectionLockControlName {
			return artifactpypi.ParseUVProjectLock(control.Body(), sandbox.PinnedPythonRuntime().PythonVersion)
		}
	}
	return artifactpypi.UVProjectLock{}, errors.New("python project cache lacks the actual selected native lock")
}

// Re-read approved bytes and filenames at both storage boundaries. Neither a
// receipt path nor a wheel's untrusted METADATA supplies approval authority.
func (c *PythonProjectCache) material(ctx context.Context, root string, set domain.ProjectVerifiedSet, update domain.ProjectDependencyUpdate, intake bool) (result pythonProjectMaterial, resultErr error) {
	lock, err := uvSyncSelectedLock(update)
	if err != nil {
		return result, err
	}
	wheels, err := lock.SelectResolvedWheels(update.Snapshot().Dependencies())
	if err != nil {
		return result, err
	}
	byURL := map[string]artifactpypi.UVLockedWheel{}
	for _, wheel := range wheels {
		byURL[wheel.URL] = wheel
	}
	inputs := []artifactpypi.UVWheelInput{}
	files := []*os.File{}
	defer func() {
		for _, file := range files {
			resultErr = errors.Join(resultErr, file.Close())
		}
		if resultErr != nil {
			result = pythonProjectMaterial{}
		}
	}()
	for _, inspection := range set.Inspected().Inspections() {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		a := inspection.Artifact()
		var resolved domain.ResolvedArtifact
		for _, candidate := range update.Snapshot().Dependencies() {
			if candidate.Identity() == a.Identity() {
				resolved = candidate
				break
			}
		}
		wheel, ok := byURL[resolved.AcquisitionLocator()]
		if !ok || wheel.SHA256 != a.Digest().String() {
			return result, errors.New("python project wheel differs from approved native selection")
		}
		path := filepath.Join(root, "artifacts", a.Digest().String()+".whl")
		if intake {
			if a.ContentHandle() != "intake:"+inspection.RunID().String()+":wheel" {
				return result, errors.New("python project intake Run binding differs")
			}
			directory := filepath.Join(c.storage.intakeRoot, inspection.RunID().String())
			name, err := readBoundedPromotionFile(filepath.Join(directory, "filename"), 4096)
			if err != nil || string(name) != wheel.Filename {
				return result, errors.New("python intake filename differs from actual native wheel")
			}
			path = filepath.Join(directory, "wheel.whl")
		}
		if rejectSymlinkPath(path) != nil {
			return result, errors.New("python project archive path is unsafe")
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || !pypiSingleLink(info) || info.Size() != wheel.Size || uint64(info.Size()) != a.SizeBytes() {
			return result, errors.New("python project archive size or identity differs")
		}
		file, err := os.Open(path)
		if err != nil {
			return result, err
		}
		files = append(files, file)
		opened, err := file.Stat()
		if err != nil || !os.SameFile(info, opened) {
			return result, errors.New("python project archive changed while opening")
		}
		inputs = append(inputs, artifactpypi.UVWheelInput{URL: wheel.URL, Reader: file, Size: info.Size()})
	}
	runtime := sandbox.PinnedPythonRuntime()
	nodes, edges, err := lock.ProjectRecords(update.Snapshot().Dependencies(), inputs, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}, artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits())
	if err != nil {
		return result, err
	}
	digest, document, sbom, err := c.records.BuildProject(ctx, c.operation, set, nodes, edges, update.SelectedControls(), sandbox.UVResolverRuntimeIdentity())
	if err != nil {
		return result, err
	}
	if digest.String() == "" || len(document) == 0 || len(document) > 32<<20 || len(sbom) == 0 || len(sbom) > 32<<20 || !json.Valid(document) || !json.Valid(sbom) {
		return result, errors.New("python project records are invalid or exceed bounds")
	}
	return pythonProjectMaterial{nodes, digest, document, sbom}, nil
}

func (c *PythonProjectCache) stageProject(ctx context.Context, set domain.ProjectVerifiedSet, update domain.ProjectDependencyUpdate) (staged domain.StagedProjectSet, resultErr error) {
	doc, err := c.approval(ctx, set, update)
	if err != nil {
		return staged, err
	}
	if err := ensureTrustedRoot(c.storage.stagingRoot); err != nil {
		return staged, err
	}
	if err := artifactpypi.CheckTemporaryDisk(c.storage.stagingRoot, artifactpypi.ResourcePolicyFromContext(ctx)); err != nil {
		return staged, err
	}
	material, err := c.material(ctx, "", set, update, true)
	if err != nil {
		return staged, err
	}
	id, err := domain.NewRunID()
	if err != nil {
		return staged, err
	}
	temporary, err := os.MkdirTemp(c.storage.stagingRoot, ".python-project-stage-")
	if err != nil {
		return staged, err
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
	payload := filepath.Join(temporary, "payload")
	artifacts := filepath.Join(payload, "artifacts")
	if err := os.MkdirAll(artifacts, 0o700); err != nil {
		return staged, err
	}
	written := map[string]bool{}
	for _, inspection := range set.Inspected().Inspections() {
		if err := ctx.Err(); err != nil {
			return staged, err
		}
		if err := c.storage.stageArtifact(artifacts, inspection, written); err != nil {
			return staged, err
		}
	}
	if err := writeRecord(payload, manifestFilename, material.document); err != nil {
		return staged, err
	}
	if err := writeRecord(payload, sbomFilename, material.sbom); err != nil {
		return staged, err
	}
	// Build records again from the copied bytes; mutation between intake reads
	// cannot preserve a receipt for a different archive.
	copied, err := c.material(ctx, payload, set, update, false)
	if err != nil || copied.manifest != material.manifest || !bytes.Equal(copied.document, material.document) || !bytes.Equal(copied.sbom, material.sbom) {
		return staged, errors.Join(errors.New("python project changed during staging"), err)
	}
	doc.ManifestID = material.manifest.String()
	doc.Files, err = pythonProjectInventory(ctx, payload, false)
	if err != nil {
		return staged, err
	}
	body, err := json.Marshal(doc)
	if err != nil || len(body) > 4<<20 {
		return staged, errors.New("python project receipt exceeds bounds")
	}
	if err := writeRecord(temporary, "receipt.json", body); err != nil {
		return staged, err
	}
	if err := sealGoCacheTree(temporary); err != nil {
		return staged, err
	}
	if err := syncTree(temporary); err != nil {
		return staged, err
	}
	if err := ctx.Err(); err != nil {
		return staged, err
	}
	final := filepath.Join(c.storage.stagingRoot, id.String())
	if err := renameNoReplace(temporary, final); err != nil {
		return staged, err
	}
	published = true
	if err := syncDirectory(c.storage.stagingRoot); err != nil {
		return staged, errors.Join(err, os.RemoveAll(final))
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return staged, err
	}
	return domain.NewStagedProjectSet(set, "project-cache:"+id.String(), digest)
}

func pythonProjectInventory(ctx context.Context, root string, sealed bool) ([]cacheFileRecord, error) {
	// Archive expansion still uses the public graph limits; this storage census
	// includes only compressed wheels and two bounded records.
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	return projectCacheInventory(ctx, root, 128, policy.MaxGraphCompressed()+(64<<20), policy.MaxGraphCompressed()+(32<<20), sealed)
}

func (c *PythonProjectCache) openProject(ctx context.Context, staged domain.StagedProjectSet, update domain.ProjectDependencyUpdate) (string, pythonProjectMaterial, error) {
	if !staged.Valid() {
		return "", pythonProjectMaterial{}, errors.New("python project cache lacks complete staging approval")
	}
	doc, err := c.approval(ctx, staged.Set(), update)
	if err != nil {
		return "", pythonProjectMaterial{}, err
	}
	id, ok := strings.CutPrefix(staged.ContentHandle(), "project-cache:")
	if !ok {
		return "", pythonProjectMaterial{}, errors.New("python project handle is invalid")
	}
	if _, err := domain.ParseRunID(id); err != nil {
		return "", pythonProjectMaterial{}, err
	}
	root := filepath.Join(c.storage.stagingRoot, id)
	if err := trustedExistingDirectory(root); err != nil {
		return "", pythonProjectMaterial{}, err
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 2 || entries[0].Name() != "payload" || entries[1].Name() != "receipt.json" {
		return "", pythonProjectMaterial{}, errors.New("python project cache contains undeclared top-level content")
	}
	path := filepath.Join(root, "receipt.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o444 || !pypiSingleLink(info) {
		return "", pythonProjectMaterial{}, errors.New("python project receipt is unsealed")
	}
	body, err := readBoundedPromotionFile(path, 4<<20)
	if err != nil {
		return "", pythonProjectMaterial{}, err
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != staged.Digest().String() {
		return "", pythonProjectMaterial{}, errors.New("python project receipt changed")
	}
	payload := filepath.Join(root, "payload")
	files, err := pythonProjectInventory(ctx, payload, true)
	if err != nil {
		return "", pythonProjectMaterial{}, err
	}
	material, err := c.material(ctx, payload, staged.Set(), update, false)
	if err != nil {
		return "", pythonProjectMaterial{}, err
	}
	doc.ManifestID = material.manifest.String()
	doc.Files = files
	want, err := json.Marshal(doc)
	if err != nil || !bytes.Equal(want, body) {
		return "", pythonProjectMaterial{}, errors.New("python project cache approval or census changed")
	}
	manifest, err := readBoundedPromotionFile(filepath.Join(payload, manifestFilename), 32<<20)
	if err != nil || !bytes.Equal(manifest, material.document) {
		return "", pythonProjectMaterial{}, errors.New("python project Manifest changed")
	}
	sbom, err := readBoundedPromotionFile(filepath.Join(payload, sbomFilename), 32<<20)
	if err != nil || !bytes.Equal(sbom, material.sbom) {
		return "", pythonProjectMaterial{}, errors.New("python project SBOM changed")
	}
	return payload, material, nil
}
