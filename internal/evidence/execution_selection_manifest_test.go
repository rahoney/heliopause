package evidence_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func selectionRecordsFixture(t *testing.T) domain.ExecutionSelectionContextData {
	t.Helper()
	old := executionRecordsFixture(t)
	var roots []domain.LockedDependencyGraph
	for _, node := range old.Resolution.Graph().Nodes() {
		root, err := domain.NewLockedDependencyWithRecordPath(node.Node(), domain.DependencyPrimary, node.Artifact(), node.RecordPath(), node.HostInstallAction())
		if err != nil {
			t.Fatal(err)
		}
		graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{root}, nil)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, graph)
	}
	selection, err := domain.NewExecutionDependencySelection(roots[0], roots[1:])
	if err != nil {
		t.Fatal(err)
	}
	var entries []domain.DependencyInspection
	for _, entry := range old.Set.Inspected().Inspections() {
		id, _ := domain.NewCheckID("current-verification")
		check, _ := domain.NewCheckExecution(id, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		current, err := domain.NewDependencyInspection(entry.Node(), entry.RunID(), entry.Artifact(), append(entry.Checks(), check), entry.Evidence(), entry.PolicyDecision())
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, current)
	}
	inspected, err := domain.NewInspectedExecutionSelection(selection, entries)
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewVerifiedExecutionSelection(inspected, old.Set.Decision())
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewExecutionSelectionResolution(selection, old.Resolution.RuntimeIdentity(), old.Resolution.LockfileDigest())
	if err != nil {
		t.Fatal(err)
	}
	return domain.ExecutionSelectionContextData{OperationID: old.OperationID, Resolution: resolution, Set: set, EntryArtifact: old.EntryArtifact, Command: old.Command, EntryPath: old.EntryPath, EntryDigest: old.EntryDigest, EntrySize: old.EntrySize, RuntimeIdentity: old.RuntimeIdentity, Arguments: old.Arguments}
}

func TestExecutionSelectionRecordsPreserveIndependentRootsAndPrivateArguments(t *testing.T) {
	data := selectionRecordsFixture(t)
	build := func() domain.ExecutionSelectionBundle {
		execution, err := domain.NewExecutionSelectionContext(data)
		if err != nil {
			t.Fatal(err)
		}
		bundle, err := (evidence.Generator{}).BuildExecutionSelection(context.Background(), execution)
		if err != nil || !bundle.Valid() {
			t.Fatal("whole records unavailable", err)
		}
		return bundle
	}
	first := build()
	var manifest struct {
		Schema  string
		Roots   []string
		Entries []json.RawMessage
		Edges   []json.RawMessage
		Target  json.RawMessage
	}
	if json.Unmarshal(first.ManifestDocument(), &manifest) != nil || manifest.Schema != "helox.verified-execution-selection-manifest/v1" || len(manifest.Roots) != 2 || len(manifest.Entries) != 2 || len(manifest.Edges) != 0 || manifest.Target != nil {
		t.Fatal("roots lost or invented edges/target")
	}
	var sbom struct {
		Components   []json.RawMessage
		Dependencies []struct {
			DependsOn []string `json:"dependsOn"`
		}
	}
	if json.Unmarshal(first.SBOMDocument(), &sbom) != nil || len(sbom.Components) != 2 || len(sbom.Dependencies) != 2 {
		t.Fatal("SBOM missed independent entries")
	}
	for _, dependency := range sbom.Dependencies {
		if len(dependency.DependsOn) != 0 {
			t.Fatal("SBOM invented an edge")
		}
	}
	data.Arguments = []string{"changed-private-argv"}
	second := build()
	if first.ManifestID() != second.ManifestID() || !bytes.Equal(first.ManifestDocument(), second.ManifestDocument()) || !bytes.Equal(first.SBOMDocument(), second.SBOMDocument()) || bytes.Contains(first.ManifestDocument(), []byte("private-argument-fixture")) {
		t.Fatal("private argv entered records")
	}
	manifestCopy := first.ManifestDocument()
	manifestCopy[0] = 'x'
	if first.ManifestDocument()[0] != '{' {
		t.Fatal("mutable bundle records")
	}
	data.Resolution, _ = domain.NewExecutionSelectionResolution(data.Resolution.Selection(), "changed-resolver", data.Resolution.LockfileDigest())
	if build().ManifestID() == first.ManifestID() {
		t.Fatal("resolver binding lost")
	}
	if b, err := (evidence.Generator{}).BuildExecutionSelection(context.Background(), domain.ExecutionSelectionContext{}); err == nil || b.Valid() {
		t.Fatal("absent whole approval became records")
	}
}
