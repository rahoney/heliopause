//go:build linux

package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestLinuxGoTidyCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_RESOLVER_PROJECT_INTEGRATION") != "1" {
		t.Skip("requires pinned Go/gVisor and authenticated helper")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user CLI")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	mod := []byte("module example.com/haa-fixture\n\ngo 1.26.0\n\nrequire github.com/spf13/pflag v1.0.9\n")
	write := func(name string, body []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, name), body, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", mod)
	write("main.go", []byte("package fixture\nimport _ \"github.com/spf13/pflag\"\nfunc init(){panic(\"tidy must never execute\")}\n"))
	write("user.note", []byte("preserve user data"))
	cache := filepath.Join(root, "cache", "heliopause")
	t.Setenv("XDG_CACHE_HOME", filepath.Dir(cache))
	t.Chdir(project)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	var out, diag bytes.Buffer
	if err := bootstrap.Run(ctx, []string{"--output", "json", "go", "mod", "tidy"}, &out, &diag); err == nil {
		t.Fatal("native project implicitly adopted")
	}
	if current, err := os.ReadFile(filepath.Join(project, "go.mod")); err != nil || !bytes.Equal(current, mod) {
		t.Fatal("refused adoption changed controls")
	}
	seen := map[string]bool{}
	apply := func(name string, count int, flags ...string) {
		t.Helper()
		out.Reset()
		diag.Reset()
		args := append([]string{"--lang", "ko", "--output", "json", "go", "mod", "tidy"}, flags...)
		if err := bootstrap.Run(ctx, args, &out, &diag); err != nil {
			t.Fatalf("tidy %s failed: %v stdout=%s stderr=%s", name, err, out.String(), diag.String())
		}
		var report uvSyncActualReport
		if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.project-tidy-result/v1" || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Policy.Decision != "ALLOW" || len(report.Entries) != count || report.DependencyFree != (count == 0) {
			t.Fatal("incomplete tidy report", out.String())
		}
		runs := map[string]string{}
		for _, entry := range report.Entries {
			if entry.Run == "" || seen[entry.Run] || len(entry.Checks) < 2 || len(entry.Evidence) < 2 {
				t.Fatal("tidy skipped fresh inspection")
			}
			seen[entry.Run] = true
			runs[entry.Identity.Name] = entry.Run
			for _, check := range entry.Checks {
				if check.Required && check.Status != "COMPLETED" {
					t.Fatal("required tidy check incomplete")
				}
			}
		}
		states, err := filepath.Glob(filepath.Join(cache, "go-projects", "*.json"))
		if err != nil || len(states) != 1 {
			t.Fatal("missing independent tidy approval")
		}
		body, err := os.ReadFile(states[0])
		if err != nil {
			t.Fatal(err)
		}
		var state struct {
			Cache       string `json:"cache"`
			CacheDigest string `json:"cache_digest"`
			Approval    struct {
				Graph    string `json:"graph"`
				Controls []struct {
					Path string `json:"path"`
					SHA  string `json:"sha256"`
				} `json:"controls"`
				Entries []struct {
					Module string `json:"module"`
					Run    string `json:"run"`
				} `json:"entries"`
			} `json:"approval"`
		}
		if json.Unmarshal(body, &state) != nil || state.Approval.Graph != report.Graph || len(state.Approval.Controls) != 2 || len(state.Approval.Entries) != count {
			t.Fatal("approval differs from inspection")
		}
		for _, control := range state.Approval.Controls {
			if control.Path != "go.mod" && control.Path != "go.sum" {
				t.Fatal("unexpected published control")
			}
			body, err := os.ReadFile(filepath.Join(project, control.Path))
			hash := sha256.Sum256(body)
			if err != nil || hex.EncodeToString(hash[:]) != control.SHA {
				t.Fatal("published controls differ from approved graph")
			}
		}
		for _, entry := range state.Approval.Entries {
			if runs[entry.Module] != entry.Run {
				t.Fatal("approval lost fresh run binding")
			}
		}
		id, ok := strings.CutPrefix(state.Cache, "project-cache:")
		if !ok {
			t.Fatal("invalid cache handle")
		}
		if _, err := domain.ParseRunID(id); err != nil {
			t.Fatal(err)
		}
		receipt, err := os.ReadFile(filepath.Join(cache, "go-verified-cache", id, "receipt.json"))
		hash := sha256.Sum256(receipt)
		if err != nil || hex.EncodeToString(hash[:]) != state.CacheDigest {
			t.Fatal("cache receipt differs from retained approval")
		}
		note, err := os.ReadFile(filepath.Join(project, "user.note"))
		if err != nil || string(note) != "preserve user data" {
			t.Fatal("user data changed")
		}
		if _, err := os.Lstat(filepath.Join(project, ".heliopause-go-transaction.lock")); !os.IsNotExist(err) {
			t.Fatal("tidy retained transaction lock")
		}
		t.Logf("actual_go_tidy_cli=PASS case=%s entries=%d fresh_checks=%t graph=%s", name, count, count > 0, report.Graph)
	}
	apply("adopt", 1, "--adopt")
	before := map[string]os.FileInfo{}
	for _, name := range []string{"go.mod", "go.sum"} {
		info, err := os.Stat(filepath.Join(project, name))
		if err != nil {
			t.Fatal(err)
		}
		before[name] = info
	}
	apply("noop", 1)
	for name, original := range before {
		current, err := os.Stat(filepath.Join(project, name))
		if err != nil || !os.SameFile(original, current) || current.Mode() != original.Mode() || !current.ModTime().Equal(original.ModTime()) {
			t.Fatal("tidy no-op replaced controls")
		}
	}
	write("main.go", []byte("package fixture\nimport _ \"fmt\"\n"))
	apply("remove-all", 0)
}
