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
	if os.Getenv("HELOX_NPM_EXECUTION_CLI_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user actual npx/npm exec CLI qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cache, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", cache)
	for i, row := range []struct {
		path             []string
		language, output string
		show             bool
		code             int
	}{
		{[]string{"npx"}, "en", "json", false, 0},
		{[]string{"npm", "exec"}, "ko", "json", true, 1},
		{[]string{"npx"}, "ko", "human", true, 0},
		{[]string{"npm", "exec"}, "en", "human", false, 1},
	} {
		args := append([]string{"--lang", row.language, "--output", row.output, "--progress", "never", "--color", "never"}, row.path...)
		args = append(args, "--command", "tsc")
		if row.show {
			args = append(args, "--show-output")
		}
		arg := "--help"
		if row.code != 0 {
			arg = "--haa-invalid-option"
		}
		args = append(args, "typescript@5.9.3", "--", arg)
		var out, diagnostic bytes.Buffer
		cause := Run(ctx, args, &out, &diagnostic)
		if row.code == 0 && cause != nil {
			t.Fatal("actual CLI zero failed", cause)
		}
		if row.code != 0 {
			var code interface{ ExitCode() int }
			var program *sandbox.NodeProgramError
			if !errors.As(cause, &code) || code.ExitCode() != row.code || !errors.As(cause, &program) {
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
		if row.show != strings.Contains(out.String(), programText) || strings.Contains(diagnostic.String(), programText) {
			t.Fatal("private output selection or diagnostic separation failed")
		}
		t.Logf("production_npm_execution_cli case=%d alias=%s format=%s language=%s show_output=%t exit_code=%d current_policy=ALLOW guest_cleanup=CONFIRMED", i+1, row.path[len(row.path)-1], row.output, row.language, row.show, row.code)
	}
	entries, err := os.ReadDir(filepath.Join(cache, "heliopause", "staging"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	if count != 4 {
		t.Fatal("CLI requests reused execution storage records", count)
	}
}
