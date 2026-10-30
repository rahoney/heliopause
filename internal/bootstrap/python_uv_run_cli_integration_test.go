//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestLinuxUVRunExecutionCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_RUN_EXECUTION_CLI_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user actual uvx/native tool-run CLI qualification")
	}
	runPythonNativeExecutionCLICases(t, false)
}

func runPythonNativeExecutionCLICases(t *testing.T, pipx bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	cache, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", cache)
	staged := filepath.Join(cache, "heliopause", "staging")
	readNames := func() map[string]bool {
		result := map[string]bool{}
		entries, err := os.ReadDir(staged)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				result[entry.Name()] = true
			}
		}
		return result
	}
	seenOperations, seenRuns := map[string]bool{}, map[string]bool{}
	type nativeRunCLICase struct {
		path, tail                []string
		language, output, command string
		show                      bool
		code, nodes               int
		refused                   bool
	}
	rows := []nativeRunCLICase{
		{[]string{"uvx"}, []string{"cowsay@6.1", "--", "-t", "private-native-cli-body"}, "en", "json", "cowsay", false, 0, 1, false},
		{[]string{"uv", "tool", "run"}, []string{"cowsay==6.1", "--", "--haa-invalid-option"}, "ko", "json", "cowsay", true, 2, 1, false},
		{[]string{"uvx"}, []string{"cowsay@6.1", "--", "--help"}, "ko", "human", "cowsay", true, 0, 1, false},
		{[]string{"uv", "tool", "run"}, []string{"--from", "python-slugify==8.0.4", "slugify", "--", "taking", "input", "from", "the", "command", "line"}, "en", "json", "slugify", true, 0, 2, false},
		{[]string{"uvx"}, []string{"--from", "cowsay@6.1", "haa-missing-console", "--", "private-refusal-argv"}, "en", "json", "haa-missing-console", false, 1, 0, true},
	}
	runtimeIdentity := "uv-tool-run-isolated:" + sandbox.UVResolverRuntimeIdentity()
	marker, sourceMarker := "production_uv_run_cli", "production_uv_run_cli_source"
	if pipx {
		runtimeIdentity = "pipx-run-isolated:" + sandbox.PipxResolverRuntimeIdentity()
		marker, sourceMarker = "production_pipx_run_cli", "production_pipx_run_cli_source"
		for i := range rows {
			rows[i].path = []string{"pipx", "run"}
			for j := range rows[i].tail {
				rows[i].tail[j] = strings.ReplaceAll(rows[i].tail[j], "--from", "--spec")
			}
		}
		sole := rows[3]
		sole.tail = []string{"python-slugify==8.0.4", "--", "taking", "input", "from", "the", "command", "line"}
		rows = append(rows, sole)
	}
	if !pipx {
		rows = append(rows,
			nativeRunCLICase{[]string{"uvx"}, []string{"--from", "python-slugify[unidecode]==8.0.4", "slugify", "--", "taking", "input", "from", "the", "command", "line"}, "en", "json", "slugify", true, 0, 3, false},
		)
		nonzero := rows[len(rows)-1]
		nonzero.path, nonzero.tail, nonzero.code, nonzero.language = []string{"uv", "tool", "run"}, []string{"--from", "python-slugify[unidecode]@8.0.4", "slugify", "--", "--haa-invalid-option"}, 2, "ko"
		rows = append(rows, nonzero)
		unknown := nonzero
		unknown.path, unknown.tail, unknown.code, unknown.nodes, unknown.refused, unknown.show, unknown.language = []string{"uvx"}, []string{"--from", "python-slugify[haa-unknown-extra]==8.0.4", "slugify"}, 1, 0, true, false, "en"
		rows = append(rows, unknown)
	}
	expectedOperations, expectedRuns := 0, 0
	for i, row := range rows {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			if !row.refused {
				expectedOperations++
				expectedRuns += row.nodes
			}
			before := readNames()
			args := append([]string{"--lang", row.language, "--output", row.output, "--progress", "never", "--color", "never"}, row.path...)
			if row.show {
				args = append(args, "--show-output")
			}
			args = append(args, row.tail...)
			var out, diagnostic bytes.Buffer
			cause := Run(ctx, args, &out, &diagnostic)
			if row.code == 0 && cause != nil {
				var failure struct {
					Quarantine, Policy, Preparation, Execution string
					Failure                                    string `json:"failure_code"`
				}
				_ = json.Unmarshal(out.Bytes(), &failure)
				t.Fatalf("actual native CLI zero: %v; axes=%+v", cause, failure)
			}
			if row.code != 0 {
				var code interface{ ExitCode() int }
				var program *sandbox.PythonProgramError
				if !errors.As(cause, &code) || code.ExitCode() != row.code || errors.As(cause, &program) == row.refused {
					t.Fatal("native guest/preparation status differs", cause)
				}
			}
			if row.output == "json" {
				var report struct {
					Schema, Quarantine, Policy, Preparation, Execution string
					ExitCode                                           *int   `json:"exit_code"`
					Signal                                             *int   `json:"signal"`
					Promotion                                          string `json:"output_promotion"`
					Cleanup                                            string `json:"execution_guest_cleanup"`
					Failure                                            string `json:"failure_code"`
					Output                                             struct {
						Untrusted   bool
						Encoding    string
						StdoutBytes int    `json:"stdout_bytes"`
						StderrBytes int    `json:"stderr_bytes"`
						Stdout      string `json:"stdout_escaped"`
						Stderr      string `json:"stderr_escaped"`
					} `json:"program_output"`
				}
				if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.execution-result/v1" || !report.Output.Untrusted || report.Promotion != "NOT_REQUESTED" {
					t.Fatal("actual separate native report missing")
				}
				if row.refused {
					if report.Quarantine != "INCOMPLETE" || report.Policy != "NOT_AVAILABLE" || report.Preparation != "INCOMPLETE" || report.Execution != "NOT_STARTED" || report.ExitCode != nil || report.Signal != nil || report.Cleanup != "NOT_STARTED" || report.Failure != "PREPARATION_FAILED" || report.Output.StdoutBytes != 0 || report.Output.StderrBytes != 0 {
						t.Fatal("native source-console refusal became approval/program result")
					}
				} else if report.Quarantine != "COMPLETED" || report.Policy != "ALLOW" || report.Preparation != "COMPLETED" || report.Execution != "COMPLETED" || report.ExitCode == nil || *report.ExitCode != row.code || report.Cleanup != "CONFIRMED" || row.code == 0 && report.Output.StdoutBytes == 0 || row.code != 0 && report.Output.StderrBytes == 0 {
					t.Fatal("current native CLI result axes differ")
				}
				if row.show != (report.Output.Encoding == "ASCII_QUOTED") || !row.show && (report.Output.Stdout != "" || report.Output.Stderr != "") {
					t.Fatal("private output or explicit ASCII choice lost")
				}
				if (i == 3 || !pipx && i == 5) && !strings.Contains(report.Output.Stdout, "taking-input-from-the-command-line") {
					t.Fatal("different package/console public program result missing")
				}
			} else if json.Valid(out.Bytes()) || !strings.Contains(out.String(), "CONFIRMED") || !strings.Contains(out.String(), "NOT_REQUESTED") || !strings.Contains(out.String(), "검역") || !strings.Contains(out.String(), "ASCII_QUOTED") || !strings.Contains(out.String(), "usage:") {
				t.Fatal("literal guest help became HAA help or lost human/private axes")
			}
			if strings.Contains(out.String(), "private-native-cli-body") || strings.Contains(diagnostic.String(), "private-native-cli-body") || strings.Contains(out.String(), "private-refusal-argv") || strings.Contains(diagnostic.String(), "private-refusal-argv") {
				t.Fatal("private native argv/default output leaked")
			}
			after := readNames()
			if row.refused {
				if len(after) != len(before) {
					t.Fatal("refusal created execution records")
				}
				t.Logf(marker+" case=%d alias=%s format=%s language=%s show_output=%t exit_code=%d current_policy=NOT_AVAILABLE runtime_attempted=false stored_manifest=0", i+1, row.path[len(row.path)-1], row.output, row.language, row.show, row.code)
				return
			}
			if len(after) != len(before)+1 {
				t.Fatal("current CLI reused/missed records")
			}
			for name := range after {
				if before[name] {
					continue
				}
				document, err := os.ReadFile(filepath.Join(staged, name, "manifest.json"))
				if err != nil {
					t.Fatal(err)
				}
				var manifest struct {
					Operation string `json:"operation_id"`
					Kind      string `json:"schema"`
					Resolver  struct {
						Runtime string `json:"runtime_identity"`
					} `json:"resolver"`
					Policy    struct{ Decision string } `json:"policy"`
					Execution struct{ Command string }  `json:"execution"`
					Entries   []struct {
						Source, Name, Version, SHA256 string
						Run                           string `json:"run_id"`
						Evidence                      []string
					} `json:"entries"`
				}
				if json.Unmarshal(document, &manifest) != nil || manifest.Kind != "helox.verified-execution-manifest/v1" || manifest.Resolver.Runtime != runtimeIdentity || manifest.Policy.Decision != "ALLOW" || manifest.Execution.Command != row.command || len(manifest.Entries) != row.nodes || seenOperations[manifest.Operation] || strings.Contains(string(document), "private-native-cli-body") {
					t.Fatal("current native source/console/records differ")
				}
				seenOperations[manifest.Operation] = true
				names := map[string]bool{}
				for _, entry := range manifest.Entries {
					if entry.Source != "pypi" || len(entry.SHA256) != 64 || len(entry.Evidence) < 3 || entry.Run == "" || seenRuns[entry.Run] {
						t.Fatal("fresh current whole source/Evidence missing")
					}
					seenRuns[entry.Run] = true
					names[entry.Name] = true
					t.Logf(sourceMarker+" case=%d name=%s version=%s sha256=%s evidence=%d", i+1, entry.Name, entry.Version, entry.SHA256, len(entry.Evidence))
				}
				if !pipx && (i == 5 || i == 6) && (!names["python-slugify"] || !names["text-unidecode"] || !names["unidecode"]) {
					t.Fatal("requested root extra did not receive whole three-wheel approval")
				}
				if (i == 3 || pipx && i == 5) && (!names["python-slugify"] || !names["text-unidecode"]) {
					t.Fatal("independent different package/console closure missing")
				}
			}
			t.Logf(marker+" case=%d alias=%s format=%s language=%s show_output=%t exit_code=%d current_policy=ALLOW nodes=%d guest_cleanup=CONFIRMED", i+1, row.path[len(row.path)-1], row.output, row.language, row.show, row.code, row.nodes)
		})
	}
	if len(readNames()) != expectedOperations || len(seenOperations) != expectedOperations || len(seenRuns) != expectedRuns {
		t.Fatal("current aliases/whole two-node request records missing")
	}
}
