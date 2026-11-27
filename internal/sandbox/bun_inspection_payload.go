package sandbox

import (
	"context"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

// These are execution data, never original controls, native cache or approval.
func prepareBunInspectionPayload(ctx context.Context, layout artifactnpm.BunInspectionLayout) (pnpmInspectionPayload, error) {
	if layout.RuntimeIdentity() != BunResolverRuntimeIdentity() {
		return pnpmInspectionPayload{}, errPNPMInspectionPayload
	}
	return prepareNPMInspectionPayload(ctx, layout, nativeRuntimeBun)
}

func copyBunInspectionPayload(ctx context.Context, runner CommandRunner, container string, payload pnpmInspectionPayload) error {
	return copyNPMInspectionPayload(ctx, runner, container, payload, nativeRuntimeBun)
}
