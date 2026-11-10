package advisory

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

var errTransport = errors.New("read-only public advisory transport is unavailable")

// NewOSVHTTPClient receives the composition root's shared public-destination
// predicate. Inspection depends on this pure rule, not a privileged service or
// concrete Linux backend. Artifact data cannot select hosts or credentials.
func NewOSVHTTPClient(publicIPv4 func(netip.Addr) bool) (*http.Client, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	return newOSVHTTPClient(publicIPv4, net.DefaultResolver.LookupNetIP, dialer.DialContext)
}

func newOSVHTTPClient(publicIPv4 func(netip.Addr) bool, lookup func(context.Context, string, string) ([]netip.Addr, error), dial func(context.Context, string, string) (net.Conn, error)) (*http.Client, error) {
	if publicIPv4 == nil || lookup == nil || dial == nil {
		return nil, errTransport
	}
	base := &http.Transport{
		Proxy: nil, ForceAttemptHTTP2: true, DisableCompression: true,
		MaxIdleConns: 4, MaxIdleConnsPerHost: 2, MaxConnsPerHost: 2,
		IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 64 << 10,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: "api.osv.dev"},
	}
	base.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if (network != "tcp" && network != "tcp4") || address != "api.osv.dev:443" {
			return nil, errTransport
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		addresses, err := lookup(ctx, "ip4", "api.osv.dev")
		if err != nil {
			return nil, errors.Join(errTransport, err)
		}
		if len(addresses) == 0 || len(addresses) > 32 {
			return nil, errTransport
		}
		// Validate every answer before dialing. A mixed public/private answer
		// is not permission to try the public subset or resolve a second time.
		for _, address := range addresses {
			address = address.Unmap()
			if !address.Is4() || !publicIPv4(address) {
				return nil, errTransport
			}
		}
		var last error
		for _, address := range addresses {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			address = address.Unmap()
			connection, err := dial(ctx, "tcp4", net.JoinHostPort(address.String(), "443"))
			if err == nil && connection != nil {
				return connection, nil
			}
			if connection != nil {
				last = errors.Join(last, connection.Close())
			}
			last = errors.Join(last, err)
		}
		return nil, errors.Join(errTransport, last)
	}
	return &http.Client{Transport: &advisoryTransport{base}, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}

type advisoryTransport struct{ base *http.Transport }

func (t *advisoryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t == nil || t.base == nil || request == nil || request.URL == nil {
		return nil, errTransport
	}
	u := request.URL
	if u.Scheme != "https" || u.Host != "api.osv.dev" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" || request.Host != "" && request.Host != "api.osv.dev" {
		return nil, errTransport
	}
	for name, values := range request.Header {
		if len(values) > 0 && (strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Proxy-Authorization") || strings.EqualFold(name, "Cookie")) {
			return nil, errTransport
		}
	}
	switch request.Method {
	case http.MethodPost:
		if u.Path != "/v1/querybatch" || request.Header.Get("Content-Type") != "application/json" || request.ContentLength < 0 || request.ContentLength > maxResponseBytes {
			return nil, errTransport
		}
	case http.MethodGet:
		id, ok := strings.CutPrefix(u.Path, "/v1/vulns/")
		if !ok || !advisoryID.MatchString(id) || id == "." || id == ".." || request.ContentLength != 0 || request.Body != nil {
			return nil, errTransport
		}
	default:
		return nil, errTransport
	}
	return t.base.RoundTrip(request)
}

func (t *advisoryTransport) CloseIdleConnections() {
	if t != nil && t.base != nil {
		t.base.CloseIdleConnections()
	}
}
