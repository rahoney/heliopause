package hosttool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"github.com/rahoney/heliopause/internal/networkpolicy"
)

type policyRunner interface {
	Output(context.Context, string, ...string) ([]byte, error)
}

type policyBackend string

const (
	policyIPTables policyBackend = "iptables"
	policyNFTables policyBackend = "nftables"
)

// PolicyEngine is reachable only from the root-owned helper. It converts an
// already-typed policy into fixed firewall invocations; no arbitrary command
// data is accepted at this layer.
type PolicyEngine struct {
	runner   policyRunner
	mu       sync.Mutex
	sessions map[string]policyRecord
}

type policyRecord struct {
	uid     uint32
	policy  networkpolicy.ResolverPolicy
	backend policyBackend
	chain   string
}

func NewPolicyEngine(runner policyRunner) (*PolicyEngine, error) {
	if runner == nil {
		return nil, errors.New("network policy runner is required")
	}
	return &PolicyEngine{runner: runner, sessions: make(map[string]policyRecord)}, nil
}

func (e *PolicyEngine) CreateForPeer(ctx context.Context, uid uint32, policy networkpolicy.ResolverPolicy) error {
	if e == nil || ctx == nil {
		return errors.New("network policy engine is unavailable")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	key := policy.Session().String()
	if _, exists := e.sessions[key]; exists {
		return errors.New("network policy session is duplicate")
	}
	if err := e.verifyNetwork(ctx, policy); err != nil {
		return err
	}
	backend, err := e.detectBackend(ctx)
	if err != nil {
		return err
	}
	record := policyRecord{uid: uid, policy: policy, backend: backend, chain: policyChain(policy)}
	if err := e.apply(ctx, record); err != nil {
		_ = e.remove(ctx, record)
		return err
	}
	if err := e.verify(ctx, record); err != nil {
		_ = e.remove(ctx, record)
		return err
	}
	e.sessions[key] = record
	return nil
}

func (e *PolicyEngine) VerifyForPeer(ctx context.Context, uid uint32, policy networkpolicy.ResolverPolicy) error {
	if e == nil || ctx == nil {
		return errors.New("network policy engine is unavailable")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	record, ok := e.sessions[policy.Session().String()]
	if !ok || record.uid != uid || !samePolicy(record.policy, policy) {
		return errors.New("network policy session is unconfirmed")
	}
	if err := e.verifyNetwork(ctx, policy); err != nil {
		return err
	}
	return e.verify(ctx, record)
}

func (e *PolicyEngine) RemoveForPeer(ctx context.Context, uid uint32, policy networkpolicy.ResolverPolicy) error {
	if e == nil || ctx == nil {
		return errors.New("network policy engine is unavailable")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	key := policy.Session().String()
	record, ok := e.sessions[key]
	if !ok || record.uid != uid || !samePolicy(record.policy, policy) {
		return errors.New("network policy session is unconfirmed")
	}
	if err := e.remove(ctx, record); err != nil {
		return err
	}
	delete(e.sessions, key)
	return nil
}

func (e *PolicyEngine) detectBackend(ctx context.Context) (policyBackend, error) {
	output, err := e.runner.Output(ctx, "docker", "info", "--format", "{{.FirewallBackend.Driver}}")
	if err == nil {
		switch policyBackend(strings.TrimSpace(string(output))) {
		case policyIPTables, policyNFTables:
			return policyBackend(strings.TrimSpace(string(output))), nil
		}
	}
	if _, iptablesErr := e.runner.Output(ctx, "iptables", "-S", "DOCKER-USER"); iptablesErr == nil {
		return policyIPTables, nil
	}
	if _, nftErr := e.runner.Output(ctx, "nft", "list", "table", "ip", "docker-bridges"); nftErr == nil {
		return policyNFTables, nil
	}
	return "", errors.New("network policy firewall backend is unavailable")
}

func (e *PolicyEngine) verifyNetwork(ctx context.Context, policy networkpolicy.ResolverPolicy) error {
	body, err := e.runner.Output(ctx, "docker", "network", "inspect", "--format", "{{json .}}", networkpolicy.NetworkName(policy.Session()))
	if err != nil || len(body) == 0 || len(body) > 64*1024 {
		return errors.New("resolver network is unavailable")
	}
	var network struct {
		ID         string            `json:"Id"`
		Name       string            `json:"Name"`
		Labels     map[string]string `json:"Labels"`
		Containers map[string]any    `json:"Containers"`
		IPAM       struct {
			Config []struct {
				Subnet string `json:"Subnet"`
			} `json:"Config"`
		} `json:"IPAM"`
	}
	if json.Unmarshal(body, &network) != nil || network.ID != policy.NetworkID() || network.Name != networkpolicy.NetworkName(policy.Session()) || network.Labels[networkpolicy.NetworkLabelKey()] != networkpolicy.NetworkLabelValue() || network.Labels[networkpolicy.SessionLabelKey()] != policy.Session().String() || len(network.Containers) != 0 || len(network.IPAM.Config) != 1 {
		return errors.New("resolver network identity is unconfirmed")
	}
	subnet, parseErr := netip.ParsePrefix(network.IPAM.Config[0].Subnet)
	if parseErr != nil || subnet.Masked() != policy.Subnet() {
		return errors.New("resolver network subnet is unconfirmed")
	}
	return nil
}

func (e *PolicyEngine) apply(ctx context.Context, record policyRecord) error {
	if record.backend == policyIPTables {
		if _, err := e.runner.Output(ctx, "iptables", "-N", record.chain); err != nil {
			return errors.New("create resolver firewall policy failed")
		}
		if _, err := e.runner.Output(ctx, "iptables", "-I", "DOCKER-USER", "1", "-s", record.policy.Subnet().String(), "-j", record.chain); err != nil {
			return errors.New("attach resolver firewall policy failed")
		}
		for _, endpoint := range record.policy.Endpoints() {
			if _, err := e.runner.Output(ctx, "iptables", "-A", record.chain, "-d", endpoint.String(), "-p", "tcp", "--dport", "443", "-j", "ACCEPT"); err != nil {
				return errors.New("add resolver firewall allow failed")
			}
		}
		if _, err := e.runner.Output(ctx, "iptables", "-A", record.chain, "-j", "DROP"); err != nil {
			return errors.New("add resolver firewall deny failed")
		}
		return nil
	}
	table := strings.ToLower(record.chain)
	if _, err := e.runner.Output(ctx, "nft", "add", "table", "ip", table); err != nil {
		return errors.New("create resolver firewall policy failed")
	}
	if _, err := e.runner.Output(ctx, "nft", "add", "chain", "ip", table, "forward", "{ type filter hook forward priority filter - 1 ; policy accept ; }"); err != nil {
		return errors.New("create resolver firewall policy failed")
	}
	for _, endpoint := range record.policy.Endpoints() {
		if _, err := e.runner.Output(ctx, "nft", "add", "rule", "ip", table, "forward", "ip", "saddr", record.policy.Subnet().String(), "ip", "daddr", endpoint.String(), "tcp", "dport", "443", "accept"); err != nil {
			return errors.New("add resolver firewall allow failed")
		}
	}
	_, err := e.runner.Output(ctx, "nft", "add", "rule", "ip", table, "forward", "ip", "saddr", record.policy.Subnet().String(), "drop")
	return err
}

func (e *PolicyEngine) verify(ctx context.Context, record policyRecord) error {
	if record.backend == policyIPTables {
		if _, err := e.runner.Output(ctx, "iptables", "-C", "DOCKER-USER", "-s", record.policy.Subnet().String(), "-j", record.chain); err != nil {
			return errors.New("verify resolver firewall policy failed")
		}
		_, err := e.runner.Output(ctx, "iptables", "-S", record.chain)
		return err
	}
	_, err := e.runner.Output(ctx, "nft", "list", "table", "ip", strings.ToLower(record.chain))
	return err
}

func (e *PolicyEngine) remove(ctx context.Context, record policyRecord) error {
	if record.backend == policyIPTables {
		if _, err := e.runner.Output(ctx, "iptables", "-D", "DOCKER-USER", "-s", record.policy.Subnet().String(), "-j", record.chain); err != nil {
			return errors.New("remove resolver firewall policy failed")
		}
		if _, err := e.runner.Output(ctx, "iptables", "-F", record.chain); err != nil {
			return errors.New("remove resolver firewall policy failed")
		}
		if _, err := e.runner.Output(ctx, "iptables", "-X", record.chain); err != nil {
			return errors.New("remove resolver firewall policy failed")
		}
		return nil
	}
	_, err := e.runner.Output(ctx, "nft", "delete", "table", "ip", strings.ToLower(record.chain))
	return err
}

func policyChain(policy networkpolicy.ResolverPolicy) string {
	return "HAA_R_" + strings.ToUpper(strings.TrimPrefix(policy.Session().String(), "sbx_")[:16])
}

func samePolicy(left, right networkpolicy.ResolverPolicy) bool {
	return left.Session().String() == right.Session().String() && left.NetworkID() == right.NetworkID() && left.Subnet() == right.Subnet() && fmt.Sprint(left.Endpoints()) == fmt.Sprint(right.Endpoints())
}
