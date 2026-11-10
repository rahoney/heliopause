package npm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunProjectSnapshot = errors.New("bun complete project selection is unavailable or differs from its controls")

type bunSnapshotLock struct {
	Roots     []BunLockedRoot
	Instances []BunLockedPackage
}

func bunSnapshotState(lock BunLock) bunSnapshotLock {
	return bunSnapshotLock{lock.Roots(), lock.Packages()}
}

// ProjectSnapshot freezes exact controls/runtime and every instance edge while
// retaining unique archive coverage. This is not an inspection or Policy result.
func (s BunSelection) ProjectSnapshot(install domain.InstallContext, controls []domain.ProjectControlFile, runtimeIdentity string) (domain.ProjectDependencySnapshot, error) {
	if !s.bound || !s.lock.valid || !install.Valid() || install.Mode() != domain.InstallNewTarget && install.Mode() != domain.InstallNPMProject || len(controls) != 2 || runtimeIdentity == "" || len(runtimeIdentity) > 1024 || !utf8.ValidString(runtimeIdentity) {
		return domain.ProjectDependencySnapshot{}, errBunProjectSnapshot
	}
	for _, char := range runtimeIdentity {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return domain.ProjectDependencySnapshot{}, errBunProjectSnapshot
		}
	}
	byName := map[string]domain.ProjectControlFile{}
	var digests []domain.ProjectControlDigest
	type controlRecord struct {
		Name, SHA256 string
		Present      bool
	}
	controlRecords := []controlRecord{}
	for _, control := range controls {
		name := control.Name()
		if name != "package.json" && name != "bun.lock" || byName[name].Name() != "" || name == "package.json" && (!control.Present() || len(control.Body()) == 0) {
			return domain.ProjectDependencySnapshot{}, errBunProjectSnapshot
		}
		byName[name] = control
		digest, err := domain.NewProjectControlDigest(name, control.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		digests = append(digests, digest)
		controlRecords = append(controlRecords, controlRecord{name, control.Digest().String(), control.Present()})
	}
	manifest, lockControl := byName["package.json"], byName["bun.lock"]
	if s.lock.manifestDigest != sha256.Sum256(manifest.Body()) || s.lock.lockDigest != lockControl.Digest() || s.lock.lockPresent != lockControl.Present() {
		return domain.ProjectDependencySnapshot{}, errBunProjectSnapshot
	}
	parsed, err := ParseBunLock(manifest.Body(), lockControl, s.lock.version)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, errBunProjectSnapshot
	}
	state := bunSnapshotState(s.lock)
	selected, err := json.Marshal(state)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	provided, err := json.Marshal(bunSnapshotState(parsed))
	if err != nil || !bytes.Equal(selected, provided) || !s.completeBunArtifacts() {
		return domain.ProjectDependencySnapshot{}, errBunProjectSnapshot
	}
	type artifactRecord struct{ Source, Name, Version, Variant, URL, Integrity string }
	document := struct {
		Schema, Target, Mode, ClientVersion, Runtime string
		Controls                                     []controlRecord
		Artifacts                                    []artifactRecord
		Lock                                         bunSnapshotLock
	}{Schema: "bun-project-selection/v1", Target: install.Target().String(), Mode: string(install.Mode()), ClientVersion: s.lock.version, Runtime: runtimeIdentity, Controls: controlRecords, Artifacts: []artifactRecord{}, Lock: state}
	for _, artifact := range s.artifacts {
		identity := artifact.Identity()
		document.Artifacts = append(document.Artifacts, artifactRecord{identity.Source().String(), identity.Name(), identity.Version(), identity.Variant(), artifact.AcquisitionLocator(), artifact.DeclaredIntegrity()})
	}
	sort.Slice(document.Controls, func(i, j int) bool { return document.Controls[i].Name < document.Controls[j].Name })
	sort.Slice(document.Artifacts, func(i, j int) bool {
		a, b := document.Artifacts[i], document.Artifacts[j]
		return a.Name+"@"+a.Version+"#"+a.Variant < b.Name+"@"+b.Version+"#"+b.Variant
	})
	body, err := json.Marshal(document)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	hash := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	source, err := domain.NewSourceID("npm")
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if parsed.DependencyFree() {
		return domain.NewDependencyFreeProjectSnapshot(install, source, digests, digest)
	}
	return domain.NewProjectDependencySnapshot(install, source, digests, s.artifacts, digest)
}

func (s BunSelection) completeBunArtifacts() bool {
	wanted := map[string]BunLockedPackage{}
	for _, pkg := range s.lock.packages {
		wanted[pkg.Name+"@"+pkg.Version] = pkg
	}
	if len(wanted) != len(s.artifacts) {
		return false
	}
	seen := map[string]bool{}
	for _, artifact := range s.artifacts {
		identity := artifact.Identity()
		key := identity.Name() + "@" + identity.Version()
		pkg, exists := wanted[key]
		if !exists || seen[key] || !bunArtifactMatches(pkg, artifact) {
			return false
		}
		seen[key] = true
	}
	return true
}
