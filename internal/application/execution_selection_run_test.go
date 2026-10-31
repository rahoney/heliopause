package application_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
)

type selectionPreparationPorts struct {
	resolution     domain.ExecutionSelectionResolution
	mode           string
	previousEntry  domain.ExecutionSelectionContext
	previousBundle domain.ExecutionSelectionBundle
	stages         int
}

func (p *selectionPreparationPorts) ResolveExecutionSelection(context.Context, domain.ArtifactReference) (domain.ExecutionSelectionResolution, error) {
	return p.resolution, nil
}
func (p *selectionPreparationPorts) BindExecutionSelectionEntry(_ context.Context, inspected application.ExecutionSelectionInspection, command string) (domain.ExecutionSelectionContext, error) {
	if p.mode == "cached entry" {
		return p.previousEntry, nil
	}
	var primary domain.AcquiredArtifact
	for _, entry := range inspected.Set().Inspected().Inspections() {
		if entry.Node() == inspected.Resolution().Selection().Primary() {
			primary = entry.Artifact()
		}
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	op, _ := domain.NewOperationID()
	if p.mode == "wrong command" {
		command = "other"
	}
	bound, err := domain.NewExecutionSelectionContext(domain.ExecutionSelectionContextData{OperationID: op, Resolution: inspected.Resolution(), Set: inspected.Set(), EntryArtifact: primary, Command: command, EntryPath: "cli.js", EntryDigest: digest, EntrySize: 4, RuntimeIdentity: "fixture-runtime", Arguments: []string{"adapter-private-argv"}})
	if err == nil {
		p.previousEntry = bound
	}
	return bound, err
}
func (p *selectionPreparationPorts) BuildExecutionSelection(ctx context.Context, execution domain.ExecutionSelectionContext) (domain.ExecutionSelectionBundle, error) {
	if p.mode == "cached records" {
		return p.previousBundle, nil
	}
	bundle, err := (evidence.Generator{}).BuildExecutionSelection(ctx, execution)
	if err == nil {
		p.previousBundle = bundle
	}
	return bundle, err
}
func (p *selectionPreparationPorts) StageExecutionSelection(_ context.Context, bundle domain.ExecutionSelectionBundle) (domain.StagedSet, error) {
	p.stages++
	handle := "staging:" + bundle.ManifestID().String()
	if p.mode == "wrong staging" {
		handle = "staging:other"
	}
	return domain.NewStagedSet(bundle.ManifestID(), handle)
}

type selectionRuntimeFixture struct {
	executionRuntimeFixture
	selections []domain.ExecutionSelectionContext
}

func (r *selectionRuntimeFixture) ExecuteSelection(_ context.Context, execution domain.ExecutionSelectionContext) (application.ExecutionRuntimeOutcome, error) {
	r.selections = append(r.selections, execution)
	if r.after != nil {
		r.after()
	}
	return &r.executionRuntimeFixture, r.err
}
func selectionWorkflowFixture(t *testing.T) (*application.ExecutionSelectionService, *selectionPreparationPorts, *selectionRuntimeFixture, application.ExecutionPreparationRequest) {
	t.Helper()
	resolution, entry := selectionInspectionFixture(t)
	whole := &selectionInspectionPolicy{decision: domain.DecisionAllow}
	fresh, err := application.NewExecutionInspectService(entry, entry, entry, entry, policy.M1{}, whole, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	p := &selectionPreparationPorts{resolution: resolution}
	prepared, err := application.NewExecutionSelectionPreparationService(p, fresh, p, p, p)
	if err != nil {
		t.Fatal(err)
	}
	runtime := &selectionRuntimeFixture{executionRuntimeFixture: executionRuntimeFixture{complete: true, exited: true, stdout: []byte("private-output")}}
	service, err := application.NewExecutionSelectionService(prepared, runtime)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "first@1.0.0")
	request, err := application.NewExecutionPreparationRequest(reference, "first", []string{"request-private-argv", "", "a b"})
	if err != nil {
		t.Fatal(err)
	}
	return service, p, runtime, request
}
func TestExecutionSelectionWorkflowRebindsFreshWholeApprovalAndPrivateRequest(t *testing.T) {
	service, _, runtime, request := selectionWorkflowFixture(t)
	var previous domain.OperationID
	for range 2 {
		result, err := service.Run(context.Background(), request)
		prepared := result.SelectionPreparation()
		if err != nil || !result.Complete() || !prepared.Valid() || result.Preparation().Valid() {
			t.Fatal("whole execution preparation missing or manufactured single-root preparation", err)
		}
		bound := runtime.selections[len(runtime.selections)-1]
		if bound.OperationID() == previous || bound.OperationID() != prepared.Inspection().OperationID() || !reflect.DeepEqual(bound.Set(), prepared.Inspection().Set()) || !slices.Equal(bound.Arguments(), request.Arguments()) || len(bound.Set().Inspected().Inspections()) != 2 {
			t.Fatal("cached/partial approval or adapter argv entered runtime")
		}
		previous = bound.OperationID()
	}
}
func TestExecutionSelectionWorkflowRefusesCachedOrSubstitutedPreparation(t *testing.T) {
	for _, row := range []struct{ mode, code string }{{"cached entry", "ENTRY_BINDING_INVALID"}, {"cached records", "RECORDS_INVALID"}, {"wrong command", "ENTRY_BINDING_INVALID"}, {"wrong staging", "STAGING_INVALID"}} {
		t.Run(row.mode, func(t *testing.T) {
			service, p, runtime, request := selectionWorkflowFixture(t)
			if first, err := service.Run(context.Background(), request); err != nil || !first.Complete() {
				t.Fatal(err)
			}
			p.mode = row.mode
			result, err := service.Run(context.Background(), request)
			if err == nil || result.Complete() || result.RuntimeAttempted() || len(runtime.selections) != 1 || result.SelectionPreparation().FailureCode() != row.code {
				t.Fatal("substitution reached runtime", err)
			}
		})
	}
}
func TestExecutionSelectionWorkflowPreservesTypedTerminalAndCancellation(t *testing.T) {
	service, _, runtime, request := selectionWorkflowFixture(t)
	cause := errors.New("original guest failure")
	runtime.code, runtime.err = 2, cause
	result, err := service.Run(context.Background(), request)
	code, exited := result.ExitCode()
	if !result.Complete() || !exited || code != 2 || !errors.Is(err, cause) {
		t.Fatal("typed nonzero was lost", err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	stop := errors.New("original caller stopped")
	runtime.after = func() { cancel(stop) }
	result, err = service.Run(ctx, request)
	if result.Complete() || result.FailureCode() != "CANCELED" || !errors.Is(err, stop) || !errors.Is(err, cause) {
		t.Fatal("cancellation became success or erased primary cause", err)
	}
}
