package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
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

// ResolveProjectDependencies freezes the complete current-project module
// state. It is separate from exact user-requested module resolution and never
// creates an arbitrary primary artifact.
func (r *GoModuleResolver) ResolveProjectDependencies(ctx context.Context, installContext domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	if r == nil || r.runner == nil || ctx == nil || !installContext.Valid() {
		return domain.ProjectDependencySnapshot{}, errors.New("valid Go project resolver request is required")
	}
	project := filepath.Clean(installContext.Target().String())
	if !filepath.IsAbs(project) || project == "/" {
		return domain.ProjectDependencySnapshot{}, errors.New("go project path is invalid")
	}
	goMod, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil || len(goMod) == 0 {
		return domain.ProjectDependencySnapshot{}, errors.New("go project go.mod is unavailable")
	}
	goSum, err := os.ReadFile(filepath.Join(project, "go.sum"))
	if err != nil || len(goSum) == 0 {
		return domain.ProjectDependencySnapshot{}, errors.New("go project go.sum is unavailable")
	}
	environment, cleanup, err := privateGoResolverEnvironment()
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	defer cleanup()
	jsonBody, err := r.runner.RunGo(ctx, project, environment, "mod", "download", "-json", "all")
	if err != nil {
		return domain.ProjectDependencySnapshot{}, errors.New("go module download failed")
	}
	records, err := artifactgomodule.ParseDownloadJSON(jsonBody)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	graphBody, err := r.runner.RunGo(ctx, project, environment, "mod", "graph")
	if err != nil {
		return domain.ProjectDependencySnapshot{}, errors.New("go module graph failed")
	}
	currentMod, modErr := os.ReadFile(filepath.Join(project, "go.mod"))
	currentSum, sumErr := os.ReadFile(filepath.Join(project, "go.sum"))
	if modErr != nil || sumErr != nil || string(currentMod) != string(goMod) || string(currentSum) != string(goSum) {
		return domain.ProjectDependencySnapshot{}, errors.New("go project changed during resolution")
	}
	return artifactgomodule.BuildProjectSnapshot(installContext, records, graphBody, goMod, goSum)
}

func (r *GoModuleResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if r == nil || r.runner == nil || ctx == nil || reference.Source() != artifactgomodule.Source() || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Go module resolver request is required")
	}
	project := filepath.Clean(installContext.Target().String())
	if !filepath.IsAbs(project) || project == "/" {
		return domain.DependencyResolution{}, errors.New("go project path is invalid")
	}
	originalMod, originalSum, err := readGoProjectControlFiles(project)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	workspace, cleanupWorkspace, err := privateGoProjectWorkspace(project)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	defer cleanupWorkspace()
	environment, cleanup, err := privateGoResolverEnvironment()
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	defer cleanup()
	if _, err := r.runner.RunGo(ctx, workspace, environment, "get", reference.Locator()); err != nil {
		return domain.DependencyResolution{}, errors.New("private go module selection failed")
	}
	jsonBody, err := r.runner.RunGo(ctx, workspace, environment, "mod", "download", "-json", "all")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("go module download failed")
	}
	records, err := artifactgomodule.ParseDownloadJSON(jsonBody)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graphBody, err := r.runner.RunGo(ctx, workspace, environment, "mod", "graph")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("go module graph failed")
	}
	currentMod, currentSum, currentErr := readGoProjectControlFiles(project)
	if currentErr != nil || string(currentMod) != string(originalMod) || string(currentSum) != string(originalSum) {
		return domain.DependencyResolution{}, errors.New("go project changed during resolution")
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

func readGoProjectControlFiles(project string) ([]byte, []byte, error) {
	goMod, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil || len(goMod) == 0 {
		return nil, nil, errors.New("go project go.mod is unavailable")
	}
	goSum, err := os.ReadFile(filepath.Join(project, "go.sum"))
	if err != nil || len(goSum) == 0 {
		return nil, nil, errors.New("go project go.sum is unavailable")
	}
	return goMod, goSum, nil
}

func privateGoProjectWorkspace(project string) (string, func(), error) {
	workspace, err := os.MkdirTemp("", "haa-go-resolve-project-")
	if err != nil {
		return "", nil, errors.New("create private Go project workspace")
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		body, readErr := os.ReadFile(filepath.Join(project, name))
		if readErr != nil || len(body) == 0 || os.WriteFile(filepath.Join(workspace, name), body, 0o600) != nil {
			_ = os.RemoveAll(workspace)
			return "", nil, errors.New("copy Go project control files")
		}
	}
	return workspace, func() { _ = os.RemoveAll(workspace) }, nil
}

func privateGoResolverEnvironment() ([]string, func(), error) {
	cache, err := os.MkdirTemp("", "haa-go-module-cache-")
	if err != nil {
		return nil, nil, errors.New("create private Go module cache")
	}
	environment, err := artifactgomodule.ResolverEnvironmentForCache(cache)
	if err != nil {
		_ = os.RemoveAll(cache)
		return nil, nil, err
	}
	return environment, func() { _ = os.RemoveAll(cache) }, nil
}
