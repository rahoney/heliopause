package npm

import (
	"errors"
	"slices"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMInspectionContext = errors.New("pnpm inspection context or acquired coverage differs from the frozen selection")

// PNPMInspectionContext retains native instance wiring beside its source-neutral
// snapshot. It authorizes neither execution nor promotion.
type PNPMInspectionContext struct {
	selection PNPMSelection
	snapshot  domain.ProjectDependencySnapshot
	controls  []domain.ProjectControlFile
	runtime   string
}

func (s PNPMSelection) InspectionContext(install domain.InstallContext, controls []domain.ProjectControlFile, runtimeIdentity string) (PNPMInspectionContext, error) {
	snapshot, err := s.ProjectSnapshot(install, controls, runtimeIdentity)
	if err != nil {
		return PNPMInspectionContext{}, err
	}
	return PNPMInspectionContext{s, snapshot, append([]domain.ProjectControlFile(nil), controls...), runtimeIdentity}, nil
}

func (c PNPMInspectionContext) Valid() bool {
	return c.selection.SourceBound() && c.selection.lock.valid && c.snapshot.Valid() && c.runtime != "" && len(c.controls) == 2
}

func (c PNPMInspectionContext) Snapshot() domain.ProjectDependencySnapshot { return c.snapshot }
func (c PNPMInspectionContext) Lock() PNPMLock                             { return c.selection.Lock() }
func (c PNPMInspectionContext) RuntimeIdentity() string                    { return c.runtime }
func (c PNPMInspectionContext) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), c.controls...)
}

// A copied opaque digest cannot substitute for the full normalized snapshot.
func (c PNPMInspectionContext) MatchesSnapshot(snapshot domain.ProjectDependencySnapshot) bool {
	return c.Valid() && snapshot.Valid() && snapshot.Context() == c.snapshot.Context() &&
		snapshot.Source() == c.snapshot.Source() && snapshot.GraphDigest() == c.snapshot.GraphDigest() &&
		snapshot.DependencyFree() == c.snapshot.DependencyFree() &&
		slices.Equal(snapshot.ControlDigests(), c.snapshot.ControlDigests()) &&
		slices.Equal(snapshot.Dependencies(), c.snapshot.Dependencies())
}

// PNPMInspectionInput is structural binding only. Verification, static-all,
// introduction-time byte checks, observation and Policy remain required.
type PNPMInspectionInput struct {
	context   PNPMInspectionContext
	artifacts []domain.AcquiredArtifact
}

func (c PNPMInspectionContext) BindAcquired(snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (PNPMInspectionInput, error) {
	if !c.MatchesSnapshot(snapshot) || len(acquired) != len(snapshot.Dependencies()) {
		return PNPMInspectionInput{}, errPNPMInspectionContext
	}
	artifacts, err := bindNPMProjectAcquired(snapshot, acquired, errPNPMInspectionContext)
	if err != nil {
		return PNPMInspectionInput{}, err
	}
	return PNPMInspectionInput{c, artifacts}, nil
}

func (i PNPMInspectionInput) Valid() bool {
	return i.context.Valid() && len(i.artifacts) == len(i.context.snapshot.Dependencies())
}
func (i PNPMInspectionInput) Context() PNPMInspectionContext { return i.context }
func (i PNPMInspectionInput) Artifacts() []domain.AcquiredArtifact {
	return append([]domain.AcquiredArtifact(nil), i.artifacts...)
}

// InstancesFor returns every peer context of this exact acquired subject. The
// caller must observe all of them; a shared archive is not a shared execution.
func (i PNPMInspectionInput) InstancesFor(artifact domain.AcquiredArtifact) ([]PNPMLockedSnapshot, error) {
	if !i.Valid() || !slices.Contains(i.artifacts, artifact) {
		return nil, errPNPMInspectionContext
	}
	key := ""
	for _, pkg := range i.context.selection.lock.packages {
		if pkg.Name == artifact.Identity().Name() && pkg.Version == artifact.Identity().Version() {
			key = pkg.Key
			break
		}
	}
	var instances []PNPMLockedSnapshot
	for _, instance := range i.context.selection.lock.snapshots {
		if key != "" && instance.Package == key {
			instance.Dependencies = clonePNPMStrings(instance.Dependencies)
			instance.TransitivePeers = append([]string(nil), instance.TransitivePeers...)
			instances = append(instances, instance)
		}
	}
	if len(instances) == 0 {
		return nil, errPNPMInspectionContext
	}
	return instances, nil
}
