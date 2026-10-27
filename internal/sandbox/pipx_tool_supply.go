package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const pipxToolGuestPath = "/haa-pipx/pipx-runtime.zip"

var errPipxToolSupply = errors.New("pipx immutable ZIP supply is unavailable or changed")

type pipxPreparedTool struct{ body []byte }

// preparePipxTool reads only the fixed protected ZIP as DATA. Supply identity
// does not authorize Python execution, release activation or a writable mount.
func preparePipxTool(ctx context.Context) (*pipxPreparedTool, error) {
	return readInstalledPipxTool(ctx, runtimeidentity.NativeInstallRoot)
}

func readInstalledPipxTool(ctx context.Context, installation string) (*pipxPreparedTool, error) {
	body, err := readInstalledNativeTool(ctx, installation, nativeRuntimePipx)
	if err != nil {
		return nil, errors.Join(errPipxToolSupply, err)
	}
	return &pipxPreparedTool{body: body}, nil
}

func (p *pipxPreparedTool) introduce(ctx context.Context, runner CommandRunner, container string) error {
	if p == nil {
		return errPipxToolSupply
	}
	if err := introduceNativeTool(ctx, runner, container, nativeRuntimePipx, p.body); err != nil {
		return errors.Join(errPipxToolSupply, err)
	}
	return nil
}
