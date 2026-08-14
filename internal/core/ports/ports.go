// Package ports defines outbound contracts consumed by Heliopause application workflows.
package ports

import (
	"context"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Artifact resolves an Artifact Reference and acquires its exact content subject.
type Artifact interface {
	Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error)
	Acquire(context.Context, domain.RunID, domain.ResolvedArtifact) (domain.AcquiredArtifact, error)
}

// Verification verifies identity and integrity properties of acquired content.
type Verification interface {
	Verify(context.Context, domain.AcquiredArtifact) (domain.VerificationReport, error)
}

// Inspection inspects acquired content and normalizes Evidence and Findings.
type Inspection interface {
	Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error)
}

// Sandbox executes exact controlled content in one ephemeral Session and returns raw observations.
type Sandbox interface {
	Execute(context.Context, domain.SandboxRequest) (domain.SandboxResult, error)
}

// Evidence records a bounded batch and returns trusted references.
type Evidence interface {
	Record(context.Context, domain.RunID, []domain.Evidence) ([]domain.EvidenceReference, error)
}

// DependencyResolver resolves one requested Artifact into a bounded exact graph.
// Implementations own package-manager lockfile/runtime details and must return
// only parser-normalized Domain values.
type DependencyResolver interface {
	ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.LockedDependencyGraph, error)
}
