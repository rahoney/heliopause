package npm

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCompositeInspectorPreservesBothRequiredChecks(t *testing.T) {
	artifact := dynamicArtifact(t)
	static := completedReport(t, artifact, "npm-static-archive", "npm-static-evidence")
	dynamic := completedReport(t, artifact, "npm-dynamic-lifecycle", "npm-dynamic-evidence")
	inspector, err := NewCompositeInspector(reportInspector{report: static}, reportInspector{report: dynamic})
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspector.Inspect(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Executions()) != 2 || len(report.Evidence()) != 2 {
		t.Fatalf("report checks/evidence = %d/%d", len(report.Executions()), len(report.Evidence()))
	}
}

type reportInspector struct {
	report domain.InspectionReport
	err    error
}

func (i reportInspector) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.report, i.err
}

func completedReport(t *testing.T, artifact domain.AcquiredArtifact, check, evidenceName string) domain.InspectionReport {
	t.Helper()
	checkID, _ := domain.NewCheckID(check)
	execution, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID(evidenceName)
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "inspection", "completed inspection")
	if err != nil {
		t.Fatal(err)
	}
	report, err := domain.NewInspectionReport(execution, nil, []domain.Evidence{evidence})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestCompositeInspectorRetainsEarlierReportsOnFailure(t *testing.T) {
	artifact := dynamicArtifact(t)
	static := completedReport(t, artifact, "npm-static-archive", "npm-static-evidence")
	cause := errors.New("dynamic provider failed")
	for _, partial := range []bool{false, true} {
		dynamic := domain.InspectionReport{}
		if partial {
			dynamic = completedReport(t, artifact, "npm-dynamic-lifecycle", "npm-dynamic-evidence")
		}
		inspector, err := NewCompositeInspector(reportInspector{report: static}, reportInspector{report: dynamic, err: cause})
		if err != nil {
			t.Fatal(err)
		}
		report, err := inspector.Inspect(context.Background(), artifact)
		want := 1
		if partial {
			want = 2
		}
		if !errors.Is(err, cause) || len(report.Executions()) != want || len(report.Evidence()) != want {
			t.Fatalf("partial facts or original error lost: %d/%d %v", len(report.Executions()), len(report.Evidence()), err)
		}
	}
}

func TestCompositeInspectorRejectsEmptySuccessfulChild(t *testing.T) {
	artifact := dynamicArtifact(t)
	static := completedReport(t, artifact, "npm-static-archive", "npm-static-evidence")
	inspector, err := NewCompositeInspector(reportInspector{report: static}, reportInspector{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inspector.Inspect(context.Background(), artifact); err == nil {
		t.Fatal("empty successful child disappeared from required coverage")
	}
}
