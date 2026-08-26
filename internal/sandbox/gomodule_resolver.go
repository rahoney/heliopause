package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// GoModuleRunner is the trusted process boundary for an isolated Go tool.
// Implementations must execute the verified absolute Go binary with exactly
// the supplied working directory and environment.
type GoModuleRunner interface {
	RunGo(context.Context, string, []string, ...string) ([]byte, error)
}

// GoModuleResolver converts canonical proxy/SumDB Go output to a generic
// exact dependency graph. It never accepts direct VCS or ambient proxy state.
type GoModuleResolver struct{ runner GoModuleRunner }

func NewGoModuleResolver(runner GoModuleRunner) (*GoModuleResolver, error) {
	if runner == nil {
		return nil, errors.New("go module resolver requires trusted Go runner")
	}
	return &GoModuleResolver{runner: runner}, nil
}

func (r *GoModuleResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if r == nil || r.runner == nil || ctx == nil || reference.Source() != artifactgomodule.Source() || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Go module resolver request is required")
	}
	project := filepath.Clean(installContext.Target().String())
	if !filepath.IsAbs(project) || project == "/" {
		return domain.DependencyResolution{}, errors.New("go project path is invalid")
	}
	if err := artifactgomodule.ValidateResolverEnvironment(artifactgomodule.ResolverEnvironment()); err != nil {
		return domain.DependencyResolution{}, err
	}
	environment := artifactgomodule.ResolverEnvironment()
	jsonBody, err := r.runner.RunGo(ctx, project, environment, "mod", "download", "-json", "all")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("go module download failed")
	}
	records, err := artifactgomodule.ParseDownloadJSON(jsonBody)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graphBody, err := r.runner.RunGo(ctx, project, environment, "mod", "graph")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("go module graph failed")
	}
	graph, err := artifactgomodule.BuildLockedGraph(reference, records, graphBody)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	lockDigest := sha256.Sum256(append(append([]byte{}, jsonBody...), graphBody...))
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(lockDigest[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, "go:proxy.golang.org;sumdb:sum.golang.org;env:"+strings.Join(environment, ";"), digest)
}
