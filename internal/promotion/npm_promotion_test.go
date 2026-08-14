package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func TestNPMPromotionUsesPinnedOfflineRuntimeAndPublishesExactSet(t *testing.T) {
	t.Parallel()
	root := realPromotionRoot(t)
	bundle, staged := stagedPromotionFixture(t, root)
	target := filepath.Join(root, "published")
	contextValue := installContextFor(t, target)
	runner := &fixturePromotionRunner{bundle: bundle}
	adapter, err := newNPMPromotion(filepath.Join(root, "staging"), runner, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Promote(context.Background(), staged, bundle, contextValue)
	if err != nil {
		t.Fatal(err)
	}
	if result.ManifestID() != bundle.ManifestID() || result.Target().String() != target || runner.calls != 1 {
		t.Fatalf("Promote() = %#v, runner calls=%d", result, runner.calls)
	}
	joined := strings.Join(runner.arguments, " ")
	for _, required := range []string{"--pull never", "--network none", "--read-only", "--cap-drop ALL", "no-new-privileges", "--entrypoint npm", promotionNodeImage, "ci --offline --ignore-scripts --no-audit --no-fund --bin-links=false"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("runtime arguments lack %q: %s", required, joined)
		}
	}
	lock, err := os.ReadFile(filepath.Join(target, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(lock), "registry.npmjs.org") || !strings.Contains(string(lock), "file:.heliopause/artifacts/") {
		t.Fatalf("generated lock contains non-local source: %s", lock)
	}
}

func TestNPMPromotionFailsClosedAndRemovesTemporaryTarget(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(*fixturePromotionRunner, string)
	}{
		{name: "runtime failure", mutate: func(r *fixturePromotionRunner, _ string) { r.err = errors.New("runtime failed") }},
		{name: "lock mutation", mutate: func(r *fixturePromotionRunner, _ string) { r.mutateLock = true }},
		{name: "target race", mutate: func(r *fixturePromotionRunner, target string) { r.racedTarget = target }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := realPromotionRoot(t)
			bundle, staged := stagedPromotionFixture(t, root)
			target := filepath.Join(root, "published")
			runner := &fixturePromotionRunner{bundle: bundle}
			test.mutate(runner, target)
			adapter, _ := newNPMPromotion(filepath.Join(root, "staging"), runner, "linux", "amd64")
			if _, err := adapter.Promote(context.Background(), staged, bundle, installContextFor(t, target)); err == nil {
				t.Fatal("Promote() error = nil")
			}
			if runner.racedTarget == "" {
				if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("failed Promotion published target: %v", err)
				}
			} else {
				body, err := os.ReadFile(filepath.Join(target, "owner"))
				if err != nil || string(body) != "external" {
					t.Fatalf("target race was overwritten: %q, %v", body, err)
				}
			}
			matches, _ := filepath.Glob(filepath.Join(root, ".published.haa-*"))
			if len(matches) != 0 {
				t.Fatalf("temporary targets remain: %v", matches)
			}
		})
	}
}

func TestNPMPromotionRejectsExistingTargetAndUnsupportedPlatform(t *testing.T) {
	t.Parallel()
	root := realPromotionRoot(t)
	bundle, staged := stagedPromotionFixture(t, root)
	target := filepath.Join(root, "published")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	runner := &fixturePromotionRunner{bundle: bundle}
	adapter, _ := newNPMPromotion(filepath.Join(root, "staging"), runner, "linux", "amd64")
	if _, err := adapter.Promote(context.Background(), staged, bundle, installContextFor(t, target)); err == nil || runner.calls != 0 {
		t.Fatalf("existing target: error=%v calls=%d", err, runner.calls)
	}
	unsupported, _ := newNPMPromotion(filepath.Join(root, "staging"), runner, "darwin", "arm64")
	if _, err := unsupported.Promote(context.Background(), staged, bundle, installContextFor(t, filepath.Join(root, "other"))); err == nil {
		t.Fatal("unsupported platform Promotion error = nil")
	}
}

func TestPromotionImageMatchesRuntimeLock(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "scripts", "runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		NodeImage struct {
			Reference string `json:"reference"`
		} `json:"node_image"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.NodeImage.Reference != promotionNodeImage {
		t.Fatalf("Promotion image=%q runtime lock=%q", promotionNodeImage, lock.NodeImage.Reference)
	}
}

type fixturePromotionRunner struct {
	bundle      domain.VerifiedBundle
	arguments   []string
	calls       int
	err         error
	mutateLock  bool
	racedTarget string
}

func (r *fixturePromotionRunner) Run(_ context.Context, project string, arguments []string) error {
	r.calls++
	r.arguments = append([]string(nil), arguments...)
	if r.err != nil {
		return r.err
	}
	if r.mutateLock {
		if err := os.WriteFile(filepath.Join(project, "package-lock.json"), []byte(`{"changed":true}`), 0o600); err != nil {
			return err
		}
	}
	for _, node := range r.bundle.Set().Inspected().Graph().Nodes() {
		directory := filepath.Join(project, filepath.FromSlash(node.RecordPath()))
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]string{"name": node.Artifact().Identity().Name(), "version": node.Artifact().Identity().Version()})
		if err := os.WriteFile(filepath.Join(directory, "package.json"), body, 0o600); err != nil {
			return err
		}
	}
	if r.racedTarget != "" {
		if err := os.Mkdir(r.racedTarget, 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(r.racedTarget, "owner"), []byte("external"), 0o600)
	}
	return nil
}

func stagedPromotionFixture(t *testing.T, root string) (domain.VerifiedBundle, domain.StagedSet) {
	t.Helper()
	intake := filepath.Join(root, "intake")
	evidenceRoot := filepath.Join(root, "evidence")
	stagingRoot := filepath.Join(root, "staging")
	source, _ := domain.NewSourceID("npm")
	contents := [][]byte{[]byte("primary tarball"), []byte("child tarball")}
	names := []string{"primary", "child"}
	dependencies := make([]domain.LockedDependency, 0, 2)
	inspections := make([]domain.DependencyInspection, 0, 2)
	for index, name := range names {
		nodeID, _ := domain.NewDependencyNodeID(name)
		identity, _ := domain.NewResolvedArtifactIdentity(source, name, "1.0.0", "tarball")
		resolved, _ := domain.NewResolvedArtifact(identity, "https://registry.npmjs.org/"+name+"/-/"+name+"-1.0.0.tgz", "sha512-declared")
		role := domain.DependencyTransitive
		if index == 0 {
			role = domain.DependencyPrimary
		}
		dependency, _ := domain.NewLockedDependencyWithRecordPath(nodeID, role, resolved, "node_modules/"+name, false)
		dependencies = append(dependencies, dependency)
		runID, _ := domain.NewRunID()
		directory := filepath.Join(intake, runID.String())
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "tarball.tgz"), contents[index], 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(contents[index])
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+runID.String()+":tarball", uint64(len(contents[index])), resolved.DeclaredIntegrity())
		checkID, _ := domain.NewCheckID("promotion-fixture")
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		evidenceID, _ := domain.NewEvidenceID("evidence-" + name)
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
	lockDigest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	installContext := installContextFor(t, filepath.Join(root, "unused-target"))
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: operationID, InstallContext: installContext, ResolverRuntime: "node:22.23.1;npm:10.9.8", LockfileDigest: lockDigest}, set)
	if err != nil {
		t.Fatal(err)
	}
	staging, _ := NewLocalStaging(intake, evidenceRoot, stagingRoot)
	staged, err := staging.Stage(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	return bundle, staged
}

func installContextFor(t *testing.T, target string) domain.InstallContext {
	t.Helper()
	value, err := domain.NewInstallTarget(target)
	if err != nil {
		t.Fatal(err)
	}
	contextValue, err := domain.NewInstallContext(value)
	if err != nil {
		t.Fatal(err)
	}
	return contextValue
}

func realPromotionRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}
