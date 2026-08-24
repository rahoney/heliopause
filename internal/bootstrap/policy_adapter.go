//go:build linux

package bootstrap

import (
	"context"
	"net/netip"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/networkpolicy"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func newSystemResolverPolicyAdapter() (sandbox.ResolverPolicyService, error) {
	service, err := hosttool.NewSystemNetworkPolicyClient()
	if err != nil {
		return nil, err
	}
	return resolverPolicyAdapter{service}, nil
}

type resolverPolicyAdapter struct{ service networkpolicy.Service }

func (resolverPolicyAdapter) NetworkLabels(session domain.SandboxSessionID) map[string]string {
	return networkpolicy.NetworkLabels(session)
}

func (a resolverPolicyAdapter) Create(c context.Context, s domain.SandboxSessionID, id string, n netip.Prefix, e []netip.Addr) error {
	p, err := networkpolicy.NewResolverPolicy(s, id, n, e)
	if err != nil {
		return err
	}
	return a.service.Create(c, p)
}

func (a resolverPolicyAdapter) Verify(c context.Context, s domain.SandboxSessionID, id string, n netip.Prefix, e []netip.Addr) error {
	p, err := networkpolicy.NewResolverPolicy(s, id, n, e)
	if err != nil {
		return err
	}
	return a.service.Verify(c, p)
}

func (a resolverPolicyAdapter) Remove(c context.Context, s domain.SandboxSessionID, id string, n netip.Prefix, e []netip.Addr) error {
	p, err := networkpolicy.NewResolverPolicy(s, id, n, e)
	if err != nil {
		return err
	}
	return a.service.Remove(c, p)
}
