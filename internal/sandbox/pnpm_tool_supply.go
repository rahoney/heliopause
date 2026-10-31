package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const pnpmToolGuestPath = "/haa-runtime/pnpm"

var errPNPMToolSupply = errors.New("pnpm immutable native tool supply is unavailable or changed")

type pnpmPreparedTool struct{ body []byte }

// preparePNPMTool reads the fixed protected installation as data. The pin was
// adopted through the upstream provenance/archive verifier and signed release
// producer. This reader neither executes a Host tool nor accepts a user path.
func preparePNPMTool(ctx context.Context) (*pnpmPreparedTool, error) {
	return readInstalledPNPMTool(ctx, runtimeidentity.NativeInstallRoot)
}

func readInstalledPNPMTool(ctx context.Context, installation string) (*pnpmPreparedTool, error) {
	body, err := readInstalledNativeTool(ctx, installation, nativeRuntimePNPM)
	if err != nil {
		return nil, errors.Join(errPNPMToolSupply, err)
	}
	return &pnpmPreparedTool{body: body}, nil
}

func (p *pnpmPreparedTool) introduce(ctx context.Context, runner CommandRunner, container string) error {
	if p == nil {
		return errPNPMToolSupply
	}
	if err := introduceNativeTool(ctx, runner, container, nativeRuntimePNPM, p.body); err != nil {
		return errors.Join(errPNPMToolSupply, err)
	}
	return nil
}

func (*pnpmPreparedTool) nativeKind() nativeRuntimeToolKind { return nativeRuntimePNPM }
