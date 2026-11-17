package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

func TestInspectSyntheticDecisions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scenario fakeworkflow.Scenario
		decision domain.Decision
		reason   string
	}{
		{name: "allow", scenario: fakeworkflow.Safe, decision: domain.DecisionAllow, reason: "M1_REQUIRED_CHECKS_COMPLETED"},
		{name: "manual review", scenario: fakeworkflow.Review, decision: domain.DecisionManualReview, reason: "M1_REQUIRED_CHECK_INCOMPLETE"},
		{name: "block", scenario: fakeworkflow.Blocked, decision: domain.DecisionBlock, reason: "M1_BLOCK_FINDING"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service, fake, request := inspectFixture(t, test.scenario)
			result, err := service.Inspect(context.Background(), request)
			if err != nil {
				t.Fatalf("Inspect() error = %v", err)
			}
			if result.Status() != domain.OperationCompleted {
				t.Fatalf("Status() = %q", result.Status())
			}
			if outcome, ok := result.RunOutcome(); !ok || outcome != domain.RunCompleted {
				t.Fatalf("RunOutcome() = %q, %v", outcome, ok)
			}
			decision, ok := result.PolicyDecision()
			if !ok || decision.Decision() != test.decision || !reflect.DeepEqual(decision.Reasons(), []string{test.reason}) {
				t.Fatalf("PolicyDecision() = %#v, %v", decision, ok)
			}
			if _, ok := result.Error(); ok {
				t.Fatal("completed result contains operational error")
			}
			if got, want := fake.Calls(), []string{"resolve", "acquire", "verify", "inspect", "evidence"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("Calls() = %v, want %v", got, want)
			}
		})
	}
}

func TestInspectOperationalFailuresPreservePartialResultAndCause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scenario    fakeworkflow.Scenario
		cause       error
		calls       []string
		hasRun      bool
		hasIdentity bool
		code        string
	}{
		{name: "resolve", scenario: fakeworkflow.ResolveError, cause: fakeworkflow.ErrResolve, calls: []string{"resolve"}, code: "ARTIFACT_RESOLVE_FAILED"},
		{name: "acquire", scenario: fakeworkflow.AcquireError, cause: fakeworkflow.ErrAcquire, calls: []string{"resolve", "acquire"}, hasRun: true, hasIdentity: true, code: "ARTIFACT_ACQUIRE_FAILED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service, fake, request := inspectFixture(t, test.scenario)
			result, err := service.Inspect(context.Background(), request)
			if !errors.Is(err, test.cause) {
				t.Fatalf("Inspect() error = %v, want wrapped %v", err, test.cause)
			}
			if result.Status() != domain.OperationFailed {
				t.Fatalf("Status() = %q", result.Status())
			}
			_, hasRun := result.RunID()
			_, hasIdentity := result.ResolvedIdentity()
			if hasRun != test.hasRun || hasIdentity != test.hasIdentity {
				t.Fatalf("partial result hasRun=%v hasIdentity=%v", hasRun, hasIdentity)
			}
			if test.hasRun {
				if outcome, ok := result.RunOutcome(); !ok || outcome != domain.RunFailed {
					t.Fatalf("RunOutcome() = %q, %v", outcome, ok)
				}
			}
			if _, ok := result.PolicyDecision(); ok {
				t.Fatal("failed result contains Policy Decision")
			}
			operationError, ok := result.Error()
			if !ok || operationError.Code() != test.code {
				t.Fatalf("Error() = %#v, %v", operationError, ok)
			}
			if got := fake.Calls(); !reflect.DeepEqual(got, test.calls) {
				t.Fatalf("Calls() = %v, want %v", got, test.calls)
			}
		})
	}
}

func TestInspectCreatesRunAfterResolveAndBeforeAcquire(t *testing.T) {
	t.Parallel()

	fake, err := fakeworkflow.New(fakeworkflow.Safe)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := fake.Reference()
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInspectRequest(reference)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := domain.ParseOperationID("op_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	runID, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	runGeneratorCalled := false
	service, err := application.NewInspectService(fake, fake, fake, fake, policy.M1{}, func() (domain.OperationID, error) {
		return operationID, nil
	}, func() (domain.RunID, error) {
		runGeneratorCalled = true
		if got := fake.Calls(); !reflect.DeepEqual(got, []string{"resolve"}) {
			t.Fatalf("calls at Run creation = %v, want resolve only", got)
		}
		return runID, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Inspect(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if !runGeneratorCalled {
		t.Fatal("Run ID generator was not called")
	}
}

func inspectFixture(t *testing.T, scenario fakeworkflow.Scenario) (*application.InspectService, *fakeworkflow.Ports, application.InspectRequest) {
	t.Helper()
	fake, err := fakeworkflow.New(scenario)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := fake.Reference()
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInspectRequest(reference)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := domain.ParseOperationID("op_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	runID, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInspectService(fake, fake, fake, fake, policy.M1{}, func() (domain.OperationID, error) { return operationID, nil }, func() (domain.RunID, error) { return runID, nil })
	if err != nil {
		t.Fatal(err)
	}
	return service, fake, request
}

// A late provider failure must not erase already confirmed facts before the
// operation's transient payloads can be finalized. These results never approve.
type operationalInspectionPorts struct {
	*fakeworkflow.Ports
	partial            bool
	foreign            bool
	cancel             context.CancelFunc
	cause, recordError error
	recorded           int
	boundedRecord      bool
}

func (p *operationalInspectionPorts) Inspect(ctx context.Context, a domain.AcquiredArtifact) (domain.InspectionReport, error) {
	report, err := p.Ports.Inspect(ctx, a)
	if err != nil {
		return report, err
	}
	if p.cancel != nil {
		p.cancel()
	}
	if !p.partial {
		report = domain.InspectionReport{}
	}
	if p.foreign {
		identity, _ := domain.NewResolvedArtifactIdentity(a.Identity().Source(), "foreign", a.Identity().Version(), a.Identity().Variant())
		id, _ := domain.NewEvidenceID("foreign-inspection")
		fact, e := domain.NewEvidence(id, report.Execution().ID(), identity, a.Digest(), "fixture", "Foreign subject must be rejected.")
		if e != nil {
			return domain.InspectionReport{}, e
		}
		report, e = domain.NewInspectionReport(report.Execution(), nil, []domain.Evidence{fact})
		if e != nil {
			return report, e
		}
	}
	return report, p.cause
}
func (p *operationalInspectionPorts) Record(ctx context.Context, run domain.RunID, facts []domain.Evidence, checks ...domain.CheckExecution) ([]domain.EvidenceReference, error) {
	p.recorded = len(facts)
	_, p.boundedRecord = ctx.Deadline()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.recordError != nil {
		return nil, p.recordError
	}
	return p.Ports.Record(ctx, run, facts, checks...)
}
func TestFailedInspectionRetainsConfirmedEvidence(t *testing.T) {
	for _, install := range []bool{false, true} {
		for _, mode := range []string{"empty-report", "partial-report", "canceled", "record-failed", "foreign"} {
			name := mode
			if install {
				name = "install/" + name
			} else {
				name = "inspect/" + name
			}
			t.Run(name, func(t *testing.T) {
				base, _ := fakeworkflow.New(fakeworkflow.Safe)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cause := errors.New("late inspection failure")
				recordError := errors.New("storage unavailable")
				fixture := &operationalInspectionPorts{Ports: base, partial: mode != "empty-report", foreign: mode == "foreign", cause: cause}
				if mode == "canceled" {
					fixture.cancel = cancel
				}
				if mode == "record-failed" {
					fixture.recordError = recordError
				}
				reference, _ := fixture.Reference()
				var result domain.OperationResult
				var err error
				if install {
					target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
					installContext, _ := domain.NewInstallContext(target)
					request, _ := application.NewInstallRequest(reference, installContext)
					service, e := application.NewInstallInspectService(&lockedResolver{graph: fixtureLockedGraph(t)}, fixture, fixture, fixture, fixture, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
					if e != nil {
						t.Fatal(e)
					}
					outcome, e := service.Inspect(ctx, request)
					err = e
					if outcome.Set().Valid() {
						t.Fatal("failed partial inspection became an approved set")
					}
					var failure *application.InspectionFailure
					if !errors.As(err, &failure) || len(failure.Results()) != 1 {
						t.Fatalf("partial result unavailable: %v", err)
					}
					result = failure.Results()[0]
				} else {
					request, _ := application.NewInspectRequest(reference)
					service, e := application.NewInspectService(fixture, fixture, fixture, fixture, policy.M1{}, domain.NewOperationID, domain.NewRunID)
					if e != nil {
						t.Fatal(e)
					}
					result, err = service.Inspect(ctx, request)
				}
				if !errors.Is(err, cause) {
					t.Fatalf("first failure lost: %v", err)
				}
				if mode == "record-failed" && !errors.Is(err, recordError) {
					t.Fatalf("secondary record failure lost: %v", err)
				}
				want := 2
				if mode == "empty-report" || mode == "foreign" {
					want = 1
				}
				if fixture.recorded != want || !fixture.boundedRecord {
					t.Fatalf("confirmed facts not saved with bounded finalization context: %d/%v", fixture.recorded, fixture.boundedRecord)
				}
				if len(result.Checks()) != want {
					t.Fatalf("actual checks lost: %d", len(result.Checks()))
				}
				wantRefs := want
				if mode == "record-failed" {
					wantRefs = 0
				}
				if len(result.Evidence()) != wantRefs {
					t.Fatalf("durable evidence references=%d, want %d", len(result.Evidence()), wantRefs)
				}
				if _, ok := result.PolicyDecision(); ok || result.Status() != domain.OperationFailed {
					t.Fatal("operational failure acquired Policy/completion")
				}
			})
		}
	}
}
