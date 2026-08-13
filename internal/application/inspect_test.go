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
