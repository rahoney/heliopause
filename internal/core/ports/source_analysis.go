package ports

import (
	"context"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Source DATA is bounded before retention and again at runtime admission.
const (
	SourceAnalysisFileLimit   = 64 << 20
	SourceAnalysisTotalLimit  = 512 << 20
	SourceAnalysisTargetLimit = 10000

	// Static analysis has its own bounded authorization. Large, valid compiler
	// sources exceed the unrelated default Python execution budget.
	SourceAnalysisResourceProfile          = "source-analysis"
	SourceAnalysisMemoryLimit        int64 = 4 << 30
	SourceAnalysisHeapLimitMiB             = 3072
	SourceAnalysisRuleTimeoutSeconds       = 15
	SourceAnalysisCPUSeconds               = 600
	SourceAnalysisWallSeconds              = 15 * 60
	SourceAnalysisMaxLaunches              = 4
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
	// Targets binds each opaque parser path to the original checked source.
	// It is controller metadata, never read from the parser's stdout.
	Targets map[string]string
}

// SourceAnalysis executes only the selected static analyzer. Source DATA is
// never installed or executed; finding interpretation remains in Inspection.
type SourceAnalysis interface {
	Analyze(context.Context, SourceAnalysisInput) (SourceAnalysisOutput, error)
}
