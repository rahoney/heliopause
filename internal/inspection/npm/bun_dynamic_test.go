package npm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestBunPartialFactsDoNotCompleteRequiredLifecycle(t *testing.T) {
	root := canonicalRoot(t)
	subject := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}})
	for _, status := range []domain.SandboxStatus{domain.SandboxCompleted, domain.SandboxIncomplete} {
		session, err := domain.NewSandboxSessionID()
		if err != nil {
			t.Fatal(err)
		}
		code := ""
		if status == domain.SandboxIncomplete {
			code = "M3_DYNAMIC_EXECUTION_FAILED"
		}
		fact, err := domain.NewCountedSandboxObservation(domain.ObservationProcess, "process-exec-unexpected", 2)
		if err != nil {
			t.Fatal(err)
		}
		result, err := domain.NewSandboxResult(session, status, code, []domain.SandboxObservation{fact})
		if err != nil {
			t.Fatal(err)
		}
		// Structural normalization only; actual layout binding is exercised by the
		// project/realcorpus consumer, not this zero-layout interpretation fixture.
		facts, err := bunInstanceFacts(subject, "tiny", artifactnpm.BunInspectionLayout{}, result)
		if err != nil || len(facts.Findings()) != 1 || facts.Findings()[0].Code() != "M3_UNEXPECTED_PROCESS" {
			t.Fatal("partial suspicious facts discarded", err)
		}
		if !strings.Contains(facts.Evidence()[0].Summary(), session.String()) || !strings.Contains(facts.Evidence()[0].Summary(), `"PROCESS:process-exec-unexpected":2`) {
			t.Fatal("actual session/count lost")
		}
		coverage, err := bunCoverageReport(subject, 2, 1, 0, true)
		if err != nil {
			t.Fatal(err)
		}
		combined, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{facts, coverage})
		if err != nil {
			t.Fatal(err)
		}
		if len(combined.Findings()) != 1 || combined.Executions()[1].Status() != domain.ExecutionIncomplete || !combined.Executions()[1].Required() {
			t.Fatal("facts completed required lifecycle")
		}
	}
}
func TestBunCoverageNeverInventsSessionOrCompletesMissingPeer(t *testing.T) {
	root := canonicalRoot(t)
	subject := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}})
	for _, test := range []struct {
		expected, attempted, completed int
		failed                         bool
		want                           domain.ExecutionStatus
	}{{2, 0, 0, false, domain.ExecutionNotExecuted}, {2, 1, 1, false, domain.ExecutionIncomplete}, {2, 2, 2, false, domain.ExecutionCompleted}, {2, 2, 2, true, domain.ExecutionIncomplete}} {
		report, err := bunCoverageReport(subject, test.expected, test.attempted, test.completed, test.failed)
		if err != nil {
			t.Fatal(err)
		}
		if report.Execution().Status() != test.want || !report.Execution().Required() || strings.Contains(report.Evidence()[0].Summary(), "sbx_") {
			t.Fatal("invalid coverage", report)
		}
	}
	if _, err := bunInstanceFacts(subject, "tiny", artifactnpm.BunInspectionLayout{}, domain.SandboxResult{}); err == nil {
		t.Fatal("zero result made observation evidence")
	}
	var zero *BunProjectInspector
	if _, err := zero.InspectProject(context.Background(), domain.ProjectDependencySnapshot{}, nil); err == nil {
		t.Fatal("zero project inspector accepted")
	}
}

type bunSequenceExecutor struct {
	after   func()
	results []domain.SandboxResult
	cause   error
	calls   []string
}

func (f *bunSequenceExecutor) ExecuteBunRuntimeInstance(_ context.Context, _ artifactnpm.BunInspectionLayout, _ domain.AcquiredArtifact, key string) (domain.SandboxResult, error) {
	f.calls = append(f.calls, key)
	if f.after != nil {
		f.after()
	}
	index := len(f.calls) - 1
	if index >= len(f.results) {
		return domain.SandboxResult{}, f.cause
	}
	if f.cause != nil && index == len(f.results)-1 {
		return f.results[index], f.cause
	}
	return f.results[index], nil
}
func TestBunActualResultAggregationRejectsReusedMissingAndFailedSessions(t *testing.T) {
	root := canonicalRoot(t)
	subject := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}})
	first, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	one, err := domain.NewSandboxResult(first, domain.SandboxCompleted, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	two, err := domain.NewSandboxResult(second, domain.SandboxCompleted, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	suspicious, err := domain.NewSandboxObservation(domain.ObservationNetwork, "network-attempt")
	if err != nil {
		t.Fatal(err)
	}
	partial, err := domain.NewSandboxResult(second, domain.SandboxIncomplete, "M3_DYNAMIC_EXECUTION_FAILED", []domain.SandboxObservation{suspicious})
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("controlled original execution cause")
	for _, test := range []struct {
		name    string
		results []domain.SandboxResult
		cause   error
		want    int
	}{{"all", []domain.SandboxResult{one, two}, nil, 2}, {"reused", []domain.SandboxResult{one, one}, nil, 1}, {"missing", []domain.SandboxResult{one, {}}, nil, 1}, {"partial", []domain.SandboxResult{one, partial}, cause, 1}} {
		t.Run(test.name, func(t *testing.T) {
			executor := &bunSequenceExecutor{results: test.results, cause: test.cause}
			inspector := &BunProjectInspector{execute: executor}
			states := []bunInspectionState{{subject: subject, instances: []artifactnpm.BunLockedPackage{{Key: "tiny"}, {Key: "parent/tiny"}}}}
			// Structural aggregation fixture: runtime execution/layout binding is not
			// claimed by these result fakes; actual project consumer validates both.
			err := inspector.observeInstances(context.Background(), artifactnpm.BunInspectionLayout{}, states)
			if (err == nil) != (test.name == "all") || states[0].completed != test.want || len(executor.calls) != 2 || executor.calls[0] == executor.calls[1] {
				t.Fatal("peer/session coverage collapsed", err)
			}
			if test.cause != nil && !errors.Is(err, test.cause) {
				t.Fatal("first cause lost", err)
			}
			if test.name == "partial" && len(states[0].leaves[1].Findings()) != 1 {
				t.Fatal("failed instance facts lost")
			}
		})
	}
}

func TestBunInvalidFullSummaryStillRetainsSuspiciousFacts(t *testing.T) {
	root := canonicalRoot(t)
	subject := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}})
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	observations := []domain.SandboxObservation{}
	for index := 0; index <= domain.MaximumObservationSummaryUniqueSubjects; index++ {
		fact, err := domain.NewSandboxObservation(domain.ObservationProcess, fmt.Sprintf("normalized-fact-%d", index))
		if err != nil {
			t.Fatal(err)
		}
		observations = append(observations, fact)
	}
	suspicious, err := domain.NewSandboxObservation(domain.ObservationHoneytoken, "honeytoken-access")
	if err != nil {
		t.Fatal(err)
	}
	observations = append(observations, suspicious)
	result, err := domain.NewSandboxResult(session, domain.SandboxIncomplete, "M3_DYNAMIC_EXECUTION_FAILED", observations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := result.ObservationSummary(); err == nil {
		t.Fatal("invalid test summary")
	}
	report, err := bunInstanceFacts(subject, "tiny", artifactnpm.BunInspectionLayout{}, result)
	if err == nil || len(report.Findings()) != 1 || report.Findings()[0].Code() != "M3_HONEYTOKEN_ACCESS" || !strings.Contains(report.Evidence()[0].Summary(), `"summary_incomplete":true`) {
		t.Fatal("summary failure erased security facts", err)
	}
}

func TestBunCancellationDuringFinalResultCannotCompleteCoverage(t *testing.T) {
	root := canonicalRoot(t)
	subject := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}})
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	result, err := domain.NewSandboxResult(session, domain.SandboxCompleted, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("controlled final-result cancellation")
	executor := &bunSequenceExecutor{results: []domain.SandboxResult{result}, after: func() { cancel(cause) }}
	inspector := &BunProjectInspector{execute: executor}
	states := []bunInspectionState{{subject: subject, instances: []artifactnpm.BunLockedPackage{{Key: "tiny"}}}}
	if err := inspector.observeInstances(ctx, artifactnpm.BunInspectionLayout{}, states); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || states[0].completed != 0 || len(states[0].leaves) != 1 {
		t.Fatal("final cancellation lost coverage/facts/cause", err)
	}
}
