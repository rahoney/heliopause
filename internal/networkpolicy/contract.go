// Package networkpolicy owns the narrow typed boundary between an ordinary
// Heliopause process and the privileged resolver firewall service.
package networkpolicy

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	maxEndpoints    = 32
	networkLabelKey = "io.heliopause.resolver-policy"
	networkLabelVal = "m8"
	sessionLabelKey = "io.heliopause.resolver-session"
)

var dockerIDPattern = regexp.MustCompile(`^[a-f0-9]{12,64}$`)

// Service is the complete authority that an ordinary resolver may request.
// It intentionally has no general command or firewall argument operation.
type Service interface {
	Create(context.Context, ResolverPolicy) error
	Verify(context.Context, ResolverPolicy) error
	Remove(context.Context, ResolverPolicy) error
}

// ResolverPolicy binds one HAA Sandbox Session to one HAA-owned Docker bridge
// and a canonical public IPv4 TCP/443 allow set.
type ResolverPolicy struct {
	session   domain.SandboxSessionID
	networkID string
	subnet    netip.Prefix
	endpoints []netip.Addr
}

// NewResolverPolicy validates and canonicalizes all data that can cross the
// ordinary-user to privileged-service boundary.
func NewResolverPolicy(session domain.SandboxSessionID, networkID string, subnet netip.Prefix, endpoints []netip.Addr) (ResolverPolicy, error) {
	if _, err := domain.ParseSandboxSessionID(session.String()); err != nil || !dockerIDPattern.MatchString(networkID) || !subnet.IsValid() || !subnet.Addr().Is4() || !subnet.Addr().IsPrivate() || subnet.Bits() < 16 || subnet.Bits() > 30 {
		return ResolverPolicy{}, errors.New("resolver policy identity is invalid")
	}
	if len(endpoints) == 0 || len(endpoints) > maxEndpoints {
		return ResolverPolicy{}, errors.New("resolver policy endpoint set is invalid")
	}
	seen := make(map[netip.Addr]struct{}, len(endpoints))
	canonical := make([]netip.Addr, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if !endpoint.IsValid() || !endpoint.Is4() || endpoint.IsPrivate() || endpoint.IsLoopback() || endpoint.IsMulticast() || endpoint.IsUnspecified() {
			return ResolverPolicy{}, errors.New("resolver policy endpoint set is invalid")
		}
		if _, duplicate := seen[endpoint]; duplicate {
			return ResolverPolicy{}, errors.New("resolver policy endpoint set is invalid")
		}
		seen[endpoint] = struct{}{}
		canonical = append(canonical, endpoint)
	}
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Less(canonical[j]) })
	return ResolverPolicy{session: session, networkID: networkID, subnet: subnet.Masked(), endpoints: canonical}, nil
}

func (p ResolverPolicy) Session() domain.SandboxSessionID { return p.session }
func (p ResolverPolicy) NetworkID() string                { return p.networkID }
func (p ResolverPolicy) Subnet() netip.Prefix             { return p.subnet }
func (p ResolverPolicy) Endpoints() []netip.Addr          { return append([]netip.Addr(nil), p.endpoints...) }

// NetworkName is derived from the typed session; the client never supplies an
// arbitrary Docker network name to the privileged service.
func NetworkName(session domain.SandboxSessionID) string {
	return "haa-resolver-" + session.String()
}

func NetworkLabels(session domain.SandboxSessionID) map[string]string {
	return map[string]string{networkLabelKey: networkLabelVal, sessionLabelKey: session.String()}
}

func NetworkLabelKey() string   { return networkLabelKey }
func NetworkLabelValue() string { return networkLabelVal }
func SessionLabelKey() string   { return sessionLabelKey }
