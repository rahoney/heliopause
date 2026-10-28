package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// PipxToolSelection contains unapproved native metadata and independently bound
// original archives. It grants no inspection, target ownership or Policy status.
type PipxToolSelection struct {
	resolution domain.DependencyResolution
	controls   []domain.ProjectControlFile
}

func (s PipxToolSelection) Resolution() domain.DependencyResolution { return s.resolution }
func (s PipxToolSelection) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}

type PipxResolver struct {
	native    *UVResolver
	resources ObservationResourceClient
}

func PipxResolverRuntimeIdentity() string {
	return "pipx:" + runtimeidentity.PipxVersion + ";backend:" + UVResolverRuntimeIdentity() + ";zip:" + runtimeidentity.PipxToolSHA256
}

func NewPipxResolver(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (PythonCapability, error), policy ResolverPolicyService, resources ObservationResourceClient) (*PipxResolver, error) {
	if resources == nil {
		return nil, errors.New("pipx resolver requires typed observation resources")
	}
	native, err := NewUVResolver(runner, endpoints, observer, probe, policy)
	if err != nil {
		return nil, err
	}
	return newPipxResolver(native, resources)
}

func NewLinuxPipxResolverWithExecutorAndPolicy(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService, resources ObservationResourceClient) (*PipxResolver, error) {
	if resources == nil {
		return nil, errors.New("pipx resolver requires typed observation resources")
	}
	native, err := NewLinuxUVResolverWithExecutorAndPolicy(executor, observer, policy)
	if err != nil {
		return nil, err
	}
	return newPipxResolver(native, resources)
}

func newPipxResolver(native *UVResolver, resources ObservationResourceClient) (*PipxResolver, error) {
	if native == nil || native.reader == nil || resources == nil {
		return nil, errors.New("pipx resolver infrastructure unavailable")
	}
	raw := native.reader.runner
	if admitted, ok := raw.(*admissionAwareCommandRunner); ok {
		raw = admitted.runner
	}
	_, input := raw.(inputCommandRunner)
	_, stream := raw.(streamingDockerOutput)
	_, discard := raw.(discardCommandRunner)
	_, bounded := raw.(boundedCommandRunner)
	_, profile := native.reader.observer.(profiledTraceObserver)
	_, mounts := native.reader.observer.(mountAnchorReadyObserver)
	if !input || !stream || !discard || !bounded || !profile || !mounts {
		return nil, errors.New("pipx resolver requires complete native transport and profile observation")
	}
	return &PipxResolver{native: native, resources: resources}, nil
}

func (r *PipxResolver) Close() error {
	if r == nil || r.native == nil {
		return nil
	}
	return r.native.Close()
}

// ResolveToolInstall freezes independent original bytes before any native pipx
// query. The reader session must finish cleanup before artifact execution starts.
func (r *PipxResolver) ResolveToolInstall(ctx context.Context, requirement artifactpypi.UpgradeRequirement) (PipxToolSelection, error) {
	if r == nil || r.native == nil || r.native.reader == nil || r.resources == nil || ctx == nil || requirement.Reference().Source() != artifactpypi.PublicPyPIProfile().Source() {
		return PipxToolSelection{}, errors.New("pipx requires one supported public tool requirement")
	}
	if err := ctx.Err(); err != nil {
		return PipxToolSelection{}, err
	}
	canonical, err := artifactpypi.ParseUpgradeRequirement(requirement.Requirement())
	if err != nil || canonical.Requirement() != requirement.Requirement() {
		return PipxToolSelection{}, errors.New("pipx requirement is not canonical")
	}
	// Missing or changed installed supply fails before DNS, Docker or selection.
	tool, err := preparePipxTool(ctx)
	if err != nil {
		return PipxToolSelection{}, err
	}
	reference := requirement.Reference()
	native, err := r.native.resolveNativeSelection(ctx, reference, requirement.Requirement())
	if err != nil {
		return PipxToolSelection{}, err
	}
	lock, err := artifactpypi.ParseUVLock(reference, native.lock, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return PipxToolSelection{}, err
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selected, err := lock.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return PipxToolSelection{}, err
	}
	var frozen *pipxNativeInput
	err = r.native.reader.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		var freezeErr error
		frozen, freezeErr = freezePipxNativeInput(ctx, lock, inputs, pages)
		return freezeErr
	})
	if err != nil {
		return PipxToolSelection{}, err
	}
	reader := r.native.reader
	output, err := runPipxNativeInstall(ctx, reader.runner, reader.observer, r.resources, tool, frozen, requirement)
	if err != nil {
		return PipxToolSelection{}, err
	}
	if err := ctx.Err(); err != nil {
		return PipxToolSelection{}, err
	}
	selection, err := pipxSelectionForOutput(requirement, native.lock, frozen, output)
	if err != nil {
		return PipxToolSelection{}, err
	}
	if err := ctx.Err(); err != nil {
		return PipxToolSelection{}, err
	}
	return selection, nil
}

func pipxSelectionForOutput(requirement artifactpypi.UpgradeRequirement, lockBody []byte, frozen *pipxNativeInput, output pipxNativeOutput) (PipxToolSelection, error) {
	if !pipxNativeRequirementMatches(requirement, frozen) || !output.receipt.Valid() || !output.receipt.Requirement().SameConstraints(requirement) {
		return PipxToolSelection{}, errors.New("pipx native selection differs from frozen request")
	}
	lock, err := artifactpypi.ParseUVLock(requirement.Reference(), lockBody, PinnedPythonRuntime().PythonVersion)
	if err != nil || !reflect.DeepEqual(lock, frozen.lock) {
		return PipxToolSelection{}, errors.New("pipx native lock differs from frozen source")
	}
	graph, err := frozen.bindNativeOutput(output)
	if err != nil {
		return PipxToolSelection{}, err
	}
	runtime := "pipx-tool-install:" + PipxResolverRuntimeIdentity()
	intent, err := json.Marshal(struct {
		Version                                             int
		Intent, Requirement, Runtime, RuntimeLock, FrozenID string
	}{1, "fresh-install", requirement.Requirement(), runtime, runtimeidentity.RuntimeLockSHA256, frozen.id})
	if err != nil {
		return PipxToolSelection{}, err
	}
	type wheelDigest struct{ DistInfo, Metadata, Wheel string }
	var wheels []wheelDigest
	for _, control := range output.controls {
		m, w := sha256.Sum256(control.Metadata), sha256.Sum256(control.Wheel)
		wheels = append(wheels, wheelDigest{control.DistInfo, hex.EncodeToString(m[:]), hex.EncodeToString(w[:])})
	}
	census, err := json.Marshal(struct {
		Version int
		Wheels  []wheelDigest
		Exposed map[string]string
	}{1, wheels, output.links})
	if err != nil {
		return PipxToolSelection{}, err
	}
	var controls []domain.ProjectControlFile
	var digests []string
	for _, file := range []struct {
		name string
		body []byte
	}{{"pipx-tool-intent.json", intent}, {"pipx-tool-receipt.json", output.receipt.Body()}, {"pipx-tool-pylock.toml", lockBody}, {"pipx-tool-census.json", census}} {
		control, err := domain.NewProjectControlFile(file.name, file.body, true)
		if err != nil {
			return PipxToolSelection{}, err
		}
		controls = append(controls, control)
		digests = append(digests, control.Name()+":"+control.Digest().String())
	}
	body, err := json.Marshal(digests)
	if err != nil {
		return PipxToolSelection{}, err
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return PipxToolSelection{}, err
	}
	resolution, err := domain.NewDependencyResolution(graph, runtime, digest)
	if err != nil {
		return PipxToolSelection{}, err
	}
	return PipxToolSelection{resolution: resolution, controls: controls}, nil
}
