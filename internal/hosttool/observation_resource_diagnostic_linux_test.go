//go:build linux

package hosttool

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestObservationReadTimingCannotEnterWire(t *testing.T) {
	lease := ObservationResourceLease{CgroupParent: "parent", CPU: 1, UsageUsec: 42, ReadTiming: ObservationResourceReadTiming{StartedUnixNS: 123, TotalNS: 500000000}}
	body, err := json.Marshal(observationResourceResponse{OK: true, Lease: lease})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "123") || strings.Contains(string(body), "Timing") || strings.Contains(string(body), "total") {
		t.Fatal("local transport timing leaked into wire", string(body))
	}
	var response observationResourceResponse
	decoder := json.NewDecoder(strings.NewReader(`{"ok":true,"lease":{"cgroup_parent":"parent","cpu":1,"usage_usec":42,"read_timing":{"TotalNS":1}}}`))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err == nil {
		t.Fatal("wire forged local timing accepted")
	}
}

func TestObservationServerReadTimingIsBoundedAndUnauthoritative(t *testing.T) {
	now := time.Unix(100, 0)
	d := observationServerReadTiming{started: now, dispatched: now.Add(time.Millisecond), authorized: now.Add(5 * time.Millisecond), decoded: now.Add(450 * time.Millisecond), applied: now.Add(490 * time.Millisecond), wrote: now.Add(495 * time.Millisecond), authenticatedRead: true}
	var out bytes.Buffer
	var count atomic.Uint32
	for i := 0; i < 30; i++ {
		d.report(&out, &count, now.Add(500*time.Millisecond))
	}
	if count.Load() != 16 || strings.Count(out.String(), "observation_resource_read_timing") != 16 || out.Len() > 8192 {
		t.Fatal("diagnostic bound failed", count.Load(), out.Len())
	}
	for _, part := range []string{"dispatch_ns=1000000", "peer_auth_ns=4000000", "request_ns=445000000", "engine_ns=40000000", "response_ns=5000000", "close_ns=5000000", "total_ns=500000000"} {
		if !strings.Contains(out.String(), part) {
			t.Fatal("missing timing", part)
		}
	}
	out.Reset()
	count.Store(0)
	d.authenticatedRead = false
	d.report(&out, &count, now.Add(time.Second))
	d.authenticatedRead = true
	d.report(&out, &count, now.Add(200*time.Millisecond))
	if out.Len() != 0 || count.Load() != 0 {
		t.Fatal("unverified or fast request produced timing")
	}
}
