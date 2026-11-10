package sandbox

import (
	"context"
	"errors"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// ValidatePythonExecutionInputs validates a current approved closure in a sole
// read-only input view and disposes it. This DATA check invokes no selected
// console/module/callable and creates no inspection or user-run completion.
func (b *PythonDynamicBackend) ValidatePythonExecutionInputs(ctx context.Context, execution domain.ExecutionContext, entry artifactpypi.ConsoleExecutionEntry) error {
	return b.withPythonExecutionInputs(ctx, execution, entry, func(t *pythonTransactionExecutor) error {
		member := entry.Metadata()
		installed, ok := t.installed.files[member.Path]
		if !ok || installed.sha256 != member.SHA256 || installed.size != member.Size {
			return errors.New("python console declaration differs from the exact readonly input")
		}
		return nil
	})
}

func (b *PythonDynamicBackend) withPythonExecutionInputs(ctx context.Context, execution domain.ExecutionContext, entry artifactpypi.ConsoleExecutionEntry, consume func(*pythonTransactionExecutor) error) error {
	return b.withPythonExecutionSource(ctx, execution, execution.Set().Inspected().Inspections(), entry, consume)
}

func (b *PythonDynamicBackend) withPythonExecutionSelectionInputs(ctx context.Context, execution domain.ExecutionSelectionContext, entry artifactpypi.ConsoleExecutionEntry, consume func(*pythonTransactionExecutor) error) error {
	return b.withPythonExecutionSource(ctx, execution, execution.Set().Inspected().Inspections(), entry, consume)
}

// Only the two immutable current execution contexts implement this private
// boundary. Complete inspected entries come from their own typed approved set.
type pythonExecutionSource interface {
	Valid() bool
	RuntimeIdentity() string
	EntryArtifact() domain.AcquiredArtifact
	Command() string
	EntryPath() string
	EntryDigest() domain.ContentDigest
	EntrySize() uint64
}

func (b *PythonDynamicBackend) withPythonExecutionSource(ctx context.Context, execution pythonExecutionSource, inspections []domain.DependencyInspection, entry artifactpypi.ConsoleExecutionEntry, consume func(*pythonTransactionExecutor) error) (resultErr error) {
	if ctx == nil {
		return errors.New("python execution inputs require a context")
	}
	defer func() { resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx)) }()
	if b == nil || b.runner == nil || b.introducer == nil || b.probe == nil || b.newSessionID == nil || b.resources == nil || consume == nil || ctx.Err() != nil || !execution.Valid() || !entry.Valid() {
		return errors.New("python execution inputs require configured lifecycle and current approved source")
	}
	member := entry.Metadata()
	if execution.RuntimeIdentity() != runtimeidentity.PythonImageReference || execution.EntryArtifact() != entry.Artifact() || execution.Command() != entry.Command() || execution.EntryPath() != member.Path || execution.EntryDigest().String() != member.SHA256 || execution.EntrySize() != uint64(member.Size) {
		return errors.New("python execution input declaration or runtime differs from its request")
	}
	var closure []domain.AcquiredArtifact
	for _, inspected := range inspections {
		artifact := inspected.Artifact()
		declared, ok := artifact.DeclaredIntegrity()
		if artifact.Identity().Source() != artifactpypi.PublicPyPIProfile().Source() || artifact.Identity().Variant() != "wheel" || !ok || declared != "sha256:"+artifact.Digest().String() {
			return errors.New("python execution inputs require the complete original public wheel closure")
		}
		closure = append(closure, artifact)
	}
	observer, ok := b.observer.(observationPhaseObserver)
	if !ok {
		return errors.New("python execution input observer is unavailable")
	}
	if _, ok := b.observer.(pythonClosureObserver); !ok {
		return errors.New("python execution input closure observer is unavailable")
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return errors.Join(errors.New("python execution input pinned runtime is unavailable"), err)
	}
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	manifest, err := b.introducer.buildClosureManifest(ctx, closure, policy)
	if err != nil {
		return err
	}
	if file, ok := manifest.files[member.Path]; !ok || file.sha256 != member.SHA256 || file.size != member.Size || file.scheme != artifactpypi.SchemeDistInfo {
		return errors.New("python execution input manifest differs from the selected declaration")
	}
	now := time.Now()
	accounting, err := observationPolicyForPython(policy, "pypi-wheel", 0, now)
	if err != nil {
		return err
	}
	// There are genuinely zero observation units here. This ledger accounts
	// preparation CPU/teardown only and never becomes a SandboxResult.
	ledger, err := newObservationTransaction(accounting, nil, now)
	if err != nil {
		return err
	}
	traceLedger, err := newObservationTraceLedger("pypi-wheel")
	if err != nil {
		return err
	}
	session, err := b.newSessionID()
	if err != nil {
		return err
	}
	run, cancel := context.WithDeadline(ctx, accounting.wallDeadline)
	defer cancel()
	t := &pythonTransactionExecutor{backend: b, observer: observer, resources: b.resources, manifest: manifest, transaction: session.String(), profile: "pypi-wheel", policy: policy, ledger: ledger, traceLedger: traceLedger, runtimes: make(map[string]*pythonPhaseRuntime), artifact: entry.Artifact(), started: now, requestCtx: run}
	defer func() {
		// Ordinary watch.stop cancels its private run context. Preserve any
		// prior run failure before cleanup; cleanup retains watch faults and
		// finalize still checks the complete wall/CPU/teardown evidence.
		runErr, runCause := run.Err(), context.Cause(run)
		resultErr = errors.Join(resultErr, runErr, runCause, t.cleanup())
		if resultErr == nil {
			resultErr = ledger.finalize(t.finalCPU, time.Now())
		}
	}()
	// Close is still attempted after an ambiguous Create failure for this
	// independently generated, controller-owned transaction identity.
	t.created = true
	lease, err := b.resources.Create(run, t.transaction, "pypi")
	if err != nil || lease.CgroupParent == "" {
		return errors.Join(errors.New("python execution input resource lease is unavailable"), err)
	}
	t.lease = lease
	volume, err := createClosureVolume(run, b.runner, t.transaction, manifest.identity, policy.RuntimeTmpfs())
	if volume.name != "" {
		t.volume, t.volumeMade = volume, true
	}
	if err != nil {
		return err
	}
	preparation, err := t.startPhase(run, phasePreparation, 0)
	if err != nil {
		return err
	}
	watch, err := startObservationCPUWatch(run, b.resources, t.transaction, ledger, cancel, func(stop context.Context) error { return t.abortAll(stop) })
	if err != nil {
		return err
	}
	t.watch = watch
	if failure := t.prepareWheelClosure(run, preparation.id, entry.Artifact(), closure); failure != nil {
		return failure
	}
	if failure := t.sealPreparedClosure(run, preparation.id); failure != nil {
		return failure
	}
	if err := t.verifyAnchorAndVolume(run, nil); err != nil {
		return err
	}
	if err := consume(t); err != nil {
		return err
	}
	after, err := verifyClosureInstallation(run, b.runner, t.anchor, manifest, policy.RuntimeTmpfs())
	if err != nil || after.identity != t.installed.identity {
		return errors.Join(errors.New("python execution readonly input changed"), err)
	}
	return t.verifyAnchorAndVolume(run, nil)
}
