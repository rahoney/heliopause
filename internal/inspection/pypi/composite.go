package pypi

import (
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// graphStaticState is private PyPI preparation state. It never crosses the
// Application boundary or becomes a Domain identity.
type graphStaticState struct {
	wheel   artifactpypi.WheelInspection
	dynamic bool
}

func (s graphStaticState) footprint() (files, uncompressed int64, ok bool) {
	if !s.dynamic {
		return 0, 0, false
	}
	for _, item := range s.wheel.Files {
		if item.Size < 0 || uncompressed > int64(^uint64(0)>>1)-item.Size {
			return 0, 0, false
		}
		files++
		uncompressed += item.Size
	}
	return files, uncompressed, true
}

// CompositeInspector keeps PyPI archive parsing before its dependent dynamic
// wheel import check. Source distributions remain static until the dedicated
// PEP 517 build boundary supplies a derived wheel.
type CompositeInspector struct {
	static  *StaticInspector
	dynamic *DynamicInspector
}

func NewCompositeInspector(static *StaticInspector, dynamic *DynamicInspector) (*CompositeInspector, error) {
	if static == nil || dynamic == nil {
		return nil, errors.New("PyPI composite inspector requires static and dynamic inspectors")
	}
	return &CompositeInspector{static: static, dynamic: dynamic}, nil
}

func (i *CompositeInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || i.static == nil || i.dynamic == nil {
		return domain.InspectionReport{}, errors.New("PyPI composite inspector is unavailable")
	}
	if artifact.Identity().Variant() == "sdist" {
		return i.static.Inspect(ctx, artifact)
	}
	wheel, static, err := i.static.InspectWheel(ctx, artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if hasBlockingFinding(static) {
		return static, nil
	}
	dynamic, err := i.dynamic.InspectWheel(ctx, artifact, wheel)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewCompositeInspectionReport([]domain.InspectionReport{static, dynamic})
}

// InspectGraph performs the PyPI-specific static-all and closure-aware dynamic
// phases for one already acquired exact dependency graph. Application owns the
// Inspection Runs, Evidence recording, and Policy decisions.
func (i *CompositeInspector) InspectGraph(ctx context.Context, graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if i == nil || i.static == nil {
		return nil, errors.New("PyPI static inspector is unavailable")
	}
	nodes := graph.Nodes()
	if len(nodes) == 0 || len(acquired) != len(nodes) {
		return nil, errors.New("PyPI graph inspection requires every acquired dependency")
	}
	states := make(map[domain.DependencyNodeID]graphStaticState, len(nodes))
	reports := make(map[domain.DependencyNodeID]domain.InspectionReport, len(nodes))
	staticReady := true
	for _, dependency := range nodes {
		artifact, ok := acquired[dependency.Node()]
		if !ok || !matchesLockedDependency(dependency, artifact) {
			return nil, errors.New("PyPI graph inspection acquired dependency binding is invalid")
		}
		state, report, err := i.inspectGraphStatic(ctx, artifact)
		if err != nil {
			return nil, err
		}
		states[dependency.Node()] = state
		reports[dependency.Node()] = report
		if !state.dynamic || report.Execution().Status() != domain.ExecutionCompleted || len(report.Findings()) != 0 {
			staticReady = false
		}
	}
	// A blocked or incomplete static result must not be made executable as a
	// dependency of another node's dynamic import fixture.
	if !staticReady {
		return reports, nil
	}
	if i.dynamic == nil {
		return nil, errors.New("PyPI dynamic inspector is unavailable")
	}
	for _, dependency := range nodes {
		artifact := acquired[dependency.Node()]
		closure, err := graphClosure(graph, dependency.Node(), acquired)
		if err != nil {
			return nil, err
		}
		if err := validateRuntimeClosure(ctx, closure, graph, states); err != nil {
			return nil, err
		}
		dynamic, err := i.dynamic.InspectWheelWithClosure(ctx, artifact, states[dependency.Node()].wheel, closure)
		if err != nil {
			return nil, err
		}
		report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{reports[dependency.Node()], dynamic})
		if err != nil {
			return nil, err
		}
		reports[dependency.Node()] = report
	}
	return reports, nil
}

func (i *CompositeInspector) inspectGraphStatic(ctx context.Context, artifact domain.AcquiredArtifact) (graphStaticState, domain.InspectionReport, error) {
	if artifact.Identity().Variant() == "sdist" {
		report, err := i.static.Inspect(ctx, artifact)
		return graphStaticState{}, report, err
	}
	wheel, report, err := i.static.InspectWheel(ctx, artifact)
	return graphStaticState{wheel: wheel, dynamic: true}, report, err
}

func matchesLockedDependency(dependency domain.LockedDependency, artifact domain.AcquiredArtifact) bool {
	locked, actual := dependency.Artifact().Identity(), artifact.Identity()
	if locked.Source() != actual.Source() || locked.Name() != actual.Name() || locked.Version() != actual.Version() || locked.Variant() != actual.Variant() || artifact.Digest().String() == "" || artifact.ContentHandle() == "" {
		return false
	}
	declared, ok := artifact.DeclaredIntegrity()
	return ok && declared == dependency.Artifact().DeclaredIntegrity()
}

func graphClosure(graph domain.LockedDependencyGraph, target domain.DependencyNodeID, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) ([]domain.AcquiredArtifact, error) {
	adjacent := make(map[domain.DependencyNodeID][]domain.DependencyNodeID)
	for _, edge := range graph.Edges() {
		adjacent[edge.From()] = append(adjacent[edge.From()], edge.To())
	}
	seen := map[domain.DependencyNodeID]bool{}
	queue := []domain.DependencyNodeID{target}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		if _, ok := acquired[current]; !ok {
			return nil, errors.New("PyPI dependency closure contains an unknown node")
		}
		seen[current] = true
		queue = append(queue, adjacent[current]...)
	}
	closure := make([]domain.AcquiredArtifact, 0, len(seen))
	for _, dependency := range graph.Nodes() {
		if !seen[dependency.Node()] {
			continue
		}
		artifact := acquired[dependency.Node()]
		if !matchesLockedDependency(dependency, artifact) || (artifact.Identity().Variant() != "wheel" && artifact.Identity().Variant() != "derived-wheel") {
			return nil, errors.New("PyPI dependency closure contains an invalid artifact")
		}
		closure = append(closure, artifact)
	}
	if len(closure) == 0 {
		return nil, errors.New("PyPI dependency closure is empty")
	}
	return closure, nil
}

func validateRuntimeClosure(ctx context.Context, closure []domain.AcquiredArtifact, graph domain.LockedDependencyGraph, states map[domain.DependencyNodeID]graphStaticState) error {
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	if policy.MaxGraphCompressed() <= 0 || policy.MaxGraphUncompressed() <= 0 || policy.RuntimeTmpfs() <= 0 {
		return errors.New("PyPI dependency closure resource policy is invalid")
	}
	limit := uint64(policy.MaxGraphCompressed())
	var compressed uint64
	for _, artifact := range closure {
		if artifact.SizeBytes() > limit || compressed > limit-artifact.SizeBytes() {
			return errors.New("PyPI dependency closure compressed resource budget exceeds bound")
		}
		compressed += artifact.SizeBytes()
	}
	var expanded int64
	var files int64
	closureSet := make(map[domain.ResolvedArtifactIdentity]bool, len(closure))
	for _, artifact := range closure {
		closureSet[artifact.Identity()] = true
	}
	for _, dependency := range graph.Nodes() {
		state := states[dependency.Node()]
		if !closureSet[dependency.Artifact().Identity()] || !state.dynamic {
			continue
		}
		count, size, ok := state.footprint()
		if !ok || expanded > int64(^uint64(0)>>1)-size || files > int64(^uint64(0)>>1)-count {
			return errors.New("PyPI dependency closure static resource accounting is invalid")
		}
		expanded += size
		files += count
	}
	if expanded > policy.MaxGraphUncompressed() || expanded > policy.RuntimeTmpfs() || files < 0 || compressed > uint64(policy.RuntimeTmpfs()-expanded) {
		return errors.New("PyPI dependency closure runtime resource budget exceeds bound")
	}
	return nil
}

func hasBlockingFinding(report domain.InspectionReport) bool { return len(report.Findings()) != 0 }
