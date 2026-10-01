package sandbox

import (
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestPythonObservationRuntimeUsesBoundedPhaseVolumeAndCgroup(t *testing.T) {
	volume := closureVolume{name: "haa-closure-test", createdAt: "2026-09-30T00:00:00Z", mountpoint: "/var/lib/docker/volumes/haa-closure-test/_data", manifestID: strings.Repeat("f", 64)}
	lease := ObservationResourceLease{CgroupParent: "haaobs0123456789abcdef01234567.slice", CPU: 3}
	for _, phase := range []pythonObservationPhase{phasePreparation, phaseAnchor, phaseObservation} {
		args, err := pythonObservationCreateArguments("transaction", 1, phase, volume, lease, artifactpypi.PublicPyPIProfile().ResourcePolicy())
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(args, " ")
		for _, expected := range []string{"--runtime runsc-trace", "--cgroup-parent " + lease.CgroupParent, "--cpuset-cpus 3", "volume-nocopy", "target=/haa-site", "--network none"} {
			if !strings.Contains(joined, expected) {
				t.Fatalf("%s omitted %q", phase, expected)
			}
		}
		if strings.Contains(joined, "type=bind") || strings.Contains(joined, "--tmpfs /haa-site") {
			t.Fatalf("%s introduced a host bind or independent writable site", phase)
		}
		readOnly := strings.Contains(joined, "volume-nocopy,readonly")
		if readOnly != (phase != phasePreparation) {
			t.Fatalf("%s has wrong closure write access", phase)
		}
	}
}
