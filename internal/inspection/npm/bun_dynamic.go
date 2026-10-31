package npm

import (
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func bunInstanceFacts(subject domain.AcquiredArtifact, instance string, layout artifactnpm.BunInspectionLayout, result domain.SandboxResult) (domain.InspectionReport, error) {
	return npmProjectInstanceFacts(subject, instance, layout, result, projectInspectionBun)
}

func bunCoverageReport(subject domain.AcquiredArtifact, expected, attempted, completed int, failed bool) (domain.InspectionReport, error) {
	return npmProjectCoverageReport(subject, expected, attempted, completed, failed, projectInspectionBun)
}
