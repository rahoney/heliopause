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

func TestLinuxNPMExecutionCLIRefusalIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_EXECUTION_CLI_REFUSAL_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user post-Policy CLI refusal qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cache, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", cache)
	for i, row := range []struct {
		name string
		args []string
	}{
		{"ambiguous", []string{"npx", "typescript@5.9.3", "--", "--help", "private-cli-argv"}},
		{"missing", []string{"npm", "exec", "--command", "haa-nonexistent-bin", "typescript@5.9.3", "--", "--help", "private-cli-argv"}},
	} {
		args := append([]string{"--lang", "en", "--output", "json", "--progress", "never", "--color", "never"}, row.args...)
		var out, diagnostic bytes.Buffer
		cause := Run(ctx, args, &out, &diagnostic)
		var code interface{ ExitCode() int }
		var completed *sandbox.NodeProgramError
		if cause == nil || !errors.As(cause, &code) || code.ExitCode() != 1 || errors.As(cause, &completed) {
			t.Fatal("entry refusal became success/completed program or lost process failure", cause)
		}
		var report struct {
			Schema      string
			Quarantine  string
			Policy      string
			Preparation string
			Execution   string
			Exit        *int   `json:"exit_code"`
			Signal      *int   `json:"signal"`
			Promotion   string `json:"output_promotion"`
			Cleanup     string `json:"execution_guest_cleanup"`
			Failure     string `json:"failure_code"`
			Output      struct {
				Untrusted bool
				Encoding  string
				Stdout    int `json:"stdout_bytes"`
				Stderr    int `json:"stderr_bytes"`
			} `json:"program_output"`
		}
		if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.execution-result/v1" || report.Quarantine != "COMPLETED" || report.Policy != "ALLOW" || report.Preparation != "INCOMPLETE" || report.Execution != "NOT_STARTED" || report.Exit != nil || report.Signal != nil || report.Promotion != "NOT_REQUESTED" || report.Cleanup != "NOT_STARTED" || report.Failure != "PREPARATION_FAILED" || !report.Output.Untrusted || report.Output.Encoding != "OMITTED" || report.Output.Stdout != 0 || report.Output.Stderr != 0 {
			t.Fatal("post-Policy entry refusal axes incorrect")
		}
		if strings.Contains(out.String(), "private-cli-argv") || strings.Contains(diagnostic.String(), "private-cli-argv") {
			t.Fatal("refused request leaked private argv")
		}
		staged, err := os.ReadDir(filepath.Join(cache, "heliopause", "staging"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if len(staged) != 0 {
			t.Fatal("entry refusal produced stored execution records")
		}
		t.Logf("production_npm_execution_refusal case=%d reason=%s current_policy=ALLOW preparation=INCOMPLETE runtime_attempted=false exit_absent=true output_bytes=0 stored_manifest=0", i+1, row.name)
	}
	runs, err := os.ReadDir(filepath.Join(cache, "heliopause", "evidence"))
	if err != nil || len(runs) != 2 {
		t.Fatal("fresh current Evidence runs missing", err)
	}
	for _, run := range runs {
		if !run.IsDir() || strings.HasPrefix(run.Name(), ".") {
			t.Fatal("uncommitted Evidence run retained")
		}
		files, err := os.ReadDir(filepath.Join(cache, "heliopause", "evidence", run.Name()))
		if err != nil || len(files) < 3 {
			t.Fatal("complete current Evidence batch absent", err)
		}
		for _, file := range files {
			body, err := os.ReadFile(filepath.Join(cache, "heliopause", "evidence", run.Name(), file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			var record struct {
				Run     string `json:"run_id"`
				Source  string `json:"source_id"`
				Name    string `json:"name"`
				Version string `json:"version"`
				Digest  string `json:"sha256"`
			}
			if json.Unmarshal(body, &record) != nil || record.Run != run.Name() || record.Source != "npm" || record.Name != "typescript" || record.Version != "5.9.3" || record.Digest != "10e108c9cf7d5f2879053dff18515fb405abf2ccef63eaaf017d9c571687a1d3" {
				t.Fatal("actual current evidence differs from fixed public source")
			}
		}
	}
	t.Log("production_npm_execution_refusal_evidence runs=2 source_sha256=10e108c9cf7d5f2879053dff18515fb405abf2ccef63eaaf017d9c571687a1d3 current_whole_inspection=true artifact_user_program_run=false")
}
