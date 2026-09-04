// Package sandbox owns Linux dynamic-inspection runtime capability checks.
package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const (
	gVisorRuntimeName   = "runsc-trace"
	registeredRunscPath = "/usr/libexec/heliopause/runsc"
)

var (
	gVisorRelease      = runtimeidentity.GVisorRelease
	nodeImageReference = runtimeidentity.NodeImageReference
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

// TrustedExecutor is the complete process capability injected by production
// bootstrap. It carries no method for environment or arbitrary shell control.
type TrustedExecutor interface {
	Executor
	CommandRunner
	inputCommandRunner
	discardCommandRunner
	RunOutput(context.Context, io.Writer, string, ...string) error
}

// Probe checks the local host without starting a sandbox or downloading an image.
func Probe(ctx context.Context) (Capability, error) {
	return probe(ctx, runtime.GOOS, nil)
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
	image, err := executor.Output(ctx, "docker", "image", "inspect", runtimeidentity.NodeImageReference, "--format", "{{.Id}}")
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
	if _, err := executor.LookPath("docker"); err != nil {
		return unavailable, nil
	}
	dockerVersion, err := executor.Output(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return unavailable, nil
	}
	if !atLeastVersion(strings.TrimSpace(string(dockerVersion)), runtimeidentity.DockerMinimumEngine) {
		return unsupported, nil
	}
	runtimeRegistration, err := executor.Output(ctx, "docker", "info", "--format", "{{json (index .Runtimes \"runsc-trace\")}}")
	if err != nil {
		return unavailable, nil
	}
	registeredPath, err := parseRegisteredRunscPath(runtimeRegistration)
	if err != nil || registeredPath != registeredRunscPath {
		return unavailable, nil
	}
	runscVersion, err := executor.Output(ctx, registeredPath, "--version")
	if err != nil || !strings.Contains(string(runscVersion), runtimeidentity.GVisorRelease) {
		return unsupported, nil
	}
	runscTraceMeta, err := executor.Output(ctx, registeredPath, "trace", "metadata")
	if err != nil || VerifyPatchCapability(string(runscTraceMeta)) != nil {
		return unsupported, nil
	}
	return "", nil
}

func parseRegisteredRunscPath(body []byte) (string, error) {
	var registration struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &registration); err != nil || !filepath.IsAbs(registration.Path) || filepath.Clean(registration.Path) != registration.Path {
		return "", errors.New("runsc-trace registration has no canonical absolute executable identity")
	}
	return registration.Path, nil
}

// RequiredObservationPoints defines the observation point schemas required by
// M12-001 filesystem attribution.
var RequiredObservationPoints = []string{
	"syscall/open_result",
	"sentry/mount_topology_snapshot",
	"sentry/mount_topology_mutation",
}

// VerifyPatchCapability confirms that runsc trace metadata advertises all
// required HAA filesystem-observation capabilities.
func VerifyPatchCapability(traceMetadata string) error {
	for _, point := range RequiredObservationPoints {
		if !strings.Contains(traceMetadata, "Name: "+point) && !strings.Contains(traceMetadata, point) {
			return fmt.Errorf("missing required observation point %q", point)
		}
	}
	return nil
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
