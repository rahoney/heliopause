//go:build linux

package sandbox

import (
	"context"
	"net/netip"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/networkpolicy"
)

// newIntegrationResolverPolicyService keeps the resolver test process
// ordinary-user while delegating firewall authority through the installed
// root-owned typed network-policy helper.
func newIntegrationResolverPolicyService(t *testing.T) ResolverPolicyService {
	t.Helper()
	service, err := hosttool.NewSystemNetworkPolicyClient()
	if err != nil {
		t.Fatal(err)
	}
	return systemResolverPolicyService{service: service}
}

type systemResolverPolicyService struct {
	service networkpolicy.Service
}

func (s systemResolverPolicyService) NetworkLabels(session domain.SandboxSessionID) map[string]string {
	return networkpolicy.NetworkLabels(session)
}

func (s systemResolverPolicyService) Create(ctx context.Context, session domain.SandboxSessionID, networkID string, subnet netip.Prefix, endpoints []netip.Addr) error {
	policy, err := networkpolicy.NewResolverPolicy(session, networkID, subnet, endpoints)
	if err != nil {
		return err
	}
	return s.service.Create(ctx, policy)
}

func (s systemResolverPolicyService) Verify(ctx context.Context, session domain.SandboxSessionID, networkID string, subnet netip.Prefix, endpoints []netip.Addr) error {
	policy, err := networkpolicy.NewResolverPolicy(session, networkID, subnet, endpoints)
	if err != nil {
		return err
	}
	return s.service.Verify(ctx, policy)
}

func (s systemResolverPolicyService) Remove(ctx context.Context, session domain.SandboxSessionID, networkID string, subnet netip.Prefix, endpoints []netip.Addr) error {
	policy, err := networkpolicy.NewResolverPolicy(session, networkID, subnet, endpoints)
	if err != nil {
		return err
	}
	return s.service.Remove(ctx, policy)
}
