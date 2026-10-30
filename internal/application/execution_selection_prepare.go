package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type ExecutionSelectionResolver interface {
	ResolveExecutionSelection(context.Context, domain.ArtifactReference) (domain.ExecutionSelectionResolution, error)
}
type ExecutionSelectionEntryBinder interface {
	BindExecutionSelectionEntry(context.Context, ExecutionSelectionInspection, string) (domain.ExecutionSelectionContext, error)
}
type ExecutionSelectionRecords interface {
	BuildExecutionSelection(context.Context, domain.ExecutionSelectionContext) (domain.ExecutionSelectionBundle, error)
}
type ExecutionSelectionStaging interface {
	StageExecutionSelection(context.Context, domain.ExecutionSelectionBundle) (domain.StagedSet, error)
}

// PreparedExecutionSelection carries the current complete roots without
// pretending to be a connected single-root preparation or an installation.
type PreparedExecutionSelection struct {
	inspection ExecutionSelectionInspection
	execution  domain.ExecutionSelectionContext
	bundle     domain.ExecutionSelectionBundle
	staged     domain.StagedSet
	failure    string
}

func (p PreparedExecutionSelection) Inspection() ExecutionSelectionInspection { return p.inspection }
func (p PreparedExecutionSelection) ExecutionContext() domain.ExecutionSelectionContext {
	return p.execution
}
func (p PreparedExecutionSelection) Bundle() domain.ExecutionSelectionBundle { return p.bundle }
func (p PreparedExecutionSelection) Staged() domain.StagedSet                { return p.staged }
func (PreparedExecutionSelection) String() string                            { return "execution selection storage preparation" }
func (p PreparedExecutionSelection) GoString() string                        { return p.String() }
func (p PreparedExecutionSelection) FailureCode() string                     { return p.failure }
func (p PreparedExecutionSelection) Valid() bool {
	return p.failure == "" && p.inspection.Valid() && p.execution.Valid() && p.bundle.Valid() && p.staged.ManifestID() == p.bundle.ManifestID() && p.staged.Handle() == "staging:"+p.bundle.ManifestID().String()
}

type ExecutionSelectionPreparationService struct {
	resolver   ExecutionSelectionResolver
	inspection *ExecutionInspectService
	entry      ExecutionSelectionEntryBinder
	records    ExecutionSelectionRecords
	staging    ExecutionSelectionStaging
}

func NewExecutionSelectionPreparationService(resolver ExecutionSelectionResolver, inspection *ExecutionInspectService, entry ExecutionSelectionEntryBinder, records ExecutionSelectionRecords, staging ExecutionSelectionStaging) (*ExecutionSelectionPreparationService, error) {
	if resolver == nil || inspection == nil || entry == nil || records == nil || staging == nil {
		return nil, errors.New("execution preparation requires complete-root resolution, fresh inspection, entry binding, records and staging")
	}
	return &ExecutionSelectionPreparationService{resolver, inspection, entry, records, staging}, nil
}

func (s *ExecutionSelectionPreparationService) Prepare(ctx context.Context, request ExecutionPreparationRequest) (outcome PreparedExecutionSelection, resultErr error) {
	fail := func(code string, cause error) (PreparedExecutionSelection, error) {
		outcome.failure = code
		if ctx != nil && ctx.Err() != nil {
			cause = errors.Join(cause, context.Cause(ctx))
		}
		return outcome, fmt.Errorf("execution preparation %s: %w", code, cause)
	}
	if ctx == nil || s == nil || s.resolver == nil || s.inspection == nil || s.entry == nil || s.records == nil || s.staging == nil {
		return fail("REQUEST_INVALID", errors.New("configured complete-root preparation and context are required"))
	}
	validated, err := NewExecutionPreparationRequest(request.Reference(), request.Command(), request.Arguments())
	if err != nil {
		return fail("REQUEST_INVALID", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "resolve"})
	resolution, err := s.resolver.ResolveExecutionSelection(ctx, validated.Reference())
	if err != nil {
		return fail("RESOLUTION_FAILED", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	outcome.inspection, err = s.inspection.InspectSelection(ctx, resolution)
	if err != nil {
		return fail("INSPECTION_FAILED", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	measured, err := s.entry.BindExecutionSelectionEntry(ctx, outcome.inspection, validated.Command())
	if err != nil {
		return fail("ENTRY_BINDING_FAILED", err)
	}
	if !measured.Valid() || validated.Command() != "" && measured.Command() != validated.Command() {
		return fail("ENTRY_BINDING_INVALID", errors.New("measured primary console differs from the requested command"))
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	// Only member facts come from the binder. This operation's whole approval
	// and this request's private argv always replace adapter-supplied values.
	outcome.execution, err = domain.NewExecutionSelectionContext(domain.ExecutionSelectionContextData{
		OperationID: outcome.inspection.OperationID(), Resolution: resolution, Set: outcome.inspection.Set(),
		EntryArtifact: measured.EntryArtifact(), Command: measured.Command(), EntryPath: measured.EntryPath(), EntryDigest: measured.EntryDigest(), EntrySize: measured.EntrySize(), RuntimeIdentity: measured.RuntimeIdentity(), Arguments: validated.Arguments(),
	})
	if err != nil {
		return fail("ENTRY_BINDING_INVALID", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "manifest"})
	outcome.bundle, err = s.records.BuildExecutionSelection(ctx, outcome.execution)
	if err != nil {
		return fail("RECORDS_FAILED", err)
	}
	if !outcome.bundle.Valid() || !reflect.DeepEqual(outcome.bundle.Set(), outcome.inspection.Set()) {
		return fail("RECORDS_INVALID", errors.New("records changed the current complete approved selection"))
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "stage"})
	outcome.staged, err = s.staging.StageExecutionSelection(ctx, outcome.bundle)
	if err != nil {
		return fail("STAGING_FAILED", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	if !outcome.Valid() {
		return fail("STAGING_INVALID", errors.New("staging receipt differs from complete current records"))
	}
	return outcome, nil
}
