package networkpolicy

import (
	"net/netip"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNewResolverPolicyCanonicalizesOnlyBoundedPublicIPv4(t *testing.T) {
	session, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := NewResolverPolicy(session, "0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.2"), netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	if got := policy.Endpoints(); len(got) != 2 || got[0] != netip.MustParseAddr("1.1.1.1") || NetworkName(session) != "haa-resolver-sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("canonical policy = %#v", policy)
	}
	for _, value := range []string{"100.63.255.255", "100.128.0.1", "192.0.0.9", "192.0.0.10"} {
		if _, err := NewResolverPolicy(session, "0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr(value)}); err != nil {
			t.Fatalf("public IPv4 %s rejected: %v", value, err)
		}
	}
}

func TestNewResolverPolicyRejectsUntrustedControlInputs(t *testing.T) {
	session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	for _, test := range []struct {
		id        string
		subnet    netip.Prefix
		endpoints []netip.Addr
	}{
		{"not-a-docker-id", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("8.8.8.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("10.0.0.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("169.254.169.254")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("255.255.255.255")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("100.100.100.200")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("0.1.2.3")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("198.18.0.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("240.0.0.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("192.0.0.170")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("192.0.2.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("192.88.99.2")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("198.51.100.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("203.0.113.1")}},
		{"0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.1.1.1")}},
	} {
		if _, err := NewResolverPolicy(session, test.id, test.subnet, test.endpoints); err == nil {
			t.Fatalf("NewResolverPolicy(%q) accepted unsafe input", test.id)
		}
	}
}
