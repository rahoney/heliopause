//go:build linux

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func dnsQuestion(name string, kind uint16) []byte {
	query := []byte{0x12, 0x34, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	for _, label := range strings.Split(name, ".") {
		query = append(query, byte(len(label)))
		query = append(query, label...)
	}
	query = append(query, 0, byte(kind>>8), byte(kind), 0, 1)
	return query
}

func TestDNSFixedResponsesAndMalformedInput(t *testing.T) {
	for _, name := range []string{"artifact.helox.invalid", "PROBE.HELOX.INVALID", "sink.helox.invalid", "outside.invalid"} {
		for _, kind := range []uint16{1, 28, 16} {
			query := dnsQuestion(name, kind)
			response := dnsResponse(query)
			if len(response) < len(query) || !bytes.Equal(response[:2], query[:2]) || !bytes.Equal(response[12:len(query)], query[12:]) {
				t.Fatal("DNS question/transaction changed")
			}
			count := binary.BigEndian.Uint16(response[6:8])
			if allowedName(name) && kind == 1 {
				if count != 1 || !bytes.Equal(response[len(response)-4:], []byte{127, 0, 0, 1}) || len(response) != len(query)+16 {
					t.Fatal("nonloopback or unexpected answer")
				}
			} else if count != 0 || len(response) != len(query) {
				t.Fatal("unsupported DNS type/name acquired answer")
			}
			if (response[3] == 3) != (!allowedName(name)) || response[3]&0x80 != 0 {
				t.Fatal("upstream recursion or NXDOMAIN mismatch")
			}
		}
	}
	query := dnsQuestion("artifact.helox.invalid", 1)
	for variant := 0; variant < 9; variant++ {
		bad := append([]byte(nil), query...)
		switch variant {
		case 0:
			bad = bad[:16]
		case 1:
			bad[2] = 0x81
		case 2:
			bad[5] = 2
		case 3:
			bad[11] = 1
		case 4:
			bad[12] = 0xc0
		case 5:
			bad[len(bad)-1] = 2
		case 6:
			bad = append(bad, 0)
		case 7:
			bad = bytes.Repeat([]byte{0}, 513)
		case 8:
			bad[12] = 63
		}
		if dnsResponse(bad) != nil {
			t.Fatalf("malformed DNS accepted: %d", variant)
		}
	}
}

func TestHTTPFixedStateAndLimits(t *testing.T) {
	failed := false
	r := &responder{fail: func() { failed = true }}
	cases := []struct {
		request string
		status  int
	}{
		{"POST /sink HTTP/1.0\r\nHost: sink.helox.invalid\r\nContent-Length: 0\r\n\r\n", 404},
		{"GET /probe HTTP/1.0\r\nHost: real.example\r\n\r\n", 421},
		{"GET http://outside.invalid/probe HTTP/1.0\r\nHost: artifact.helox.invalid\r\n\r\n", 421},
		{"GET /probe?code=x HTTP/1.0\r\nHost: artifact.helox.invalid\r\n\r\n", 421},
		{"GET /probe HTTP/1.0\r\nHost: probe.helox.invalid:80\r\n\r\n", 200},
		{"POST /sink HTTP/1.0\r\nHost: sink.helox.invalid\r\nContent-Length: 5\r\n\r\ndummy", 204},
		{"POST /sink HTTP/1.0\r\nHost: sink.helox.invalid\r\nContent-Length: 513\r\n\r\n" + strings.Repeat("x", 513), 413},
		{"GET /probe HTTP/1.0\r\nHost: 169.254.169.254\r\n\r\n", 421},
		{"invalid\r\n\r\n", 400},
	}
	for _, test := range cases {
		reply := r.httpResponse(&io.LimitedReader{R: strings.NewReader(test.request), N: 8193})
		response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(reply)), nil)
		if err != nil || response.StatusCode != test.status {
			t.Fatalf("fixed response: status=%d err=%v", test.status, err)
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil || len(body) > len(probeBody) || response.Header.Get("Location") != "" || bytes.Contains(body, []byte("dummy")) {
			t.Fatal("request data or redirect exposed")
		}
	}
	if failed {
		t.Fatal("bounded requests failed")
	}
	_ = r.httpResponse(&io.LimitedReader{R: strings.NewReader("GET /probe HTTP/1.0\r\nX: " + strings.Repeat("x", 9000)), N: 8193})
	if !failed {
		t.Fatal("oversize parser input did not fail closed")
	}
}

func TestSharedProtocolBudget(t *testing.T) {
	r := &responder{}
	for i := 0; i < maximumRequests; i++ {
		if !r.charge(1, 1) {
			t.Fatal("early limit")
		}
	}
	if r.charge(0, 0) {
		t.Fatal("request budget replenished")
	}
	for _, direction := range []bool{false, true} {
		r = &responder{}
		if direction {
			if !r.charge(maximumBytes, 0) || r.charge(1, 0) {
				t.Fatal("input limit")
			}
		} else {
			if !r.charge(0, maximumBytes) || r.charge(0, 1) {
				t.Fatal("output limit")
			}
		}
	}
}

// Real loopback transport and controller shutdown, with no Host/real DNS or URL.
func TestResponderTransportAndControl(t *testing.T) {
	for _, validControl := range []bool{true, false} {
		t.Run(map[bool]string{true: "stop", false: "forged"}[validControl], func(t *testing.T) {
			dns, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			web, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				_ = dns.Close()
				t.Fatal(err)
			}
			input, control := io.Pipe()
			reader, output := io.Pipe()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- serve(ctx, input, output, dns, web); _ = output.Close() }()
			defer control.Close()
			defer reader.Close()
			protocol := bufio.NewReader(reader)
			ready, err := protocol.ReadString('\n')
			if err != nil || ready != "READY\n" {
				t.Fatal("not ready")
			}
			client, err := net.DialUDP("udp4", nil, dns.LocalAddr().(*net.UDPAddr))
			if err != nil {
				t.Fatal(err)
			}
			_ = client.SetDeadline(time.Now().Add(time.Second))
			query := dnsQuestion("artifact.helox.invalid", 1)
			if _, err = client.Write(query); err != nil {
				t.Fatal(err)
			}
			buffer := make([]byte, 512)
			size, err := client.Read(buffer)
			_ = client.Close()
			if err != nil || !bytes.Equal(buffer[:size], dnsResponse(query)) {
				t.Fatal("fixed DNS exchange failed")
			}
			connection, err := net.DialTCP("tcp4", nil, web.Addr().(*net.TCPAddr))
			if err != nil {
				t.Fatal(err)
			}
			_ = connection.SetDeadline(time.Now().Add(time.Second))
			_, err = io.WriteString(connection, "GET /probe HTTP/1.0\r\nHost: artifact.helox.invalid\r\n\r\n")
			if err != nil {
				t.Fatal(err)
			}
			response, err := http.ReadResponse(bufio.NewReader(connection), nil)
			if err != nil || response.StatusCode != 200 {
				t.Fatal("fixed HTTP exchange failed")
			}
			_ = response.Body.Close()
			_ = connection.Close()
			command := "STOP\n"
			if !validControl {
				command = "DONE\n"
			}
			if _, err = io.WriteString(control, command); err != nil {
				t.Fatal(err)
			}
			_ = control.Close()
			stopped, readErr := protocol.ReadString('\n')
			select {
			case err = <-done:
			case <-ctx.Done():
				t.Fatal("controller shutdown did not complete")
			}
			if validControl {
				if err != nil || readErr != nil || stopped != "STOPPED\n" {
					t.Fatalf("shutdown: %v %v", err, readErr)
				}
			} else if err == nil || stopped != "" {
				t.Fatal("invalid controller input reported completion")
			}
		})
	}
}

func FuzzDNSResponseBounded(f *testing.F) {
	f.Add(dnsQuestion("artifact.helox.invalid", 1))
	f.Add([]byte{0, 1, 2})
	f.Fuzz(func(t *testing.T, query []byte) {
		response := dnsResponse(query)
		if len(response) > 528 {
			t.Fatal("DNS response exceeds bound")
		}
	})
}
