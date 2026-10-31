//go:build linux && realcorpus

package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
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

func TestLinuxPythonExecutionInputsIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_EXECUTION_INPUT_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user readonly Python input qualification")
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
	var contexts []domain.ExecutionContext
	var entries []artifactpypi.ConsoleExecutionEntry
	for i, command := range []string{"", "cowsay"} {
		request, err := application.NewExecutionPreparationRequest(reference, command, []string{"private-python-input-argv", "--help"})
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := service.Prepare(ctx, request)
		if err != nil || !prepared.Valid() || prepared.Inspection().Set().Decision().Decision() != domain.DecisionAllow {
			t.Fatal("actual current Python preparation unavailable", err)
		}
		execution := prepared.ExecutionContext()
		inventory, err := static.ReadConsoleExecutionInventory(ctx, execution.EntryArtifact())
		if err != nil {
			t.Fatal(err)
		}
		entry, err := artifactpypi.SelectConsoleExecutionEntry(inventory, command)
		if err != nil {
			t.Fatal(err)
		}
		if err := backend.ValidatePythonExecutionInputs(ctx, execution, entry); err != nil {
			t.Fatal("actual current readonly Python input failed", err)
		}
		contexts = append(contexts, execution)
		entries = append(entries, entry)
		t.Logf("python_execution_inputs case=%d current_policy=ALLOW readonly_view_verified=true complete_trace_cpu_cleanup=true source_sha256=%s user_program_run=false", i+1, execution.EntryArtifact().Digest().String())
	}
	if contexts[0].EntryArtifact().ContentHandle() == contexts[1].EntryArtifact().ContentHandle() {
		t.Fatal("fresh actual acquisition was reused")
	}
	if err := backend.ValidatePythonExecutionInputs(ctx, contexts[1], entries[0]); err == nil {
		t.Fatal("different current acquisition substituted")
	}
	t.Log("python_execution_inputs case=3 acquisition_substitution_refused=true user_program_run=false")
	artifact := contexts[0].EntryArtifact()
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 {
		t.Fatal("actual intake handle malformed")
	}
	path := filepath.Join(bindings.intakeRoot, parts[1], "wheel.whl")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(original)
	if hex.EncodeToString(sum[:]) != artifact.Digest().String() {
		t.Fatal("original actual source changed")
	}
	defer func() {
		if err := os.WriteFile(path, original, 0600); err != nil {
			t.Error("restore owned intake", err)
		}
	}()
	changed := append([]byte(nil), original...)
	changed[len(changed)/2] ^= 1
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := backend.ValidatePythonExecutionInputs(ctx, contexts[0], entries[0]); err == nil {
		t.Fatal("changed intake reused current approval")
	}
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum = sha256.Sum256(restored)
	if hex.EncodeToString(sum[:]) != artifact.Digest().String() {
		t.Fatal("owned intake restoration failed")
	}
	t.Log("python_execution_inputs case=4 intake_tamper_refused=true original_restored=true user_program_run=false")
}
