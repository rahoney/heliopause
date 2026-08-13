package fakeworkflow_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

func TestSyntheticScenarioContracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		scenario   fakeworkflow.Scenario
		capability domain.Capability
		status     domain.ExecutionStatus
		finding    string
	}{
		{name: "safe", scenario: fakeworkflow.Safe, capability: domain.CapabilitySupported, status: domain.ExecutionCompleted},
		{name: "review", scenario: fakeworkflow.Review, capability: domain.CapabilityUnsupported, status: domain.ExecutionNotExecuted},
		{name: "blocked", scenario: fakeworkflow.Blocked, capability: domain.CapabilitySupported, status: domain.ExecutionCompleted, finding: "M1_BLOCK_FINDING"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fake := mustFake(t, test.scenario)
			reference, err := fake.Reference()
			if err != nil {
				t.Fatal(err)
			}
			identity, err := fake.Resolve(context.Background(), reference)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := fake.Acquire(context.Background(), identity)
			if err != nil {
				t.Fatal(err)
			}
			verification, err := fake.Verify(context.Background(), artifact)
			if err != nil || verification.Outcome() != domain.VerificationVerified {
				t.Fatalf("Verify() = %#v, %v", verification, err)
			}
			inspection, err := fake.Inspect(context.Background(), artifact)
			if err != nil {
				t.Fatal(err)
			}
			if inspection.Execution().Capability() != test.capability || inspection.Execution().Status() != test.status {
				t.Fatalf("inspection execution = %#v", inspection.Execution())
			}
			findings := inspection.Findings()
			if test.finding == "" && len(findings) != 0 {
				t.Fatalf("findings = %v, want none", findings)
			}
			if test.finding != "" && (len(findings) != 1 || findings[0].Code() != test.finding) {
				t.Fatalf("findings = %v, want %q", findings, test.finding)
			}
			evidence := append(verification.Evidence(), inspection.Evidence()...)
			runID, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
			if err != nil {
				t.Fatal(err)
			}
			references, err := fake.Record(context.Background(), runID, evidence)
			if err != nil || len(references) != len(evidence) {
				t.Fatalf("Record() references = %d, error = %v", len(references), err)
			}
			if got, want := fake.Calls(), []string{"resolve", "acquire", "verify", "inspect", "evidence"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("Calls() = %v, want %v", got, want)
			}
		})
	}
}

func TestSyntheticOperationalFailuresStopAtOwningPort(t *testing.T) {
	t.Parallel()

	resolveFake := mustFake(t, fakeworkflow.ResolveError)
	reference, err := resolveFake.Reference()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveFake.Resolve(context.Background(), reference); !errors.Is(err, fakeworkflow.ErrResolve) {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got := resolveFake.Calls(); !reflect.DeepEqual(got, []string{"resolve"}) {
		t.Fatalf("resolve calls = %v", got)
	}

	acquireFake := mustFake(t, fakeworkflow.AcquireError)
	reference, err = acquireFake.Reference()
	if err != nil {
		t.Fatal(err)
	}
	identity, err := acquireFake.Resolve(context.Background(), reference)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireFake.Acquire(context.Background(), identity); !errors.Is(err, fakeworkflow.ErrAcquire) {
		t.Fatalf("Acquire() error = %v", err)
	}
	if got := acquireFake.Calls(); !reflect.DeepEqual(got, []string{"resolve", "acquire"}) {
		t.Fatalf("acquire calls = %v", got)
	}
}

func TestSyntheticPortsRejectCancellation(t *testing.T) {
	t.Parallel()

	fake := mustFake(t, fakeworkflow.Safe)
	reference, err := fake.Reference()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fake.Resolve(ctx, reference); !errors.Is(err, context.Canceled) {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func mustFake(t *testing.T, scenario fakeworkflow.Scenario) *fakeworkflow.Ports {
	t.Helper()
	fake, err := fakeworkflow.New(scenario)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return fake
}
