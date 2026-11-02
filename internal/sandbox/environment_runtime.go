package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type environmentTraceObserver interface {
	observationPhaseObserver
	StartEnvironmentProfile(context.Context, string, bool, *observationTraceLedger) (TraceReader, error)
}

// This is the existing npm lifecycle's resource owner, extended to two fresh
// workspaces. CPU, wall and trace accounting are the existing common owners.
// Artifact output cannot complete an experiment or replenish a budget.
type environmentInspection struct {
	backend       *Backend
	observer      environmentTraceObserver
	owner         domain.SandboxSessionID
	lease         ObservationResourceLease
	ledger        *observationTransaction
	traceBudget   *observationTraceLedger
	watch         *observationCPUWatch
	policy        artifactpypi.ResourcePolicy
	mu            sync.Mutex
	runtimes      map[string]*pythonPhaseRuntime
	volume        closureVolume
	uncertain     bool
	attempted     bool
	finalCPU      uint64
	events, bytes uint64
	observed      []domain.SandboxObservation
	states        []domain.SandboxEnvironmentState
}

func (b *Backend) executeEnvironmentInspection(ctx context.Context, request domain.SandboxRequest, owner domain.SandboxSessionID) (domain.SandboxResult, error) {
	observer, ok := b.observer.(environmentTraceObserver)
	_, inputOK := b.runner.(inputCommandRunner)
	_, streamOK := b.runner.(streamingDockerOutput)
	_, discardOK := b.runner.(discardCommandRunner)
	if !ok || !inputOK || !streamOK || !discardOK || b.resources == nil {
		return incomplete(owner, "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE")
	}
	if request.Artifact().Identity().Source().String() != "npm" {
		return incomplete(owner, "M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED")
	}
	if err := ctx.Err(); err != nil {
		result, e := incomplete(owner, "M3_DYNAMIC_EXECUTION_FAILED")
		return result, errors.Join(e, err)
	}
	now := time.Now()
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	units := []observationUnit{
		{id: string(minimalEnvironment), kind: observationLifecycle, candidate: string(minimalEnvironment), coverage: artifactpypi.RequiredObservation},
		{id: string(developmentEnvironment), kind: observationLifecycle, candidate: string(developmentEnvironment), coverage: artifactpypi.RequiredObservation},
	}
	accounting, err := observationPolicyForPython(policy, "npm-lifecycle", len(units), now)
	if err != nil {
		return incomplete(owner, "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE")
	}
	accounting.maxUnits = len(units)
	accounting.maxLaunches = 2 * len(units)
	accounting.wallDeadline = now.Add(b.wallTimeout)
	ledger, err := newObservationTransaction(accounting, units, now)
	if err != nil {
		return incomplete(owner, "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE")
	}
	budget, err := newObservationTraceLedger("npm-lifecycle")
	if err != nil {
		return incomplete(owner, "M3_DYNAMIC_OBSERVER_FAILED")
	}
	run, cancel := context.WithDeadline(ctx, accounting.wallDeadline)
	defer cancel()
	operation := &environmentInspection{backend: b, observer: observer, owner: owner, ledger: ledger, traceBudget: budget, policy: policy, runtimes: make(map[string]*pythonPhaseRuntime)}
	code, cause := operation.inspect(run, cancel, request.Artifact())
	// stop() intentionally cancels its private context. Preserve genuine causes
	// before that normal stop, and also retain a late caller cancellation.
	runCause := run.Err()
	cleanupErr := operation.close()
	if cause == nil {
		cause = runCause
	}
	cause = errors.Join(cause, ctx.Err(), cleanupErr)
	if code == "" && (runCause != nil || ctx.Err() != nil) {
		code = "M3_DYNAMIC_TIMEOUT"
	}
	if code == "" && cleanupErr != nil {
		code = "M3_DYNAMIC_CLEANUP_FAILED"
	}
	if code == "" {
		ledger.mu.Lock()
		ledger.cleanupOK = true
		ledger.anchorAlive = false
		ledger.mu.Unlock()
		if err := ledger.finalize(operation.finalCPU, time.Now()); err != nil {
			code = "M3_DYNAMIC_ACCOUNTING_INCOMPLETE"
			cause = errors.Join(cause, err)
		}
	}
	status := domain.SandboxCompleted
	if code != "" {
		status = domain.SandboxIncomplete
	}
	result, resultErr := domain.NewSandboxResult(owner, status, code, operation.observed, operation.states...)
	return result, errors.Join(resultErr, cause)
}

func (n *environmentInspection) inspect(ctx context.Context, cancel context.CancelFunc, artifact domain.AcquiredArtifact) (string, error) {
	n.attempted = true
	lease, err := n.backend.resources.Create(ctx, n.owner.String(), "npm-lifecycle")
	if err != nil || lease.CgroupParent == "" || lease.CPU < 0 {
		return "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE", err
	}
	n.lease = lease
	n.watch, err = startObservationCPUWatch(ctx, n.backend.resources, n.owner.String(), n.ledger, cancel, n.abort)
	if err != nil {
		return "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE", err
	}
	seen := map[string]bool{n.owner.String(): true}
	for _, kind := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
		session, err := n.backend.newSessionID()
		if err != nil || session.String() == "" || seen[session.String()] {
			return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
		}
		seen[session.String()] = true
		profile, err := newInspectionEnvironmentProfile(kind, session)
		if err != nil {
			return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
		}
		if code, err := n.experiment(ctx, artifact, session, profile); code != "" || err != nil {
			return code, err
		}
	}
	fact, err := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
	if err != nil {
		return "M3_DYNAMIC_OBSERVER_FAILED", err
	}
	n.observed = append(n.observed, fact)
	return "", nil
}

func (n *environmentInspection) experiment(ctx context.Context, artifact domain.AcquiredArtifact, session domain.SandboxSessionID, profile inspectionEnvironmentProfile) (string, error) {
	volume, err := createEnvironmentWorkspace(ctx, n.backend.runner, n.owner, profile, n.policy.RuntimeTmpfs())
	n.volume = volume
	if err != nil {
		n.uncertain = volume.name != ""
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	writer, err := n.start(ctx, session, profile, false)
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	payload, err := profile.payload()
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	input := n.backend.runner.(inputCommandRunner)
	if err := input.RunInput(ctx, bytes.NewReader(payload), "docker", "cp", "--archive", "-", writer.id+":/tmp"); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := n.backend.introducer.Introduce(ctx, writer.id, artifact); err != nil {
		return "M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED", err
	}
	anchor, err := n.start(ctx, session, profile, true)
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	if err := n.volume.verifyAttachments(ctx, n.backend.runner, map[string]bool{writer.id: false, anchor.id: true}); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	// Keep each reported change count exact within the existing summary bound.
	limits := environmentStateLimits{entries: int(domain.MaximumObservationSummaryCount), bytes: n.policy.RuntimeTmpfs(), archiveBytes: 2 * n.policy.RuntimeTmpfs()}
	before, err := captureEnvironmentState(ctx, n.backend.runner, anchor.id, limits)
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := profile.verifyPlacement(before); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	original, ok := before.files["artifact.tgz"]
	if !ok || original.kind != tar.TypeReg || original.digest != artifact.Digest().String() || original.size != int64(artifact.SizeBytes()) {
		return "M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED", errEnvironmentState
	}
	n.ledger.mu.Lock()
	n.ledger.preparationOK = true
	n.ledger.anchorAlive = true
	n.ledger.mu.Unlock()
	if err := n.ledger.beginUnit(string(profile.kind), writer.id, time.Now()); err != nil {
		return "M3_DYNAMIC_ACCOUNTING_INCOMPLETE", err
	}
	commandErr := discardCommand(ctx, n.backend.runner, "docker", boundaryExecArguments(writer.id, boundaryLaunchMode, "/bin/sh", "-ceu", npmLifecycleCommand)...)
	outcome := observationZeroExit
	primaryCode := ""
	if commandErr != nil {
		primaryCode = "M3_DYNAMIC_EXECUTION_FAILED"
		outcome = observationNonzeroExit
		if ctx.Err() != nil {
			primaryCode = "M3_DYNAMIC_TIMEOUT"
			outcome = observationTimedOut
		}
	}
	if err := n.stop(writer.id); err != nil {
		if primaryCode != "" {
			return primaryCode, errors.Join(commandErr, err)
		}
		return "M3_DYNAMIC_OBSERVER_FAILED", errors.Join(commandErr, err)
	}
	if primaryCode != "" {
		return primaryCode, commandErr
	}
	if err := n.volume.verifyAttachments(ctx, n.backend.runner, map[string]bool{anchor.id: true}); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_INCOMPLETE", err
	}
	after, err := captureEnvironmentState(ctx, n.backend.runner, anchor.id, limits)
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_INCOMPLETE", err
	}
	if after.files["artifact.tgz"] != original {
		return "M3_DYNAMIC_ENVIRONMENT_INCOMPLETE", errEnvironmentState
	}
	changes, err := compareEnvironmentStates(before, after)
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_INCOMPLETE", err
	}
	if err := n.stop(anchor.id); err != nil {
		return "M3_DYNAMIC_OBSERVER_FAILED", errors.Join(commandErr, err)
	}
	n.ledger.mu.Lock()
	n.ledger.anchorAlive = false
	n.ledger.mu.Unlock()
	events, bytes, err := n.observer.observationUsage(n.traceBudget)
	if err != nil || events < n.events || bytes < n.bytes {
		return "M3_DYNAMIC_OBSERVER_FAILED", err
	}
	_, err = n.watch.finishUnit(ctx, externalUnitEvidence{unitID: string(profile.kind), containerID: writer.id, terminalOutcome: outcome, observerComplete: true, containerGone: true, cgroupDrained: true, closureUnchanged: true, events: events - n.events, bytes: bytes - n.bytes})
	if err != nil {
		return "M3_DYNAMIC_EXECUTION_FAILED", errors.Join(commandErr, err)
	}
	n.events, n.bytes = events, bytes
	fixtureDigest, _ := domain.NewSHA256Digest(profile.identity)
	beforeDigest, _ := domain.NewSHA256Digest(before.digest)
	afterDigest, _ := domain.NewSHA256Digest(after.digest)
	n.states = append(n.states, domain.SandboxEnvironmentState{Profile: profile.kind, FixtureDigest: fixtureDigest, Before: beforeDigest, After: afterDigest, Created: changes.created, Deleted: changes.deleted, Modified: changes.modified})
	for _, change := range []struct {
		subject string
		count   uint64
	}{{"workspace-created", changes.created}, {"workspace-deleted", changes.deleted}, {"workspace-modified", changes.modified}} {
		if change.count == 0 {
			continue
		}
		fact, err := domain.NewCountedSandboxObservation(domain.ObservationFilesystem, change.subject, change.count)
		if err != nil {
			return "M3_DYNAMIC_ENVIRONMENT_INCOMPLETE", err
		}
		n.observed = append(n.observed, fact)
	}
	fact, err := domain.NewSandboxObservation(domain.ObservationProcess, "environment-"+string(profile.kind)+"-completed")
	if err != nil {
		return "M3_DYNAMIC_OBSERVER_FAILED", err
	}
	n.observed = append(n.observed, fact)
	if err := closeWritableOutputVolume(ctx, n.backend.runner, &n.volume, environmentWorkspaceRoot); err != nil {
		return "M3_DYNAMIC_CLEANUP_FAILED", err
	}
	n.volume = closureVolume{}
	return "", nil
}

func (n *environmentInspection) start(ctx context.Context, session domain.SandboxSessionID, profile inspectionEnvironmentProfile, readOnly bool) (runtime *pythonPhaseRuntime, resultErr error) {
	if err := n.watch.beginTransition(ctx); err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, n.watch.endTransition(context.Background())) }()
	args, err := environmentContainerArguments(session, profile, n.volume, readOnly)
	if err != nil {
		return nil, err
	}
	image := slices.Index(args, nodeImageReference)
	if image < 0 {
		return nil, errEnvironmentState
	}
	phase := phaseObservation
	if readOnly {
		phase = phaseAnchor
	}
	extra := []string{"--cpuset-cpus", strconv.Itoa(n.lease.CPU), "--cgroup-parent", n.lease.CgroupParent, "--label", "io.heliopause.observation-phase=" + string(phase)}
	args = append(append(append([]string{}, args[:image]...), extra...), args[image:]...)
	created, err := n.backend.runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		n.uncertain = true
		return nil, errors.Join(errEnvironmentState, err)
	}
	runtime = &pythonPhaseRuntime{id: id, phase: phase}
	n.mu.Lock()
	n.runtimes[id] = runtime
	n.mu.Unlock()
	if err := n.volume.verifyContainerMount(ctx, n.backend.runner, id, readOnly); err != nil {
		return nil, err
	}
	runtime.trace, err = n.observer.StartEnvironmentProfile(ctx, id, readOnly, n.traceBudget)
	if err != nil {
		return nil, err
	}
	if err := discardCommand(ctx, n.backend.runner, "docker", "start", id); err != nil {
		return nil, err
	}
	lease, err := n.backend.resources.Register(ctx, n.owner.String(), id, string(phase))
	if err != nil || lease.CgroupParent != n.lease.CgroupParent || lease.CPU != n.lease.CPU {
		return nil, errors.Join(errEnvironmentState, err)
	}
	runtime.registered = true
	if err := awaitBoundaryHelper(ctx, n.backend.runner, id); err != nil {
		return nil, err
	}
	if err := n.observer.AwaitMountAnchors(ctx, id); err != nil {
		return nil, err
	}
	if err := verifyObservationRuntimeAlive(ctx, n.backend.runner, id); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (n *environmentInspection) stop(id string) (resultErr error) {
	if err := n.watch.beginTermination(context.Background()); err != nil {
		resultErr = err
	} else {
		defer func() { resultErr = errors.Join(resultErr, n.watch.endTermination(context.Background())) }()
	}
	n.mu.Lock()
	runtime := n.runtimes[id]
	n.mu.Unlock()
	if runtime == nil || runtime.collected || runtime.trace == nil || !runtime.registered {
		return errors.Join(resultErr, errEnvironmentState)
	}
	lease, trace, err := runtime.stopAndCollect(n.backend.resources, n.owner.String())
	n.observed = append(n.observed, trace.observations...)
	if err != nil || trace.limitation != "" || !trace.diagnostic.SessionComplete || lease.CgroupParent != n.lease.CgroupParent {
		return errors.Join(resultErr, errEnvironmentState, err, fmt.Errorf("environment trace: %s", trace.diagnostic.String()))
	}
	n.mu.Lock()
	runtime.collected = true
	delete(n.runtimes, id)
	n.mu.Unlock()
	return resultErr
}

func (n *environmentInspection) ids() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	ids := make([]string, 0, len(n.runtimes))
	for id := range n.runtimes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (n *environmentInspection) abort(ctx context.Context) error {
	var failure error
	for _, id := range n.ids() {
		done := make(chan error, 1)
		go func() { _, err := n.backend.resources.Terminate(ctx, n.owner.String(), id); done <- err }()
		removeErr := discardCommand(ctx, n.backend.runner, "docker", "rm", "--force", id)
		remaining, queryErr := n.backend.runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "id="+id, "--format", "{{.ID}}")
		helperErr := <-done
		if helperErr != nil || queryErr != nil || len(remaining) > 1024 || strings.TrimSpace(string(remaining)) != "" {
			failure = errors.Join(failure, errEnvironmentState, helperErr, queryErr, removeErr)
		}
	}
	return failure
}

func (n *environmentInspection) close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var failure error
	if n.watch != nil {
		failure = errors.Join(failure, n.watch.stop())
	}
	for _, id := range n.ids() {
		n.mu.Lock()
		runtime := n.runtimes[id]
		n.mu.Unlock()
		if err := n.stop(id); err != nil {
			failure = errors.Join(failure, err)
			// A failed registration/trace never becomes completion. Independently
			// remove the exact controller-created ID, and preserve unknown outcomes.
			if err := discardCommand(ctx, n.backend.runner, "docker", "rm", "--force", id); err != nil {
				failure = errors.Join(failure, err)
			}
			remaining, err := n.backend.runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "id="+id, "--format", "{{.ID}}")
			if err != nil || len(remaining) > 1024 || strings.TrimSpace(string(remaining)) != "" {
				n.uncertain = true
				failure = errors.Join(failure, errEnvironmentState, err)
			}
			if runtime != nil && runtime.trace != nil {
				trace := runtime.collect()
				if trace.limitation != "" || !trace.diagnostic.SessionComplete {
					failure = errors.Join(failure, errEnvironmentState)
				}
			}
		}
	}
	if n.attempted {
		lease, err := n.backend.resources.Close(ctx, n.owner.String())
		failure = errors.Join(failure, err)
		if err == nil && lease.CgroupParent == n.lease.CgroupParent {
			n.finalCPU = lease.UsageUsec
		} else if err == nil {
			failure = errors.Join(failure, errEnvironmentState)
		}
	}
	if n.volume.name != "" && !n.uncertain {
		failure = errors.Join(failure, closeWritableOutputVolume(ctx, n.backend.runner, &n.volume, environmentWorkspaceRoot))
	}
	if n.uncertain {
		failure = errors.Join(failure, errors.New("uncertain environment ownership retained"))
	}
	return failure
}
