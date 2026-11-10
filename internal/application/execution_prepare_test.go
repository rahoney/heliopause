package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
)

type executionPreparationPorts struct {
	resolution                         domain.DependencyResolution
	mode                               string
	err                                error
	cancel                             context.CancelFunc
	resolves, entries, records, stages int
	previousEntry                      domain.ExecutionContext
	previousBundle                     domain.VerifiedBundle
}

func (p *executionPreparationPorts) ResolveExecutionDependencies(context.Context, domain.ArtifactReference) (domain.DependencyResolution, error) {
	p.resolves++
	if p.mode == "resolve error" {
		return domain.DependencyResolution{}, p.err
	}
	if p.mode == "resolve cancel" {
		p.cancel()
	}
	return p.resolution, nil
}

func (p *executionPreparationPorts) BindExecutionEntry(_ context.Context, inspected application.ExecutionInspection, command string) (domain.ExecutionContext, error) {
	p.entries++
	if p.mode == "entry error" {
		return domain.ExecutionContext{}, p.err
	}
	if p.mode == "entry zero" {
		return domain.ExecutionContext{}, nil
	}
	if p.mode == "cached entry" && p.previousEntry.Valid() {
		return p.previousEntry, nil
	}
	if p.mode == "entry cancel" {
		p.cancel()
	}
	if command == "" {
		command = "first"
	}
	if p.mode == "wrong command" {
		command = "other"
	}
	primary := inspected.Set().Inspected().Inspections()[0].Artifact()
	member, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	// Adapter context metadata is deliberately different. The workflow must
	// use only its measured member facts and rebind the current request/set.
	operation, err := domain.NewOperationID()
	if err != nil {
		return domain.ExecutionContext{}, err
	}
	lock, _ := domain.NewSHA256Digest(strings.Repeat("e", 64))
	otherResolution, err := domain.NewDependencyResolution(inspected.Resolution().Graph(), "adapter-context-runtime", lock)
	if err != nil {
		return domain.ExecutionContext{}, err
	}
	bound, err := domain.NewExecutionContext(domain.ExecutionContextData{OperationID: operation, Resolution: otherResolution, Set: inspected.Set(), EntryArtifact: primary,
		Command: command, EntryPath: "cli.js", EntryDigest: member, EntrySize: 4, RuntimeIdentity: "fixture-node-runtime", Arguments: []string{"adapter-argv-must-be-ignored"}})
	if err == nil {
		p.previousEntry = bound
	}
	return bound, err
}

func (p *executionPreparationPorts) BuildExecution(ctx context.Context, execution domain.ExecutionContext) (domain.VerifiedBundle, error) {
	p.records++
	if p.mode == "records error" {
		return domain.VerifiedBundle{}, p.err
	}
	if p.mode == "records zero" {
		return domain.VerifiedBundle{}, nil
	}
	if p.mode == "cached records" && p.previousBundle.Valid() {
		return p.previousBundle, nil
	}
	bundle, err := (evidence.Generator{}).BuildExecution(ctx, execution)
	if err == nil {
		p.previousBundle = bundle
	}
	if p.mode == "records cancel" {
		p.cancel()
	}
	return bundle, err
}

func (p *executionPreparationPorts) Stage(_ context.Context, bundle domain.VerifiedBundle) (domain.StagedSet, error) {
	p.stages++
	if p.mode == "stage error" {
		return domain.StagedSet{}, p.err
	}
	if p.mode == "stage zero" {
		return domain.StagedSet{}, nil
	}
	id := bundle.ManifestID()
	handle := "staging:" + id.String()
	if p.mode == "stage id" {
		id, _ = domain.NewSHA256Digest(strings.Repeat("f", 64))
		handle = "staging:" + id.String()
	}
	if p.mode == "stage handle" {
		handle = "staging:other"
	}
	if p.mode == "stage cancel" {
		p.cancel()
	}
	return domain.NewStagedSet(id, handle)
}

func executionPreparationFixture(t *testing.T) (*application.ExecutionPreparationService, *executionPreparationPorts, *multiInspectionPorts, application.ExecutionPreparationRequest) {
	t.Helper()
	graph := twoNodeGraph(t)
	p := &executionPreparationPorts{resolution: executionResolution(t, graph), err: errors.New("original preparation provider failure")}
	entry := newMultiInspectionPorts(t, graph)
	fresh, err := application.NewExecutionInspectService(entry, entry, entry, entry, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewExecutionPreparationService(p, fresh, p, p, p)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "first@1.0.0")
	request, err := application.NewExecutionPreparationRequest(reference, "first", []string{"private-argv-fixture", "", "a b", "--"})
	if err != nil {
		t.Fatal(err)
	}
	return service, p, entry, request
}

func TestExecutionPreparationBindsCurrentInspectionAndPrivateRequest(t *testing.T) {
	s, p, entry, request := executionPreparationFixture(t)
	result, err := s.Prepare(context.Background(), request)
	if err != nil || !result.Valid() || !result.Inspection().Valid() {
		t.Fatal("preparation failed", err)
	}
	execution := result.ExecutionContext()
	if execution.OperationID() != result.Inspection().OperationID() || execution.SourceFingerprint() != p.resolution.LockfileDigest() || execution.Resolution().RuntimeIdentity() != p.resolution.RuntimeIdentity() || !slices.Equal(execution.Arguments(), request.Arguments()) || execution.EntryArtifact() != result.Inspection().Set().Inspected().Inspections()[0].Artifact() {
		t.Fatal("adapter context replaced current operation/resolution/approval/argv")
	}
	if p.resolves != 1 || p.entries != 1 || p.records != 1 || p.stages != 1 || len(entry.calls) != 8 {
		t.Fatal("preparation phase duplicated or bypassed", p, entry.calls)
	}
	for _, body := range [][]byte{result.Bundle().ManifestDocument(), result.Bundle().SBOMDocument()} {
		if bytes.Contains(body, []byte("private-argv-fixture")) || bytes.Contains(body, []byte("adapter-argv")) {
			t.Fatal("private argv persisted")
		}
	}
	first := result.Inspection()
	result, err = s.Prepare(context.Background(), request)
	if err != nil || !result.Valid() || first.OperationID() == result.Inspection().OperationID() || first.Set().Inspected().Inspections()[0].RunID() == result.Inspection().Set().Inspected().Inspections()[0].RunID() || p.resolves != 2 || p.stages != 2 || len(entry.calls) != 16 {
		t.Fatal("repeated request reused approval", err)
	}
	if (application.PreparedExecution{}).Valid() {
		t.Fatal("zero preparation accepted")
	}
}

func TestExecutionPreparationRejectsCachedEntryAndRecords(t *testing.T) {
	for _, mode := range []string{"cached entry", "cached records"} {
		t.Run(mode, func(t *testing.T) {
			s, p, _, request := executionPreparationFixture(t)
			if result, err := s.Prepare(context.Background(), request); err != nil || !result.Valid() {
				t.Fatal(err)
			}
			p.mode = mode
			result, err := s.Prepare(context.Background(), request)
			if err == nil || result.Valid() || p.stages != 1 || p.resolves != 2 {
				t.Fatal("cached binding/approval reached staging", err)
			}
			want := "ENTRY_BINDING_INVALID"
			if mode == "cached records" {
				want = "RECORDS_INVALID"
			}
			if result.FailureCode() != want {
				t.Fatal("wrong failed boundary", result.FailureCode())
			}
		})
	}
}

func TestExecutionPreparationStopsAtExactProviderOrBindingFailure(t *testing.T) {
	for _, test := range []struct {
		mode, code               string
		entries, records, stages int
	}{
		{"resolve error", "RESOLUTION_FAILED", 0, 0, 0},
		{"entry error", "ENTRY_BINDING_FAILED", 1, 0, 0},
		{"entry zero", "ENTRY_BINDING_INVALID", 1, 0, 0},
		{"wrong command", "ENTRY_BINDING_INVALID", 1, 0, 0},
		{"records error", "RECORDS_FAILED", 1, 1, 0},
		{"records zero", "RECORDS_INVALID", 1, 1, 0},
		{"stage error", "STAGING_FAILED", 1, 1, 1},
		{"stage zero", "STAGING_INVALID", 1, 1, 1},
		{"stage id", "STAGING_INVALID", 1, 1, 1},
		{"stage handle", "STAGING_INVALID", 1, 1, 1},
	} {
		t.Run(test.mode, func(t *testing.T) {
			s, p, _, request := executionPreparationFixture(t)
			p.mode = test.mode
			result, err := s.Prepare(context.Background(), request)
			if err == nil || result.Valid() || result.FailureCode() != test.code || p.entries != test.entries || p.records != test.records || p.stages != test.stages {
				t.Fatal("failed boundary continued or approved", err, result.FailureCode())
			}
			if strings.HasSuffix(test.mode, " error") && !errors.Is(err, p.err) {
				t.Fatal("provider cause lost", err)
			}
			if test.entries > 0 && !result.Inspection().Valid() {
				t.Fatal("completed quarantine result lost")
			}
		})
	}
}

func TestExecutionPreparationCancellationNeverCompletesOrCallsLaterPorts(t *testing.T) {
	for _, test := range []struct {
		mode                     string
		entries, records, stages int
	}{
		{"before", 0, 0, 0}, {"resolve cancel", 0, 0, 0}, {"entry cancel", 1, 0, 0}, {"records cancel", 1, 1, 0}, {"stage cancel", 1, 1, 1},
	} {
		t.Run(test.mode, func(t *testing.T) {
			s, p, _, request := executionPreparationFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller stopped preparation")
			p.mode = test.mode
			p.cancel = func() { cancel(cause) }
			if test.mode == "before" {
				cancel(cause)
			}
			result, err := s.Prepare(ctx, request)
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || result.Valid() || result.FailureCode() != "CANCELED" || p.entries != test.entries || p.records != test.records || p.stages != test.stages {
				t.Fatal("cancellation lost or later boundary ran", err, result.FailureCode())
			}
			if test.mode == "before" && p.resolves != 0 {
				t.Fatal("canceled input reached resolver")
			}
		})
	}
}

func TestExecutionPreparationInspectionFailurePreservesFactsAndStopsBinding(t *testing.T) {
	_, p, entry, request := executionPreparationFixture(t)
	original := errors.New("original complete inspection set failure")
	fresh, err := application.NewExecutionInspectService(entry, entry, entry, entry, policy.M1{}, executionSetPolicy{err: original}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	s, err := application.NewExecutionPreparationService(p, fresh, p, p, p)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Prepare(context.Background(), request)
	if !errors.Is(err, original) || result.Valid() || result.Inspection().OperationID().String() == "" || result.FailureCode() != "INSPECTION_FAILED" || p.entries != 0 || p.records != 0 || p.stages != 0 {
		t.Fatal("inspection failure reached binding/staging or lost original operation", err)
	}
}

func TestExecutionPreparationInputPreflightAndPrivateCopies(t *testing.T) {
	s, p, _, request := executionPreparationFixture(t)
	for _, test := range []struct {
		command  string
		args     []string
		accepted bool
	}{
		{"", nil, true}, {"first", []string{"", "a b"}, true}, {strings.Repeat("a", 214), nil, true},
		{strings.Repeat("a", 215), nil, false}, {"/bin/sh", nil, false}, {"..", nil, false}, {"bad\nname", nil, false}, {"bad:name", nil, false},
		{"first", []string{"bad\x00value"}, false}, {"first", []string{string([]byte{0xff})}, false},
		{"first", make([]string, domain.MaximumExecutionArguments), true}, {"first", make([]string, domain.MaximumExecutionArguments+1), false},
		{"first", []string{strings.Repeat("x", domain.MaximumExecutionArgumentBytes)}, true}, {"first", []string{strings.Repeat("x", domain.MaximumExecutionArgumentBytes+1)}, false},
	} {
		value, err := application.NewExecutionPreparationRequest(request.Reference(), test.command, test.args)
		if (err == nil) != test.accepted {
			t.Fatal("input bound differs", err)
		}
		if !test.accepted {
			if result, err := s.Prepare(context.Background(), value); err == nil || result.Valid() || p.resolves != 0 {
				t.Fatal("invalid constructor result reached I/O")
			}
		}
	}
	args := []string{"private-argv-fixture", "a b", ""}
	owned, err := application.NewExecutionPreparationRequest(request.Reference(), "first", args)
	if err != nil {
		t.Fatal(err)
	}
	args[0] = "mutated"
	returned := owned.Arguments()
	returned[1] = "mutated"
	if !slices.Equal(owned.Arguments(), []string{"private-argv-fixture", "a b", ""}) {
		t.Fatal("request retained mutable argv alias")
	}
	if owned.Reference() != request.Reference() || owned.Command() != "first" {
		t.Fatal("request lost reference/command")
	}
	for _, formatted := range []string{fmt.Sprint(owned), fmt.Sprintf("%+v", owned), fmt.Sprintf("%#v", owned)} {
		if strings.Contains(formatted, "private-argv-fixture") || strings.Contains(formatted, "a b") {
			t.Fatal("default formatter leaked argv")
		}
	}
	encoded, err := json.Marshal(owned)
	if err == nil || len(encoded) != 0 || strings.Contains(err.Error(), "private-argv-fixture") {
		t.Fatal("private request serialization was accepted or leaked argv", err)
	}
	for _, ctx := range []context.Context{nil} {
		if result, err := s.Prepare(ctx, owned); err == nil || result.Valid() || p.resolves != 0 {
			t.Fatal("missing context reached I/O")
		}
	}
	var absent *application.ExecutionPreparationService
	if result, err := absent.Prepare(context.Background(), owned); err == nil || result.Valid() {
		t.Fatal("nil service accepted")
	}
	if service, err := application.NewExecutionPreparationService(nil, nil, nil, nil, nil); err == nil || service != nil {
		t.Fatal("missing preparation dependencies accepted")
	}
}
