//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxNodeExecutionPreparationIntegration(t *testing.T) {
	if os.Getenv("HELOX_NODE_PREPARATION_INTEGRATION") != "1" || os.Geteuid() == 0 {
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
	native, err := sandbox.NewLinuxNPMResolverWithExecutorAndPolicy(executor, observer.Observer(), resolverPolicy)
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
	service, err := newNPMExecutionPreparationService(native, source, static, backend, store, staging)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactnpm.ParseReference("typescript@5.9.3")
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewExecutionPreparationRequest(reference, "tsc", []string{"--help"})
	if err != nil {
		t.Fatal(err)
	}
	var previous application.PreparedExecution
	for attempt := 1; attempt <= 2; attempt++ {
		prepared, err := service.Prepare(ctx, request)
		if err != nil || !prepared.Valid() {
			t.Fatalf("production preparation attempt=%d boundary=%s: %v", attempt, prepared.FailureCode(), err)
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
		if identity.Source().String() != "npm" || identity.Name() != "typescript" || identity.Version() != "5.9.3" || !ok || declared != "sha512-jl1vZzPDinLr9eUt3J/t7V6FgNEw9QjvBPdysz9KfQDD41fQrC2Y4vKQdiaUpFT4bXlb1RHhLpp8wtm6M5TgSw==" {
			t.Fatal("actual source differs from fixed official fixture")
		}
		checks := map[string]bool{}
		for _, check := range entry.Checks() {
			if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
				t.Fatal("required preparation check incomplete", check.ID().String())
			}
			checks[check.ID().String()] = true
		}
		for _, name := range []string{"npm-declared-integrity", "npm-static-archive", "npm-dynamic-lifecycle"} {
			if !checks[name] {
				t.Fatal("actual required check absent", name)
			}
		}
		if entry.PolicyDecision().PolicyID() != "m3-npm-dynamic-inspect" || entry.PolicyDecision().Decision() != domain.DecisionAllow || len(entry.Evidence()) < 3 {
			t.Fatal("current production entry Policy/Evidence missing")
		}
		execution := prepared.ExecutionContext()
		if execution.OperationID() != inspected.OperationID() || execution.EntryArtifact() != subject || execution.Command() != "tsc" || execution.EntryPath() != "bin/tsc" || execution.EntrySize() == 0 || execution.RuntimeIdentity() != runtimeidentity.NodeImageReference || execution.SourceFingerprint() != inspected.Resolution().LockfileDigest() || !slices.Equal(execution.Arguments(), request.Arguments()) {
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
		materialized, err := materializeNodeExecution(ctx, backend, static, execution)
		if err != nil || !materialized.MatchesExecution(execution) || materialized.EntryPath() != "node_modules/typescript/bin/tsc" {
			t.Fatal("actual offline materialization did not retain current source/entry", err)
		}
		body, err := materialized.ReadFile(materialized.EntryPath())
		if err != nil || uint64(len(body)) != execution.EntrySize() {
			t.Fatal("materialized Node entry unavailable", err)
		}
		for _, file := range materialized.Entries() {
			if file.Name == "node_modules/.package-lock.json" {
				t.Fatal("native metadata entered executable source tree")
			}
		}
		if err := backend.ValidateNodeExecutionInputs(ctx, execution, materialized); err != nil {
			t.Fatal("actual readonly Node input view failed", err)
		}
		t.Logf("production_node_inputs attempt=%d layout=%s readonly=true user_program_run=false", attempt, materialized.Digest().String())
		t.Logf("production_node_materialization attempt=%d layout=%s entry=%s source_sha256=%s offline=true lifecycle_scripts=false user_program_run=false", attempt, materialized.Digest().String(), materialized.EntryPath(), subject.Digest().String())
		t.Logf("production_node_preparation attempt=%d subjects=%d checks=%d Evidence=%d entry=%s source_sha256=%s manifest=%s current_policy=ALLOW storage_only=true user_program_run=false", attempt, len(entries), len(entry.Checks()), len(entry.Evidence()), execution.EntryPath(), subject.Digest().String(), prepared.Bundle().ManifestID().String())
		previous = prepared
	}
}
