package bootstrap_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/ports"
)

// Reuse display events for timing only. No payload, locator or download chunk
// is logged; saturation drops diagnostics without changing the operation.
func newPyTorchStageDiagnostic(emit func(string, time.Duration)) func(ports.Progress) {
	started := time.Now()
	var mu sync.Mutex
	last, count := "", 0
	return func(event ports.Progress) {
		switch event.Stage {
		case "qualification-prepare", "qualification-returned", "resolve", "download", "verify", "inspect", "policy", "manifest", "stage", "install", "operation-returned":
		default:
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if event.Stage == last || count >= 128 {
			return
		}
		last = event.Stage
		count++
		emit(event.Stage, time.Since(started))
	}
}

func TestPyTorchStageDiagnosticBoundaries(t *testing.T) {
	count := 0
	ctx := ports.WithProgress(context.Background(), newPyTorchStageDiagnostic(func(stage string, elapsed time.Duration) {
		count++
		if stage != "download" || elapsed < 0 {
			t.Errorf("unexpected diagnostic %q/%v", stage, elapsed)
		}
	}))
	ports.ReportProgress(ctx, ports.Progress{Stage: "untrusted-locator"})
	for i := range 1024 {
		ports.ReportProgress(ctx, ports.Progress{Stage: "download", Current: int64(i), Unit: "untrusted-unit"})
	}
	if count != 1 {
		t.Fatalf("repeated download diagnostics = %d, want 1", count)
	}
	var events []string
	ctx = ports.WithProgress(context.Background(), newPyTorchStageDiagnostic(func(stage string, _ time.Duration) {
		events = append(events, stage)
	}))
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 256 {
				ports.ReportProgress(ctx, ports.Progress{Stage: "resolve"})
				ports.ReportProgress(ctx, ports.Progress{Stage: "inspect"})
			}
		})
	}
	workers.Wait()
	if len(events) != 128 {
		t.Fatalf("bounded diagnostics = %d, want 128", len(events))
	}
}

type pytorchPressureRecord struct {
	AtUnixNS    int64             `json:"at_unix_ns"`
	Values      map[string]uint64 `json:"values"`
	Unavailable []string          `json:"unavailable,omitempty"`
}

// Test diagnostics only: no extra process, admission decision or qualification
// input. The caller joins the sampler before the integration test returns.
func startPyTorchPressureDiagnostic(t *testing.T, ctx context.Context) func() {
	t.Helper()
	deadline, ok := ctx.Deadline()
	if runtime.GOOS != "linux" || !ok {
		return func() {}
	}
	limit := int(time.Until(deadline)/time.Second) + 2
	if limit < 1 || limit > 4096 {
		t.Log("pytorch_os_pressure: diagnostic deadline unavailable")
		return func() {}
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for i := 0; i < limit; i++ {
			record := pytorchPressureSnapshot(readPyTorchPressureFile)
			body, err := json.Marshal(record)
			if err == nil {
				t.Logf("pytorch_os_pressure %s", body)
			}
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return func() { close(stop); <-done }
}

func readPyTorchPressureFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	body, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(body) > 65536 {
		return nil, errors.New("bounded proc diagnostic unavailable")
	}
	return body, nil
}

func pytorchPressureSnapshot(read func(string) ([]byte, error)) pytorchPressureRecord {
	r := pytorchPressureRecord{AtUnixNS: time.Now().UnixNano(), Values: map[string]uint64{}}
	for _, source := range []struct {
		id, path, unit string
		names          map[string]string
	}{
		{"memory", "/proc/meminfo", "kB", map[string]string{"MemTotal": "mem_total_kib", "MemAvailable": "mem_available_kib", "SwapTotal": "swap_total_kib", "SwapFree": "swap_free_kib", "Shmem": "shmem_kib", "Dirty": "dirty_kib"}},
		{"vm", "/proc/vmstat", "", map[string]string{"pgscan_direct": "pgscan_direct", "pgsteal_direct": "pgsteal_direct", "pgmajfault": "pgmajfault", "pswpin": "pswpin", "pswpout": "pswpout", "oom_kill": "oom_kill"}},
		{"controller", "/proc/self/status", "kB", map[string]string{"VmRSS": "controller_rss_kib", "VmHWM": "controller_peak_rss_kib"}},
	} {
		body, err := read(source.path)
		values, parseErr := parsePyTorchPressureCounters(body, source.names, source.unit)
		if err != nil || parseErr != nil {
			r.Unavailable = append(r.Unavailable, source.id)
			continue
		}
		for k, v := range values {
			r.Values[k] = v
		}
	}
	for _, name := range []string{"cpu", "memory", "io"} {
		body, err := read("/proc/pressure/" + name)
		values, parseErr := parsePyTorchPSI(body)
		if err != nil || parseErr != nil {
			r.Unavailable = append(r.Unavailable, "psi_"+name)
			continue
		}
		for k, v := range values {
			r.Values["psi_"+name+"_"+k+"_total_us"] = v
		}
	}
	return r
}

func parsePyTorchPressureCounters(body []byte, names map[string]string, unit string) (map[string]uint64, error) {
	values := map[string]uint64{}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		name, selected := names[strings.TrimSuffix(fields[0], ":")]
		if !selected {
			continue
		}
		want := 2
		if unit != "" {
			want = 3
		}
		if len(fields) != want || unit != "" && fields[2] != unit {
			return nil, errors.New("proc diagnostic format")
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		if _, duplicate := values[name]; duplicate {
			return nil, errors.New("duplicate proc diagnostic")
		}
		values[name] = value
	}
	if len(values) != len(names) {
		return nil, errors.New("missing proc diagnostic")
	}
	return values, nil
}

func parsePyTorchPSI(body []byte) (map[string]uint64, error) {
	values := map[string]uint64{}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] != "some" && fields[0] != "full" {
			return nil, errors.New("unknown PSI diagnostic")
		}
		for _, field := range fields[1:] {
			if strings.HasPrefix(field, "total=") {
				if _, duplicate := values[fields[0]]; duplicate {
					return nil, errors.New("duplicate PSI diagnostic")
				}
				value, err := strconv.ParseUint(strings.TrimPrefix(field, "total="), 10, 64)
				if err != nil {
					return nil, err
				}
				values[fields[0]] = value
			}
		}
	}
	if len(values) != 2 {
		return nil, errors.New("missing PSI diagnostic")
	}
	return values, nil
}

func TestPyTorchPressureDiagnosticsKeepUnknownSeparate(t *testing.T) {
	if _, err := parsePyTorchPressureCounters([]byte("MemAvailable: 123 kB\n"), map[string]string{"MemAvailable": "available"}, "kB"); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"", "MemAvailable: -1 kB\n", "MemAvailable: 123 bytes\n", "MemAvailable: 123 kB\nMemAvailable: 124 kB\n"} {
		if _, err := parsePyTorchPressureCounters([]byte(body), map[string]string{"MemAvailable": "available"}, "kB"); err == nil {
			t.Fatal("invalid proc diagnostic accepted", body)
		}
	}
	values, err := parsePyTorchPSI([]byte("some avg10=0.00 total=12\nfull avg10=0.00 total=3\n"))
	if err != nil || values["some"] != 12 || values["full"] != 3 {
		t.Fatal(values, err)
	}
	for _, body := range []string{"", "some total=0\n", "some total=-1\nfull total=0\n", "some total=1 total=2\nfull total=0\n"} {
		if _, err := parsePyTorchPSI([]byte(body)); err == nil {
			t.Fatal("invalid PSI accepted", body)
		}
	}
	r := pytorchPressureSnapshot(func(string) ([]byte, error) { return nil, errors.New("query failed") })
	if len(r.Values) != 0 || len(r.Unavailable) != 6 {
		t.Fatal("query failure became zero pressure", r)
	}
}

func TestPyTorchPressureDiagnosticJoinsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	cancel()
	stop := startPyTorchPressureDiagnostic(t, ctx)
	stop()
}
