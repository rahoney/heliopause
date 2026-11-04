package content

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type sourceCapture struct {
	body     []byte
	overflow bool
}

func (c sourceCapture) Stdout() []byte       { return c.body }
func (sourceCapture) Stderr() []byte         { return nil }
func (c sourceCapture) StdoutOverflow() bool { return c.overflow }
func (sourceCapture) StderrOverflow() bool   { return false }

type sourceRuntime struct {
	output ports.SourceAnalysisOutput
	err    error
	calls  int
}

func (r *sourceRuntime) Analyze(context.Context, ports.SourceAnalysisInput) (ports.SourceAnalysisOutput, error) {
	r.calls++
	return r.output, r.err
}

func sourceFixture(t *testing.T) (ports.SourceAnalysisInput, ports.SourceAnalysisOutput, map[string]any) {
	t.Helper()
	rules, err := os.ReadFile("../../../tools/semgrep/rules.json")
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := sourceRuleCatalogue(rules)
	if err != nil {
		t.Fatal(err)
	}
	id := "haa-s04-js-credential-body"
	rule := catalogue[id]
	source, _ := domain.NewSourceID("npm")
	identity, err := domain.NewResolvedArtifactIdentity(source, "source-test", "1.0.0", "tarball")
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:source-test:tarball", 12, "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	input := ports.SourceAnalysisInput{Artifact: artifact, Files: []ports.SourceAnalysisFile{{Path: "private-original-name.js", Language: "javascript", Body: []byte("const secret = 'PRIVATE_SOURCE_SENTINEL';\n")}}}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := domain.NewSandboxResult(session, domain.SandboxCompleted, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	opaque := "/parser-input/source.js"
	engines := []any{}
	for _, value := range catalogue {
		engines = append(engines, []any{value.ID, "OSS"})
	}
	timing := map[string]any{"rules": []any{}, "rules_parse_time": 0, "profiling_times": map[string]any{}, "parsing_time": map[string]any{}, "scanning_time": map[string]any{}, "matching_time": map[string]any{}, "tainting_time": map[string]any{}, "fixpoint_timeouts": []any{}, "prefiltering": map[string]any{}, "targets": []any{}, "total_bytes": 0, "max_memory_bytes": 0}
	document := map[string]any{"version": runtimeidentity.SelectedSemgrepSupply().Version, "engine_requested": "OSS", "interfile_languages_used": []any{}, "profiling_results": []any{}, "skipped_rules": []any{}, "rules_by_engine": engines, "time": timing, "errors": []any{}, "paths": map[string]any{"scanned": []any{opaque}}, "results": []any{map[string]any{"check_id": id, "path": opaque, "start": map[string]any{"line": 1, "col": 1, "offset": 0}, "end": map[string]any{"line": 1, "col": 6, "offset": 5}, "extra": map[string]any{"metavars": map[string]any{"$SECRET": map[string]any{"abstract_content": "PRIVATE_METAVAR_SENTINEL"}}, "engine_kind": "OSS", "is_ignored": true, "message": rule.Message, "metadata": map[string]any{"rule_version": rule.Metadata.Version, "disposition": rule.Metadata.Disposition}, "severity": rule.Severity, "validation_state": "NO_VALIDATOR"}}}}
	body, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return input, ports.SourceAnalysisOutput{Runtime: runtime, Rules: rules, Targets: map[string]string{opaque: input.Files[0].Path}, Output: sourceCapture{body: body}}, document
}

func TestSourceReportPreservesIgnoredHitBindingsAndPrivateSource(t *testing.T) {
	input, output, _ := sourceFixture(t)
	runtime := &sourceRuntime{output: output}
	report, err := InspectSources(context.Background(), runtime, input.Artifact, input.Files)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.calls != 1 || len(report.Findings()) != 1 || report.Findings()[0].Code() != "M13_SOURCE_REVIEW" || len(report.Executions()) != 2 {
		t.Fatal("source finding/checks lost")
	}
	checks, err := SourceChecks()
	if err != nil || len(checks) != 2 {
		t.Fatal(err)
	}
	for n, check := range report.Executions() {
		if check.Status() != domain.ExecutionCompleted || check.ID() != checks[n].ID() || check.ProfileDigest() != checks[n].ProfileDigest() || check.Required() != (n == 0) {
			t.Fatal("declared binding changed", n)
		}
	}
	for _, fact := range report.Evidence() {
		if fact.Digest() != input.Artifact.Digest() || fact.Identity() != input.Artifact.Identity() {
			t.Fatal("subject binding lost")
		}
		for _, private := range []string{"PRIVATE_SOURCE_SENTINEL", "PRIVATE_METAVAR_SENTINEL", input.Files[0].Path, "metavars", "abstract_content"} {
			if strings.Contains(fact.Summary(), private) {
				t.Fatal("private data persisted", private)
			}
		}
	}
}

func TestSourcePartialCoverageRetainsCompletedSignals(t *testing.T) {
	for _, fault := range []string{"parse-error", "skipped-target", "skipped-rule", "fixpoint-timeout", "missing-target"} {
		t.Run(fault, func(t *testing.T) {
			input, output, document := sourceFixture(t)
			switch fault {
			case "parse-error":
				document["errors"] = []any{map[string]any{"error_type": "Timeout"}}
			case "skipped-target":
				document["paths"].(map[string]any)["skipped"] = []any{map[string]any{"path": "/parser-input/skipped.js"}}
			case "skipped-rule":
				document["skipped_rules"] = []any{map[string]any{"rule_id": "unavailable"}}
			case "fixpoint-timeout":
				document["time"].(map[string]any)["fixpoint_timeouts"] = []any{map[string]any{"rule_id": "timed-out"}}
			case "missing-target":
				input.Files = append(input.Files, ports.SourceAnalysisFile{Path: "second.js", Language: "javascript", Body: []byte("const x = 1;\n")})
				output.Targets["/parser-input/second.js"] = "second.js"
			}
			body, _ := json.Marshal(document)
			output.Output = sourceCapture{body: body}
			report, err := InspectSources(context.Background(), &sourceRuntime{output: output}, input.Artifact, input.Files)
			if err != nil || len(report.Findings()) != 1 || len(report.Executions()) != 2 || report.Executions()[0].Status() != domain.ExecutionIncomplete || report.Executions()[1].Status() != domain.ExecutionCompleted {
				t.Fatal("partial source facts or required coverage lost", err)
			}
		})
	}
}

func TestSourceOutputRefusesForgedIncompleteAndAmbiguousDocuments(t *testing.T) {
	for _, fault := range []string{"version", "engine", "no-rules", "duplicate-rule", "unknown-rule", "missing-scanned", "duplicate-scanned", "foreign-path", "position", "column", "language", "unsupported-language", "unsupported-encoding", "rule-version", "disposition", "severity", "message", "missing-extra", "null-errors", "missing-timeout-field", "duplicate-match", "map-mismatch", "rules-bytes", "runtime-incomplete", "overflow", "duplicate-key", "folded-key", "trailing", "invalid-utf8", "depth"} {
		t.Run(fault, func(t *testing.T) {
			input, output, document := sourceFixture(t)
			result := document["results"].([]any)[0].(map[string]any)
			extra := result["extra"].(map[string]any)
			switch fault {
			case "version":
				document["version"] = "unknown"
			case "engine":
				document["engine_requested"] = "PRO"
			case "no-rules":
				document["rules_by_engine"] = []any{}
			case "duplicate-rule":
				rows := document["rules_by_engine"].([]any)
				rows[1] = rows[0]
			case "unknown-rule":
				result["check_id"] = "artifact-provided-rule"
			case "missing-scanned":
				delete(document["paths"].(map[string]any), "scanned")
			case "duplicate-scanned":
				document["paths"].(map[string]any)["scanned"] = []any{"/parser-input/source.js", "/parser-input/source.js"}
			case "foreign-path":
				result["path"] = "/etc/passwd"
			case "position":
				result["end"].(map[string]any)["offset"] = 99999
			case "column":
				result["end"].(map[string]any)["col"] = 5
			case "language":
				input.Files[0].Language = "rust"
			case "unsupported-language":
				input.Files[0].Language = "unsupported"
				document["results"] = []any{}
			case "unsupported-encoding":
				input.Files[0].Body = []byte{0xff}
				document["results"] = []any{}
			case "rule-version":
				extra["metadata"].(map[string]any)["rule_version"] = 42
			case "disposition":
				extra["metadata"].(map[string]any)["disposition"] = "safe"
			case "severity":
				extra["severity"] = "INFO"
			case "message":
				extra["message"] = "artifact override"
			case "missing-extra":
				delete(extra, "is_ignored")
			case "null-errors":
				document["errors"] = nil
			case "missing-timeout-field":
				delete(document["time"].(map[string]any), "fixpoint_timeouts")
			case "duplicate-match":
				document["results"] = []any{result, result}
			case "map-mismatch":
				output.Targets["/parser-input/source.js"] = "unknown.js"
			case "rules-bytes":
				output.Rules = append([]byte{}, output.Rules...)
				output.Rules[0] ^= 1
			case "runtime-incomplete":
				session, _ := domain.NewSandboxSessionID()
				output.Runtime, _ = domain.NewSandboxResult(session, domain.SandboxIncomplete, "TEST_INCOMPLETE", nil)
			}
			body, _ := json.Marshal(document)
			switch fault {
			case "duplicate-key":
				body = append([]byte(`{"version":"ignored",`), body[1:]...)
			case "folded-key":
				body = append([]byte(`{"Version":"ignored",`), body[1:]...)
			case "trailing":
				body = append(body, []byte(` {}`)...)
			case "invalid-utf8":
				body = append(body, 0xff)
			case "depth":
				body = []byte(strings.Repeat("[", 40) + strings.Repeat("]", 40))
			}
			output.Output = sourceCapture{body: body, overflow: fault == "overflow"}
			report, err := InspectSources(context.Background(), &sourceRuntime{output: output}, input.Artifact, input.Files)
			if err == nil || len(report.Findings()) != 0 || len(report.Executions()) != 2 || report.Executions()[0].Status() != domain.ExecutionIncomplete {
				t.Fatal("unqualified source output became a completed verdict", err)
			}
		})
	}
}

func TestSourceRuntimeFailurePreservesCauseAndNeverInventsEmptySuccess(t *testing.T) {
	input, output, _ := sourceFixture(t)
	cause := errors.New("test source runtime failure")
	report, err := InspectSources(context.Background(), &sourceRuntime{output: output, err: cause}, input.Artifact, input.Files)
	if !errors.Is(err, cause) || len(report.Findings()) != 0 || report.Executions()[0].Status() != domain.ExecutionIncomplete {
		t.Fatal("first runtime failure lost", err)
	}
	runtime := &sourceRuntime{output: output}
	report, err = InspectSources(context.Background(), runtime, input.Artifact, nil)
	if err == nil || runtime.calls != 0 || report.Executions()[0].Status() != domain.ExecutionIncomplete {
		t.Fatal("no targets became a clean scan")
	}
}

func TestSourceNoMatchAndSignalLimitKeepDistinctCoverage(t *testing.T) {
	input, output, document := sourceFixture(t)
	document["results"] = []any{}
	body, _ := json.Marshal(document)
	output.Output = sourceCapture{body: body}
	report, err := InspectSources(context.Background(), &sourceRuntime{output: output}, input.Artifact, input.Files)
	if err != nil || len(report.Findings()) != 0 || report.Executions()[0].Status() != domain.ExecutionCompleted {
		t.Fatal("complete no-match scan lost", err)
	}
	input, output, document = sourceFixture(t)
	input.Files[0].Body = []byte(strings.Repeat("x", sourceSignalLimit+2))
	template := document["results"].([]any)[0]
	raw, _ := json.Marshal(template)
	results := []any{}
	for index := 0; index <= sourceSignalLimit; index++ {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			t.Fatal(err)
		}
		row["start"] = map[string]any{"line": 1, "col": index + 1, "offset": index}
		row["end"] = map[string]any{"line": 1, "col": index + 2, "offset": index + 1}
		results = append(results, row)
	}
	document["results"] = results
	body, _ = json.Marshal(document)
	output.Output = sourceCapture{body: body}
	report, err = InspectSources(context.Background(), &sourceRuntime{output: output}, input.Artifact, input.Files)
	if err != nil || len(report.Findings()) != sourceSignalLimit || report.Executions()[0].Status() != domain.ExecutionIncomplete || report.Executions()[1].Status() != domain.ExecutionCompleted {
		t.Fatal("signal bound became clean or discarded partial facts", err)
	}
}

func TestSourceAllSelectedLanguagesShareCanonicalRuleAliases(t *testing.T) {
	input, output, document := sourceFixture(t)
	input.Files = nil
	output.Targets = map[string]string{}
	paths := []any{}
	for index, language := range []string{"javascript", "typescript", "python", "go", "rust", "hcl"} {
		name := language + "-fixture"
		opaque := strings.Repeat("/source", index+1)
		input.Files = append(input.Files, ports.SourceAnalysisFile{Path: name, Language: language, Body: []byte("fixture\n")})
		output.Targets[opaque] = name
		paths = append(paths, opaque)
	}
	document["results"] = []any{}
	document["paths"] = map[string]any{"scanned": paths}
	body, _ := json.Marshal(document)
	output.Output = sourceCapture{body: body}
	report, err := InspectSources(context.Background(), &sourceRuntime{output: output}, input.Artifact, input.Files)
	if err != nil || len(report.Findings()) != 0 || report.Execution().Status() != domain.ExecutionCompleted {
		t.Fatal("selected language alias rejected", err)
	}
}

func TestSourceOptionalDataflowExplanationNeverPersists(t *testing.T) {
	input, output, document := sourceFixture(t)
	extra := document["results"].([]any)[0].(map[string]any)["extra"].(map[string]any)
	extra["dataflow_trace"] = map[string]any{"taint_source": "PRIVATE_TRACE_SENTINEL", "intermediate_vars": []any{}}
	body, _ := json.Marshal(document)
	output.Output = sourceCapture{body: body}
	report, err := InspectSources(context.Background(), &sourceRuntime{output: output}, input.Artifact, input.Files)
	if err != nil || len(report.Findings()) != 1 || report.Execution().Status() != domain.ExecutionCompleted {
		t.Fatal("valid optional explanation rejected", err)
	}
	for _, fact := range report.Evidence() {
		if strings.Contains(fact.Summary(), "PRIVATE_TRACE_SENTINEL") || strings.Contains(fact.Summary(), "dataflow_trace") {
			t.Fatal("source explanation persisted")
		}
	}
	extra["dataflow_trace"] = true
	body, _ = json.Marshal(document)
	output.Output = sourceCapture{body: body}
	if report, err := InspectSources(context.Background(), &sourceRuntime{output: output}, input.Artifact, input.Files); err == nil || len(report.Findings()) != 0 {
		t.Fatal("malformed explanation accepted")
	}
}
