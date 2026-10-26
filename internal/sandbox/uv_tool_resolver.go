package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// UVToolSelection is fresh native selection and independently bound archives.
// It grants no inspection/Policy, target ownership or Host installation status.
type UVToolSelection struct {
	resolution domain.DependencyResolution
	controls   []domain.ProjectControlFile
}

func (s UVToolSelection) Resolution() domain.DependencyResolution { return s.resolution }
func (s UVToolSelection) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}

// ResolveToolInstall never opens an existing tool environment. Its fixed native
// session creates a fresh private tool home; original-environment upgrades need
// a separate authenticated pre-query guard and are not routed here.
func (r *UVResolver) ResolveToolInstall(ctx context.Context, requirement artifactpypi.UpgradeRequirement) (UVToolSelection, error) {
	if r == nil || r.reader == nil || ctx == nil || requirement.Reference().Source() != artifactpypi.PublicPyPIProfile().Source() {
		return UVToolSelection{}, errors.New("UV tool selection requires one supported public requirement")
	}
	if err := ctx.Err(); err != nil {
		return UVToolSelection{}, err
	}
	canonical, err := artifactpypi.ParseUpgradeRequirement(requirement.Requirement())
	if err != nil || canonical.Requirement() != requirement.Requirement() {
		return UVToolSelection{}, errors.New("UV tool requirement is not canonical")
	}
	var tool uvToolNativeOutput
	native, err := r.withUVNativeSessionForToolInstall(ctx, func(ctx context.Context, session uvNativeSession) (uvNativeOutput, error) {
		args := []string{uvToolGuestPath, "tool", "install", requirement.Requirement(), "--no-sources", "--link-mode", "copy"}
		args = append(args, uvNativeCommonArguments()...)
		exec := boundaryExecArguments(session.container, boundaryLaunchMode, args...)
		if _, err := r.reader.runner.Output(ctx, "docker", exec...); err != nil {
			return uvNativeOutput{}, uvReaderFailure{"native UV tool install selection failed", err}
		}
		first, err := readUVToolNativeOutput(ctx, r.reader.runner, session.container, requirement.Project(), nil)
		if err != nil {
			return uvNativeOutput{}, err
		}
		if !first.receipt.Requirement().SameConstraints(requirement) {
			return uvNativeOutput{}, errors.New("UV tool changed the requested root constraint")
		}
		frozen, reference, err := uvToolFrozenRequirements(first, requirement)
		if err != nil {
			return uvNativeOutput{}, err
		}
		compile := []string{uvToolGuestPath, "pip", "compile", "-", "--format", "pylock.toml", "--output-file", pythonSitePath + "/pylock.toml", "--python-version", session.runtime.PythonVersion, "--python-platform", session.platform}
		compile = append(compile, uvNativeCommonArguments()...)
		input := r.reader.runner.(inputCommandRunner)
		if err := input.RunInput(ctx, strings.NewReader(frozen), "docker", boundaryInputExecArguments(session.container, boundaryLaunchMode, compile...)...); err != nil {
			return uvNativeOutput{}, uvReaderFailure{"export exact UV tool versions failed", err}
		}
		lock, err := readUVNativeLock(ctx, r.reader.runner, session.container)
		if err != nil {
			return uvNativeOutput{}, err
		}
		if _, err := artifactpypi.ParseUVLock(reference, lock, session.runtime.PythonVersion); err != nil {
			return uvNativeOutput{}, err
		}
		final, err := readUVToolNativeOutput(ctx, r.reader.runner, session.container, requirement.Project(), lock)
		if err != nil {
			return uvNativeOutput{}, err
		}
		if !bytes.Equal(first.receipt.Body(), final.receipt.Body()) || !reflect.DeepEqual(first.controls, final.controls) || !reflect.DeepEqual(first.links, final.links) {
			return uvNativeOutput{}, errors.New("UV tool receipt or complete native selection changed during export")
		}
		if err := session.output.verifyAttachments(ctx, r.reader.runner, map[string]bool{session.container: false}); err != nil {
			return uvNativeOutput{}, err
		}
		tool = final
		return uvNativeOutput{lock: lock, controls: final.controls}, nil
	}, true)
	if err != nil {
		return UVToolSelection{}, err
	}
	_, reference, err := uvToolFrozenRequirements(tool, requirement)
	if err != nil {
		return UVToolSelection{}, err
	}
	lock, err := artifactpypi.ParseUVLock(reference, native.lock, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return UVToolSelection{}, err
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selected, err := lock.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return UVToolSelection{}, err
	}
	controls, err := uvToolSelectionControls(requirement, tool)
	if err != nil {
		return UVToolSelection{}, err
	}
	var result UVToolSelection
	err = r.reader.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		graph, err := tool.receipt.BindWheels(lock, inputs, pages, target, artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits())
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		var digests []string
		for _, c := range controls {
			digests = append(digests, c.Name()+":"+c.Digest().String())
		}
		body, err := json.Marshal(digests)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		if err != nil {
			return err
		}
		resolution, err := domain.NewDependencyResolution(graph, "uv-tool-install:"+UVResolverRuntimeIdentity(), digest)
		if err != nil {
			return err
		}
		result = UVToolSelection{resolution: resolution, controls: controls}
		return nil
	})
	if err != nil {
		return UVToolSelection{}, err
	}
	return result, nil
}

// Only the tool session gets these fixed controller values. Direct exec keeps
// its existing nonce/admission grammar; no Host environment is forwarded.
func uvToolCreateArguments(base []string) ([]string, error) {
	index := -1
	for i, value := range base {
		if value == pythonImageReference {
			if index != -1 {
				return nil, errors.New("UV tool container image is ambiguous")
			}
			index = i
		}
	}
	if index < 1 || len(base)-index != 4 || base[0] != "create" || base[index+1] != "/bin/sh" || base[index+2] != "-ceu" || base[index+3] != boundaryContainerCommand() {
		return nil, errors.New("UV tool container runtime differs")
	}
	result := append([]string(nil), base[:index]...)
	result = append(result, "--env", "UV_TOOL_DIR="+uvNativeToolHome, "--env", "UV_TOOL_BIN_DIR="+uvNativeToolBin, "--env", "UV_COMPILE_BYTECODE=false", "--env", "PYTHONDONTWRITEBYTECODE=1")
	return append(result, base[index:]...), nil
}

func uvToolFrozenRequirements(output uvToolNativeOutput, requirement artifactpypi.UpgradeRequirement) (string, domain.ArtifactReference, error) {
	var files []artifactpypi.InstalledMetadataFile
	for _, c := range output.controls {
		files = append(files, artifactpypi.InstalledMetadataFile{Directory: c.DistInfo, Body: c.Metadata})
	}
	input, err := artifactpypi.NewUpgradeInput(requirement, files, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return "", domain.ArtifactReference{}, err
	}
	var lines []string
	var reference domain.ArtifactReference
	for _, project := range input.Installed() {
		lines = append(lines, project.Project()+"=="+project.Version())
		if project.Project() == requirement.Project() {
			if !requirement.AcceptsVersion(project.Version()) {
				return "", domain.ArtifactReference{}, errors.New("native UV tool root violates request")
			}
			reference, err = artifactpypi.ParseReference(project.Project() + "@" + project.Version())
			if err != nil {
				return "", domain.ArtifactReference{}, err
			}
		}
	}
	if reference.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return "", domain.ArtifactReference{}, errors.New("native UV tool root is absent")
	}
	return strings.Join(lines, "\n") + "\n", reference, nil
}

func uvToolSelectionControls(requirement artifactpypi.UpgradeRequirement, output uvToolNativeOutput) ([]domain.ProjectControlFile, error) {
	intent, err := json.Marshal(struct {
		Version                                   int
		Intent, Requirement, Runtime, RuntimeLock string
	}{1, "fresh-install", requirement.Requirement(), "uv-tool-install:" + UVResolverRuntimeIdentity(), runtimeidentity.RuntimeLockSHA256})
	if err != nil {
		return nil, err
	}
	type wheelDigest struct{ DistInfo, Metadata, Wheel string }
	var wheels []wheelDigest
	for _, c := range output.controls {
		m, w := sha256.Sum256(c.Metadata), sha256.Sum256(c.Wheel)
		wheels = append(wheels, wheelDigest{c.DistInfo, hex.EncodeToString(m[:]), hex.EncodeToString(w[:])})
	}
	census, err := json.Marshal(struct {
		Version int
		Wheels  []wheelDigest
		Exposed map[string]string
	}{1, wheels, output.links})
	if err != nil {
		return nil, err
	}
	var controls []domain.ProjectControlFile
	for _, file := range []struct {
		name string
		body []byte
	}{{"uv-tool-intent.json", intent}, {"uv-tool-receipt.toml", output.receipt.Body()}, {"uv-tool-pylock.toml", output.lock}, {"uv-tool-census.json", census}} {
		c, err := domain.NewProjectControlFile(file.name, file.body, true)
		if err != nil {
			return nil, err
		}
		controls = append(controls, c)
	}
	return controls, nil
}
