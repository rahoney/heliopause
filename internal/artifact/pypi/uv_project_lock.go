package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// UVProjectLock is a complete unapproved wheel selection. Unlike UVLock, it
// has no requested primary. Only an explicitly present packages=[] represents
// an empty selection; neither its parsing nor binding grants deletion/Policy.
type UVProjectLock struct {
	python   string
	packages []UVLockedPackage
	body     []byte
}

func ParseUVProjectLock(body []byte, expectedPython string) (UVProjectLock, error) {
	packages, err := parseUVLockedPackages(body, expectedPython, "", "", false, true)
	if err != nil {
		return UVProjectLock{}, err
	}
	return UVProjectLock{expectedPython, packages, append([]byte(nil), body...)}, nil
}

func (l UVProjectLock) Packages() []UVLockedPackage { return cloneUVLockedPackages(l.packages) }

func (l UVProjectLock) Control() (domain.ProjectControlFile, error) {
	if len(l.body) == 0 {
		return domain.ProjectControlFile{}, errors.New("complete UV lock is unavailable")
	}
	return domain.NewProjectControlFile(UVSelectionLockControlName, l.body, true)
}

func (l UVProjectLock) SelectInstalledWheelControls(controls []UVInstalledWheelControls, target WheelTarget) ([]UVLockedWheel, error) {
	if len(l.body) == 0 {
		return nil, errors.New("complete UV lock is unavailable")
	}
	return selectUVInstalledWheelControls(l.packages, l.python, controls, target, true)
}

// BindProject reuses independent full-archive/metadata/constraint validation.
// Its snapshot is still unapproved. The caller must separately bind the actual
// runtime, original guard and source intent before entry/set Policy/publication.
func (l UVProjectLock) BindProject(install domain.InstallContext, controls []domain.ProjectControlFile, inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) (domain.ProjectDependencySnapshot, error) {
	if len(l.body) == 0 || !install.Valid() || install.Mode() != domain.InstallPythonVenv || !utf8.ValidString(install.Target().String()) || !utf8.ValidString(target.Platform) || target.Platform == "" || len(controls) == 0 || len(controls) > 32 {
		return domain.ProjectDependencySnapshot{}, errors.New("complete UV project context or controls are unavailable")
	}
	digests := make([]domain.ProjectControlDigest, 0, len(controls))
	seen := map[string]bool{}
	foundLock := false
	for _, control := range controls {
		if !control.Present() || seen[control.Name()] {
			return domain.ProjectDependencySnapshot{}, errors.New("complete UV controls are absent or ambiguous")
		}
		seen[control.Name()] = true
		if control.Name() == UVSelectionLockControlName {
			if !bytes.Equal(control.Body(), l.body) {
				return domain.ProjectDependencySnapshot{}, errors.New("complete UV native lock control changed")
			}
			foundLock = true
		}
		digest, err := domain.NewProjectControlDigest(control.Name(), control.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		digests = append(digests, digest)
	}
	if !foundLock {
		return domain.ProjectDependencySnapshot{}, errors.New("complete UV native lock control is missing")
	}
	candidates, err := bindUVSelectedWheels(l.packages, l.python, "", inputs, pages, target, limits)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	artifacts := make([]domain.ResolvedArtifact, 0, len(candidates))
	for _, candidate := range candidates {
		seenDependencies := map[string]bool{}
		for _, dependency := range candidate.dependencies {
			if dependency == candidate.project || seenDependencies[dependency] {
				return domain.ProjectDependencySnapshot{}, errors.New("complete UV dependency edge is invalid or repeated")
			}
			seenDependencies[dependency] = true
		}
		identity, err := domain.NewResolvedArtifactIdentity(candidate.source, candidate.project, candidate.version, "wheel")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		artifact, err := domain.NewResolvedArtifact(identity, candidate.url, "sha256:"+candidate.sha256)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		artifacts = append(artifacts, artifact)
	}
	digest, err := uvProjectSelectionDigest(install, l.python, target, l.body, digests, artifacts)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if len(l.packages) == 0 {
		return domain.NewDependencyFreeProjectSnapshot(install, PublicPyPIProfile().Source(), digests, digest)
	}
	return domain.NewProjectDependencySnapshot(install, PublicPyPIProfile().Source(), digests, artifacts, digest)
}

// Bind exact wheel choice and every selected control, not just a pylock with
// several alternatives. Input order cannot select a different approval subject.
func uvProjectSelectionDigest(install domain.InstallContext, python string, target WheelTarget, lock []byte, controls []domain.ProjectControlDigest, artifacts []domain.ResolvedArtifact) (domain.ContentDigest, error) {
	type controlRecord struct {
		Name   string
		Digest string
	}
	type artifactRecord struct{ Source, Name, Version, Variant, URL, Integrity string }
	document := struct {
		Schema, Target, Mode, Python string
		Tags                         WheelTarget
		Lock                         string
		Controls                     []controlRecord
		Artifacts                    []artifactRecord
	}{Schema: "uv-project-selection/v1", Target: install.Target().String(), Mode: string(install.Mode()), Python: python, Tags: target, Controls: []controlRecord{}, Artifacts: []artifactRecord{}}
	sum := sha256.Sum256(lock)
	document.Lock = hex.EncodeToString(sum[:])
	for _, control := range controls {
		document.Controls = append(document.Controls, controlRecord{control.Name(), control.Digest().String()})
	}
	for _, artifact := range artifacts {
		id := artifact.Identity()
		document.Artifacts = append(document.Artifacts, artifactRecord{id.Source().String(), id.Name(), id.Version(), id.Variant(), artifact.AcquisitionLocator(), artifact.DeclaredIntegrity()})
	}
	sort.Slice(document.Controls, func(i, j int) bool { return document.Controls[i].Name < document.Controls[j].Name })
	sort.Slice(document.Artifacts, func(i, j int) bool { return document.Artifacts[i].Name < document.Artifacts[j].Name })
	body, err := json.Marshal(document)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	sum = sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
}
