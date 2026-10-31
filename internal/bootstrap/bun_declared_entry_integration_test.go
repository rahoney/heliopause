//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxBunDeclaredEntryCLIIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_RUN_EXECUTION_CLI_INTEGRATION") != "1" {
		t.Fatal("declare ordinary-user Bun entry CLI qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cache, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", cache)
	for i, row := range []struct {
		path  []string
		force bool
		code  int
	}{
		{[]string{"bunx"}, false, 0},
		{[]string{"bun", "x"}, false, 1},
		{[]string{"bunx"}, true, 0},
	} {
		args := append([]string{"--output", "json", "--progress", "never", "--color", "never"}, row.path...)
		if row.force {
			args = append(args, "--bun")
		}
		arg := "--help"
		if row.code != 0 {
			arg = "--haa-invalid-option"
		}
		args = append(args, "--show-output", bunDeclaredEntryFixture.selector, "--", arg)
		var output, diagnostic bytes.Buffer
		cause := Run(ctx, args, &output, &diagnostic)
		if row.code == 0 && cause != nil {
			t.Fatal("Bun TS CLI help failed", cause)
		}
		if row.code != 0 {
			var code interface{ ExitCode() int }
			var program *sandbox.BunProgramError
			if !errors.As(cause, &code) || code.ExitCode() != row.code || !errors.As(cause, &program) {
				t.Fatal("automatic Bun runtime lost typed guest nonzero", cause)
			}
		}
		var result struct {
			Schema, Quarantine, Policy, Preparation, Execution string
			ExitCode                                           *int   `json:"exit_code"`
			Promotion                                          string `json:"output_promotion"`
			Cleanup                                            string `json:"execution_guest_cleanup"`
			Output                                             struct {
				Untrusted bool
				Encoding  string
			} `json:"program_output"`
		}
		if json.Unmarshal(output.Bytes(), &result) != nil || result.Schema != "helox.execution-result/v1" || result.Quarantine != "COMPLETED" || result.Policy != "ALLOW" || result.Preparation != "COMPLETED" || result.Execution != "COMPLETED" || result.ExitCode == nil || *result.ExitCode != row.code || result.Promotion != "NOT_REQUESTED" || result.Cleanup != "CONFIRMED" || !result.Output.Untrusted || result.Output.Encoding != "ASCII_QUOTED" {
			t.Fatal("Bun TS CLI lacks separate current result/cleanup axes")
		}
		t.Logf("production_bun_declared_entry_cli case=%d alias=%s forced=%t exit_code=%d current_policy=ALLOW guest_cleanup=CONFIRMED", i+1, row.path[len(row.path)-1], row.force, row.code)
	}
	entries, err := os.ReadDir(filepath.Join(cache, "heliopause", "staging"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		count++
		body, err := os.ReadFile(filepath.Join(cache, "heliopause", "staging", entry.Name(), "manifest.json"))
		if err != nil || !strings.Contains(string(body), sandbox.BunExecutionRuntimeIdentity()) || !strings.Contains(string(body), "bun-install") || !strings.Contains(string(body), "src/index.ts") {
			t.Fatal("Bun TS records lost original member or pinned runtime", err)
		}
	}
	if count != 3 {
		t.Fatal("Bun TS CLI did not create fresh current records", count)
	}
}
