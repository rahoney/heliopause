//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	evidencerecord "github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

func TestPythonRequirementsPublicInstallationIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_REQUIREMENTS_INTEGRATION") != "1" {
		t.Skip("requires declared protected Python requirements integration")
	}
	if os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user requirements qualification with protected runtime and policy helper")
	}
	if !t.Run("normalized-block-contract", TestRequirementsSyntheticBlockPreservesRealChecks) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	executor, observer := pythonRequirementsIntegrationRuntime(t, ctx)
	for _, client := range []string{"pip", "uv"} {
		t.Run(client, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			policyService, err := newSystemResolverPolicyAdapter()
			if err != nil {
				t.Fatal(err)
			}
			var native ports.DependencyResolver
			if client == "pip" {
				native, err = sandbox.NewLinuxPyPIResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
			} else {
				native, err = sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if closer, ok := native.(interface{ Close() error }); ok {
					if err := closer.Close(); err != nil {
						t.Error(err)
					}
				}
			}()
			intakeRoot, evidenceRoot, stagingRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
			intake, err := artifactpypi.NewPublicIntake(intakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			static, err := newPyPIStaticInspector(intakeRoot, executor, observer.Observer())
			if err != nil {
				t.Fatal(err)
			}
			backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot, executor, observer.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			dynamic, err := inspectionpypi.NewDynamicInspector(backend)
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectionpypi.NewCompositeInspector(static, dynamic)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(evidenceRoot)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := application.NewInstallInspectService(native, intake, verificationpypi.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			staging, err := promotion.NewLocalStaging(intakeRoot, evidenceRoot, stagingRoot)
			if err != nil {
				t.Fatal(err)
			}
			promotionRunner := &requirementsPromotionRunner{Executor: executor}
			promoter, err := promotion.NewPyPIPromotionWithRunner(stagingRoot, promotionRunner)
			if err != nil {
				t.Fatal(err)
			}
			base, err := application.NewInstallService(entry, evidencerecord.Generator{}, staging, promoter)
			if err != nil {
				t.Fatal(err)
			}
			blockInspector := &requirementsBlockInspector{CompositeInspector: inspector}
			bindings := pythonClientBindings{installer: base, promoter: promoter, resolver: native, nativeLookup: native, intake: intake, intakeRoot: intakeRoot, evidenceRoot: evidenceRoot, stagingRoot: stagingRoot, inspector: blockInspector, evidence: store}
			venv := filepath.Join(root, "venv")
			site := filepath.Join(venv, "lib/python3.14/site-packages")
			for _, directory := range []string{site, filepath.Join(venv, "bin")} {
				if err := os.MkdirAll(directory, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), []byte("version = "+sandbox.PinnedPythonRuntime().PythonVersion+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(site, "user-note.txt"), []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "requirements.txt")
			seen := map[string]bool{}
			hashes := map[string]string{}
			packagingVersion := "24.2"
			for _, step := range []struct {
				name, body          string
				upgrade, noop, deny bool
				minimum             int
			}{
				{"initial-empty", "# empty\n", false, true, false, 0},
				{"two-roots", "packaging==24.2\nidna==3.11\n", false, false, false, 2},
				{"retained-compatible", "requests==2.32.5\npackaging>=24\n", false, false, false, 6},
				{"update-roots", "packaging==25.0\nidna==3.11\n", true, false, false, 2},
				{"joint-conflict", "packaging==24.2\npackaging==25.0\n", false, false, true, 0},
				{"invalid-grammar", "packaging>=\n", false, false, true, 0},
				{"excluded-marker", "absent-project==0 ; python_version < '3.0'\n", false, true, false, 0},
				{"include-extra", "-r nested/extra.txt\n-c constraints.txt\npackaging>=24\n", false, false, false, 7},
				{"shared-transitive", "requests==2.32.5\nhttpcore==1.0.9\n", false, false, false, 7},
				{"hash-match", "--require-hashes\npackaging==25.0 --hash=sha256:PACKAGE_HASH\n", false, false, false, 1},
				{"hash-missing", "--require-hashes\npackaging==25.0\n", false, false, true, 0},
				{"hash-mismatch", "packaging==25.0 --hash=sha256:" + strings.Repeat("0", 64) + "\n", false, false, true, 0},
				{"transitive-block", "requests==2.32.5\n", false, false, true, 0},
				{"installation-failure", "packaging==25.0\n", false, false, true, 0},
				{"source-drift", "packaging==25.0\n", false, false, true, 0},
				{"final-empty", "# empty after installation\n", false, true, false, 0},
			} {
				promotionRunner.after = nil
				promotionRunner.calls = 0
				blockInspector.block = step.name == "transitive-block"
				blockInspector.hit = false
				promotionRunner.hit = false
				if step.name == "update-roots" {
					packagingVersion = "25.0"
				}
				if step.name == "include-extra" {
					if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
						t.Fatal(err)
					}
					for name, body := range map[string]string{"nested/extra.txt": "requests[socks]>=2.32,<3 ; python_version >= '3.14'\n", "constraints.txt": "packaging==25.0\nrequests==2.32.5\n"} {
						if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				if step.name == "hash-match" {
					hash := hashes["packaging@25.0"]
					if len(hash) != 64 {
						t.Fatal("missing actual approved archive digest")
					}
					step.body = strings.ReplaceAll(step.body, "PACKAGE_HASH", hash)
				}
				if step.name == "installation-failure" {
					promotionRunner.after = func() error { return errors.New("synthetic failure after isolated materialization") }
				}
				if step.name == "source-drift" {
					promotionRunner.after = func() error { return os.WriteFile(path, []byte("packaging==24.2\n"), 0600) }
				}
				if err := os.WriteFile(path, []byte(step.body), 0600); err != nil {
					t.Fatal(err)
				}
				before := requirementsTargetTree(t, venv)
				var evidenceBefore map[string]string
				if step.noop {
					evidenceBefore = requirementsTargetTree(t, evidenceRoot)
				}
				var output, diagnostic bytes.Buffer
				command, err := cli.New(&output, &diagnostic)
				if err != nil {
					t.Fatal(err)
				}
				prefix := []string{"pip", "install"}
				if client == "pip" {
					err = bindPipPackageClient(command, bindings)
				} else {
					prefix = []string{"uv", "pip", "install"}
					err = bindUVPipPackageClient(command, bindings)
				}
				if err != nil {
					t.Fatal(err)
				}
				args := append([]string{"--output", "json", "--lang", "en"}, prefix...)
				args = append(args, "-r", path, "--target", venv)
				if step.upgrade {
					args = append(args, "--upgrade")
				}
				if _, err := cli.PreparePresentation(command, args); err != nil {
					t.Fatal(err)
				}
				command.SetArgs(args)
				started := time.Now()
				cause := command.ExecuteContext(ctx)
				t.Logf("requirements_actual_report client=%s case=%s json=%s", client, step.name, bytes.TrimSpace(output.Bytes()))
				if step.deny {
					if cause == nil || !reflect.DeepEqual(before, requirementsTargetTree(t, venv)) {
						t.Fatal("denial changed the target", step.name, cause, output.String())
					}
					if step.name == "transitive-block" {
						var refusal *application.ProjectPolicyFailure
						if !blockInspector.hit || promotionRunner.calls != 0 || !errors.As(cause, &refusal) || refusal.Decision().Decision() != domain.DecisionBlock {
							t.Fatal("transitive block did not deny whole set before installation", cause)
						}
					}
					if (step.name == "installation-failure" || step.name == "source-drift") && !promotionRunner.hit {
						t.Fatal("failure did not reach isolated materialization", cause)
					}
					if step.name == "invalid-grammar" && !strings.Contains(cause.Error(), `requirements "requirements.txt":1:`) {
						t.Fatal("grammar refusal has no frozen source location", cause)
					}
					t.Logf("requirements_actual client=%s case=%s denied=true seconds=%.3f", client, step.name, time.Since(started).Seconds())
					continue
				}
				if cause != nil {
					var driver *exec.ExitError
					if errors.As(cause, &driver) {
						t.Logf("requirements_native_failure exit=%d stderr=%q", driver.ExitCode(), driver.Stderr[:min(len(driver.Stderr), 4096)])
					}
					t.Fatalf("requirements %s/%s: %v\n%s\n%s", client, step.name, cause, output.String(), diagnostic.String())
				}
				var report struct {
					Status    string `json:"operation_status"`
					Promotion string `json:"promotion_status"`
					Entries   []struct {
						Identity struct {
							Name    string `json:"name"`
							Version string `json:"version"`
						} `json:"identity"`
						Digest struct {
							Value string `json:"value"`
						} `json:"digest"`
						Run      string            `json:"run_id"`
						Checks   []json.RawMessage `json:"checks"`
						Evidence []json.RawMessage `json:"evidence_references"`
					} `json:"entries"`
				}
				if json.Unmarshal(output.Bytes(), &report) != nil {
					t.Fatal("invalid requirements JSON", output.String())
				}
				if step.noop {
					if !reflect.DeepEqual(evidenceBefore, requirementsTargetTree(t, evidenceRoot)) || report.Status != "SKIPPED" || report.Promotion != "NOT_PERFORMED" || len(report.Entries) != 0 || !reflect.DeepEqual(before, requirementsTargetTree(t, venv)) {
						t.Fatal("no-op inspected, published or changed target", output.String())
					}
				} else {
					if report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || len(report.Entries) < step.minimum {
						t.Fatal("requirements set did not complete", output.String())
					}
					members := map[string]bool{}
					for _, item := range report.Entries {
						if item.Run == "" || seen[item.Run] || len(item.Checks) == 0 || len(item.Evidence) == 0 {
							t.Fatal("missing fresh required checks/Evidence")
						}
						seen[item.Run] = true
						if members[item.Identity.Name] {
							t.Fatal("repeated whole-set dependency", item.Identity.Name)
						}
						members[item.Identity.Name] = true
						hashes[item.Identity.Name+"@"+item.Identity.Version] = item.Digest.Value
					}
					if step.name == "shared-transitive" && (!members["idna"] || !members["certifi"] || !members["requests"] || !members["httpcore"]) {
						t.Fatal("shared transitive closure incomplete")
					}
					version := packagingVersion
					if _, err := os.Stat(filepath.Join(site, "packaging-"+version+".dist-info", "METADATA")); err != nil {
						t.Fatal("compatible installed version was not preserved or updated", err)
					}
					if body, err := os.ReadFile(filepath.Join(site, "user-note.txt")); err != nil || string(body) != "preserve" {
						t.Fatal("user file changed", err)
					}
					if step.upgrade {
						if _, err := os.Lstat(filepath.Join(site, "packaging-24.2.dist-info", "METADATA")); !os.IsNotExist(err) {
							t.Fatal("obsolete selected metadata remains", err)
						}
						if _, err := os.Stat(filepath.Join(site, "requests", "__init__.py")); err != nil {
							t.Fatal("additive update removed unrelated installed package", err)
						}
					}
				}
				t.Logf("requirements_actual client=%s case=%s entries=%d noop=%t seconds=%.3f", client, step.name, len(report.Entries), step.noop, time.Since(started).Seconds())
			}
		})
		if t.Failed() {
			return
		}
	}
}

func requirementsTargetTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	if _, err := os.Lstat(root); os.IsNotExist(err) {
		return result
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprint(info.Mode())
		if info.Mode().IsRegular() {
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(body)
			value += "/" + hex.EncodeToString(sum[:])
		} else if !info.IsDir() {
			return fmt.Errorf("unexpected target entry %s", strings.TrimSpace(relative))
		}
		result[relative] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// This test-only seam returns a failure or mutates a frozen source after the
// real offline installer completes. It grants no runtime or Policy authority.
type requirementsPromotionRunner struct {
	*hosttool.Executor
	after func() error
	hit   bool
	calls int
}

func (r *requirementsPromotionRunner) Run(ctx context.Context, root string, args []string) error {
	if err := r.Executor.Run(ctx, root, args); err != nil {
		return err
	}
	r.calls++
	if r.after != nil {
		r.hit = true
		return r.after()
	}
	return nil
}

// The actual inspectors still run. A normalized synthetic transitive Finding
// then verifies whole-set denial, rather than claiming malware detection.
type requirementsBlockInspector struct {
	*inspectionpypi.CompositeInspector
	block, hit bool
}

func (i *requirementsBlockInspector) InspectProjectSelection(ctx context.Context, update domain.ProjectDependencyUpdate, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	reports, err := i.CompositeInspector.InspectProjectSelection(ctx, update, acquired)
	if err != nil || !i.block {
		return reports, err
	}
	for node, a := range acquired {
		if a.Identity().Name() != "idna" {
			continue
		}
		reports[node], err = requirementsBlockReport(a, reports[node])
		if err != nil {
			return nil, err
		}
		i.hit = true
		return reports, nil
	}
	return nil, errors.New("synthetic transitive subject is absent")
}

func requirementsBlockReport(a domain.AcquiredArtifact, original domain.InspectionReport) (domain.InspectionReport, error) {
	checkID, _ := domain.NewCheckID("synthetic-req-block")
	check, err := domain.NewCheckExecution(checkID, domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidenceID, _ := domain.NewEvidenceID("synthetic-req-block")
	item, err := domain.NewEvidence(evidenceID, checkID, a.Identity(), a.Digest(), "synthetic", "Synthetic transitive refusal for transaction qualification only.")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	finding, err := domain.NewFinding("SYNTHETIC_TRANSITIVE_BLOCK", []domain.EvidenceID{evidenceID})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	report, err := domain.NewInspectionReport(check, []domain.Finding{finding}, []domain.Evidence{item})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if len(original.Executions()) == 0 || len(original.Findings()) != 0 || len(original.Diagnostics()) != 0 {
		return domain.InspectionReport{}, errors.New("synthetic block needs complete clean fixture checks")
	}
	// Domain combines primitive check reports. Preserve every actual check and
	// its own Evidence before adding the synthetic optional Finding.
	byCheck := map[domain.CheckID][]domain.Evidence{}
	for _, item := range original.Evidence() {
		byCheck[item.CheckID()] = append(byCheck[item.CheckID()], item)
	}
	components := make([]domain.InspectionReport, 0, len(original.Executions())+1)
	for _, execution := range original.Executions() {
		component, err := domain.NewInspectionReport(execution, nil, byCheck[execution.ID()])
		if err != nil {
			return domain.InspectionReport{}, err
		}
		components = append(components, component)
	}
	components = append(components, report)
	return domain.NewCompositeInspectionReport(components)
}

func TestRequirementsSyntheticBlockPreservesRealChecks(t *testing.T) {
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "idna", "3.11", "wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	a, err := domain.NewAcquiredArtifact(identity, digest, "intake:synthetic", 1)
	if err != nil {
		t.Fatal(err)
	}
	var components []domain.InspectionReport
	for _, name := range []string{"static", "dynamic"} {
		id, _ := domain.NewCheckID(name)
		check, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		if err != nil {
			t.Fatal(err)
		}
		evidenceID, _ := domain.NewEvidenceID(name + "-result")
		item, err := domain.NewEvidence(evidenceID, id, identity, digest, "synthetic", "Normalized fixture facts.")
		if err != nil {
			t.Fatal(err)
		}
		report, err := domain.NewInspectionReport(check, nil, []domain.Evidence{item})
		if err != nil {
			t.Fatal(err)
		}
		components = append(components, report)
	}
	original, err := domain.NewCompositeInspectionReport(components)
	if err != nil {
		t.Fatal(err)
	}
	decorated, err := requirementsBlockReport(a, original)
	if err != nil {
		t.Fatal("synthetic block lost normalized real-check evidence", err)
	}
	if len(decorated.Executions()) != 3 || len(decorated.Evidence()) != 3 || len(decorated.Findings()) != 1 || decorated.Findings()[0].Code() != "SYNTHETIC_TRANSITIVE_BLOCK" {
		t.Fatal("synthetic block omitted real check facts")
	}
	for index, check := range original.Executions() {
		if !reflect.DeepEqual(check, decorated.Executions()[index]) {
			t.Fatal("real required check changed")
		}
	}
	if len(original.Executions()) != 2 || len(original.Findings()) != 0 {
		t.Fatal("original report was mutated")
	}
}

// Public intake and the production Application/store qualify the normalized
// evidence separately from the backend-only controlled-wheel scenarios.
func TestPythonEnvironmentEvidenceIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_ENVIRONMENT_EVIDENCE_INTEGRATION") != "1" {
		t.Skip("requires protected Python environment evidence integration")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client and protected infrastructure")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	executor, observer := pythonRequirementsIntegrationRuntime(t, ctx)
	policyService, err := newSystemResolverPolicyAdapter()
	if err != nil {
		t.Fatal(err)
	}
	native, err := sandbox.NewLinuxPyPIResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := native.Close(); err != nil {
			t.Error(err)
		}
	}()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	intakeRoot := filepath.Join(root, "intake")
	intake, err := artifactpypi.NewPublicIntake(intakeRoot)
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactpypi.NewGraphArtifact(native, intake)
	if err != nil {
		t.Fatal(err)
	}
	runtime := sandbox.PinnedPythonRuntime()
	static, err := inspectionpypi.NewStaticInspector(intakeRoot, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		t.Fatal(err)
	}
	backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot, executor, observer.Observer(), newObservationResourceAdapter())
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := inspectionpypi.NewDynamicInspector(backend)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectionpypi.NewCompositeInspector(static, dynamic)
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInstallInspectService(native, intake, verificationpypi.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactpypi.ParseReference("packaging@24.2")
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(filepath.Join(root, "target"))
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInstallRequest(reference, install)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision().Decision() != domain.DecisionAllow || len(result.Set().Inspections()) != 1 {
		t.Fatalf("clean public wheel did not complete/allow: decision=%+v", result.Decision())
	}
	entry := result.Set().Inspections()[0]
	run, artifact := entry.RunID(), entry.Artifact()
	identity, digest := artifact.Identity(), artifact.Digest()
	var check domain.CheckExecution
	for _, execution := range entry.Checks() {
		if execution.ID().String() == "pypi-dynamic-import" {
			check = execution
		}
	}
	if !check.Required() || check.Status() != domain.ExecutionCompleted {
		t.Fatal("current required dynamic check missing")
	}
	profiles := map[domain.SandboxEnvironmentProfile]bool{}
	observationSeen := false
	for _, ref := range entry.Evidence() {
		fact, _, err := store.ReadReference(ctx, run, ref, identity, digest)
		if err != nil {
			t.Fatal(err)
		}
		if fact.CheckID() != check.ID() {
			continue
		}
		if fact.ProfileDigest() != check.ProfileDigest() || len(fact.Summary()) > 1024 {
			t.Fatal("persisted current check/profile binding lost")
		}
		if fact.Kind() == "pypi-dynamic-import" {
			if !strings.Contains(fact.Summary(), "filesystem-workspace-create") || !strings.Contains(fact.Summary(), "filesystem-workspace-write") {
				t.Fatal("actual VFS facts missing from persisted observation")
			}
			observationSeen = true
		}
		if fact.Kind() != "workspace-state" {
			continue
		}
		var state struct {
			Schema     string                           `json:"schema"`
			Profile    domain.SandboxEnvironmentProfile `json:"profile"`
			Fixture    string                           `json:"fixture_digest"`
			Before     string                           `json:"baseline_digest"`
			After      string                           `json:"final_digest"`
			Scope      string                           `json:"scope"`
			Transient  string                           `json:"transient_changes"`
			Operations string                           `json:"operation_scope"`
		}
		if err := json.Unmarshal([]byte(fact.Summary()), &state); err != nil {
			t.Fatal(err)
		}
		if state.Schema != domain.SandboxEnvironmentStateSchema || state.Scope != "MUTABLE_WORKSPACE" || state.Transient != "SESSION_VFS_OPERATIONS" || state.Operations != "CREATE_WRITE_PWRITE_RENAME_UNLINK_MODE_OWNER_SIZE" || len(state.Fixture) != 64 || len(state.Before) != 64 || len(state.After) != 64 || profiles[state.Profile] {
			t.Fatalf("persisted environment evidence incomplete/duplicate: %+v", state)
		}
		profiles[state.Profile] = true
	}
	if !observationSeen || len(profiles) != 2 || !profiles[domain.SandboxEnvironmentMinimal] || !profiles[domain.SandboxEnvironmentDevelopment] {
		t.Fatalf("actual persisted coverage missing: observations=%t profiles=%v", observationSeen, profiles)
	}
	// A separately approved inspection input exercises the actual v2 metadata
	// encoding and store, without changing the original selected dependency set.
	inputReference, err := artifactpypi.ParseReference("typing-extensions@4.15.0")
	if err != nil {
		t.Fatal(err)
	}
	resolvedInput, err := source.Resolve(ctx, inputReference)
	if err != nil {
		t.Fatal(err)
	}
	inputRun, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	input, err := source.Acquire(ctx, inputRun, resolvedInput)
	if err != nil {
		t.Fatal(err)
	}
	wheel, _, err := static.InspectWheel(ctx, artifact)
	if err != nil {
		t.Fatal(err)
	}
	inputWheel, _, err := static.InspectWheel(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifactpypi.ValidateInspectionPrerequisite(inputWheel, []artifactpypi.WheelInspection{wheel}, artifactpypi.ResourcePolicyFromContext(ctx)); err != nil {
		t.Fatalf("declared leaf fixture does not meet prerequisite contract: %v", err)
	}
	supplemented, err := dynamic.InspectWheelWithPrerequisites(ctx, artifact, wheel, []domain.AcquiredArtifact{artifact}, []domain.AcquiredArtifact{input})
	if err != nil || supplemented.Execution().Status() != domain.ExecutionCompleted {
		t.Fatalf("actual supplemented inspection failed: %+v %v", supplemented.Executions(), err)
	}
	supplementedRun, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	refs, err := store.Record(ctx, supplementedRun, supplemented.Evidence(), supplemented.Executions()...)
	if err != nil {
		t.Fatal(err)
	}
	var metadata, observations domain.Evidence
	stateCount := 0
	for _, ref := range refs {
		fact, _, err := store.ReadReference(ctx, supplementedRun, ref, identity, digest)
		if err != nil {
			t.Fatal(err)
		}
		if fact.ProfileDigest() != supplemented.Execution().ProfileDigest() || fact.CheckID() != supplemented.Execution().ID() {
			t.Fatal("supplemented current check/profile lost on readback")
		}
		switch fact.Kind() {
		case "pypi-dynamic-import-supplemented":
			metadata = fact
		case "pypi-supplemented-observations":
			observations = fact
		case "workspace-state":
			stateCount++
		}
	}
	var binding struct {
		Schema            string `json:"schema"`
		Original          string `json:"original_environment"`
		Input             string `json:"input_sha256"`
		ObservationID     string `json:"observations_evidence"`
		ObservationDigest string `json:"observation_summary_sha256"`
	}
	if err := json.Unmarshal([]byte(metadata.Summary()), &binding); err != nil {
		t.Fatal(err)
	}
	observedDigest := sha256.Sum256([]byte(observations.Summary()))
	if binding.Schema != "m5-inspection-environment/v2" || binding.Original != "NOT_ATTESTED" || binding.Input != input.Digest().String() || binding.ObservationID != observations.ID().String() || binding.ObservationDigest != hex.EncodeToString(observedDigest[:]) || observations.Summary() == "" || stateCount != 2 {
		t.Fatalf("actual supplemented metadata/observation/state readback lost: %+v states=%d", binding, stateCount)
	}
	if !strings.Contains(observations.Summary(), "filesystem-workspace-create") || !strings.Contains(observations.Summary(), "filesystem-workspace-write") {
		t.Fatal("supplemented actual VFS observations omitted")
	}
	if len(result.Set().Inspections()) != 1 || result.Set().Inspections()[0].Artifact().Digest() != artifact.Digest() {
		t.Fatal("inspection input changed the original dependency set")
	}
	t.Log("actual pinned public Python wheel -> two fresh environments/VFS -> current required check -> persisted same-subject/profile Evidence readback -> Policy ALLOW; actual separate prerequisite metadata/observation/state readback; original set preserved")
}

func pythonRequirementsIntegrationRuntime(t *testing.T, ctx context.Context) (*hosttool.Executor, *sandbox.ObserverSupervisor) {
	t.Helper()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	})
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
	t.Cleanup(func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	})
	return executor, observer
}
