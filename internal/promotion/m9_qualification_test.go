package promotion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestM9QualificationRejectsUnmanagedNPMDependencyGraph(t *testing.T) {
	root := realPromotionRoot(t)
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{"unmanaged":"1.0.0"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(`{"packages":{"":{"dependencies":{"unmanaged":"1.0.0"}},"node_modules/unmanaged":{"version":"1.0.0"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeNPMProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.verifyManagedOrEmpty(); err == nil {
		t.Fatal("unmanaged npm dependency graph was accepted")
	}
}

func TestM9QualificationRejectsInterruptedNPMCommit(t *testing.T) {
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
	if err := os.Mkdir(filepath.Join(root, ".heliopause-npm-commit-stale"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := beginNPMProjectTransaction(plan, workspace); err == nil {
		t.Fatal("interrupted npm transaction was reused")
	}
}

func TestM9QualificationRejectsModifiedHAAOwnedVenvState(t *testing.T) {
	_, plan := makePypiVenvFixture(t)
	firstOutput := filepath.Join(plan.root, "output-first")
	if err := os.MkdirAll(firstOutput, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firstOutput, "owned.py"), []byte("v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstState, err := plan.outputState(firstOutput)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(firstOutput, firstState); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.site, "owned.py"), []byte("host-modified\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondOutput := filepath.Join(plan.root, "output-second")
	if err := os.MkdirAll(secondOutput, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondOutput, "new.py"), []byte("v2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondState, err := plan.outputState(secondOutput)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(secondOutput, secondState); err == nil {
		t.Fatal("modified HAA-owned venv state was accepted")
	}
}
