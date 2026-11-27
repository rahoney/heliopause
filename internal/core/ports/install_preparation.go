package ports

import (
	"context"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// InstallSession retains one target guard through selection, inspection and
// publication. It confers neither content approval nor update intent.
type InstallSession interface {
	DependencyResolver
	Promotion
	Close() error
}

// InstallPreparation freezes the original target before ordinary installation.
// The original reference remains the caller's; installed update selection uses
// the separate InstallUpdatePreparation/Reference contract.
type InstallPreparation interface {
	BeginInstall(context.Context, domain.ArtifactReference, domain.InstallContext) (InstallSession, error)
}
