//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxNodeExecutionWorkflowIntegration(t *testing.T) {
	runNodeExecutionWorkflowIntegration(t, "npm")
}

func TestLinuxPNPMRunExecutionWorkflowIntegration(t *testing.T) {
	runNodeExecutionWorkflowIntegration(t, "pnpm")
}

func TestLinuxBunRunExecutionWorkflowIntegration(t *testing.T) {
	runNodeExecutionWorkflowIntegration(t, "bun")
}

func TestLinuxBunProgramExecutionWorkflowIntegration(t *testing.T) {
	runNodeExecutionWorkflowIntegration(t, "bun-program")
}

func TestLinuxBunDeclaredEntryWorkflowIntegration(t *testing.T) {
	runNodeExecutionFixtureIntegration(t, "bun", bunDeclaredEntryFixture)
}

type nodeExecutionPublicFixture struct {
	selector, command, name, version, member, integrity, normal, failure string
	bunDefault, failureStderr                                            bool
}

var bunDeclaredEntryFixture = nodeExecutionPublicFixture{
	selector: "bun-install@0.5.0", command: "bun-install", name: "bun-install", version: "0.5.0", member: "src/index.ts",
	integrity: "sha512-mpe+XGXMsSw9rGxtH2i4kzpHiO+3hAYckVYoXWqgnKj6H+wJVLvvv80Hi/OQUENagkSbVp+KvnqGo1m98T2veg==",
	normal:    "bun-install", failure: "--haa-invalid-option", bunDefault: true, failureStderr: true,
}

type bunProgramWorkflowRunner struct{ bunRunExecutionWorkflow }

func (r bunProgramWorkflowRunner) Run(ctx context.Context, request application.ExecutionPreparationRequest) (application.ExecutionResult, error) {
	return r.RunBun(ctx, request)
}

func runNodeExecutionWorkflowIntegration(t *testing.T, client string) {
	runNodeExecutionFixtureIntegration(t, client, nodeExecutionPublicFixture{
		selector: "typescript@5.9.3", command: "tsc", name: "typescript", version: "5.9.3", member: "bin/tsc",
		integrity: "sha512-jl1vZzPDinLr9eUt3J/t7V6FgNEw9QjvBPdysz9KfQDD41fQrC2Y4vKQdiaUpFT4bXlb1RHhLpp8wtm6M5TgSw==",
		normal:    "Version 5.9.3", failure: "Unknown compiler option",
	})
}

func runNodeExecutionFixtureIntegration(t *testing.T, client string, fixture nodeExecutionPublicFixture) {
	t.Helper()
	declared := os.Getenv("HELOX_NODE_EXECUTION_WORKFLOW_INTEGRATION") == "1"
	if client == "pnpm" {
		declared = os.Getenv("HELOX_PNPM_RUN_EXECUTION_WORKFLOW_INTEGRATION") == "1"
	}
	if client == "bun" || client == "bun-program" {
		declared = os.Getenv("HELOX_BUN_RUN_EXECUTION_WORKFLOW_INTEGRATION") == "1"
	}
	if client == "bun-program" {
		declared = os.Getenv("HELOX_BUN_PROGRAM_EXECUTION_WORKFLOW_INTEGRATION") == "1"
	}
	if !declared || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user production Node preparation qualification")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	t.Cleanup(cancel)
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
	resolverPolicy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		t.Fatal(err)
	}
	intake, evidenceRoot, stagingRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
	source, err := artifactnpm.NewPublicResolver(intake)
	if err != nil {
		t.Fatal(err)
	}
	static, err := inspectionnpm.NewInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := sandbox.NewLinuxBackendWithExecutor(intake, executor, observer.Observer())
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := promotion.NewLocalStaging(intake, evidenceRoot, stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	var service cli.ExecutionRunner
	if client == "pnpm" {
		var nativeExecutor sandbox.TrustedExecutor = executor
		if directory := os.Getenv("HELOX_PNPM_CURRENT_LAYOUT_CAPTURE_DIR"); directory != "" {
			nativeExecutor = &pnpmCurrentLayoutRecorder{Executor: executor, directory: directory}
		}
		native, err := sandbox.NewLinuxPNPMResolverWithExecutorAndPolicy(nativeExecutor, observer.Observer(), resolverPolicy)
		if err != nil {
			t.Fatal(err)
		}
		prepare, err := sandbox.NewPNPMInspectionPreparer(native, intake)
		if err != nil {
			t.Fatal(err)
		}
		service = pnpmRunExecutionWorkflow{native, source, static, prepare, backend, store, staging}
	} else if client == "bun" || client == "bun-program" {
		native, err := sandbox.NewLinuxBunResolverWithExecutorAndPolicy(executor, observer.Observer(), resolverPolicy)
		if err != nil {
			t.Fatal(err)
		}
		prepare, err := sandbox.NewBunInspectionPreparer(native, intake)
		if err != nil {
			t.Fatal(err)
		}
		service = bunRunExecutionWorkflow{native, source, static, prepare, backend, store, staging}
		if client == "bun-program" {
			service = bunProgramWorkflowRunner{service.(bunRunExecutionWorkflow)}
		}
	} else {
		native, err := sandbox.NewLinuxNPMResolverWithExecutorAndPolicy(executor, observer.Observer(), resolverPolicy)
		if err != nil {
			t.Fatal(err)
		}
		service, err = newNPMExecutionService(native, source, static, backend, store, staging)
		if err != nil {
			t.Fatal(err)
		}
	}
	reference, err := artifactnpm.ParseReference(fixture.selector)
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewExecutionPreparationRequest(reference, fixture.command, []string{"--help"})
	if err != nil {
		t.Fatal(err)
	}
	var previous application.PreparedExecution
	for attempt := 1; attempt <= 2; attempt++ {
		if attempt == 2 {
			request, err = application.NewExecutionPreparationRequest(reference, fixture.command, []string{"--haa-invalid-option"})
			if err != nil {
				t.Fatal(err)
			}
		}
		executed, runErr := service.Run(ctx, request)
		prepared := executed.Preparation()
		if !prepared.Valid() {
			t.Fatalf("production workflow preparation attempt=%d boundary=%s: %v", attempt, executed.FailureCode(), runErr)
		}
		inspected := prepared.Inspection()
		set := inspected.Set()
		entries := set.Inspected().Inspections()
		if len(entries) != 1 || len(inspected.Resolution().Graph().Nodes()) != 1 || set.Decision().Decision() != domain.DecisionAllow {
			t.Fatal("current complete single-subject graph was not approved")
		}
		entry := entries[0]
		subject := entry.Artifact()
		identity := subject.Identity()
		declared, ok := subject.DeclaredIntegrity()
		if identity.Source().String() != "npm" || identity.Name() != fixture.name || identity.Version() != fixture.version || !ok || declared != fixture.integrity {
			t.Fatal("actual source differs from fixed official fixture")
		}
		checks := map[string]bool{}
		for _, check := range entry.Checks() {
			if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
				t.Fatal("required preparation check incomplete", check.ID().String())
			}
			checks[check.ID().String()] = true
		}
		dynamicCheck := "npm-dynamic-lifecycle"
		if client == "pnpm" {
			dynamicCheck = "pnpm-dynamic-lifecycle"
		}
		if client == "bun" || client == "bun-program" {
			dynamicCheck = "bun-dynamic-lifecycle"
		}
		for _, name := range []string{"npm-declared-integrity", "npm-static-archive", dynamicCheck} {
			if !checks[name] {
				t.Fatal("actual required check absent", name)
			}
		}
		if entry.PolicyDecision().PolicyID() != "m3-npm-dynamic-inspect" || entry.PolicyDecision().Decision() != domain.DecisionAllow || len(entry.Evidence()) < 3 {
			t.Fatal("current production entry Policy/Evidence missing")
		}
		execution := prepared.ExecutionContext()
		runtime := runtimeidentity.NodeImageReference
		if client == "bun-program" || fixture.bunDefault {
			runtime = sandbox.BunExecutionRuntimeIdentity()
		}
		if execution.OperationID() != inspected.OperationID() || execution.EntryArtifact() != subject || execution.Command() != fixture.command || execution.EntryPath() != fixture.member || execution.EntrySize() == 0 || execution.RuntimeIdentity() != runtime || execution.SourceFingerprint() != inspected.Resolution().LockfileDigest() || !slices.Equal(execution.Arguments(), request.Arguments()) {
			t.Fatal("Node proof/runtime/current request binding lost")
		}
		for name, expected := range map[string][]byte{"manifest.json": prepared.Bundle().ManifestDocument(), "sbom.cdx.json": prepared.Bundle().SBOMDocument()} {
			body, err := os.ReadFile(filepath.Join(stagingRoot, prepared.Bundle().ManifestID().String(), name))
			if err != nil || !bytes.Equal(body, expected) {
				t.Fatal("actual staged records differ", name, err)
			}
		}
		if previous.Valid() && (previous.Inspection().OperationID() == inspected.OperationID() || previous.Inspection().Set().Inspected().Inspections()[0].RunID() == entry.RunID() || previous.Bundle().ManifestID() == prepared.Bundle().ManifestID()) {
			t.Fatal("repeated production request reused approval/records")
		}
		code, exited := executed.ExitCode()
		if !executed.Complete() || !exited || (attempt == 1 || !fixture.failureStderr) && len(executed.Stderr()) != 0 {
			// This fixed public fixture preserves the bounded original driver
			// diagnostic; production runtime output remains private.
			var driver *exec.ExitError
			if errors.As(runErr, &driver) {
				t.Logf("node_execution_driver_failure exit_code=%d stderr=%q", driver.ExitCode(), driver.Stderr[:min(len(driver.Stderr), 4096)])
			}
			t.Fatal("actual Artifact Node execution is incomplete", runErr)
		}
		if attempt == 1 {
			if runErr != nil || code != 0 || !strings.Contains(string(executed.Stdout()), fixture.normal) {
				t.Fatal("approved Node help execution failed", runErr)
			}
		} else {
			var program *sandbox.NodeProgramError
			typed := errors.As(runErr, &program)
			if client == "bun-program" || fixture.bunDefault {
				var bun *sandbox.BunProgramError
				typed = errors.As(runErr, &bun)
			}
			output := executed.Stdout()
			if fixture.failureStderr {
				output = executed.Stderr()
			}
			if code != 1 || !typed || !strings.Contains(string(output), fixture.failure) {
				t.Fatal("guest nonzero differs from runtime failure", runErr)
			}
		}
		marker := "production_node_application"
		if client == "pnpm" {
			marker = "production_pnpm_run_application"
			if inspected.Resolution().RuntimeIdentity() != "pnpm-dlx-isolated:"+sandbox.PNPMResolverRuntimeIdentity() {
				t.Fatal("pnpm native resolver identity lost")
			}
		}
		if client == "bun" || client == "bun-program" {
			marker = "production_bun_run_application"
			if inspected.Resolution().RuntimeIdentity() != "bunx-isolated:"+sandbox.BunResolverRuntimeIdentity() {
				t.Fatal("bun native resolver identity lost")
			}
		}
		if client == "bun-program" {
			marker = "production_bun_program_application"
		}
		if fixture.bunDefault {
			marker = "production_bun_declared_entry_application"
		}
		t.Logf("%s attempt=%d artifact_role=true target_loaded=true exit_code=%d stdout_bytes=%d stderr_bytes=%d user_program_run=true", marker, attempt, code, len(executed.Stdout()), len(executed.Stderr()))
		previous = prepared
	}
}

// This diagnostic tee preserves the actual executor's optional capabilities.
// Only the first fixed layout DATA export is copied; it grants no authority.
type pnpmCurrentLayoutRecorder struct {
	*hosttool.Executor
	directory string
	captured  bool
}

func (r *pnpmCurrentLayoutRecorder) RunOutput(ctx context.Context, output io.Writer, name string, args ...string) error {
	if r.captured || name != "docker" || len(args) != 3 || args[0] != "cp" || !strings.HasSuffix(args[1], ":/tmp/haa-pnpm-input/project/node_modules") || args[2] != "-" {
		return r.Executor.RunOutput(ctx, output, name, args...)
	}
	file, err := os.OpenFile(filepath.Join(r.directory, "pnpm-current-layout.tar"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	r.captured = true
	return errors.Join(r.Executor.RunOutput(ctx, io.MultiWriter(output, file), name, args...), file.Close())
}
