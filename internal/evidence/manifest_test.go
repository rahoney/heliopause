package evidence_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func TestBuildVerifiedBundleIsDeterministicAndComplete(t *testing.T) {
	t.Parallel()
	set := verifiedFixture(t)
	operationID, _ := domain.NewOperationID()
	target, _ := domain.NewInstallTarget("/tmp/haa-target")
	installContext, _ := domain.NewInstallContext(target)
	lockDigest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	context := evidence.ManifestContext{OperationID: operationID, InstallContext: installContext, ResolverRuntime: "node@24.6.0+npm@11.5.1", LockfileDigest: lockDigest}

	first, err := evidence.BuildVerifiedBundle(context, set)
	if err != nil {
		t.Fatal(err)
	}
	second, err := evidence.BuildVerifiedBundle(context, set)
	if err != nil {
		t.Fatal(err)
	}
	if string(first.ManifestDocument()) != string(second.ManifestDocument()) || string(first.SBOMDocument()) != string(second.SBOMDocument()) || first.ManifestID() != second.ManifestID() {
		t.Fatal("BuildVerifiedBundle() is not deterministic")
	}
	var manifest struct {
		ManifestID string `json:"manifest_id"`
		Schema     string `json:"schema"`
		Entries    []struct {
			Node              string `json:"node"`
			RecordPath        string `json:"record_path"`
			SHA256            string `json:"sha256"`
			DeclaredIntegrity string `json:"declared_integrity"`
		} `json:"entries"`
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(first.ManifestDocument(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != "helox.verified-manifest/v1" || manifest.ManifestID != first.ManifestID().String() || len(manifest.Entries) != 2 || manifest.Entries[0].Node != "first" || manifest.Entries[0].RecordPath != "node_modules/first" || len(manifest.Edges) != 1 {
		t.Fatalf("Manifest = %#v", manifest)
	}
	var bom struct {
		BOMFormat, SpecVersion string
		Components             []json.RawMessage `json:"components"`
		Dependencies           []json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal(first.SBOMDocument(), &bom); err != nil {
		t.Fatal(err)
	}
	if bom.BOMFormat != "CycloneDX" || bom.SpecVersion != "1.7" || len(bom.Components) != 2 || len(bom.Dependencies) != 2 {
		t.Fatalf("SBOM = %#v", bom)
	}
	manifestCopy := first.ManifestDocument()
	manifestCopy[0] = 'x'
	if first.ManifestDocument()[0] == 'x' {
		t.Fatal("VerifiedBundle exposed mutable Manifest bytes")
	}
}

func verifiedFixture(t *testing.T) domain.VerifiedSet {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	firstID, _ := domain.NewDependencyNodeID("first")
	secondID, _ := domain.NewDependencyNodeID("second")
	firstIdentity, _ := domain.NewResolvedArtifactIdentity(source, "first", "1.0.0", "tarball")
	secondIdentity, _ := domain.NewResolvedArtifactIdentity(source, "second", "2.0.0", "tarball")
	firstResolved, _ := domain.NewResolvedArtifact(firstIdentity, "https://registry.npmjs.org/first/-/first-1.0.0.tgz", "sha512-first")
	secondResolved, _ := domain.NewResolvedArtifact(secondIdentity, "https://registry.npmjs.org/second/-/second-2.0.0.tgz", "sha512-second")
	first, _ := domain.NewLockedDependencyWithRecordPath(firstID, domain.DependencyPrimary, firstResolved, "node_modules/first", false)
	second, _ := domain.NewLockedDependencyWithRecordPath(secondID, domain.DependencyTransitive, secondResolved, "node_modules/first/node_modules/second", false)
	edge, _ := domain.NewDependencyEdge(firstID, secondID)
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{second, first}, []domain.DependencyEdge{edge})
	inspections := []domain.DependencyInspection{inspectionFixture(t, second, "b"), inspectionFixture(t, first, "a")}
	inspected, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "m4-set-policy", 1, []string{"M4_SET_ALLOW"})
	set, err := domain.NewVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func inspectionFixture(t *testing.T, dependency domain.LockedDependency, digestByte string) domain.DependencyInspection {
	t.Helper()
	runID, _ := domain.NewRunID()
	digest, _ := domain.NewSHA256Digest(strings.Repeat(digestByte, 64))
	artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(dependency.Artifact().Identity(), digest, "intake:"+runID.String()+":tarball", 1, dependency.Artifact().DeclaredIntegrity())
	checkID, _ := domain.NewCheckID("m4-fixture-check")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("evidence-" + dependency.Node().String())
	reference, _ := domain.NewEvidenceReference(evidenceID, "evidence:"+runID.String()+":"+evidenceID.String())
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "m4-entry-policy", 1, []string{"M4_ENTRY_ALLOW"})
	inspection, err := domain.NewDependencyInspection(dependency.Node(), runID, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)
	if err != nil {
		t.Fatal(err)
	}
	return inspection
}
