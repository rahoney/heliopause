package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// DockerArtifactIntroducer is trusted control-plane code that copies exactly
// one acquired tarball into a newly created container. The Artifact only sees
// /artifact.tgz in the container's private layer; no Host path is added to its
// runtime command. The layer becomes read-only when the container starts.
type DockerArtifactIntroducer struct {
	intakeRoot string
	runner     CommandRunner
}

func NewDockerArtifactIntroducer(intakeRoot string, runner CommandRunner) (*DockerArtifactIntroducer, error) {
	if !filepath.IsAbs(intakeRoot) {
		return nil, errors.New("sandbox intake root must be absolute")
	}
	if runner == nil {
		return nil, errors.New("sandbox command runner is required")
	}
	return &DockerArtifactIntroducer{intakeRoot: filepath.Clean(intakeRoot), runner: runner}, nil
}

func (i *DockerArtifactIntroducer) Introduce(ctx context.Context, containerID string, artifact domain.AcquiredArtifact) error {
	if i == nil || i.runner == nil || !containerIDPattern.MatchString(containerID) {
		return errors.New("sandbox artifact introducer is not configured")
	}
	if ctx == nil {
		return errors.New("context is required")
	}
	source, err := i.tarballPath(artifact.ContentHandle())
	if err != nil {
		return err
	}
	info, err := os.Stat(source)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("controlled Sandbox Artifact is unavailable")
	}
	if _, err := i.runner.Output(ctx, "docker", "cp", source, containerID+":/artifact.tgz"); err != nil {
		return fmt.Errorf("introduce controlled Sandbox Artifact: %w", err)
	}
	return nil
}

func (i *DockerArtifactIntroducer) tarballPath(handle string) (string, error) {
	parts := strings.Split(handle, ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != "tarball" {
		return "", errors.New("sandbox content handle is invalid")
	}
	runID, err := domain.ParseRunID(parts[1])
	if err != nil {
		return "", errors.New("sandbox content handle is invalid")
	}
	path := filepath.Join(i.intakeRoot, runID.String(), "tarball.tgz")
	relative, err := filepath.Rel(i.intakeRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("sandbox tarball path escapes intake root")
	}
	return path, nil
}
