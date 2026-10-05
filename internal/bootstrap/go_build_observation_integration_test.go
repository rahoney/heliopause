package bootstrap_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxGoDependencyFreeRetainedBuildObservationIntegration(t *testing.T) {
	goRetainedBuildObservationIntegration(t, "", "package main\nfunc main() {}\n")
}

func TestLinuxGoPublicRetainedBuildObservationIntegration(t *testing.T) {
	goRetainedBuildObservationIntegration(t, "github.com/spf13/pflag@v1.0.9", "package main\nimport \"github.com/spf13/pflag\"\nfunc main() { pflag.Parse() }\n")
}

func TestLinuxGoTransitiveRetainedBuildObservationIntegration(t *testing.T) {
	goRetainedBuildObservationIntegration(t, "google.golang.org/grpc@v1.76.0", "package main\nimport \"google.golang.org/grpc\"\nfunc main() { _ = grpc.NewServer() }\n")
}

func TestLinuxGoExplicitInvalidTestdataBuildIntegration(t *testing.T) {
	goRetainedBuildSelectionIntegration(t, "", "package main\nfunc main() {}\n", "./testdata", true)
}

// This gate starts with actual CLI acquisition/verification/inspection/Policy
// and retained cache approval, then passes the guard's source/cache snapshots
// to the observed builder. It does not qualify output publication or build CLI.
func goRetainedBuildObservationIntegration(t *testing.T, reference, main string) {
	t.Helper()
	goRetainedBuildSelectionIntegration(t, reference, main, "./...", false)
}

func goRetainedBuildSelectionIntegration(t *testing.T, reference, main, selector string, invalid bool) {
	t.Helper()
	if os.Getenv("HELOX_GO_RETAINED_BUILD_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Go qualification requires Linux amd64")
	}
	ctx, root, project := prepareGoBuildIntegrationProject(t, reference, main)
	state := filepath.Join(root, "cache", "heliopause")
	evidence, err := local.NewStore(filepath.Join(state, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	cache, err := promotion.NewGoVerifiedCache(filepath.Join(state, "intake"), filepath.Join(state, "evidence"), filepath.Join(state, "go-verified-cache"), evidence)
	if err != nil {
		t.Fatal(err)
	}
	promoter, err := promotion.NewGoProjectPromotion(cache, filepath.Join(state, "go-projects"))
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	guard, err := promoter.Begin(ctx, install)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := guard.Close(); err != nil {
			t.Error(err)
		}
	}()
	snapshot, _, err := guard.OpenBuildInputs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	source, err := guard.SnapshotBuildSource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	inputCache, err := guard.SnapshotBuildCache(ctx)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		t.Fatal(err)
	}
	observer, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(start, remote, output)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	}()
	builder, err := sandbox.NewLinuxObservedGoBuilder(filepath.Join(state, "intake"), executor, observer.Observer())
	if err != nil {
		t.Fatal(err)
	}
	result, err := builder.ObserveBuild(ctx, snapshot, source, inputCache, selector)
	if invalid {
		if err == nil || result.Status() != domain.SandboxIncomplete || !strings.Contains(err.Error(), "command_status=EXIT_STATUS_1") {
			t.Fatalf("explicit invalid source did not preserve compiler failure: %v status=%s", err, result.Status())
		}
	} else if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("actual retained-cache observed build: %v status=%s", err, result.Status())
	}
	for _, observed := range result.Observations() {
		if observed.Category() == domain.ObservationNetwork || observed.Category() == domain.ObservationHoneytoken || observed.Subject() == "filesystem-outside-workspace" || observed.Subject() == "process-exec-unexpected" {
			t.Fatal("normal retained-cache build crossed controlled boundary")
		}
	}
	if err := guard.VerifyUnchanged(ctx); err != nil {
		t.Fatal(err)
	}
	if err := guard.VerifyBuildSource(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("actual retained-cache build reference=%s graph=%s source=%s cache=%s observations=%d", reference, snapshot.GraphDigest(), source.Digest(), inputCache.Digest(), len(result.Observations()))
}

// A fresh project and controlled CLI approval are common inputs for observation
// and whole build gates; their distinct completion assertions remain separate.
func prepareGoBuildIntegrationProject(t *testing.T, reference, main string) (context.Context, string, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct{ name, body string }{{"go.mod", "module example.com/haa-retained-build\ngo 1.26\n"}, {"main.go", main}} {
		if err := os.WriteFile(filepath.Join(project, file.name), []byte(file.body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(project, "testdata"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "testdata", "bad.go"), []byte("this is deliberately invalid Go source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Chdir(project)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	t.Cleanup(cancel)
	var stdout, stderr bytes.Buffer
	args := []string{"go", "mod", "download"}
	if reference != "" {
		args = []string{"go", "get", reference}
	}
	if err := bootstrap.Run(ctx, args, &stdout, &stderr); err != nil {
		t.Fatalf("actual retained approval: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	return ctx, root, project
}
