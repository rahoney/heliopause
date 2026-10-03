package sandbox

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

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
func (r *GoModuleResolver) ResolveProjectDependencies(ctx context.Context, installContext domain.InstallContext) (snapshot domain.ProjectDependencySnapshot, resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || !installContext.Valid() {
		return domain.ProjectDependencySnapshot{}, errors.New("valid Go project resolver request is required")
	}
	defer func() {
		if resultErr != nil {
			snapshot = domain.ProjectDependencySnapshot{}
		}
	}()
	project := filepath.Clean(installContext.Target().String())
	if !filepath.IsAbs(project) || project == "/" {
		return domain.ProjectDependencySnapshot{}, errors.New("go project path is invalid")
	}
	goMod, goSum, err := readGoProjectControlFiles(project)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	workspace, cleanupWorkspace, err := privateGoProjectWorkspace(goMod, goSum)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanupWorkspace()) }()
	environment, cleanup, err := privateGoResolverEnvironment()
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanup()) }()
	jsonBody, err := r.runner.RunGo(ctx, workspace, environment, "mod", "download", "-json", "all")
	if err != nil {
		return domain.ProjectDependencySnapshot{}, errors.New("go module download failed")
	}
	records, err := artifactgomodule.ParseDownloadJSON(jsonBody)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	graphBody, err := r.runner.RunGo(ctx, workspace, environment, "mod", "graph")
	if err != nil {
		return domain.ProjectDependencySnapshot{}, errors.New("go module graph failed")
	}
	currentMod, currentSum, currentErr := readGoProjectControlFiles(project)
	if currentErr != nil || string(currentMod) != string(goMod) || string(currentSum) != string(goSum) {
		return domain.ProjectDependencySnapshot{}, errors.New("go project changed during resolution")
	}
	return artifactgomodule.BuildProjectSnapshot(installContext, records, graphBody, goMod, goSum)
}

func (r *GoModuleResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (resolution domain.DependencyResolution, resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || reference.Source() != artifactgomodule.Source() || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Go module resolver request is required")
	}
	defer func() {
		if resultErr != nil {
			resolution = domain.DependencyResolution{}
		}
	}()
	project := filepath.Clean(installContext.Target().String())
	if !filepath.IsAbs(project) || project == "/" {
		return domain.DependencyResolution{}, errors.New("go project path is invalid")
	}
	originalMod, originalSum, err := readGoProjectControlFiles(project)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	workspace, cleanupWorkspace, err := privateGoProjectWorkspace(originalMod, originalSum)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanupWorkspace()) }()
	environment, cleanup, err := privateGoResolverEnvironment()
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanup()) }()
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
	selectedMod, selectedSum, err := readGoProjectControlFiles(workspace)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := artifactgomodule.BuildLockedGraph(reference, records, graphBody, selectedMod)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digest, err := artifactgomodule.FreezeResolutionDigest(records, graphBody, selectedMod, selectedSum)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, "go:proxy.golang.org;sumdb:sum.golang.org;cache:operation-private;goenv:off;gowork:off;toolchain:local", digest)
}

func readGoProjectControlFiles(project string) ([]byte, []byte, error) {
	info, err := os.Lstat(project)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, errors.New("go project root is not an exact directory")
	}
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, nil, errors.New("go project root is unavailable")
	}
	defer root.Close()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return nil, nil, errors.New("go project root identity changed")
	}
	goMod, err := readGoControlFile(root, "go.mod")
	if err != nil {
		return nil, nil, errors.New("go project go.mod is unavailable")
	}
	if err := artifactgomodule.ValidateProjectMod(goMod); err != nil {
		return nil, nil, err
	}
	goSum, err := readGoControlFile(root, "go.sum")
	if err != nil {
		return nil, nil, errors.New("go project go.sum is unavailable")
	}
	return goMod, goSum, nil
}

func readGoControlFile(root *os.Root, name string) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > artifactgomodule.MaxProjectControlBytes {
		return nil, errors.New("go control file is not bounded regular content")
	}
	// A raced replacement with a FIFO must fail the regular-file check rather
	// than blocking before it. Root also confines any symlink resolution.
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("open Go control file")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("go control file identity changed")
	}
	body, err := io.ReadAll(io.LimitReader(file, artifactgomodule.MaxProjectControlBytes+1))
	after, statErr := file.Stat()
	current, currentErr := root.Lstat(name)
	if err != nil || statErr != nil || currentErr != nil || !os.SameFile(info, current) || len(body) != int(info.Size()) || after.Size() != info.Size() || after.ModTime() != info.ModTime() {
		return nil, errors.New("go control file changed during bounded read")
	}
	return body, nil
}

func privateGoProjectWorkspace(goMod, goSum []byte) (string, func() error, error) {
	workspace, err := os.MkdirTemp("", "haa-go-resolve-project-")
	if err != nil {
		return "", nil, errors.New("create private Go project workspace")
	}
	for name, body := range map[string][]byte{"go.mod": goMod, "go.sum": goSum} {
		if len(body) == 0 || os.WriteFile(filepath.Join(workspace, name), body, 0o600) != nil {
			_ = os.RemoveAll(workspace)
			return "", nil, errors.New("copy Go project control files")
		}
	}
	return workspace, func() error {
		if err := os.RemoveAll(workspace); err != nil {
			return errors.New("dispose private Go project workspace failed")
		}
		return nil
	}, nil
}

func privateGoResolverEnvironment() ([]string, func() error, error) {
	cache, err := os.MkdirTemp("", "haa-go-module-cache-")
	if err != nil {
		return nil, nil, errors.New("create private Go module cache")
	}
	environment, err := artifactgomodule.ResolverEnvironmentForCache(cache)
	if err != nil {
		_ = os.RemoveAll(cache)
		return nil, nil, err
	}
	return environment, func() error {
		if err := os.RemoveAll(cache); err != nil {
			return errors.New("dispose private Go module cache failed")
		}
		return nil
	}, nil
}
