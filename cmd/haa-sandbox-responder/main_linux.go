//go:build linux

// haa-sandbox-responder is a fixed guest-only stimulus, never an egress proxy.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	maximumRequests = 32
	maximumBytes    = 16 << 10
	maximumBody     = 512
	maximumLifetime = 180 * time.Second
	probeBody       = "{\"status\":\"ok\"}\n"
)

type responder struct {
	mu                      sync.Mutex
	requests, input, output int
	probe                   bool
	fail                    func()
}

func main() {
	os.Clearenv()
	runtime.GOMAXPROCS(1)
	if len(os.Args) == 1 || len(os.Args) == 2 && os.Args[1] == "--help" {
		_, _ = fmt.Fprintln(os.Stdout, "haa-sandbox-responder serve: bounded guest-only DNS/HTTP stimulus")
		return
	}
	if len(os.Args) != 2 || os.Args[1] != "serve" || os.Geteuid() != 1001 || os.Getegid() != 1001 {
		_, _ = fmt.Fprintln(os.Stderr, "RESPONDER_INVOCATION_INVALID")
		os.Exit(1)
	}
	dns, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 53})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "RESPONDER_DNS_BIND_FAILED")
		os.Exit(1)
	}
	web, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 80})
	if err != nil {
		_ = dns.Close()
		_, _ = fmt.Fprintln(os.Stderr, "RESPONDER_HTTP_BIND_FAILED")
		os.Exit(1)
	}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signals, maximumLifetime)
	defer cancel()
	if serve(ctx, os.Stdin, os.Stdout, dns, web) != nil {
		_, _ = fmt.Fprintln(os.Stderr, "RESPONDER_SESSION_FAILED")
		os.Exit(1)
	}
	// Keep the init alive after the bounded protocol has stopped. The existing
	// resource owner terminates and attests this process before final inventory;
	// a READY/STOPPED line alone never supplies completion authority.
	<-ctx.Done()
}

// One budget spans both protocols. No request body, name or credential is logged.
func (r *responder) charge(input, output int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if input < 0 || output < 0 || r.requests >= maximumRequests ||
		input > maximumBytes-r.input || output > maximumBytes-r.output {
		return false
	}
	r.requests++
	r.input += input
	r.output += output
	return true
}

func allowedName(name string) bool {
	switch strings.ToLower(name) {
	case "artifact.helox.invalid", "probe.helox.invalid", "sink.helox.invalid":
		return true
	}
	return false
}

// Accept one uncompressed IN question. Unsupported/malformed messages produce
// no response; unknown names return NXDOMAIN. Never resolve or forward a name.
func dnsResponse(query []byte) []byte {
	if len(query) < 17 || len(query) > 512 || query[2]&0xfe != 0 || query[3] != 0 ||
		binary.BigEndian.Uint16(query[4:6]) != 1 || !bytes.Equal(query[6:12], make([]byte, 6)) {
		return nil
	}
	labels := make([]string, 0, 4)
	end := 12
	for {
		if end >= len(query) || end-12 > 253 {
			return nil
		}
		size := int(query[end])
		end++
		if size == 0 {
			break
		}
		if size > 63 || size > len(query)-end {
			return nil
		}
		labels = append(labels, string(query[end:end+size]))
		end += size
	}
	if end+4 != len(query) || binary.BigEndian.Uint16(query[end+2:]) != 1 {
		return nil
	}
	known := allowedName(strings.Join(labels, "."))
	answer := known && binary.BigEndian.Uint16(query[end:end+2]) == 1
	response := append([]byte(nil), query...)
	response[2], response[3] = 0x84|(query[2]&1), 0 // authoritative, no recursion
	if !known {
		response[3] = 3
	}
	if answer {
		response[7] = 1
		response = append(response, 0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 1, 0, 4, 127, 0, 0, 1)
	}
	return response
}

// The listener processes one connection at a time. The standard HTTP parser
// reads at most 8193 bytes, with a deadline; bodies are limited separately.
func (r *responder) httpResponse(input *io.LimitedReader) []byte {
	status, body := http.StatusBadRequest, "unavailable\n"
	request, err := http.ReadRequest(bufio.NewReaderSize(input, 512))
	if err == nil {
		payload, readErr := io.ReadAll(io.LimitReader(request.Body, maximumBody+1))
		host := strings.TrimSuffix(request.Host, ":80")
		switch {
		case readErr != nil || len(payload) > maximumBody || len(request.TransferEncoding) != 0:
			status = http.StatusRequestEntityTooLarge
		case !allowedName(host) || request.URL.IsAbs() || request.URL.RawQuery != "":
			status = http.StatusMisdirectedRequest
		default:
			r.mu.Lock()
			switch {
			case request.Method == http.MethodGet && request.URL.Path == "/probe" && len(payload) == 0:
				r.probe = true
				status, body = http.StatusOK, probeBody
			case request.Method == http.MethodPost && request.URL.Path == "/sink" && r.probe:
				status, body = http.StatusNoContent, ""
			default:
				status = http.StatusNotFound
			}
			r.mu.Unlock()
		}
	}
	var output bytes.Buffer
	response := http.Response{StatusCode: status, ProtoMajor: 1, ProtoMinor: 0, Close: true,
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body))}
	if response.Write(&output) != nil || input.N == 0 || !r.charge(8193-int(input.N), output.Len()) {
		r.fail()
		return nil
	}
	return output.Bytes()
}

func serve(parent context.Context, input io.ReadCloser, output io.Writer, dns *net.UDPConn, web *net.TCPListener) error {
	ctx, cancel := context.WithTimeout(parent, maximumLifetime)
	defer cancel()
	shutdown := make(chan struct{})
	failed := make(chan struct{}, 1)
	r := &responder{fail: func() {
		select {
		case failed <- struct{}{}:
		default:
		}
	}}
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for {
			connection, err := web.AcceptTCP()
			if err != nil {
				select {
				case <-shutdown:
				default:
					r.fail()
				}
				return
			}
			if connection.SetDeadline(time.Now().Add(time.Second)) != nil {
				_ = connection.Close()
				r.fail()
				return
			}
			response := r.httpResponse(&io.LimitedReader{R: connection, N: 8193})
			if len(response) != 0 {
				if _, err := io.Copy(connection, bytes.NewReader(response)); err != nil {
					r.fail()
				}
			}
			_ = connection.Close()
		}
	}()
	go func() {
		defer workers.Done()
		var buffer [513]byte
		for {
			size, peer, err := dns.ReadFromUDP(buffer[:])
			if err != nil {
				select {
				case <-shutdown:
				default:
					r.fail()
				}
				return
			}
			response := dnsResponse(buffer[:size])
			if !peer.IP.IsLoopback() || !r.charge(size, len(response)) {
				r.fail()
				return
			}
			if len(response) != 0 {
				if _, err := dns.WriteToUDP(response, peer); err != nil {
					r.fail()
					return
				}
			}
		}
	}()
	control := make(chan bool, 1)
	go func() {
		value, err := io.ReadAll(io.LimitReader(input, 7))
		control <- err == nil && string(value) == "STOP\n"
	}()
	var result error
	controlRead := false
	if _, err := io.WriteString(output, "READY\n"); err != nil {
		result = errors.New("responder readiness failed")
	} else {
		select {
		case stopped := <-control:
			controlRead = true
			if !stopped {
				result = errors.New("responder control failed")
			}
		case <-failed:
			result = errors.New("responder protocol or budget failed")
		case <-ctx.Done():
			result = errors.New("responder deadline or cancellation")
		}
	}
	close(shutdown)
	_ = input.Close()
	if !controlRead {
		select {
		case <-control:
		case <-time.After(time.Second):
			result = errors.Join(result, errors.New("responder control did not drain"))
		}
	}
	_ = dns.Close()
	result = errors.Join(result, web.Close())
	workers.Wait()
	if ctx.Err() != nil {
		result = errors.Join(result, errors.New("responder deadline or cancellation"))
	}
	select {
	case <-failed:
		result = errors.Join(result, errors.New("responder worker failed"))
	default:
	}
	if result == nil {
		_, result = io.WriteString(output, "STOPPED\n")
	}
	return result
}
