package networkpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type serverTestPeer struct{}

func (serverTestPeer) AuthorizePeer(*net.UnixConn) (uint32, error) { return 1000, nil }

type serverTestService struct{ calls int }

func (s *serverTestService) CreateForPeer(context.Context, uint32, ResolverPolicy) error {
	s.calls++
	return nil
}
func (*serverTestService) VerifyForPeer(context.Context, uint32, ResolverPolicy) error {
	return errors.New("unexpected verify")
}
func (*serverTestService) RemoveForPeer(context.Context, uint32, ResolverPolicy) error {
	return errors.New("unexpected remove")
}

func TestServerBoundsRequestBeforeNewline(t *testing.T) {
	session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	policy, err := NewResolverPolicy(session, "0123456789abcdef", netip.MustParsePrefix("172.30.0.0/24"), []netip.Addr{netip.MustParseAddr("1.1.1.1")})
	if err != nil {
		t.Fatal(err)
	}
	request, err := encodeRequest(createOperation, policy)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		body      []byte
		wantCalls int
	}{
		{"valid", append(valid, '\n'), 1},
		{"oversized_without_newline", bytes.Repeat([]byte("x"), maxMessageBytes+1), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Keep the socket path below macOS's sun_path limit.
			directory, err := os.MkdirTemp("", "np-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.RemoveAll(directory); err != nil {
					t.Error(err)
				}
			})
			endpoint := &net.UnixAddr{Name: filepath.Join(directory, "server.sock"), Net: "unix"}
			listener, err := net.ListenUnix("unix", endpoint)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			client, err := net.DialUnix("unix", nil, endpoint)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			connection, err := listener.AcceptUnix()
			if err != nil {
				t.Fatal(err)
			}
			service := &serverTestService{}
			server := &Server{authorizer: serverTestPeer{}, service: service}
			done := make(chan struct{})
			go func() {
				defer close(done)
				server.handle(context.Background(), connection)
			}()
			defer func() { _ = client.Close(); <-done }()
			if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Write(tc.body); err != nil {
				t.Fatal(err)
			}
			response, err := io.ReadAll(client)
			if err != nil {
				t.Fatalf("server did not finish a bounded request: %v", err)
			}
			<-done
			if service.calls != tc.wantCalls || (len(response) != 0) != (tc.wantCalls != 0) {
				t.Fatalf("calls=%d response=%q", service.calls, response)
			}
		})
	}
}

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
		[]byte(`{"operation":"create","session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","network_id":"0123456789abcdef","subnet":"172.30.0.0/24","endpoints":["100.100.100.200"]}`),
		[]byte(`{"operation":"iptables -F","session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","network_id":"0123456789abcdef","subnet":"172.30.0.0/24","endpoints":["1.1.1.1"]}`),
		[]byte(`{"operation":"create","session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","network_id":"0123456789abcdef","subnet":"172.30.0.0/24","endpoints":["1.1.1.1"],"args":["-F"]}`),
		[]byte(`{"operation":"create","session":"sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa","network_id":"0123456789abcdef","subnet":"172.30.0.0/24","endpoints":["1.1.1.1"]} trailing`),
	} {
		if _, _, err := decodeRequest(invalid); err == nil {
			t.Fatalf("decodeRequest accepted %q", invalid)
		}
	}
}
