package sandbox

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	sandboxWallTimeout = 45 * time.Second
	cleanupTimeout     = 3 * time.Second
)

var containerIDPattern = regexp.MustCompile(`^[a-f0-9]{12,64}$`)

// CommandRunner is the narrow process boundary used by the gVisor backend.
// It deliberately exposes neither Host environment nor shell evaluation.
type CommandRunner interface {
	Output(context.Context, string, ...string) ([]byte, error)
}

// ArtifactIntroducer copies exact acquired content into a newly created Sandbox.
// Its implementation is trusted infrastructure: the untrusted process only sees
// the fixed in-container destination, never a Host path or Content Handle.
type ArtifactIntroducer interface {
	Introduce(context.Context, string, domain.AcquiredArtifact) error
}

// CapabilityProbe reports whether the locked Linux gVisor runtime is usable.
type CapabilityProbe func(context.Context) (Capability, error)

// Backend owns fresh gVisor experiments and their common inspection budget.
type Backend struct {
	runner       CommandRunner
	introducer   ArtifactIntroducer
	observer     TraceObserver
	resources    ObservationResourceClient
	probe        CapabilityProbe
	newSessionID func() (domain.SandboxSessionID, error)
	wallTimeout  time.Duration
	cleanupWait  time.Duration
}

// NewBackend constructs the M3 Linux dynamic-inspection backend.
func NewBackend(runner CommandRunner, introducer ArtifactIntroducer, observer TraceObserver, probe CapabilityProbe) (*Backend, error) {
	if runner == nil {
		return nil, errors.New("sandbox command runner is required")
	}
	if introducer == nil {
		return nil, errors.New("sandbox artifact introducer is required")
	}
	if observer == nil {
		return nil, errors.New("sandbox trace observer is required")
	}
	if probe == nil {
		return nil, errors.New("sandbox capability probe is required")
	}
	return &Backend{runner: admissionAwareRunner(runner), introducer: introducer, observer: observer, probe: probe, newSessionID: domain.NewSandboxSessionID, wallTimeout: sandboxWallTimeout, cleanupWait: cleanupTimeout}, nil
}

// Execute runs the exact artifact in each required fresh environment and
// disposes of its runtimes before returning a bounded raw result.
func (b *Backend) Execute(ctx context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	if b == nil || b.runner == nil || b.introducer == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil {
		return domain.SandboxResult{}, errors.New("sandbox backend is not configured")
	}
	if ctx == nil {
		return domain.SandboxResult{}, errors.New("context is required")
	}
	sessionID, err := b.newSessionID()
	if err != nil {
		return domain.SandboxResult{}, fmt.Errorf("create Sandbox Session ID: %w", err)
	}
	capability, err := b.probe(ctx)
	if err != nil {
		return incomplete(sessionID, "M3_DYNAMIC_CAPABILITY_ERROR")
	}
	if !capability.Available {
		return incomplete(sessionID, capability.LimitationCode)
	}

	return b.executeEnvironmentInspection(ctx, request, sessionID)
}

func (b *Backend) cleanup(containerID string) error {
	cleanupContext, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
	defer cancel()
	_, err := b.runner.Output(cleanupContext, "docker", "rm", "--force", containerID)
	return err
}

func incomplete(sessionID domain.SandboxSessionID, limitation string) (domain.SandboxResult, error) {
	return domain.NewSandboxResult(sessionID, domain.SandboxIncomplete, limitation, nil)
}

func createArguments(sessionID domain.SandboxSessionID) []string {
	arguments := []string{
		"create",
		"--runtime", gVisorRuntimeName,
		"--network", "none",
		"--read-only",
		"--cap-drop", "ALL",
		"--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP",
		"--security-opt", "no-new-privileges",
		"--pids-limit", "64",
		"--memory", "512m",
		"--cpus", "1",
		"--ulimit", "cpu=30:30",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=256m,uid=1000,gid=1000,mode=0700",
		"--tmpfs", boundaryHelperMount,
	}
	arguments = append(arguments, isolatedContainerEnvironmentArguments()...)
	arguments = append(arguments, "--name", "heliopause-"+sessionID.String(), nodeImageReference, "/bin/sh", "-ceu", boundaryContainerCommand())
	return arguments
}

const npmLifecycleCommand = "mkdir -p /tmp/package /tmp/.npm; cd /tmp/package; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm install --ignore-scripts=false --no-audit --no-fund --offline --no-update-notifier /tmp/artifact.tgz"
