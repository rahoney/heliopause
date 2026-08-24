package sandbox

import (
	"context"
	"errors"
	"runtime"
	"strings"
)

const (
	pythonImageReference = "python:3.14.7-slim-bookworm@sha256:23c59390fc717bf09f9336908199a0ae75d9c4264bf296123f94ad772fea3b52"
	pythonRuntimeVersion = "3.14.7"
	pipRuntimeVersion    = "26.2.1"

	pythonInterpreterTag = "cp314"
	pythonABITag         = "cp314"
	pythonPlatformTag    = "manylinux_2_36_x86_64"
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
		PythonVersion:  pythonRuntimeVersion,
		PipVersion:     pipRuntimeVersion,
		InterpreterTag: pythonInterpreterTag,
		ABITag:         pythonABITag,
		PlatformTag:    pythonPlatformTag,
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
