//go:build linux

package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUVPublicVirtualProjectAddAndUpgradeIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_ADD_INTEGRATION") != "1" {
		t.Skip("requires pinned UV/Python/gVisor and authenticated typed helper")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user caller")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}
	native, err := NewUVResolver(runner, systemNamedEndpointResolver{}, supervisor.Observer(), integrationPythonCapabilityProbe(runner), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewUVProjectResolver(native)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	install := uvSyncTestContext(t)
	manifest := []byte(fmt.Sprintf("[project]\nname='haa-public-add'\nversion='0.1.0'\nrequires-python='==%s'\ndependencies=[]\n", PinnedPythonRuntime().PythonVersion))
	selectProject := func(body, lock []byte, intent artifactpypi.UVProjectIntent, selector string) domain.ProjectDependencyUpdate {
		t.Helper()
		input, err := artifactpypi.ParseUVProjectInput(body, lock, len(lock) != 0, intent, selector)
		if err != nil {
			t.Fatal(err)
		}
		original, err := NewUVProjectOriginalControls(input)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		result, err := resolver.ResolveProjectLockUpdate(ctx, install, original, "")
		if err != nil {
			t.Fatalf("actual UV project intent=%s selector=%s failed: %v", intent, selector, err)
		}
		if !result.Valid() || result.HasRequestedResolution() || len(result.SelectedControls()) != 6 {
			t.Fatal("incomplete project selection")
		}
		before := map[string]domain.ProjectControlFile{}
		for _, c := range original {
			before[c.Name()] = c
		}
		for _, c := range result.OriginalControls() {
			want := before[c.Name()]
			if c.Digest() != want.Digest() || c.Present() != want.Present() {
				t.Fatal("native mutation changed frozen original controls")
			}
		}
		return result
	}
	controls := func(result domain.ProjectDependencyUpdate) (body, lock []byte) {
		t.Helper()
		for _, c := range result.SelectedControls() {
			switch c.Name() {
			case artifactpypi.UVProjectInputControl:
				body = c.Body()
			case artifactpypi.UVNativeLockControl:
				lock = c.Body()
			}
		}
		if len(body) == 0 || len(lock) == 0 {
			t.Fatal("native selected controls absent")
		}
		return
	}
	version := func(result domain.ProjectDependencyUpdate, name string) string {
		t.Helper()
		for _, a := range result.Snapshot().Dependencies() {
			if a.Identity().Name() == name {
				return a.Identity().Version()
			}
		}
		t.Fatal("selected public project missing", name)
		return ""
	}

	old := selectProject(manifest, nil, artifactpypi.UVProjectAdd, "packaging==24.2")
	if version(old, "packaging") != "24.2" {
		t.Fatal("exact native add selected wrong old version")
	}
	oldBody, oldLock := controls(old)
	changed := selectProject(oldBody, oldLock, artifactpypi.UVProjectAdd, "packaging==25.0")
	if version(changed, "packaging") != "25.0" {
		t.Fatal("explicit native add did not change version")
	}
	changedBody, changedLock := controls(changed)
	if bytes.Equal(oldBody, changedBody) || bytes.Equal(oldLock, changedLock) {
		t.Fatal("version change lost native manifest/lock edit")
	}
	retained := selectProject(changedBody, changedLock, artifactpypi.UVProjectAdd, "packaging")
	if version(retained, "packaging") != "25.0" {
		t.Fatal("bare existing add changed locked selection")
	}
	t.Log("actual_native_add_exact_change=24.2->25.0 existing_bare_constraint=retained; complete independent source binding")

	bare := selectProject(manifest, nil, artifactpypi.UVProjectAdd, "idna")
	bareBody, bareLock := controls(bare)
	if version(bare, "idna") == "" {
		t.Fatal("new bare native add selected no version")
	}
	twoRoots := selectProject(bareBody, bareLock, artifactpypi.UVProjectAdd, "packaging==25.0")
	if len(twoRoots.Snapshot().Dependencies()) != 2 || version(twoRoots, "idna") != version(bare, "idna") {
		t.Fatal("add lost unrelated direct dependency or complete set")
	}
	t.Logf("actual_native_new_bare_and_second_root=PASS idna=%s complete_entries=2", version(bare, "idna"))

	// Only the initial seed's root constraint metadata is synthetic. A native
	// sync revalidates/canonicalizes it before the actual version-changing upgrade.
	seedBody := bytes.Replace(oldBody, []byte("packaging==24.2"), []byte("packaging>=24,<26"), 1)
	seedLock := bytes.Replace(oldLock, []byte(`specifier = "==24.2"`), []byte(`specifier = ">=24,<26"`), 1)
	if bytes.Equal(seedBody, oldBody) || bytes.Equal(seedLock, oldLock) {
		t.Fatal("constrained native-readable seed was not constructed")
	}
	locked := selectProject(seedBody, seedLock, artifactpypi.UVProjectSync, "")
	if version(locked, "packaging") != "24.2" {
		t.Fatal("native sync did not retain eligible locked version")
	}
	lockedBody, lockedLock := controls(locked)
	upgraded := selectProject(lockedBody, lockedLock, artifactpypi.UVProjectLockUpgrade, "")
	upgradedBody, upgradedLock := controls(upgraded)
	if version(upgraded, "packaging") == "24.2" || !bytes.Equal(lockedBody, upgradedBody) || bytes.Equal(lockedLock, upgradedLock) {
		t.Fatal("actual lock-upgrade failed version delta or changed manifest")
	}
	t.Logf("actual_native_lock_upgrade_delta=24.2->%s initial_seed=synthetic_constraint native_sync_revalidated=true manifest_unchanged=true Policy=NOT_RUN Host_install=NOT_RUN", version(upgraded, "packaging"))
}
