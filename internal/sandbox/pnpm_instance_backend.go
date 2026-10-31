package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

var errPNPMInstanceExecution = errors.New("pnpm instance inspection is unavailable or incomplete")

type pnpmTargetTraceObserver interface {
	StartPNPMTargetLifecycleProfile(context.Context, string) (TraceReader, error)
}

// ExecutePNPMInstance observes one exact frozen instance in a fresh session.
// The layout is static input binding, not Policy approval or Host authority.
func (b *Backend) ExecutePNPMInstance(ctx context.Context, layout artifactnpm.PNPMInspectionLayout, subject domain.AcquiredArtifact, instance string) (domain.SandboxResult, error) {
	if b == nil || ctx == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || layout.RuntimeIdentity() != PNPMResolverRuntimeIdentity() {
		return domain.SandboxResult{}, errPNPMInstanceExecution
	}
	if err := ctx.Err(); err != nil {
		return domain.SandboxResult{}, errors.Join(errPNPMInstanceExecution, err, context.Cause(ctx))
	}
	observer, ok := b.observer.(pnpmTargetTraceObserver)
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
	return b.executeNPMInstance(ctx, nativeRuntimePNPM, target, payload, observer.StartPNPMTargetLifecycleProfile)
}

// Both typed callers validate their opaque layout before this resource owner.
func (b *Backend) executeNPMInstance(ctx context.Context, kind nativeRuntimeToolKind, target string, payload pnpmInspectionPayload, startTrace func(context.Context, string) (TraceReader, error)) (domain.SandboxResult, error) {
	return b.executeNPMInstanceWithBunRuntime(ctx, kind, target, payload, startTrace, nil)
}

func (b *Backend) executeNPMInstanceWithBunRuntime(ctx context.Context, kind nativeRuntimeToolKind, target string, payload pnpmInspectionPayload, startTrace func(context.Context, string) (TraceReader, error), prepare func(context.Context) (nativeProjectTool, error)) (domain.SandboxResult, error) {
	spec, known := nativeProjectIdentity(kind)
	validTarget := kind == nativeRuntimePNPM && validPNPMTargetLifecyclePath(target) || kind == nativeRuntimeBun && validBunTargetLifecyclePath(target)
	if !known || !validTarget || payload.kind != kind || payload.layout.String() == "" || startTrace == nil || prepare != nil && kind != nativeRuntimeBun {
		return domain.SandboxResult{}, errPNPMInstanceExecution
	}
	session, err := b.newSessionID()
	if err != nil || session.String() == "" {
		return domain.SandboxResult{}, errors.Join(errPNPMInstanceExecution, err)
	}
	var project closureVolume
	var native nativeRuntimeVolume
	var tool nativeProjectTool
	var container string
	var trace TraceReader
	finish := func(code string, cause error) (domain.SandboxResult, error) {
		if ctx.Err() != nil {
			if code == "" {
				code = "M3_DYNAMIC_EXECUTION_FAILED"
			}
			cause = errors.Join(cause, ctx.Err(), context.Cause(ctx))
		}
		if container != "" {
			if err := b.cleanup(container); err != nil {
				if code == "" {
					code = "M3_DYNAMIC_CLEANUP_FAILED"
				}
				cause = errors.Join(cause, err)
			}
		}
		var observations []domain.SandboxObservation
		if trace != nil {
			collect, done := resolverCleanupContext()
			facts, limitation, diagnostic := collectTraceDiagnostic(collect, trace)
			done()
			observations = facts
			if limitation != "" {
				observations = retainedSuspiciousTraceFacts(diagnostic.KindCounts)
				if code == "" {
					code = limitation
				}
				cause = errors.Join(cause, errors.New(limitation))
			}
		}
		if project.name != "" {
			cleanup, done := resolverCleanupContext()
			err := closeWritableOutputVolume(cleanup, b.runner, &project, spec.root)
			done()
			if err != nil {
				if code == "" {
					code = "M3_DYNAMIC_CLEANUP_FAILED"
				}
				cause = errors.Join(cause, err)
			}
		}
		if native.name != "" {
			cleanup, done := resolverCleanupContext()
			err := native.close(cleanup, b.runner)
			done()
			if err != nil {
				if code == "" {
					code = "M3_DYNAMIC_CLEANUP_FAILED"
				}
				cause = errors.Join(cause, err)
			}
		}
		if code != "" {
			result, err := domain.NewSandboxResult(session, domain.SandboxIncomplete, code, observations)
			return result, errors.Join(errPNPMInstanceExecution, cause, err)
		}
		completed, err := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
		if err != nil {
			return domain.SandboxResult{}, err
		}
		return domain.NewSandboxResult(session, domain.SandboxCompleted, "", append(observations, completed))
	}
	capability, err := b.probe(ctx)
	if err != nil {
		return finish("M3_DYNAMIC_CAPABILITY_ERROR", err)
	}
	if !capability.Available {
		return finish("M3_DYNAMIC_CAPABILITY_UNAVAILABLE", nil)
	}
	if prepare != nil {
		tool, err = prepare(ctx)
		if err != nil || tool == nil || tool.nativeKind() != nativeRuntimeBun {
			return finish("M3_DYNAMIC_SETUP_FAILED", errors.Join(errBunToolSupply, err))
		}
		native, err = createNativeRuntimeVolume(ctx, b.runner, nativeRuntimeBun)
		if err != nil {
			return finish("M3_DYNAMIC_SETUP_FAILED", err)
		}
	}
	project, err = createNativeProjectVolume(ctx, b.runner, session.String(), kind)
	if err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", err)
	}
	args, err := npmInstanceCreateArguments(session, project, kind)
	if err == nil && native.name != "" {
		args, err = bunArtifactRuntimeCreateArguments(args, native)
	}
	if err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", err)
	}
	created, err := b.runner.Output(ctx, "docker", args...)
	if err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", err)
	}
	container = strings.TrimSpace(string(created))
	if !exactObservationContainerID(container) {
		container = ""
		return finish("M3_DYNAMIC_SETUP_FAILED", errPNPMInstanceExecution)
	}
	trace, err = startTrace(ctx, container)
	if err != nil || trace == nil {
		return finish("M3_DYNAMIC_OBSERVER_FAILED", err)
	}
	run, cancel := context.WithTimeout(ctx, b.wallTimeout)
	defer cancel()
	if _, err := b.runner.Output(run, "docker", "start", container); err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", errors.Join(err, run.Err(), context.Cause(run)))
	}
	if err := awaitBoundaryHelper(run, b.runner, container); err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", errors.Join(err, run.Err(), context.Cause(run)))
	}
	if err := awaitMountAnchors(run, b.observer, container); err != nil {
		return finish("M3_DYNAMIC_OBSERVER_FAILED", err)
	}
	if native.name != "" {
		if err := native.verifyTarget(run, b.runner, container); err != nil {
			return finish("M3_DYNAMIC_SETUP_FAILED", err)
		}
		if err := tool.introduce(run, b.runner, container); err != nil {
			return finish("M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED", err)
		}
		if err := native.verifyTarget(run, b.runner, container); err != nil {
			return finish("M3_DYNAMIC_SETUP_FAILED", err)
		}
	}
	if err := project.verifyAttachments(run, b.runner, map[string]bool{container: false}); err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", err)
	}
	if err := copyNPMInspectionPayload(run, b.runner, container, payload, kind); err != nil {
		return finish("M3_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED", err)
	}
	if err := verifyPNPMInstanceNetwork(run, b.runner, container); err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", err)
	}
	if err := project.verifyAttachments(run, b.runner, map[string]bool{container: false}); err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", err)
	}
	if native.name != "" {
		if err := native.verifyTarget(run, b.runner, container); err != nil {
			return finish("M3_DYNAMIC_SETUP_FAILED", err)
		}
	}
	var command []string
	if kind == nativeRuntimeBun {
		if native.name != "" {
			command, err = bunRuntimeTargetLifecycleExecArguments(container, target)
		} else {
			command, err = bunTargetLifecycleExecArguments(container, target)
		}
	} else {
		command, err = pnpmTargetLifecycleExecArguments(container, target)
	}
	if err != nil {
		return finish("M3_DYNAMIC_SETUP_FAILED", err)
	}
	_, err = b.runner.Output(run, "docker", command...)
	if err != nil || run.Err() != nil {
		code := "M3_DYNAMIC_EXECUTION_FAILED"
		if errors.Is(run.Err(), context.DeadlineExceeded) {
			code = "M3_DYNAMIC_TIMEOUT"
		}
		return finish(code, errors.Join(err, run.Err(), context.Cause(run)))
	}
	return finish("", nil)
}

func npmInstanceCreateArguments(session domain.SandboxSessionID, project closureVolume, kind nativeRuntimeToolKind) ([]string, error) {
	spec, known := nativeProjectIdentity(kind)
	if !known || project.destination() != spec.root || project.projectBuildExecutable {
		return nil, errPNPMInstanceExecution
	}
	mount, err := project.mountArgument(false)
	if err != nil {
		return nil, err
	}
	args := createArguments(session)
	image := slices.Index(args, runtimeidentity.NodeImageReference)
	if image < 0 {
		return nil, errPNPMInstanceExecution
	}
	return append(append(append([]string{}, args[:image]...), "--mount", mount), args[image:]...), nil
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
