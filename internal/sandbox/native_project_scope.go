package sandbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// nativeProjectScope is shared only by the two compiled npm native clients.
// It owns resources and raw observation, never graph approval or Policy.
type nativeProjectTool interface {
	introduce(context.Context, CommandRunner, string) error
	nativeKind() nativeRuntimeToolKind
}

type nativeProjectScope struct {
	kind      nativeRuntimeToolKind
	runner    CommandRunner
	endpoints EndpointResolver
	observer  TraceObserver
	probe     CapabilityProbe
	policy    ResolverPolicyService
	prepare   func(context.Context) (nativeProjectTool, error)
}

// withNativeProject reuses the existing selection infrastructure ownership.
// Only fixed internal selection/preparation actions enter this scope. A caller
// must discard its captured output whenever the scope or deferred cleanup fails.
func (r *nativeProjectScope) run(ctx context.Context, earlyStore bool, action func(context.Context, string, string, *closureVolume) error) (resultErr error) {
	if r == nil || r.runner == nil || r.endpoints == nil || r.observer == nil || r.probe == nil || r.policy == nil || r.prepare == nil || ctx == nil || action == nil {
		return errors.New("native project infrastructure unavailable")
	}
	spec, known := nativeProjectIdentity(r.kind)
	ownership, ownsKind := nativeResolverOwnership(r.kind)
	if !known || !ownsKind {
		return errors.New("native project kind unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	defer func() {
		if err := ctx.Err(); err != nil {
			resultErr = errors.Join(resultErr, err, context.Cause(ctx))
		}
	}()
	if err := ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	capability, err := r.probe(ctx)
	if err != nil || !capability.Available {
		return errors.Join(fmt.Errorf("%s fixed runtime unavailable", spec.name), err)
	}
	tool, err := r.prepare(ctx)
	if err != nil {
		return err
	}
	if tool == nil || tool.nativeKind() != r.kind {
		return errors.New("native project tool identity differs")
	}

	addresses, err := r.endpoints.Resolve(ctx, []string{"registry.npmjs.org"})
	if err != nil {
		return errors.Join(fmt.Errorf("%s endpoint preflight failed", spec.name), ctx.Err())
	}
	hosts, err := npmNetworkArguments(addresses)
	if err != nil {
		return err
	}
	policy, err := NewResolverNetworkPolicy(r.runner, r.policy)
	if err != nil {
		return err
	}
	network, err := policy.Prepare(ctx, addresses)
	if err != nil {
		return err
	}
	var native nativeRuntimeVolume
	var project closureVolume
	var trace TraceReader
	name, owner, container := "", "", ""
	createAttempted := false
	defer func() {
		var cleanupErr error
		if createAttempted {
			cleanup, done := resolverCleanupContext()
			removeErr := removeOwnedNativeResolverContainer(cleanup, r.runner, name, owner, container, r.kind)
			done()
			if removeErr != nil {
				cleanupErr = fmt.Errorf("%s resolver container cleanup unproven name=%s: %w", spec.name, name, removeErr)
			}
		}
		if trace != nil {
			cleanup, done := resolverCleanupContext()
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			done()
			if limitation != "" {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("%s observation incomplete: %s", spec.name, diagnostic))
			}
			for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if diagnostic.KindCounts[kind] != 0 {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("%s selection contains forbidden observation %s", spec.name, kind))
				}
			}
		}
		if project.name != "" {
			cleanup, done := resolverCleanupContext()
			cleanupErr = errors.Join(cleanupErr, closeWritableOutputVolume(cleanup, r.runner, &project, spec.root))
			done()
		}
		if native.name != "" {
			cleanup, done := resolverCleanupContext()
			cleanupErr = errors.Join(cleanupErr, native.close(cleanup, r.runner))
			done()
		}
		cleanup, done := resolverCleanupContext()
		cleanupErr = errors.Join(cleanupErr, policy.Close(cleanup))
		done()
		if cleanupErr != nil {
			resultErr = errors.Join(resultErr, cleanupErr)
		}
	}()
	native, err = createNativeRuntimeVolume(ctx, r.runner, r.kind)
	if err != nil {
		return err
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return err
	}
	owner, name = session.String(), ownership.prefix+session.String()
	project, err = createNativeProjectVolume(ctx, r.runner, owner, r.kind)
	if err != nil {
		return err
	}
	arguments, err := nativeProjectCreateArguments(network, hosts, native, project, earlyStore)
	if err != nil {
		return err
	}
	listed, err := r.runner.Output(ctx, "docker", "ps", "--all", "--quiet", "--no-trunc", "--filter", "name=^/"+name+"$")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return fmt.Errorf("%s resolver name preflight unavailable or nonempty", spec.name)
	}
	arguments = append([]string{"create", "--name", name, "--label", ownership.label + "=" + owner}, arguments[1:]...)
	createAttempted = true
	created, err := r.runner.Output(ctx, "docker", arguments...)
	if err != nil || !exactObservationContainerID(strings.TrimSpace(string(created))) {
		return errors.Join(fmt.Errorf("%s create outcome unavailable", spec.name), err, ctx.Err())
	}
	container = strings.TrimSpace(string(created))
	if err := verifyOwnedNativeResolverContainer(ctx, r.runner, name, owner, container, r.kind); err != nil {
		return err
	}
	trace, err = startTrace(ctx, r.observer, container, spec.profile)
	if err != nil {
		return errors.Join(fmt.Errorf("%s observer start failed", spec.name), err)
	}
	if _, err := r.runner.Output(ctx, "docker", "start", container); err != nil {
		return errors.Join(fmt.Errorf("%s container start failed", spec.name), err)
	}
	if err := awaitMountAnchors(ctx, r.observer, container); err != nil {
		return err
	}
	if err := awaitBoundaryHelper(ctx, r.runner, container); err != nil {
		return err
	}
	if err := native.verifyTarget(ctx, r.runner, container); err != nil {
		return err
	}
	if err := project.verifyContainerMount(ctx, r.runner, container, false); err != nil {
		return err
	}
	if err := project.verifyAttachments(ctx, r.runner, map[string]bool{container: false}); err != nil {
		return err
	}
	if err := tool.introduce(ctx, r.runner, container); err != nil {
		return err
	}
	if err := native.verifyTarget(ctx, r.runner, container); err != nil {
		return err
	}

	if err := action(ctx, container, network, &project); err != nil {
		return err
	}
	return project.verifyAttachments(ctx, r.runner, map[string]bool{container: false})
}
