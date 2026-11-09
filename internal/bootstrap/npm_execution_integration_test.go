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

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/content"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
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

// nodeExecutionFailureSummary reports inspection axes and per-node gaps for a
// failed CLI case. It parses report structure only and never echoes the raw
// result, so program output cannot reach the failure message.
func nodeExecutionFailureSummary(body []byte) string {
	const maxSummary = 2048
	bounded := func(text string) string { return text[:min(len(text), maxSummary)] }
	var report struct {
		Schema, Quarantine, Policy, Preparation, Execution string
		Cleanup                                            string `json:"execution_guest_cleanup"`
		Checks                                             []struct {
			ID, Node string
			Required bool
			Status   string `json:"execution_status"`
		} `json:"checks"`
	}
	if json.Unmarshal(body, &report) == nil {
		var missing []string
		for _, check := range report.Checks {
			if check.Required && check.Status != "COMPLETED" {
				missing = append(missing, check.Node+"/"+check.ID+"="+check.Status)
			}
		}
		return bounded(fmt.Sprintf("result_bytes=%d schema=%s quarantine=%s policy=%s preparation=%s execution=%s cleanup=%s checks=%d required_gaps=%v", len(body), report.Schema, report.Quarantine, report.Policy, report.Preparation, report.Execution, report.Cleanup, len(report.Checks), missing))
	}
	var nodes []string
	gaps := map[string][]string{}
	current := ""
	for _, line := range strings.Split(string(body), "\n") {
		for _, header := range []string{"Partial inspection: ", "중단 시점 검사: "} {
			if name, found := strings.CutPrefix(line, header); found {
				current = name
				nodes = append(nodes, name)
			}
		}
		check, found := strings.CutPrefix(line, "  ! ")
		if !found || current == "" || strings.Contains(check, "[선택]") || strings.Contains(check, "[optional]") {
			continue
		}
		if id, _, ok := strings.Cut(check, " "); ok {
			gaps[current] = append(gaps[current], id)
		}
	}
	last, completed, parts := "", 0, []string{}
	for _, node := range nodes {
		last = node
		if len(gaps[node]) == 1 && gaps[node][0] == "dependency-advisory" {
			completed++
		}
		parts = append(parts, node+":"+strings.Join(gaps[node], ","))
	}
	return bounded(fmt.Sprintf("result_bytes=%d partial_nodes=%d nodes_missing_only_graph_advisory=%d last_node=%s node_gaps=%v", len(body), len(nodes), completed, last, parts))
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
		started := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		cause := Run(ctx, args, &out, &diagnostic)
		elapsed := time.Since(started)
		contextErr, contextCause := ctx.Err(), context.Cause(ctx)
		cancel()
		t.Logf("node_execution_cli_timing client=%s case=%d seconds=%.3f context_err=%v context_cause=%v deadline_expired=%t run_failed=%t run_error_deadline=%t", client, i+1, elapsed.Seconds(), contextErr, contextCause, errors.Is(contextErr, context.DeadlineExceeded), cause != nil, errors.Is(cause, context.DeadlineExceeded))
		if row.code == 0 && cause != nil {
			t.Fatalf("actual CLI zero failed: %v; %s; diagnostic=%q", cause, nodeExecutionFailureSummary(out.Bytes()), diagnostic.Bytes()[:min(diagnostic.Len(), 2048)])
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
				t.Fatal("actual guest nonzero lost typed cause or process exit", cause, nodeExecutionFailureSummary(out.Bytes()))
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
				Checks      []struct {
					ID, Node string
					Required bool
					Status   string `json:"execution_status"`
					Profile  string `json:"profile_sha256"`
				} `json:"checks"`
				Output struct {
					Untrusted   bool
					Encoding    string
					StdoutBytes int    `json:"stdout_bytes"`
					Stdout      string `json:"stdout_escaped"`
				} `json:"program_output"`
			}
			if json.Unmarshal(out.Bytes(), &report) != nil || report.Schema != "helox.execution-result/v2" || report.Quarantine != "COMPLETED" || report.Policy != "ALLOW" || report.Preparation != "COMPLETED" || report.Execution != "COMPLETED" || report.ExitCode == nil || *report.ExitCode != row.code || report.Promotion != "NOT_REQUESTED" || report.Cleanup != "CONFIRMED" || !report.Output.Untrusted || report.Output.StdoutBytes == 0 {
				t.Fatal("actual separate execution JSON incomplete")
			}
			declarations, err := content.ArchiveEvidenceRequirements()
			if err != nil {
				t.Fatal(err)
			}
			expected := map[string]string{}
			for _, declaration := range declarations {
				if declaration.Check.Required() {
					expected[declaration.Check.ID().String()] = declaration.Check.ProfileDigest().String()
				}
			}
			seen := map[string]map[string]bool{}
			for _, check := range report.Checks {
				if check.Required && check.Status != "COMPLETED" {
					t.Fatal("CLI lost completed required coverage", check.ID)
				}
				profile, known := expected[check.ID]
				if !known {
					continue
				}
				if check.Node == "" || !check.Required || check.Status != "COMPLETED" || check.Profile != profile {
					t.Fatal("CLI lost current required source/native/census", check.ID)
				}
				if seen[check.Node] == nil {
					seen[check.Node] = map[string]bool{}
				}
				if seen[check.Node][check.ID] {
					t.Fatal("CLI duplicated a current content check")
				}
				seen[check.Node][check.ID] = true
			}
			if len(seen) == 0 {
				t.Fatal("CLI omitted its current content checks")
			}
			for _, checks := range seen {
				if len(checks) != len(expected) {
					t.Fatal("CLI omitted source/native/census for a selected node")
				}
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

// The exact public CLI fixture must also pass its configured source boundary.
// This isolates content failure before SDK selection, command preparation or run.
func TestNPMExecutionSourceContentIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_EXECUTION_SOURCE_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user current npm source qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ports.SourceAnalysisWallSeconds+60)*time.Second)
	defer cancel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	intake := filepath.Join(root, "intake")
	source, err := artifactnpm.NewPublicResolver(intake)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := domain.NewSourceID("npm")
	fixture := typescriptExecutionFixture
	identity, err := domain.NewResolvedArtifactIdentity(sourceID, fixture.name, fixture.version, "tarball")
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the qualified exact URL/integrity fixture. Whole-package metadata
	// discovery is a separate boundary and is unnecessary for source readback.
	resolved, err := domain.NewResolvedArtifact(identity, "https://registry.npmjs.org/"+fixture.name+"/-/"+fixture.name+"-"+fixture.version+".tgz", fixture.integrity)
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := source.Acquire(ctx, run, resolved)
	if err != nil || artifact.Digest().String() != "10e108c9cf7d5f2879053dff18515fb405abf2ccef63eaaf017d9c571687a1d3" {
		t.Fatal("exact public TypeScript bytes unavailable", err)
	}
	executor, observer := pythonRequirementsIntegrationRuntime(t, ctx)
	analyzer, err := sandbox.NewSourceAnalyzer(executor, observer.Observer(), newObservationResourceAdapter())
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectionnpm.NewInspector(intake, sourceParserDiagnostic{analyzer, t})
	if err != nil {
		t.Fatal(err)
	}
	inventory, files, report, err := inspector.InspectArchiveContent(ctx, artifact)
	for _, check := range report.Executions() {
		limitation, _ := check.LimitationCode()
		t.Logf("typescript_content_check id=%s required=%t status=%s limitation=%s", check.ID(), check.Required(), check.Status(), limitation)
	}
	if err != nil || !inventory.Matches(artifact) || len(files) == 0 || len(report.Findings()) != 0 {
		t.Fatal("exact TypeScript content failed", err)
	}
	for _, check := range report.Executions() {
		if check.Required() && check.Status() != domain.ExecutionCompleted {
			t.Fatal("TypeScript content remained incomplete", check.ID())
		}
	}
	t.Logf("typescript_current_source targets=%d check_count=%d artifact_sha256=%s", len(files), len(report.Executions()), artifact.Digest())
}
