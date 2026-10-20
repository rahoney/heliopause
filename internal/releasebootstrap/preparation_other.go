//go:build !linux

package releasebootstrap

import (
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func (*Preparer) Prepare(context.Context, string) (ports.BootstrapPreparationResult, error) {
	return ports.BootstrapPreparationResult{}, ports.NewBootstrapPreparationFailure(ports.BootstrapUnsupported, errors.New("native bootstrap preparation requires Linux amd64"))
}
