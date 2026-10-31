package sandbox

import (
	"context"
	"errors"
)

// Preserve the qualified PNPM surface and its typed preparation seam.
func (r *PNPMResolver) withNativeProject(ctx context.Context, earlyStore bool, action func(context.Context, string, string, *closureVolume) error) error {
	if r == nil || r.prepare == nil {
		return errors.New("pnpm native infrastructure unavailable")
	}
	scope := nativeProjectScope{kind: nativeRuntimePNPM, runner: r.runner, endpoints: r.endpoints, observer: r.observer, probe: r.probe, policy: r.policy, prepare: func(ctx context.Context) (nativeProjectTool, error) { return r.prepare(ctx) }}
	return scope.run(ctx, earlyStore, action)
}
