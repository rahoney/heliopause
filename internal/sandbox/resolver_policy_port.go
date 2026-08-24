package sandbox

import (
	"context"
	"net/netip"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// ResolverPolicyService is the Sandbox infrastructure port for the narrow
// privileged firewall lifecycle. It deliberately contains no tool path,
// command fragment, firewall backend or Host credential detail.
type ResolverPolicyService interface {
	// NetworkLabels supplies the fixed ownership labels required before the
	// privileged service will accept a Docker network. These are installation
	// policy, not resolver input.
	NetworkLabels(domain.SandboxSessionID) map[string]string
	Create(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error
	Verify(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error
	Remove(context.Context, domain.SandboxSessionID, string, netip.Prefix, []netip.Addr) error
}
