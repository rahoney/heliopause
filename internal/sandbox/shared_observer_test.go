package sandbox

import (
	"bytes"
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
	var diagnostic bytes.Buffer
	observer.diagnostic = &diagnostic
	for _, record := range []helperRecord{
		{ContainerID: "0123456789abcdef", Kind: "container-start"},
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "SOCKET", Family: "INET", ProcessRelation: "DIRECT_EXEC_SESSION", ProcessClass: "PYTHON"},
		{ContainerID: "0123456789abcdef", Kind: "stream-end"},
	} {
		body, _ := json.Marshal(record)
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
	if got, want := diagnostic.String(), "observer_attribution sequence=1 profile=default counts=NETWORK/SOCKET/INET/DIRECT_EXEC_SESSION/PYTHON:1\n"; got != want {
		t.Fatalf("diagnostic = %q, want %q", got, want)
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
	body, _ := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "CONNECT", Family: "INET6", ProcessRelation: "UNKNOWN", ProcessClass: "OTHER"})
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
	f.Add([]byte(`{"container_id":"0123456789abcdef","kind":"network-attempt","event_source":"SOCKET","family":"INET","process_relation":"UNKNOWN","process_class":"OTHER"}`))
	f.Add([]byte(`{"container_id":"invalid","kind":"network-attempt"}`))
	f.Add([]byte(`not-json`))
	f.Fuzz(func(t *testing.T, body []byte) {
		record, err := decodeHelperRecord(body)
		if err == nil && !containerIDPattern.MatchString(record.ContainerID) {
			t.Fatalf("accepted invalid container ID %q", record.ContainerID)
		}
	})
}

func TestDecodeHelperRecordAcceptsOnlyBoundedAttribution(t *testing.T) {
	valid := []helperRecord{
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "CONNECT", Family: "PACKET", ProcessRelation: "TRACKED_EXPECTED_GROUP", ProcessClass: "NPM"},
		{ContainerID: "0123456789abcdef", Kind: "process-exec-unexpected", EventSource: "SENTRY_EXEC", ProcessClass: "SHELL", ClassificationReason: "UNMODELED_PARENT", ParentRelation: "UNTRACKED_PARENT"},
	}
	for _, record := range valid {
		payload, _ := json.Marshal(record)
		if _, err := decodeHelperRecord(payload); err != nil {
			t.Fatalf("valid attribution rejected: %#v: %v", record, err)
		}
	}
	invalid := []helperRecord{
		{ContainerID: "0123456789abcdef", Kind: "network-attempt"},
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "SENDTO", Family: "INET", ProcessRelation: "UNKNOWN", ProcessClass: "OTHER"},
		{ContainerID: "0123456789abcdef", Kind: "process-exec-unexpected", EventSource: "SYSCALL_EXECVE", ProcessClass: "/usr/bin/sh", ClassificationReason: "OTHER", ParentRelation: "ROOT"},
		{ContainerID: "0123456789abcdef", Kind: "filesystem-open", EventSource: "SOCKET"},
	}
	for _, record := range invalid {
		payload, _ := json.Marshal(record)
		if _, err := decodeHelperRecord(payload); err == nil {
			t.Fatalf("invalid attribution accepted: %#v", record)
		}
	}
}

func TestSharedObserverAttributionAggregationIsBoundedAndDeterministic(t *testing.T) {
	observer := &SharedObserver{diagnostic: io.Discard}
	counts := make(map[string]uint64)
	for iteration := 0; iteration < maximumPyTorchCPUTraceEvents; iteration++ {
		record := helperRecord{Kind: "network-attempt", EventSource: "SOCKET", Family: "INET", ProcessRelation: "TRACKED_EXPECTED_GROUP", ProcessClass: "PYTHON"}
		counts[attributionKey(record)]++
	}
	if len(counts) != 1 || counts["NETWORK/SOCKET/INET/TRACKED_EXPECTED_GROUP/PYTHON"] != maximumPyTorchCPUTraceEvents {
		t.Fatalf("bounded counts = %#v", counts)
	}
	var output bytes.Buffer
	observer.diagnostic = &output
	observer.writeAttributionDiagnostic(sharedAttributionDiagnostic{sequence: 2, profile: "pypi-wheel-pytorch-cpu", counts: map[string]uint64{
		"PROCESS/SYSCALL_EXECVE/OTHER/UNKNOWN_CLASS/UNTRACKED_PARENT": 2,
		"NETWORK/CONNECT/INET6/DIRECT_EXEC_SESSION/PYTHON":            1,
	}})
	const want = "observer_attribution sequence=2 profile=pypi-wheel-pytorch-cpu counts=NETWORK/CONNECT/INET6/DIRECT_EXEC_SESSION/PYTHON:1,PROCESS/SYSCALL_EXECVE/OTHER/UNKNOWN_CLASS/UNTRACKED_PARENT:2\n"
	if output.String() != want {
		t.Fatalf("diagnostic = %q, want %q", output.String(), want)
	}
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
