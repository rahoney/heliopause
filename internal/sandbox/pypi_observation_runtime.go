package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

type pythonObservationPhase string

const (
	phasePreparation pythonObservationPhase = "preparation"
	phaseAnchor      pythonObservationPhase = "anchor"
	phaseObservation pythonObservationPhase = "observation"
)

func pythonObservationCreateArguments(transaction string, ordinal int, phase pythonObservationPhase,
	volume closureVolume, lease ObservationResourceLease, policy artifactpypi.ResourcePolicy) ([]string, error) {
	if transaction == "" || ordinal < 0 || lease.CgroupParent == "" || lease.CPU < 0 ||
		(phase != phasePreparation && phase != phaseAnchor && phase != phaseObservation) {
		return nil, errors.New("python observation runtime identity is invalid")
	}
	readOnly := phase != phasePreparation
	mount, err := volume.mountArgument(readOnly)
	if err != nil {
		return nil, err
	}
	if policy.RuntimeMemory() <= 0 || policy.RuntimeTmpfs() <= 0 ||
		policy.RuntimeMemory() > math.MaxInt64-policy.RuntimeTmpfs() {
		return nil, errors.New("python observation preparation memory is unbounded")
	}
	memory := policy.RuntimeMemory()
	if phase == phasePreparation {
		// Retained volume tmpfs pages and pip's own working set can coexist.
		memory += policy.RuntimeTmpfs()
	}
	nameDigest := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", transaction, phase, ordinal)))
	name := "haa-pypi-" + hex.EncodeToString(nameDigest[:12])
	args := []string{
		"create", "--pull", "never", "--runtime", gVisorRuntimeName,
		"--network", "none", "--read-only", "--cap-drop", "ALL",
		"--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP",
		"--security-opt", "no-new-privileges", "--pids-limit", "64",
		"--memory", strconv.FormatInt(memory, 10), "--cpus", "1",
		"--cpuset-cpus", strconv.Itoa(lease.CPU),
		"--cgroup-parent", lease.CgroupParent,
		"--ulimit", "cpu=" + strconv.Itoa(policy.RuntimeCPUSecs()) + ":" + strconv.Itoa(policy.RuntimeCPUSecs()),
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=" + strconv.FormatInt(policy.RuntimeTmpfs(), 10) + ",uid=1000,gid=1000,mode=0700",
		"--tmpfs", boundaryHelperMount,
		"--mount", mount,
		"--label", closureVolumeLabel + "=" + transaction,
		"--label", closureManifestLabel + "=" + volume.manifestID,
		"--label", "io.heliopause.observation-phase=" + string(phase),
	}
	args = append(args, isolatedContainerEnvironmentArguments()...)
	return append(args, "--name", name, pythonImageReference, "/bin/sh", "-ceu", boundaryContainerCommand()), nil
}

func verifyObservationRuntimeAlive(ctx context.Context, runner CommandRunner, containerID string) error {
	if ctx == nil || runner == nil || !exactObservationContainerID(containerID) {
		return errors.New("python observation runtime identity is invalid")
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", `{{.Id}}|{{.State.Running}}|{{.State.Pid}}`, containerID)
	if err != nil || len(body) > 256 {
		return errors.New("python observation runtime state is unavailable")
	}
	parts := strings.Split(strings.TrimSpace(string(body)), "|")
	if len(parts) != 3 || parts[0] != containerID || parts[1] != "true" {
		return errors.New("python observation runtime is no longer alive")
	}
	pid, err := strconv.Atoi(parts[2])
	if err != nil || pid <= 1 {
		return errors.New("python observation runtime process identity is unavailable")
	}
	return nil
}
