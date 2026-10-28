package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func nodeContextFixture(t *testing.T) (domain.OperationID, domain.DependencyResolution, domain.VerifiedSet, artifactnpm.NodeExecutionEntry) {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tool", "1.0.0", "tarball")
	resolved, _ := domain.NewResolvedArtifact(identity, "registry:npm:tool@1.0.0", "sha512-tool")
	node, _ := domain.NewDependencyNodeID("tool")
	dependency, _ := domain.NewLockedDependency(node, domain.DependencyPrimary, resolved)
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{dependency}, nil)
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":tarball", 1, resolved.DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	checkID, _ := domain.NewCheckID("fixture-check")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("fixture-evidence")
	reference, _ := domain.NewEvidenceReference(evidenceID, "fixture:evidence")
	allow, _ := domain.NewPolicyDecision(domain.DecisionAllow, "fixture-policy", 1, []string{"ALLOW"})
	inspection, err := domain.NewDependencyInspection(node, run, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, allow)
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
	lock, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
	resolution, err := domain.NewDependencyResolution(graph, "original-npm-resolver", lock)
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"name":"tool","version":"1.0.0","bin":"cli.js"}`)
	body := []byte("console.log('fixture')\n")
	memberDigest := func(body []byte) domain.ContentDigest {
		sum := sha256.Sum256(body)
		d, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	inventory, err := artifactnpm.NewArchiveInventory(artifact, []artifactnpm.ArchiveEntry{
		{Path: "package.json", Size: int64(len(manifest)), Mode: 0o644, Digest: memberDigest(manifest)},
		{Path: "cli.js", Size: int64(len(body)), Mode: 0o644, Digest: memberDigest(body)},
	}, manifest)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifactnpm.SelectExecutionEntry(inventory, "")
	if err != nil {
		t.Fatal(err)
	}
	proof, err := entry.BindNodeTarget(body)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	return operation, resolution, set, proof
}

func TestNodeExecutionCompositionKeepsProofAndCanonicalRuntime(t *testing.T) {
	operation, resolution, set, proof := nodeContextFixture(t)
	args := []string{"", "a b", "--", "$(data)"}
	execution, err := newNodeExecutionContext(operation, resolution, set, proof, args)
	if err != nil || !execution.Valid() {
		t.Fatal(err)
	}
	entry, member := proof.Entry(), proof.Entry().Target()
	if execution.RuntimeIdentity() != runtimeidentity.NodeImageReference || execution.EntryArtifact() != entry.Artifact() || execution.Command() != entry.Command() || execution.EntryPath() != member.Path || execution.EntryDigest() != member.Digest || execution.EntrySize() != uint64(member.Size) || execution.OperationID() != operation || execution.SourceFingerprint() != resolution.LockfileDigest() || !slices.Equal(execution.Arguments(), args) {
		t.Fatal("composition changed exact proof, runtime or request")
	}
	args[0] = "changed"
	if execution.Arguments()[0] != "" {
		t.Fatal("composition retained mutable argv")
	}
}

func TestNodeExecutionCompositionRejectsUnboundOrDifferentAcquisition(t *testing.T) {
	operation, resolution, set, proof := nodeContextFixture(t)
	if execution, err := newNodeExecutionContext(operation, resolution, set, artifactnpm.NodeExecutionEntry{}, nil); err == nil || execution.Valid() {
		t.Fatal("missing byte proof accepted")
	}
	_, _, _, other := nodeContextFixture(t)
	if execution, err := newNodeExecutionContext(operation, resolution, set, other, nil); err == nil || execution.Valid() {
		t.Fatal("same package and bytes from another intake substituted")
	}
	if execution, err := newNodeExecutionContext(operation, resolution, domain.VerifiedSet{}, proof, nil); err == nil || execution.Valid() {
		t.Fatal("missing whole-set approval accepted")
	}
	if execution, err := newNodeExecutionContext(operation, resolution, set, proof, []string{"bad\x00argument"}); err == nil || execution.Valid() {
		t.Fatal("invalid argument bypassed domain preflight")
	}
}
