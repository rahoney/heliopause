package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func writeManagedGoProject(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mod := []byte("module example.com/app\n\ngo 1.25\n")
	sum := []byte("example.com/dep v1.0.0 h1:abc\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), mod, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), sum, 0o600); err != nil {
		t.Fatal(err)
	}
	modHash, sumHash := sha256.Sum256(mod), sha256.Sum256(sum)
	metaDir := filepath.Join(root, ".heliopause")
	if err := os.Mkdir(metaDir, 0o700); err != nil {
		t.Fatal(err)
	}
	metadata := "{\"go_mod_sha256\":\"" + hex.EncodeToString(modHash[:]) + "\",\"go_sum_sha256\":\"" + hex.EncodeToString(sumHash[:]) + "\"}\n"
	if err := os.WriteFile(filepath.Join(metaDir, "go-transaction.json"), []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestGoProjectTransactionPublishesControlFilesTogether(t *testing.T) {
	root := writeManagedGoProject(t)
	plan, err := freezeGoProject(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/app\n\ngo 1.25\n\nrequire example.com/new v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	transaction, err := beginGoProjectTransaction(plan, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.commit(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) == "module example.com/app\n\ngo 1.25\n" {
		t.Fatal("Go project control file was not published")
	}
	if _, err := os.Stat(transaction.backup); !os.IsNotExist(err) {
		t.Fatalf("rollback backup remains: %v", err)
	}
}

func TestGoProjectTransactionRejectsDriftAndConcurrentLock(t *testing.T) {
	root := writeManagedGoProject(t)
	plan, err := freezeGoProject(root)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := acquireGoProjectGuard(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireGoProjectGuard(root); err == nil {
		t.Fatal("concurrent Go transaction was accepted")
	}
	if err := guard.release(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), []byte("drift\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := plan.verifyUnchanged(); err == nil {
		t.Fatal("Go control-file drift was accepted")
	}
}

func TestGoProjectRequiresManagedMetadata(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), []byte("sum\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeGoProject(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if _, err := beginGoProjectTransaction(plan, workspace); err == nil {
		t.Fatal("unmanaged Go project was accepted")
	}
}
