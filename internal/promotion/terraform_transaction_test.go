package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func writeManagedTerraformProject(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/private/tmp", "haa-terraform-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	lock := []byte("provider \"registry.terraform.io/hashicorp/aws\" {}\n")
	if err := os.WriteFile(filepath.Join(root, ".terraform.lock.hcl"), lock, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(lock)
	metadataDir := filepath.Join(root, ".heliopause")
	if err := os.Mkdir(metadataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	value := "{\"lock_sha256\":\"" + hex.EncodeToString(hash[:]) + "\"}\n"
	if err := os.WriteFile(filepath.Join(metadataDir, "terraform-transaction.json"), []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestTerraformTransactionPublishesLockAtomically(t *testing.T) {
	root := writeManagedTerraformProject(t)
	plan, err := freezeTerraformProject(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if err := os.WriteFile(filepath.Join(workspace, ".terraform.lock.hcl"), []byte("updated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	transaction, err := beginTerraformProjectTransaction(plan, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.commit(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, ".terraform.lock.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "updated\n" {
		t.Fatalf("lock = %q", body)
	}
}

func TestTerraformTransactionRejectsUnmanagedProject(t *testing.T) {
	root, err := os.MkdirTemp("/private/tmp", "haa-terraform-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.WriteFile(filepath.Join(root, ".terraform.lock.hcl"), []byte("lock\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeTerraformProject(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if _, err := beginTerraformProjectTransaction(plan, workspace); err == nil {
		t.Fatal("unmanaged Terraform project was accepted")
	}
}
