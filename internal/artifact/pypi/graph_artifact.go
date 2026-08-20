package pypi

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// DependencyGraphResolver is the isolated PyPI resolver boundary required to
// turn a caller reference into the exact primary Artifact for direct inspect.
// It intentionally exposes only generic Domain resolution values.
type DependencyGraphResolver interface {
	ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error)
}

// GraphArtifact adapts the primary member of a complete isolated resolution
// graph to the ordinary Artifact port; secondary dependencies never become a
// hidden direct-inspect input.
type GraphArtifact struct {
	resolver DependencyGraphResolver
	intake   *Intake
}

func NewGraphArtifact(resolver DependencyGraphResolver, intake *Intake) (*GraphArtifact, error) {
	if resolver == nil || intake == nil {
		return nil, errors.New("PyPI graph Artifact requires resolver and intake")
	}
	return &GraphArtifact{resolver: resolver, intake: intake}, nil
}

func (a *GraphArtifact) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if a == nil || a.resolver == nil || ctx == nil || reference.Source().String() != "pypi" {
		return domain.ResolvedArtifact{}, errors.New("PyPI graph Artifact resolve request is invalid")
	}
	target, _ := domain.NewInstallTarget("/tmp/heliopause-pypi-inspect")
	install, _ := domain.NewInstallContext(target)
	resolution, err := a.resolver.ResolveDependencies(ctx, reference, install)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	graph := resolution.Graph()
	for _, node := range graph.Nodes() {
		if node.Node() == graph.Primary() {
			return node.Artifact(), nil
		}
	}
	return domain.ResolvedArtifact{}, errors.New("PyPI resolver primary Artifact is absent")
}

func (a *GraphArtifact) Acquire(ctx context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if a == nil || a.intake == nil {
		return domain.AcquiredArtifact{}, errors.New("PyPI graph Artifact intake is unavailable")
	}
	return a.intake.Acquire(ctx, runID, resolved)
}
