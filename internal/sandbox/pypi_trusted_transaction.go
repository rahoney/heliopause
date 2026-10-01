package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func pythonLiteral(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func pythonClosureFailure(sessionID domain.SandboxSessionID, phase string, cause error) (domain.SandboxResult, error) {
	result, err := pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLOSURE_INVALID")
	return result, errors.Join(err, fmt.Errorf("python closure %s: %w", phase, cause))
}

// A hook experiment executes only its selected statement. The separate
// startup scenario executes every statement in canonical installed order.
// Both run with -I -S, so ambient site startup cannot run implicitly.
func pythonPTHProgram(lines []artifactpypi.SiteHookLine, selected *artifactpypi.PlannedObservationUnit) string {
	var program strings.Builder
	program.WriteString("import os, sys\nroot = '/haa-site'\nsys.path.insert(0, root)\n")
	program.WriteString("def add_path(value):\n    candidate = os.path.realpath(os.path.join(root, value))\n    if os.path.commonpath((root, candidate)) != root or not os.path.isdir(candidate):\n        raise RuntimeError('active site path is invalid')\n    if candidate not in sys.path:\n        sys.path.append(candidate)\n")
	for _, line := range lines {
		if selected != nil && (line.File > selected.HookFile || line.File == selected.HookFile && line.Line > selected.HookLine) {
			break
		}
		if line.Path != "" {
			program.WriteString("add_path(" + pythonLiteral(line.Path) + ")\n")
		} else if selected == nil || line.File == selected.HookFile && line.Line == selected.HookLine {
			program.WriteString("exec(" + pythonLiteral(line.Statement) + ")\n")
		}
	}
	return program.String()
}

func freezePythonObservationUnits(plan artifactpypi.ObservationPlan, closureID string) ([]observationUnit, error) {
	if closureID == "" {
		return nil, errors.New("python observation closure identity is absent")
	}
	units := make([]observationUnit, 0, len(plan.Units))
	for index, declared := range plan.Units {
		unit := observationUnit{kind: observationUnitKind(declared.Kind), candidate: declared.Candidate}
		switch declared.Kind {
		case artifactpypi.DirectImportUnit:
			unit.program = pythonDirectObservation
		case artifactpypi.ActivePTHHookUnit:
			unit.candidate = fmt.Sprintf("%s:%d", declared.HookFile, declared.HookLine)
			unit.program = pythonPTHProgram(plan.SiteHookLines, &declared)
		case artifactpypi.InstalledStartupUnit:
			unit.candidate = "installed-site-startup"
			unit.program = pythonPTHProgram(plan.SiteHookLines, nil)
		default:
			return nil, errors.New("python observation unit kind is unsupported")
		}
		identity := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s\x00%s\x00%s", closureID, index, unit.kind, unit.candidate, unit.program)))
		unit.id = string(unit.kind) + ":" + hex.EncodeToString(identity[:])
		units = append(units, unit)
	}
	return units, nil
}

// A declarative .pth addition may refer to a directory installed by another
// authenticated wheel. Validate it against the final closure, rather than the
// wheel that owns the hook. Runtime realpath checks still reject path escape.
func validatePTHPathsInClosure(plan artifactpypi.ObservationPlan, manifest closureManifest) error {
	for _, line := range plan.SiteHookLines {
		if line.Path == "" {
			continue
		}
		prefix := line.Path + "/"
		found := false
		for destination, file := range manifest.files {
			if file.scheme == artifactpypi.SchemeSite && strings.HasPrefix(destination, prefix) {
				found = true
				break
			}
		}
		if !found {
			return errors.New("active site path is absent from authenticated closure")
		}
	}
	return nil
}

// pythonDirectObservation contains no completion token. A zero exit is only
// the externally observed terminal outcome of this one controller-owned unit.
// Isolated mode suppresses ambient user/site configuration; -S suppresses
// automatic site startup, which is separately observed when applicable.
const pythonDirectObservation = `import importlib, importlib.machinery, os, sys
root = '/haa-site'
name = sys.argv[1]
parts = name.split('.')
if not parts or any(not part.isidentifier() for part in parts):
    raise RuntimeError('invalid target import identity')
if parts[0] in sys.modules:
    raise RuntimeError('target already present in interpreter startup')
search = [root]
for depth in range(len(parts)):
    spec = importlib.machinery.PathFinder.find_spec('.'.join(parts[:depth + 1]), search)
    if spec is None:
        raise RuntimeError('target not found in authenticated site')
    if spec.origin not in (None, 'namespace') and os.path.commonpath((root, os.path.realpath(spec.origin))) != root:
        raise RuntimeError('target origin outside authenticated site')
    locations = spec.submodule_search_locations
    if locations is not None and (not locations or any(os.path.commonpath((root, os.path.realpath(p))) != root for p in locations)):
        raise RuntimeError('target namespace outside authenticated site')
    if depth + 1 < len(parts):
        if locations is None:
            raise RuntimeError('target parent is not a package')
        search = locations
sys.path.insert(0, root)
importlib.import_module(name)
`

type pythonClosureObserver interface {
	StartPythonClosureProfileWithBudget(context.Context, string, string, pythonClosurePhase, *observationTraceLedger) (TraceReader, error)
	AwaitMountAnchors(context.Context, string) error
	observationUsage(*observationTraceLedger) (uint64, uint64, error)
}

type pythonPhaseRuntime struct {
	id         string
	phase      pythonObservationPhase
	trace      TraceReader
	registered bool
	collected  bool
}

type pythonTransactionExecutor struct {
	backend     *PythonDynamicBackend
	observer    pythonClosureObserver
	resources   ObservationResourceClient
	volume      closureVolume
	manifest    closureManifest
	installed   closureInstallation
	transaction string
	profile     string
	policy      artifactpypi.ResourcePolicy
	ledger      *observationTransaction
	traceLedger *observationTraceLedger
	lease       ObservationResourceLease
	mu          sync.Mutex
	runtimes    map[string]*pythonPhaseRuntime
	anchor      string
	created     bool
	volumeMade  bool
	allObserved []domain.SandboxObservation
	watch       *observationCPUWatch
	finalCPU    uint64
}

func observationPolicyForPython(policy artifactpypi.ResourcePolicy, profile string, unitCount int, now time.Time) (observationResourcePolicy, error) {
	if policy.RuntimeCPUSecs() <= 0 || unitCount < 0 || unitCount > policy.MaxObservationImportsPerArtifact() ||
		policy.Duration() <= 0 || uint64(policy.RuntimeCPUSecs()) > math.MaxUint64/1_000_000 {
		return observationResourcePolicy{}, errors.New("python transaction policy is invalid")
	}
	budget := traceBudgetForProfile(profile)
	return observationResourcePolicy{
		cpuCeilingUsec: uint64(policy.RuntimeCPUSecs()) * 1_000_000,
		pollInterval:   time.Second, stopBound: 7 * time.Second,
		// The artifact runtime is pinned to one CPU. The trusted keeper can
		// briefly run on another CPU, so the stopping reserve covers both.
		uncertaintyUsec: 1_000_000, parallelism: 2,
		maxEvents: uint64(budget.events), maxBytes: budget.bytes,
		maxUnits:     policy.MaxObservationImportsPerArtifact(),
		maxLaunches:  2*policy.MaxObservationImportsPerArtifact() + 4,
		wallDeadline: now.Add(policy.Duration()),
	}, nil
}

func (b *PythonDynamicBackend) executeTrustedTransaction(ctx context.Context, sessionID domain.SandboxSessionID,
	artifact domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, closure []domain.AcquiredArtifact) (result domain.SandboxResult, resultErr error) {
	if b == nil || b.resources == nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_TRUSTED_TRANSACTION_UNAVAILABLE")
	}
	observer, ok := b.observer.(pythonClosureObserver)
	if !ok {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_OBSERVER_FAILED")
	}
	if !plan.Admissible() {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_SURFACE_UNSUPPORTED")
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_RUNTIME_UNAVAILABLE")
	}
	profile, err := pythonDynamicObserverProfile(artifactpypi.RootSourceProfileNameFromContext(ctx))
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_SETUP_FAILED")
	}
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	manifest, err := b.introducer.buildClosureManifest(ctx, closure, policy)
	if err != nil {
		return pythonClosureFailure(sessionID, "build authenticated manifest", err)
	}
	owner := artifact.Identity().Source().String() + ":" + artifact.Identity().Name() + ":" + artifact.Identity().Version() + ":" + artifact.Digest().String()
	fromBytes, ok := manifest.inspections[owner]
	if !ok {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLOSURE_INVALID")
	}
	exactPlan, err := artifactpypi.BuildObservationPlan(fromBytes, policy)
	expectedPlan, encodeExpectedErr := json.Marshal(exactPlan)
	providedPlan, encodeProvidedErr := json.Marshal(plan)
	if err != nil || encodeExpectedErr != nil || encodeProvidedErr != nil || !bytes.Equal(expectedPlan, providedPlan) {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_PLAN_MISMATCH")
	}
	if err := validatePTHPathsInClosure(plan, manifest); err != nil {
		return pythonClosureFailure(sessionID, "validate startup paths", err)
	}
	units, err := freezePythonObservationUnits(plan, manifest.identity)
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_RESOURCE_POLICY_INVALID")
	}
	now := time.Now()
	resourcePolicy, err := observationPolicyForPython(policy, profile, len(units), now)
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_RESOURCE_POLICY_INVALID")
	}
	ledger, err := newObservationTransaction(resourcePolicy, units, now)
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_RESOURCE_POLICY_INVALID")
	}
	traceLedger, err := newObservationTraceLedger(profile)
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_OBSERVER_FAILED")
	}
	runCtx, cancel := context.WithDeadline(ctx, resourcePolicy.wallDeadline)
	defer cancel()
	transaction := &pythonTransactionExecutor{
		backend: b, observer: observer, resources: b.resources, manifest: manifest,
		transaction: sessionID.String(), profile: profile, policy: policy,
		ledger: ledger, traceLedger: traceLedger, runtimes: make(map[string]*pythonPhaseRuntime),
	}
	// Cleanup is part of the security decision. A successful experiment is
	// downgraded if any runtime, observer stream, cgroup or volume remains.
	defer func() {
		cleanupErr := transaction.cleanup()
		if cleanupErr != nil {
			priorErr := resultErr
			result, resultErr = pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLEANUP_FAILED")
			resultErr = errors.Join(priorErr, resultErr, fmt.Errorf("python transaction cleanup: %w", cleanupErr))
			return
		}
		if resultErr != nil {
			return
		}
		if result.Status() == domain.SandboxCompleted {
			if transaction.ledger.finalize(transaction.finalCPU, time.Now()) != nil {
				result, resultErr = pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_TRUSTED_TRANSACTION_INCOMPLETE")
				return
			}
			completed, err := domain.NewSandboxObservation(domain.ObservationProcess, "python-observation-experiment-completed")
			if err != nil {
				result, resultErr = pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_TRUSTED_TRANSACTION_INCOMPLETE")
				return
			}
			transaction.allObserved = append(transaction.allObserved, completed)
			result, resultErr = domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", transaction.allObserved)
		}
	}()
	lease, err := b.resources.Create(runCtx, transaction.transaction, artifactpypi.RootSourceProfileNameFromContext(ctx))
	if err != nil || lease.CgroupParent == "" {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_ACCOUNTING_UNAVAILABLE")
	}
	transaction.created, transaction.lease = true, lease
	volume, err := createClosureVolume(runCtx, b.runner, transaction.transaction, manifest.identity, policy.RuntimeTmpfs())
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLOSURE_UNAVAILABLE")
	}
	transaction.volume, transaction.volumeMade = volume, true
	preparation, err := transaction.startPhase(runCtx, phasePreparation, 0)
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_PREPARATION_FAILED")
	}
	watch, watchErr := startObservationCPUWatch(runCtx, b.resources, transaction.transaction, ledger, cancel, func(stopCtx context.Context) error {
		return transaction.abortAll(stopCtx)
	})
	if watchErr != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_ACCOUNTING_UNAVAILABLE")
	}
	transaction.watch = watch
	validated, err := b.introducer.validatedWheelDestinations(artifact, closure)
	if err != nil {
		return pythonClosureFailure(sessionID, "validate wheel destinations", err)
	}
	wheelPaths := make([]string, 0, len(validated))
	for _, item := range validated {
		if err := b.introducer.introduceWheelAt(runCtx, preparation.id, item.artifact, item.destination); err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_INTRODUCTION_FAILED")
		}
		wheelPaths = append(wheelPaths, item.destination)
	}
	install := append(boundaryExecArguments(preparation.id, boundaryLaunchMode, "python", "-I", "-B", "-m", "pip", "install",
		"--no-index", "--no-deps", "--no-compile", "--disable-pip-version-check", "--no-cache-dir", "--target", pythonSitePath), wheelPaths...)
	if err := discardCommand(runCtx, b.runner, "docker", install...); err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_INSTALL_FAILED")
	}
	installed, err := verifyClosureInstallation(runCtx, b.runner, preparation.id, manifest, policy.RuntimeTmpfs())
	if err != nil {
		return pythonClosureFailure(sessionID, "verify prepared installation", err)
	}
	transaction.installed = installed
	anchor, err := transaction.startPhase(runCtx, phaseAnchor, 1)
	if err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_ANCHOR_FAILED")
	}
	transaction.anchor = anchor.id
	if err := transaction.volume.verifyAttachments(runCtx, b.runner, map[string]bool{preparation.id: false, anchor.id: true}); err != nil {
		return pythonClosureFailure(sessionID, "verify closure attachments", err)
	}
	if err := transaction.terminatePhase(preparation.id); err != nil {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_PREPARATION_FAILED")
	}
	if after, err := verifyClosureInstallation(runCtx, b.runner, anchor.id, manifest, policy.RuntimeTmpfs()); err != nil || after.identity != installed.identity {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLOSURE_CHANGED")
	}
	ledger.mu.Lock()
	ledger.preparationOK, ledger.anchorAlive = true, true
	ledger.mu.Unlock()
	for index, unit := range units {
		if err := transaction.verifyAnchorAndVolume(runCtx, nil); err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLOSURE_CHANGED")
		}
		probe, err := transaction.startPhase(runCtx, phaseObservation, index+2)
		if err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_SETUP_FAILED")
		}
		if err := transaction.verifyAnchorAndVolume(runCtx, map[string]bool{probe.id: true}); err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLOSURE_CHANGED")
		}
		if err := ledger.beginUnit(unit.id, probe.id, time.Now()); err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_RESOURCE_POLICY_INVALID")
		}
		beforeEvents, beforeBytes, err := observer.observationUsage(traceLedger)
		if err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_OBSERVER_FAILED")
		}
		arguments := boundaryExecArguments(probe.id, boundaryPythonHandoffMode, "python", "-I", "-S", "-B", "-c", unit.program)
		if unit.kind == observationDirectImport {
			arguments = append(arguments, unit.candidate)
		}
		commandErr := discardCommand(runCtx, b.runner, "docker", arguments...)
		outcome := observationZeroExit
		if commandErr != nil {
			outcome = observationNonzeroExit
		}
		if err := transaction.terminatePhase(probe.id); err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE")
		}
		if err := transaction.verifyAnchorAndVolume(runCtx, nil); err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_CLOSURE_CHANGED")
		}
		afterEvents, afterBytes, err := observer.observationUsage(traceLedger)
		if err != nil || afterEvents < beforeEvents || afterBytes < beforeBytes {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_OBSERVER_FAILED")
		}
		cpuLease, err := b.resources.Read(runCtx, transaction.transaction)
		if err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_ACCOUNTING_UNAVAILABLE")
		}
		if err := ledger.finishUnit(externalUnitEvidence{
			unitID: unit.id, containerID: probe.id, terminalOutcome: outcome,
			observerComplete: true, containerGone: true, cgroupDrained: true,
			closureUnchanged: true, events: afterEvents - beforeEvents, bytes: afterBytes - beforeBytes,
			cumulativeCPUUsec: cpuLease.UsageUsec,
		}, time.Now()); err != nil {
			return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE")
		}
	}
	return domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", nil)
}

func (t *pythonTransactionExecutor) startPhase(ctx context.Context, phase pythonObservationPhase, ordinal int) (started *pythonPhaseRuntime, resultErr error) {
	if t != nil && t.watch != nil {
		if err := t.watch.beginTransition(ctx); err != nil {
			return nil, err
		}
		defer func() {
			if err := t.watch.endTransition(context.Background()); err != nil {
				resultErr = errors.Join(resultErr, err)
			}
		}()
	}
	if t == nil || ctx == nil || t.volume.verify(ctx, t.backend.runner) != nil {
		return nil, errors.New("python transaction volume is unavailable")
	}
	args, err := pythonObservationCreateArguments(t.transaction, ordinal, phase, t.volume, t.lease, t.policy)
	if err != nil {
		return nil, err
	}
	created, err := t.backend.runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		return nil, errors.New("python observation runtime creation failed")
	}
	runtime := &pythonPhaseRuntime{id: id, phase: phase}
	t.mu.Lock()
	t.runtimes[id] = runtime
	t.mu.Unlock()
	readOnly := phase != phasePreparation
	if err := t.volume.verifyContainerMount(ctx, t.backend.runner, id, readOnly); err != nil {
		return nil, err
	}
	observerPhase := pythonClosureObservation
	if phase == phasePreparation {
		observerPhase = pythonClosurePreparation
	} else if phase == phaseAnchor {
		observerPhase = pythonClosureAnchor
	}
	trace, err := t.observer.StartPythonClosureProfileWithBudget(ctx, id, t.profile, observerPhase, t.traceLedger)
	if err != nil {
		return nil, err
	}
	runtime.trace = trace
	startErr := discardCommand(ctx, t.backend.runner, "docker", "start", id)
	if startErr != nil &&
		verifyObservationRuntimeAlive(ctx, t.backend.runner, id) != nil {
		return nil, errors.New("python observation runtime startup is incomplete")
	}
	lease, err := t.resources.Register(ctx, t.transaction, id, string(phase))
	if err != nil || lease.CgroupParent != t.lease.CgroupParent || lease.CPU != t.lease.CPU {
		return nil, errors.New("python observation runtime cgroup admission failed")
	}
	runtime.registered = true
	if startErr != nil || awaitBoundaryHelper(ctx, t.backend.runner, id) != nil ||
		t.observer.AwaitMountAnchors(ctx, id) != nil ||
		verifyObservationRuntimeAlive(ctx, t.backend.runner, id) != nil {
		return nil, errors.New("python observation runtime startup is incomplete")
	}
	return runtime, nil
}

func (t *pythonTransactionExecutor) terminatePhase(id string) (resultErr error) {
	if t == nil || !exactObservationContainerID(id) {
		return errors.New("python observation runtime cleanup identity is invalid")
	}
	if t.watch != nil {
		if err := t.watch.beginTransition(context.Background()); err != nil {
			return err
		}
		defer func() {
			if err := t.watch.endTransition(context.Background()); err != nil {
				resultErr = errors.Join(resultErr, err)
			}
		}()
	}
	t.mu.Lock()
	runtime := t.runtimes[id]
	t.mu.Unlock()
	if runtime == nil || runtime.collected || runtime.trace == nil || !runtime.registered {
		return errors.New("python observation runtime cleanup evidence is unavailable")
	}
	type traceResult struct {
		observations []domain.SandboxObservation
		limitation   string
		diagnostic   TraceDiagnostic
	}
	traceDone := make(chan traceResult, 1)
	go func() {
		traceCtx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		observations, limitation, diagnostic := collectTraceDiagnostic(traceCtx, runtime.trace)
		traceDone <- traceResult{observations, limitation, diagnostic}
	}()
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	lease, err := t.resources.Terminate(cleanupCtx, t.transaction, id)
	cancel()
	trace := <-traceDone
	if err != nil || trace.limitation != "" || !trace.diagnostic.SessionComplete || lease.CgroupParent != t.lease.CgroupParent {
		return errors.New("python observation runtime drainage or observer evidence is incomplete")
	}
	t.mu.Lock()
	runtime.collected = true
	delete(t.runtimes, id)
	t.allObserved = append(t.allObserved, trace.observations...)
	t.mu.Unlock()
	return nil
}

func (t *pythonTransactionExecutor) abortAll(ctx context.Context) error {
	if t == nil || ctx == nil {
		return errors.New("python observation abort is unavailable")
	}
	t.mu.Lock()
	ids := make([]string, 0, len(t.runtimes))
	for id := range t.runtimes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		left, right := t.runtimes[ids[i]].phase, t.runtimes[ids[j]].phase
		if left == right {
			return ids[i] < ids[j]
		}
		if left == phaseAnchor {
			return false
		}
		if right == phaseAnchor {
			return true
		}
		return left == phaseObservation
	})
	t.mu.Unlock()
	var joined error
	for _, id := range ids {
		// A missing resource helper cannot be allowed to delay the controller's
		// exact-ID Docker kill. Both trusted paths run under the same stop bound;
		// missing helper accounting still makes the transaction nonqualifying.
		type terminationResult struct{ err error }
		helperDone := make(chan terminationResult, 1)
		go func(container string) {
			_, err := t.resources.Terminate(ctx, t.transaction, container)
			helperDone <- terminationResult{err: err}
		}(id)
		removeErr := discardCommand(ctx, t.backend.runner, "docker", "rm", "--force", id)
		remaining, inspectErr := t.backend.runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "id="+id, "--format", "{{.ID}}")
		helper := <-helperDone
		if helper.err != nil || (removeErr != nil && (inspectErr != nil || strings.TrimSpace(string(remaining)) != "")) ||
			inspectErr != nil || len(remaining) > 1024 || strings.TrimSpace(string(remaining)) != "" {
			joined = errors.Join(joined, errors.New("python observation abort could not prove exact runtime drainage"))
		}
	}
	return joined
}

func (t *pythonTransactionExecutor) verifyAnchorAndVolume(ctx context.Context, additional map[string]bool) error {
	if t == nil || t.anchor == "" || verifyObservationRuntimeAlive(ctx, t.backend.runner, t.anchor) != nil ||
		t.volume.verifyContainerMount(ctx, t.backend.runner, t.anchor, true) != nil {
		return errors.New("python observation anchor is unavailable")
	}
	attachments := map[string]bool{t.anchor: true}
	for id, readOnly := range additional {
		attachments[id] = readOnly
	}
	return t.volume.verifyAttachments(ctx, t.backend.runner, attachments)
}

func (t *pythonTransactionExecutor) cleanup() error {
	if t == nil {
		return errors.New("python observation cleanup unavailable")
	}
	// Use a fresh bounded context. The request deadline may have expired while
	// artifact code ran; that must not suppress trusted cleanup.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var failed error
	if t.watch != nil {
		// Wait for any emergency abort to finish before ordinary teardown.
		// Otherwise both paths can race to remove the same exact container.
		if err := t.watch.stop(); err != nil {
			failed = errors.Join(failed, err)
		}
	}
	t.mu.Lock()
	ids := make([]string, 0, len(t.runtimes))
	for id := range t.runtimes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		left, right := t.runtimes[ids[i]].phase, t.runtimes[ids[j]].phase
		if left == right {
			return ids[i] < ids[j]
		}
		if left == phaseAnchor {
			return false
		}
		if right == phaseAnchor {
			return true
		}
		return left == phaseObservation
	})
	t.mu.Unlock()
	for _, id := range ids {
		if err := t.terminatePhase(id); err != nil {
			failed = errors.Join(failed, err)
			// Even an unregistered/uncertain runtime must be removed, but this
			// cannot turn its failed observer/cgroup evidence into completion.
			if removeErr := discardCommand(ctx, t.backend.runner, "docker", "rm", "--force", id); removeErr != nil {
				failed = errors.Join(failed, removeErr)
			}
			remaining, inspectErr := t.backend.runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "id="+id, "--format", "{{.ID}}")
			if inspectErr != nil || len(remaining) > 1024 || strings.TrimSpace(string(remaining)) != "" {
				failed = errors.Join(failed, errors.New("python observation runtime absence is unproven"))
			}
		}
	}
	if t.created {
		lease, err := t.resources.Close(ctx, t.transaction)
		if err != nil {
			failed = errors.Join(failed, err)
		} else {
			t.finalCPU = lease.UsageUsec
		}
	}
	// After Close, the privileged helper has proven there are no artifact
	// runtimes in the transaction subtree and has returned its final counter.
	// Volume disposal may be slow for a large CUDA closure, but it cannot grant
	// artifact CPU time; qualification still waits for exact volume removal.
	if t.volumeMade {
		if err := t.volume.verify(ctx, t.backend.runner); err != nil {
			failed = errors.Join(failed, err)
		}
		if err := discardCommand(ctx, t.backend.runner, "docker", "volume", "rm", t.volume.name); err != nil {
			failed = errors.Join(failed, err)
		}
	}
	if failed == nil {
		t.ledger.mu.Lock()
		t.ledger.anchorAlive = false
		t.ledger.cleanupOK = true
		t.ledger.mu.Unlock()
	}
	return failed
}
