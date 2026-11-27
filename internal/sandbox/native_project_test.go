package sandbox

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestNativeProjectKindsKeepPNPMSurfaceAndBunMountDistinct(t *testing.T) {
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimeToolKind(99)} {
		if _, ok := nativeProjectIdentity(kind); ok {
			t.Fatal("unsupported tool gained project role")
		}
	}
	bun, ok := nativeProjectIdentity(nativeRuntimeBun)
	if !ok || bun.lock != "bun.lock" || bun.root != bunProjectRoot || bun.profile != bunProjectResolverProfile {
		t.Fatal("Bun project identity differs")
	}
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimePNPM, nativeRuntimeBun} {
		f, volume, _ := nativeVolumeFixture(t, kind)
		spec, _ := nativeProjectIdentity(kind)
		project := closureVolume{name: "haa-closure-project", transaction: "tx", manifestID: strings.Repeat("a", 64), createdAt: "fixed", mountpoint: "/project", capacity: 128 << 20, projectBuildDestination: spec.root}
		args, err := nativeProjectCreateArguments("none", nil, volume, project, false)
		if err != nil {
			t.Fatal(err)
		}
		if kind == nativeRuntimePNPM {
			old, err := pnpmCreateArgumentsWithStore("none", nil, volume, project, false)
			if err != nil || !slices.Equal(args, old) {
				t.Fatal("PNPM create recipe changed", err)
			}
		} else {
			for _, entry := range []string{"BUN_INSTALL_CACHE_DIR=" + bunProjectRoot + "/cache", "XDG_CACHE_HOME=" + bunProjectRoot + "/cache", "TMPDIR=" + bunProjectRoot + "/tmp"} {
				if !slices.Contains(args, entry) {
					t.Fatal("Bun private environment missing", entry)
				}
			}
			project.projectBuildExecutable = true
			if _, err := nativeProjectCreateArguments("none", nil, volume, project, false); err == nil {
				t.Fatal("Bun project gained executable mount")
			}
			project.projectBuildExecutable = false
			if _, err := nativeProjectCreateArguments("none", nil, volume, project, true); err == nil {
				t.Fatal("Bun acquired PNPM early-store mode")
			}
			project.projectBuildDestination = pnpmProjectRoot
			if _, err := nativeProjectCreateArguments("none", nil, volume, project, false); err == nil {
				t.Fatal("Bun acquired PNPM project")
			}
		}
		f.listed = ""
		if err := volume.close(context.Background(), f); err != nil {
			t.Fatal(err)
		}
	}
	query, _ := observerExpectedTopology(bunResolverProfile)
	project, _ := observerExpectedTopology(bunProjectResolverProfile)
	if len(query) != 3 || len(project) != 4 || project[2].Mountpoint != bunProjectRoot || project[2].FSType != "9p" || project[2].Class != "workspace" {
		t.Fatal("Bun query and project topology mixed")
	}
}

func TestBunProjectScopeStopsBeforeNetworkOnCancellationOrForeignTool(t *testing.T) {
	runner := &pnpmControlTransport{}
	endpoints := &pnpmCountingEndpoints{}
	probeCalls := 0
	scope, err := newBunNativeProjectScope(runner, endpoints, &recordingObserver{}, func(context.Context) (Capability, error) { probeCalls++; return Capability{Available: true}, nil }, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("caller stopped Bun producer")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	actionCalls := 0
	if err := scope.run(ctx, false, func(context.Context, string, string, *closureVolume) error { actionCalls++; return nil }); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || runner.calls != 0 || endpoints.calls != 0 || probeCalls != 0 || actionCalls != 0 {
		t.Fatal("canceled Bun did side effects or lost cause", err)
	}
	scope.prepare = func(context.Context) (nativeProjectTool, error) { return &pnpmPreparedTool{}, nil }
	if err := scope.run(context.Background(), false, func(context.Context, string, string, *closureVolume) error { actionCalls++; return nil }); err == nil || runner.calls != 0 || endpoints.calls != 0 || actionCalls != 0 {
		t.Fatal("foreign prepared tool reached Bun network/volume", err)
	}
	scope.kind = nativeRuntimeUV
	if err := scope.run(context.Background(), false, func(context.Context, string, string, *closureVolume) error { return nil }); err == nil || runner.calls != 0 || endpoints.calls != 0 {
		t.Fatal("unsupported native kind reached project resources", err)
	}
}

func TestBunProjectVolumeConstructorUsesOnlyFixedNonexecutableRoot(t *testing.T) {
	runner := &pnpmInstanceRunner{t: t}
	volume, err := createNativeProjectVolume(context.Background(), runner, "bun-volume-fixture", nativeRuntimeBun)
	if err != nil || volume.destination() != bunProjectRoot || volume.projectBuildExecutable || volume.capacity != 128<<20 || !strings.HasSuffix(volume.tmpfsOptions(), ",noexec") {
		t.Fatal("Bun actual constructor contract refused", err)
	}
	if err := closeWritableOutputVolume(context.Background(), runner, &volume, bunProjectRoot); err != nil {
		t.Fatal(err)
	}
	for _, config := range []projectBuildVolumeConfig{{bunProjectRoot, true}, {bunProjectRoot + "/other", false}, {"/home/host", false}} {
		before := runner.calls
		if _, err := createInputVolume(context.Background(), runner, "foreign", strings.Repeat("a", 64), 128<<20, false, config); err == nil || runner.calls != before {
			t.Fatal("unsupported destination/execution reached Docker", err)
		}
	}
}
