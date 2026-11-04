package sandbox

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func sourceAnalysisRuntimeFixture(t *testing.T) (*readonlyToolVolume, *readonlyToolVolume, string, ObservationResourceLease) {
	t.Helper()
	identity, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	input, err := prepareSourceAnalysisInput(context.Background(), sourceAnalysisInputFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	volume := func(suffix, digest, destination string, capacity int64, executable bool) closureVolume {
		transaction := identity.String() + suffix
		name, err := closureVolumeName(transaction)
		if err != nil {
			t.Fatal(err)
		}
		return closureVolume{name: name, transaction: transaction, manifestID: digest, createdAt: "2026-11-04T00:00:00Z", mountpoint: "/var/lib/docker/volumes/" + name + "/_data", capacity: capacity, projectBuildDestination: destination, projectBuildExecutable: executable}
	}
	supply := &readonlyToolVolume{volume: volume("-tool", semgrepSupplyDigest(), semgrepRuntimeRoot, semgrepSupplyVolumeCapacity, true), semgrep: true, seeded: true, sealed: true}
	source := &readonlyToolVolume{volume: volume("-source", input.manifestID, sourceAnalysisRoot, sourceAnalysisVolumeCapacity, false), source: input, seeded: true, sealed: true}
	return supply, source, identity.String(), ObservationResourceLease{CgroupParent: "heliopause-source-test", CPU: 1}
}

func TestSourceAnalysisRuntimePinsOfflineExecutionAndSeparateReadonlyInputs(t *testing.T) {
	supply, source, transaction, lease := sourceAnalysisRuntimeFixture(t)
	args, err := sourceAnalysisCreateArguments(supply, source, transaction, lease)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string][]string{}
	for i := 1; i < len(args)-4; i++ {
		switch args[i] {
		case "--network", "--runtime", "--cpuset-cpus", "--cgroup-parent", "--memory", "--ulimit", "--tmpfs", "--mount", "--label":
			values[args[i]] = append(values[args[i]], args[i+1])
			i++
		}
	}
	for flag, want := range map[string]string{"--network": "none", "--runtime": gVisorRuntimeName, "--cpuset-cpus": "1", "--cgroup-parent": lease.CgroupParent, "--memory": "512m", "--ulimit": "cpu=60:60"} {
		if !reflect.DeepEqual(values[flag], []string{want}) {
			t.Fatal(flag, values[flag])
		}
	}
	if len(values["--mount"]) != 2 || len(values["--tmpfs"]) != 2 {
		t.Fatal("input/helper topology changed", values)
	}
	for _, mount := range values["--mount"] {
		if !strings.HasSuffix(mount, ",readonly") || (!strings.Contains(mount, "target="+semgrepRuntimeRoot+",") && !strings.Contains(mount, "target="+sourceAnalysisRoot+",")) {
			t.Fatal("writable or unexpected input", mount)
		}
	}
	if !reflect.DeepEqual(args[len(args)-4:], []string{pythonImageReference, "/bin/sh", "-ceu", boundaryContainerCommand()}) {
		t.Fatal("bootstrap image/command drift")
	}
	for _, arg := range args {
		if arg == "--env-file" || arg == "--add-host" || arg == "--volume" || arg == "-v" {
			t.Fatal("ambient input admitted", arg)
		}
	}
	topology, ok := observerExpectedTopology(sourceAnalysisProfile)
	if !ok || len(topology) != 5 || !validObserverProfile(sourceAnalysisProfile) || isPythonObserverProfile(sourceAnalysisProfile) {
		t.Fatal("source profile lost isolation")
	}
	for _, mount := range topology[3:] {
		if !mount.ReadOnly || mount.Class != "workspace" || mount.FSType != "9p" || mount.Parent != "/" {
			t.Fatal("source supply/target topology drift", mount)
		}
	}
	ledger, err := newObservationTraceLedger(sourceAnalysisProfile)
	if err != nil || ledger.maxEvents != uint64(defaultTraceBudget.events) || ledger.maxBytes != defaultTraceBudget.bytes {
		t.Fatal("source budget expanded", err)
	}
	command := sourceAnalysisCommand()
	if len(command) != 17 || command[0] != semgrepRuntimeRoot+"/semgrep-core-linux-amd64" || command[8] != semgrepRuntimeRoot+"/semgrep-rules.json" || command[10] != sourceAnalysisRoot+"/targets.json" {
		t.Fatal("fixed scanner input changed", command)
	}
	for _, arg := range command {
		if strings.Contains(arg, source.source.files[0].Path) {
			t.Fatal("original filename became command input")
		}
	}
}

func TestSourceAnalysisRuntimeRefusesUnsealedMixedOrUnaccountedInput(t *testing.T) {
	for _, fault := range []string{"nil-tool", "nil-source", "tool-unsealed", "source-unsealed", "mutable-source", "wrong-tool", "same-volume", "unseeded", "unbound", "lease", "cpu"} {
		t.Run(fault, func(t *testing.T) {
			supply, source, transaction, lease := sourceAnalysisRuntimeFixture(t)
			switch fault {
			case "nil-tool":
				supply = nil
			case "nil-source":
				source = nil
			case "tool-unsealed":
				supply.sealed = false
			case "source-unsealed":
				source.sealed = false
			case "mutable-source":
				source.volume.projectBuildExecutable = true
			case "wrong-tool":
				supply.volume.manifestID = strings.Repeat("a", 64)
			case "same-volume":
				source.volume.name = supply.volume.name
			case "unseeded":
				source.seeded = false
			case "unbound":
				transaction = "not-a-session"
			case "lease":
				lease.CgroupParent = ""
			case "cpu":
				lease.CPU = -1
			}
			if args, err := sourceAnalysisCreateArguments(supply, source, transaction, lease); err == nil || args != nil {
				t.Fatal("unsafe source execution admitted")
			}
		})
	}
}
