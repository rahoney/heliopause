// Package ports defines outbound contracts consumed by Heliopause application workflows.
package ports

import (
	"context"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Artifact resolves an Artifact Reference and acquires its exact content subject.
type Artifact interface {
	Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifactIdentity, error)
	Acquire(context.Context, domain.ResolvedArtifactIdentity) (domain.AcquiredArtifact, error)
}

// Verification verifies identity and integrity properties of acquired content.
type Verification interface {
	Verify(context.Context, domain.AcquiredArtifact) (domain.VerificationReport, error)
}

// Inspection inspects acquired content and normalizes Evidence and Findings.
type Inspection interface {
	Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error)
}

// Evidence records a bounded batch and returns trusted references.
type Evidence interface {
	Record(context.Context, domain.RunID, []domain.Evidence) ([]domain.EvidenceReference, error)
}
