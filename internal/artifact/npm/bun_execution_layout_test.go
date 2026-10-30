package npm

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"path"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Synthetic measured members exercise structural binding, not native execution
// or real integrity verification. The public current workflow covers those.
func bunExecutionLayoutFixture(t *testing.T) (domain.ExecutionContext, BunInspectionLayout, ArchiveInventory) {
	t.Helper()
	source, cache, reference, _ := bunRunResolutionFixture(t, false)
	frozen, err := source.BindBunRunCache(context.Background(), cache, reference, "fixture-bun-runtime")
	if err != nil {
		t.Fatal(err)
	}
	graph := frozen.ExecutionResolution().Graph()
	node := graph.Nodes()[0]
	run, _ := domain.NewRunID()
	subject, err := domain.NewAcquiredArtifactWithIntegrity(node.Artifact().Identity(), inventoryDigest([]byte("synthetic original")), "intake:"+run.String()+":tarball", 1, node.Artifact().DeclaredIntegrity(), node.Artifact().DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	input, err := frozen.BindExecutionAcquired(graph, map[domain.DependencyNodeID]domain.AcquiredArtifact{node.Node(): subject})
	if err != nil {
		t.Fatal(err)
	}
	pkg := cache.lock.packages[0]
	manifest, err := json.Marshal(bunMetadata(t, pkg))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"package.json": string(manifest)}
	for _, member := range pkg.Bins {
		files[member] = "#!/usr/bin/env node\nconsole.log('synthetic member');\n"
	}
	var members []ArchiveEntry
	entries := map[string]layoutTarTestEntry{}
	add := func(entry layoutTarTestEntry) {
		entries[entry.header.Name] = entry
		for parent := path.Dir(entry.header.Name); parent != "."; parent = path.Dir(parent) {
			entries[parent] = layoutTestEntry(parent, tar.TypeDir, "", "")
		}
	}
	for name, body := range files {
		mode := int64(0o644)
		if name != "package.json" {
			mode = 0o755
		}
		members = append(members, ArchiveEntry{Path: name, Mode: mode, Size: int64(len(body)), Digest: inventoryDigest([]byte(body))})
		entry := layoutTestEntry(node.RecordPath()+"/"+name, tar.TypeReg, body, "")
		entry.header.Mode = mode
		add(entry)
	}
	for command, member := range pkg.Bins {
		add(layoutTestEntry("node_modules/.bin/"+command, tar.TypeSymlink, "", "../"+pkg.Name+"/"+member))
	}
	inventory, err := NewArchiveInventory(subject, members, manifest)
	if err != nil {
		t.Fatal(err)
	}
	var tree []layoutTarTestEntry
	for _, entry := range entries {
		tree = append(tree, entry)
	}
	archive, err := ParseBunLayoutArchive(context.Background(), layoutTarBody(t, tree))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := input.BindLayout(context.Background(), []ArchiveInventory{inventory}, archive)
	if err != nil {
		t.Fatal(err)
	}
	allow, _ := domain.NewPolicyDecision(domain.DecisionAllow, "fixture-policy", 1, []string{"ALLOW"})
	checkID, _ := domain.NewCheckID("fixture-check")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("fixture-evidence")
	evidence, _ := domain.NewEvidenceReference(evidenceID, "fixture:evidence")
	inspection, err := domain.NewDependencyInspection(node.Node(), run, subject, []domain.CheckExecution{check}, []domain.EvidenceReference{evidence}, allow)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{inspection})
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewVerifiedSet(inspected, allow)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := SelectExecutionEntry(inventory, "tsc")
	if err != nil {
		t.Fatal(err)
	}
	operation, _ := domain.NewOperationID()
	execution, err := domain.NewExecutionContext(domain.ExecutionContextData{OperationID: operation, Resolution: frozen.ExecutionResolution(), Set: set, EntryArtifact: subject, Command: "tsc", EntryPath: selected.Target().Path, EntryDigest: selected.Target().Digest, EntrySize: uint64(selected.Target().Size), RuntimeIdentity: "fixture-node-runtime"})
	if err != nil {
		t.Fatal(err)
	}
	return execution, prepared, inventory
}

func TestBunNodeExecutionLayoutRequiresCurrentWholeSourceAndKeepsSeparateIdentity(t *testing.T) {
	execution, prepared, primary := bunExecutionLayoutFixture(t)
	layout, err := BindBunNodeExecutionLayout(context.Background(), execution, prepared, primary)
	if err != nil || !layout.MatchesExecution(execution) || layout.nativeClient != "bun" || layout.EntryPath() != "node_modules/typescript/bin/tsc" {
		t.Fatal("current Bun tree lost Node binding", err)
	}
	if body, err := RenderNodeExecutionInputs(context.Background(), execution, layout); err != nil || len(body) == 0 {
		t.Fatal("bound native relative links lost", err)
	}
	pnpmIdentity, err := bindNodeExecutionLayoutFacts(context.Background(), execution, layout.layout, layout.entry, "pnpm")
	if err != nil || pnpmIdentity.Digest() == layout.Digest() {
		t.Fatal("Bun proof mislabeled as PNPM", err)
	}
	for _, invalid := range []BunInspectionLayout{{}, bindBunFixture(t, bunLayoutFixture(t, false))} {
		if bound, err := BindBunNodeExecutionLayout(context.Background(), execution, invalid, primary); err == nil || bound.Valid() {
			t.Fatal("absent or project-only proof accepted")
		}
	}
	_, foreign, _ := bunExecutionLayoutFixture(t)
	if bound, err := BindBunNodeExecutionLayout(context.Background(), execution, foreign, primary); err == nil || bound.Valid() {
		t.Fatal("other acquired-run proof accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if bound, err := BindBunNodeExecutionLayout(ctx, execution, prepared, primary); !errors.Is(err, context.Canceled) || bound.Valid() {
		t.Fatal("canceled introduction survived", err)
	}
	layout.nativeClient = ""
	if _, err := RenderNodeExecutionInputs(context.Background(), execution, layout); err == nil {
		t.Fatal("native proof broadened generic npm links")
	}
}
