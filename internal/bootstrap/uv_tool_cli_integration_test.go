//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestLinuxUVToolInstallCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_INSTALL_CLI_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user public UV tool CLI qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "cache")
	t.Setenv("XDG_CACHE_HOME", cache)
	staging := filepath.Join(cache, "heliopause/staging")
	interpreter := uvToolTargetInterpreterFixture(t)
	for _, step := range []struct{ name, requirement, output string }{{"json", "Cowsay@6.1", "json"}, {"human with spaces", "cowsay>=6.1,<=6.1", "human"}} {
		t.Run(step.name, func(t *testing.T) {
			venv, bin := filepath.Join(root, step.name, "cowsay"), filepath.Join(root, step.name, "commands")
			prepareUVToolTargetFixture(t, venv, bin, interpreter)
			user := filepath.Join(venv, "user-note.txt")
			if err := os.WriteFile(user, []byte("keep user data"), 0600); err != nil {
				t.Fatal(err)
			}
			before := uvToolStagedIDs(t, staging)
			args := []string{"--lang", "ko", "--output", step.output, "uv", "tool", "install", step.requirement, "--target", venv, "--bin-dir", bin}
			var out, diagnostic bytes.Buffer
			if err := Run(ctx, args, &out, &diagnostic); err != nil {
				t.Fatal("actual public tool CLI", err, out.String(), diagnostic.String())
			}
			var result struct {
				Schema    string `json:"schema_version"`
				Operation string
				Status    string `json:"operation_status"`
				Promotion string `json:"promotion_status"`
				Policy    struct{ Decision string }
				Verified  struct {
					Manifest string `json:"manifest_id"`
					Count    int    `json:"entry_count"`
				} `json:"verified_set"`
				Checks []struct {
					Required bool
					Status   string `json:"execution_status"`
				}
				Evidence []struct{ ID, Handle string } `json:"evidence_references"`
			}
			if step.output == "json" {
				if json.Unmarshal(out.Bytes(), &result) != nil || result.Schema != "helox.operation-result/v1" || result.Operation != "INSTALL" || result.Status != "COMPLETED" || result.Promotion != "COMPLETED" || result.Policy.Decision != "ALLOW" || result.Verified.Count != 1 || result.Verified.Manifest == "" || len(result.Checks) < 3 || len(result.Evidence) < 3 {
					t.Fatal("incomplete tool operation result", out.String())
				}
				for _, check := range result.Checks {
					if check.Required && check.Status != "COMPLETED" {
						t.Fatal("tool required check incomplete")
					}
				}
			} else if json.Valid(out.Bytes()) || len(bytes.TrimSpace(out.Bytes())) == 0 {
				t.Fatal("explicit human result missing")
			}
			after := uvToolStagedIDs(t, staging)
			var added []string
			for id := range after {
				if !before[id] {
					added = append(added, id)
				}
			}
			if len(added) != 1 || step.output == "json" && added[0] != result.Verified.Manifest {
				t.Fatal("current tool manifest not uniquely staged")
			}
			manifest, err := os.ReadFile(filepath.Join(staging, added[0], "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Resolver struct {
					Runtime string `json:"runtime_identity"`
				}
				Controls []struct {
					Name    string
					Content []byte
				} `json:"project_controls"`
			}
			if json.Unmarshal(manifest, &doc) != nil || !strings.HasPrefix(doc.Resolver.Runtime, "uv-tool-install:") || len(doc.Controls) != 6 {
				t.Fatal("public tool used another client or lacks current controls")
			}
			receipt, err := os.ReadFile(filepath.Join(venv, "uv-receipt.toml"))
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := artifactpypi.ParseUVToolReceipt(receipt)
			if err != nil {
				t.Fatal(err)
			}
			requirement, err := artifactpypi.ParseUpgradeRequirement(step.requirement)
			if err != nil || !parsed.Requirement().SameConstraints(requirement) {
				t.Fatal("native tool constraint changed")
			}
			matched := false
			for _, control := range doc.Controls {
				if control.Name == "uv-tool-host-receipt.toml" {
					matched = bytes.Equal(control.Content, receipt)
				}
			}
			if !matched {
				t.Fatal("native receipt is not the current approved control")
			}
			if link, err := os.Readlink(filepath.Join(bin, "cowsay")); err != nil || link != filepath.Join(venv, "bin/cowsay") {
				t.Fatal("public tool command not exposed correctly")
			}
			metadata, err := os.ReadFile(filepath.Join(venv, "lib/python3.14/site-packages/cowsay-6.1.dist-info/METADATA"))
			if err != nil || !bytes.Contains(metadata, []byte("Version: 6.1")) {
				t.Fatal("approved tool wheel missing")
			}
			if data, err := os.ReadFile(user); err != nil || string(data) != "keep user data" {
				t.Fatal("tool user data changed")
			}
			for _, directory := range []string{venv, bin} {
				entries, err := os.ReadDir(directory)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".haa-") || strings.HasPrefix(entry.Name(), ".heliopause-") {
						t.Fatal("tool work data or mutation guard leaked", entry.Name())
					}
				}
			}
			state := uvToolTargetState(t, venv, bin)
			uvToolCLIRefusal(t, ctx, []string{"--output", "json", "uv", "tool", "install", "cowsay==6.1", "--target", venv, "--bin-dir", bin}, staging)
			if !reflect.DeepEqual(state, uvToolTargetState(t, venv, bin)) {
				t.Fatal("fresh tool refusal changed existing target")
			}
		})
	}
	for _, kind := range []string{"unowned-site", "occupied-command"} {
		t.Run(kind, func(t *testing.T) {
			venv, bin := filepath.Join(root, kind, "cowsay"), filepath.Join(root, kind, "commands")
			prepareUVToolTargetFixture(t, venv, bin, interpreter)
			foreign := filepath.Join(venv, "lib/python3.14/site-packages/foreign.pth")
			if kind == "occupied-command" {
				foreign = filepath.Join(bin, "cowsay")
			}
			if err := os.WriteFile(foreign, []byte("preserve foreign data"), 0600); err != nil {
				t.Fatal(err)
			}
			before := uvToolTargetState(t, venv, bin)
			uvToolCLIRefusal(t, ctx, []string{"--output", "json", "uv", "tool", "install", "cowsay==6.1", "--target", venv, "--bin-dir", bin}, staging)
			if !reflect.DeepEqual(before, uvToolTargetState(t, venv, bin)) {
				t.Fatal("foreign tool refusal changed data")
			}
		})
	}
}

func uvToolStagedIDs(t *testing.T, root string) map[string]bool {
	t.Helper()
	result := map[string]bool{}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return result
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 64 {
			result[entry.Name()] = true
		}
	}
	return result
}

func uvToolCLIRefusal(t *testing.T, ctx context.Context, args []string, staging string) {
	t.Helper()
	before := uvToolStagedIDs(t, staging)
	var out, diagnostic bytes.Buffer
	if err := Run(ctx, args, &out, &diagnostic); err == nil {
		t.Fatal("unsupported existing/foreign tool accepted", out.String())
	}
	var result struct {
		Status    string `json:"operation_status"`
		Promotion string `json:"promotion_status"`
		Policy    struct{ Decision string }
	}
	if json.Unmarshal(out.Bytes(), &result) != nil || result.Status == "COMPLETED" || result.Promotion == "COMPLETED" || result.Policy.Decision == "ALLOW" {
		t.Fatal("refusal reported approval/publication", out.String(), diagnostic.String())
	}
	if !reflect.DeepEqual(before, uvToolStagedIDs(t, staging)) {
		t.Fatal("refused tool created approved staging")
	}
}
