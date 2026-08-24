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

func TestNPMProjectNodeModulesSwapReplacesOnlyAfterWorkspaceReady(t *testing.T) {
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
	if err := plan.swapNodeModules(workspace); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules", "new")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules", "old")); !os.IsNotExist(err) {
		t.Fatal("old node_modules survived")
	}
}
