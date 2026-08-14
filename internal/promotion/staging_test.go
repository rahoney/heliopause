package promotion_test

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
	"github.com/rahoney/heliopause/internal/promotion"
)

func TestLocalStagingRehashesAndAtomicallyPersistsVerifiedSet(t *testing.T) {
	t.Parallel()
	root := realTemporaryRoot(t)
	intake, evidenceRoot, staging := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
	bundle := stagingFixture(t, intake, []byte("primary tarball"), []byte("dependency tarball"))
	adapter, err := promotion.NewLocalStaging(intake, evidenceRoot, staging)
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Stage(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if result.ManifestID() != bundle.ManifestID() || result.Handle() != "staging:"+bundle.ManifestID().String() {
		t.Fatalf("StagedSet = %#v", result)
	}
	directory := filepath.Join(staging, bundle.ManifestID().String())
	for _, name := range []string{"manifest.json", "sbom.cdx.json"} {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil || info.Mode().Perm() != 0o400 {
			t.Fatalf("%s mode/error = %v, %v", name, info, err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(directory, "artifacts"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("staged artifacts = %v, %v", entries, err)
	}
	if info, _ := os.Stat(directory); info.Mode().Perm() != 0o700 {
		t.Fatalf("staging directory mode = %o", info.Mode().Perm())
	}
	if _, err := adapter.Stage(context.Background(), bundle); err == nil {
		t.Fatal("Stage() overwrote an existing Manifest directory")
	}
}

func TestLocalStagingFailsClosedOnChangedOrSymlinkedIntake(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(string, domain.VerifiedBundle)
	}{
		{name: "changed bytes", mutate: func(intake string, bundle domain.VerifiedBundle) {
			inspection := bundle.Set().Inspected().Inspections()[0]
			_ = os.WriteFile(filepath.Join(intake, inspection.RunID().String(), "tarball.tgz"), []byte("changed content"), 0o600)
		}},
		{name: "symlink", mutate: func(intake string, bundle domain.VerifiedBundle) {
			inspection := bundle.Set().Inspected().Inspections()[0]
			path := filepath.Join(intake, inspection.RunID().String(), "tarball.tgz")
			_ = os.Remove(path)
			_ = os.Symlink(filepath.Join(intake, bundle.Set().Inspected().Inspections()[1].RunID().String(), "tarball.tgz"), path)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := realTemporaryRoot(t)
			intake, evidenceRoot, staging := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
			bundle := stagingFixture(t, intake, []byte("primary tarball"), []byte("dependency tarball"))
			test.mutate(intake, bundle)
			adapter, _ := promotion.NewLocalStaging(intake, evidenceRoot, staging)
			if _, err := adapter.Stage(context.Background(), bundle); err == nil {
				t.Fatal("Stage() error = nil")
			}
			entries, err := os.ReadDir(staging)
			if err != nil || len(entries) != 0 {
				t.Fatalf("failed Stage left files: %v, %v", entries, err)
			}
		})
	}
}

func TestLocalStagingRejectsOverlappingRootsAndTamperedDocuments(t *testing.T) {
	t.Parallel()
	root := realTemporaryRoot(t)
	if _, err := promotion.NewLocalStaging(root, filepath.Join(root, "evidence"), filepath.Join(root, "staging")); err == nil {
		t.Fatal("NewLocalStaging() accepted overlapping roots")
	}
	intake, evidenceRoot, staging := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
	bundle := stagingFixture(t, intake, []byte("primary tarball"), []byte("dependency tarball"))
	tampered, err := domain.NewVerifiedBundle(bundle.ManifestID(), bundle.Set(), []byte(`{"manifest_id":"`+bundle.ManifestID().String()+`"}`), bundle.SBOMDocument())
	if err != nil {
		t.Fatal(err)
	}
	adapter, _ := promotion.NewLocalStaging(intake, evidenceRoot, staging)
	if _, err := adapter.Stage(context.Background(), tampered); err == nil {
		t.Fatal("Stage() accepted a Manifest whose content did not match its identity")
	}
}

func realTemporaryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func stagingFixture(t *testing.T, intake string, contents ...[]byte) domain.VerifiedBundle {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	dependencies := make([]domain.LockedDependency, 0, len(contents))
	inspections := make([]domain.DependencyInspection, 0, len(contents))
	for index, content := range contents {
		nodeID, _ := domain.NewDependencyNodeID([]string{"primary", "child"}[index])
		identity, _ := domain.NewResolvedArtifactIdentity(source, []string{"primary", "child"}[index], "1.0.0", "tarball")
		resolved, _ := domain.NewResolvedArtifact(identity, "https://registry.npmjs.org/pkg/-/pkg.tgz", "sha512-declared")
		role := domain.DependencyTransitive
		if index == 0 {
			role = domain.DependencyPrimary
		}
		dependency, _ := domain.NewLockedDependencyWithRecordPath(nodeID, role, resolved, "node_modules/"+identity.Name(), false)
		dependencies = append(dependencies, dependency)
		runID, _ := domain.NewRunID()
		directory := filepath.Join(intake, runID.String())
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "tarball.tgz"), content, 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+runID.String()+":tarball", uint64(len(content)), resolved.DeclaredIntegrity())
		checkID, _ := domain.NewCheckID("stage-fixture")
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		evidenceID, _ := domain.NewEvidenceID("evidence-" + nodeID.String())
		reference, _ := domain.NewEvidenceReference(evidenceID, "evidence:"+runID.String()+":"+evidenceID.String())
		decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "entry-policy", 1, []string{"ALLOW"})
		inspection, _ := domain.NewDependencyInspection(nodeID, runID, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)
		inspections = append(inspections, inspection)
	}
	edge, _ := domain.NewDependencyEdge(dependencies[0].Node(), dependencies[1].Node())
	graph, _ := domain.NewLockedDependencyGraph(dependencies, []domain.DependencyEdge{edge})
	inspected, _ := domain.NewInspectedDependencySet(graph, inspections)
	setDecision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "set-policy", 1, []string{"ALLOW"})
	set, _ := domain.NewVerifiedSet(inspected, setDecision)
	operationID, _ := domain.NewOperationID()
	target, _ := domain.NewInstallTarget(filepath.Join(t.TempDir(), "target"))
	installContext, _ := domain.NewInstallContext(target)
	lockDigest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: operationID, InstallContext: installContext, ResolverRuntime: "fixture-runtime", LockfileDigest: lockDigest}, set)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}
