package sandbox

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestPipxBackendRejectsUnboundInputBeforeCommands(t *testing.T) {
	runner := &recordingRunner{}
	backend, err := newPythonObservationBackend(runner, &emptyObserver{}, availablePythonProbe)
	if err != nil || backend.introducer != nil {
		t.Fatal("ZIP backend unexpectedly requires a wheel directory", err)
	}
	result, err := backend.InspectPipxRuntime(context.Background(), pipxObservationArtifact(t, ""), artifactpypi.PipxRuntimePlan{})
	if err == nil || result.SessionID().String() != "" || len(runner.calls) != 0 || len(runner.inputCalls) != 0 {
		t.Fatal("unbound plan reached commands", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = backend.InspectPipxRuntime(ctx, pipxObservationArtifact(t, ""), artifactpypi.PipxRuntimePlan{})
	if !errors.Is(err, context.Canceled) || len(runner.calls) != 0 {
		t.Fatal("cancelled input reached commands", err)
	}
	if _, err := NewLinuxPipxRuntimeBackendWithExecutor(nil, &emptyObserver{}, nil); err == nil {
		t.Fatal("missing executor admitted")
	}
}

func TestPipxObservationRuntimeKeepsDefaultMemoryAndFixedDataMount(t *testing.T) {
	volume, _ := pipxVolumeFixture()
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	lease := ObservationResourceLease{CgroupParent: "haaobs0123456789abcdef01234567.slice", CPU: 3}
	for _, phase := range []pythonObservationPhase{phasePreparation, phaseAnchor, phaseObservation} {
		args, err := pipxObservationCreateArguments("transaction", 1, phase, volume.volume, lease, policy)
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(args, " ")
		for _, expected := range []string{"target=/haa-pipx", "--network none", "--runtime runsc-trace", "--cpuset-cpus 3", "--cgroup-parent " + lease.CgroupParent, "--memory " + strconv.FormatInt(policy.RuntimeMemory(), 10)} {
			if !strings.Contains(joined, expected) {
				t.Fatal("fixed runtime omitted constraint", phase, expected)
			}
		}
		if strings.Contains(joined, "type=bind") || strings.Contains(joined, "/haa-site") || strings.Contains(joined, "pip install") {
			t.Fatal("ZIP inherited wheel preparation", phase)
		}
		if strings.Contains(joined, "volume-nocopy,readonly") != (phase != phasePreparation) {
			t.Fatal("DATA phase permission changed", phase)
		}
	}
	for _, mode := range []string{"foreign-volume", "capacity", "manifest", "policy"} {
		t.Run(mode, func(t *testing.T) {
			bad := volume.volume
			selected := policy
			switch mode {
			case "foreign-volume":
				bad.projectBuildDestination = "/haa-site"
			case "capacity":
				bad.capacity++
			case "manifest":
				bad.manifestID = strings.Repeat("a", 64)
			case "policy":
				root, ok := artifactpypi.PyTorchProfile("cpu")
				if !ok {
					t.Fatal("profile missing")
				}
				selected = root.ResourcePolicy()
			}
			if args, err := pipxObservationCreateArguments("transaction", 1, phaseObservation, bad, lease, selected); err == nil || args != nil {
				t.Fatal("foreign volume or budget admitted")
			}
		})
	}
}
