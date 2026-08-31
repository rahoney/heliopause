package sandbox

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// GitHubELFBackend executes only a verified GitHub Release ELF in an isolated
// gVisor session. It owns no GitHub API or Policy concern.
type GitHubELFBackend struct {
	runner       CommandRunner
	intakeRoot   string
	observer     TraceObserver
	probe        CapabilityProbe
	newSessionID func() (domain.SandboxSessionID, error)
}

func NewGitHubELFBackend(runner CommandRunner, intakeRoot string, observer TraceObserver, probe CapabilityProbe) (*GitHubELFBackend, error) {
	if runner == nil || observer == nil || probe == nil || !filepath.IsAbs(intakeRoot) {
		return nil, errors.New("GitHub ELF backend configuration is invalid")
	}
	return &GitHubELFBackend{runner: runner, intakeRoot: filepath.Clean(intakeRoot), observer: observer, probe: probe, newSessionID: domain.NewSandboxSessionID}, nil
}

// NewLinuxGitHubELFBackendWithExecutor uses the composition-root validated
// Host executor for the complete lifecycle.
func NewLinuxGitHubELFBackendWithExecutor(intakeRoot string, executor TrustedExecutor, observer TraceObserver) (*GitHubELFBackend, error) {
	return newLinuxGitHubELFBackend(intakeRoot, executor, observer)
}

func newLinuxGitHubELFBackend(intakeRoot string, executor TrustedExecutor, observer TraceObserver) (*GitHubELFBackend, error) {
	if observer == nil {
		return nil, errors.New("process-scoped observer is required")
	}
	capabilityProbe := func(ctx context.Context) (Capability, error) {
		return probe(ctx, runtime.GOOS, executor)
	}
	backend, err := NewGitHubELFBackend(executor, intakeRoot, observer, capabilityProbe)
	if err != nil {
		return nil, err
	}
	return backend, nil
}

func (b *GitHubELFBackend) Execute(ctx context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	if b == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || ctx == nil || request.Artifact().Identity().Source().String() != "github-release" {
		return domain.SandboxResult{}, errors.New("GitHub ELF sandbox request is invalid")
	}
	sessionID, err := b.newSessionID()
	if err != nil {
		return domain.SandboxResult{}, err
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available {
		if err != nil {
			return incomplete(sessionID, "M6_DYNAMIC_CAPABILITY_ERROR")
		}
		return incomplete(sessionID, capability.LimitationCode)
	}
	created, err := b.runner.Output(ctx, "docker", githubELFCreateArguments(sessionID)...)
	if err != nil {
		return incomplete(sessionID, "M6_DYNAMIC_SETUP_FAILED")
	}
	containerID := strings.TrimSpace(string(created))
	if !containerIDPattern.MatchString(containerID) {
		return incomplete(sessionID, "M6_DYNAMIC_SETUP_FAILED")
	}
	trace, err := startTrace(ctx, b.observer, containerID, "github-elf")
	if err != nil {
		if b.cleanup(containerID) != nil {
			return incomplete(sessionID, "M6_DYNAMIC_CLEANUP_FAILED")
		}
		return incomplete(sessionID, "M6_DYNAMIC_OBSERVER_FAILED")
	}
	runContext, cancel := context.WithTimeout(ctx, sandboxWallTimeout)
	defer cancel()
	if _, err := b.runner.Output(runContext, "docker", "start", containerID); err != nil {
		_ = b.cleanup(containerID)
		return incomplete(sessionID, "M6_DYNAMIC_SETUP_FAILED")
	}
	if err := awaitBoundaryHelper(runContext, b.runner, containerID); err != nil {
		_ = b.cleanup(containerID)
		return incomplete(sessionID, "M6_DYNAMIC_SETUP_FAILED")
	}
	if err := b.introduce(runContext, containerID, request.Artifact()); err != nil {
		if b.cleanup(containerID) != nil {
			return incomplete(sessionID, "M6_DYNAMIC_CLEANUP_FAILED")
		}
		return incomplete(sessionID, "M6_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED")
	}
	_, waitErr := b.runner.Output(runContext, "docker", boundaryExecArguments(containerID, boundaryELFHandoffMode, "/work/artifact")...)
	collectCtx, collectCancel := context.WithTimeout(context.Background(), cleanupTimeout)
	observations, limitation := collectTrace(collectCtx, trace)
	collectCancel()
	cleanupErr := b.cleanup(containerID)
	if cleanupErr != nil {
		return incomplete(sessionID, "M6_DYNAMIC_CLEANUP_FAILED")
	}
	if waitErr != nil {
		if runContext.Err() == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded {
			return incomplete(sessionID, "M6_DYNAMIC_TIMEOUT")
		}
		return incomplete(sessionID, "M6_DYNAMIC_EXECUTION_FAILED")
	}
	if limitation != "" {
		return incomplete(sessionID, limitation)
	}
	completed, _ := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
	observations = append(observations, completed)
	return domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", observations)
}

func (b *GitHubELFBackend) cleanup(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	_, err := b.runner.Output(ctx, "docker", "rm", "--force", containerID)
	return err
}

func (b *GitHubELFBackend) introduce(ctx context.Context, containerID string, artifact domain.AcquiredArtifact) error {
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != "github-release" {
		return errors.New("GitHub ELF intake handle is invalid")
	}
	runID, err := domain.ParseRunID(parts[1])
	if err != nil {
		return errors.New("GitHub ELF intake handle is invalid")
	}
	assetPath := filepath.Join(b.intakeRoot, runID.String(), "asset")
	pathInfo, pathErr := os.Lstat(assetPath)
	if pathErr != nil || !pathInfo.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("GitHub ELF controlled intake is invalid")
	}
	file, err := os.Open(assetPath)
	if err != nil {
		return errors.New("GitHub ELF controlled intake is unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || uint64(info.Size()) != artifact.SizeBytes() {
		return errors.New("GitHub ELF controlled intake is invalid")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil || fmt.Sprintf("%x", hash.Sum(nil)) != artifact.Digest().String() {
		return errors.New("GitHub ELF controlled intake digest mismatch")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	input, ok := b.runner.(inputCommandRunner)
	if !ok {
		return errors.New("GitHub ELF stream runner is unavailable")
	}
	return input.RunInput(ctx, file, "docker", boundaryInputExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; cat > /work/artifact; chmod 500 /work/artifact")...)
}

func githubELFCreateArguments(sessionID domain.SandboxSessionID) []string {
	return []string{"create", "--runtime", gVisorRuntimeName, "--network", "none", "--read-only", "--cap-drop", "ALL", "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1", "--ulimit", "cpu=30:30", "--tmpfs", "/work:rw,exec,nosuid,nodev,size=256m,uid=1000,gid=1000,mode=0700", "--tmpfs", boundaryHelperMount, "--name", "heliopause-github-elf-" + sessionID.String(), nodeImageReference, "/bin/sh", "-ceu", boundaryContainerCommand()}
}
