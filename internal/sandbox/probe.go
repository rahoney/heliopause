// Package sandbox owns Linux dynamic-inspection runtime capability checks.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

const (
	minimumDockerEngine = "29.6.0"
	gVisorRelease       = "release-20260810.0"
	gVisorRuntimeName   = "runsc-trace"
	nodeImageReference  = "node:22.23.1-slim@sha256:6c74791e557ce11fc957704f6d4fe134a7bc8d6f5ca4403205b2966bd488f6b3"
)

// Capability describes whether this host can run M3's gVisor backend.
type Capability struct {
	Available      bool
	LimitationCode string
}

// Executor isolates process lookup and execution for probe tests.
type Executor interface {
	LookPath(string) (string, error)
	Output(context.Context, string, ...string) ([]byte, error)
}

// Probe checks the local host without starting a sandbox or downloading an image.
func Probe(ctx context.Context) (Capability, error) {
	return probe(ctx, runtime.GOOS, systemExecutor{})
}

func probe(ctx context.Context, operatingSystem string, executor Executor) (Capability, error) {
	if ctx == nil {
		return Capability{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return Capability{}, err
	}
	limitation, err := probeGVisorRuntime(ctx, operatingSystem, executor, "M3_LINUX_ONLY", "M3_RUNTIME_UNAVAILABLE", "M3_RUNTIME_VERSION_UNSUPPORTED")
	if err != nil || limitation != "" {
		return Capability{LimitationCode: limitation}, err
	}
	image, err := executor.Output(ctx, "docker", "image", "inspect", nodeImageReference, "--format", "{{.Id}}")
	if err != nil || strings.TrimSpace(string(image)) == "" {
		return Capability{LimitationCode: "M3_IMAGE_UNAVAILABLE"}, nil
	}
	return Capability{Available: true}, nil
}

func probeGVisorRuntime(ctx context.Context, operatingSystem string, executor Executor, linuxOnly, unavailable, unsupported string) (string, error) {
	if ctx == nil {
		return "", errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if operatingSystem != "linux" {
		return linuxOnly, nil
	}
	if executor == nil {
		return "", errors.New("runtime probe executor is required")
	}
	for _, binary := range []string{"docker", "runsc"} {
		if _, err := executor.LookPath(binary); err != nil {
			return unavailable, nil
		}
	}
	dockerVersion, err := executor.Output(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return unavailable, nil
	}
	if !atLeastVersion(strings.TrimSpace(string(dockerVersion)), minimumDockerEngine) {
		return unsupported, nil
	}
	runscVersion, err := executor.Output(ctx, "runsc", "--version")
	if err != nil || !strings.Contains(string(runscVersion), gVisorRelease) {
		return unsupported, nil
	}
	runtimeRegistration, err := executor.Output(ctx, "docker", "info", "--format", "{{json (index .Runtimes \"runsc-trace\")}}")
	if err != nil || strings.TrimSpace(string(runtimeRegistration)) == "" || strings.Contains(string(runtimeRegistration), "<no value>") {
		return unavailable, nil
	}
	return "", nil
}

func atLeastVersion(actual, minimum string) bool {
	var actualMajor, actualMinor, actualPatch, minimumMajor, minimumMinor, minimumPatch int
	if _, err := fmt.Sscanf(actual, "%d.%d.%d", &actualMajor, &actualMinor, &actualPatch); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(minimum, "%d.%d.%d", &minimumMajor, &minimumMinor, &minimumPatch); err != nil {
		return false
	}
	return actualMajor > minimumMajor || actualMajor == minimumMajor && (actualMinor > minimumMinor || actualMinor == minimumMinor && actualPatch >= minimumPatch)
}

type systemExecutor struct{}

func (systemExecutor) LookPath(binary string) (string, error) { return exec.LookPath(binary) }
func (systemExecutor) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	return exec.CommandContext(ctx, binary, arguments...).Output()
}

func (systemExecutor) RunInput(ctx context.Context, input io.Reader, binary string, arguments ...string) error {
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Stdin = input
	return command.Run()
}

func (systemExecutor) RunOutput(ctx context.Context, output io.Writer, binary string, arguments ...string) error {
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Stdout = output
	return command.Run()
}

func (systemExecutor) RunDiscard(ctx context.Context, binary string, arguments ...string) error {
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Run()
}
