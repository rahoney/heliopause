package sandbox

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSharedObserverAttributesOneContainerAndClosesItsStream(t *testing.T) {
	endpoint := observerEndpoint(t)
	observer, err := NewSharedObserver(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := observer.Start(context.Background(), "0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	for _, kind := range []string{"container-start", "network-attempt", "stream-end"} {
		body, _ := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: kind})
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	record, err := reader.Next(context.Background())
	if err != nil || record.Kind != "network-attempt" {
		t.Fatalf("Next() = (%#v, %v)", record, err)
	}
	if _, err := reader.Next(context.Background()); err != io.EOF {
		t.Fatalf("stream end = %v, want EOF", err)
	}
}

func TestSharedObserverFailsClosedForUnknownContainer(t *testing.T) {
	endpoint := observerEndpoint(t)
	observer, err := NewSharedObserver(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	body, _ := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "network-attempt"})
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 100; attempt++ {
		observer.mu.Lock()
		fault := observer.fault
		observer.mu.Unlock()
		if fault != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("unknown container did not fail closed")
}

func TestSharedObserverFailsClosedForLatchedStreamFault(t *testing.T) {
	endpoint := observerEndpoint(t)
	observer, err := NewSharedObserver(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := observer.Start(context.Background(), "0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	body, _ := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: "EVENT_LIMIT"})
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := reader.Next(ctx); err == nil {
		t.Fatal("faulted stream completed without an incomplete error")
	} else {
		fault, ok := err.(traceFault)
		if !ok || fault.TraceFaultReason() != "EVENT_LIMIT" {
			t.Fatalf("fault = %v, want typed EVENT_LIMIT", err)
		}
	}
	if _, err := observer.Start(context.Background(), "fedcba9876543210"); err == nil {
		t.Fatal("faulted observer accepted a subsequent container mapping")
	}
}

func FuzzDecodeHelperRecord(f *testing.F) {
	f.Add([]byte(`{"container_id":"0123456789abcdef","kind":"network-attempt"}`))
	f.Add([]byte(`{"container_id":"invalid","kind":"network-attempt"}`))
	f.Add([]byte(`not-json`))
	f.Fuzz(func(t *testing.T, body []byte) {
		record, err := decodeHelperRecord(body)
		if err == nil && !containerIDPattern.MatchString(record.ContainerID) {
			t.Fatalf("accepted invalid container ID %q", record.ContainerID)
		}
	})
}

func TestDecodeHelperRecordRejectsOversizedPayload(t *testing.T) {
	if _, err := decodeHelperRecord(make([]byte, maximumHelperRecordBytes+1)); err == nil {
		t.Fatal("oversized observer record was accepted")
	}
	payload, err := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: strings.Repeat("A", maximumHelperRecordBytes)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHelperRecord(payload); err == nil {
		t.Fatal("oversized observer reason was accepted")
	}
}

func TestDecodeHelperRecordAcceptsBoundedSocketFaultReasons(t *testing.T) {
	for _, reason := range []string{
		"SOCKET_AF_UNSPEC", "SOCKET_AF_NETLINK", "SOCKET_AF_PACKET", "SOCKET_OTHER_FAMILY",
		"CONNECT_ADDRESS_TOO_SHORT", "CONNECT_AF_UNSPEC", "CONNECT_AF_UNIX_INVALID_LENGTH",
		"CONNECT_AF_INET_INVALID_LENGTH", "CONNECT_AF_INET6_INVALID_LENGTH", "CONNECT_AF_NETLINK_INVALID_LENGTH",
		"CONNECT_AF_PACKET_INVALID_LENGTH", "CONNECT_UNKNOWN_FAMILY",
	} {
		payload, err := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: reason})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeHelperRecord(payload); err != nil {
			t.Fatalf("reason %q rejected: %v", reason, err)
		}
	}
}

func TestDecodeHelperRecordRejectsUntrustedFaultReason(t *testing.T) {
	payload, err := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: "SOCKET_AF_123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHelperRecord(payload); err == nil {
		t.Fatal("untrusted observer reason was accepted")
	}
}

func TestObserverProfileAllowlistRejectsUntrustedBudgetSelection(t *testing.T) {
	if !validObserverProfile("pypi-wheel-pytorch-cpu") || !validObserverProfile("pypi-wheel-pytorch-cu126") {
		t.Fatal("canonical PyTorch profiles rejected")
	}
	for _, profile := range []string{"pytorch:cpu", "pypi-wheel-pytorch-other", "https://example.test"} {
		if validObserverProfile(profile) {
			t.Fatalf("untrusted profile accepted: %s", profile)
		}
	}
}

func observerEndpoint(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp("/tmp", "haa-observer-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	return filepath.Join(directory, "o.sock")
}
