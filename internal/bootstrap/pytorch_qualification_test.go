package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This explicit plan belongs only to the trusted qualification binary. Normal
// installs neither read its environment selector nor infer inputs from errors.
const cudaBindingsNumPyPlan = "cuda-bindings-numpy"

func qualificationPrerequisite(resolution domain.DependencyResolution, profile, version, plan string) (inspectionPrerequisiteRequest, []byte, error) {
	var empty inspectionPrerequisiteRequest
	if plan != cudaBindingsNumPyPlan || (profile != "cu126" && profile != "cu130" && profile != "cu132") {
		return empty, nil, errors.New("unknown or non-CUDA qualification plan")
	}
	source, ok := artifactpypi.PyTorchProfile(profile)
	if !ok {
		return empty, nil, errors.New("qualification source unavailable")
	}
	python := sandbox.PinnedPythonRuntime()
	expectedRuntime := "python:" + python.PythonVersion + ";pip:" + python.PipVersion + ";target:" + python.InterpreterTag + "/" + python.ABITag + "/" + python.PlatformTag + ";source:" + source.Name()
	if resolution.RuntimeIdentity() != expectedRuntime {
		return empty, nil, errors.New("qualification metadata runtime differs from selected owner")
	}
	graph := resolution.Graph()
	var target domain.ResolvedArtifact
	found, rootFound := 0, false
	for _, node := range graph.Nodes() {
		artifact := node.Artifact()
		if node.Node() == graph.Primary() {
			rootFound = artifact.Identity().Source() == source.Source() && artifact.Identity().Name() == "torch" && artifact.Identity().Version() == version
		}
		if artifact.Identity().Name() == "numpy" {
			return empty, nil, errors.New("qualification auxiliary would replace an original dependency")
		}
		if artifact.Identity().Name() != "cuda-bindings" {
			continue
		}
		if node.Node() == graph.Primary() || artifact.Identity().Source() != artifactpypi.PublicPyPIProfile().Source() {
			return empty, nil, errors.New("qualification target must be a public non-root dependency")
		}
		target = artifact
		found++
	}
	if !rootFound || found != 1 || !strings.HasPrefix(target.DeclaredIntegrity(), "sha256:") {
		return empty, nil, errors.New("qualification root/unique target metadata is unavailable")
	}
	request := inspectionPrerequisiteRequest{
		TargetSHA256: strings.TrimPrefix(target.DeclaredIntegrity(), "sha256:"),
		Source:       "pypi", Project: "numpy", Version: "2.4.6",
		Filename: "numpy-2.4.6-cp314-cp314-manylinux_2_27_x86_64.manylinux_2_28_x86_64.whl",
		SHA256:   "a2c306dea656c12c68f51f4cea133cbe78ca7435eb28c735eac1d3ebe73be6e8", Reason: "BROADER_MODULE_PROBES",
	}
	_, _, pythonTags, abiTags, _, err := artifactpypi.ParseWheelFilename(request.Filename)
	if err != nil || !slices.Contains(pythonTags, python.InterpreterTag) || !slices.Contains(abiTags, python.ABITag) {
		return empty, nil, errors.New("qualification auxiliary pin is incompatible with the selected Python ABI")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return empty, nil, err
	}
	if _, err := parseInspectionPrerequisiteRequest(string(body)); err != nil {
		return empty, nil, err
	}
	if err := validateInspectionPrerequisiteTarget(request, graph); err != nil {
		return empty, nil, err
	}
	receipt, err := json.MarshalIndent(map[string]any{
		"schema_version": 1, "plan": plan, "profile": profile, "torch_version": version,
		"runtime": resolution.RuntimeIdentity(), "resolution_sha256": resolution.LockfileDigest().String(),
		"target_project": target.Identity().Name(), "target_version": target.Identity().Version(),
		"target_source": target.Identity().Source().String(), "target_url": target.AcquisitionLocator(),
		"target_declared_integrity": target.DeclaredIntegrity(), "explicit_input": request,
		"expected_identity_origin": "official source resolver metadata, before main intake; never payload bytes or artifact output",
	}, "", "  ")
	return request, receipt, err
}

// PreparePyTorchQualification is exported only from a _test.go file so the
// external integration test can reuse the real composition, not a second resolver.
func PreparePyTorchQualification(ctx context.Context, profile, version, targetPath, plan string) (raw string, receipt []byte, resultErr error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || plan != cudaBindingsNumPyPlan || (profile != "cu126" && profile != "cu130" && profile != "cu132") {
		return "", nil, errors.New("explicit supported Linux CUDA qualification plan required")
	}
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		return "", nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, executor.Close()) }()
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		return "", nil, err
	}
	observer, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(start, remote, output)
	})
	if err != nil {
		return "", nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, observer.Close()) }()
	resolver, err := pypiInstallDependencyResolver(runtime.GOOS, runtime.GOARCH, executor, observer.Observer())
	if err != nil {
		return "", nil, err
	}
	source, _ := artifactpypi.PyTorchProfile(profile)
	resourceContext, err := artifactpypi.ContextWithResourcePolicy(ctx, source)
	if err != nil {
		return "", nil, err
	}
	reference, err := artifactpypi.ParseReferenceForSource("torch@"+version, source.Source())
	if err != nil {
		return "", nil, err
	}
	target, err := domain.NewInstallTarget(targetPath)
	if err != nil {
		return "", nil, err
	}
	install, err := domain.NewInstallContext(target)
	if err != nil {
		return "", nil, err
	}
	resolution, err := resolver.ResolveDependencies(resourceContext, reference, install)
	if err != nil {
		return "", nil, fmt.Errorf("qualification metadata resolution: %w", err)
	}
	request, receipt, err := qualificationPrerequisite(resolution, profile, version, plan)
	if err != nil {
		return "", nil, err
	}
	body, err := json.Marshal(request)
	return string(body), receipt, err
}
