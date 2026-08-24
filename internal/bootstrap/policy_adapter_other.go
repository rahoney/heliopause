//go:build !linux

package bootstrap

import (
	"errors"

	"github.com/rahoney/heliopause/internal/sandbox"
)

func newSystemResolverPolicyAdapter() (sandbox.ResolverPolicyService, error) {
	return nil, errors.New("resolver network policy service requires Linux")
}
