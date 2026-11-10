package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type cargoUpdateCLI struct {
	calls    int
	selector string
	adopt    bool
	install  domain.InstallContext
	set      domain.ProjectVerifiedSet
	failure  error
}

func (f *cargoUpdateCLI) Update(_ context.Context, install domain.InstallContext, selector string, adopt bool) (domain.ProjectVerifiedSet, error) {
	f.calls++
	f.install, f.selector, f.adopt = install, selector, adopt
	return f.set, f.failure
}

func TestCargoUpdateCLIRejectsUnsupportedInputBeforeComposition(t *testing.T) {
	for _, tail := range [][]string{{"itoa"}, {"-p", ""}, {"-p", "itoa", "--package", "itoa"}, {"-p", "itoa,serde"}, {"--precise", "1.0.17"}, {"--offline"}, {"--locked"}, {"--workspace"}, {"--recursive"}, {"--breaking"}, {"--dry-run"}, {"--manifest-path", "/tmp/foreign"}, {"--config", "foreign"}, {"--target", "/tmp/foreign"}, {"-p", "https://example.invalid"}, {"-p", "itoa@1.0"}} {
		for _, preflight := range []bool{false, true} {
			var out, diag bytes.Buffer
			root, _ := cli.New(&out, &diag)
			f := &cargoUpdateCLI{}
			if err := cli.AddCargoUpdate(root, f); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"cargo", "update"}, tail...)
			var err error
			if preflight {
				_, err = cli.PreparePresentation(root, args)
			} else {
				root.SetArgs(args)
				err = root.ExecuteContext(context.Background())
			}
			if err == nil || f.calls != 0 {
				t.Fatalf("unsupported input reached updater: %v %v", args, err)
			}
		}
	}
}

func TestCargoUpdateCLIPreflightDoesNotDuplicatePackageSelection(t *testing.T) {
	for _, tail := range [][]string{nil, {"-p", "itoa"}, {"--package", "itoa@1.0.16", "--adopt"}} {
		for _, preflight := range []bool{false, true} {
			var out, diag bytes.Buffer
			root, _ := cli.New(&out, &diag)
			primary := errors.New("trusted update failure")
			f := &cargoUpdateCLI{failure: primary}
			if err := cli.AddCargoUpdate(root, f); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"--output", "json", "cargo", "update"}, tail...)
			if preflight {
				if source, err := cli.PreparePresentation(root, args); err != nil || source != "cargo" {
					t.Fatal(source, err)
				}
			}
			root.SetArgs(args)
			if err := root.ExecuteContext(context.Background()); !errors.Is(err, primary) {
				t.Fatalf("primary failure lost: %v", err)
			}
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			selector := ""
			if len(tail) != 0 {
				selector = tail[1]
			}
			if f.calls != 1 || f.selector != selector || f.adopt != (len(tail) == 3) || f.install.Target().String() != cwd {
				t.Fatalf("selection/context lost: %+v", f)
			}
			var doc map[string]json.RawMessage
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil || string(doc["operation_status"]) != `"FAILED"` || string(doc["promotion_status"]) != `"NOT_COMPLETED"` || bytes.Contains(out.Bytes(), []byte("\x1b[")) {
				t.Fatal(out.String(), err)
			}
		}
	}
}

// Synthetic independent facts test presentation only, not public authentication.
func cargoUpdateFacts(t *testing.T, decision domain.Decision) (domain.InspectedProjectSet, domain.PolicyDecision) {
	return projectUpdateFactsForSource(t, decision, "crates-io")
}

func projectUpdateFactsForSource(t *testing.T, decision domain.Decision, sourceID string) (domain.InspectedProjectSet, domain.PolicyDecision) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(cwd)
	install, _ := domain.NewInstallContext(target)
	source, _ := domain.NewSourceID(sourceID)
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	name, version, variant := "itoa", "1.0.17", "crate"
	verification, inspection := "cargo-registry-integrity", "cargo-crate-static"
	if sourceID == "go-proxy" {
		name, version, variant = "example.com/module", "v1.2.3", "module"
		verification, inspection = "go-module-sumdb", "go-module-static"
	}
	if sourceID == "pypi" {
		install, _ = domain.NewPythonVenvInstallContext(target)
		name, version, variant = "packaging", "25.0", "wheel"
		verification, inspection = "pypi-declared-sha256", "pypi-static-archive"
	}
	if sourceID == "terraform-registry" {
		name, version, variant = "hashicorp_random", "3.7.2", "linux/amd64"
		verification, inspection = "terraform-signed-package", "terraform-provider-static"
	}
	id, _ := domain.NewResolvedArtifactIdentity(source, name, version, variant)
	resolved, _ := domain.NewResolvedArtifact(id, "https://static.crates.io/crates/itoa/itoa-1.0.17.crate", "sha256:"+digest.String())
	manifest, _ := domain.NewProjectControlDigest("Cargo.toml", digest)
	lock, _ := domain.NewProjectControlDigest("Cargo.lock", digest)
	snapshot, err := domain.NewProjectDependencySnapshot(install, source, []domain.ProjectControlDigest{manifest, lock}, []domain.ResolvedArtifact{resolved}, digest)
	if err != nil {
		t.Fatal(err)
	}
	run, _ := domain.NewRunID()
	node, _ := domain.ProjectDependencyNodeID(id)
	acquired, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, digest, "intake:"+run.String()+":"+variant, 10, resolved.DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	var checks []domain.CheckExecution
	var evidence []domain.EvidenceReference
	specs := []struct {
		id   string
		kind domain.CheckKind
	}{{verification, domain.CheckVerification}, {inspection, domain.CheckInspection}}
	if sourceID == "terraform-registry" {
		specs = append(specs, struct {
			id   string
			kind domain.CheckKind
		}{"terraform-provider-dynamic", domain.CheckInspection})
	}
	for _, spec := range specs {
		checkID, _ := domain.NewCheckID(spec.id)
		check, _ := domain.NewCheckExecution(checkID, spec.kind, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		checks = append(checks, check)
		eid, _ := domain.NewEvidenceID(spec.id + "-result")
		ref, _ := domain.NewEvidenceReference(eid, "evidence:"+run.String()+":"+eid.String())
		evidence = append(evidence, ref)
	}
	policy, _ := domain.NewPolicyDecision(decision, "fixture-policy", 1, []string{"FIXTURE_DECISION"})
	entry, err := domain.NewDependencyInspection(node, run, acquired, checks, evidence, policy)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, []domain.DependencyInspection{entry})
	if err != nil {
		t.Fatal(err)
	}
	return inspected, policy
}

type cargoFailedUpdateFacts struct {
	inspected domain.InspectedProjectSet
	decision  domain.PolicyDecision
	cause     error
}

func (f cargoFailedUpdateFacts) Error() string                         { return f.cause.Error() }
func (f cargoFailedUpdateFacts) Unwrap() error                         { return f.cause }
func (f cargoFailedUpdateFacts) Inspected() domain.InspectedProjectSet { return f.inspected }
func (f cargoFailedUpdateFacts) Decision() domain.PolicyDecision       { return f.decision }

func TestCargoUpdateReportSeparatesInspectionAndCompletedPublication(t *testing.T) {
	for _, kind := range []string{"completed", "policy-block", "publication-failed"} {
		for _, output := range []string{"json", "human"} {
			for _, lang := range []string{"en", "ko"} {
				t.Run(kind+"/"+output+"/"+lang, func(t *testing.T) {
					decision := domain.DecisionAllow
					if kind == "policy-block" {
						decision = domain.DecisionBlock
					}
					inspected, policy := cargoUpdateFacts(t, decision)
					f := &cargoUpdateCLI{}
					primary := errors.New("trusted publication or policy failure")
					if kind == "completed" {
						var err error
						f.set, err = domain.NewProjectVerifiedSet(inspected, policy)
						if err != nil {
							t.Fatal(err)
						}
					} else {
						f.failure = cargoFailedUpdateFacts{inspected, policy, primary}
					}
					var out, diag bytes.Buffer
					root, _ := cli.New(&out, &diag)
					if err := cli.AddCargoUpdate(root, f); err != nil {
						t.Fatal(err)
					}
					root.SetArgs([]string{"--lang", lang, "--output", output, "--color", "never", "cargo", "update"})
					err := root.ExecuteContext(context.Background())
					if (err == nil) != (kind == "completed") || kind != "completed" && !errors.Is(err, primary) {
						t.Fatal(err)
					}
					if output == "json" {
						var doc struct {
							Schema    string `json:"schema_version"`
							Status    string `json:"operation_status"`
							Promotion string `json:"promotion_status"`
							Entries   []struct {
								Checks   []json.RawMessage `json:"checks"`
								Evidence []json.RawMessage `json:"evidence_references"`
							} `json:"entries"`
							Policy struct {
								Decision string `json:"decision"`
							} `json:"policy"`
						}
						if err := json.Unmarshal(out.Bytes(), &doc); err != nil || doc.Schema != "helox.project-update-result/v1" || len(doc.Entries) != 1 || len(doc.Entries[0].Checks) != 2 || len(doc.Entries[0].Evidence) != 2 || doc.Policy.Decision != string(decision) || bytes.Contains(out.Bytes(), []byte("\x1b[")) {
							t.Fatal(out.String(), err)
						}
						if (doc.Promotion == "COMPLETED") != (kind == "completed") || (doc.Status == "COMPLETED") != (kind == "completed") {
							t.Fatal("inspection ALLOW became publication success", out.String())
						}
					} else {
						if !strings.Contains(out.String(), "itoa@1.0.17") || !strings.Contains(out.String(), "helox cargo build") || lang == "ko" && (!strings.Contains(out.String(), "잠금파일 업데이트:") || !strings.Contains(out.String(), "정적 검사")) {
							t.Fatal(out.String())
						}
					}
				})
			}
		}
	}
}

func TestProjectFailureBeforePolicyDoesNotInventADecision(t *testing.T) {
	var out bytes.Buffer
	root, err := cli.New(&out, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	inspected, _ := cargoUpdateFacts(t, domain.DecisionAllow)
	cause := errors.New("local source verification failed before project Policy")
	updater := &cargoUpdateCLI{failure: cargoFailedUpdateFacts{inspected, domain.PolicyDecision{}, cause}}
	if err = cli.AddCargoUpdate(root, updater); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"--output", "json", "cargo", "update"})
	if err = root.ExecuteContext(context.Background()); !errors.Is(err, cause) {
		t.Fatal("original project error lost", err)
	}
	var report map[string]json.RawMessage
	if err = json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if _, present := report["policy"]; present {
		t.Fatal("missing Policy was reported as a decision", out.String())
	}
	if string(report["operation_status"]) != `"FAILED"` || string(report["promotion_status"]) != `"NOT_COMPLETED"` {
		t.Fatal("failed project claimed completion")
	}
}

func (f *cargoUpdateCLI) Fetch(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	return f.Update(ctx, install, "", false)
}

func TestCargoFetchReportsOnlyDependencyPreparation(t *testing.T) {
	for _, failed := range []bool{false, true} {
		for _, output := range []string{"json", "human"} {
			t.Run(output+map[bool]string{false: "-complete", true: "-failed"}[failed], func(t *testing.T) {
				var out bytes.Buffer
				root, err := cli.New(&out, io.Discard)
				if err != nil {
					t.Fatal(err)
				}
				inspected, decision := cargoUpdateFacts(t, domain.DecisionAllow)
				verified, err := domain.NewProjectVerifiedSet(inspected, decision)
				if err != nil {
					t.Fatal(err)
				}
				fetcher := &cargoUpdateCLI{set: verified}
				primary := errors.New("controlled cache preparation failure")
				if failed {
					fetcher.failure = cargoFailedUpdateFacts{inspected, decision, primary}
				}
				if err = cli.AddCargoFetch(root, fetcher); err != nil {
					t.Fatal(err)
				}
				args := []string{"--lang", "en", "--output", output, "cargo", "fetch"}
				if source, err := cli.PreparePresentation(root, args); err != nil || source != "cargo" {
					t.Fatal("fetch route", source, err)
				}
				root.SetArgs(args)
				err = root.ExecuteContext(context.Background())
				if failed && !errors.Is(err, primary) || !failed && err != nil || fetcher.calls != 1 || fetcher.selector != "" || fetcher.adopt {
					t.Fatal("fetch lost guarded operation", err)
				}
				if output == "json" {
					var report struct{ Schema, Operation, Scope, Status, Promotion string }
					var fields map[string]json.RawMessage
					if err = json.Unmarshal(out.Bytes(), &fields); err != nil {
						t.Fatal(err)
					}
					decode := func(key string) string {
						var text string
						if err := json.Unmarshal(fields[key], &text); err != nil {
							t.Fatal(key, err)
						}
						return text
					}
					report.Schema = decode("schema_version")
					report.Operation = decode("operation")
					report.Scope = decode("inspection_scope")
					report.Status = decode("operation_status")
					report.Promotion = decode("promotion_status")
					if report.Schema != "helox.project-fetch-result/v1" || report.Operation != "FETCH" || report.Scope != "DEPENDENCY_PREPARATION" {
						t.Fatal("fetch claimed another operation", out.String())
					}
					if failed && (report.Status != "FAILED" || report.Promotion != "NOT_COMPLETED") || !failed && (report.Status != "COMPLETED" || report.Promotion != "COMPLETED") {
						t.Fatal("fetch failure became completed", out.String())
					}
				} else if !strings.Contains(out.String(), "Dependency preparation:") || !strings.Contains(out.String(), "were not executed") {
					t.Fatal("fetch scope absent", out.String())
				}
			})
		}
	}
}

func TestCargoFetchRejectsUnsupportedInputBeforeUseCase(t *testing.T) {
	for _, tail := range [][]string{{"extra"}, {"--adopt"}, {"-p", "itoa"}, {"--target", "foreign"}, {"--config", "foreign"}, {"--offline"}} {
		root, err := cli.New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		fetcher := &cargoUpdateCLI{}
		if err = cli.AddCargoFetch(root, fetcher); err != nil {
			t.Fatal(err)
		}
		args := append([]string{"cargo", "fetch"}, tail...)
		if _, err = cli.PreparePresentation(root, args); err == nil || fetcher.calls != 0 {
			t.Fatal("unsupported fetch input reached operation", args, err)
		}
	}
}

// The legacy project binders must expose existing trusted facts without
// performing another selection or changing the original error contract.
func TestLegacyExactProjectFailureKeepsRecordedFactsAndFirstCause(t *testing.T) {
	for _, source := range []string{"go-proxy", "go-download", "crates-io", "terraform-registry"} {
		for _, output := range []string{"human", "auto"} {
			for _, withFacts := range []bool{true, false} {
				t.Run(source+"/"+output+map[bool]string{true: "/recorded", false: "/early"}[withFacts], func(t *testing.T) {
					factsSource := source
					if source == "go-download" {
						factsSource = "go-proxy"
					}
					inspected, decision := projectUpdateFactsForSource(t, domain.DecisionBlock, factsSource)
					primary := errors.New("controlled project rejection")
					resolver := &legacyProjectFailureResolver{failure: primary}
					if withFacts {
						resolver.failure = cargoFailedUpdateFacts{inspected, decision, primary}
					}
					var out, diag bytes.Buffer
					root, err := cli.New(&out, &diag)
					if err != nil {
						t.Fatal(err)
					}
					var args []string
					switch source {
					case "go-proxy":
						err = cli.AddGoModuleGet(root, resolver)
						args = []string{"go", "get", "example.com/module@v1.2.3"}
					case "go-download":
						err = cli.AddGoModuleDownload(root, legacyDownloadFailureResolver{resolver})
						args = []string{"go", "mod", "download"}
					case "crates-io":
						err = cli.AddCargoAdd(root, resolver)
						args = []string{"cargo", "add", "itoa@1.0.17"}
					case "terraform-registry":
						err = cli.AddTerraformInit(root, resolver)
						args = []string{"terraform", "init", "hashicorp/random@3.7.2"}
					}
					if err != nil {
						t.Fatal(err)
					}
					root.SetArgs(append([]string{"--lang", "en", "--color", "never", "--output", output}, args...))
					err = root.ExecuteContext(context.Background())
					if !errors.Is(err, primary) || resolver.calls != 1 {
						t.Fatal("lost primary cause or repeated selection", err, resolver.calls)
					}
					if withFacts {
						entry := inspected.Inspections()[0]
						for _, text := range []string{"FIXTURE_DECISION", entry.RunID().String(), entry.Artifact().Identity().Name()} {
							if !strings.Contains(out.String(), text) {
								t.Fatal("recorded project failure facts omitted", text, out.String())
							}
						}
					} else if out.Len() != 0 {
						t.Fatal("early error fabricated inspection facts", out.String())
					}
				})
			}
		}
	}
}

type legacyProjectFailureResolver struct {
	failure error
	calls   int
}

func (r *legacyProjectFailureResolver) Get(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	r.calls++
	return domain.DependencyResolution{}, r.failure
}
func (r *legacyProjectFailureResolver) Resolve(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	return r.Get(ctx, reference, install)
}
func (r *legacyProjectFailureResolver) Init(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (application.TerraformInitResult, error) {
	_, err := r.Get(ctx, reference, install)
	return application.TerraformInitResult{}, err
}

type legacyDownloadFailureResolver struct{ *legacyProjectFailureResolver }

func (r legacyDownloadFailureResolver) Resolve(ctx context.Context, install domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	_, err := r.Get(ctx, domain.ArtifactReference{}, install)
	return domain.ProjectDependencySnapshot{}, err
}
