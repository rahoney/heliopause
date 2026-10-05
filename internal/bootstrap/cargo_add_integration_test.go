package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

// The installed CLI owns the single private selection, complete project
// inspection, independent Evidence/cache and original project publication.
func TestLinuxCargoAddIntegration(t *testing.T)           { cargoAddIntegration(t, "itoa@1.0.17", 1) }
func TestLinuxCargoTransitiveAddIntegration(t *testing.T) { cargoAddIntegration(t, "serde@1.0.228", 7) }

func cargoAddIntegration(t *testing.T, reference string, crates int) {
	t.Helper()
	if os.Getenv("HELOX_CARGO_ADD_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Cargo qualification requires Linux amd64")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[package]\nname='haa_schema_fixture'\nversion='0.1.0'\nedition='2021'\n")
	source := []byte("fn main() {}\n")
	if err := os.WriteFile(filepath.Join(project, "Cargo.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "src/main.rs"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Chdir(project)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	for attempt := 0; attempt < 2; attempt++ {
		var stdout, stderr bytes.Buffer
		addErr := bootstrap.Run(ctx, []string{"cargo", "add", reference}, &stdout, &stderr)
		if addErr != nil {
			if attempt == 0 {
				body, e := os.ReadFile(filepath.Join(project, "Cargo.toml"))
				_, lockErr := os.Lstat(filepath.Join(project, "Cargo.lock"))
				states, e2 := filepath.Glob(filepath.Join(root, "cache", "heliopause", "cargo-projects", "*.json"))
				if e != nil || !bytes.Equal(body, manifest) || !os.IsNotExist(lockErr) || e2 != nil || len(states) != 0 {
					t.Error("failed add changed original controls/approval")
				}
			}
			_, guardErr := os.Lstat(filepath.Join(project, ".heliopause-cargo-transaction.lock"))
			if !os.IsNotExist(guardErr) {
				t.Error("failed add leaked original guard")
			}
			t.Fatalf("actual Cargo add attempt%d: %v stdout=%s stderr=%s", attempt, addErr, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), fmt.Sprintf("Source: crates-io\nCrates: %d\nLock digest: ", crates)) {
			t.Fatalf("complete exact selection result missing: expected_crates=%d actual_cli=%s", crates, stdout.String())
		}
		body, e := os.ReadFile(filepath.Join(project, "Cargo.toml"))
		lock, e2 := os.ReadFile(filepath.Join(project, "Cargo.lock"))
		src, e3 := os.ReadFile(filepath.Join(project, "src/main.rs"))
		if e != nil || e2 != nil || e3 != nil || !strings.Contains(string(body), "="+strings.Split(reference, "@")[1]) || !strings.Contains(string(lock), "version = \""+strings.Split(reference, "@")[1]+"\"") || !bytes.Equal(src, source) {
			t.Fatal("published controls or source differ from frozen selection")
		}
		t.Logf("actual_cargo_add_attempt=%d %s", attempt, strings.TrimSpace(stdout.String()))
	}
	states, err := filepath.Glob(filepath.Join(root, "cache", "heliopause", "cargo-projects", "*.json"))
	if err != nil || len(states) != 1 {
		t.Fatal("independent approval missing")
	}
	body, err := os.ReadFile(states[0])
	if err != nil {
		t.Fatal(err)
	}
	var approval struct {
		Schema   int `json:"schema"`
		Approval struct {
			Graph   string `json:"graph"`
			Entries []struct {
				Crate    string            `json:"crate"`
				Evidence []json.RawMessage `json:"evidence"`
			} `json:"entries"`
		} `json:"approval"`
	}
	if json.Unmarshal(body, &approval) != nil || approval.Schema != 1 || len(approval.Approval.Entries) != crates {
		t.Fatal("retained project lacks complete recorded Evidence")
	}
	for _, entry := range approval.Approval.Entries {
		if len(entry.Evidence) != 2 {
			t.Fatal("entry Evidence incomplete")
		}
	}
}
