package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// These narrow preparation boundaries have concrete resolver, Node binding
// and execution-record adapters. None admits or executes a user program.
type ExecutionResolver interface {
	ResolveExecutionDependencies(context.Context, domain.ArtifactReference) (domain.DependencyResolution, error)
}
type ExecutionEntryBinder interface {
	BindExecutionEntry(context.Context, ExecutionInspection, string) (domain.ExecutionContext, error)
}
type ExecutionRecords interface {
	BuildExecution(context.Context, domain.ExecutionContext) (domain.VerifiedBundle, error)
}

type ExecutionPreparationRequest struct {
	reference domain.ArtifactReference
	command   string
	arguments []string
}

func NewExecutionPreparationRequest(reference domain.ArtifactReference, command string, arguments []string) (ExecutionPreparationRequest, error) {
	if reference.Source().String() == "" || command != "" && domain.ValidateExecutionCommand(command) != nil {
		return ExecutionPreparationRequest{}, errors.New("execution preparation requires a valid reference and optional explicit command")
	}
	if err := domain.ValidateExecutionArguments(arguments); err != nil {
		return ExecutionPreparationRequest{}, err
	}
	return ExecutionPreparationRequest{reference, command, append([]string(nil), arguments...)}, nil
}

func (r ExecutionPreparationRequest) Reference() domain.ArtifactReference { return r.reference }
func (r ExecutionPreparationRequest) Command() string                     { return r.command }
func (r ExecutionPreparationRequest) Arguments() []string {
	return append([]string(nil), r.arguments...)
}
func (r ExecutionPreparationRequest) String() string   { return "execution preparation request" }
func (r ExecutionPreparationRequest) GoString() string { return r.String() }

// MarshalJSON rejects persistence of private execution request data.
func (r ExecutionPreparationRequest) MarshalJSON() ([]byte, error) {
	return nil, errors.New("execution preparation request is private memory data")
}

// PreparedExecution is storage preparation, never proof of a program's exit,
// output, admission or completed sandbox lifecycle.
type PreparedExecution struct {
	inspection ExecutionInspection
	execution  domain.ExecutionContext
	bundle     domain.VerifiedBundle
	staged     domain.StagedSet
	failure    string
}

func (p PreparedExecution) Inspection() ExecutionInspection           { return p.inspection }
func (p PreparedExecution) ExecutionContext() domain.ExecutionContext { return p.execution }
func (p PreparedExecution) Bundle() domain.VerifiedBundle             { return p.bundle }
func (p PreparedExecution) Staged() domain.StagedSet                  { return p.staged }
func (p PreparedExecution) FailureCode() string                       { return p.failure }
func (p PreparedExecution) String() string                            { return "execution storage preparation" }
func (p PreparedExecution) GoString() string                          { return p.String() }
func (p PreparedExecution) Valid() bool {
	return p.failure == "" && p.inspection.Valid() && p.execution.Valid() && p.bundle.Valid() && p.staged.ManifestID() == p.bundle.ManifestID() && p.staged.Handle() == "staging:"+p.bundle.ManifestID().String()
}

type ExecutionPreparationService struct {
	resolver   ExecutionResolver
	inspection *ExecutionInspectService
	entry      ExecutionEntryBinder
	records    ExecutionRecords
	staging    ports.Staging
}

func NewExecutionPreparationService(resolver ExecutionResolver, inspection *ExecutionInspectService, entry ExecutionEntryBinder, records ExecutionRecords, staging ports.Staging) (*ExecutionPreparationService, error) {
	if resolver == nil || inspection == nil || entry == nil || records == nil || staging == nil {
		return nil, errors.New("execution preparation requires resolver, fresh inspection, entry binding, records and staging")
	}
	return &ExecutionPreparationService{resolver, inspection, entry, records, staging}, nil
}

func (s *ExecutionPreparationService) Prepare(ctx context.Context, request ExecutionPreparationRequest) (PreparedExecution, error) {
	outcome := PreparedExecution{}
	fail := func(code string, cause error) (PreparedExecution, error) {
		outcome.failure = code
		if ctx != nil && ctx.Err() != nil {
			cause = errors.Join(cause, context.Cause(ctx))
		}
		return outcome, fmt.Errorf("execution preparation %s: %w", code, cause)
	}
	if ctx == nil || s == nil || s.resolver == nil || s.inspection == nil || s.entry == nil || s.records == nil || s.staging == nil {
		return fail("REQUEST_INVALID", errors.New("valid context and configured preparation service are required"))
	}
	validated, err := NewExecutionPreparationRequest(request.reference, request.command, request.arguments)
	if err != nil {
		return fail("REQUEST_INVALID", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "resolve"})
	resolution, err := s.resolver.ResolveExecutionDependencies(ctx, validated.reference)
	if err != nil {
		return fail("RESOLUTION_FAILED", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	outcome.inspection, err = s.inspection.InspectResolved(ctx, resolution)
	if err != nil {
		return fail("INSPECTION_FAILED", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	measured, err := s.entry.BindExecutionEntry(ctx, outcome.inspection, validated.command)
	if err != nil {
		return fail("ENTRY_BINDING_FAILED", err)
	}
	if !measured.Valid() || validated.command != "" && measured.Command() != validated.command {
		return fail("ENTRY_BINDING_INVALID", errors.New("entry binding differs from explicit command or is incomplete"))
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	// Consume measured member facts only. Current approval and private argv
	// always come from this inspection/request, never an adapter's cached set.
	outcome.execution, err = domain.NewExecutionContext(domain.ExecutionContextData{
		OperationID: outcome.inspection.OperationID(), Resolution: resolution, Set: outcome.inspection.Set(),
		EntryArtifact: measured.EntryArtifact(), Command: measured.Command(), EntryPath: measured.EntryPath(),
		EntryDigest: measured.EntryDigest(), EntrySize: measured.EntrySize(), RuntimeIdentity: measured.RuntimeIdentity(), Arguments: validated.arguments,
	})
	if err != nil {
		return fail("ENTRY_BINDING_INVALID", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "manifest"})
	outcome.bundle, err = s.records.BuildExecution(ctx, outcome.execution)
	if err != nil {
		return fail("RECORDS_FAILED", err)
	}
	if !outcome.bundle.Valid() || !reflect.DeepEqual(outcome.bundle.Set(), outcome.inspection.Set()) {
		return fail("RECORDS_INVALID", errors.New("execution records changed the current complete approved set"))
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "stage"})
	outcome.staged, err = s.staging.Stage(ctx, outcome.bundle)
	if err != nil {
		return fail("STAGING_FAILED", err)
	}
	if err := ctx.Err(); err != nil {
		return fail("CANCELED", err)
	}
	if outcome.staged.ManifestID() != outcome.bundle.ManifestID() || outcome.staged.Handle() != "staging:"+outcome.bundle.ManifestID().String() {
		return fail("STAGING_INVALID", errors.New("staging receipt differs from current execution records"))
	}
	return outcome, nil
}
