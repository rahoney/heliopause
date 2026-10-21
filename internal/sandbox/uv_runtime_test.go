package sandbox

import (
	"reflect"
	"testing"
)

func TestUVRuntimeKeepsPythonIsolationAndSeparateToolMount(t *testing.T) {
	if _, err := uvCreateArguments("none", nil, uvRuntimeVolume{}); err == nil {
		t.Fatal("unattested tool volume accepted")
	}
	hosts := []string{"--add-host", "pypi.org:192.0.2.1"}
	old := pypiCreateArguments("isolated-test", hosts)
	volume := uvRuntimeVolume{name: "haa-uv-runtime-test", owner: "test", createdAt: "fixed", mountpoint: "/owned"}
	uv, err := uvCreateArguments("isolated-test", hosts, volume)
	if err != nil {
		t.Fatal(err)
	}
	if len(old) != len(uv) {
		t.Fatal("UV changed the Python container boundary")
	}
	differences := 0
	for i := range old {
		if old[i] != uv[i] {
			differences++
			if !(old[i] == "--tmpfs" && uv[i] == "--mount") && !(old[i] == boundaryHelperMount && uv[i] == "type=volume,source=haa-uv-runtime-test,target=/haa-runtime,volume-nocopy") {
				t.Fatalf("unexpected UV boundary change at %d", i)
			}
		}
	}
	if differences != 2 || boundaryHelperMount != "/haa-runtime:rw,exec,nosuid,nodev,size=1m,uid=0,gid=0,mode=0755" {
		t.Fatal("existing Python helper capacity changed")
	}
	pythonTopology, ok := observerExpectedTopology("pypi-wheel")
	uvTopology, uvOK := observerExpectedTopology(uvResolverProfile)
	for i := range uvTopology {
		if uvTopology[i].Mountpoint == "/haa-runtime" {
			if uvTopology[i].FSType != "9p" || uvTopology[i].NoSUID {
				t.Fatal("wrong UV named tmpfs guest topology")
			}
			uvTopology[i].FSType = "tmpfs"
			uvTopology[i].NoSUID = true
		}
	}
	if !ok || !uvOK || !reflect.DeepEqual(pythonTopology, uvTopology) {
		t.Fatal("UV must use the sealed Python image/site/helper topology")
	}
	if isPythonObserverProfile(uvResolverProfile) {
		t.Fatal("UV must not inherit Python closure authority")
	}
	if _, err := newObservationTraceLedger(uvResolverProfile); err == nil {
		t.Fatal("UV acquired Python observation transaction authority")
	}
	if _, ok := pythonClosureExpectedTopology(uvResolverProfile, pythonClosureObservation); ok {
		t.Fatal("UV acquired Python closure topology")
	}
	if got, want := traceBudgetForProfile(uvResolverProfile), traceBudgetForProfile("pypi-wheel"); got != want {
		t.Fatal("UV changed the existing finite default collector ceiling")
	}
	for _, profile := range []string{"uv", "uv-pip", "uv-pip-resolver-extra"} {
		if validObserverProfile(profile) {
			t.Fatalf("accepted unknown profile %s", profile)
		}
	}
}
