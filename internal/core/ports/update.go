package ports

import (
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// InstallUpdateFailure carries a controller-selected reason, never artifact
// output. The cause remains available for cancellation/primary-error handling.
type InstallUpdateFailure struct {
	code  string
	cause error
}

func NewInstallUpdateFailure(code string, cause error) error {
	switch code {
	case "UPDATE_ADOPTION_REQUIRED", "UPDATE_OWNERSHIP_INVALID", "UPDATE_ORIGINAL_STATE_UNSUPPORTED", "UPDATE_SOURCE_CHANGED", "UPDATE_UNSUPPORTED_PROJECT":
	default:
		code = "UPDATE_PREPARATION_FAILED"
	}
	return &InstallUpdateFailure{code, cause}
}
func (e *InstallUpdateFailure) Error() string { return e.code }
func (e *InstallUpdateFailure) Unwrap() error { return e.cause }
func InstallUpdateFailureCode(err error) string {
	var failure *InstallUpdateFailure
	if errors.As(err, &failure) {
		return failure.code
	}
	return ""
}

// DependencyUpdateResolver selects from controls captured before resolution.
// It returns candidates only; all existing inspection and Policy gates follow.
type DependencyUpdateResolver interface {
	ResolveDependencyUpdate(context.Context, domain.ArtifactReference, domain.InstallContext, []domain.ProjectControlFile) (domain.DependencyResolution, error)
}

// InstallUpdateSession owns one original-state guard through exact selection,
// existing inspection/staging, guarded replacement and cleanup. It grants no
// arbitrary native command forwarding or implicit foreign-state ownership.
type InstallUpdateSession interface {
	DependencyResolver
	Promotion
	Reference() domain.ArtifactReference
	Close() error
}

type InstallUpdatePreparation interface {
	BeginInstallUpdate(context.Context, domain.SourceID, string, domain.InstallContext, bool) (InstallUpdateSession, error)
}
