package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const maxResolverEndpoints = 32

// ResolverNetworkPolicy owns the unprivileged portion of one disposable
// resolver network lifecycle. Firewall mutation is intentionally absent: the
// root-owned service is the only implementation of that authority.
type ResolverNetworkPolicy struct {
	runner   CommandRunner
	newID    func() (domain.SandboxSessionID, error)
	service  ResolverPolicyService
	prepared *resolverNetwork
}

type resolverNetwork struct {
	name      string
	session   domain.SandboxSessionID
	networkID string
	subnet    netip.Prefix
	endpoints []netip.Addr
}

func NewResolverNetworkPolicy(runner CommandRunner, service ResolverPolicyService) (*ResolverNetworkPolicy, error) {
	if runner == nil || service == nil {
		return nil, errors.New("resolver network policy service is required")
	}
	return &ResolverNetworkPolicy{runner: runner, newID: domain.NewSandboxSessionID, service: service}, nil
}

// Prepare creates the HAA-labelled bridge, then requests Create and Verify
// from the privileged service before any resolver container exists.
func (p *ResolverNetworkPolicy) Prepare(ctx context.Context, endpoints []netip.Addr) (string, error) {
	if p == nil || p.runner == nil || p.newID == nil || p.service == nil || p.prepared != nil || ctx == nil {
		return "", errors.New("resolver network policy is not ready")
	}
	if err := validateResolverEndpoints(endpoints); err != nil {
		return "", err
	}
	session, err := p.newID()
	if err != nil {
		return "", fmt.Errorf("create resolver network identity: %w", err)
	}
	labels := p.service.NetworkLabels(session)
	if len(labels) != 2 {
		return "", errors.New("resolver network ownership labels are unavailable")
	}
	keys := make([]string, 0, len(labels))
	for key, value := range labels {
		if key == "" || value == "" {
			return "", errors.New("resolver network ownership labels are unavailable")
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	name := "haa-resolver-" + session.String()
	arguments := []string{"network", "create", "--driver", "bridge", "--opt", "com.docker.network.bridge.enable_ipv6=false"}
	for _, key := range keys {
		arguments = append(arguments, "--label", key+"="+labels[key])
	}
	arguments = append(arguments, name)
	created, err := p.runner.Output(ctx, "docker", arguments...)
	networkID := strings.TrimSpace(string(created))
	if err != nil || networkID == "" {
		return "", errors.New("create resolver Docker network failed")
	}
	subnetOutput, err := p.runner.Output(ctx, "docker", "network", "inspect", "--format", "{{range .IPAM.Config}}{{.Subnet}}{{end}}", name)
	subnet, parseErr := netip.ParsePrefix(strings.TrimSpace(string(subnetOutput)))
	if err != nil || parseErr != nil || !subnet.Addr().Is4() {
		return "", errors.Join(errors.New("resolver Docker network subnet is unavailable"), p.removeCreatedNetwork(name))
	}
	network := &resolverNetwork{name: name, session: session, networkID: networkID, subnet: subnet, endpoints: append([]netip.Addr(nil), endpoints...)}
	if err := p.service.Create(ctx, session, networkID, subnet, endpoints); err != nil {
		return "", errors.Join(errors.New("create resolver network policy failed"), p.removeCreatedNetwork(name))
	}
	if err := p.service.Verify(ctx, session, networkID, subnet, endpoints); err != nil {
		cleanupCtx, cancel := resolverCleanupContext()
		removeErr := p.service.Remove(cleanupCtx, session, networkID, subnet, endpoints)
		cancel()
		return "", errors.Join(errors.New("verify resolver network policy failed"), removeErr, p.removeCreatedNetwork(name))
	}
	p.prepared = network
	return name, nil
}

func resolverCleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cleanupTimeout)
}

func (p *ResolverNetworkPolicy) removeCreatedNetwork(name string) error {
	ctx, cancel := resolverCleanupContext()
	defer cancel()
	_, err := p.runner.Output(ctx, "docker", "network", "rm", name)
	return err
}

// Close removes policy before the bridge. Either acknowledgement uncertainty
// is fatal to the caller and cannot become an ALLOW result.
func (p *ResolverNetworkPolicy) Close(ctx context.Context) error {
	if p == nil || p.prepared == nil || ctx == nil {
		return errors.New("resolver network policy is not prepared")
	}
	network := p.prepared
	p.prepared = nil
	policyErr := p.service.Remove(ctx, network.session, network.networkID, network.subnet, network.endpoints)
	_, networkErr := p.runner.Output(ctx, "docker", "network", "rm", network.name)
	if policyErr != nil || networkErr != nil {
		return errors.New("resolver network policy cleanup failed")
	}
	return nil
}

func validateResolverEndpoints(endpoints []netip.Addr) error {
	if len(endpoints) == 0 || len(endpoints) > maxResolverEndpoints {
		return errors.New("resolver endpoint set is invalid")
	}
	seen := map[netip.Addr]bool{}
	for _, endpoint := range endpoints {
		if !endpoint.IsValid() || !endpoint.Is4() || endpoint.IsPrivate() || endpoint.IsLoopback() || endpoint.IsMulticast() || endpoint.IsUnspecified() || seen[endpoint] {
			return errors.New("resolver endpoint set is invalid")
		}
		seen[endpoint] = true
	}
	return nil
}
