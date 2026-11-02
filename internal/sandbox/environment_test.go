package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestInspectionEnvironmentIsSyntheticFreshAndRoleSeparated(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "host-secret-must-not-appear")
	t.Setenv("HOME", "/host/private-home-must-not-appear")
	first, _ := domain.NewSandboxSessionID()
	second, _ := domain.NewSandboxSessionID()
	minimal, err := newInspectionEnvironmentProfile(minimalEnvironment, first)
	if err != nil {
		t.Fatal(err)
	}
	mock, err := newInspectionEnvironmentProfile(developmentEnvironment, first)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := newInspectionEnvironmentProfile(developmentEnvironment, second)
	if err != nil || fresh.identity == mock.identity || mock.identity == minimal.identity {
		t.Fatal("independent sessions/profiles share identity", err)
	}
	if len(minimal.fixtures) != 1 || !minimal.fixtures[0].canary {
		t.Fatal("minimal environment has unexpected mock configuration")
	}
	for _, profile := range []inspectionEnvironmentProfile{minimal, mock, fresh} {
		payload, err := profile.payload()
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(profile.environment, "\x00") + string(payload)
		for _, forbidden := range []string{"host-secret-must-not-appear", "/host/private-home-must-not-appear", "LD_PRELOAD=", "HTTP_PROXY=", "HTTPS_PROXY=", "SSL_CERT_FILE="} {
			if strings.Contains(joined, forbidden) {
				t.Fatal("profile imports Host or executable/network authority")
			}
		}
		for _, file := range profile.fixtures {
			if file.canary != (file.name == ".haa-honeytoken") {
				t.Fatal("normal configuration was classified as canary")
			}
		}
		root := environmentTarFixture(t, nil, "")
		state, err := scanEnvironmentState(context.Background(), bytes.NewReader(append(root[:512], payload...)), environmentStateLimits{entries: 64, bytes: 32 << 10, archiveBytes: 64 << 10})
		if err != nil || profile.verifyPlacement(state) != nil {
			t.Fatal("controller payload not verifiable", err)
		}
		if profile.kind == developmentEnvironment {
			if !strings.Contains(joined, "HOME=/tmp") || !strings.Contains(joined, "project/.env") || !strings.Contains(joined, ".aws/credentials") {
				t.Fatal("mock development cues missing")
			}
		}
		file := state.files[".haa-honeytoken"]
		file.mode = 0600
		state.files[".haa-honeytoken"] = file
		if profile.verifyPlacement(state) == nil {
			t.Fatal("wrong placement mode accepted")
		}
	}
	if _, err := newInspectionEnvironmentProfile("artifact-selected", first); err == nil {
		t.Fatal("unknown profile admitted")
	}
	if _, err := newInspectionEnvironmentProfile(minimalEnvironment, domain.SandboxSessionID{}); err == nil {
		t.Fatal("missing fresh session admitted")
	}
}

func TestInspectionEnvironmentPlacementRejectsMissingChangedOrForeignFiles(t *testing.T) {
	session, _ := domain.NewSandboxSessionID()
	profile, err := newInspectionEnvironmentProfile(developmentEnvironment, session)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := profile.payload()
	if err != nil {
		t.Fatal(err)
	}
	root := environmentTarFixture(t, nil, "")
	state, err := scanEnvironmentState(context.Background(), bytes.NewReader(append(root[:512], payload...)), environmentStateLimits{entries: 64, bytes: 32 << 10, archiveBytes: 64 << 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(environmentFileState) environmentFileState{
		func(f environmentFileState) environmentFileState { f.uid = 0; return f },
		func(f environmentFileState) environmentFileState { f.gid = 0; return f },
		func(f environmentFileState) environmentFileState { f.size++; return f },
		func(f environmentFileState) environmentFileState { f.digest = strings.Repeat("0", 64); return f },
		func(f environmentFileState) environmentFileState { f.kind = tar.TypeSymlink; return f },
	} {
		original := state.files["project/.env"]
		state.files["project/.env"] = mutate(original)
		if profile.verifyPlacement(state) == nil {
			t.Fatal("changed placement accepted")
		}
		state.files["project/.env"] = original
	}
	delete(state.files, "project/.env")
	if profile.verifyPlacement(state) == nil {
		t.Fatal("missing placement accepted")
	}
}

func TestEnvironmentContainerKeepsExistingBoundaryAndAttestsOnlyFixedWorkspace(t *testing.T) {
	session, _ := domain.NewSandboxSessionID()
	profile, err := newInspectionEnvironmentProfile(developmentEnvironment, session)
	if err != nil {
		t.Fatal(err)
	}
	volume := closureVolume{name: "haa-closure-test", transaction: session.String(), manifestID: profile.identity,
		createdAt: "2026-11-02T00:00:00Z", mountpoint: "/var/lib/docker/volumes/haa-closure-test/_data",
		capacity: 256 << 20, projectBuildDestination: environmentWorkspaceRoot}
	args, err := environmentContainerArguments(session, profile, volume, false)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, "\x00")
	for _, required := range []string{"--runtime\x00" + gVisorRuntimeName, "--network\x00none", "--read-only", "--tmpfs\x00" + boundaryHelperMount,
		"--ulimit\x00cpu=30:30", "--memory\x00512m", "--cpus\x001", "HOME=/tmp", "--annotation\x00" + environmentWorkspacePolicy, boundaryContainerCommand()} {
		if !strings.Contains(joined, required) {
			t.Fatalf("existing boundary lost: %q", required)
		}
	}
	if strings.Count(joined, "--mount\x00") != 1 || strings.Contains(joined, "--tmpfs\x00/tmp:") ||
		!strings.Contains(joined, "target=/tmp,volume-nocopy") || !strings.Contains(volume.tmpfsOptions(), ",noexec") {
		t.Fatal("workspace/environment duplicated or Host noexec missing")
	}
	for _, alter := range []func(*closureVolume){
		func(v *closureVolume) { v.projectBuildDestination = "/etc" },
		func(v *closureVolume) { v.projectBuildExecutable = true },
		func(v *closureVolume) { v.manifestID = strings.Repeat("0", 64) },
		func(v *closureVolume) { v.transaction = "foreign" },
		func(v *closureVolume) { v.createdAt = "" },
	} {
		changed := volume
		alter(&changed)
		if _, err := environmentContainerArguments(session, profile, changed, false); err == nil {
			t.Fatal("changed workspace accepted")
		}
	}
	if _, err := createEnvironmentWorkspace(context.Background(), nil, session, profile, 0); err == nil {
		t.Fatal("unbounded/unavailable workspace accepted")
	}
	base, _ := observerExpectedTopology("npm-lifecycle")
	variant, ok := environmentExpectedTopology(false)
	if !ok || len(base) != len(variant) || base[1].FSType != "tmpfs" || variant[1].FSType != "9p" || variant[1].ReadOnly || !variant[1].NoExec || !variant[1].NoSUID || variant[1].NoDev {
		t.Fatal("guest topology changed base or omitted mandatory guest flags")
	}
	budget, err := newObservationTraceLedger("npm-lifecycle")
	if err != nil || budget.maxEvents != uint64(defaultTraceBudget.events) || budget.maxBytes != defaultTraceBudget.bytes {
		t.Fatal("workspace profile replenished or enlarged the trace budget", err)
	}
	if _, err := (&SharedObserver{}).StartEnvironmentProfile(context.Background(), strings.Repeat("a", 64), false, nil); err == nil {
		t.Fatal("environment observer started without a shared budget")
	}
}

func TestNPMInstanceEnvironmentKeepsExistingMountAuthority(t *testing.T) {
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimePNPM, nativeRuntimeBun, nativeRuntimeNPMProject} {
		spec, _ := nativeProjectIdentity(kind)
		for _, readOnly := range []bool{false, true} {
			topology, ok := npmInstanceEnvironmentExpectedTopology(kind, false, readOnly)
			if !ok {
				t.Fatal("current instance topology unavailable", kind)
			}
			var workspaces, projects int
			for _, mount := range topology {
				switch mount.Mountpoint {
				case "/tmp":
					if mount.FSType != "9p" || mount.ReadOnly != readOnly || !mount.NoExec || !mount.NoSUID || mount.NoDev {
						t.Fatal("parent workspace restriction lost")
					}
					workspaces++
				case spec.root:
					if mount.FSType != "9p" || mount.Parent != "/tmp" || mount.ReadOnly != readOnly || mount.NoExec || mount.NoSUID || mount.NoDev {
						t.Fatal("existing project authority changed")
					}
					projects++
				case "/haa-runtime":
					if mount.FSType != "tmpfs" {
						t.Fatal("ordinary writer/keeper gained a native runtime")
					}
				}
			}
			if workspaces != 1 || projects != 1 {
				t.Fatal("missing or duplicate workspace")
			}
		}
	}
	native, ok := npmInstanceEnvironmentExpectedTopology(nativeRuntimeBun, true, false)
	if !ok {
		t.Fatal("existing Bun runtime writer unavailable")
	}
	for _, mount := range native {
		if mount.Mountpoint == "/haa-runtime" && (mount.FSType != "9p" || mount.NoSUID) {
			t.Fatal("existing Bun runtime topology changed")
		}
	}
	for _, input := range []struct {
		kind           nativeRuntimeToolKind
		native, reader bool
	}{
		{0, false, false}, {nativeRuntimePNPM, true, false}, {nativeRuntimeNPMProject, true, false}, {nativeRuntimeBun, true, true},
	} {
		if _, ok := npmInstanceEnvironmentExpectedTopology(input.kind, input.native, input.reader); ok {
			t.Fatal("unknown/foreign runtime or native DATA keeper admitted")
		}
	}
	if _, err := (&SharedObserver{}).startNPMInstanceEnvironmentProfile(context.Background(), strings.Repeat("a", 64), nativeRuntimePNPM, false, false, nil); err == nil {
		t.Fatal("instance observer started without shared trace budget")
	}
}

func TestInspectionEnvironmentUnitRootsAreIsolated(t *testing.T) {
	session, _ := domain.NewSandboxSessionID()
	profile, _ := newInspectionEnvironmentProfile(developmentEnvironment, session)
	first, second := "unit-"+strings.Repeat("a", 64), "unit-"+strings.Repeat("b", 64)
	payload, err := profile.payload(first, second)
	if err != nil {
		t.Fatal(err)
	}
	root := environmentTarFixture(t, nil, "")
	state, err := scanEnvironmentState(context.Background(), bytes.NewReader(append(root[:512], payload...)), environmentStateLimits{entries: 64, bytes: 32 << 10, archiveBytes: 64 << 10})
	if err != nil || profile.verifyPlacement(state, first, second) != nil {
		t.Fatal("unit fixtures unavailable", err)
	}
	if _, ok := state.files["project/.env"]; ok {
		t.Fatal("unit fixture leaked to whole-volume root")
	}
	delete(state.files, second+"/project/.env")
	if profile.verifyPlacement(state, first) != nil || profile.verifyPlacement(state, first, second) == nil {
		t.Fatal("one unit substituted for another unit's fixture")
	}
	for _, roots := range [][]string{{""}, {"../escape"}, {first, first}, {first + "/child"}, {"unit-" + strings.Repeat("A", 64)}} {
		if _, err := profile.payload(roots...); err == nil {
			t.Fatal("invalid unit roots accepted", roots)
		}
	}
}

func TestInspectionEnvironmentAppliesToExistingPythonRuntimeArguments(t *testing.T) {
	session, _ := domain.NewSandboxSessionID()
	profile, _ := newInspectionEnvironmentProfile(developmentEnvironment, session)
	volume := closureVolume{name: "haa-closure-environment", transaction: session.String(), manifestID: profile.identity, createdAt: "2026-11-03T00:00:00Z", mountpoint: "/var/lib/docker/volumes/haa-closure-environment/_data", capacity: 1 << 20, projectBuildDestination: environmentWorkspaceRoot}
	closure := volume
	closure.name = "haa-closure-site"
	closure.projectBuildDestination = ""
	base, err := pythonObservationCreateArguments(session.String(), 5, phaseObservation, closure, ObservationResourceLease{CgroupParent: "fixed-parent", CPU: 2}, artifactpypi.PublicPyPIProfile().ResourcePolicy())
	if err != nil {
		t.Fatal(err)
	}
	subpath := "unit-" + strings.Repeat("a", 64)
	args, err := applyEnvironmentWorkspace(base, pythonImageReference, profile, volume, false, subpath)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, "\x00")
	for _, required := range []string{pythonImageReference, "--cpuset-cpus\x002", "--cgroup-parent\x00fixed-parent", "target=" + pythonSitePath + ",volume-nocopy,readonly", "target=/tmp,volume-nocopy,volume-subpath=" + subpath, "--annotation\x00" + environmentWorkspacePolicy, "HOME=/tmp", "GITHUB_TOKEN=haa-invalid-", boundaryContainerCommand()} {
		if !strings.Contains(joined, required) {
			t.Fatalf("existing Python boundary lost: %s", required)
		}
	}
	if strings.Count(joined, "--mount\x00") != 2 || strings.Count(joined, "--label\x00"+closureManifestLabel+"=") != 1 || strings.Contains(joined, "--tmpfs\x00/tmp:") {
		t.Fatal("duplicated or conflicting workspace/closure configuration")
	}
	if _, err := applyEnvironmentWorkspace(base, "foreign-image", profile, volume, false, subpath); err == nil {
		t.Fatal("unlocked environment image accepted")
	}
}
