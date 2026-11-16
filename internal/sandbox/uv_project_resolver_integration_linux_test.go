//go:build linux

package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"sort"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestUVPublicVirtualProjectResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires fixed UV/Python/gVisor and authenticated typed policy helper")
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
	for _, dependencies := range []string{"[]", `["packaging==25.0"]`} {
		manifest := []byte(fmt.Sprintf("[project]\nname='haa-public-project'\nversion='0.1.0'\nrequires-python='==%s'\ndependencies=%s\n", PinnedPythonRuntime().PythonVersion, dependencies))
		input, err := artifactpypi.ParseUVProjectInput(manifest, nil, false, artifactpypi.UVProjectSync, "")
		if err != nil {
			t.Fatal(err)
		}
		original, err := NewUVProjectOriginalControls(input)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		update, err := resolver.ResolveProjectLockUpdate(ctx, install, original, "")
		cancel()
		if err != nil {
			t.Fatalf("first actual virtual project selection failed dependencies=%s: %v", dependencies, err)
		}
		if !update.Valid() || update.HasRequestedResolution() || len(update.SelectedControls()) != 6 {
			t.Fatal("project selection fabricated a primary or lost controls")
		}
		expected := 1
		if dependencies == "[]" {
			expected = 0
		}
		if len(update.Snapshot().Dependencies()) != expected || update.Snapshot().DependencyFree() != (expected == 0) {
			t.Fatal("actual project selection differs from complete input")
		}
		var lock []byte
		for _, c := range update.SelectedControls() {
			if c.Name() == artifactpypi.UVNativeLockControl {
				lock = c.Body()
			}
		}
		upgradeInput, err := artifactpypi.ParseUVProjectInput(manifest, lock, true, artifactpypi.UVProjectLockUpgrade, "")
		if err != nil {
			t.Fatal(err)
		}
		upgradeOriginal, err := NewUVProjectOriginalControls(upgradeInput)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		upgraded, err := resolver.ResolveProjectLockUpdate(ctx, install, upgradeOriginal, "")
		cancel()
		if err != nil || !upgraded.Valid() || len(upgraded.Snapshot().Dependencies()) != expected {
			t.Fatal("actual project lock-upgrade selection failed", err)
		}
		t.Logf("actual_virtual_project_selection=PASS dependencies=%d sync+lock-upgrade; independent source binding; no Policy/Host install claim", expected)
	}
}

func TestUVProjectSelectionOptionsIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires pinned UV/Python/gVisor and authenticated typed policy helper")
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
	manifest := []byte(`[project]
name='haa-options-project'
version='0.1.0'
requires-python='>=3.10,<4'
dependencies=["idna>=3.10; python_version >= '3.12'", "idna<3.9; python_version < '3.12'", "packaging==25.0; sys_platform == 'linux'", "colorama==0.4.6; sys_platform == 'win32'"]
[project.optional-dependencies]
net=['six==1.17.0']
[dependency-groups]
dev=['typing-extensions==4.15.0',{include-group='lint'}]
lint=['packaging==25.0']
empty=[]
[tool.uv]
package=false
default-groups=['lint']
`)
	var seed []byte
	for _, test := range []struct {
		name            string
		selection       artifactpypi.UVProjectSelection
		expected        []string
		stale, refuse   bool
		defaultsChanged bool
		manifest        []byte
	}{
		{name: "default", expected: []string{"idna", "packaging"}},
		{name: "include", selection: artifactpypi.UVProjectSelection{Groups: []string{"dev"}}, expected: []string{"idna", "packaging", "typing-extensions"}},
		{name: "only", selection: artifactpypi.UVProjectSelection{OnlyGroups: []string{"dev"}}, expected: []string{"packaging", "typing-extensions"}},
		{name: "exclude-inline-group", selection: artifactpypi.UVProjectSelection{Groups: []string{"dev"}, NoGroups: []string{"lint"}}, expected: []string{"idna", "packaging", "typing-extensions"}},
		{name: "extra", selection: artifactpypi.UVProjectSelection{Extras: []string{"net"}}, expected: []string{"idna", "packaging", "six"}},
		{name: "all-extras", selection: artifactpypi.UVProjectSelection{AllExtras: true}, expected: []string{"idna", "packaging", "six"}},
		{name: "empty-only", selection: artifactpypi.UVProjectSelection{OnlyGroups: []string{"empty"}}, expected: []string{}},
		{name: "locked", selection: artifactpypi.UVProjectSelection{Locked: true}, expected: []string{"idna", "packaging"}},
		{name: "stale-locked", selection: artifactpypi.UVProjectSelection{Locked: true}, stale: true, refuse: true},
		{name: "stale-frozen", selection: artifactpypi.UVProjectSelection{Frozen: true}, stale: true, expected: []string{"idna", "packaging"}},
		{name: "frozen-current-default", selection: artifactpypi.UVProjectSelection{Frozen: true}, defaultsChanged: true, expected: []string{"idna", "packaging", "typing-extensions"}},
		{name: "dependency-extra", manifest: []byte(`[project]
name='haa-dependency-extra'
version='0.1.0'
requires-python='>=3.10,<4'
dependencies=['requests[socks]==2.32.5','charset-normalizer==2.0.12']
[tool.uv]
package=false
`), expected: []string{"certifi", "charset-normalizer", "idna", "pysocks", "requests", "urllib3"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			current := manifest
			if test.stale {
				current = bytes.ReplaceAll(manifest, []byte("packaging==25.0"), []byte("packaging==24.2"))
			}
			if test.defaultsChanged {
				current = bytes.ReplaceAll(current, []byte("default-groups=['lint']"), []byte("default-groups=['dev']"))
			}
			lockInput := seed
			if test.manifest != nil {
				current, lockInput = test.manifest, nil
			}
			input, err := artifactpypi.ParseUVProjectInput(current, lockInput, len(lockInput) > 0, artifactpypi.UVProjectSync, "", test.selection)
			if err != nil {
				t.Fatal(err)
			}
			controls, err := NewUVProjectOriginalControls(input)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			update, err := resolver.ResolveProjectLockUpdate(ctx, uvSyncTestContext(t), controls, "")
			if test.refuse {
				if err == nil || update.Valid() {
					t.Fatal("stale locked input accepted")
				}
				t.Log("stale_locked=REFUSED no Policy/Host install claim")
				return
			}
			if err != nil {
				t.Fatal("actual UV target selection failed", err)
			}
			if err := ValidateUVProjectUpdate(update); err != nil {
				t.Fatal("saved target proof rejected", err)
			}
			actual := []string{}
			for _, artifact := range update.Snapshot().Dependencies() {
				actual = append(actual, artifact.Identity().Name())
			}
			sort.Strings(actual)
			if !slices.Equal(actual, test.expected) {
				t.Fatalf("native target differs got=%v want=%v", actual, test.expected)
			}
			for _, control := range update.SelectedControls() {
				if control.Name() == artifactpypi.UVNativeLockControl && test.manifest == nil {
					if len(seed) == 0 {
						seed = control.Body()
					} else if !bytes.Equal(seed, control.Body()) {
						t.Fatal("unchanged project selection rewrote lock bytes")
					}
				}
			}
			t.Logf("actual_uv_scope=%s projects=%v lock_reused=%t independent_archive_metadata=true no Policy/Host install claim", test.name, actual, len(lockInput) > 0)
		})
		if t.Failed() {
			break
		}
	}
}
