package networkpolicy

import (
	"encoding/json"
	"net/netip"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestTypedWireRequestRoundTripRejectsUnknownOrRawInput(t *testing.T) {
	session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	policy, err := NewResolverPolicy(session, "0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	request, err := encodeRequest(createOperation, policy)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(request)
	op, decoded, err := decodeRequest(body)
	if err != nil || op != createOperation || decoded.NetworkID() != policy.NetworkID() {
		t.Fatalf("decodeRequest() = %q, %#v, %v", op, decoded, err)
	}
	for _, invalid := range [][]byte{
		[]byte(`{"operation":"iptables -F","session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","network_id":"0123456789abcdef","subnet":"172.30.0.0/24","endpoints":["1.1.1.1"]}`),
		[]byte(`{"operation":"create","session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","network_id":"0123456789abcdef","subnet":"172.30.0.0/24","endpoints":["1.1.1.1"],"args":["-F"]}`),
		[]byte(`{"operation":"create","session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","network_id":"0123456789abcdef","subnet":"172.30.0.0/24","endpoints":["1.1.1.1"]} trailing`),
	} {
		if _, _, err := decodeRequest(invalid); err == nil {
			t.Fatalf("decodeRequest accepted %q", invalid)
		}
	}
}
