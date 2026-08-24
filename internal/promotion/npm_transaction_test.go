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
