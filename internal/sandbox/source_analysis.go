package sandbox

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// SourceAnalyzer runs the fixed parser over readonly DATA. It owns runtime
// state only; it interprets no rules and makes no artifact safety decision.
type SourceAnalyzer struct {
	runner    CommandRunner
	capture   programCapturedRunner
	observer  TraceObserver
	resources ObservationResourceClient
	probe     CapabilityProbe
}

func NewSourceAnalyzer(executor TrustedExecutor, observer TraceObserver, resources ObservationResourceClient) (*SourceAnalyzer, error) {
	capture, captures := executor.(programCapturedRunner)
	if executor == nil || !captures || observer == nil || resources == nil {
		return nil, errors.New("source analysis infrastructure is unavailable")
	}
	if _, ok := observer.(profiledTraceObserver); !ok {
		return nil, errors.New("source analysis requires a profiled observer")
	}
	if _, ok := observer.(mountAnchorReadyObserver); !ok {
		return nil, errors.New("source analysis requires mount attestation")
	}
	return &SourceAnalyzer{runner: admissionAwareRunner(executor), capture: capture, observer: observer, resources: resources, probe: func(ctx context.Context) (Capability, error) {
		available, err := probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
		return Capability{Available: available.Available, LimitationCode: available.LimitationCode}, err
	}}, nil
}

func (b *SourceAnalyzer) Analyze(ctx context.Context, input ports.SourceAnalysisInput) (result ports.SourceAnalysisOutput, resultErr error) {
	if b == nil || ctx == nil || b.runner == nil || b.capture == nil || b.observer == nil || b.resources == nil || b.probe == nil {
		return result, errors.New("source analyzer is unavailable")
	}
	identity, err := domain.NewSandboxSessionID()
	if err != nil {
		return result, err
	}
	fail := func(code string, cause error) (ports.SourceAnalysisOutput, error) {
		outcome, err := domain.NewSandboxResult(identity, domain.SandboxIncomplete, code, nil)
		return ports.SourceAnalysisOutput{Runtime: outcome}, errors.Join(cause, err)
	}
	if err := ctx.Err(); err != nil {
		return fail("S04_SOURCE_ANALYSIS_CANCELLED", err)
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available {
		return fail("S04_SOURCE_ANALYSIS_UNAVAILABLE", err)
	}
	prepared, err := prepareSourceAnalysisInput(ctx, input)
	if err != nil {
		return fail("S04_SOURCE_INPUT_INVALID", err)
	}
	tool, err := readInstalledSemgrepTool(ctx, runtimeidentity.NativeInstallRoot)
	if err != nil {
		return fail("S04_SOURCE_TOOL_UNAVAILABLE", err)
	}
	now := time.Now()
	policy, err := observationPolicyForPython(artifactpypi.PublicPyPIProfile().ResourcePolicy(), sourceAnalysisProfile, 0, now)
	if err != nil {
		return fail("S04_SOURCE_RESOURCE_UNAVAILABLE", err)
	}
	ledger, err := newObservationTransaction(policy, nil, now)
	if err != nil {
		return fail("S04_SOURCE_RESOURCE_UNAVAILABLE", err)
	}
	run, cancel := context.WithDeadline(ctx, policy.wallDeadline)
	defer cancel()
	session := &observedNativeSession{runner: b.runner, observer: b.observer, resources: b.resources, transaction: identity.String(), ledger: ledger}
	defer func() {
		cleanupErr := errors.Join(session.cleanup(), ctx.Err())
		if cleanupErr != nil {
			code := "S04_SOURCE_LIFECYCLE_INCOMPLETE"
			if previous, ok := result.Runtime.LimitationCode(); ok {
				code = previous
			}
			outcome, err := domain.NewSandboxResult(identity, domain.SandboxIncomplete, code, nil)
			result = ports.SourceAnalysisOutput{Runtime: outcome}
			resultErr = errors.Join(resultErr, cleanupErr, err)
		}
	}()
	// Retain the generated owner even when CREATE's response is lost. Cleanup
	// remains mandatory and cannot turn an uncertain creation into success.
	session.resourceCreated = true
	session.lease, err = b.resources.Create(run, session.transaction, "pypi")
	if err != nil || session.lease.CgroupParent == "" || session.lease.CPU < 0 {
		return fail("S04_SOURCE_RESOURCE_UNAVAILABLE", err)
	}
	session.data, err = createSemgrepSupplyVolume(run, b.runner, session.transaction)
	if err != nil {
		return fail("S04_SOURCE_TOOL_UNAVAILABLE", err)
	}
	session.analysisInput, err = createSourceAnalysisVolume(run, b.runner, session.transaction, prepared)
	if err != nil {
		return fail("S04_SOURCE_INPUT_UNAVAILABLE", err)
	}
	toolRW, err := session.data.preparationMount()
	if err != nil {
		return fail("S04_SOURCE_TOOL_UNAVAILABLE", err)
	}
	sourceRW, err := session.analysisInput.preparationMount()
	if err != nil {
		return fail("S04_SOURCE_INPUT_UNAVAILABLE", err)
	}
	writer, err := session.dataKeeper(run, "0:0", "--mount", toolRW, "--mount", sourceRW)
	if err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	if err := session.data.seed(run, b.runner, tool, writer.id); err != nil {
		return fail("S04_SOURCE_TOOL_CHANGED", err)
	}
	if err := session.analysisInput.seed(run, b.runner, prepared, writer.id); err != nil {
		return fail("S04_SOURCE_INPUT_CHANGED", err)
	}
	toolRO, err := session.data.readerMount()
	if err != nil {
		return fail("S04_SOURCE_TOOL_UNAVAILABLE", err)
	}
	sourceRO, err := session.analysisInput.readerMount()
	if err != nil {
		return fail("S04_SOURCE_INPUT_UNAVAILABLE", err)
	}
	anchor, err := session.dataKeeper(run, "1000:1000", "--mount", toolRO, "--mount", sourceRO)
	if err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	if err := session.data.bindAnchor(run, b.runner, anchor.id); err != nil {
		return fail("S04_SOURCE_TOOL_CHANGED", err)
	}
	if err := session.analysisInput.bindAnchor(run, b.runner, anchor.id); err != nil {
		return fail("S04_SOURCE_INPUT_CHANGED", err)
	}
	if err := session.removeKeeper(run, writer); err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	if err := session.data.seal(run, b.runner); err != nil {
		return fail("S04_SOURCE_TOOL_CHANGED", err)
	}
	if err := session.analysisInput.seal(run, b.runner); err != nil {
		return fail("S04_SOURCE_INPUT_CHANGED", err)
	}
	arguments, err := sourceAnalysisCreateArguments(session.data, session.analysisInput, session.transaction, session.lease)
	if err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	session.reader, err = session.newKeeper()
	if err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	if err := session.createKeeper(run, session.reader, arguments); err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	session.trace, err = b.observer.(profiledTraceObserver).StartProfile(run, session.reader.id, sourceAnalysisProfile)
	if err != nil {
		return fail("S04_SOURCE_OBSERVER_UNAVAILABLE", err)
	}
	if _, err := b.runner.Output(run, "docker", "start", session.reader.id); err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	session.registrationAttempted = true
	registered, err := b.resources.Register(run, session.transaction, session.reader.id, string(phasePreparation))
	if err != nil || registered.CgroupParent != session.lease.CgroupParent || registered.CPU != session.lease.CPU {
		return fail("S04_SOURCE_RESOURCE_UNAVAILABLE", err)
	}
	session.registered = true
	id := session.reader.id
	session.watch, err = startObservationCPUWatch(run, b.resources, session.transaction, ledger, cancel, func(stop context.Context) error { return session.abortRuntime(stop, id) })
	if err != nil {
		return fail("S04_SOURCE_RESOURCE_UNAVAILABLE", err)
	}
	if err := awaitMountAnchors(run, b.observer, id); err != nil {
		return fail("S04_SOURCE_OBSERVER_UNAVAILABLE", err)
	}
	if err := awaitBoundaryHelper(run, b.runner, id); err != nil {
		return fail("S04_SOURCE_PREPARATION_FAILED", err)
	}
	verify := func() error {
		if err := session.data.verifyReader(run, b.runner, id); err != nil {
			return err
		}
		return session.analysisInput.verifyReader(run, b.runner, id)
	}
	if err := verify(); err != nil {
		return fail("S04_SOURCE_INPUT_CHANGED", err)
	}
	failed := errors.New("source analyzer execution was not qualified")
	captured, err := captureAdmittedProgram(run, b.capture, id, boundaryELFHandoffMode, sourceAnalysisCommand(), failed, func(cause error) error { return cause }, verify)
	if err != nil || !captured.complete || captured.output == nil || captured.output.StdoutOverflow() || captured.output.StderrOverflow() {
		return fail("S04_SOURCE_EXECUTION_INCOMPLETE", err)
	}
	// A successful capture is still private DATA until the deferred trace,
	// resource and input disposal checks have all succeeded.
	result.Runtime, err = domain.NewSandboxResult(identity, domain.SandboxCompleted, "", nil)
	if err != nil {
		return fail("S04_SOURCE_EXECUTION_INCOMPLETE", err)
	}
	result.Output = captured.output
	result.Rules = bytes.Clone(tool.rules)
	return result, nil
}

var _ ports.SourceAnalysis = (*SourceAnalyzer)(nil)
