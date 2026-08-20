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
	ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error)
}

// Derivation produces controller-owned derived Artifacts from an already
// inspected set. The Application re-runs verification, inspection, Evidence
// recording and entry Policy on each returned Artifact.
type Derivation interface {
	Derive(context.Context, []domain.DependencyInspection) ([]domain.DerivedDependency, error)
}

// Staging persists one immutable Manifest/SBOM-bound Verified Set after
// rechecking every intake artifact at the Quarantine-to-Staging boundary.
type Staging interface {
	Stage(context.Context, domain.VerifiedBundle) (domain.StagedSet, error)
}

// Manifest creates deterministic records from one complete ALLOW set.
type Manifest interface {
	Build(context.Context, domain.OperationID, domain.InstallContext, domain.DependencyResolution, domain.VerifiedSet) (domain.VerifiedBundle, error)
}

// Promotion installs only an already-staged exact bundle into a new target.
type Promotion interface {
	Promote(context.Context, domain.StagedSet, domain.VerifiedBundle, domain.InstallContext) (domain.PromotedInstall, error)
}
