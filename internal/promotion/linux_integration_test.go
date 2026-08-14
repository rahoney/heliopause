package promotion

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func TestLinuxNPMPromotionIntegration(t *testing.T) {
	if os.Getenv("HELOX_PROMOTION_INTEGRATION") != "1" {
		t.Skip("set HELOX_PROMOTION_INTEGRATION=1 in the controlled Linux qualification environment")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bundle, staged := validStagedNPMFixture(t, root)
	target := filepath.Join(root, "target")
	adapter, err := NewNPMPromotion(filepath.Join(root, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Promote(context.Background(), staged, bundle, installContextFor(t, target))
	if err != nil {
		t.Fatal(err)
	}
	if result.Target().String() != target {
		t.Fatalf("Promoted target = %s", result.Target())
	}
	body, err := os.ReadFile(filepath.Join(target, "node_modules", "haa-promotion-fixture", "index.js"))
	if err != nil || string(body) != "module.exports = 42;\n" {
		t.Fatalf("installed package body=%q error=%v", body, err)
	}
}

func validStagedNPMFixture(t *testing.T, root string) (domain.VerifiedBundle, domain.StagedSet) {
	t.Helper()
	runID, _ := domain.NewRunID()
	intake := filepath.Join(root, "intake")
	runDirectory := filepath.Join(intake, runID.String())
	if err := os.MkdirAll(runDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	tarball := filepath.Join(runDirectory, "tarball.tgz")
	writeFixtureTarball(t, tarball)
	content, err := os.ReadFile(tarball)
	if err != nil {
		t.Fatal(err)
	}
	sha256Sum := sha256.Sum256(content)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sha256Sum[:]))
	sha512Sum := sha512.Sum512(content)
	declared := "sha512-" + base64.StdEncoding.EncodeToString(sha512Sum[:])
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "haa-promotion-fixture", "1.0.0", "tarball")
	resolved, _ := domain.NewResolvedArtifact(identity, "https://registry.npmjs.org/haa-promotion-fixture/-/haa-promotion-fixture-1.0.0.tgz", declared)
	nodeID, _ := domain.NewDependencyNodeID("fixture")
	dependency, _ := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, resolved, "node_modules/haa-promotion-fixture", false)
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{dependency}, nil)
	artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+runID.String()+":tarball", uint64(len(content)), declared)
	checkID, _ := domain.NewCheckID("promotion-integration")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("promotion-integration")
	reference, _ := domain.NewEvidenceReference(evidenceID, "evidence:"+runID.String()+":"+evidenceID.String())
	entryDecision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "entry-policy", 1, []string{"ALLOW"})
	inspection, _ := domain.NewDependencyInspection(nodeID, runID, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, entryDecision)
	inspected, _ := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{inspection})
	setDecision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "set-policy", 1, []string{"ALLOW"})
	set, _ := domain.NewVerifiedSet(inspected, setDecision)
	operationID, _ := domain.NewOperationID()
	lockDigest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: operationID, InstallContext: installContextFor(t, filepath.Join(root, "target")), ResolverRuntime: "node:22.23.1;npm:10.9.8", LockfileDigest: lockDigest}, set)
	if err != nil {
		t.Fatal(err)
	}
	staging, _ := NewLocalStaging(intake, filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
	staged, err := staging.Stage(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	return bundle, staged
}

func writeFixtureTarball(t *testing.T, path string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	files := map[string]string{
		"package/package.json": `{"name":"haa-promotion-fixture","version":"1.0.0","main":"index.js"}`,
		"package/index.js":     "module.exports = 42;\n",
	}
	for _, name := range []string{"package/package.json", "package/index.js"} {
		body := []byte(files[name])
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
