package npm

import (
	"errors"
	"reflect"
	"slices"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunInspectionContext = errors.New("bun inspection context or acquired coverage differs from the frozen selection")

// BunInspectionContext retains native instance wiring beside its source-neutral
// snapshot. It authorizes neither execution nor promotion.
type BunInspectionContext struct {
	selection BunSelection
	snapshot  domain.ProjectDependencySnapshot
	controls  []domain.ProjectControlFile
	runtime   string
	execution domain.DependencyResolution
}

func (s BunSelection) InspectionContext(install domain.InstallContext, controls []domain.ProjectControlFile, runtimeIdentity string) (BunInspectionContext, error) {
	snapshot, err := s.ProjectSnapshot(install, controls, runtimeIdentity)
	if err != nil {
		return BunInspectionContext{}, err
	}
	return BunInspectionContext{selection: s, snapshot: snapshot, controls: append([]domain.ProjectControlFile(nil), controls...), runtime: runtimeIdentity}, nil
}

func (c BunInspectionContext) Valid() bool {
	if !c.selection.SourceBound() || !c.selection.lock.valid || c.runtime == "" || len(c.controls) != 2 {
		return false
	}
	run := c.execution.Graph().Primary().String() != "" && c.execution.LockfileDigest().String() != ""
	return c.snapshot.Valid() != run
}

func (c BunInspectionContext) Snapshot() domain.ProjectDependencySnapshot { return c.snapshot }
func (c BunInspectionContext) Lock() BunLock                              { return c.selection.Lock() }
func (c BunInspectionContext) RuntimeIdentity() string                    { return c.runtime }
func (c BunInspectionContext) ClientVersion() string                      { return c.selection.lock.version }

func (c BunInspectionContext) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), c.controls...)
}

// A copied opaque digest cannot substitute for the full normalized snapshot.
func (c BunInspectionContext) MatchesSnapshot(snapshot domain.ProjectDependencySnapshot) bool {
	return c.Valid() && c.snapshot.Valid() && snapshot.Valid() && snapshot.Context() == c.snapshot.Context() &&
		snapshot.Source() == c.snapshot.Source() && snapshot.GraphDigest() == c.snapshot.GraphDigest() &&
		snapshot.DependencyFree() == c.snapshot.DependencyFree() &&
		slices.Equal(snapshot.ControlDigests(), c.snapshot.ControlDigests()) &&
		slices.Equal(snapshot.Dependencies(), c.snapshot.Dependencies())
}

// BunInspectionInput is structural binding only. Verification, static-all,
// introduction-time byte checks, observation and Policy remain required.
type BunInspectionInput struct {
	context   BunInspectionContext
	artifacts []domain.AcquiredArtifact
}

func (c BunInspectionContext) BindAcquired(snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (BunInspectionInput, error) {
	if !c.MatchesSnapshot(snapshot) || len(acquired) != len(snapshot.Dependencies()) {
		return BunInspectionInput{}, errBunInspectionContext
	}
	artifacts, err := bindNPMProjectAcquired(snapshot, acquired, errBunInspectionContext)
	if err != nil {
		return BunInspectionInput{}, err
	}
	return BunInspectionInput{c, artifacts}, nil
}

func (i BunInspectionInput) Valid() bool {
	return i.context.Valid() && len(i.artifacts) == len(i.context.selection.artifacts)
}
func (i BunInspectionInput) Context() BunInspectionContext { return i.context }
func (i BunInspectionInput) Artifacts() []domain.AcquiredArtifact {
	return append([]domain.AcquiredArtifact(nil), i.artifacts...)
}

// InstancesFor preserves every native tree placement of this exact acquired
// subject. Shared bytes do not collapse distinct dependency resolution contexts.
func (i BunInspectionInput) InstancesFor(artifact domain.AcquiredArtifact) ([]BunLockedPackage, error) {
	if !i.Valid() || !slices.Contains(i.artifacts, artifact) {
		return nil, errBunInspectionContext
	}
	var instances []BunLockedPackage
	for _, instance := range i.context.selection.lock.Packages() {
		if instance.Name == artifact.Identity().Name() && instance.Version == artifact.Identity().Version() {
			instances = append(instances, instance)
		}
	}
	if len(instances) == 0 {
		return nil, errBunInspectionContext
	}
	return instances, nil
}

func (c BunInspectionContext) ExecutionResolution() domain.DependencyResolution { return c.execution }
func (c BunInspectionContext) MatchesExecutionResolution(resolution domain.DependencyResolution) bool {
	return c.Valid() && !c.snapshot.Valid() && c.execution.Graph().Primary().String() != "" && reflect.DeepEqual(c.execution, resolution)
}
func (c BunInspectionContext) bindingDigest() domain.ContentDigest {
	if c.snapshot.Valid() {
		return c.snapshot.GraphDigest()
	}
	return c.execution.LockfileDigest()
}
func (c BunInspectionContext) matchesContext(other BunInspectionContext) bool {
	if c.snapshot.Valid() {
		return c.MatchesSnapshot(other.snapshot)
	}
	return c.MatchesExecutionResolution(other.execution)
}
func (c BunInspectionContext) BindExecutionAcquired(graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (BunInspectionInput, error) {
	if !c.MatchesExecutionResolution(c.execution) || !reflect.DeepEqual(graph, c.execution.Graph()) || len(acquired) != len(graph.Nodes()) {
		return BunInspectionInput{}, errBunInspectionContext
	}
	subjects, err := bindNativeExecutionAcquired(graph, acquired, c.selection.artifacts, errBunInspectionContext)
	if err != nil {
		return BunInspectionInput{}, err
	}
	return BunInspectionInput{c, subjects}, nil
}
