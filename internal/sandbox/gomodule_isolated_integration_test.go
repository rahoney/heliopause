package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// This gate executes registered kernel observation and the authenticated typed
// network helper. Default unit tests do not stand in for this qualification.
func TestLinuxGoIsolatedResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	goIsolatedResolverFixture(t, [][]string{{"get", "github.com/spf13/pflag@v1.0.9"}, {"mod", "download", "-json", "all"}, {"mod", "graph"}})
}

func TestLinuxGoIsolatedProjectResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_RESOLVER_PROJECT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	goIsolatedResolverFixture(t, [][]string{{"mod", "download", "-json", "all"}, {"mod", "graph"}})
}

func TestLinuxGoSourceProjectSnapshotIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_RESOLVER_PROJECT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	mod := []byte("module example.com/haa-fixture\n\ngo 1.26\n\nrequire github.com/spf13/pflag v1.0.9\n")
	sum := []byte("github.com/spf13/pflag v1.0.9 h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY=\ngithub.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=\n")
	goSourceSnapshotFixture(t, mod, sum, 1)
}

func TestLinuxGoDependencyFreeProjectSnapshotIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_RESOLVER_PROJECT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	goSourceSnapshotFixture(t, []byte("module example.com/haa-fixture\n\ngo 1.26\n"), nil, 0)
}

func goSourceSnapshotFixture(t *testing.T, mod, sum []byte, modules int) {
	t.Helper()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}
	isolated, err := newIsolatedGoModuleRunner(runner, goNamedEndpointResolver{}, supervisor.Observer(), func(ctx context.Context) (GoCapability, error) { return ProbeGo(ctx, runner) }, integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	project, cleanup, err := privateGoProjectWorkspace(mod, sum)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	target, err := domain.NewInstallTarget(project)
	if err != nil {
		t.Fatal(err)
	}
	installContext, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewGoModuleResolver(isolated)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	snapshot, err := resolver.ResolveProjectDependencies(ctx, installContext)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Valid() || len(snapshot.Dependencies()) != modules || snapshot.DependencyFree() != (modules == 0) || snapshot.Context() != installContext {
		t.Fatal("actual isolated project snapshot is incomplete")
	}
	t.Logf("actual_project_graph_sha256=%s modules=%d", snapshot.GraphDigest(), len(snapshot.Dependencies()))
}

func goIsolatedResolverFixture(t *testing.T, commands [][]string) {
	t.Helper()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}
	isolated, err := newIsolatedGoModuleRunner(runner, goNamedEndpointResolver{}, supervisor.Observer(), func(ctx context.Context) (GoCapability, error) { return ProbeGo(ctx, runner) }, integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	mod := []byte("module example.com/haa-fixture\n\ngo 1.26\n\nrequire github.com/spf13/pflag v1.0.9\n")
	sum := []byte("github.com/spf13/pflag v1.0.9 h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY=\ngithub.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=\n")
	workspace, cleanup, err := privateGoProjectWorkspace(mod, sum)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	env, cleanupEnv, err := privateGoResolverEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	for _, command := range commands {
		body, err := isolated.RunGo(ctx, workspace, env, command...)
		if err != nil {
			t.Fatalf("isolated %s: %v", strings.Join(command, " "), err)
		}
		if command[0] == "mod" && len(body) == 0 {
			t.Fatal("resolver result empty")
		}
		digest := sha256.Sum256(body)
		t.Logf("isolated_command=%s output_bytes=%d output_sha256=%s", strings.Join(command, " "), len(body), hex.EncodeToString(digest[:]))
	}
}

func TestLinuxGoTidyProjectResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_RESOLVER_PROJECT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	runner := integrationRunner{t: t}
	isolated, err := newIsolatedGoModuleRunner(runner, goNamedEndpointResolver{}, supervisor.Observer(), func(ctx context.Context) (GoCapability, error) { return ProbeGo(ctx, runner) }, integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewGoModuleResolver(isolated)
	if err != nil {
		t.Fatal(err)
	}
	project, cleanup, err := privateGoProjectWorkspace([]byte("module example.com/haa-fixture\n\ngo 1.26.0\n\nrequire github.com/spf13/pflag v1.0.9\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("main.go", "package fixture\nimport _ \"github.com/google/uuid\"\nfunc init(){panic(\"tidy must not execute project code\")}\n")
	write("main_test.go", "package fixture\nimport _ \"github.com/spf13/pflag\"\n")
	write("tagged.go", "//go:build haa_unselected_platform\n\npackage fixture\nimport _ \"github.com/davecgh/go-spew/spew\"\n")
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for _, test := range []string{"imports-tests-tags", "noop", "remove-all"} {
		t.Run(test, func(t *testing.T) {
			if test == "remove-all" {
				write("main.go", "package fixture\nimport _ \"fmt\"\n")
				for _, name := range []string{"main_test.go", "tagged.go"} {
					if err := os.Remove(filepath.Join(project, name)); err != nil {
						t.Fatal(err)
					}
				}
			}
			original, err := readGoFrozenControls(project)
			if err != nil {
				t.Fatal(err)
			}
			source, err := CaptureGoProjectSource(ctx, project)
			if err != nil {
				t.Fatal(err)
			}
			update, selectionErr := resolver.ResolveProjectTidy(ctx, install, original, source)
			closeErr := source.Close()
			if selectionErr != nil || closeErr != nil {
				t.Fatalf("tidy: %v close: %v", selectionErr, closeErr)
			}
			if !update.Valid() || update.HasRequestedResolution() {
				t.Fatal("tidy lost whole-project selection")
			}
			names := map[string]bool{}
			for _, item := range update.Snapshot().Dependencies() {
				names[item.Identity().Name()] = true
			}
			if test == "remove-all" {
				if len(names) != 0 {
					t.Fatal("unused requirements retained")
				}
			} else {
				if len(names) != 3 || !names["github.com/google/uuid"] || !names["github.com/spf13/pflag"] || !names["github.com/davecgh/go-spew"] {
					t.Fatalf("test/tag/import closure=%v", names)
				}
			}
			current, err := readGoFrozenControls(project)
			if err != nil || !sameGoFrozenControls(original, current) {
				t.Fatal("native resolver changed original controls")
			}
			if test == "noop" && !sameGoFrozenControls(original, update.SelectedControls()) {
				t.Fatal("native tidy no-op changed controls")
			}
			for _, control := range update.SelectedControls() {
				if err := os.WriteFile(filepath.Join(project, control.Name()), control.Body(), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("actual_go_tidy=PASS case=%s modules=%d graph=%s project_execution=NOT_REQUESTED", test, len(names), update.Snapshot().GraphDigest())
		})
		if t.Failed() {
			break
		}
	}
}
