//go:build linux && realcorpus

package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxPythonExecutionIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_EXECUTION_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user selected Python execution qualification")
	}
	ctx, bindings, _, _, _, backend := toolApprovedIntegrationFixtureWithPythonBackend(t, false)
	native := bindings.nativeLookup.(*sandbox.UVResolver)
	intake := bindings.intake.(*artifactpypi.Intake)
	inspector := bindings.inspector.(*inspectionpypi.CompositeInspector)
	store := bindings.evidence.(*local.Store)
	runtime := sandbox.PinnedPythonRuntime()
	static, err := inspectionpypi.NewStaticInspector(bindings.intakeRoot, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		t.Fatal(err)
	}
	staging, err := promotion.NewLocalStaging(bindings.intakeRoot, bindings.evidenceRoot, bindings.stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	service, err := newPythonExecutionPreparationService(native, intake, inspector, static, store, staging)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactpypi.ParseReference("cowsay@6.1")
	if err != nil {
		t.Fatal(err)
	}
	var previousHandle string
	for i, args := range [][]string{{"-t", "private-console-body"}, {"--invalid-haa-option"}} {
		request, err := application.NewExecutionPreparationRequest(reference, "cowsay", args)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := service.Prepare(ctx, request)
		if err != nil || !prepared.Valid() || prepared.Inspection().Set().Decision().Decision() != domain.DecisionAllow {
			t.Fatal("actual current Python preparation unavailable", err)
		}
		execution := prepared.ExecutionContext()
		if execution.EntryArtifact().ContentHandle() == previousHandle {
			t.Fatal("fresh current acquisition reused")
		}
		previousHandle = execution.EntryArtifact().ContentHandle()
		inventory, err := static.ReadConsoleExecutionInventory(ctx, execution.EntryArtifact())
		if err != nil {
			t.Fatal(err)
		}
		entry, err := artifactpypi.SelectConsoleExecutionEntry(inventory, "cowsay")
		if err != nil {
			t.Fatal(err)
		}
		outcome, runErr := backend.ExecutePythonExecution(ctx, execution, entry)
		code, exited := outcome.ExitCode()
		var failed *sandbox.PythonProgramError
		if !outcome.Complete() || !exited {
			t.Fatal("actual selected Python execution incomplete", runErr)
		}
		if i == 0 && (runErr != nil || code != 0 || !strings.Contains(string(outcome.Stdout()), "private-console-body")) {
			t.Fatal("actual normal console result wrong", runErr, code)
		}
		if i == 1 && (!errors.As(runErr, &failed) || code != 2 || len(outcome.Stderr()) == 0) {
			t.Fatal("actual guest failure lost", runErr, code)
		}
		if _, err := json.Marshal(outcome); err == nil || strings.Contains(fmt.Sprintf("%#v", outcome), "private-console-body") {
			t.Fatal("private output exposed")
		}
		t.Logf("python_execution case=%d current_policy=ALLOW complete=true exit_code=%d full_source_trace_cpu_cleanup=true source_sha256=%s", i+1, code, execution.EntryArtifact().Digest().String())
	}
}
