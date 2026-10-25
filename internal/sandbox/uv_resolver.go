package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// UVResolver uses native UV for selection and an independent isolated Python
// reader for registry/archive binding. Neither step executes artifact code.
type UVResolver struct{ reader *PyPIResolver }

const uvResolverOwnerLabel = "io.heliopause.uv-resolver"

// UVResolverRuntimeIdentity binds the reviewed solver and target separately
// from the offline installer. No caller-supplied prefix establishes this pin.
func UVResolverRuntimeIdentity() string {
	runtime := PinnedPythonRuntime()
	return "uv:" + runtimeidentity.UVVersion + ";python:" + runtime.PythonVersion + ";target:" + runtime.InterpreterTag + "/" + runtime.ABITag + "/" + runtime.PlatformTag + ";source:pypi"
}

func NewUVResolver(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (PythonCapability, error), policy ResolverPolicyService) (*UVResolver, error) {
	reader, err := NewPyPIResolver(runner, endpoints, observer, probe, policy)
	if err != nil {
		return nil, err
	}
	return newUVResolver(reader)
}

func newUVResolver(reader *PyPIResolver) (*UVResolver, error) {
	if reader == nil || reader.runner == nil || reader.profile.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return nil, errors.New("uv resolver requires isolated public reader infrastructure")
	}
	runner := reader.runner
	if admitted, ok := runner.(*admissionAwareCommandRunner); ok {
		runner = admitted.runner
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errors.New("uv resolver requires native stdin transport")
	}
	if _, ok := runner.(streamingDockerOutput); !ok {
		return nil, errors.New("uv resolver requires bounded archive transport")
	}
	return &UVResolver{reader: reader}, nil
}

func NewLinuxUVResolverWithExecutorAndPolicy(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService) (*UVResolver, error) {
	reader, err := NewLinuxPyPIResolverWithExecutorAndPolicy(executor, observer, policy)
	if err != nil {
		return nil, err
	}
	return newUVResolver(reader)
}

func (r *UVResolver) Close() error {
	if r == nil {
		return nil
	}
	return r.reader.Close()
}

func (r *UVResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	if r == nil || r.reader == nil || ctx == nil || reference.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return domain.DependencyResolution{}, errors.New("uv resolution requires a public PyPI request")
	}
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, err
	}
	request, err := pypiRequirement(reference)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	native, err := r.resolveNativeSelection(ctx, reference, request)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return r.reader.bindUVNativeSelection(ctx, reference, native)
}

func (r *UVResolver) resolveNativeSelection(ctx context.Context, reference domain.ArtifactReference, request string) (uvNativeOutput, error) {
	return r.resolveNativeSelectionWithInput(ctx, reference, request, nil)
}

func (r *UVResolver) resolveNativeSelectionWithInput(ctx context.Context, reference domain.ArtifactReference, request string, selected *uvInstalledSelection) (uvNativeOutput, error) {
	return r.withUVNativeSession(ctx, func(ctx context.Context, session uvNativeSession) (uvNativeOutput, error) {
		return r.materializeUVSelection(ctx, reference, request, selected, session)
	})
}

type uvNativeSession struct {
	container string
	runtime   PythonRuntime
	platform  string
	output    closureVolume
}

// Both actual native consumers share the fixed tool, network/observer and
// volume lifecycle, including clearing output when any cleanup is unproven.
func (r *UVResolver) withUVNativeSession(ctx context.Context, operation func(context.Context, uvNativeSession) (uvNativeOutput, error)) (result uvNativeOutput, resultErr error) {
	return r.withUVNativeSessionForToolInstall(ctx, operation, false)
}

func (r *UVResolver) withUVNativeSessionForToolInstall(ctx context.Context, operation func(context.Context, uvNativeSession) (uvNativeOutput, error), toolInstall bool) (result uvNativeOutput, resultErr error) {
	if r == nil || r.reader == nil || ctx == nil || operation == nil {
		return uvNativeOutput{}, errors.New("uv native session is unavailable")
	}
	reader := r.reader
	capability, err := reader.probe(ctx)
	if err != nil {
		return uvNativeOutput{}, uvReaderFailure{"uv native runtime probe failed", err}
	}
	if !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return uvNativeOutput{}, errors.New("uv native runtime is unavailable")
	}
	platform, err := uvTargetPlatform(capability.Runtime)
	if err != nil {
		return uvNativeOutput{}, err
	}
	tool, err := prepareUVTool(ctx, reader.runner)
	if err != nil {
		return uvNativeOutput{}, err
	}
	addressesByName, err := reader.endpoints.Resolve(ctx, pypiResolverEndpoints)
	if err != nil {
		return uvNativeOutput{}, errors.New("uv resolver endpoint preflight failed")
	}
	addresses, hosts, err := resolverNetworkArguments(artifactpypi.PublicPyPIProfile(), addressesByName)
	if err != nil {
		return uvNativeOutput{}, errors.New("uv resolver endpoint preflight is unsafe")
	}
	policy, err := NewResolverNetworkPolicy(reader.runner, reader.policy)
	if err != nil {
		return uvNativeOutput{}, err
	}
	network, err := policy.Prepare(ctx, addresses)
	if err != nil {
		return uvNativeOutput{}, err
	}
	var runtimeVolume uvRuntimeVolume
	var output closureVolume
	var trace TraceReader
	container := ""
	name, owner := "", ""
	createAttempted := false
	defer func() {
		var cleanupErr error
		if createAttempted {
			cleanup, cancel := resolverCleanupContext()
			removeErr := removeOwnedUVResolverContainer(cleanup, reader.runner, name, owner, container)
			cancel()
			if removeErr != nil {
				cleanupErr = uvReaderFailure{fmt.Sprintf("uv resolver container cleanup unproven name=%s", name), removeErr}
			}
			if trace != nil {
				collect, done := context.WithTimeout(context.Background(), cleanupTimeout)
				_, limitation, diagnostic := collectTraceDiagnostic(collect, trace)
				done()
				if limitation != "" {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("uv resolver observation is incomplete: %s", diagnostic))
				}
			}
		}
		if output.name != "" {
			cleanup, cancel := resolverCleanupContext()
			cleanupErr = errors.Join(cleanupErr, closeUVOutputVolume(cleanup, reader.runner, &output))
			cancel()
		}
		if runtimeVolume.name != "" {
			cleanup, cancel := resolverCleanupContext()
			cleanupErr = errors.Join(cleanupErr, runtimeVolume.close(cleanup, reader.runner))
			cancel()
		}
		cleanup, cancel := resolverCleanupContext()
		cleanupErr = errors.Join(cleanupErr, policy.Close(cleanup))
		cancel()
		if cleanupErr != nil {
			result = uvNativeOutput{}
			resultErr = errors.Join(resultErr, cleanupErr)
		}
	}()
	runtimeVolume, err = createUVRuntimeVolume(ctx, reader.runner)
	if err != nil {
		return uvNativeOutput{}, err
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return uvNativeOutput{}, err
	}
	owner = session.String()
	name = "haa-uv-resolver-" + owner
	output, err = createClosureVolume(ctx, reader.runner, session.String()+"-uv-output", runtimeidentity.UVToolSHA256, artifactpypi.PublicPyPIProfile().ResourcePolicy().PromotionTmpfs())
	if err != nil {
		return uvNativeOutput{}, err
	}
	arguments, err := uvCreateArguments(network, hosts, runtimeVolume, output)
	if toolInstall && err == nil {
		arguments, err = uvToolCreateArguments(arguments)
	}
	if err != nil {
		return uvNativeOutput{}, err
	}
	listed, err := reader.runner.Output(ctx, "docker", "ps", "--all", "--quiet", "--no-trunc", "--filter", "name=^/"+name+"$")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return uvNativeOutput{}, errors.New("uv resolver name preflight is unavailable or nonempty")
	}
	arguments = append([]string{"create", "--name", name, "--label", uvResolverOwnerLabel + "=" + owner}, arguments[1:]...)
	createAttempted = true
	created, err := reader.runner.Output(ctx, "docker", arguments...)
	if err != nil || !exactObservationContainerID(strings.TrimSpace(string(created))) {
		return uvNativeOutput{}, uvReaderFailure{fmt.Sprintf("create uv resolver container identity unavailable name=%s", name), errors.Join(err, ctx.Err())}
	}
	container = strings.TrimSpace(string(created))
	if err := verifyOwnedUVResolverContainer(ctx, reader.runner, name, owner, container); err != nil {
		return uvNativeOutput{}, err
	}
	trace, err = startTrace(ctx, reader.observer, container, uvResolverProfile)
	if err != nil {
		return uvNativeOutput{}, errors.New("start uv resolver observer failed")
	}
	if _, err := reader.runner.Output(ctx, "docker", "start", container); err != nil {
		return uvNativeOutput{}, errors.New("start uv resolver container failed")
	}
	if err := awaitMountAnchors(ctx, reader.observer, container); err != nil {
		return uvNativeOutput{}, err
	}
	if err := awaitBoundaryHelper(ctx, reader.runner, container); err != nil {
		return uvNativeOutput{}, err
	}
	if err := runtimeVolume.verifyTarget(ctx, reader.runner, container); err != nil {
		return uvNativeOutput{}, err
	}
	if err := output.verifyContainerMount(ctx, reader.runner, container, false); err != nil {
		return uvNativeOutput{}, err
	}
	if err := output.verifyAttachments(ctx, reader.runner, map[string]bool{container: false}); err != nil {
		return uvNativeOutput{}, err
	}
	if err := tool.introduce(ctx, reader.runner, container); err != nil {
		return uvNativeOutput{}, err
	}
	if err := runtimeVolume.verifyTarget(ctx, reader.runner, container); err != nil {
		return uvNativeOutput{}, err
	}
	selection, cancel := context.WithTimeout(ctx, pypiResolverTimeout)
	defer cancel()
	return operation(selection, uvNativeSession{container, capability.Runtime, platform, output})
}

func (r *UVResolver) materializeUVSelection(selection context.Context, reference domain.ArtifactReference, request string, selected *uvInstalledSelection, session uvNativeSession) (result uvNativeOutput, resultErr error) {
	reader := r.reader
	container, output, platform := session.container, session.output, session.platform
	var err error
	common := uvNativeCommonArguments()
	compile := append([]string{"pip", "compile", "-", "--format", "pylock.toml", "--output-file", pythonSitePath + "/pylock.toml", "--python-version", session.runtime.PythonVersion, "--python-platform", platform}, common...)
	input := reader.runner.(inputCommandRunner)
	var expectedInput map[string][]byte
	if selected != nil {
		expectedInput = uvOriginalSelectionFiles(selected.input)
		if err := introduceUVSelectionFiles(selection, input, container, expectedInput, true); err != nil {
			return uvNativeOutput{}, err
		}
		dryRun := []string{"pip", "install", "--dry-run", "--output-format", "json", "--target", pythonSitePath + "/original"}
		dryRun = append(dryRun, common...)
		switch selected.intent {
		case artifactpypi.UVInstallSelection:
		case artifactpypi.UVUpgradeSelection:
			dryRun = append(dryRun, "--upgrade")
		case artifactpypi.UVUpgradePackageSelection:
			dryRun = append(dryRun, "--upgrade-package", selected.input.Requirement().Project())
		default:
			return uvNativeOutput{}, errors.New("UV native selection intent is unsupported")
		}
		dryRun = append(dryRun, request)
		body, err := uvBoundedSelectionOutput(selection, reader.runner, container, append([]string{uvToolGuestPath}, dryRun...))
		if err != nil {
			return uvNativeOutput{}, err
		}
		selected.proposal, err = selected.input.ParseUVSelection(body)
		if err != nil {
			return uvNativeOutput{}, err
		}
		constraints := selected.proposal.Constraints()
		if err := introduceUVSelectionFiles(selection, input, container, map[string][]byte{"constraints.txt": constraints}, false); err != nil {
			return uvNativeOutput{}, err
		}
		expectedInput["constraints.txt"] = constraints
		request, err = pypiRequirement(selected.proposal.Reference())
		if err != nil {
			return uvNativeOutput{}, err
		}
		compile = append(compile, "--constraint", pythonSitePath+"/constraints.txt")
	}
	if err := input.RunInput(selection, strings.NewReader(request+"\n"), "docker", boundaryInputExecArguments(container, boundaryLaunchMode, append([]string{uvToolGuestPath}, compile...)...)...); err != nil {
		return uvNativeOutput{}, uvReaderFailure{"native uv compile failed", err}
	}
	lockSnapshot, err := readUVNativeLock(selection, reader.runner, container)
	if err != nil {
		return uvNativeOutput{}, err
	}
	if _, err := artifactpypi.ParseUVLock(reference, lockSnapshot, session.runtime.PythonVersion); err != nil {
		return uvNativeOutput{}, err
	}
	sync := append([]string{"pip", "sync", pythonSitePath + "/pylock.toml", "--target", pythonSitePath + "/site", "--link-mode", "copy"}, common...)
	if _, err := reader.runner.Output(selection, "docker", boundaryExecArguments(container, boundaryLaunchMode, append([]string{uvToolGuestPath}, sync...)...)...); err != nil {
		return uvNativeOutput{}, uvReaderFailure{"native uv pylock materialization failed", err}
	}
	if err := output.verifyAttachments(selection, reader.runner, map[string]bool{container: false}); err != nil {
		return uvNativeOutput{}, err
	}
	if selected == nil {
		result, err = readUVNativeOutput(selection, reader.runner, container)
	} else {
		result, err = readUVNativeTar(selection, reader.runner, container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
			return scanUVNativeOutputWithInput(input, expectedInput)
		})
	}
	if err != nil {
		return uvNativeOutput{}, err
	}
	if !bytes.Equal(result.lock, lockSnapshot) {
		return uvNativeOutput{}, errors.New("uv native lock changed during materialization")
	}
	if err := output.verifyAttachments(selection, reader.runner, map[string]bool{container: false}); err != nil {
		return uvNativeOutput{}, err
	}
	return result, nil
}

func uvNativeCommonArguments() []string {
	return []string{"--no-build", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--python", "/usr/local/bin/python", "--index-url", artifactpypi.PublicPyPIProfile().IndexURL()}
}

func removeOwnedUVResolverContainer(ctx context.Context, runner CommandRunner, name, owner, expectedID string) error {
	return removeOwnedNativeResolverContainer(ctx, runner, name, owner, expectedID, nativeRuntimeUV)
}

func verifyOwnedUVResolverContainer(ctx context.Context, runner CommandRunner, name, owner, id string) error {
	return verifyOwnedNativeResolverContainer(ctx, runner, name, owner, id, nativeRuntimeUV)
}

func uvTargetPlatform(runtime PythonRuntime) (string, error) {
	if runtime != PinnedPythonRuntime() || !strings.HasPrefix(runtime.PlatformTag, "manylinux_") || !strings.HasSuffix(runtime.PlatformTag, "_x86_64") {
		return "", errors.New("uv native target platform is unsupported")
	}
	return "x86_64-" + strings.TrimSuffix(runtime.PlatformTag, "_x86_64"), nil
}
