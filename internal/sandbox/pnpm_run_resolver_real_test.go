//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/hosttool"
)

type nativeRunCaptureExecutor struct {
	*hosttool.Executor
	directory, cacheRoot, archiveName string
}

func (r nativeRunCaptureExecutor) RunOutput(ctx context.Context, output io.Writer, name string, args ...string) error {
	if name != "docker" || len(args) != 3 || args[0] != "cp" || !strings.HasSuffix(args[1], ":"+r.cacheRoot) || args[2] != "-" {
		return r.Executor.RunOutput(ctx, output, name, args...)
	}
	file, err := os.OpenFile(filepath.Join(r.directory, r.archiveName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	runErr := r.Executor.RunOutput(ctx, io.MultiWriter(output, file), name, args...)
	return errors.Join(runErr, file.Close())
}

func TestPNPMPublicColdToolRunSelectionIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_TOOL_RUN_SELECTION_INTEGRATION") != "1" {
		t.Fatal("requires ordinary-user observed cold pnpm one-off selection")
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
	if directory := os.Getenv("HELOX_PNPM_RUN_CAPTURE_DIR"); directory != "" {
		observed = nativeRunCaptureExecutor{Executor: executor, directory: directory, cacheRoot: pnpmRunCacheBase + "/dlx", archiveName: "pnpm-cold-cache.tar"}
	}
	resolver, err := NewLinuxPNPMResolverWithExecutorAndPolicy(observed, supervisor.Observer(), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactnpm.ParseReference("typescript@5.9.3")
	selected, err := resolver.SelectToolRun(ctx, reference)
	if err != nil || selected.Root() == "" || len(selected.Controls()) != 2 || len(selected.Selection().Roots()) != 1 || len(selected.Selection().Packages()) != 1 {
		var exited *exec.ExitError
		if errors.As(err, &exited) && len(exited.Stderr) != 0 {
			t.Logf("native fixed selection stderr=%q", exited.Stderr)
		}
		t.Fatal("cold native DATA selection incomplete", err)
	}
	source, err := artifactnpm.NewPublicResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bound, err := source.BindPNPMLock(ctx, selected.Selection())
	if err != nil || !bound.SourceBound() || len(bound.Artifacts()) != 1 || bound.Artifacts()[0].Identity().Name() != "typescript" || bound.Artifacts()[0].Identity().Version() != "5.9.3" {
		t.Fatal("independent native source metadata binding missing", err)
	}
	t.Logf("pnpm_cold_run roots=1 packages=1 instances=%d cache_entries=%d source_artifacts=1 selected=typescript@5.9.3 command=SEALED_MISSING_PATH native_DATA=UNAPPROVED", len(selected.Selection().Snapshots()), len(selected.Layout().Entries()))
}
