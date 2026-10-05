package promotion

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestCargoProjectPlanRejectsAliasedControls(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			root := writeManagedCargoProject(t)
			control := filepath.Join(root, "Cargo.toml")
			if kind == "oversize" {
				if err := os.WriteFile(control, bytes.Repeat([]byte("x"), 4<<20+1), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				outside := filepath.Join(canonicalGoTestRoot(t), "owned-fixture.toml")
				body, err := os.ReadFile(control)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(outside, body, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(control); err != nil {
					t.Fatal(err)
				}
				if kind == "symlink" {
					err = os.Symlink(outside, control)
				} else {
					err = os.Link(outside, control)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := freezeCargoProject(root); err == nil {
				t.Fatal("unbounded or aliased Cargo control accepted")
			}
		})
	}
}

func TestCargoProjectPlanPreservesOriginalControlIdentity(t *testing.T) {
	root := writeManagedCargoProject(t)
	plan, err := freezeCargoProject(root)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, "Cargo.toml")
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(name, name+".original"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := plan.verifyUnchanged(); err == nil {
		t.Fatal("same-byte control replacement accepted")
	}
}

func writeManagedCargoProject(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
