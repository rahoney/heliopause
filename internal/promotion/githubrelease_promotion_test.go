package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func TestGitHubReleasePromotionPublishesExactStagedAsset(t *testing.T) {
	root := realPromotionRoot(t)
	bundle, staged, content := githubReleasePromotionFixture(t, root)
	target := filepath.Join(root, "target")
	promoter, err := NewGitHubReleasePromotion(filepath.Join(root, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := promoter.Promote(context.Background(), staged, bundle, installContextFor(t, target))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(target, "tool.zip"))
	if err != nil || string(actual) != string(content) {
		t.Fatalf("published bytes=%q err=%v", actual, err)
	}
	info, _ := os.Stat(filepath.Join(target, "tool.zip"))
	if info.Mode().Perm() != 0o400 || result.Target().String() != target {
		t.Fatalf("mode/result=%o/%s", info.Mode().Perm(), result.Target())
	}
}

func TestGitHubReleasePromotionNeverOverwritesExistingTarget(t *testing.T) {
	root := realPromotionRoot(t)
	bundle, staged, _ := githubReleasePromotionFixture(t, root)
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "keep")
	if err := os.WriteFile(marker, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	promoter, err := NewGitHubReleasePromotion(filepath.Join(root, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := promoter.Promote(context.Background(), staged, bundle, installContextFor(t, target)); err == nil {
		t.Fatal("Promote error = nil, want existing-target rejection")
	}
	content, err := os.ReadFile(marker)
	if err != nil || string(content) != "untouched" {
		t.Fatalf("existing target changed: %q, %v", content, err)
	}
}

func githubReleasePromotionFixture(t *testing.T, root string) (domain.VerifiedBundle, domain.StagedSet, []byte) {
	t.Helper()
	content := []byte("github release promotion fixture")
	runID, _ := domain.NewRunID()
	intake := filepath.Join(root, "intake", runID.String())
	if err := os.MkdirAll(intake, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(intake, "asset"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	source, _ := domain.NewSourceID("github-release")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", "v1", "tool.zip")
	resolved, _ := domain.NewResolvedArtifact(identity, "github-release://owner/repo?asset=tool.zip", "sha256:"+digest.String())
	nodeID, _ := domain.NewDependencyNodeID("github-release-primary")
	dependency, _ := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, resolved, "tool.zip", false)
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{dependency}, nil)
	artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+runID.String()+":github-release", uint64(len(content)), "sha256:"+digest.String())
	checkID, _ := domain.NewCheckID("github-release-promotion")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("github-release-promotion")
	reference, _ := domain.NewEvidenceReference(evidenceID, "evidence:"+runID.String())
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "m6-github-release", 1, []string{"M6_VERIFIED_SET_COMPLETED"})
	inspection, _ := domain.NewDependencyInspection(nodeID, runID, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)
	inspected, _ := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{inspection})
	setDecision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "m6-github-release", 1, []string{"M6_VERIFIED_SET_COMPLETED"})
	set, _ := domain.NewVerifiedSet(inspected, setDecision)
	op, _ := domain.NewOperationID()
	lock, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: op, InstallContext: installContextFor(t, filepath.Join(root, "target")), ResolverRuntime: "github-release-standalone", LockfileDigest: lock}, set)
	if err != nil {
		t.Fatal(err)
	}
	staging, _ := NewLocalStaging(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
	staged, err := staging.Stage(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	return bundle, staged, content
}
