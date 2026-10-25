package sandbox

import (
	"context"
	"errors"
	"slices"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type bunRuntimeTargetTraceObserver interface {
	StartBunRuntimeTargetLifecycleProfile(context.Context, string) (TraceReader, error)
}

// ExecuteBunRuntimeInstance supplies the pinned Bun image in a separately owned
// lifecycle session. The current command still observes npm/Node lifecycle;
// native Artifact execution and whole required coverage are separate consumers.
func (b *Backend) ExecuteBunRuntimeInstance(ctx context.Context, layout artifactnpm.BunInspectionLayout, subject domain.AcquiredArtifact, instance string) (domain.SandboxResult, error) {
	return b.executeBunRuntimeInstance(ctx, layout, subject, instance, func(ctx context.Context) (nativeProjectTool, error) {
		return prepareBunTool(ctx)
	})
}

func (b *Backend) executeBunRuntimeInstance(ctx context.Context, layout artifactnpm.BunInspectionLayout, subject domain.AcquiredArtifact, instance string, prepare func(context.Context) (nativeProjectTool, error)) (domain.SandboxResult, error) {
	if b == nil || ctx == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || prepare == nil || layout.RuntimeIdentity() != BunResolverRuntimeIdentity() {
		return domain.SandboxResult{}, errBunInstanceExecution
	}
	if err := ctx.Err(); err != nil {
		return domain.SandboxResult{}, errors.Join(errBunInstanceExecution, err, context.Cause(ctx))
	}
	observer, ok := b.observer.(bunRuntimeTargetTraceObserver)
	_, inputOK := b.runner.(inputCommandRunner)
	_, outputOK := b.runner.(streamingDockerOutput)
	if !ok || !inputOK || !outputOK {
		return domain.SandboxResult{}, errBunInstanceExecution
	}
	relative, err := layout.Target(subject, instance)
	target := bunProjectDirectory + "/" + relative
	if err != nil || !validBunTargetLifecyclePath(target) {
		return domain.SandboxResult{}, errors.Join(errBunInstanceExecution, err)
	}
	payload, err := prepareBunInspectionPayload(ctx, layout)
	if err != nil {
		return domain.SandboxResult{}, errors.Join(errBunInstanceExecution, err)
	}
	result, err := b.executeNPMInstanceWithBunRuntime(ctx, nativeRuntimeBun, target, payload, observer.StartBunRuntimeTargetLifecycleProfile, prepare)
	if err != nil {
		return result, errors.Join(errBunInstanceExecution, err)
	}
	return result, nil
}

func bunArtifactRuntimeCreateArguments(arguments []string, native nativeRuntimeVolume) ([]string, error) {
	if native.kind != nativeRuntimeBun {
		return nil, errBunToolSupply
	}
	mount, err := native.mountArgument()
	if err != nil {
		return nil, err
	}
	args := slices.Clone(arguments)
	matched := 0
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--tmpfs" && args[i+1] == boundaryHelperMount {
			args[i], args[i+1] = "--mount", mount
			matched++
		}
	}
	if matched != 1 {
		return nil, errBunToolSupply
	}
	return args, nil
}

func bunRuntimeTargetLifecycleExpectedTopology() ([]observerMountExpectation, bool) {
	topology, ok := bunTargetLifecycleExpectedTopology()
	if !ok {
		return nil, false
	}
	matched := 0
	for i := range topology {
		if topology[i].Mountpoint == "/haa-runtime" {
			topology[i].FSType, topology[i].NoSUID = "9p", false
			matched++
		}
	}
	return topology, matched == 1
}
