package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const maxResolverEndpoints = 32

type firewallBackend string

const (
	firewallBackendIPTables firewallBackend = "iptables"
	firewallBackendNFTables firewallBackend = "nftables"
)

// ResolverNetworkPolicy owns the Linux network and firewall lifecycle for one
// disposable npm resolver. It accepts resolved endpoint IPs only; hostname and
// package-manager details remain outside this infrastructure boundary.
type ResolverNetworkPolicy struct {
	runner   CommandRunner
	newID    func() (domain.SandboxSessionID, error)
	backend  firewallBackend
	prepared *resolverNetwork
}

type resolverNetwork struct {
	name      string
	subnet    netip.Prefix
	chainName string
	backend   firewallBackend
}

// NewResolverNetworkPolicy probes Docker's active firewall backend. Unknown,
// unsupported, or unverified backends are rejected rather than guessed.
func NewResolverNetworkPolicy(ctx context.Context, runner CommandRunner) (*ResolverNetworkPolicy, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if runner == nil {
		return nil, errors.New("resolver network policy requires a command runner")
	}
	output, err := runner.Output(ctx, "docker", "info", "--format", "{{.FirewallBackend}}")
	if err != nil {
		return nil, errors.New("resolver firewall backend is unavailable")
	}
	backend := firewallBackend(strings.TrimSpace(string(output)))
	if backend == "<no value>" {
		backend = ""
	}
	if backend != "" && backend != firewallBackendIPTables && backend != firewallBackendNFTables {
		return nil, errors.New("resolver firewall backend is unsupported")
	}
	return &ResolverNetworkPolicy{runner: runner, newID: domain.NewSandboxSessionID, backend: backend}, nil
}

func probeDockerFirewallBackend(ctx context.Context, runner CommandRunner) (firewallBackend, error) {
	// Docker's documented default is iptables. Some Docker 29 installations do
	// not expose FirewallBackend in `docker info`, while iptables-nft can make
	// unrelated nftables tables visible at the same time. DOCKER-USER is the
	// authoritative, Docker-owned insertion point for the iptables backend, so
	// prefer it when it is present and directly queryable. Do not infer a
	// backend from the mere presence of an nftables table.
	_, iptablesErr := runner.Output(ctx, "iptables", "-S", "DOCKER-USER")
	if iptablesErr == nil {
		return firewallBackendIPTables, nil
	}
	_, nftablesErr := runner.Output(ctx, "nft", "list", "table", "ip", "docker-bridges")
	if nftablesErr == nil {
		return firewallBackendNFTables, nil
	}
	return "", errors.Join(
		errors.New("resolver firewall backend is unsupported"),
		fmt.Errorf("iptables DOCKER-USER probe: %w", iptablesErr),
		fmt.Errorf("nftables docker-bridges probe: %w", nftablesErr),
	)
}

// Prepare creates an isolated Docker network then applies and verifies a
// default-deny egress policy for the supplied trusted endpoint IP set.
func (p *ResolverNetworkPolicy) Prepare(ctx context.Context, endpoints []netip.Addr) (string, error) {
	if p == nil || p.runner == nil || p.newID == nil || p.prepared != nil {
		return "", errors.New("resolver network policy is not ready")
	}
	if ctx == nil {
		return "", errors.New("context is required")
	}
	if err := validateResolverEndpoints(endpoints); err != nil {
		return "", err
	}
	id, err := p.newID()
	if err != nil {
		return "", fmt.Errorf("create resolver network identity: %w", err)
	}
	name := "haa-resolver-" + id.String()
	created, err := p.runner.Output(ctx, "docker", "network", "create", "--driver", "bridge", "--opt", "com.docker.network.bridge.enable_ipv6=false", name)
	if err != nil || strings.TrimSpace(string(created)) == "" {
		return "", errors.New("create resolver Docker network failed")
	}
	subnetOutput, err := p.runner.Output(ctx, "docker", "network", "inspect", "--format", "{{range .IPAM.Config}}{{.Subnet}}{{end}}", name)
	subnet, parseErr := netip.ParsePrefix(strings.TrimSpace(string(subnetOutput)))
	if err != nil || parseErr != nil || !subnet.Addr().Is4() {
		_ = p.removeNetwork(context.Background(), name)
		return "", errors.New("resolver Docker network subnet is unavailable")
	}
	backend := p.backend
	if backend == "" {
		// Docker creates its firewall hooks lazily when the first bridge network
		// is created. Probe only after that network exists; no resolver container
		// has been started at this point, so this ordering cannot expose egress.
		backend, err = probeDockerFirewallBackend(ctx, p.runner)
		if err != nil {
			_ = p.removeNetwork(context.Background(), name)
			return "", err
		}
	}
	chainSuffix := strings.ToUpper(strings.TrimPrefix(id.String(), "sbx_"))
	network := &resolverNetwork{name: name, subnet: subnet, chainName: "HAA_R_" + chainSuffix[:16], backend: backend}
	if err := p.applyAndVerify(ctx, network, endpoints); err != nil {
		cleanupErr := p.cleanupNetwork(context.Background(), network)
		if cleanupErr != nil {
			return "", errors.Join(err, cleanupErr)
		}
		return "", err
	}
	p.prepared = network
	return name, nil
}

// Close removes the firewall policy before deleting its Docker network. A
// cleanup failure remains an error so callers can fail closed.
func (p *ResolverNetworkPolicy) Close(ctx context.Context) error {
	if p == nil || p.prepared == nil {
		return errors.New("resolver network policy is not prepared")
	}
	if ctx == nil {
		return errors.New("context is required")
	}
	network := p.prepared
	p.prepared = nil
	return p.cleanupNetwork(ctx, network)
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

func (p *ResolverNetworkPolicy) applyAndVerify(ctx context.Context, network *resolverNetwork, endpoints []netip.Addr) error {
	switch network.backend {
	case firewallBackendIPTables:
		return p.applyIPTables(ctx, network, endpoints)
	case firewallBackendNFTables:
		return p.applyNFTables(ctx, network, endpoints)
	default:
		return errors.New("resolver firewall backend is unsupported")
	}
}

func (p *ResolverNetworkPolicy) applyIPTables(ctx context.Context, network *resolverNetwork, endpoints []netip.Addr) error {
	if _, err := p.runner.Output(ctx, "iptables", "-N", network.chainName); err != nil {
		return errors.New("create resolver iptables chain failed")
	}
	if _, err := p.runner.Output(ctx, "iptables", "-I", "DOCKER-USER", "1", "-s", network.subnet.String(), "-j", network.chainName); err != nil {
		return errors.New("attach resolver iptables chain failed")
	}
	for _, endpoint := range endpoints {
		if _, err := p.runner.Output(ctx, "iptables", "-A", network.chainName, "-d", endpoint.String(), "-p", "tcp", "--dport", "443", "-j", "ACCEPT"); err != nil {
			return errors.New("add resolver allow rule failed")
		}
	}
	if _, err := p.runner.Output(ctx, "iptables", "-A", network.chainName, "-j", "DROP"); err != nil {
		return errors.New("add resolver deny rule failed")
	}
	if _, err := p.runner.Output(ctx, "iptables", "-C", "DOCKER-USER", "-s", network.subnet.String(), "-j", network.chainName); err != nil {
		return errors.New("verify resolver iptables attachment failed")
	}
	if _, err := p.runner.Output(ctx, "iptables", "-S", network.chainName); err != nil {
		return errors.New("verify resolver iptables rules failed")
	}
	return nil
}

func (p *ResolverNetworkPolicy) applyNFTables(ctx context.Context, network *resolverNetwork, endpoints []netip.Addr) error {
	table := strings.ToLower(network.chainName)
	if _, err := p.runner.Output(ctx, "nft", "add", "table", "ip", table); err != nil {
		return errors.New("create resolver nftables table failed")
	}
	chain := "{ type filter hook forward priority filter - 1 ; policy accept ; }"
	if _, err := p.runner.Output(ctx, "nft", "add", "chain", "ip", table, "forward", chain); err != nil {
		return errors.New("create resolver nftables chain failed")
	}
	for _, endpoint := range endpoints {
		if _, err := p.runner.Output(ctx, "nft", "add", "rule", "ip", table, "forward", "ip", "saddr", network.subnet.String(), "ip", "daddr", endpoint.String(), "tcp", "dport", "443", "accept"); err != nil {
			return errors.New("add resolver nftables allow rule failed")
		}
	}
	if _, err := p.runner.Output(ctx, "nft", "add", "rule", "ip", table, "forward", "ip", "saddr", network.subnet.String(), "drop"); err != nil {
		return errors.New("add resolver nftables deny rule failed")
	}
	if _, err := p.runner.Output(ctx, "nft", "list", "table", "ip", table); err != nil {
		return errors.New("verify resolver nftables rules failed")
	}
	return nil
}

func (p *ResolverNetworkPolicy) cleanupNetwork(ctx context.Context, network *resolverNetwork) error {
	var policyErr error
	if network.backend == firewallBackendIPTables {
		_, policyErr = p.runner.Output(ctx, "iptables", "-D", "DOCKER-USER", "-s", network.subnet.String(), "-j", network.chainName)
		_, _ = p.runner.Output(ctx, "iptables", "-F", network.chainName)
		_, _ = p.runner.Output(ctx, "iptables", "-X", network.chainName)
	} else {
		_, policyErr = p.runner.Output(ctx, "nft", "delete", "table", "ip", strings.ToLower(network.chainName))
	}
	networkErr := p.removeNetwork(ctx, network.name)
	if policyErr != nil || networkErr != nil {
		return errors.New("resolver network policy cleanup failed")
	}
	return nil
}

func (p *ResolverNetworkPolicy) removeNetwork(ctx context.Context, name string) error {
	_, err := p.runner.Output(ctx, "docker", "network", "rm", name)
	return err
}
