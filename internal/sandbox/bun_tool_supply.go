package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const bunToolGuestPath = "/haa-runtime/bun"

var errBunToolSupply = errors.New("bun immutable native tool supply is unavailable or changed")

type bunPreparedTool struct{ body []byte }

// prepareBunTool reads the fixed protected installation as data. The pin was
// adopted through the upstream signed-checksum/archive verifier and signed release
// producer. This reader neither executes a Host tool nor accepts a user path.
func prepareBunTool(ctx context.Context) (*bunPreparedTool, error) {
	return readInstalledBunTool(ctx, runtimeidentity.NativeInstallRoot)
}

func readInstalledBunTool(ctx context.Context, installation string) (*bunPreparedTool, error) {
	body, err := readInstalledNativeTool(ctx, installation, nativeRuntimeBun)
	if err != nil {
		return nil, errors.Join(errBunToolSupply, err)
	}
	return &bunPreparedTool{body: body}, nil
}

func (p *bunPreparedTool) introduce(ctx context.Context, runner CommandRunner, container string) error {
	if p == nil {
		return errBunToolSupply
	}
	if err := introduceNativeTool(ctx, runner, container, nativeRuntimeBun, p.body); err != nil {
		return errors.Join(errBunToolSupply, err)
	}
	return nil
}
