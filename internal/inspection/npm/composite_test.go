package npm

import (
	"context"
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

type reportInspector struct{ report domain.InspectionReport }

func (i reportInspector) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.report, nil
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
