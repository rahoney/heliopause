package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const uvProjectRuntimeControlName = "uv-project-runtime.json"

// UVProjectResolver reuses the existing fixed native session and independent
// source reader. Its result is unapproved selection, never Host installation.
type UVProjectResolver struct{ native *UVResolver }

func NewUVProjectResolver(native *UVResolver) (*UVProjectResolver, error) {
	if native == nil || native.reader == nil {
		return nil, errors.New("UV project resolver requires fixed native infrastructure")
	}
	return &UVProjectResolver{native}, nil
}

func (r *UVProjectResolver) Close() error {
	if r == nil || r.native == nil {
		return nil
	}
	return r.native.Close()
}

func uvProjectRuntimeControl() (domain.ProjectControlFile, error) {
	body, err := json.Marshal(struct {
		Version       int
		Runtime, Lock string
	}{1, "uv-project:" + UVResolverRuntimeIdentity(), runtimeidentity.RuntimeLockSHA256})
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	return domain.NewProjectControlFile(uvProjectRuntimeControlName, body, true)
}

func NewUVProjectOriginalControls(input artifactpypi.UVProjectInput) ([]domain.ProjectControlFile, error) {
	controls, err := input.Controls()
	if err != nil {
		return nil, err
	}
	runtime, err := uvProjectRuntimeControl()
	if err != nil {
		return nil, err
	}
	export, err := domain.NewProjectControlFile(artifactpypi.UVProjectExportControl, nil, false)
	if err != nil {
		return nil, err
	}
	view, err := domain.NewProjectControlFile(artifactpypi.UVSelectionLockControlName, nil, false)
	if err != nil {
		return nil, err
	}
	return append(controls, runtime, export, view), nil
}

func parseUVProjectOriginalControls(controls []domain.ProjectControlFile) (artifactpypi.UVProjectInput, error) {
	if len(controls) != 6 {
		return artifactpypi.UVProjectInput{}, errors.New("UV project requires six frozen original controls")
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, control := range controls {
		if _, seen := byName[control.Name()]; seen {
			return artifactpypi.UVProjectInput{}, errors.New("UV project original controls repeat")
		}
		byName[control.Name()] = control
	}
	runtime, err := uvProjectRuntimeControl()
	if err != nil || !byName[runtime.Name()].Present() || !bytes.Equal(byName[runtime.Name()].Body(), runtime.Body()) {
		return artifactpypi.UVProjectInput{}, errors.New("UV project current runtime differs")
	}
	for _, name := range []string{artifactpypi.UVProjectExportControl, artifactpypi.UVSelectionLockControlName} {
		control, present := byName[name]
		if !present || control.Present() || len(control.Body()) != 0 {
			return artifactpypi.UVProjectInput{}, errors.New("UV project original contains a selected export or view")
		}
	}
	input, err := artifactpypi.ParseUVProjectControls([]domain.ProjectControlFile{byName[artifactpypi.UVProjectInputControl], byName[artifactpypi.UVNativeLockControl], byName[artifactpypi.UVProjectIntentControl]})
	if err != nil {
		return artifactpypi.UVProjectInput{}, err
	}
	if input.Intent() != artifactpypi.UVProjectSync && input.Intent() != artifactpypi.UVProjectLockUpgrade && input.Intent() != artifactpypi.UVProjectAdd {
		return artifactpypi.UVProjectInput{}, errors.New("UV project selection intent is unsupported")
	}
	if body, present := input.NativeLock(); present {
		if _, err := artifactpypi.ParseUVNativeProjectLock(input, body); err != nil {
			return artifactpypi.UVProjectInput{}, err
		}
	}
	return input, nil
}

func (r *UVProjectResolver) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if r == nil || r.native == nil || ctx == nil || selector != "" || !install.Valid() || install.Mode() != domain.InstallPythonVenv {
		return domain.ProjectDependencyUpdate{}, errors.New("UV project requires a frozen public virtual-project selection")
	}
	if err := ctx.Err(); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	input, err := parseUVProjectOriginalControls(original)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	var export artifactpypi.UVProjectExport
	native, err := r.native.withUVNativeSession(ctx, func(ctx context.Context, session uvNativeSession) (uvNativeOutput, error) {
		var output uvNativeOutput
		var err error
		export, output, err = r.materializeProject(ctx, input, session)
		return output, err
	})
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	provenance, err := export.Controls()
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	selected := append([]domain.ProjectControlFile(nil), original...)
	for i := range selected {
		for _, control := range provenance {
			if selected[i].Name() == control.Name() {
				selected[i] = control
			}
		}
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	wheels, err := export.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	var snapshot domain.ProjectDependencySnapshot
	err = r.native.reader.withUVArchiveInputs(ctx, wheels, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		var err error
		snapshot, err = export.BindProject(install, selected, inputs, pages, target, artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits())
		return err
	})
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	return domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
}

func (r *UVProjectResolver) materializeProject(ctx context.Context, input artifactpypi.UVProjectInput, session uvNativeSession) (artifactpypi.UVProjectExport, uvNativeOutput, error) {
	reader := r.native.reader
	transport := reader.runner.(inputCommandRunner)
	files := map[string][]byte{artifactpypi.UVProjectInputControl: input.Body()}
	if body, present := input.NativeLock(); present {
		files[artifactpypi.UVNativeLockControl] = body
	}
	if err := introduceUVProjectFiles(ctx, transport, session.container, files, nil); err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	for name, expected := range files {
		body, err := readUVProjectControl(ctx, reader.runner, session.container, name)
		if err != nil || !bytes.Equal(body, expected) {
			return artifactpypi.UVProjectExport{}, uvNativeOutput{}, errors.Join(errors.New("UV project introduced input changed"), err)
		}
	}
	common := append(uvNativeCommonArguments(), "--project", uvNativeProjectRoot)
	lockArgs := []string{"lock"}
	if input.Intent() == artifactpypi.UVProjectLockUpgrade {
		lockArgs = append(lockArgs, "--upgrade")
	} else if input.Intent() == artifactpypi.UVProjectAdd {
		lockArgs = []string{"add", input.Selector(), "--no-sync"}
	}
	lockArgs = append(lockArgs, common...)
	lockArgs = append(lockArgs, "--no-sources")
	if _, err := reader.runner.Output(ctx, "docker", boundaryExecArguments(session.container, boundaryLaunchMode, append([]string{uvToolGuestPath}, lockArgs...)...)...); err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, uvReaderFailure{"native UV project selection failed", err}
	}
	manifest, err := readUVProjectControl(ctx, reader.runner, session.container, artifactpypi.UVProjectInputControl)
	if err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	selectedInput, err := input.SelectManifest(manifest)
	if err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	files[artifactpypi.UVProjectInputControl] = manifest
	lockBody, err := readUVProjectControl(ctx, reader.runner, session.container, artifactpypi.UVNativeLockControl)
	if err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	lock, err := artifactpypi.ParseUVNativeProjectLock(selectedInput, lockBody)
	if err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	args := append([]string{"export", "--format", "pylock.toml", "--frozen", "--no-emit-project", "--no-default-groups", "--output-file", uvNativeProjectRoot + "/pylock.toml"}, common...)
	if _, err := reader.runner.Output(ctx, "docker", boundaryExecArguments(session.container, boundaryLaunchMode, append([]string{uvToolGuestPath}, args...)...)...); err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, uvReaderFailure{"native UV project export failed", err}
	}
	exportBody, err := readUVProjectControl(ctx, reader.runner, session.container, artifactpypi.UVProjectExportControl)
	if err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	export, err := lock.ParseTargetExport(exportBody)
	if err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	if input.Intent() == artifactpypi.UVProjectAdd {
		if err := input.ValidateAddSelection(selectedInput, export.Packages()); err != nil {
			return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
		}
	}
	files[artifactpypi.UVNativeLockControl] = lockBody
	for name, expected := range files {
		body, err := readUVProjectControl(ctx, reader.runner, session.container, name)
		if err != nil || !bytes.Equal(body, expected) {
			return artifactpypi.UVProjectExport{}, uvNativeOutput{}, errors.Join(errors.New("UV project native inputs changed during export"), err)
		}
	}
	files[artifactpypi.UVProjectExportControl] = exportBody
	controls, err := export.Controls()
	if err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	view := controls[3].Body()
	if err := introduceUVProjectFiles(ctx, transport, session.container, files, view); err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	syncArgs := append([]string{"pip", "sync", pythonSitePath + "/pylock.toml", "--target", pythonSitePath + "/site", "--link-mode", "copy"}, uvNativeCommonArguments()...)
	if _, err := reader.runner.Output(ctx, "docker", boundaryExecArguments(session.container, boundaryLaunchMode, append([]string{uvToolGuestPath}, syncArgs...)...)...); err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, uvReaderFailure{"native UV project wheel materialization failed", err}
	}
	if err := session.output.verifyAttachments(ctx, reader.runner, map[string]bool{session.container: false}); err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	expected := map[string][]byte{}
	for name, body := range files {
		expected["project/"+uvProjectFileName(name)] = body
	}
	output, err := readUVNativeTar(ctx, reader.runner, session.container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		return scanUVCompleteNativeOutputWithInput(input, session.runtime.PythonVersion, expected)
	})
	if err != nil || !bytes.Equal(output.lock, view) {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, errors.Join(errors.New("UV project frozen output changed during materialization"), err)
	}
	if err := session.output.verifyAttachments(ctx, reader.runner, map[string]bool{session.container: false}); err != nil {
		return artifactpypi.UVProjectExport{}, uvNativeOutput{}, err
	}
	return export, output, nil
}
