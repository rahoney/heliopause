package sandbox

import (
	"context"
	"errors"
	"runtime"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const (
	pythonImageReference = runtimeidentity.PythonImageReference
)

// PythonRuntime identifies the immutable runtime used by M5 resolver and
// Promotion paths. The version values are verified before any future pip use.
type PythonRuntime struct {
	ImageReference string
	PythonVersion  string
	PipVersion     string
	InterpreterTag string
	ABITag         string
	PlatformTag    string
}

// PinnedPythonRuntime returns M5's only automatic Linux amd64 target basis.
func PinnedPythonRuntime() PythonRuntime {
	return PythonRuntime{
		ImageReference: pythonImageReference,
		PythonVersion:  runtimeidentity.PythonVersion,
		PipVersion:     runtimeidentity.PipVersion,
		InterpreterTag: runtimeidentity.PythonInterpreterTag,
		ABITag:         runtimeidentity.PythonABITag,
		PlatformTag:    runtimeidentity.PythonPlatformTag,
	}
}

// PythonCapability describes whether this host can start M5's pinned trusted
// runtime. It does not execute pip or resolve a project.
type PythonCapability struct {
	Available      bool
	LimitationCode string
	Runtime        PythonRuntime
}

// ProbePython checks M5's Linux amd64 gVisor prerequisites and the presence of
// the exact immutable Python runtime image without downloading it.
func ProbePython(ctx context.Context) (PythonCapability, error) {
	return probePython(ctx, runtime.GOOS, runtime.GOARCH, nil)
}

func probePython(ctx context.Context, operatingSystem, architecture string, executor Executor) (PythonCapability, error) {
	locked := PinnedPythonRuntime()
	if ctx == nil {
		return PythonCapability{Runtime: locked}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return PythonCapability{Runtime: locked}, err
	}
	if architecture != "amd64" {
		return PythonCapability{LimitationCode: "M5_PYPI_LINUX_AMD64_ONLY", Runtime: locked}, nil
	}
	limitation, err := probeGVisorRuntime(ctx, operatingSystem, executor, "M5_PYPI_LINUX_AMD64_ONLY", "M5_PYPI_RUNTIME_UNAVAILABLE", "M5_PYPI_RUNTIME_VERSION_UNSUPPORTED")
	if err != nil || limitation != "" {
		return PythonCapability{LimitationCode: limitation, Runtime: locked}, err
	}
	image, err := executor.Output(ctx, "docker", "image", "inspect", pythonImageReference, "--format", "{{.Id}}")
	if err != nil || strings.TrimSpace(string(image)) == "" {
		return PythonCapability{LimitationCode: "M5_PYPI_IMAGE_UNAVAILABLE", Runtime: locked}, nil
	}
	return PythonCapability{Available: true, Runtime: locked}, nil
}
