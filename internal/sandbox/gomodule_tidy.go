package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func validateGoProjectSource(files map[string][]byte) error {
	if err := artifactgo.ValidateProjectMod(files["go.mod"]); err != nil {
		return err
	}
	for name := range files {
		if name == "go.work" || name == "go.work.sum" || strings.HasSuffix(name, "/go.mod") {
			return errors.New("go tidy workspaces and nested modules are unsupported")
		}
	}
	return nil
}

// ResolveProjectTidy operates on the held source witness, not the user's live
// directory. The existing public proxy/SumDB and complete graph checks follow.
func (r *GoModuleResolver) ResolveProjectTidy(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, source *GoProjectSource) (update domain.ProjectDependencyUpdate, resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || !install.Valid() || source == nil {
		return update, errors.New("go tidy requires guarded frozen source")
	}
	runner, ok := r.runner.(interface {
		RunGoProject(context.Context, string, []string, ...string) ([]byte, error)
	})
	if !ok {
		return update, errors.New("go tidy source runner is unavailable")
	}
	originalMod, originalSum, err := goFrozenControlBodies(original)
	if err != nil {
		return update, err
	}
	files := source.Files()
	if !bytes.Equal(files["go.mod"], originalMod) || !bytes.Equal(files["go.sum"], originalSum) {
		return update, errors.New("go tidy source differs from guarded controls")
	}
	if err := source.Verify(ctx); err != nil {
		return update, err
	}
	workspace, cleanup, err := source.privateWorkspace(ctx)
	if err != nil {
		return update, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, cleanup())
		if resultErr != nil {
			update = domain.ProjectDependencyUpdate{}
		}
	}()
	// The existing isolated runner publishes a complete private control pair.
	// Normalize only the disposable copy; original absence stays in the guard.
	if _, present := files["go.sum"]; !present {
		file, err := os.OpenFile(filepath.Join(workspace, "go.sum"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return update, errors.New("create private Go tidy checksum control")
		}
		if err := file.Close(); err != nil {
			return update, err
		}
	}
	environment, cleanupEnv, err := privateGoResolverEnvironment()
	if err != nil {
		return update, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanupEnv()) }()
	if _, err := runner.RunGoProject(ctx, workspace, environment, "mod", "tidy"); err != nil {
		return update, goModuleRunnerFailure("isolated Go tidy failed", err)
	}
	selected, err := readGoFrozenControls(workspace)
	if err != nil {
		return update, err
	}
	selectedMod, selectedSum, err := goFrozenControlBodies(selected)
	if err != nil {
		return update, err
	}
	if err := artifactgo.ValidateTidyControls(originalMod, selectedMod); err != nil {
		return update, err
	}
	jsonBody, err := runner.RunGoProject(ctx, workspace, environment, "mod", "download", "-json", "all")
	if err != nil {
		return update, goModuleRunnerFailure("Go tidy selected download failed", err)
	}
	records, err := artifactgo.ParseProjectDownloadJSON(jsonBody)
	if err != nil {
		return update, err
	}
	graph, err := runner.RunGoProject(ctx, workspace, environment, "mod", "graph")
	if err != nil {
		return update, goModuleRunnerFailure("Go tidy selected graph failed", err)
	}
	current, err := readGoFrozenControls(workspace)
	if err != nil || !sameGoFrozenControls(selected, current) {
		return update, errors.New("go tidy controls changed after selection")
	}
	if err := source.Verify(ctx); err != nil {
		return update, err
	}
	snapshot, err := artifactgo.BuildProjectSnapshot(install, records, graph, selectedMod, selectedSum)
	if err != nil {
		return update, err
	}
	return domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
}
