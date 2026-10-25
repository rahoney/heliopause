//go:build realcorpus

package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// Actual acquisition/static inputs only: no native runtime, dynamic execution,
// Policy or Host publication is exercised by this source consumer.
func TestBunPublicInspectionSourceInputs(t *testing.T) {
	if os.Getenv("HELOX_BUN_PUBLIC_SOURCE") != "1" {
		t.Fatal("declare HELOX_BUN_PUBLIC_SOURCE=1 for public inspection input qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("versioned fixture source unavailable")
	}
	fixtures := filepath.Join(filepath.Dir(file), "..", "artifact", "npm", "testdata", "bun-lock")
	for _, name := range []string{"empty", "single", "transitive", "peer"} {
		if !t.Run(name, func(t *testing.T) {
			intake := t.TempDir()
			source, err := artifactnpm.NewPublicResolver(intake)
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectionnpm.NewInspector(intake)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := os.ReadFile(filepath.Join(fixtures, name+".package.json"))
			if err != nil {
				t.Fatal(err)
			}
			var lockBody []byte
			if name != "empty" {
				lockBody, err = os.ReadFile(filepath.Join(fixtures, name+".bun.lock"))
				if err != nil {
					t.Fatal(err)
				}
			}
			manifestControl, err := domain.NewProjectControlFile("package.json", manifest, true)
			if err != nil {
				t.Fatal(err)
			}
			lockControl, err := domain.NewProjectControlFile("bun.lock", lockBody, name != "empty")
			if err != nil {
				t.Fatal(err)
			}
			lock, err := artifactnpm.ParseBunLock(manifest, lockControl, runtimeidentity.BunVersion)
			if err != nil {
				t.Fatal(err)
			}
			selected, err := source.BindBunLock(ctx, lock)
			if err != nil {
				t.Fatal(err)
			}
			target, err := domain.NewInstallTarget(filepath.Join(t.TempDir(), "project"))
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			frozen, err := selected.InspectionContext(install, []domain.ProjectControlFile{manifestControl, lockControl}, BunResolverRuntimeIdentity())
			if err != nil {
				t.Fatal(err)
			}
			acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
			for _, resolved := range frozen.Snapshot().Dependencies() {
				run, err := domain.NewRunID()
				if err != nil {
					t.Fatal(err)
				}
				archive, err := source.Acquire(ctx, run, resolved)
				if err != nil {
					t.Fatal(err)
				}
				verified, err := (verificationnpm.IntegrityVerifier{}).Verify(ctx, archive)
				if err != nil || verified.Outcome() != domain.VerificationVerified {
					t.Fatal("actual archive integrity unavailable", err)
				}
				inventory, report, err := static.InspectArchive(ctx, archive)
				if err != nil || !inventory.Matches(archive) || len(report.Findings()) != 0 || report.Execution().Status() != domain.ExecutionCompleted {
					t.Fatal("actual archive full static inventory unavailable", err)
				}
				node, err := domain.ProjectDependencyNodeID(archive.Identity())
				if err != nil {
					t.Fatal(err)
				}
				acquired[node] = archive
			}
			input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
			if err != nil || !input.Valid() {
				t.Fatal("complete actual acquired binding failed", err)
			}
			instances := 0
			for _, archive := range input.Artifacts() {
				placements, err := input.InstancesFor(archive)
				if err != nil {
					t.Fatal(err)
				}
				instances += len(placements)
			}
			if instances != len(lock.Packages()) || frozen.Snapshot().DependencyFree() != (name == "empty") {
				t.Fatal("actual frozen instance coverage differs")
			}
			t.Logf("bun_static_source=%s archives=%d instances=%d exact_controls=true; native/dynamic/Policy/Host_install=NOT_EXERCISED", name, len(input.Artifacts()), instances)
		}) {
			return
		}
	}
}
