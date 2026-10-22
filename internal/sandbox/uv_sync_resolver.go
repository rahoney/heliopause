package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const uvSyncRuntimeControlName = "uv-sync-runtime.json"

func uvSyncRuntimeControl() (domain.ProjectControlFile, error) {
	body, err := json.Marshal(struct {
		Version  int    `json:"version"`
		Resolver string `json:"resolver_runtime"`
		Lock     string `json:"runtime_lock_sha256"`
	}{1, UVResolverRuntimeIdentity(), runtimeidentity.RuntimeLockSHA256})
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	return domain.NewProjectControlFile(uvSyncRuntimeControlName, body, true)
}

// NewUVSyncOriginalControls prepares unapproved transaction data only. The
// actual native producer still verifies the fixed tool/runtime/observer pins.
func NewUVSyncOriginalControls(input artifactpypi.UVSyncInput) ([]domain.ProjectControlFile, error) {
	if !input.Valid() || input.ShouldSkip() || input.Python() != PinnedPythonRuntime().PythonVersion {
		return nil, errors.New("UV sync requires positive source for its pinned target")
	}
	controls, err := input.Controls()
	if err != nil {
		return nil, err
	}
	runtime, err := uvSyncRuntimeControl()
	if err != nil {
		return nil, err
	}
	absent, err := domain.NewProjectControlFile(artifactpypi.UVSelectionLockControlName, nil, false)
	if err != nil {
		return nil, err
	}
	return append(controls, runtime, absent), nil
}

func parseUVSyncOriginalControls(controls []domain.ProjectControlFile) (artifactpypi.UVSyncInput, error) {
	if len(controls) != 4 {
		return artifactpypi.UVSyncInput{}, errors.New("UV sync original controls are incomplete")
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, control := range controls {
		if _, seen := byName[control.Name()]; seen {
			return artifactpypi.UVSyncInput{}, errors.New("UV sync original controls repeat")
		}
		switch control.Name() {
		case artifactpypi.UVSyncSourceControlName, artifactpypi.UVSyncIntentControlName, uvSyncRuntimeControlName:
			if !control.Present() {
				return artifactpypi.UVSyncInput{}, errors.New("UV sync source/runtime control is absent")
			}
		case artifactpypi.UVSelectionLockControlName:
			if control.Present() || len(control.Body()) != 0 {
				return artifactpypi.UVSyncInput{}, errors.New("UV sync original already contains a selected lock")
			}
		default:
			return artifactpypi.UVSyncInput{}, errors.New("UV sync original control is unknown")
		}
		byName[control.Name()] = control
	}
	runtime, err := uvSyncRuntimeControl()
	if err != nil || !bytes.Equal(runtime.Body(), byName[uvSyncRuntimeControlName].Body()) {
		return artifactpypi.UVSyncInput{}, errors.New("UV sync runtime control differs from current pins")
	}
	input, err := artifactpypi.ParseUVSyncControls([]domain.ProjectControlFile{byName[artifactpypi.UVSyncSourceControlName], byName[artifactpypi.UVSyncIntentControlName]}, PinnedPythonRuntime().PythonVersion)
	if err != nil || input.ShouldSkip() {
		return artifactpypi.UVSyncInput{}, errors.New("UV sync original source is invalid or skips selection")
	}
	return input, nil
}

// ResolveProjectLockUpdate implements the existing primary-free selection
// port. This returns no Policy, ownership, staging or deletion authority.
func (r *UVResolver) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if r == nil || r.reader == nil || ctx == nil || selector != "" || !install.Valid() || install.Mode() != domain.InstallPythonVenv || r.reader.profile.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return domain.ProjectDependencyUpdate{}, errors.New("UV sync requires complete frozen public Python input")
	}
	if err := ctx.Err(); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	input, err := parseUVSyncOriginalControls(original)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	native, err := r.withUVNativeSession(ctx, func(ctx context.Context, session uvNativeSession) (uvNativeOutput, error) {
		return r.materializeUVSyncSelection(ctx, input, session)
	})
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	lockControl, err := domain.NewProjectControlFile(artifactpypi.UVSelectionLockControlName, native.lock, true)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	selected := append([]domain.ProjectControlFile(nil), original...)
	for i := range selected {
		if selected[i].Name() == lockControl.Name() {
			selected[i] = lockControl
		}
	}
	snapshot, err := r.reader.bindUVNativeProjectSelection(ctx, install, selected, input, native)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	return domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
}

func (r *UVResolver) materializeUVSyncSelection(ctx context.Context, input artifactpypi.UVSyncInput, session uvNativeSession) (uvNativeOutput, error) {
	if !input.Valid() || input.ShouldSkip() || input.Python() != session.runtime.PythonVersion {
		return uvNativeOutput{}, errors.New("UV sync native source differs from target")
	}
	reader, container := r.reader, session.container
	transport := reader.runner.(inputCommandRunner)
	if err := introduceUVSyncInput(ctx, transport, container, input); err != nil {
		return uvNativeOutput{}, err
	}
	common := uvNativeCommonArguments()
	if input.SourceFormat() == artifactpypi.UVSyncRequirements {
		compile := append([]string{"pip", "compile", "-", "--no-deps", "--format", "pylock.toml", "--output-file", pythonSitePath + "/pylock.toml", "--python-version", session.runtime.PythonVersion, "--python-platform", session.platform}, common...)
		if err := transport.RunInput(ctx, bytes.NewReader(input.Body()), "docker", boundaryInputExecArguments(container, boundaryLaunchMode, append([]string{uvToolGuestPath}, compile...)...)...); err != nil {
			return uvNativeOutput{}, uvReaderFailure{"native UV complete compile failed", err}
		}
	}
	body, err := readUVNativeLock(ctx, reader.runner, container)
	if err != nil {
		return uvNativeOutput{}, err
	}
	lock, err := artifactpypi.ParseUVProjectLock(body, session.runtime.PythonVersion)
	if err != nil {
		return uvNativeOutput{}, err
	}
	if err := input.ValidateSelection(lock); err != nil {
		return uvNativeOutput{}, err
	}
	sync := append([]string{"pip", "sync", pythonSitePath + "/pylock.toml", "--target", pythonSitePath + "/site", "--link-mode", "copy"}, common...)
	if _, err := reader.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, append([]string{uvToolGuestPath}, sync...)...)...); err != nil {
		return uvNativeOutput{}, uvReaderFailure{"native UV complete materialization failed", err}
	}
	if err := session.output.verifyAttachments(ctx, reader.runner, map[string]bool{container: false}); err != nil {
		return uvNativeOutput{}, err
	}
	result, err := readUVNativeTar(ctx, reader.runner, container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		return scanUVCompleteNativeOutput(input, session.runtime.PythonVersion)
	})
	if err != nil {
		return uvNativeOutput{}, err
	}
	if !bytes.Equal(result.lock, body) {
		return uvNativeOutput{}, errors.New("UV complete native lock changed during materialization")
	}
	if err := session.output.verifyAttachments(ctx, reader.runner, map[string]bool{container: false}); err != nil {
		return uvNativeOutput{}, err
	}
	return result, nil
}

// Fresh controller-created site and optional immutable pylock only. Existing
// named-output tar transport/ordinary ownership is shared with UV selection.
func introduceUVSyncInput(ctx context.Context, runner inputCommandRunner, container string, input artifactpypi.UVSyncInput) error {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) || !input.Valid() || input.ShouldSkip() {
		return errors.New("UV sync input transport is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var buffer bytes.Buffer
	archive := tar.NewWriter(&buffer)
	if err := archive.WriteHeader(&tar.Header{Name: "site/", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
		return err
	}
	if input.SourceFormat() == artifactpypi.UVSyncPylock {
		body := input.Body()
		if err := archive.WriteHeader(&tar.Header{Name: "pylock.toml", Typeflag: tar.TypeReg, Mode: 0400, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
			return err
		}
		if _, err := archive.Write(body); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	return runner.RunInput(ctx, bytes.NewReader(buffer.Bytes()), "docker", "cp", "--archive", "-", container+":"+pythonSitePath)
}

func (r *PyPIResolver) bindUVNativeProjectSelection(ctx context.Context, install domain.InstallContext, controls []domain.ProjectControlFile, input artifactpypi.UVSyncInput, native uvNativeOutput) (domain.ProjectDependencySnapshot, error) {
	runtime := PinnedPythonRuntime()
	lock, err := artifactpypi.ParseUVProjectLock(native.lock, runtime.PythonVersion)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if err := input.ValidateSelection(lock); err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selected, err := lock.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	var snapshot domain.ProjectDependencySnapshot
	err = r.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		var err error
		snapshot, err = lock.BindProject(install, controls, inputs, pages, target, artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits())
		return err
	})
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	return snapshot, nil
}
