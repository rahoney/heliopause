//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestBunPublicColdToolRunSelectionIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_TOOL_RUN_SELECTION_INTEGRATION") != "1" {
		t.Fatal("requires ordinary-user observed public Bun one-off selection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	var observed TrustedExecutor = executor
	if directory := os.Getenv("HELOX_BUN_RUN_CAPTURE_DIR"); directory != "" {
		observed = nativeRunCaptureExecutor{Executor: executor, directory: directory, cacheRoot: bunRunCacheBase, archiveName: "bun-cold-cache.tar"}
	}
	resolver, err := NewLinuxBunResolverWithExecutorAndPolicy(observed, supervisor.Observer(), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	locator, name := "typescript@5.9.3", "typescript"
	if os.Getenv("HELOX_BUN_RUN_SELECTION_BARE") == "1" {
		locator, name = "semver", "semver"
	}
	reference, _ := artifactnpm.ParseReference(locator)
	selected, err := resolver.SelectToolRun(ctx, reference)
	if err != nil || selected.Root() == "" || len(selected.Controls()) != 2 || len(selected.Selection().Roots()) != 1 {
		var exited *exec.ExitError
		if errors.As(err, &exited) && len(exited.Stderr) != 0 {
			t.Logf("fixed native selection stderr=%q", exited.Stderr)
		}
		t.Fatal("public cold native DATA selection incomplete", err)
	}
	source, err := artifactnpm.NewPublicResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bound, err := source.BindBunLock(ctx, selected.Selection())
	if err != nil || !bound.SourceBound() || len(bound.Artifacts()) == 0 {
		t.Fatal("independent native source binding incomplete", err)
	}
	found := false
	for _, artifact := range bound.Artifacts() {
		if artifact.Identity().Name() == name && (name != "typescript" || artifact.Identity().Version() == "5.9.3") {
			found = true
		}
	}
	if !found {
		t.Fatal("requested primary absent from independently authenticated graph")
	}
	t.Logf("bun_cold_run request=%s roots=1 instances=%d cache_entries=%d source_artifacts=%d command=SEALED_MISSING_PATH native_DATA=UNAPPROVED Policy/program=NOT_RUN", locator, len(selected.Selection().Packages()), len(selected.Layout().Entries()), len(bound.Artifacts()))
}
