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

var errPNPMProjectSnapshot = errors.New("pnpm complete project selection is unavailable or differs from its controls")

type pnpmSnapshotLock struct {
	Roots     []PNPMLockedRoot
	Packages  []PNPMLockedPackage
	Instances []PNPMLockedSnapshot
}

func pnpmSnapshotLockState(lock PNPMLock) pnpmSnapshotLock {
	return pnpmSnapshotLock{lock.Roots(), lock.Packages(), lock.Snapshots()}
}

// ProjectSnapshot freezes unique archive coverage and the entire peer-instance
// wiring. Source binding is required but never substitutes for quarantine.
func (s PNPMSelection) ProjectSnapshot(install domain.InstallContext, controls []domain.ProjectControlFile, runtimeIdentity string) (domain.ProjectDependencySnapshot, error) {
	if !s.bound || !s.lock.valid || !install.Valid() || (install.Mode() != domain.InstallNewTarget && install.Mode() != domain.InstallNPMProject) || len(controls) != 2 || runtimeIdentity == "" || len(runtimeIdentity) > 1024 || !utf8.ValidString(runtimeIdentity) {
		return domain.ProjectDependencySnapshot{}, errPNPMProjectSnapshot
	}
	for _, r := range runtimeIdentity {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return domain.ProjectDependencySnapshot{}, errPNPMProjectSnapshot
		}
	}
	byName := map[string]domain.ProjectControlFile{}
	var digests []domain.ProjectControlDigest
	for _, control := range controls {
		if (control.Name() != "package.json" && control.Name() != "pnpm-lock.yaml") || !control.Present() || len(control.Body()) == 0 || byName[control.Name()].Name() != "" {
			return domain.ProjectDependencySnapshot{}, errPNPMProjectSnapshot
		}
		byName[control.Name()] = control
		digest, err := domain.NewProjectControlDigest(control.Name(), control.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		digests = append(digests, digest)
	}
	parsed, err := ParsePNPMLock(byName["package.json"].Body(), byName["pnpm-lock.yaml"].Body(), s.lock.version)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, errPNPMProjectSnapshot
	}
	state := pnpmSnapshotLockState(s.lock)
	selected, err := json.Marshal(state)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	provided, err := json.Marshal(pnpmSnapshotLockState(parsed))
	if err != nil || !bytes.Equal(selected, provided) {
		return domain.ProjectDependencySnapshot{}, errPNPMProjectSnapshot
	}
	if len(s.artifacts) != len(s.lock.packages) {
		return domain.ProjectDependencySnapshot{}, errPNPMProjectSnapshot
	}

	type controlRecord struct{ Name, SHA256 string }
	type artifactRecord struct{ Source, Name, Version, Variant, URL, Integrity string }
	document := struct {
		Schema, Target, Mode, ClientVersion, Runtime string
		Controls                                     []controlRecord
		Artifacts                                    []artifactRecord
		Lock                                         pnpmSnapshotLock
	}{Schema: "pnpm-project-selection/v1", Target: install.Target().String(), Mode: string(install.Mode()), ClientVersion: s.lock.version, Runtime: runtimeIdentity, Controls: []controlRecord{}, Artifacts: []artifactRecord{}, Lock: state}
	for _, control := range digests {
		document.Controls = append(document.Controls, controlRecord{control.Name(), control.Digest().String()})
	}
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
