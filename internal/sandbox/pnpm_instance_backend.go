package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMInstanceExecution = errors.New("pnpm instance inspection is unavailable or incomplete")

// ExecutePNPMInstance observes one exact frozen instance in a fresh session.
// The layout is static input binding, not Policy approval or Host authority.
func (b *Backend) ExecutePNPMInstance(ctx context.Context, layout artifactnpm.PNPMInspectionLayout, subject domain.AcquiredArtifact, instance string) (domain.SandboxResult, error) {
	if b == nil || ctx == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || layout.RuntimeIdentity() != PNPMResolverRuntimeIdentity() {
		return domain.SandboxResult{}, errPNPMInstanceExecution
	}
	if err := ctx.Err(); err != nil {
		return domain.SandboxResult{}, errors.Join(errPNPMInstanceExecution, err, context.Cause(ctx))
	}
	_, ok := b.observer.(npmInstanceEnvironmentTraceObserver)
	_, inputOK := b.runner.(inputCommandRunner)
	_, outputOK := b.runner.(streamingDockerOutput)
	if !ok || !inputOK || !outputOK {
		return domain.SandboxResult{}, errPNPMInstanceExecution
	}
	relative, err := layout.Target(subject, instance)
	target := pnpmProjectDirectory + "/" + relative
	if err != nil || !validPNPMTargetLifecyclePath(target) {
		return domain.SandboxResult{}, errors.Join(errPNPMInstanceExecution, err)
	}
	payload, err := preparePNPMInspectionPayload(ctx, layout)
	if err != nil {
		return domain.SandboxResult{}, errors.Join(errPNPMInstanceExecution, err)
	}
	return b.executeNPMInstance(ctx, nativeRuntimePNPM, target, payload)
}

// Both typed callers validate their opaque layout before this resource owner.
func (b *Backend) executeNPMInstance(ctx context.Context, kind nativeRuntimeToolKind, target string, payload pnpmInspectionPayload) (domain.SandboxResult, error) {
	return b.executeNPMInstanceWithBunRuntime(ctx, kind, target, payload, nil)
}

func (b *Backend) executeNPMInstanceWithBunRuntime(ctx context.Context, kind nativeRuntimeToolKind, target string, payload pnpmInspectionPayload, prepare func(context.Context) (nativeProjectTool, error)) (result domain.SandboxResult, resultErr error) {
	defer func() {
		if result.Status() != domain.SandboxCompleted {
			resultErr = errors.Join(errPNPMInstanceExecution, resultErr)
		}
	}()
	_, known := nativeProjectIdentity(kind)
	validTarget := kind == nativeRuntimePNPM && validPNPMTargetLifecyclePath(target) || kind == nativeRuntimeBun && validBunTargetLifecyclePath(target) || kind == nativeRuntimeNPMProject && (validNPMProjectTargetLifecyclePath(target) || target == nodeExecutionProject)
	if !known || !validTarget || payload.kind != kind || payload.layout.String() == "" || prepare != nil && kind != nativeRuntimeBun {
		return domain.SandboxResult{}, errPNPMInstanceExecution
	}
	session, err := b.newSessionID()
	if err != nil || session.String() == "" {
		return domain.SandboxResult{}, errors.Join(errPNPMInstanceExecution, err)
	}
	observer, ok := b.observer.(environmentTraceObserver)
	_, instanceObserver := b.observer.(npmInstanceEnvironmentTraceObserver)
	_, inputOK := b.runner.(inputCommandRunner)
	_, streamOK := b.runner.(streamingDockerOutput)
	_, discardOK := b.runner.(discardCommandRunner)
	if !ok || !instanceObserver || !inputOK || !streamOK || !discardOK || b.resources == nil {
		return incomplete(session, "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE")
	}
	capability, err := b.probe(ctx)
	if err != nil {
		result, failure := incomplete(session, "M3_DYNAMIC_CAPABILITY_ERROR")
		return result, errors.Join(failure, err)
	}
	if !capability.Available {
		return incomplete(session, "M3_DYNAMIC_CAPABILITY_UNAVAILABLE")
	}
	operation, code, err := b.newEnvironmentInspection(session, observer, "npm-lifecycle")
	if err != nil {
		result, resultErr := incomplete(session, code)
		return result, errors.Join(resultErr, err)
	}
	run, cancel := context.WithDeadline(ctx, operation.ledger.policy.wallDeadline)
	defer cancel()
	if err := operation.initialize(run, cancel); err != nil {
		return operation.finish(ctx, run, "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE", err)
	}
	seen := map[string]bool{session.String(): true}
	for _, profileKind := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
		experiment, err := b.newSessionID()
		if err != nil || experiment.String() == "" || seen[experiment.String()] {
			return operation.finish(ctx, run, "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err)
		}
		seen[experiment.String()] = true
		profile, err := newInspectionEnvironmentProfile(profileKind, experiment)
		if err != nil {
			return operation.finish(ctx, run, "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err)
		}
		code, cause := operation.instanceExperiment(run, experiment, profile, kind, target, payload, prepare)
		if code != "" || cause != nil {
			return operation.finish(ctx, run, code, cause)
		}
	}
	completed, err := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
	if err != nil {
		return operation.finish(ctx, run, "M3_DYNAMIC_OBSERVER_FAILED", err)
	}
	operation.observed = append(operation.observed, completed)
	return operation.finish(ctx, run, "", nil)
}

// The original frozen payload, null network, native runtime and exact lifecycle
// argv remain this instance owner's responsibility. Resource termination and
// workspace evidence use the same owner as ordinary npm archive inspection.
func (n *environmentInspection) instanceExperiment(ctx context.Context, session domain.SandboxSessionID, profile inspectionEnvironmentProfile, kind nativeRuntimeToolKind, target string, payload pnpmInspectionPayload, prepare func(context.Context) (nativeProjectTool, error)) (string, error) {
	b := n.backend
	n.projectKind = kind
	var tool nativeProjectTool
	var err error
	if prepare != nil {
		tool, err = prepare(ctx)
		if err != nil || tool == nil || tool.nativeKind() != nativeRuntimeBun {
			return "M3_DYNAMIC_SETUP_FAILED", errors.Join(errBunToolSupply, err)
		}
		n.native, err = createNativeRuntimeVolume(ctx, b.runner, nativeRuntimeBun)
		if err != nil {
			return "M3_DYNAMIC_SETUP_FAILED", err
		}
	}
	n.project, err = createNativeProjectVolume(ctx, b.runner, session.String(), kind)
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	n.volume, err = createEnvironmentWorkspace(ctx, b.runner, n.owner, profile, n.policy.RuntimeTmpfs())
	if err != nil {
		n.uncertain = n.volume.name != ""
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	writer, err := n.start(ctx, session, profile, false)
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	container := writer.id
	if n.native.name != "" {
		if err := n.native.verifyTarget(ctx, b.runner, container); err != nil {
			return "M3_DYNAMIC_SETUP_FAILED", err
		}
		if err := tool.introduce(ctx, b.runner, container); err != nil {
			return "M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED", err
		}
		if err := n.native.verifyTarget(ctx, b.runner, container); err != nil {
			return "M3_DYNAMIC_SETUP_FAILED", err
		}
	}
	if err := n.project.verifyAttachments(ctx, b.runner, map[string]bool{container: false}); err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	if err := copyNPMInspectionPayload(ctx, b.runner, container, payload, kind); err != nil {
		return "M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED", err
	}
	if err := verifyPNPMInstanceNetwork(ctx, b.runner, container); err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	if err := n.project.verifyAttachments(ctx, b.runner, map[string]bool{container: false}); err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	if n.native.name != "" {
		if err := n.native.verifyTarget(ctx, b.runner, container); err != nil {
			return "M3_DYNAMIC_SETUP_FAILED", err
		}
	}
	seed, err := profile.payload()
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := b.runner.(inputCommandRunner).RunInput(ctx, bytes.NewReader(seed), "docker", "cp", "--archive", "-", container+":/tmp"); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	anchor, err := n.start(ctx, session, profile, true)
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	attachments := map[string]bool{container: false, anchor.id: true}
	if err := n.project.verifyAttachments(ctx, b.runner, attachments); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := n.volume.verifyAttachments(ctx, b.runner, attachments); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	// Existing source membership plus the bounded change allowance covers large
	// unchanged project trees without using the summary-count cap as a file cap.
	capacity := n.policy.RuntimeTmpfs() + n.project.capacity
	limits := environmentStateLimits{entries: len(payload.entries) + int(domain.MaximumObservationSummaryCount), bytes: capacity, archiveBytes: 2 * capacity}
	before, err := captureEnvironmentState(ctx, b.runner, anchor.id, limits)
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := profile.verifyPlacement(before); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	n.ledger.mu.Lock()
	n.ledger.preparationOK = true
	n.ledger.anchorAlive = true
	n.ledger.mu.Unlock()
	if err := n.ledger.beginUnit(string(profile.kind), container, time.Now()); err != nil {
		return "M3_DYNAMIC_ACCOUNTING_INCOMPLETE", err
	}
	var command []string
	if kind == nativeRuntimeBun {
		if n.native.name != "" {
			command, err = bunRuntimeTargetLifecycleExecArguments(container, target)
		} else {
			command, err = bunTargetLifecycleExecArguments(container, target)
		}
	} else if kind == nativeRuntimeNPMProject && target == nodeExecutionProject {
		command, err = npmProjectRootLifecycleExecArguments(container)
	} else if kind == nativeRuntimeNPMProject {
		command, err = npmProjectTargetLifecycleExecArguments(container, target)
	} else {
		command, err = pnpmTargetLifecycleExecArguments(container, target)
	}
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	commandErr := discardCommand(ctx, b.runner, "docker", command...)
	primary := ""
	if commandErr != nil || ctx.Err() != nil {
		primary = "M3_DYNAMIC_EXECUTION_FAILED"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			primary = "M3_DYNAMIC_TIMEOUT"
		}
		commandErr = errors.Join(commandErr, ctx.Err(), context.Cause(ctx))
	}
	if err := n.stop(container); err != nil {
		if primary != "" {
			return primary, errors.Join(commandErr, err)
		}
		return "M3_DYNAMIC_OBSERVER_FAILED", err
	}
	if primary != "" {
		return primary, commandErr
	}
	if err := n.project.verifyAttachments(ctx, b.runner, map[string]bool{anchor.id: true}); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_INCOMPLETE", err
	}
	return n.completeEnvironment(ctx, profile, writer, anchor, before, limits)
}

// A successful query must prove the canonical Docker null network and absence
// of addresses. An empty/failed response is not evidence of isolation.
func verifyPNPMInstanceNetwork(ctx context.Context, runner CommandRunner, container string) error {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) {
		return errPNPMInstanceExecution
	}
	mode, err := runner.Output(ctx, "docker", "inspect", "--format", "{{.HostConfig.NetworkMode}}", container)
	if err != nil || len(mode) > 64 || strings.TrimSpace(string(mode)) != "none" {
		return errors.Join(errPNPMInstanceExecution, err)
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", "{{json .NetworkSettings.Networks}}", container)
	var networks map[string]struct{ NetworkID, EndpointID, IPAddress, GlobalIPv6Address string }
	if err != nil || len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &networks) != nil || networks == nil || len(networks) > 1 {
		return errors.Join(errPNPMInstanceExecution, err)
	}
	body, err = runner.Output(ctx, "docker", "network", "inspect", "none", "--format", "{{json .}}")
	var none struct{ Name, Id, Driver, Scope string }
	if err != nil || len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &none) != nil || none.Name != "none" || none.Driver != "null" || none.Scope != "local" || !exactObservationContainerID(none.Id) {
		return errors.Join(errPNPMInstanceExecution, err)
	}
	for name, view := range networks {
		if name != "none" || view.NetworkID != none.Id || view.IPAddress != "" || view.GlobalIPv6Address != "" {
			return errPNPMInstanceExecution
		}
	}
	return nil
}
