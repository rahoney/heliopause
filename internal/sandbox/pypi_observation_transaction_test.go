package sandbox

import (
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"strings"
	"testing"
	"time"
)

func testObservationPolicy(now time.Time) observationResourcePolicy {
	return observationResourcePolicy{
		cpuCeilingUsec:  30_000_000,
		pollInterval:    50 * time.Millisecond,
		stopBound:       3 * time.Second,
		uncertaintyUsec: 100_000,
		parallelism:     1,
		maxEvents:       100,
		maxBytes:        1000,
		maxUnits:        2,
		maxLaunches:     4,
		wallDeadline:    now.Add(time.Minute),
	}
}

func TestObservationTransactionCannotSkipOrSubstituteUnit(t *testing.T) {
	now := time.Now()
	transaction, err := newObservationTransaction(testObservationPolicy(now), []observationUnit{
		{id: "a", kind: observationDirectImport, candidate: "alpha"},
		{id: "b", kind: observationDirectImport, candidate: "beta"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	transaction.preparationOK, transaction.anchorAlive = true, true
	if err := transaction.sampleCPU(100, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	containerA := strings.Repeat("a", 64)
	if err := transaction.beginUnit("a", containerA, now.Add(2*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := transaction.finishUnit(externalUnitEvidence{
		unitID: "a", containerID: containerA, terminalOutcome: observationZeroExit,
		observerComplete: true, containerGone: true, cgroupDrained: true,
		closureUnchanged: true, events: 2, bytes: 20, cumulativeCPUUsec: 1000,
	}, now.Add(3*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	transaction.anchorAlive, transaction.cleanupOK = false, true
	if err := transaction.finalize(1000, now.Add(4*time.Millisecond)); err == nil {
		t.Fatal("missing controller-owned unit finalized")
	}
	transaction.anchorAlive = true
	if err := transaction.beginUnit("b", strings.Repeat("b", 64), now.Add(4*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := transaction.beginUnit("a", containerA, now.Add(5*time.Millisecond)); err == nil {
		t.Fatal("duplicate or substituted unit launched")
	}
}

func TestObservationTransactionReserveAndCumulativeLimits(t *testing.T) {
	now := time.Now()
	policy := testObservationPolicy(now)
	transaction, err := newObservationTransaction(policy, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if transaction.reserveUsec != 3_150_000 {
		t.Fatalf("reserve = %d", transaction.reserveUsec)
	}
	if err := transaction.sampleCPU(100, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := transaction.sampleCPU(policy.cpuCeilingUsec-transaction.reserveUsec, now.Add(2*time.Millisecond)); err == nil {
		t.Fatal("CPU threshold did not terminate transaction")
	}
	late, err := newObservationTransaction(policy, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := late.sampleCPU(100, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := late.sampleCPU(200, now.Add(100*time.Millisecond)); err == nil {
		t.Fatal("delayed sample accepted")
	}
	decreased, err := newObservationTransaction(policy, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := decreased.sampleCPU(100, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := decreased.sampleCPU(99, now.Add(2*time.Millisecond)); err == nil {
		t.Fatal("decreasing cgroup counter accepted")
	}
}

func TestObservationTransitionUsesReservedIntervalWithoutResettingCPU(t *testing.T) {
	now := time.Now()
	policy := testObservationPolicy(now)
	transaction, err := newObservationTransaction(policy, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.sampleCPU(100, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	started := now.Add(2 * time.Millisecond)
	if err := transaction.sampleCPUAfterTransition(200, started, started.Add(2*time.Second)); err != nil {
		t.Fatalf("bounded lifecycle accounting: %v", err)
	}
	if transaction.usageUsec != 200 {
		t.Fatalf("cumulative CPU reset: %d", transaction.usageUsec)
	}
	if err := transaction.sampleCPUAfterTransition(201, started.Add(2*time.Second), started.Add(6*time.Second)); err == nil {
		t.Fatal("transition exceeding the reserved stop bound qualified")
	}
}

func TestFinalAccountingRemainsValidDuringArtifactFreeVolumeDisposal(t *testing.T) {
	now := time.Now()
	policy := testObservationPolicy(now)
	transaction, err := newObservationTransaction(policy, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	transaction.preparationOK, transaction.cleanupOK = true, true
	if err := transaction.sampleCPU(100, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := transaction.finalize(150, now.Add(5*time.Second)); err != nil {
		t.Fatalf("trusted final counter was rejected after artifact-free disposal: %v", err)
	}
	if err := transaction.finalize(policy.cpuCeilingUsec+1, now.Add(5*time.Second)); err == nil {
		t.Fatal("final CPU above the absolute transaction ceiling qualified")
	}
}

func TestPythonObservationTraceBudgetCannotResetAcrossStreams(t *testing.T) {
	ledger, err := newObservationTraceLedger("pypi-wheel")
	if err != nil {
		t.Fatal(err)
	}
	ledger.maxEvents, ledger.maxBytes = 3, 12
	if err := ledger.charge(helperRecord{Kind: "container-start"}, 3); err != nil {
		t.Fatal(err)
	}
	count := uint64(2)
	if err := ledger.charge(helperRecord{Kind: "filesystem-workspace-access", Count: &count}, 5); err != nil {
		t.Fatal(err)
	}
	if ledger.events != 3 || ledger.bytes != 8 {
		t.Fatalf("cumulative observer usage = %d events/%d bytes", ledger.events, ledger.bytes)
	}
	if err := ledger.charge(helperRecord{Kind: "container-start"}, 1); err == nil {
		t.Fatal("a later stream replenished the event authorization")
	}
	ledger.maxEvents = 4
	if err := ledger.charge(helperRecord{Kind: "container-start"}, 5); err == nil {
		t.Fatal("a later stream exceeded the byte authorization")
	}
}

func TestObservationTransactionRejectsUnknownUnitAndNonzeroOutcome(t *testing.T) {
	now := time.Now()
	if _, err := newObservationTransaction(testObservationPolicy(now), []observationUnit{{id: "unknown", kind: "UNKNOWN", candidate: "alpha"}}, now); err == nil {
		t.Fatal("unknown unit kind admitted")
	}
	transaction, err := newObservationTransaction(testObservationPolicy(now), []observationUnit{{id: "a", kind: observationDirectImport, candidate: "alpha"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	transaction.preparationOK, transaction.anchorAlive = true, true
	if err := transaction.sampleCPU(1, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	container := strings.Repeat("a", 64)
	if err := transaction.beginUnit("a", container, now.Add(2*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := transaction.finishUnit(externalUnitEvidence{
		unitID: "a", containerID: container, terminalOutcome: observationNonzeroExit,
		observerComplete: true, containerGone: true, cgroupDrained: true,
		closureUnchanged: true, cumulativeCPUUsec: 2,
	}, now.Add(3*time.Millisecond)); err == nil {
		t.Fatal("nonzero external outcome qualified")
	}
}

func TestObservationCommandTerminalDoesNotRelaxTrustedEvidence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		coverage artifactpypi.ObservationCoverage
		terminal observationTerminalOutcome
		broken   string
		accept   bool
	}{
		{"command zero", artifactpypi.PostInstallCommandObservation, observationZeroExit, "", true},
		{"command nonzero", artifactpypi.PostInstallCommandObservation, observationNonzeroExit, "", true},
		{"required nonzero", artifactpypi.RequiredObservation, observationNonzeroExit, "", false},
		{"command signal", artifactpypi.PostInstallCommandObservation, observationSignaled, "", false},
		{"command timeout", artifactpypi.PostInstallCommandObservation, observationTimedOut, "", false},
		{"observer", artifactpypi.PostInstallCommandObservation, observationNonzeroExit, "observer", false},
		{"drain", artifactpypi.PostInstallCommandObservation, observationNonzeroExit, "drain", false},
		{"accounting", artifactpypi.PostInstallCommandObservation, observationNonzeroExit, "accounting", false},
		{"cleanup", artifactpypi.PostInstallCommandObservation, observationNonzeroExit, "cleanup", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			tx, err := newObservationTransaction(testObservationPolicy(now), []observationUnit{{id: "u", kind: observationDirectImport, candidate: "pkg.cli", coverage: tc.coverage, ownerDigest: strings.Repeat("a", 64)}}, now)
			if err != nil {
				t.Fatal(err)
			}
			tx.preparationOK, tx.anchorAlive = true, true
			if err := tx.sampleCPU(100, now.Add(time.Millisecond)); err != nil {
				t.Fatal(err)
			}
			container := strings.Repeat("b", 64)
			if err := tx.beginUnit("u", container, now.Add(2*time.Millisecond)); err != nil {
				t.Fatal(err)
			}
			e := externalUnitEvidence{unitID: "u", containerID: container, terminalOutcome: tc.terminal, observerComplete: true, containerGone: true, cgroupDrained: true, closureUnchanged: true, events: 2, bytes: 20, cumulativeCPUUsec: 1000}
			switch tc.broken {
			case "observer":
				e.observerComplete = false
			case "drain":
				e.cgroupDrained = false
			case "accounting":
				e.cumulativeCPUUsec = 1
			}
			finished := tx.finishUnit(e, now.Add(3*time.Millisecond))
			tx.anchorAlive, tx.cleanupOK = false, tc.broken != "cleanup"
			final := tx.finalize(1000, now.Add(4*time.Millisecond))
			if (finished == nil && final == nil) != tc.accept {
				t.Fatalf("finish=%v final=%v", finished, final)
			}
			if tc.accept {
				got := tx.commandObservations()
				if len(got) != 1 || got[0].ZeroExit != (tc.terminal == observationZeroExit) {
					t.Fatal(got)
				}
			}
		})
	}
}
