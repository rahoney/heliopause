//go:build linux

package hosttool

import (
	"context"
	"errors"
	"os"
)

// NewSystemInstallExecutor reuses the installed runtime/tool identities and
// adds only systemctl for the separate root installation role. Ordinary
// executors and network-policy requests gain no installation operation.
func NewSystemInstallExecutor(ctx context.Context) (*Executor, error) {
	if ctx == nil || os.Geteuid() != 0 {
		return nil, errors.New("protected installation executor requires root")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	executor, err := NewSystem(ctx)
	if err != nil {
		return nil, err
	}
	control, err := verifySystemExecutable("/usr/bin/systemctl")
	if err != nil {
		return nil, errors.Join(err, executor.Close())
	}
	executor.tools["systemctl"] = control
	return executor, nil
}
