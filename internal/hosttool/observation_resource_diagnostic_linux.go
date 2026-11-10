//go:build linux

package hosttool

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// These timings are diagnostics, never helper authorization or accounting
// evidence. Logging runs after connection close and is capped per service.
type observationServerReadTiming struct {
	started, dispatched, authorized, decoded, applied, wrote time.Time
	authenticatedRead                                        bool
}

func (d observationServerReadTiming) report(out io.Writer, count *atomic.Uint32, closed time.Time) {
	if !d.authenticatedRead || out == nil || count == nil || d.started.IsZero() || closed.Sub(d.started) < 250*time.Millisecond {
		return
	}
	for {
		used := count.Load()
		if used >= 16 {
			return
		}
		if count.CompareAndSwap(used, used+1) {
			break
		}
	}
	_, _ = fmt.Fprintf(out, "observation_resource_read_timing started_unix_ns=%d dispatch_ns=%d peer_auth_ns=%d request_ns=%d engine_ns=%d response_ns=%d close_ns=%d total_ns=%d\n", d.started.UnixNano(), d.dispatched.Sub(d.started).Nanoseconds(), d.authorized.Sub(d.dispatched).Nanoseconds(), d.decoded.Sub(d.authorized).Nanoseconds(), d.applied.Sub(d.decoded).Nanoseconds(), d.wrote.Sub(d.applied).Nanoseconds(), closed.Sub(d.wrote).Nanoseconds(), closed.Sub(d.started).Nanoseconds())
}
