package ports

import (
	"context"
	"errors"
)

// BootstrapPreparationResult describes downloaded provenance-bound bytes. It
// contains no quarantine, Policy, root installation or healthy-runtime grant.
type BootstrapPreparationResult struct {
	Version        string   `json:"version"`
	ManifestSHA256 string   `json:"manifest_sha256"`
	Directory      string   `json:"directory"`
	Files          []string `json:"files"`
	Proofs         []string `json:"proofs"`
}
type BootstrapPreparation interface {
	Prepare(context.Context, string) (BootstrapPreparationResult, error)
}

type BootstrapFailureCode string

const (
	BootstrapInvalid           BootstrapFailureCode = "BOOTSTRAP_VERSION_INVALID"
	BootstrapUnsupported       BootstrapFailureCode = "BOOTSTRAP_PLATFORM_UNSUPPORTED"
	BootstrapRoot              BootstrapFailureCode = "BOOTSTRAP_ORDINARY_USER_REQUIRED"
	BootstrapAcquisitionFailed BootstrapFailureCode = "BOOTSTRAP_ACQUISITION_FAILED"
	BootstrapProvenanceFailed  BootstrapFailureCode = "BOOTSTRAP_PROVENANCE_FAILED"
	BootstrapExportFailed      BootstrapFailureCode = "BOOTSTRAP_EXPORT_FAILED"
	BootstrapCleanupFailed     BootstrapFailureCode = "BOOTSTRAP_CLEANUP_FAILED"
	BootstrapCancelled         BootstrapFailureCode = "BOOTSTRAP_CANCELLED"
)

type BootstrapPreparationFailure struct {
	code  BootstrapFailureCode
	cause error
}

func (f *BootstrapPreparationFailure) Error() string              { return string(f.code) }
func (f *BootstrapPreparationFailure) Unwrap() error              { return f.cause }
func (f *BootstrapPreparationFailure) Code() BootstrapFailureCode { return f.code }
func NewBootstrapPreparationFailure(code BootstrapFailureCode, cause error) error {
	switch code {
	case BootstrapInvalid, BootstrapUnsupported, BootstrapRoot, BootstrapAcquisitionFailed, BootstrapProvenanceFailed, BootstrapExportFailed, BootstrapCleanupFailed, BootstrapCancelled:
	default:
		code = BootstrapExportFailed
	}
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		code = BootstrapCancelled
	}
	return &BootstrapPreparationFailure{code, cause}
}
