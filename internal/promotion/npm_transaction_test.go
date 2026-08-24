package promotion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNPMProjectPlanFailsClosedOnControlFileChange(t *testing.T) {
	root := realPromotionRoot(t)
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"lockfileVersion":3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeNPMProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.verifyUnchanged(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"lockfileVersion":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := plan.verifyUnchanged(); err == nil {
		t.Fatal("changed project was accepted")
	}
}

func TestNPMProjectGuardRejectsConcurrentMutation(t *testing.T) {
	root := realPromotionRoot(t)
	first, err := acquireNPMProjectGuard(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireNPMProjectGuard(root); err == nil {
		t.Fatal("concurrent mutation lock was accepted")
	}
	if err := first.release(); err != nil {
		t.Fatal(err)
	}
}

func TestNPMProjectMetadataBindsCommittedControlFiles(t *testing.T) {
	root := realPromotionRoot(t)
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeNPMProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.writeMetadata(); err != nil {
		t.Fatal(err)
	}
	if err := plan.verifyManagedOrEmpty(); err != nil {
		t.Fatal(err)
	}
}

func TestNPMProjectPrivateWorkspaceDoesNotMutateSource(t *testing.T) {
	root := realPromotionRoot(t)
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeNPMProject(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"changed":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil || string(body) != `{"dependencies":{}}` {
		t.Fatalf("source project changed: %q, %v", body, err)
	}
}

func TestNPMProjectTransactionPublishesVerifiedSetTogether(t *testing.T) {
	root := realPromotionRoot(t)
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "node_modules"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "old"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, _ := freezeNPMProject(root)
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if err := os.Mkdir(filepath.Join(workspace, "node_modules"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "node_modules", "new"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, ".heliopause", "artifacts"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".heliopause", "artifacts", "verified.tgz"), []byte("verified"), 0o600); err != nil {
		t.Fatal(err)
	}
	transaction, err := beginNPMProjectTransaction(plan, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules", "new")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules", "old")); !os.IsNotExist(err) {
		t.Fatal("old node_modules survived")
	}
	committed, err := freezeNPMProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := committed.verifyManagedOrEmpty(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".heliopause", "artifacts", "verified.tgz")); err != nil {
		t.Fatalf("verified local artifact was not committed: %v", err)
	}
}

func TestNPMProjectTransactionRollsBackPartialPublish(t *testing.T) {
	root := realPromotionRoot(t)
	oldManifest := []byte(`{"name":"old","dependencies":{}}`)
	oldLock := []byte(`{"packages":{"":{}}}`)
	for name, body := range map[string][]byte{"package.json": oldManifest, "package-lock.json": oldLock} {
		if err := os.WriteFile(filepath.Join(root, name), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "node_modules"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "old"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeNPMProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.writeMetadata(); err != nil {
		t.Fatal(err)
	}
	originalMetadata, err := os.ReadFile(filepath.Join(root, npmTransactionMetadata))
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"name":"new"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace, "node_modules"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace, ".heliopause"), 0o700); err != nil {
		t.Fatal(err)
	}
	// package-lock.json is deliberately absent: package.json has already been
	// published when this failure is reached, so the test exercises rollback.
	if err := os.Remove(filepath.Join(workspace, "package-lock.json")); err != nil {
		t.Fatal(err)
	}
	transaction, err := beginNPMProjectTransaction(plan, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.commit(); err == nil {
		t.Fatal("partial publish was accepted")
	}
	manifest, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil || string(manifest) != string(oldManifest) {
		t.Fatalf("package.json was not rolled back: %q, %v", manifest, err)
	}
	lock, err := os.ReadFile(filepath.Join(root, "package-lock.json"))
	if err != nil || string(lock) != string(oldLock) {
		t.Fatalf("package-lock.json was not rolled back: %q, %v", lock, err)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules", "old")); err != nil {
		t.Fatalf("node_modules was not rolled back: %v", err)
	}
	metadata, err := os.ReadFile(filepath.Join(root, npmTransactionMetadata))
	if err != nil || string(metadata) != string(originalMetadata) {
		t.Fatalf("metadata was not rolled back: %q, %v", metadata, err)
	}
	if err := rejectInterruptedNPMTransaction(root); err != nil {
		t.Fatalf("rolled back transaction left a blocking state: %v", err)
	}
}
