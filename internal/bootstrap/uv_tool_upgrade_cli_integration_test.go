//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestLinuxUVToolUpgradeCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_UPDATE_CLI_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user public managed UV tool upgrade qualification")
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
	venv, bin := filepath.Join(root, "env with spaces", "cowsay"), filepath.Join(root, "commands")
	prepareUVToolTargetFixture(t, venv, bin, uvToolTargetInterpreterFixture(t))
	seed := []string{"--output", "json", "uv", "tool", "install", "cowsay==6.0", "--target", venv, "--bin-dir", bin}
	var out, diagnostic bytes.Buffer
	if err := Run(ctx, seed, &out, &diagnostic); err != nil {
		t.Fatal("public old tool preparation", err, out.String(), diagnostic.String())
	}
	// The public6.0 files have actual current HAA ownership. Only the prior
	// unversioned receipt condition is synthetic; an exact6.0 native pin stays.
	receiptPath := filepath.Join(venv, "uv-receipt.toml")
	body, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := toml.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	wire["tool"].(map[string]any)["requirements"] = []map[string]any{{"name": "cowsay"}}
	body, err = toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifactpypi.ParseUVToolReceipt(body); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("uv_tool_cli_original_requirement=synthetic_unversioned; old_files=actual_HAA_public6.0; no_native_adoption")
	for _, format := range []string{"json", "human"} {
		t.Run(format, func(t *testing.T) {
			before := uvToolStagedIDs(t, staging)
			out.Reset()
			diagnostic.Reset()
			args := []string{"--lang", "ko", "--output", format, "uv", "tool", "upgrade", "Cowsay@6.1", "--target", venv, "--bin-dir", bin}
			if err := Run(ctx, args, &out, &diagnostic); err != nil {
				t.Fatal("public tool upgrade", err, out.String(), diagnostic.String())
			}
			var result struct {
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
			if format == "json" {
				if json.Unmarshal(out.Bytes(), &result) != nil || result.Operation != "UPDATE" || result.Status != "COMPLETED" || result.Promotion != "COMPLETED" || result.Policy.Decision != "ALLOW" || result.Verified.Count != 1 || result.Verified.Manifest == "" || len(result.Checks) < 3 || len(result.Evidence) < 3 {
					t.Fatal("public update result incomplete", out.String())
				}
				for _, check := range result.Checks {
					if check.Required && check.Status != "COMPLETED" {
						t.Fatal("required current check incomplete")
					}
				}
			} else if json.Valid(out.Bytes()) || len(bytes.TrimSpace(out.Bytes())) == 0 {
				t.Fatal("human update result missing")
			}
			var added []string
			for id := range uvToolStagedIDs(t, staging) {
				if !before[id] {
					added = append(added, id)
				}
			}
			if len(added) != 1 || format == "json" && added[0] != result.Verified.Manifest {
				t.Fatal("new current Manifest missing")
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
			if json.Unmarshal(manifest, &doc) != nil || !strings.HasPrefix(doc.Resolver.Runtime, "uv-tool-upgrade:uv:") || len(doc.Controls) != 8 {
				t.Fatal("wrong client or missing original/target provenance")
			}
			published, err := os.ReadFile(receiptPath)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := artifactpypi.ParseUVToolReceipt(published)
			if err != nil || receipt.Requirement().Requirement() != "cowsay" {
				t.Fatal("original native constraint changed", err)
			}
			if err := receipt.ValidateExposedCommands(venv, bin, filepath.Join(venv, "bin/python"), map[string]string{filepath.Join(bin, "cowsay"): filepath.Join(venv, "bin/cowsay")}); err != nil {
				t.Fatal(err)
			}
			bound := false
			for _, c := range doc.Controls {
				if c.Name == "uv-tool-host-receipt.toml" {
					bound = bytes.Equal(c.Content, published)
				}
			}
			if !bound {
				t.Fatal("published receipt differs from current approval")
			}
			t.Logf("uv_tool_upgrade_cli output=%s operation=UPDATE controls=8 current_manifest=%s", format, added[0])
		})
	}
	state := uvToolTargetState(t, venv, bin)
	approved := uvToolStagedIDs(t, staging)
	out.Reset()
	diagnostic.Reset()
	if err := Run(ctx, []string{"--output", "json", "uv", "tool", "upgrade", "cowsay", "--target", venv, "--bin-dir", bin, "--adopt"}, &out, &diagnostic); err == nil || len(bytes.TrimSpace(out.Bytes())) != 0 || !reflect.DeepEqual(approved, uvToolStagedIDs(t, staging)) {
		t.Fatal("unsupported option did not fail before operation/approval", err, out.String())
	}
	if !reflect.DeepEqual(state, uvToolTargetState(t, venv, bin)) {
		t.Fatal("unsupported adoption changed managed tool")
	}
	foreign, foreignBin := filepath.Join(root, "unowned", "cowsay"), filepath.Join(root, "unowned", "commands")
	prepareUVToolTargetFixture(t, foreign, foreignBin, uvToolTargetInterpreterFixture(t))
	state = uvToolTargetState(t, foreign, foreignBin)
	uvToolCLIRefusal(t, ctx, []string{"--output", "json", "uv", "tool", "upgrade", "cowsay", "--target", foreign, "--bin-dir", foreignBin}, staging)
	if !reflect.DeepEqual(state, uvToolTargetState(t, foreign, foreignBin)) {
		t.Fatal("unowned upgrade changed target")
	}
}
