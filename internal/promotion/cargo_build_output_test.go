package promotion

import (
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionbuild "github.com/rahoney/heliopause/internal/inspection/projectbuild"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationbuild "github.com/rahoney/heliopause/internal/verification/projectbuild"
	"os"
	"path/filepath"
	"testing"
)

type cargoOutputMutation struct{ owner *CargoProjectPromotion }

func (m cargoOutputMutation) BeginBuild(ctx context.Context, install domain.InstallContext) (ports.ProjectBuildGuard, error) {
	return m.owner.Begin(ctx, install)
}
func TestCargoBuildApplicationPublishesOnlyRecordedBoundApproval(t *testing.T) {
	for _, mode := range []string{"normal", "observer-failure", "network", "process", "filesystem", "resource", "output-link", "output-tamper", "source-drift", "cache-drift", "evidence-missing", "before-fault", "after-fault", "existing-output", "stage-tamper", "foreign-output", "parent-replacement"} {
		t.Run(mode, func(t *testing.T) {
			p, guard, update, staged := approvedCargoFixture(t)
			if err := guard.Commit(context.Background(), update, staged); err != nil {
				t.Fatal(err)
			}
			install := guard.context
			project := guard.plan.root

			if err := guard.Close(); err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(p.cache.evidenceRoot)
			if err != nil {
				t.Fatal(err)
			}
			reader, err := sandbox.NewCargoBuildSourceReader(p.cache.intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			verifier, err := verificationbuild.NewSourceVerifier(reader, "cargo")
			if err != nil {
				t.Fatal(err)
			}
			backend := &goOutputBackend{t: t, intake: p.cache.intakeRoot, mode: mode}
			inspector, err := inspectionbuild.NewInspector(backend, "cargo")
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewProjectBuildService(cargoOutputMutation{p}, verifier, inspector, store, policy.M3{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			p.checkpoint = func(phase string) error {
				name := filepath.Join(project, ".heliopause", "builds", backend.inputs.RunID().String())
				stage := filepath.Join(project, ".heliopause", "builds", ".haa-cargo-output-"+backend.inputs.RunID().String())
				if phase == "BUILD_BEFORE_PUBLISH" {
					switch mode {
					case "source-drift":
						writeGoBuildSource(t, project, "src/main.rs", "fn main() { panic!(\"changed\"); }\n")
					case "cache-drift":
						entries, err := os.ReadDir(p.cache.cacheRoot)
						if err != nil || len(entries) == 0 {
							t.Fatal(err)
						}
						writeGoBuildSource(t, filepath.Join(p.cache.cacheRoot, entries[0].Name(), "vendor"), "poison.go", "poison")
					case "evidence-missing":
						if err := os.RemoveAll(filepath.Join(p.cache.evidenceRoot, backend.inputs.RunID().String())); err != nil {
							t.Fatal(err)
						}
					case "before-fault":
						return errors.New("injected pre-publication fault")
					case "existing-output":
						if err := os.Mkdir(name, 0o700); err != nil {
							t.Fatal(err)
						}
						writeGoBuildSource(t, name, "sentinel", "foreign")
					case "stage-tamper":
						if err := os.Chmod(filepath.Join(stage, "program"), 0o700); err != nil {
							t.Fatal(err)
						}
						writeGoBuildSource(t, stage, "program", "changed")
					}
				}
				if phase == "BUILD_AFTER_PUBLISH" {
					switch mode {
					case "after-fault":
						return errors.New("injected post-publication fault")
					case "foreign-output":
						if err := os.Rename(name, name+"-moved"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(name, 0o700); err != nil {
							t.Fatal(err)
						}
						writeGoBuildSource(t, name, "sentinel", "foreign")
					case "parent-replacement":
						parent := filepath.Dir(name)
						if err := os.Rename(parent, parent+"-moved"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(parent, 0o700); err != nil {
							t.Fatal(err)
						}
						writeGoBuildSource(t, parent, "sentinel", "foreign")
					}
				}
				return nil
			}
			result, published, err := service.Build(context.Background(), install, "default")
			if mode == "normal" {
				if err != nil || !published.Valid() || result.Status() != domain.OperationCompleted || len(result.Checks()) != 2 || len(result.Evidence()) != 3 {
					t.Fatalf("normal guarded output: %v", err)
				}
				body, err := os.ReadFile(filepath.Join(project, published.RelativeDirectory(), "program"))
				if err != nil || string(body) != "fixture" {
					t.Fatalf("published exact output: %v", err)
				}
				if info, err := os.Stat(filepath.Join(project, published.RelativeDirectory(), "program")); err != nil || info.Mode().Perm() != 0o500 {
					t.Fatal("output permissions differ")
				}
				if _, err := os.Stat(filepath.Join(project, published.RelativeDirectory(), ".haa-build.json")); err != nil {
					t.Fatal(err)
				}
			} else {
				if published.Valid() {
					t.Fatal("unapproved or uncertain build published")
				}
				if mode == "network" || mode == "process" || mode == "filesystem" || mode == "resource" {
					decision, ok := result.PolicyDecision()
					if err != nil || !ok || decision.Decision() == domain.DecisionAllow {
						t.Fatalf("raw adverse fact lost: %v", err)
					}
				} else if err == nil {
					t.Fatal("operational failure became success")
				}
				if backend.inputs.Valid() && mode != "foreign-output" && mode != "existing-output" {
					if _, err := os.Lstat(filepath.Join(project, ".heliopause", "builds", backend.inputs.RunID().String())); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("failed owned output retained")
					}
				}
				if mode == "foreign-output" || mode == "existing-output" {
					body, err := os.ReadFile(filepath.Join(project, ".heliopause", "builds", backend.inputs.RunID().String(), "sentinel"))
					if err != nil || string(body) != "foreign" {
						t.Fatal("foreign output was removed")
					}
				}
			}
			if _, err := os.Lstat(filepath.Join(project, ".heliopause-cargo-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("guard was not released")
			}
			stages, err := filepath.Glob(filepath.Join(project, ".heliopause", "builds", ".haa-cargo-output-*"))
			if err != nil || len(stages) != 0 {
				t.Fatal("partial output staging retained")
			}
		})
	}
}
