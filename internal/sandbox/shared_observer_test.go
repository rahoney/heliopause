package sandbox

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
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
	body, _ := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault"})
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := reader.Next(ctx); err == nil {
		t.Fatal("faulted stream completed without an incomplete error")
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
