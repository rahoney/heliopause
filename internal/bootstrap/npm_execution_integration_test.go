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

func TestLinuxNPMExecutionCLIIntegration(t *testing.T) {
	testNodeExecutionCLIIntegration(t, "npm")
}

func TestLinuxPNPMRunExecutionCLIIntegration(t *testing.T) {
	testNodeExecutionCLIIntegration(t, "pnpm")
}

func TestLinuxBunRunExecutionCLIIntegration(t *testing.T) {
	testNodeExecutionCLIIntegration(t, "bun")
}

func TestLinuxBunProgramExecutionCLIIntegration(t *testing.T) {
	testNodeExecutionCLIIntegration(t, "bun-program")
}

func testNodeExecutionCLIIntegration(t *testing.T, client string) {
	t.Helper()
	declared := os.Getenv("HELOX_NPM_EXECUTION_CLI_INTEGRATION") == "1"
	if client == "pnpm" {
		declared = os.Getenv("HELOX_PNPM_RUN_EXECUTION_CLI_INTEGRATION") == "1"
	}
	if client == "bun" || client == "bun-program" {
		declared = os.Getenv("HELOX_BUN_RUN_EXECUTION_CLI_INTEGRATION") == "1"
	}
	if client == "bun-program" {
		declared = os.Getenv("HELOX_BUN_PROGRAM_EXECUTION_CLI_INTEGRATION") == "1"
	}
	if !declared || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user actual npx/npm exec CLI qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cache, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", cache)
	type cliCase struct {
		path             []string
		language, output string
		show             bool
		code             int
	}
	rows := []cliCase{
		{[]string{"npx"}, "en", "json", false, 0},
		{[]string{"npm", "exec"}, "ko", "json", true, 1},
		{[]string{"npx"}, "ko", "human", true, 0},
		{[]string{"npm", "exec"}, "en", "human", false, 1},
	}
	if client == "pnpm" {
		rows = []cliCase{
			{[]string{"pnpm", "dlx"}, "en", "json", false, 0},
			{[]string{"pnx"}, "ko", "json", true, 1},
			{[]string{"pnpx"}, "ko", "human", true, 0},
		}
	}
	if client == "bun" || client == "bun-program" {
		rows = []cliCase{{[]string{"bunx"}, "en", "json", false, 0}, {[]string{"bun", "x"}, "ko", "json", true, 1}, {[]string{"bunx"}, "ko", "human", true, 0}}
	}
	for i, row := range rows {
		implicit := client == "pnpm" && row.path[0] == "pnpx" || (client == "bun" || client == "bun-program") && i == 2
		args := append([]string{"--lang", row.language, "--output", row.output, "--progress", "never", "--color", "never"}, row.path...)
		if client == "bun-program" {
			args = append(args, "--bun")
		}
		if client == "pnpm" || (client == "bun" || client == "bun-program") {
			if !implicit {
				args = append(args, "--package", "typescript@5.9.3")
			}
		} else {
			args = append(args, "--command", "tsc")
		}
		if row.show {
			args = append(args, "--show-output")
		}
		arg := "--help"
		if row.code != 0 {
			arg = "--haa-invalid-option"
		}
		selector := "typescript@5.9.3"
		if client == "pnpm" || (client == "bun" || client == "bun-program") {
			selector = "tsc"
		}
		if implicit {
			selector = "rimraf@3.0.2"
		}
		args = append(args, selector, "--", arg)
		var out, diagnostic bytes.Buffer
		cause := Run(ctx, args, &out, &diagnostic)
		if row.code == 0 && cause != nil {
			t.Fatal("actual CLI zero failed", cause)
		}
		if row.code != 0 {
			var code interface{ ExitCode() int }
			var program *sandbox.NodeProgramError
			typed := errors.As(cause, &program)
			if client == "bun-program" {
				var bun *sandbox.BunProgramError
				typed = errors.As(cause, &bun)
			}
			if !errors.As(cause, &code) || code.ExitCode() != row.code || !typed {
				t.Fatal("actual guest nonzero lost typed cause or process exit", cause)
			}
		}
		if row.output == "json" {
			var report struct {
				Schema      string
				Quarantine  string
				Policy      string
				Preparation string
				Execution   string
				ExitCode    *int   `json:"exit_code"`
				Promotion   string `json:"output_promotion"`
				Cleanup     string `json:"execution_guest_cleanup"`
				Output      struct {
					Untrusted   bool
					Encoding    string
					StdoutBytes int    `json:"stdout_bytes"`
					Stdout      string `json:"stdout_escaped"`
				} `json:"program_output"`
			}
			if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.execution-result/v1" || report.Quarantine != "COMPLETED" || report.Policy != "ALLOW" || report.Preparation != "COMPLETED" || report.Execution != "COMPLETED" || report.ExitCode == nil || *report.ExitCode != row.code || report.Promotion != "NOT_REQUESTED" || report.Cleanup != "CONFIRMED" || !report.Output.Untrusted || report.Output.StdoutBytes == 0 {
				t.Fatal("actual separate execution JSON incomplete")
			}
			if row.show != (report.Output.Encoding == "ASCII_QUOTED") || !row.show && report.Output.Stdout != "" {
				t.Fatal("explicit output selection lost")
			}
		} else {
			if json.Valid(out.Bytes()) || !strings.Contains(out.String(), "CONFIRMED") || !strings.Contains(out.String(), "NOT_REQUESTED") || row.language == "ko" && !strings.Contains(out.String(), "검역") {
				t.Fatal("actual human execution axes absent")
			}
		}
		programText := "Version 5.9.3"
		if row.code != 0 {
			programText = "Unknown compiler option"
		}
		if implicit {
			programText = "Usage: rimraf"
		}
		if row.show != strings.Contains(out.String(), programText) || strings.Contains(diagnostic.String(), programText) {
			t.Fatal("private output selection or diagnostic separation failed")
		}
		marker := "production_npm_execution_cli"
		if client == "pnpm" {
			marker = "production_pnpm_run_cli"
		}
		if client == "bun" || client == "bun-program" {
			marker = "production_bun_run_cli"
		}
		if client == "bun-program" {
			marker = "production_bun_program_cli"
		}
		t.Logf("%s case=%d alias=%s format=%s language=%s show_output=%t exit_code=%d current_policy=ALLOW guest_cleanup=CONFIRMED", marker, i+1, row.path[len(row.path)-1], row.output, row.language, row.show, row.code)
	}
	entries, err := os.ReadDir(filepath.Join(cache, "heliopause", "staging"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	multi := false
	for _, entry := range entries {
		if entry.IsDir() {
			count++
			if client == "pnpm" || (client == "bun" || client == "bun-program") {
				body, err := os.ReadFile(filepath.Join(cache, "heliopause", "staging", entry.Name(), "manifest.json"))
				var manifest struct {
					Entries   []json.RawMessage
					Edges     []json.RawMessage
					Execution struct{ Command string }
				}
				identity := "pnpm-dlx-isolated:" + sandbox.PNPMResolverRuntimeIdentity()
				if client == "bun" || client == "bun-program" {
					identity = "bunx-isolated:" + sandbox.BunResolverRuntimeIdentity()
				}
				if err != nil || json.Unmarshal(body, &manifest) != nil || !strings.Contains(string(body), identity) {
					t.Fatal("CLI lost its current native whole records", err)
				}
				if client == "bun-program" && !strings.Contains(string(body), sandbox.BunExecutionRuntimeIdentity()) {
					t.Fatal("forced Bun CLI lost runtime identity in records")
				}
				if manifest.Execution.Command == "rimraf" && len(manifest.Entries) > 1 && len(manifest.Edges) > 0 {
					multi = true
					marker := "production_pnpm_run_cli_graph"
					if client == "bun" || client == "bun-program" {
						marker = "production_bun_run_cli_graph"
					}
					if client == "bun-program" {
						marker = "production_bun_program_cli_graph"
					}
					t.Logf("%s nodes=%d edges=%d primary=rimraf current_policy=ALLOW", marker, len(manifest.Entries), len(manifest.Edges))
				}
			}
		}
	}
	if count != len(rows) || (client == "pnpm" || (client == "bun" || client == "bun-program")) && !multi {
		t.Fatal("CLI requests reused execution storage records", count)
	}
}
