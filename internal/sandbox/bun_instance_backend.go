package sandbox

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunInstanceExecution = errors.New("bun instance inspection is unavailable or incomplete")

type bunTargetTraceObserver interface {
	StartBunTargetLifecycleProfile(context.Context, string) (TraceReader, error)
}

// ExecuteBunInstance consumes one exact bound hoisted instance in a fresh session.
// This current lane executes npm/Node lifecycle and grants no Policy authority.
func (b *Backend) ExecuteBunInstance(ctx context.Context, layout artifactnpm.BunInspectionLayout, subject domain.AcquiredArtifact, instance string) (domain.SandboxResult, error) {
	if b == nil || ctx == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || layout.RuntimeIdentity() != BunResolverRuntimeIdentity() {
		return domain.SandboxResult{}, errBunInstanceExecution
	}
	if err := ctx.Err(); err != nil {
		return domain.SandboxResult{}, errors.Join(errBunInstanceExecution, err, context.Cause(ctx))
	}
	observer, ok := b.observer.(bunTargetTraceObserver)
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
	result, err := b.executeNPMInstance(ctx, nativeRuntimeBun, target, payload, observer.StartBunTargetLifecycleProfile)
	if err != nil {
		return result, errors.Join(errBunInstanceExecution, err)
	}
	return result, nil
}
