package sandbox

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

func TestResolverNetworkPolicyIPTablesCreatesVerifiesAndCleansUp(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{
		[]byte("iptables"), []byte("network-id"), []byte("172.30.0.0/24"),
	}}
	policy, err := NewResolverNetworkPolicy(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	name, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil || name == "" {
		t.Fatalf("Prepare() = %q, %v", name, err)
	}
	if len(runner.calls) != 9 {
		t.Fatalf("prepare calls = %#v", runner.calls)
	}
	if got := runner.calls[3]; got.binary != "iptables" || got.arguments[0] != "-N" {
		t.Fatalf("chain create = %#v", got)
	}
	if got := runner.calls[4]; got.binary != "iptables" || got.arguments[0] != "-I" || got.arguments[1] != "DOCKER-USER" {
		t.Fatalf("chain attach = %#v", got)
	}
	if got := runner.calls[6]; got.binary != "iptables" || got.arguments[0] != "-A" || got.arguments[len(got.arguments)-1] != "DROP" {
		t.Fatalf("default deny = %#v", got)
	}
	if err := policy.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[len(runner.calls)-1]; got.binary != "docker" || !sameStrings(got.arguments, []string{"network", "rm", name}) {
		t.Fatalf("network cleanup = %#v", got)
	}
}

func TestResolverNetworkPolicyRejectsUnknownBackendAndFailedCleanup(t *testing.T) {
	unknown := &recordingRunner{responses: [][]byte{[]byte("unknown")}}
	if _, err := NewResolverNetworkPolicy(context.Background(), unknown); err == nil {
		t.Fatal("NewResolverNetworkPolicy() accepted an unknown backend")
	}

	runner := &recordingRunner{
		responses: [][]byte{[]byte("iptables"), []byte("network-id"), []byte("172.30.0.0/24")},
		errors:    []error{nil, nil, nil, nil, nil, nil, nil, nil, errors.New("detach failed")},
	}
	policy, err := NewResolverNetworkPolicy(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")}); err == nil {
		t.Fatal("Prepare() accepted a policy verification failure")
	}
}

func TestResolverNetworkPolicyNFTablesUsesSeparateTable(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{
		[]byte("nftables"), []byte("network-id"), []byte("172.30.0.0/24"),
	}}
	policy, err := NewResolverNetworkPolicy(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	name, err := policy.Prepare(context.Background(), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[3]; got.binary != "nft" || !sameStrings(got.arguments[:4], []string{"add", "table", "ip", got.arguments[3]}) {
		t.Fatalf("nft table create = %#v", got)
	}
	if got := runner.calls[4]; got.binary != "nft" || got.arguments[0] != "add" || got.arguments[1] != "chain" {
		t.Fatalf("nft chain create = %#v", got)
	}
	if got := runner.calls[6]; got.binary != "nft" || got.arguments[len(got.arguments)-1] != "drop" {
		t.Fatalf("nft default deny = %#v", got)
	}
	if err := policy.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := runner.calls[len(runner.calls)-2]; got.binary != "nft" || !sameStrings(got.arguments[:3], []string{"delete", "table", "ip"}) {
		t.Fatalf("nft table cleanup = %#v", got)
	}
	if got := runner.calls[len(runner.calls)-1]; got.binary != "docker" || !sameStrings(got.arguments, []string{"network", "rm", name}) {
		t.Fatalf("network cleanup = %#v", got)
	}
}

func TestValidateResolverEndpointsRejectsUnsafeOrAmbiguousInputs(t *testing.T) {
	for _, endpoints := range [][]netip.Addr{
		nil,
		{netip.MustParseAddr("127.0.0.1")},
		{netip.MustParseAddr("10.0.0.1")},
		{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.1.1.1")},
		{netip.MustParseAddr("2606:4700:4700::1111")},
	} {
		if err := validateResolverEndpoints(endpoints); err == nil {
			t.Fatalf("validateResolverEndpoints(%v) error = nil", endpoints)
		}
	}
}
