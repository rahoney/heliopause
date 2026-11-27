package sandbox

import (
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func pipxNativeRuntimeFixture(t *testing.T) (*pipxDataVolume, *pipxWheelVolume, nativeRuntimeVolume, closureVolume) {
	t.Helper()
	zip, _ := pipxVolumeFixture()
	zip.seeded, zip.sealed = true, true
	wheels, _ := pipxWheelVolumeFixture(t)
	wheels.seeded, wheels.sealed = true, true
	_, runtime, _ := nativeVolumeFixture(t, nativeRuntimeUV)
	output := wheels.volume
	output.transaction = "pipx-native-output"
	output.name, _ = closureVolumeName(output.transaction)
	output.mountpoint = "/var/lib/docker/volumes/" + output.name + "/_data"
	output.projectBuildDestination = ""
	output.capacity = artifactpypi.PublicPyPIProfile().ResourcePolicy().PromotionTmpfs()
	return zip, wheels, runtime, output
}
func TestPipxNativeRuntimeClosedOfflineArgumentsAndTopology(t *testing.T) {
	zip, wheels, runtime, output := pipxNativeRuntimeFixture(t)
	args, err := pipxNativeCreateArguments(zip, wheels, runtime, output)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string][]string{}
	for i := 1; i < len(args)-4; i++ {
		switch args[i] {
		case "--network", "--runtime", "--workdir", "--memory", "--ulimit", "--tmpfs", "--mount", "--env", "--cap-add":
			values[args[i]] = append(values[args[i]], args[i+1])
			i++
		}
	}
	for flag, want := range map[string]string{"--network": "none", "--runtime": gVisorRuntimeName, "--workdir": "/", "--memory": "512m", "--ulimit": "cpu=30:30"} {
		if !reflect.DeepEqual(values[flag], []string{want}) {
			t.Fatal(flag, values[flag])
		}
	}
	if len(values["--mount"]) != 4 || !reflect.DeepEqual(values["--tmpfs"], []string{"/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700"}) {
		t.Fatal("mount/tmpfs drift", values)
	}
	for _, mount := range values["--mount"] {
		if strings.Contains(mount, "target="+pipxRuntimeRoot+",") || strings.Contains(mount, "target="+pipxNativeWheels+",") {
			if !strings.HasSuffix(mount, ",readonly") {
				t.Fatal("writable input", mount)
			}
		}
	}
	if !reflect.DeepEqual(args[len(args)-4:], []string{pythonImageReference, "/bin/sh", "-ceu", boundaryContainerCommand()}) {
		t.Fatal("bootstrap drift")
	}
	for _, arg := range args {
		if arg == "--user" || arg == "--add-host" || arg == "--env-file" || arg == "--volume" || arg == "-v" {
			t.Fatal("native runtime acquired an override", arg)
		}
	}
	for _, want := range []string{"UV_OFFLINE=true", "UV_NO_INDEX=true", "UV_FIND_LINKS=/haa-wheels", "UV_NO_CONFIG=true", "PIPX_DEFAULT_BACKEND=uv"} {
		found := false
		for _, env := range values["--env"] {
			if env == want {
				found = true
			}
		}
		if !found {
			t.Fatal("fixed environment missing", want)
		}
	}
	uv, _ := observerExpectedTopology(uvResolverProfile)
	oldZIP, _ := observerExpectedTopology(pipxObserverProfile)
	topology, ok := observerExpectedTopology(pipxNativeResolverProfile)
	if !ok || len(topology) != 6 || !validObserverProfile(pipxNativeResolverProfile) || isPythonObserverProfile(pipxNativeResolverProfile) {
		t.Fatal("dedicated profile topology missing or inherited wheel inspection")
	}
	mounts := map[string]observerMountExpectation{}
	for _, m := range topology {
		mounts[m.Mountpoint] = m
	}
	for _, name := range []string{pipxRuntimeRoot, pipxNativeWheels} {
		m := mounts[name]
		if !m.ReadOnly || m.FSType != "9p" || m.Class != "workspace" || m.Parent != "/" || m.NoExec {
			t.Fatal("guest input topology", m)
		}
	}
	if mounts[pythonSitePath].ReadOnly || mounts["/haa-runtime"].FSType != "9p" {
		t.Fatal("runtime/output topology")
	}
	if budget := traceBudgetForProfile(pipxNativeResolverProfile); budget.events != 10000 || budget.bytes != 2<<20 {
		t.Fatal("native trace budget expanded")
	}
	topology[0].ReadOnly = false
	fresh, _ := observerExpectedTopology(pipxNativeResolverProfile)
	if !fresh[0].ReadOnly {
		t.Fatal("shared topology mutated")
	}
	actualUV, _ := observerExpectedTopology(uvResolverProfile)
	actualZIP, _ := observerExpectedTopology(pipxObserverProfile)
	if !reflect.DeepEqual(uv, actualUV) || !reflect.DeepEqual(oldZIP, actualZIP) || len(actualUV) != 4 || len(actualZIP) != 4 {
		t.Fatal("existing profile changed")
	}
	if _, ok := pythonClosureExpectedTopology(pipxNativeResolverProfile, pythonClosureObservation); ok {
		t.Fatal("native profile gained wheel transaction phases")
	}
}
func TestPipxNativeRuntimeRefusesUnsealedChangedAndForeignOwners(t *testing.T) {
	for _, fault := range []string{"nil ZIP", "nil wheels", "unsealed ZIP", "unsealed wheels", "changed original", "foreign runtime", "foreign output root", "different output ID", "output budget"} {
		t.Run(fault, func(t *testing.T) {
			zip, wheels, runtime, output := pipxNativeRuntimeFixture(t)
			switch fault {
			case "nil ZIP":
				zip = nil
			case "nil wheels":
				wheels = nil
			case "unsealed ZIP":
				zip.sealed = false
			case "unsealed wheels":
				wheels.sealed = false
			case "changed original":
				wheels.input.files[0].body[0] ^= 1
			case "foreign runtime":
				runtime.kind = nativeRuntimePNPM
			case "foreign output root":
				output.projectBuildDestination = pipxNativeWheels
			case "different output ID":
				output.manifestID = strings.Repeat("a", 64)
			case "output budget":
				output.capacity++
			}
			if args, err := pipxNativeCreateArguments(zip, wheels, runtime, output); err == nil || args != nil {
				t.Fatal("native runtime accepted invalid owner")
			}
		})
	}
}
