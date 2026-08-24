package hosttool

import (
	"context"
	"net/netip"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/networkpolicy"
)

func TestPolicyEngineUsesOnlyFixedPolicyOperations(t *testing.T) {
	session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	policy, _ := networkpolicy.NewResolverPolicy(session, "0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	network := `{"Id":"0123456789abcdef","Name":"haa-resolver-sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","Labels":{"io.heliopause.resolver-policy":"m8","io.heliopause.resolver-session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa"},"Containers":{},"IPAM":{"Config":[{"Subnet":"172.30.0.0/24"}]}}`
	runner := &policyRecordingRunner{responses: [][]byte{[]byte(network), []byte("iptables"), nil, nil, nil, nil, nil, nil, []byte(network), nil, nil, nil, nil}}
	engine, err := NewPolicyEngine(runner)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.CreateForPeer(context.Background(), 1001, policy); err != nil {
		t.Fatal(err)
	}
	if err := engine.VerifyForPeer(context.Background(), 1001, policy); err != nil {
		t.Fatal(err)
	}
	if err := engine.RemoveForPeer(context.Background(), 1001, policy); err != nil {
		t.Fatal(err)
	}
	for _, call := range runner.calls {
		if call.name != "docker" && call.name != "iptables" && call.name != "nft" {
			t.Fatalf("unexpected executable %q", call.name)
		}
		if strings.Contains(strings.Join(call.args, " "), "1.1.1.1;") {
			t.Fatalf("raw command reached engine: %#v", call)
		}
	}
}

func TestPolicyEngineRejectsDuplicateAndWrongPeer(t *testing.T) {
	session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	policy, _ := networkpolicy.NewResolverPolicy(session, "0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	network := `{"Id":"0123456789abcdef","Name":"haa-resolver-sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","Labels":{"io.heliopause.resolver-policy":"m8","io.heliopause.resolver-session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa"},"Containers":{},"IPAM":{"Config":[{"Subnet":"172.30.0.0/24"}]}}`
	runner := &policyRecordingRunner{responses: [][]byte{[]byte(network), []byte("iptables"), nil, nil, nil, nil, nil}}
	engine, _ := NewPolicyEngine(runner)
	if err := engine.CreateForPeer(context.Background(), 1001, policy); err != nil {
		t.Fatal(err)
	}
	if err := engine.CreateForPeer(context.Background(), 1001, policy); err == nil {
		t.Fatal("duplicate policy accepted")
	}
	if err := engine.VerifyForPeer(context.Background(), 1002, policy); err == nil {
		t.Fatal("wrong peer accepted")
	}
}

type policyCall struct {
	name string
	args []string
}
type policyRecordingRunner struct {
	responses [][]byte
	calls     []policyCall
}

func (r *policyRecordingRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, policyCall{name, append([]string(nil), args...)})
	if len(r.responses) == 0 {
		return nil, nil
	}
	result := r.responses[0]
	r.responses = r.responses[1:]
	return result, nil
}
