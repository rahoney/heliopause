package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const cargoResolutionDescriptor = "cargo:crates.io;sparse:index.crates.io;metadata-v1;frozen-lock;operation-private"

type cargoProjectSelection struct {
	metadata     []byte
	metadataRoot string
	files        map[string][]byte
	snapshot     domain.ProjectDependencySnapshot
}

func (r *CargoResolver) metadataRoot(workspace string) string {
	if _, isolated := r.runner.(*IsolatedCargoRunner); isolated {
		return cargoResolverGuestProject
	}
	return workspace
}

func privateCargoResolverEnvironment() (environment []string, cleanup func() error, resultErr error) {
	home, err := os.MkdirTemp("", "haa-cargo-home-")
	if err != nil {
		return nil, nil, errors.New("create private Cargo resolver home")
	}
	cleanup = func() error {
		if err := os.RemoveAll(home); err != nil {
			return errors.New("private Cargo home cleanup failed")
		}
		return nil
	}
	environment, err = CargoResolverEnvironmentForHome(home)
	if err != nil {
		return nil, nil, errors.Join(err, cleanup())
	}
	return environment, cleanup, nil
}

func cargoRunnerFailure(message string, cause error) error {
	var trusted *isolatedCargoResolverFailure
	if errors.As(cause, &trusted) {
		return fmt.Errorf("%s: %w", message, trusted)
	}
	return errors.New(message)
}

func (r *CargoResolver) ResolveProjectDependencies(ctx context.Context, install domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	selection, err := r.resolveLockedProject(ctx, install)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	return selection.snapshot, nil
}

func (r *CargoResolver) resolveLockedProject(ctx context.Context, install domain.InstallContext) (selection cargoProjectSelection, resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || !install.Valid() {
		return selection, errors.New("valid Cargo project request is required")
	}
	defer func() {
		if resultErr != nil {
			selection = cargoProjectSelection{}
		}
	}()
	source, err := captureCargoProject(ctx, install.Target().String())
	if err != nil {
		return selection, err
	}
	defer func() { resultErr = errors.Join(resultErr, source.close()) }()
	frozen := source.files()
	if len(frozen["Cargo.lock"]) == 0 {
		return selection, errors.New("locked Cargo project requires Cargo.lock")
	}
	workspace, cleanup, err := source.privateWorkspace(ctx)
	if err != nil {
		return selection, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanup()) }()
	privateManifest, privateLock, err := readCargoControlFiles(workspace)
	if err != nil || !bytes.Equal(privateManifest, frozen["Cargo.toml"]) || !bytes.Equal(privateLock, frozen["Cargo.lock"]) {
		return selection, errors.New("private Cargo controls differ from frozen input")
	}
	environment, cleanupEnv, err := privateCargoResolverEnvironment()
	if err != nil {
		return selection, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanupEnv()) }()
	metadata, err := r.runner.RunCargo(ctx, workspace, environment, "metadata", "--locked", "--format-version", "1")
	if err != nil {
		return selection, cargoRunnerFailure("cargo metadata resolution failed", err)
	}
	if err := source.verify(ctx); err != nil {
		return selection, errors.New("cargo project changed during resolution")
	}
	private, err := captureCargoProject(ctx, workspace)
	if err != nil {
		return selection, err
	}
	files := private.files()
	if err := private.close(); err != nil {
		return selection, err
	}
	if !sameCargoProjectData(frozen, files, false) {
		return selection, errors.New("cargo metadata changed private source or controls")
	}
	root := r.metadataRoot(workspace)
	snapshot, err := artifactcargo.BuildProjectSnapshot(install, metadata, files["Cargo.toml"], files["Cargo.lock"], root, PinnedCargoRuntime().ImageReference, files)
	if err != nil {
		return selection, err
	}
	return cargoProjectSelection{metadata, root, files, snapshot}, nil
}

func sameCargoProjectData(before, after map[string][]byte, allowSelectedControls bool) bool {
	for name, body := range before {
		if allowSelectedControls && (name == "Cargo.toml" || name == "Cargo.lock") {
			continue
		}
		other, present := after[name]
		if !present || !bytes.Equal(body, other) {
			return false
		}
	}
	for name := range after {
		if allowSelectedControls && (name == "Cargo.toml" || name == "Cargo.lock") {
			continue
		}
		if _, present := before[name]; !present {
			return false
		}
	}
	return true
}

func cargoFrozenControls(files map[string][]byte) ([]domain.ProjectControlFile, error) {
	var result []domain.ProjectControlFile
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		body, present := files[name]
		if name == "Cargo.toml" && (!present || artifactcargo.ValidateProjectManifest(body, name) != nil) {
			return nil, errors.New("cargo manifest is absent or unsupported")
		}
		control, err := domain.NewProjectControlFile(name, body, present)
		if err != nil {
			return nil, err
		}
		result = append(result, control)
	}
	return result, nil
}

func (r *CargoResolver) ResolveProjectDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, original []domain.ProjectControlFile) (update domain.ProjectDependencyUpdate, resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || !install.Valid() || reference.Source() != artifactcargo.Source() {
		return update, errors.New("valid Cargo project update request is required")
	}
	if _, err := artifactcargo.ParseReference(reference.Locator()); err != nil {
		return update, err
	}
	defer func() {
		if resultErr != nil {
			update = domain.ProjectDependencyUpdate{}
		}
	}()
	source, err := captureCargoProject(ctx, install.Target().String())
	if err != nil {
		return update, err
	}
	defer func() { resultErr = errors.Join(resultErr, source.close()) }()
	frozen := source.files()
	current, err := cargoFrozenControls(frozen)
	if err != nil || !artifactcargo.EqualControls(original, current) {
		return update, errors.New("cargo original controls changed before selection")
	}
	workspace, cleanup, err := source.privateWorkspace(ctx)
	if err != nil {
		return update, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanup()) }()
	environment, cleanupEnv, err := privateCargoResolverEnvironment()
	if err != nil {
		return update, err
	}
	defer func() { resultErr = errors.Join(resultErr, cleanupEnv()) }()
	parts := strings.Split(reference.Locator(), "@")
	if _, err := r.runner.RunCargo(ctx, workspace, environment, "add", parts[0]+"@="+parts[1]); err != nil {
		return update, cargoRunnerFailure("private Cargo selection failed", err)
	}
	selectedSource, err := captureCargoProject(ctx, workspace)
	if err != nil {
		return update, err
	}
	selectedFiles := selectedSource.files()
	if err := selectedSource.close(); err != nil {
		return update, err
	}
	if len(selectedFiles["Cargo.lock"]) == 0 || !sameCargoProjectData(frozen, selectedFiles, true) {
		return update, errors.New("cargo selection changed source outside root controls or omitted lock")
	}
	metadata, err := r.runner.RunCargo(ctx, workspace, environment, "metadata", "--locked", "--format-version", "1")
	if err != nil {
		return update, cargoRunnerFailure("cargo selected metadata failed", err)
	}
	if err := source.verify(ctx); err != nil {
		return update, errors.New("cargo project changed during resolution")
	}
	private, err := captureCargoProject(ctx, workspace)
	if err != nil {
		return update, err
	}
	files := private.files()
	if err := private.close(); err != nil {
		return update, err
	}
	if !sameCargoProjectData(selectedFiles, files, false) {
		return update, errors.New("cargo metadata changed selected frozen source or controls")
	}
	selected, err := cargoFrozenControls(files)
	if err != nil {
		return update, err
	}
	root := r.metadataRoot(workspace)
	snapshot, err := artifactcargo.BuildProjectSnapshot(install, metadata, files["Cargo.toml"], files["Cargo.lock"], root, PinnedCargoRuntime().ImageReference, files)
	if err != nil {
		return update, err
	}
	records, edges, _, err := artifactcargo.ParseLockedMetadata(metadata, files["Cargo.lock"], root)
	if err != nil {
		return update, err
	}
	graph, err := artifactcargo.BuildLockedGraph(reference, records, edges)
	if err != nil {
		return update, err
	}
	resolution, err := domain.NewDependencyResolution(graph, cargoResolutionDescriptor, snapshot.GraphDigest())
	if err != nil {
		return update, err
	}
	return domain.NewProjectDependencyUpdate(original, selected, snapshot, resolution)
}
