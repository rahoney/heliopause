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
	resources    ObservationResourceClient
	newSessionID func() (domain.SandboxSessionID, error)
}

func NewGitHubELFBackend(runner CommandRunner, intakeRoot string, observer TraceObserver, probe CapabilityProbe) (*GitHubELFBackend, error) {
	if runner == nil || observer == nil || probe == nil || !filepath.IsAbs(intakeRoot) {
		return nil, errors.New("GitHub ELF backend configuration is invalid")
	}
	return &GitHubELFBackend{runner: admissionAwareRunner(runner), intakeRoot: filepath.Clean(intakeRoot), observer: observer, probe: probe, newSessionID: domain.NewSandboxSessionID}, nil
}

// NewLinuxGitHubELFBackendWithExecutor uses the composition-root validated
// Host executor for the complete lifecycle.
func NewLinuxGitHubELFBackendWithExecutor(intakeRoot string, executor TrustedExecutor, observer TraceObserver, resources ObservationResourceClient) (*GitHubELFBackend, error) {
	return newLinuxGitHubELFBackend(intakeRoot, executor, observer, resources)
}

func newLinuxGitHubELFBackend(intakeRoot string, executor TrustedExecutor, observer TraceObserver, resources ObservationResourceClient) (*GitHubELFBackend, error) {
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
	backend.resources = resources
	return backend, nil
}

func (b *GitHubELFBackend) Execute(ctx context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	return b.executeELF(ctx, request, "github-release", "github-elf", b.introduce, nil)
}

// executeELF shares only fixed isolation/lifecycle mechanics. Source identity,
// profile and introduction are supplied by the owning backend, never input.
func (b *GitHubELFBackend) executeELF(ctx context.Context, request domain.SandboxRequest, source, profile string, introduce func(context.Context, string, domain.AcquiredArtifact) error, probeTerminal func(context.Context, string) (string, error)) (domain.SandboxResult, error) {
	if b == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || ctx == nil || request.Artifact().Identity().Source().String() != source {
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
	return b.executeELFEnvironments(ctx, request, sessionID, profile, introduce, probeTerminal)
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
