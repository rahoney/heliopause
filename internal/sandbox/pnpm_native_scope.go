package sandbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// withNativeProject reuses the existing selection infrastructure ownership.
// Only fixed internal selection/preparation actions enter this scope. A caller
// must discard its captured output whenever the scope or deferred cleanup fails.
func (r *PNPMResolver) withNativeProject(ctx context.Context, earlyStore bool, action func(context.Context, string, string, *closureVolume) error) (resultErr error) {
	if r == nil || r.runner == nil || r.endpoints == nil || r.observer == nil || r.probe == nil || r.policy == nil || r.prepare == nil || ctx == nil || action == nil {
		return errors.New("pnpm native infrastructure unavailable")
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
		return errors.Join(errors.New("pnpm fixed runtime unavailable"), err)
	}
	tool, err := r.prepare(ctx)
	if err != nil {
		return err
	}
	addresses, err := r.endpoints.Resolve(ctx, []string{"registry.npmjs.org"})
	if err != nil {
		return errors.Join(errors.New("pnpm endpoint preflight failed"), ctx.Err())
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
			removeErr := removeOwnedNativeResolverContainer(cleanup, r.runner, name, owner, container, nativeRuntimePNPM)
			done()
			if removeErr != nil {
				cleanupErr = fmt.Errorf("pnpm resolver container cleanup unproven name=%s: %w", name, removeErr)
			}
		}
		if trace != nil {
			cleanup, done := resolverCleanupContext()
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			done()
			if limitation != "" {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("pnpm observation incomplete: %s", diagnostic))
			}
			for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if diagnostic.KindCounts[kind] != 0 {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("pnpm selection contains forbidden observation %s", kind))
				}
			}
		}
		if project.name != "" {
			cleanup, done := resolverCleanupContext()
			cleanupErr = errors.Join(cleanupErr, closePNPMProjectVolume(cleanup, r.runner, &project))
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
	native, err = createPNPMRuntimeVolume(ctx, r.runner)
	if err != nil {
		return err
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return err
	}
	owner, name = session.String(), "haa-pnpm-resolver-"+session.String()
	project, err = createPNPMProjectVolume(ctx, r.runner, owner)
	if err != nil {
		return err
	}
	arguments, err := pnpmCreateArgumentsWithStore(network, hosts, native, project, earlyStore)
	if err != nil {
		return err
	}
	listed, err := r.runner.Output(ctx, "docker", "ps", "--all", "--quiet", "--no-trunc", "--filter", "name=^/"+name+"$")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return errors.New("pnpm resolver name preflight unavailable or nonempty")
	}
	arguments = append([]string{"create", "--name", name, "--label", pnpmResolverOwnerLabel + "=" + owner}, arguments[1:]...)
	createAttempted = true
	created, err := r.runner.Output(ctx, "docker", arguments...)
	if err != nil || !exactObservationContainerID(strings.TrimSpace(string(created))) {
		return errors.Join(errors.New("pnpm create outcome unavailable"), err, ctx.Err())
	}
	container = strings.TrimSpace(string(created))
	if err := verifyOwnedNativeResolverContainer(ctx, r.runner, name, owner, container, nativeRuntimePNPM); err != nil {
		return err
	}
	trace, err = startTrace(ctx, r.observer, container, pnpmResolverProfile)
	if err != nil {
		return errors.Join(errors.New("pnpm observer start failed"), err)
	}
	if _, err := r.runner.Output(ctx, "docker", "start", container); err != nil {
		return errors.Join(errors.New("pnpm container start failed"), err)
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
