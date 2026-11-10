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
	runPublicToolUpgradeCLIIntegration(t, false)
}

func TestLinuxPipxToolUpgradeCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_TOOL_UPDATE_CLI_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user public managed pipx upgrade CLI qualification")
	}
	runPublicToolUpgradeCLIIntegration(t, true)
}

func runPublicToolUpgradeCLIIntegration(t *testing.T, pipx bool) {
	t.Helper()
	installArgs, upgradeArgs := []string{"uv", "tool", "install"}, []string{"uv", "tool", "upgrade"}
	receiptName, receiptControl, runtimePrefix := "uv-receipt.toml", "uv-tool-host-receipt.toml", "uv-tool-upgrade:uv:"
	selector, controls := "Cowsay@6.1", 8
	parseReceipt := artifactpypi.ParseUVToolReceipt
	if pipx {
		installArgs, upgradeArgs = []string{"pipx", "install"}, []string{"pipx", "upgrade"}
		receiptName, receiptControl, runtimePrefix = "pipx_metadata.json", "pipx-tool-host-receipt.json", "pipx-tool-upgrade:pipx:"
		selector, controls = "cowsay", 9
		parseReceipt = artifactpypi.ParsePipxToolReceipt
	}
	arguments := func(prefix []string, requirement, format, target, bin string) []string {
		args := append([]string{"--lang", "ko", "--output", format}, prefix...)
		return append(args, requirement, "--target", target, "--bin-dir", bin)
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
	note := filepath.Join(venv, "user-note.txt")
	if err := os.WriteFile(note, []byte("keep managed user data"), 0600); err != nil {
		t.Fatal(err)
	}
	seed := arguments(installArgs, "cowsay==6.0", "json", venv, bin)
	var out, diagnostic bytes.Buffer
	if err := Run(ctx, seed, &out, &diagnostic); err != nil {
		t.Fatal("public old tool preparation", err, out.String(), diagnostic.String())
	}
	// UV alone models a historical unversioned requirement below. Pipx keeps
	// its genuine approved exact6.0 receipt, as its upgrade drops the old pin.
	receiptPath := filepath.Join(venv, receiptName)
	if !pipx {
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
	} else {
		body, err := os.ReadFile(receiptPath)
		if err != nil {
			t.Fatal(err)
		}
		old, err := artifactpypi.ParsePipxToolReceipt(body)
		if err != nil || old.Version() != "6.0" || old.Requirement().Requirement() != "cowsay==6.0" {
			t.Fatal("genuine pipx old receipt changed", err)
		}
		t.Log("pipx_tool_cli_original_requirement=authentic_exact6.0; old_files=actual_HAA_public6.0; no_native_adoption")
	}
	for _, format := range []string{"json", "human"} {
		t.Run(format, func(t *testing.T) {
			before := uvToolStagedIDs(t, staging)
			out.Reset()
			diagnostic.Reset()
			args := arguments(upgradeArgs, selector, format, venv, bin)
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
			if json.Unmarshal(manifest, &doc) != nil || !strings.HasPrefix(doc.Resolver.Runtime, runtimePrefix) || len(doc.Controls) != controls {
				t.Fatal("wrong client or missing original/target provenance")
			}
			published, err := os.ReadFile(receiptPath)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := parseReceipt(published)
			if err != nil || receipt.Requirement().Requirement() != "cowsay" {
				t.Fatal("original native constraint changed", err)
			}
			if err := receipt.ValidateExposedCommands(venv, bin, filepath.Join(venv, "bin/python"), map[string]string{filepath.Join(bin, "cowsay"): filepath.Join(venv, "bin/cowsay")}); err != nil {
				t.Fatal(err)
			}
			bound := false
			for _, c := range doc.Controls {
				if c.Name == receiptControl {
					bound = bytes.Equal(c.Content, published)
				}
			}
			if !bound {
				t.Fatal("published receipt differs from current approval")
			}
			if data, err := os.ReadFile(note); err != nil || string(data) != "keep managed user data" {
				t.Fatal("managed user data changed")
			}
			t.Logf("public_tool_upgrade_cli pipx=%t output=%s operation=UPDATE controls=%d current_manifest=%s", pipx, format, controls, added[0])
		})
	}
	state := uvToolTargetState(t, venv, bin)
	approved := uvToolStagedIDs(t, staging)
	out.Reset()
	diagnostic.Reset()
	if err := Run(ctx, append(arguments(upgradeArgs, "cowsay", "json", venv, bin), "--all"), &out, &diagnostic); err == nil || len(bytes.TrimSpace(out.Bytes())) != 0 || !reflect.DeepEqual(approved, uvToolStagedIDs(t, staging)) {
		t.Fatal("unsupported option did not fail before operation/approval", err, out.String())
	}
	if !reflect.DeepEqual(state, uvToolTargetState(t, venv, bin)) {
		t.Fatal("unsupported option changed managed tool")
	}
	foreign, foreignBin := filepath.Join(root, "unowned", "cowsay"), filepath.Join(root, "unowned", "commands")
	prepareUVToolTargetFixture(t, foreign, foreignBin, uvToolTargetInterpreterFixture(t))
	state = uvToolTargetState(t, foreign, foreignBin)
	uvToolCLIRefusal(t, ctx, arguments(upgradeArgs, "cowsay", "json", foreign, foreignBin), staging)
	if !reflect.DeepEqual(state, uvToolTargetState(t, foreign, foreignBin)) {
		t.Fatal("unowned upgrade changed target")
	}
}
