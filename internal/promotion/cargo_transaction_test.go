package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func writeManagedCargoProject(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/private/tmp", "haa-cargo-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	toml, lock := []byte("[package]\nname = \"app\"\nversion = \"0.1.0\"\n"), []byte("version = 3\n")
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), toml, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.lock"), lock, 0o600); err != nil {
		t.Fatal(err)
	}
	tomlHash, lockHash := sha256.Sum256(toml), sha256.Sum256(lock)
	metaDir := filepath.Join(root, ".heliopause")
	if err := os.Mkdir(metaDir, 0o700); err != nil {
		t.Fatal(err)
	}
	metadata := "{\"cargo_toml_sha256\":\"" + hex.EncodeToString(tomlHash[:]) + "\",\"cargo_lock_sha256\":\"" + hex.EncodeToString(lockHash[:]) + "\"}\n"
	if err := os.WriteFile(filepath.Join(metaDir, "cargo-transaction.json"), []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCargoProjectTransactionPublishesBothControlFiles(t *testing.T) {
	root := writeManagedCargoProject(t)
	plan, err := freezeCargoProject(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "Cargo.toml"), []byte("[package]\nname=\"app\"\nversion=\"0.1.0\"\n[dependencies]\nserde=\"1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	transaction, err := beginCargoProjectTransaction(plan, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.commit(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) == "[package]\nname = \"app\"\nversion = \"0.1.0\"\n" {
		t.Fatal("Cargo control file was not published")
	}
}

func TestCargoProjectTransactionRejectsUnmanagedProject(t *testing.T) {
	root, err := os.MkdirTemp("/private/tmp", "haa-cargo-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname=\"app\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.lock"), []byte("version=3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := freezeCargoProject(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	if _, err := beginCargoProjectTransaction(plan, workspace); err == nil {
		t.Fatal("unmanaged Cargo project was accepted")
	}
}
