package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
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
	if !snapshot.Valid() || len(snapshot.Dependencies()) != 1 || snapshot.Context() != installContext {
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
