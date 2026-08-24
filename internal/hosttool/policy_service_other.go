//go:build !linux

package hosttool

import (
	"errors"

	"github.com/rahoney/heliopause/internal/networkpolicy"
)

func NewSystemNetworkPolicyClient() (networkpolicy.Service, error) {
	return nil, errors.New("privileged resolver policy service requires Linux")
}
