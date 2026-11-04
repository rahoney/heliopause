package ports

import (
	"context"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// SourceAnalysisFile contains normalized, archive-checked DATA. It supplies
// neither a Host path nor executable commands/configuration to the backend.
type SourceAnalysisFile struct {
	Path, Language string
	Body           []byte
}

type SourceAnalysisInput struct {
	Artifact domain.AcquiredArtifact
	Files    []SourceAnalysisFile
}

// Output bytes are never completion authority. Inspection must also bind the
// existing trusted runtime outcome, pinned rules and exact target inventory.
type SourceAnalysisOutput struct {
	Runtime domain.SandboxResult
	Output  domain.CapturedProcessOutput
	Rules   []byte
}

// SourceAnalysis executes only the selected static analyzer. Source DATA is
// never installed or executed; finding interpretation remains in Inspection.
type SourceAnalysis interface {
	Analyze(context.Context, SourceAnalysisInput) (SourceAnalysisOutput, error)
}
