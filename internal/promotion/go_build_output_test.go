package promotion

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongo "github.com/rahoney/heliopause/internal/inspection/gomodule"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationgo "github.com/rahoney/heliopause/internal/verification/gomodule"
)

type goOutputMutation struct{ owner *GoProjectPromotion }

func (m goOutputMutation) BeginBuild(ctx context.Context, install domain.InstallContext) (ports.ProjectBuildGuard, error) {
	return m.owner.Begin(ctx, install)
}

type goOutputBackend struct {
	t      *testing.T
	intake string
	mode   string
	inputs domain.ProjectBuildInputs
	output domain.AcquiredArtifact
}

func (b *goOutputBackend) Build(ctx context.Context, inputs domain.ProjectBuildInputs) (domain.ProjectBuildObservation, error) {
	b.inputs = inputs
	if b.mode == "observer-failure" {
		return domain.ProjectBuildObservation{}, errors.New("trusted observer failure")
	}
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	header := &tar.Header{Name: "program", Mode: 0o755, Typeflag: tar.TypeReg, Uid: 1000, Gid: 1000, Size: 7}
	if b.mode == "output-link" {
		header.Typeflag = tar.TypeSymlink
		header.Linkname = "/outside"
		header.Size = 0
	}
	if err := writer.WriteHeader(header); err != nil {
		b.t.Fatal(err)
	}
	if header.Typeflag == tar.TypeReg {
		if _, err := writer.Write([]byte("fixture")); err != nil {
			b.t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		b.t.Fatal(err)
	}
	output, err := artifactgo.CaptureBuildOutput(ctx, b.intake, inputs, &archive)
	if err != nil {
		return domain.ProjectBuildObservation{}, err
	}
	b.output = output.Artifact()
	if err := output.Close(true); err != nil {
		return domain.ProjectBuildObservation{}, err
	}
	if b.mode == "output-tamper" {
		name := filepath.Join(b.intake, inputs.RunID().String(), "go-output.tar")
		if err := os.Chmod(name, 0o600); err != nil {
			b.t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("tampered"), 0o400); err != nil {
			b.t.Fatal(err)
		}
		if err := os.Chmod(name, 0o400); err != nil {
			b.t.Fatal(err)
		}
	}
	var facts []domain.SandboxObservation
	subject := map[string]string{"network": "network-attempt", "process": "process-exec-unexpected", "filesystem": "filesystem-outside-workspace", "resource": "resource-limit"}[b.mode]
	if subject != "" {
		category := map[string]domain.ObservationCategory{"network": domain.ObservationNetwork, "process": domain.ObservationProcess, "filesystem": domain.ObservationFilesystem, "resource": domain.ObservationResource}[b.mode]
		fact, _ := domain.NewSandboxObservation(category, subject)
		facts = append(facts, fact)
	}
	completed, _ := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
	facts = append(facts, completed)
	session, _ := domain.NewSandboxSessionID()
	result, _ := domain.NewSandboxResult(session, domain.SandboxCompleted, "", facts)
	recipe, _ := domain.NewSHA256Digest(strings.Repeat("f", 64))
	binding, err := domain.NewDerivationBinding(inputs.Source().Digest(), []domain.ContentDigest{inputs.Cache().Digest(), inputs.Snapshot().GraphDigest()}, "go-build-linux-amd64", recipe)
	if err != nil {
		return domain.ProjectBuildObservation{}, err
	}
	return domain.NewProjectBuildObservation(inputs, result, b.output, binding)
}

func TestGoBuildApplicationPublishesOnlyRecordedBoundApproval(t *testing.T) {
	for _, mode := range []string{"normal", "observer-failure", "network", "process", "filesystem", "resource", "output-link", "output-tamper", "source-drift", "cache-drift", "evidence-missing", "before-fault", "after-fault", "existing-output", "stage-tamper", "foreign-output", "parent-replacement"} {
		t.Run(mode, func(t *testing.T) {
			p, guard := retainedGoBuildFixture(t)
			install := guard.context
			project := guard.plan.root
			writeGoBuildSource(t, project, "main.go", "package main\nfunc main() {}\n")
			if err := guard.Close(); err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(p.cache.evidenceRoot)
			if err != nil {
				t.Fatal(err)
			}
			reader, err := sandbox.NewGoBuildSourceReader(p.cache.intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			verifier, err := verificationgo.NewBuildSourceVerifier(reader)
			if err != nil {
				t.Fatal(err)
			}
			backend := &goOutputBackend{t: t, intake: p.cache.intakeRoot, mode: mode}
			inspector, err := inspectiongo.NewBuildInspector(backend)
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewProjectBuildService(goOutputMutation{p}, verifier, inspector, store, policy.M3{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			p.buildCheckpoint = func(phase string) error {
				name := filepath.Join(project, ".heliopause", "builds", backend.inputs.RunID().String())
				stage := filepath.Join(project, ".heliopause", "builds", ".haa-go-output-"+backend.inputs.RunID().String())
				if phase == "BEFORE_PUBLISH" {
					switch mode {
					case "source-drift":
						writeGoBuildSource(t, project, "main.go", "package main\nfunc main() { panic(1) }\n")
					case "cache-drift":
						entries, err := os.ReadDir(p.cache.cacheRoot)
						if err != nil || len(entries) == 0 {
							t.Fatal(err)
						}
						writeGoBuildSource(t, filepath.Join(p.cache.cacheRoot, entries[0].Name(), "modcache"), "poison.go", "poison")
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
				if phase == "AFTER_PUBLISH" {
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
			result, published, err := service.Build(context.Background(), install, "./...")
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
			if _, err := os.Lstat(filepath.Join(project, ".heliopause-go-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("guard was not released")
			}
			stages, err := filepath.Glob(filepath.Join(project, ".heliopause", "builds", ".haa-go-output-*"))
			if err != nil || len(stages) != 0 {
				t.Fatal("partial output staging retained")
			}
		})
	}
}
