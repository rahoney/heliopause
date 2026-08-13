package sandbox

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
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

// Backend creates one gVisor-backed Docker container per Sandbox Session.
type Backend struct {
	runner       CommandRunner
	introducer   ArtifactIntroducer
	observer     TraceObserver
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
	return &Backend{runner: runner, introducer: introducer, observer: observer, probe: probe, newSessionID: domain.NewSandboxSessionID, wallTimeout: sandboxWallTimeout, cleanupWait: cleanupTimeout}, nil
}

// Execute introduces the acquired artifact exactly once, runs it under gVisor,
// and disposes of the container before returning a bounded raw result.
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

	created, err := b.runner.Output(ctx, "docker", createArguments(sessionID)...)
	if err != nil {
		return incomplete(sessionID, "M3_DYNAMIC_SETUP_FAILED")
	}
	containerID := strings.TrimSpace(string(created))
	if !containerIDPattern.MatchString(containerID) {
		return incomplete(sessionID, "M3_DYNAMIC_SETUP_FAILED")
	}
	trace, err := b.observer.Start(ctx, containerID)
	if err != nil {
		if b.cleanup(containerID) != nil {
			return incomplete(sessionID, "M3_DYNAMIC_CLEANUP_FAILED")
		}
		return incomplete(sessionID, "M3_DYNAMIC_OBSERVER_FAILED")
	}

	runContext, cancel := context.WithTimeout(ctx, b.wallTimeout)
	defer cancel()
	if _, err := b.runner.Output(runContext, "docker", "start", containerID); err != nil {
		if b.cleanup(containerID) != nil {
			return incomplete(sessionID, "M3_DYNAMIC_CLEANUP_FAILED")
		}
		return incomplete(sessionID, "M3_DYNAMIC_SETUP_FAILED")
	}
	if err := b.introducer.Introduce(runContext, containerID, request.Artifact()); err != nil {
		cleanupErr := b.cleanup(containerID)
		if cleanupErr != nil {
			return incomplete(sessionID, "M3_DYNAMIC_CLEANUP_FAILED")
		}
		return incomplete(sessionID, "M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED")
	}

	waitOutput, runErr := b.runner.Output(runContext, "docker", "wait", containerID)
	timedOut := runContext.Err() == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded
	collectContext, collectCancel := context.WithTimeout(context.Background(), b.cleanupWait)
	observations, observationLimitation := collectTrace(collectContext, trace)
	collectCancel()
	cleanupErr := b.cleanup(containerID)
	if cleanupErr != nil {
		return incomplete(sessionID, "M3_DYNAMIC_CLEANUP_FAILED")
	}
	if runErr != nil || strings.TrimSpace(string(waitOutput)) != "0" {
		if timedOut {
			return incomplete(sessionID, "M3_DYNAMIC_TIMEOUT")
		}
		return incomplete(sessionID, "M3_DYNAMIC_EXECUTION_FAILED")
	}
	if observationLimitation != "" {
		return incomplete(sessionID, observationLimitation)
	}
	observation, err := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
	if err != nil {
		return domain.SandboxResult{}, fmt.Errorf("create Sandbox observation: %w", err)
	}
	observations = append(observations, observation)
	return domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", observations)
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
	return []string{
		"create",
		"--runtime", gVisorRuntimeName,
		"--user", "1000:1000",
		"--network", "none",
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--pids-limit", "64",
		"--memory", "512m",
		"--cpus", "1",
		"--ulimit", "cpu=30:30",
		"--tmpfs", "/work:rw,nosuid,nodev,size=256m,uid=1000,gid=1000,mode=0700",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=256m,uid=1000,gid=1000,mode=0700",
		"--name", "heliopause-" + sessionID.String(),
		nodeImageReference,
		"/bin/sh", "-ceu", "while [ ! -f /work/artifact.tgz ]; do sleep 0.05; done; mkdir -p /work/package; npm install --ignore-scripts=false --no-audit --no-fund /work/artifact.tgz",
	}
}
