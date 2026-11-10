package npm

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func executionLayoutFixture(t *testing.T, childPath string) (domain.ExecutionContextData, []ArchiveInventory, []layoutTarTestEntry) {
	t.Helper()
	graph, err := ParsePackageLockV3(lockReference(t, "primary@1.0.0"), []byte(packageLockJSON(t, false)))
	if err != nil {
		t.Fatal(err)
	}
	if childPath != "" {
		nodes := graph.Nodes()
		for i, node := range nodes {
			if node.Artifact().Identity().Name() == "child" {
				nodes[i], err = domain.NewLockedDependencyWithRecordPath(node.Node(), node.Role(), node.Artifact(), childPath, false)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		graph, err = domain.NewLockedDependencyGraph(nodes, graph.Edges())
		if err != nil {
			t.Fatal(err)
		}
	}
	allow, err := domain.NewPolicyDecision(domain.DecisionAllow, "fixture-policy", 1, []string{"ALLOW"})
	if err != nil {
		t.Fatal(err)
	}
	checkID, _ := domain.NewCheckID("fixture-check")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("fixture-evidence")
	evidence, _ := domain.NewEvidenceReference(evidenceID, "fixture:evidence")
	var inventories []ArchiveInventory
	var inspections []domain.DependencyInspection
	var entries []layoutTarTestEntry
	directories := map[string]bool{"node_modules": true}
	var primary domain.AcquiredArtifact
	for _, node := range graph.Nodes() {
		name := node.Artifact().Identity().Name()
		manifest := fmt.Sprintf(`{"name":%q,"version":"1.0.0"}`, name)
		if name == "primary" {
			manifest = `{"name":"primary","version":"1.0.0","bin":"cli.js","dependencies":{"child":"1.0.0"}}`
		}
		body := "module.exports = 1;\n"
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(node.Artifact().Identity(), inventoryDigest([]byte(name)), "intake:"+run.String()+":tarball", 1, node.Artifact().DeclaredIntegrity())
		if err != nil {
			t.Fatal(err)
		}
		inventory, err := NewArchiveInventory(artifact, []ArchiveEntry{
			{Path: "package.json", Mode: 0o644, Size: int64(len(manifest)), Digest: inventoryDigest([]byte(manifest))},
			{Path: "cli.js", Mode: 0o644, Size: int64(len(body)), Digest: inventoryDigest([]byte(body))},
		}, []byte(manifest))
		if err != nil {
			t.Fatal(err)
		}
		inventories = append(inventories, inventory)
		inspection, err := domain.NewDependencyInspection(node.Node(), run, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{evidence}, allow)
		if err != nil {
			t.Fatal(err)
		}
		inspections = append(inspections, inspection)
		entries = append(entries, layoutTestEntry(node.RecordPath()+"/package.json", tar.TypeReg, manifest, ""), layoutTestEntry(node.RecordPath()+"/cli.js", tar.TypeReg, body, ""))
		for root := node.RecordPath(); root != ""; root = parentDirectory(root) {
			directories[root] = true
		}
		if node.Node() == graph.Primary() {
			primary = artifact
		}
	}
	for directory := range directories {
		entries = append(entries, layoutTestEntry(directory, tar.TypeDir, "", ""))
	}
	slices.SortFunc(entries, func(a, b layoutTarTestEntry) int { return strings.Compare(a.header.Name, b.header.Name) })
	inspected, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewVerifiedSet(inspected, allow)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewDependencyResolution(graph, "fixture-npm-resolver", inventoryDigest([]byte("original lock")))
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	return domain.ExecutionContextData{OperationID: operation, Resolution: resolution, Set: set, EntryArtifact: primary, Command: "primary", EntryPath: "cli.js", EntryDigest: inventoryDigest([]byte("module.exports = 1;\n")), EntrySize: uint64(len("module.exports = 1;\n")), RuntimeIdentity: "fixture-node-runtime", Arguments: []string{"private-argument"}}, inventories, entries
}

func parentDirectory(value string) string {
	index := strings.LastIndexByte(value, '/')
	if index < 0 {
		return ""
	}
	return value[:index]
}

func bindExecutionFixture(t *testing.T, data domain.ExecutionContextData, inventories []ArchiveInventory, entries []layoutTarTestEntry) (NodeExecutionLayout, error) {
	t.Helper()
	execution, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
	if err != nil {
		t.Fatal("fixture envelope", err)
	}
	return BindNodeExecutionLayout(context.Background(), execution, inventories, layout)
}

func TestNodeExecutionLayoutBindsWholeGraphAndPrivateRequest(t *testing.T) {
	t.Parallel()
	data, inventories, entries := executionLayoutFixture(t, "")
	bound, err := bindExecutionFixture(t, data, inventories, entries)
	if err != nil || !bound.Valid() || bound.EntryPath() != "node_modules/primary/cli.js" {
		t.Fatal(err)
	}
	execution, _ := domain.NewExecutionContext(data)
	if !bound.MatchesExecution(execution) {
		t.Fatal("fresh binding lost")
	}
	body, err := bound.ReadFile(bound.EntryPath())
	if err != nil {
		t.Fatal(err)
	}
	body[0] = 'X'
	again, _ := bound.ReadFile(bound.EntryPath())
	if again[0] != 'm' {
		t.Fatal("body getter mutated layout")
	}
	facts := bound.Entries()
	facts[0].Name = "foreign"
	if bound.Entries()[0].Name == "foreign" {
		t.Fatal("entry getter mutated layout")
	}
	data.Arguments = []string{"different-private-argument"}
	other, err := bindExecutionFixture(t, data, inventories, entries)
	if err != nil || other.Digest() != bound.Digest() {
		t.Fatal("argv changed file identity", err)
	}
	execution, _ = domain.NewExecutionContext(data)
	if !bound.MatchesExecution(execution) {
		t.Fatal("argv became layout authority")
	}
	data.OperationID, _ = domain.NewOperationID()
	execution, _ = domain.NewExecutionContext(data)
	if bound.MatchesExecution(execution) {
		t.Fatal("another operation reused current binding")
	}
	if body, err := json.Marshal(bound); err == nil || len(body) != 0 {
		t.Fatal("raw layout serialized")
	}
	if strings.Contains(fmt.Sprintf("%#v", bound), "module.exports") {
		t.Fatal("raw source formatted")
	}
}

func TestNodeExecutionLayoutRejectsChangedFullCensus(t *testing.T) {
	t.Parallel()
	for _, change := range []string{"primary bytes", "secondary bytes", "secondary missing", "foreign file", "file mode", "symlink", "wrong inventory", "missing inventory", "wrong handle", "wrong entry", "wrong command"} {
		t.Run(change, func(t *testing.T) {
			data, inventories, entries := executionLayoutFixture(t, "")
			for i := range entries {
				if entries[i].header.Name == "node_modules/child/cli.js" {
					switch change {
					case "secondary bytes":
						entries[i].body = strings.Replace(entries[i].body, "1", "2", 1)
					case "secondary missing":
						entries = append(entries[:i], entries[i+1:]...)
					case "file mode":
						entries[i].header.Mode = 0o755
					case "symlink":
						entries[i] = layoutTestEntry(entries[i].header.Name, tar.TypeSymlink, "", "package.json")
					}
					break
				}
			}
			switch change {
			case "primary bytes":
				for i := range entries {
					if entries[i].header.Name == "node_modules/primary/cli.js" {
						entries[i].body = strings.Replace(entries[i].body, "1", "2", 1)
					}
				}
			case "foreign file":
				entries = append(entries, layoutTestEntry("node_modules/foreign", tar.TypeReg, "extra", ""))
			case "wrong inventory":
				inventories[0].entries[0].Digest = inventoryDigest([]byte("different"))
			case "missing inventory":
				inventories = inventories[:1]
			case "wrong handle":
				run, _ := domain.NewRunID()
				old := inventories[0].artifact
				declared, _ := old.DeclaredIntegrity()
				inventories[0].artifact, _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(old.Identity(), old.Digest(), "intake:"+run.String()+":tarball", old.SizeBytes(), declared)
			case "wrong entry":
				data.EntryDigest = inventoryDigest([]byte("wrong"))
			case "wrong command":
				data.Command = "unselected"
			}
			if bound, err := bindExecutionFixture(t, data, inventories, entries); err == nil || bound.Valid() {
				t.Fatal("changed source/layout accepted")
			}
		})
	}
}

func TestNodeExecutionLayoutRejectsUnreachableNodeLookupAndCanceledInput(t *testing.T) {
	t.Parallel()
	data, inventories, entries := executionLayoutFixture(t, "node_modules/foreign/node_modules/child")
	if bound, err := bindExecutionFixture(t, data, inventories, entries); err == nil || bound.Valid() {
		t.Fatal("graph edge differs from Node lookup")
	}
	data, inventories, entries = executionLayoutFixture(t, "")
	execution, _ := domain.NewExecutionContext(data)
	layout, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("fixture stopped")
	cancel(cause)
	if bound, err := BindNodeExecutionLayout(ctx, execution, inventories, layout); bound.Valid() || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation cause lost", err)
	}
	for _, input := range []struct {
		ctx       context.Context
		execution domain.ExecutionContext
		layout    PNPMLayoutArchive
	}{{nil, execution, layout}, {context.Background(), domain.ExecutionContext{}, layout}, {context.Background(), execution, PNPMLayoutArchive{}}} {
		if bound, err := BindNodeExecutionLayout(input.ctx, input.execution, inventories, input.layout); err == nil || bound.Valid() {
			t.Fatal("invalid input accepted")
		}
	}
}
