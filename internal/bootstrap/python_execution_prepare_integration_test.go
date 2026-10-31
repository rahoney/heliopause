//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxPythonExecutionPreparationIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_EXECUTION_PREPARATION_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user current Python preparation qualification")
	}
	// Reuse the existing real native/current-inspection lifecycle. Its returned
	// interpreter is authenticated image DATA and is not executed or published.
	ctx, bindings, _, _, _ := toolApprovedIntegrationFixture(t, false)
	native, ok := bindings.nativeLookup.(*sandbox.UVResolver)
	if !ok {
		t.Fatal("actual native selector missing")
	}
	intake, ok := bindings.intake.(*artifactpypi.Intake)
	if !ok {
		t.Fatal("actual public intake missing")
	}
	inspector, ok := bindings.inspector.(*inspectionpypi.CompositeInspector)
	if !ok {
		t.Fatal("actual current PyPI inspector missing")
	}
	store, ok := bindings.evidence.(*local.Store)
	if !ok {
		t.Fatal("actual current Evidence missing")
	}
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
	seen := map[string]bool{}
	for caseID, command := range []string{"", "cowsay", "missing-cowsay-command"} {
		arguments := []string{"haa-python-preparation-private-argv", "", "a b", "--help", "$(literal)"}
		request, err := application.NewExecutionPreparationRequest(reference, command, arguments)
		if err != nil {
			t.Fatal(err)
		}
		prepared, prepareErr := service.Prepare(ctx, request)
		current := prepared.Inspection()
		if !current.Valid() || current.Set().Decision().Decision() != domain.DecisionAllow || !strings.HasPrefix(current.Resolution().RuntimeIdentity(), "uv-tool-install:") {
			t.Fatal("fresh whole approval or honest native identity missing", prepareErr)
		}
		var primary domain.AcquiredArtifact
		for _, entry := range current.Set().Inspected().Inspections() {
			if seen[entry.RunID().String()] || len(entry.Evidence()) < 3 {
				t.Fatal("current Evidence was reused or incomplete")
			}
			seen[entry.RunID().String()] = true
			required := 0
			for _, check := range entry.Checks() {
				if check.Required() {
					if check.Status() != domain.ExecutionCompleted {
						t.Fatal("current required check incomplete")
					}
					required++
				}
			}
			if required < 3 || entry.Artifact().Identity().Source() != artifactpypi.PublicPyPIProfile().Source() {
				t.Fatal("current whole source/check binding missing")
			}
			for _, evidence := range entry.Evidence() {
				if _, _, err := store.ReadReference(ctx, entry.RunID(), evidence, entry.Artifact().Identity(), entry.Artifact().Digest()); err != nil {
					t.Fatal("current Evidence readback failed", err)
				}
			}
			if entry.Node() == current.Resolution().Graph().Primary() {
				primary = entry.Artifact()
			}
		}
		if primary.Identity().Name() != "cowsay" || primary.Identity().Version() != "6.1" || primary.Digest().String() == "" {
			t.Fatal("actual public primary missing")
		}
		if caseID == 2 {
			if prepareErr == nil || prepared.Valid() || prepared.FailureCode() != "ENTRY_BINDING_FAILED" || prepared.ExecutionContext().Valid() || prepared.Bundle().Valid() {
				t.Fatal("missing console passed current post-Policy preparation", prepareErr)
			}
			t.Log("python_execution_preparation case=3 policy=ALLOW missing_command_refused=true stored=false user_program_run=false")
			continue
		}
		if prepareErr != nil || !prepared.Valid() {
			t.Fatal("actual current preparation failed", prepareErr)
		}
		inventory, err := static.ReadConsoleExecutionInventory(ctx, primary)
		if err != nil {
			t.Fatal(err)
		}
		selected, err := artifactpypi.SelectConsoleExecutionEntry(inventory, command)
		if err != nil {
			t.Fatal(err)
		}
		execution, member := prepared.ExecutionContext(), selected.Metadata()
		if execution.EntryArtifact() != primary || execution.Command() != selected.Command() || execution.EntryPath() != member.Path || execution.EntryDigest().String() != member.SHA256 || execution.EntrySize() != uint64(member.Size) || execution.RuntimeIdentity() != runtimeidentity.PythonImageReference || !slices.Equal(execution.Arguments(), arguments) {
			t.Fatal("exact console declaration, runtime or private argv changed")
		}
		bundle := prepared.Bundle()
		if !bytes.Contains(bundle.ManifestDocument(), []byte("helox.verified-execution-manifest/v1")) || bytes.Contains(bundle.ManifestDocument(), []byte(arguments[0])) || bytes.Contains(bundle.SBOMDocument(), []byte(arguments[0])) {
			t.Fatal("execution records missing or private argv persisted")
		}
		for name, expected := range map[string][]byte{"manifest.json": bundle.ManifestDocument(), "sbom.cdx.json": bundle.SBOMDocument()} {
			body, err := os.ReadFile(filepath.Join(bindings.stagingRoot, bundle.ManifestID().String(), name))
			if err != nil || !bytes.Equal(body, expected) {
				t.Fatal("current staged records differ", err)
			}
		}
		t.Logf("python_execution_preparation case=%d policy=ALLOW preparation=COMPLETE source_sha256=%s exact_console_declaration=true private_argv=true user_program_run=false", caseID+1, primary.Digest().String())
	}
	entries, err := os.ReadDir(bindings.stagingRoot)
	if err != nil || len(entries) != 2 || len(seen) != 3 {
		t.Fatal("fresh current3/stored2 scope changed", err)
	}
	if err := context.Cause(ctx); err != nil {
		t.Fatal("current preparation context failed", err)
	}
	t.Log("python_execution_preparation_evidence fresh_runs=3 stored_manifests=2 native_selector=uv-tool-install user_program_run=false")
}
