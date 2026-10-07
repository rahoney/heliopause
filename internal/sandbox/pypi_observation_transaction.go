package sandbox

import (
	"errors"
	"fmt"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"math"
	"sort"
	"sync"
	"time"
)

// observationUnit is frozen by the controller before any artifact execution.
// Neither process output nor Python state is an input to this ledger.
type observationUnit struct {
	id          string
	kind        observationUnitKind
	candidate   string
	program     string
	ownerDigest string
	coverage    artifactpypi.ObservationCoverage
}

type observationUnitKind string

const (
	observationDirectImport     observationUnitKind = "DIRECT_IMPORT"
	observationActivePTHHook    observationUnitKind = "ACTIVE_PTH_HOOK"
	observationInstalledStartup observationUnitKind = "INSTALLED_STARTUP_SCENARIO"
)

type observationTerminalOutcome string

const (
	observationZeroExit    observationTerminalOutcome = "ZERO_EXIT"
	observationNonzeroExit observationTerminalOutcome = "NONZERO_EXIT"
	observationSignaled    observationTerminalOutcome = "SIGNALED"
	observationTimedOut    observationTerminalOutcome = "TIMED_OUT"
)

type observationUnitState struct {
	unit      observationUnit
	container string
	launched  bool
	finished  bool
	outcome   observationTerminalOutcome
}

// observationResourcePolicy is a single artifact-transaction authorization.
// The CPU reserve is deducted from cpuCeilingUsec, never added to it.
type observationResourcePolicy struct {
	cpuCeilingUsec  uint64
	pollInterval    time.Duration
	stopBound       time.Duration
	uncertaintyUsec uint64
	parallelism     uint64
	maxEvents       uint64
	maxBytes        uint64
	maxUnits        int
	maxLaunches     int
	wallDeadline    time.Time
}

type observationTransaction struct {
	mu              sync.Mutex
	policy          observationResourcePolicy
	reserveUsec     uint64
	expected        map[string]*observationUnitState
	order           []string
	active          string
	launched        int
	usageUsec       uint64
	events          uint64
	bytes           uint64
	lastSample      time.Time
	failed          bool
	terminalFailure bool
	preparationOK   bool
	anchorAlive     bool
	cleanupOK       bool
}

func newObservationTransaction(policy observationResourcePolicy, units []observationUnit, now time.Time) (*observationTransaction, error) {
	if policy.cpuCeilingUsec == 0 || policy.pollInterval <= 0 || policy.stopBound <= 0 ||
		policy.parallelism == 0 || policy.maxEvents == 0 || policy.maxBytes == 0 ||
		policy.maxUnits < 0 || policy.maxLaunches <= 0 || policy.wallDeadline.IsZero() ||
		!now.Before(policy.wallDeadline) || len(units) > policy.maxUnits {
		return nil, errors.New("python observation transaction policy is invalid")
	}
	// Ceil each component independently so rounding can only shrink the
	// available execution allowance.
	pollUsec := uint64((policy.pollInterval + time.Microsecond - 1) / time.Microsecond)
	stopUsec := uint64((policy.stopBound + time.Microsecond - 1) / time.Microsecond)
	if pollUsec > math.MaxUint64-stopUsec ||
		policy.parallelism > (math.MaxUint64-policy.uncertaintyUsec)/(pollUsec+stopUsec) {
		return nil, errors.New("python observation CPU reserve overflows")
	}
	reserve := policy.parallelism*(pollUsec+stopUsec) + policy.uncertaintyUsec
	if reserve >= policy.cpuCeilingUsec {
		return nil, errors.New("python observation CPU reserve consumes authorization")
	}
	transaction := &observationTransaction{
		policy:      policy,
		reserveUsec: reserve,
		expected:    make(map[string]*observationUnitState, len(units)),
		order:       make([]string, 0, len(units)),
	}
	for _, unit := range units {
		if unit.id == "" || unit.candidate == "" || transaction.expected[unit.id] != nil ||
			(unit.kind != observationDirectImport && unit.kind != observationActivePTHHook && unit.kind != observationInstalledStartup) {
			return nil, errors.New("python observation unit identity is missing or repeated")
		}
		if unit.coverage != artifactpypi.RequiredObservation && (unit.coverage != artifactpypi.PostInstallCommandObservation || unit.kind != observationDirectImport) {
			return nil, errors.New("invalid observation coverage")
		}
		transaction.expected[unit.id] = &observationUnitState{unit: unit}
		transaction.order = append(transaction.order, unit.id)
	}
	sort.Strings(transaction.order)
	return transaction, nil
}

func (t *observationTransaction) sampleCPU(usageUsec uint64, at time.Time) error {
	if t == nil {
		return errors.New("python observation CPU accounting is unavailable")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.sampleCPULocked(usageUsec, at)
}

// sampleCPUAfterTransition accounts for one controller-owned Docker lifecycle
// operation. The controller freezes an external stop bound before starting it;
// the reserve already includes this interval, and no new unit can begin until
// the resulting parent-cgroup sample has been accepted.
func (t *observationTransaction) sampleCPUAfterTransition(usageUsec uint64, started, at time.Time) error {
	return t.accountCPUAfterTransition(usageUsec, started, at, false)
}

// Termination still accounts exact usage after a nonqualifying unit. The
// failed bit is retained; these samples cannot admit or complete any unit.
func (t *observationTransaction) accountCPUAfterTransition(usageUsec uint64, started, at time.Time, termination bool) error {
	if t == nil {
		return errors.New("python observation CPU accounting is unavailable")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if (t.failed && !termination) || t.lastSample.IsZero() || started.Before(t.lastSample) ||
		started.Sub(t.lastSample) > t.policy.pollInterval || at.Before(started) ||
		at.Sub(started) > t.policy.stopBound || usageUsec < t.usageUsec {
		t.failed = true
		return errors.New("python observation lifecycle exceeded its reserved accounting interval")
	}
	t.usageUsec = usageUsec
	t.lastSample = at
	if usageUsec >= t.policy.cpuCeilingUsec-t.reserveUsec {
		t.failed = true
		return errors.New("python observation CPU authorization exhausted")
	}
	return nil
}

func (t *observationTransaction) sampleCPULocked(usageUsec uint64, at time.Time) error {
	return t.accountCPULocked(usageUsec, at, false)
}

func (t *observationTransaction) accountCPULocked(usageUsec uint64, at time.Time, termination bool) error {
	if t == nil {
		return errors.New("python observation CPU accounting is unavailable or late")
	}
	if (t.failed && !termination) || !at.Before(t.policy.wallDeadline) ||
		(!t.lastSample.IsZero() && (at.Before(t.lastSample) || at.Sub(t.lastSample) > t.policy.pollInterval)) ||
		usageUsec < t.usageUsec {
		// Keep the rejecting trusted operands before making failure sticky.
		// These bounded scalars diagnose ordering/latency without exposing
		// artifact output or changing the accounting decision.
		err := fmt.Errorf("python observation CPU accounting is unavailable or late: failed_before=%t termination=%t last_sample_present=%t gap_ns=%d poll_ns=%d wall_remaining_ns=%d usage_us=%d prior_usage_us=%d", t.failed, termination, !t.lastSample.IsZero(), at.Sub(t.lastSample).Nanoseconds(), t.policy.pollInterval.Nanoseconds(), t.policy.wallDeadline.Sub(at).Nanoseconds(), usageUsec, t.usageUsec)
		t.failed = true
		return err
	}
	t.usageUsec = usageUsec
	t.lastSample = at
	if usageUsec >= t.policy.cpuCeilingUsec-t.reserveUsec {
		t.failed = true
		return errors.New("python observation CPU authorization exhausted")
	}
	return nil
}

// beginUnit may only be called after prior runtime drainage and a fresh
// trusted cgroup sample. A process cannot authorize another unit.
func (t *observationTransaction) beginUnit(id, container string, at time.Time) error {
	if t == nil {
		return errors.New("python observation unit admission is unavailable")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.failed || !t.preparationOK || !t.anchorAlive || !t.activeIsEmpty() ||
		!at.Before(t.policy.wallDeadline) || t.lastSample.IsZero() ||
		at.Sub(t.lastSample) > t.policy.pollInterval || t.launched >= t.policy.maxLaunches ||
		!exactObservationContainerID(container) {
		return errors.New("python observation unit admission is unavailable")
	}
	unit := t.expected[id]
	if unit == nil || unit.launched || unit.finished {
		t.failed = true
		return errors.New("python observation unit identity is unexpected or repeated")
	}
	unit.container = container
	unit.launched = true
	t.active = id
	t.launched++
	return nil
}

func (t *observationTransaction) activeIsEmpty() bool { return t.active == "" }

func exactObservationContainerID(id string) bool {
	return len(id) == 64 && containerIDPattern.MatchString(id)
}

type externalUnitEvidence struct {
	unitID            string
	containerID       string
	terminalOutcome   observationTerminalOutcome
	observerComplete  bool
	containerGone     bool
	cgroupDrained     bool
	closureUnchanged  bool
	events            uint64
	bytes             uint64
	cumulativeCPUUsec uint64
}

func (t *observationTransaction) finishUnit(e externalUnitEvidence, at time.Time) error {
	if t == nil {
		return errors.New("python observation unit has no active controller launch")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.failed || t.active == "" {
		return errors.New("python observation unit has no active controller launch")
	}
	unit := t.expected[t.active]
	if unit == nil || e.unitID != t.active || e.containerID != unit.container ||
		(e.terminalOutcome != observationZeroExit && e.terminalOutcome != observationNonzeroExit &&
			e.terminalOutcome != observationSignaled && e.terminalOutcome != observationTimedOut) ||
		!e.observerComplete || !e.containerGone ||
		!e.cgroupDrained || !e.closureUnchanged ||
		e.events > t.policy.maxEvents-t.events || e.bytes > t.policy.maxBytes-t.bytes {
		t.failed = true
		return errors.New("python observation unit evidence is incomplete")
	}
	// A final accounting sample is mandatory. The reserve applies throughout
	// teardown, so even a well observed unit cannot consume the stop margin.
	if err := t.sampleCPULocked(e.cumulativeCPUUsec, at); err != nil {
		return err
	}
	if e.terminalOutcome != observationZeroExit && !(unit.unit.coverage == artifactpypi.PostInstallCommandObservation && e.terminalOutcome == observationNonzeroExit) {
		t.failed = true
		// Only complete external terminal evidence and an accepted final sample
		// reach this state. Periodic samples now account teardown; they cannot
		// restore admission, required coverage or transaction completion.
		t.terminalFailure = true
		return errors.New("python observation unit had a nonqualifying external outcome")
	}
	t.events += e.events
	t.bytes += e.bytes
	unit.finished = true
	unit.outcome = e.terminalOutcome
	t.active = ""
	return nil
}

func (t *observationTransaction) finalize(finalCPUUsec uint64, at time.Time) error {
	if t == nil {
		return errors.New("python observation transaction cannot be finalized")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.failed || t.active != "" || !t.preparationOK ||
		t.anchorAlive || !t.cleanupOK || !at.Before(t.policy.wallDeadline) ||
		t.lastSample.IsZero() || at.Before(t.lastSample) ||
		finalCPUUsec < t.usageUsec || finalCPUUsec > t.policy.cpuCeilingUsec {
		return errors.New("python observation transaction cannot be finalized")
	}
	for _, id := range t.order {
		if !t.expected[id].finished {
			return fmt.Errorf("python observation unit %q was not externally finalized", id)
		}
	}
	return nil
}

func (t *observationTransaction) commandObservations() []PythonCommandObservation {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out []PythonCommandObservation
	for _, id := range t.order {
		state := t.expected[id]
		if state.finished && state.unit.coverage == artifactpypi.PostInstallCommandObservation {
			out = append(out, PythonCommandObservation{Module: state.unit.candidate, UnitID: id, OwnerSHA256: state.unit.ownerDigest, ZeroExit: state.outcome == observationZeroExit})
		}
	}
	return out
}
