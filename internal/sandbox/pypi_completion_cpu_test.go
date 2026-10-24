package sandbox

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type unitCompletionCPUClient struct {
	terminationDiagnosticClient
	read func(context.Context) (ObservationResourceLease, error)
}

func (c unitCompletionCPUClient) Read(ctx context.Context, _ string) (ObservationResourceLease, error) {
	return c.read(ctx)
}

func unitCompletionCPUFixture(t *testing.T) (*observationTransaction, externalUnitEvidence) {
	t.Helper()
	now := time.Now()
	policy := testObservationPolicy(now)
	policy.pollInterval = time.Second
	ledger, err := newObservationTransaction(policy, []observationUnit{{id: "a", kind: observationDirectImport, candidate: "alpha"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	ledger.preparationOK, ledger.anchorAlive = true, true
	if err := ledger.sampleCPU(100, now); err != nil {
		t.Fatal(err)
	}
	container := strings.Repeat("a", 64)
	if err := ledger.beginUnit("a", container, now); err != nil {
		t.Fatal(err)
	}
	return ledger, externalUnitEvidence{unitID: "a", containerID: container, terminalOutcome: observationZeroExit, observerComplete: true, containerGone: true, cgroupDrained: true, closureUnchanged: true}
}

func TestPythonCPUCompletionSerializesFinalReadAndAcceptance(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ledger, evidence := unitCompletionCPUFixture(t)
		entered, release := make(chan struct{}), make(chan struct{})
		var reads atomic.Int32
		client := unitCompletionCPUClient{read: func(ctx context.Context) (ObservationResourceLease, error) {
			if reads.Add(1) == 1 {
				close(entered)
				select {
				case <-release:
				case <-ctx.Done():
					return ObservationResourceLease{}, ctx.Err()
				}
				return ObservationResourceLease{UsageUsec: 100}, nil
			}
			return ObservationResourceLease{UsageUsec: 101}, nil
		}}
		watch := &observationCPUWatch{client: client, transaction: "transaction", ledger: ledger}
		type result struct {
			sampled bool
			err     error
		}
		finished := make(chan result, 1)
		go func() {
			sampled, err := watch.finishUnit(context.Background(), evidence)
			finished <- result{sampled, err}
		}()
		<-entered
		// A periodic actor uses this same mutex before its real Read/acceptance.
		// The barrier makes the old concurrent read deterministic without sleeps.
		concurrent := watch.sampleMu.TryLock()
		var periodicErr error
		if concurrent {
			periodicErr = watch.sampleAccountingLocked(context.Background(), false, true)
			watch.sampleMu.Unlock()
		}
		close(release)
		got := <-finished
		if concurrent || periodicErr != nil || !got.sampled || got.err != nil || ledger.failed || !ledger.expected["a"].finished || ledger.usageUsec != 100 {
			t.Fatalf("completion allowed a newer periodic sample before its stale final read: concurrent=%t periodic=%v sampled=%t finish=%v failed=%t usage=%d", concurrent, periodicErr, got.sampled, got.err, ledger.failed, ledger.usageUsec)
		}
		if err := watch.sample(context.Background()); err != nil || reads.Load() != 2 || ledger.usageUsec != 101 {
			t.Fatal("periodic accounting could not continue after completion", err, reads.Load(), ledger.usageUsec)
		}
	})
}

func TestPythonCPUCompletionRetainsFailureBoundaries(t *testing.T) {
	for _, failure := range []string{"read", "deadline", "cancelled", "regressed", "reserve", "incomplete", "terminal", "failed", "transition"} {
		t.Run(failure, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ledger, evidence := unitCompletionCPUFixture(t)
				readFailure := errors.New("trusted counter missing")
				var reads int
				client := unitCompletionCPUClient{read: func(ctx context.Context) (ObservationResourceLease, error) {
					reads++
					switch failure {
					case "read":
						return ObservationResourceLease{}, readFailure
					case "deadline":
						<-ctx.Done()
						return ObservationResourceLease{}, ctx.Err()
					case "regressed":
						return ObservationResourceLease{UsageUsec: 99}, nil
					case "reserve":
						return ObservationResourceLease{UsageUsec: ledger.policy.cpuCeilingUsec - ledger.reserveUsec}, nil
					}
					return ObservationResourceLease{UsageUsec: 100}, nil
				}}
				watch := &observationCPUWatch{client: client, transaction: "transaction", ledger: ledger}
				ctx := context.Background()
				switch failure {
				case "cancelled":
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				case "incomplete":
					evidence.observerComplete = false
				case "terminal":
					evidence.terminalOutcome = observationNonzeroExit
				case "failed":
					ledger.failed = true
				case "transition":
					watch.transition = time.Now()
				}
				sampled, err := watch.finishUnit(ctx, evidence)
				readRejected := failure == "read" || failure == "deadline" || failure == "cancelled" || failure == "transition"
				if err == nil || sampled == readRejected || ledger.expected["a"].finished {
					t.Fatal("failed final accounting completed a unit or changed error category", sampled, err)
				}
				if failure == "read" && !errors.Is(err, readFailure) || failure == "deadline" && !errors.Is(err, context.DeadlineExceeded) || failure == "cancelled" && !errors.Is(err, context.Canceled) {
					t.Fatal("read cause lost", err)
				}
				if (failure == "cancelled" || failure == "transition") && reads != 0 {
					t.Fatal("invalid admission read the counter")
				}
				if !readRejected && !ledger.failed {
					t.Fatal("rejected outcome repaired sticky failure")
				}
			})
		})
	}
}
