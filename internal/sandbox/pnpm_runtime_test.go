package sandbox

import (
	"archive/tar"
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestPNPMRuntimeKeepsNodeBoundaryAndRejectsOtherToolAuthority(t *testing.T) {
	f := &uvVolumeFake{}
	volume, err := createPNPMRuntimeVolume(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	project := closureVolume{name: "haa-closure-pnpm-test", transaction: "test-pnpm-project", manifestID: strings.Repeat("f", 64), createdAt: "fixed", mountpoint: "/project", capacity: 128 << 20, projectBuildDestination: pnpmProjectRoot}
	args, err := pnpmCreateArguments("none", nil, volume, project)
	if err != nil {
		t.Fatal(err)
	}
	early, err := pnpmCreateArgumentsWithStore("none", nil, volume, project, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"XDG_CACHE_HOME=" + pnpmProjectRoot + "/cache", "TMPDIR=" + pnpmProjectRoot + "/tmp", "PNPM_CONFIG_STORE_DIR=" + pnpmProjectRoot + "/store"} {
		index := slices.Index(early, value)
		if index <= 0 || early[index-1] != "--env" || slices.Contains(args, value) || slices.Contains(early[index+1:], value) {
			t.Fatal("early store context missing, repeated or leaked into original selection", value)
		}
		early = slices.Delete(early, index-1, index+1)
	}
	if !slices.Equal(early, args) {
		t.Fatal("early store changed unrelated native boundary")
	}
	old := npmCreateArguments("none", nil)
	projectMount, err := project.mountArgument(false)
	if err != nil || len(args) != len(old)+6 {
		t.Fatal("changed Node execution boundary")
	}
	projectIndex := -1
	for i := 1; i < len(args); i++ {
		if args[i] == projectMount && args[i-1] == "--mount" {
			projectIndex = i - 1
			break
		}
	}
	if projectIndex < 0 {
		t.Fatal("project mount missing")
	}
	args = append(append([]string{}, args[:projectIndex]...), args[projectIndex+2:]...)
	if args[projectIndex] != "--env" || args[projectIndex+1] != "NPM_CONFIG_MANAGE_PACKAGE_MANAGER_VERSIONS=false" {
		t.Fatal("PNPM package-manager switching is not disabled")
	}
	args = append(args[:projectIndex], args[projectIndex+2:]...)
	if args[projectIndex] != "--env" || args[projectIndex+1] != "PNPM_CONFIG_UPDATE_NOTIFIER=false" {
		t.Fatal("PNPM background update notifier is not disabled")
	}
	args = append(args[:projectIndex], args[projectIndex+2:]...)
	changes := 0
	for i := range old {
		if args[i] != old[i] {
			changes++
			if !(old[i] == "--tmpfs" && args[i] == "--mount") && !(old[i] == boundaryHelperMount && strings.Contains(args[i], "source="+volume.name+",target=/haa-runtime,volume-nocopy")) {
				t.Fatal("unexpected Node boundary change", i)
			}
		}
	}
	if changes != 2 {
		t.Fatal("helper mount was not isolated")
	}
	for _, other := range []nativeRuntimeVolume{{}, {kind: nativeRuntimeToolKind(99)}, {kind: nativeRuntimeUV, name: "haa-uv-runtime-test", owner: "test", createdAt: "fixed", mountpoint: "/owned"}} {
		if _, err := pnpmCreateArguments("none", nil, other, project); err == nil {
			t.Fatal("other or unattested tool volume admitted")
		}
	}
	if _, err := createNativeRuntimeVolume(context.Background(), f, nativeRuntimeToolKind(99)); err == nil {
		t.Fatal("unknown native tool accepted")
	}
	otherKind := volume
	otherKind.kind = nativeRuntimeUV
	if err := otherKind.verify(context.Background(), f); err == nil {
		t.Fatal("PNPM volume became UV authority")
	}
	if err := volume.close(context.Background(), f); err != nil || !f.removed {
		t.Fatal("owned pnpm volume cleanup failed", err)
	}
	topology, ok := observerExpectedTopology(pnpmResolverProfile)
	npm, npmOK := observerExpectedTopology("npm-lifecycle")
	filtered := make([]observerMountExpectation, 0, len(topology))
	for _, mount := range topology {
		if mount.Mountpoint == pnpmProjectRoot {
			if mount.Parent != "/tmp" || mount.FSType != "9p" || mount.Class != "workspace" {
				t.Fatal("wrong project topology")
			}
			continue
		}
		filtered = append(filtered, mount)
	}
	topology = filtered
	for i := range topology {
		if topology[i].Mountpoint == "/haa-runtime" {
			topology[i].FSType = "tmpfs"
			topology[i].NoSUID = true
		}
	}
	if !ok || !npmOK || !reflect.DeepEqual(topology, npm) {
		t.Fatal("PNPM changed Node topology")
	}
	if isPythonObserverProfile(pnpmResolverProfile) {
		t.Fatal("PNPM acquired Python role")
	}
	if _, err := newObservationTraceLedger(pnpmResolverProfile); err == nil {
		t.Fatal("PNPM acquired Python ledger authority")
	}
	for _, profile := range []string{"pnpm", "pnpm-resolver-extra", "pnpm-dlx"} {
		if validObserverProfile(profile) {
			t.Fatal("unknown profile admitted", profile)
		}
	}
}

func TestPNPMTransportCannotAcceptUVOrUnknownTool(t *testing.T) {
	body, id := uvSyntheticTar(t, nil, false)
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimePNPM, nativeRuntimeToolKind(99)} {
		if _, err := scanNativeToolTar(body, id, kind); err == nil {
			t.Fatal("UV payload admitted as another tool")
		}
	}
	body, id = uvSyntheticTar(t, func(h *tar.Header) { h.Name = "pnpm" }, false)
	if _, err := scanNativeToolTar(body, id, nativeRuntimePNPM); err != nil {
		t.Fatal("structural PNPM transport refused", err)
	}
	if _, err := scanNativeToolTar(body, id, nativeRuntimeUV); err == nil {
		t.Fatal("PNPM payload admitted as UV")
	}
}

func TestPNPMProjectVolumeRequiresNonexecutableFixedDestination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := createPNPMProjectVolume(ctx, &uvVolumeFake{}, "test"); err == nil {
		t.Fatal("canceled project creation accepted")
	}
	volume := closureVolume{name: "haa-closure-pnpm", transaction: "tx", manifestID: strings.Repeat("f", 64), createdAt: "fixed", mountpoint: "/owned", capacity: 128 << 20, projectBuildDestination: pnpmProjectRoot}
	if volume.destination() != pnpmProjectRoot || !strings.HasSuffix(volume.tmpfsOptions(), ",noexec") {
		t.Fatal("wrong PNPM project execution boundary")
	}
	runtime := nativeRuntimeVolume{name: "haa-pnpm-runtime-test", owner: "test", createdAt: "fixed", mountpoint: "/runtime", kind: nativeRuntimePNPM}
	for _, bad := range []closureVolume{{}, {projectBuildDestination: pythonSitePath}, {projectBuildDestination: pnpmProjectRoot, projectBuildExecutable: true}} {
		if _, err := pnpmCreateArguments("none", nil, runtime, bad); err == nil {
			t.Fatal("foreign or executable project volume accepted")
		}
	}
}
