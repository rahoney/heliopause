package advisory

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

func TestPublicAdvisoryDialPinsValidatedAddressesWithoutSecondResolution(t *testing.T) {
	public := netip.MustParseAddr("8.8.8.8")
	lookups, dials := 0, 0
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	client, err := newOSVHTTPClient(func(ip netip.Addr) bool { return ip == public }, func(ctx context.Context, network, host string) ([]netip.Addr, error) {
		lookups++
		if network != "ip4" || host != "api.osv.dev" {
			t.Fatal("unapproved advisory DNS lookup")
		}
		return []netip.Addr{public}, nil
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		dials++
		if network != "tcp4" || address != "8.8.8.8:443" {
			t.Fatal("DNS was resolved again or endpoint changed")
		}
		return left, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	transport := client.Transport.(*advisoryTransport).base
	connection, err := transport.DialContext(context.Background(), "tcp", "api.osv.dev:443")
	if err != nil || connection != left || lookups != 1 || dials != 1 {
		t.Fatal("public pinned connection failed", err)
	}
	if transport.Proxy != nil || transport.TLSClientConfig.InsecureSkipVerify || transport.TLSClientConfig.ServerName != "api.osv.dev" || transport.TLSClientConfig.MinVersion < tls.VersionTLS12 || !transport.DisableCompression || client.Jar != nil {
		t.Fatal("advisory TLS/proxy/credential boundary changed")
	}
}

func TestPublicAdvisoryDialRejectsUnsafeOrUnknownAnswersBeforeConnect(t *testing.T) {
	for _, addresses := range [][]netip.Addr{
		nil, {netip.MustParseAddr("127.0.0.1")}, {netip.MustParseAddr("169.254.169.254")},
		{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")},
		{netip.MustParseAddr("2001:4860:4860::8888")}, make([]netip.Addr, 33),
	} {
		dials := 0
		client, err := newOSVHTTPClient(func(ip netip.Addr) bool { return ip == netip.MustParseAddr("8.8.8.8") }, func(context.Context, string, string) ([]netip.Addr, error) { return addresses, nil }, func(context.Context, string, string) (net.Conn, error) {
			dials++
			return nil, errors.New("must not dial")
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Transport.(*advisoryTransport).base.DialContext(context.Background(), "tcp", "api.osv.dev:443"); err == nil || dials != 0 {
			t.Fatal("unsafe/unknown DNS answers reached connect", addresses)
		}
	}
	cause := errors.New("DNS observation unavailable")
	client, _ := newOSVHTTPClient(func(netip.Addr) bool { return true }, func(context.Context, string, string) ([]netip.Addr, error) { return nil, cause }, func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("failed lookup reached dial")
		return nil, nil
	})
	if _, err := client.Transport.(*advisoryTransport).base.DialContext(context.Background(), "tcp", "api.osv.dev:443"); !errors.Is(err, cause) {
		t.Fatal("primary lookup error lost", err)
	}
}

func TestPublicAdvisoryTransportRejectsOtherOperationsBeforeDNS(t *testing.T) {
	lookups := 0
	client, err := newOSVHTTPClient(func(netip.Addr) bool { return true }, func(context.Context, string, string) ([]netip.Addr, error) {
		lookups++
		return nil, errors.New("must not lookup")
	}, func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("unapproved operation reached dial")
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, target, header string }{
		{"GET", "http://api.osv.dev/v1/vulns/OSV-1", ""},
		{"GET", "https://api.osv.dev:443/v1/vulns/OSV-1", ""},
		{"GET", "https://user:secret@api.osv.dev/v1/vulns/OSV-1", ""},
		{"GET", "https://other.invalid/v1/vulns/OSV-1", ""},
		{"GET", "https://api.osv.dev/v1/vulns/../private", ""},
		{"GET", "https://api.osv.dev/v1/vulns/%2e%2e", ""},
		{"GET", "https://api.osv.dev/v1/vulns/OSV-1?extra=value", ""},
		{"GET", "https://api.osv.dev/v1/vulns/OSV-1?", ""},
		{"DELETE", "https://api.osv.dev/v1/vulns/OSV-1", ""},
		{"POST", "https://api.osv.dev/v1experimental/importfindings", ""},
		{"GET", "https://api.osv.dev/v1/vulns/OSV-1", "authorization"},
		{"GET", "https://api.osv.dev/v1/vulns/OSV-1", "cookie"},
		{"GET", "https://api.osv.dev/v1/vulns/OSV-1", "Proxy-Authorization"},
	} {
		request, err := http.NewRequest(tc.method, tc.target, nil)
		if err != nil {
			t.Fatal(err)
		}
		if tc.header != "" {
			request.Header[tc.header] = []string{"never-forward"}
		}
		if _, err := client.Transport.RoundTrip(request); err == nil || lookups != 0 {
			t.Fatal("operation escaped read-only origin", tc)
		}
	}
	for _, body := range []string{"not-json", ""} {
		request, _ := http.NewRequest(http.MethodPost, osvEndpoint, strings.NewReader(body))
		request.Header.Set("Content-Type", "text/plain")
		if _, err := client.Transport.RoundTrip(request); err == nil || lookups != 0 {
			t.Fatal("untyped query body reached endpoint")
		}
	}
}

func TestPublicAdvisoryTransportPermitsOnlyItsTypedReadRequests(t *testing.T) {
	lookups := 0
	cause := errors.New("controlled public DNS failure")
	client, _ := newOSVHTTPClient(func(netip.Addr) bool { return true }, func(context.Context, string, string) ([]netip.Addr, error) { lookups++; return nil, cause }, func(context.Context, string, string) (net.Conn, error) { return nil, nil })
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		target := osvEndpoint
		var body *strings.Reader
		if method == http.MethodGet {
			target = "https://api.osv.dev/v1/vulns/OSV-1"
		} else {
			body = strings.NewReader(`{"queries":[]}`)
		}
		var request *http.Request
		if body == nil {
			request, _ = http.NewRequest(method, target, nil)
		} else {
			request, _ = http.NewRequest(method, target, body)
			request.Header.Set("Content-Type", "application/json")
		}
		before := lookups
		if _, err := client.Do(request); !errors.Is(err, cause) || lookups <= before {
			t.Fatal("typed read request was rejected before its controlled endpoint", method, err)
		}
	}
}

func TestPublicAdvisoryDialPreservesCancellationAndRestrictsDestination(t *testing.T) {
	lookups := 0
	client, _ := newOSVHTTPClient(func(netip.Addr) bool { return true }, func(context.Context, string, string) ([]netip.Addr, error) { lookups++; return nil, nil }, func(context.Context, string, string) (net.Conn, error) { return nil, nil })
	dial := client.Transport.(*advisoryTransport).base.DialContext
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := dial(ctx, "tcp", "api.osv.dev:443"); !errors.Is(err, context.Canceled) || lookups != 0 {
		t.Fatal("cancellation was lost")
	}
	for _, address := range []string{"127.0.0.1:443", "api.osv.dev:80", "other.invalid:443"} {
		if _, err := dial(context.Background(), "tcp", address); err == nil || lookups != 0 {
			t.Fatal("unapproved destination reached resolver")
		}
	}
	if client, err := NewOSVHTTPClient(nil); err == nil || client != nil {
		t.Fatal("missing endpoint policy accepted")
	}
}
