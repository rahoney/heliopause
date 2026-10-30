package npm

import (
	"errors"
	"reflect"
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
	execution domain.DependencyResolution
}

func (s PNPMSelection) InspectionContext(install domain.InstallContext, controls []domain.ProjectControlFile, runtimeIdentity string) (PNPMInspectionContext, error) {
	snapshot, err := s.ProjectSnapshot(install, controls, runtimeIdentity)
	if err != nil {
		return PNPMInspectionContext{}, err
	}
	return PNPMInspectionContext{selection: s, snapshot: snapshot, controls: append([]domain.ProjectControlFile(nil), controls...), runtime: runtimeIdentity}, nil
}

func (c PNPMInspectionContext) Valid() bool {
	if !c.selection.SourceBound() || !c.selection.lock.valid || c.runtime == "" || len(c.controls) != 2 {
		return false
	}
	run := c.execution.Graph().Primary().String() != "" && c.execution.LockfileDigest().String() != ""
	return c.snapshot.Valid() != run
}

func (c PNPMInspectionContext) Snapshot() domain.ProjectDependencySnapshot       { return c.snapshot }
func (c PNPMInspectionContext) Lock() PNPMLock                                   { return c.selection.Lock() }
func (c PNPMInspectionContext) RuntimeIdentity() string                          { return c.runtime }
func (c PNPMInspectionContext) ExecutionResolution() domain.DependencyResolution { return c.execution }
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

func (c PNPMInspectionContext) MatchesExecutionResolution(resolution domain.DependencyResolution) bool {
	return c.Valid() && !c.snapshot.Valid() && c.execution.Graph().Primary().String() != "" && reflect.DeepEqual(c.execution, resolution)
}

func (c PNPMInspectionContext) bindingDigest() domain.ContentDigest {
	if c.snapshot.Valid() {
		return c.snapshot.GraphDigest()
	}
	return c.execution.LockfileDigest()
}

// BindExecutionAcquired preserves every graph node while selecting one verified
// original per unique tarball for the existing complete PNPM inspector.
func (c PNPMInspectionContext) BindExecutionAcquired(graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (PNPMInspectionInput, error) {
	if !c.MatchesExecutionResolution(c.execution) || !reflect.DeepEqual(graph, c.execution.Graph()) || len(acquired) != len(graph.Nodes()) {
		return PNPMInspectionInput{}, errPNPMInspectionContext
	}
	byIdentity := map[domain.ResolvedArtifactIdentity]domain.AcquiredArtifact{}
	runs := map[domain.RunID]domain.ResolvedArtifactIdentity{}
	for _, node := range graph.Nodes() {
		subject, present := acquired[node.Node()]
		run, err := npmAcquiredRun(subject, node.Artifact(), errPNPMInspectionContext)
		if !present || err != nil {
			return PNPMInspectionInput{}, errPNPMInspectionContext
		}
		if identity, duplicate := runs[run]; duplicate && identity != subject.Identity() {
			return PNPMInspectionInput{}, errPNPMInspectionContext
		}
		runs[run] = subject.Identity()
		previous, duplicate := byIdentity[subject.Identity()]
		if duplicate && (previous.Digest() != subject.Digest() || previous.SizeBytes() != subject.SizeBytes()) {
			return PNPMInspectionInput{}, errPNPMInspectionContext
		}
		if !duplicate || node.Node() == graph.Primary() {
			byIdentity[subject.Identity()] = subject
		}
	}
	var subjects []domain.AcquiredArtifact
	for _, original := range c.selection.artifacts {
		subject, present := byIdentity[original.Identity()]
		if !present {
			return PNPMInspectionInput{}, errPNPMInspectionContext
		}
		subjects = append(subjects, subject)
	}
	return PNPMInspectionInput{c, subjects}, nil
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
	return i.context.Valid() && len(i.artifacts) == len(i.context.selection.artifacts)
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
