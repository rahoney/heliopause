//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func TestLinuxPythonUVRunExecutionWorkflowIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_UV_RUN_EXECUTION_WORKFLOW_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user current Python application workflow qualification")
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
	service, err := newUVRunExecutionService(native, "cowsay", intake, inspector, static, backend, store, staging)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactpypi.ParseReference("cowsay@6.1")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var previous application.PreparedExecution
	for i, caseData := range []struct {
		command string
		args    []string
		code    uint8
		missing bool
	}{
		{"cowsay", []string{"-t", "private-native-run-workflow-body"}, 0, false},
		{"cowsay", []string{"--haa-invalid-option"}, 2, false},
		{"haa-missing-console", nil, 0, true},
	} {
		before, err := os.ReadDir(bindings.stagingRoot)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		request, err := application.NewExecutionPreparationRequest(reference, caseData.command, caseData.args)
		if err != nil {
			t.Fatal(err)
		}
		executed, runErr := service.Run(ctx, request)
		prepared := executed.Preparation()
		inspection := prepared.Inspection()
		if !inspection.Valid() || inspection.Set().Decision().Decision() != domain.DecisionAllow || inspection.Set().Decision().PolicyID() != "m5-pypi-pip" || inspection.Resolution().RuntimeIdentity() != "uv-tool-run-isolated:"+sandbox.UVResolverRuntimeIdentity() {
			t.Fatal("current public preparation/whole Policy or honest native identity missing", runErr)
		}
		entries := inspection.Set().Inspected().Inspections()
		if len(entries) != 1 || len(inspection.Resolution().Graph().Nodes()) != 1 {
			t.Fatal("current public single-subject graph missing")
		}
		entry := entries[0]
		subject := entry.Artifact()
		if subject.Identity().Name() != "cowsay" || subject.Identity().Version() != "6.1" || subject.Digest().String() != "274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a" || len(entry.Evidence()) < 3 || seen[inspection.OperationID().String()] || seen[entry.RunID().String()] || seen[subject.ContentHandle()] {
			t.Fatal("current source/Evidence reused or changed")
		}
		seen[inspection.OperationID().String()], seen[entry.RunID().String()], seen[subject.ContentHandle()] = true, true, true
		required := 0
		for _, check := range entry.Checks() {
			if check.Required() {
				required++
				if check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted {
					t.Fatal("actual required check incomplete")
				}
			}
		}
		if required < 3 {
			t.Fatal("current required checks missing")
		}
		for _, evidence := range entry.Evidence() {
			if _, _, err := store.ReadReference(ctx, entry.RunID(), evidence, subject.Identity(), subject.Digest()); err != nil {
				t.Fatal("current Evidence unreadable", err)
			}
		}
		if caseData.missing {
			after, err := os.ReadDir(bindings.stagingRoot)
			if err != nil {
				t.Fatal(err)
			}
			names := func(entries []os.DirEntry) []string {
				result := make([]string, 0, len(entries))
				for _, entry := range entries {
					result = append(result, entry.Name())
				}
				return result
			}
			if runErr == nil || prepared.Valid() || executed.Complete() || executed.RuntimeAttempted() || executed.FailureCode() != "PREPARATION_FAILED" || !slices.Equal(names(before), names(after)) {
				t.Fatal("configured command mismatch after actual Policy attempted runtime/records or became complete", runErr)
			}
			if _, exited := executed.ExitCode(); exited {
				t.Fatal("unattempted runtime acquired an exit")
			}
		} else {
			execution := prepared.ExecutionContext()
			if !prepared.Valid() || !executed.RuntimeAttempted() || !executed.Complete() || execution.OperationID() != inspection.OperationID() || execution.EntryArtifact() != subject || execution.Command() != "cowsay" || execution.RuntimeIdentity() != runtimeidentity.PythonImageReference || !slices.Equal(execution.Arguments(), caseData.args) {
				t.Fatal("same-context actual runtime binding missing", runErr)
			}
			code, exited := executed.ExitCode()
			if !exited || code != caseData.code {
				t.Fatal("completed Python terminal differs", runErr, code)
			}
			if i == 0 && (runErr != nil || !strings.Contains(string(executed.Stdout()), "private-native-run-workflow-body") || len(executed.Stderr()) != 0) {
				t.Fatal("current normal Python application result differs", runErr)
			}
			if i == 1 {
				var program *sandbox.PythonProgramError
				if !errors.As(runErr, &program) || len(executed.Stderr()) == 0 {
					t.Fatal("completed Python guest error became operational failure", runErr)
				}
			}
			if previous.Valid() && previous.Bundle().ManifestID() == prepared.Bundle().ManifestID() {
				t.Fatal("current records reused")
			}
			for name, want := range map[string][]byte{"manifest.json": prepared.Bundle().ManifestDocument(), "sbom.cdx.json": prepared.Bundle().SBOMDocument()} {
				body, err := os.ReadFile(filepath.Join(bindings.stagingRoot, prepared.Bundle().ManifestID().String(), name))
				if err != nil || !bytes.Equal(body, want) {
					t.Fatal("current staged record differs", name, err)
				}
			}
			previous = prepared
		}
		if _, err := json.Marshal(executed); err == nil || strings.Contains(fmt.Sprintf("%#v", executed), "private-native-run-workflow-body") {
			t.Fatal("private Python workflow output exposed")
		}
		t.Logf("python_uv_run_application case=%d current_policy=ALLOW selector=uv-tool-run-isolated runtime_attempted=%t complete=%t source_sha256=%s", i+1, executed.RuntimeAttempted(), executed.Complete(), subject.Digest().String())
	}
}
