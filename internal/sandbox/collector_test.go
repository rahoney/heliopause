package sandbox

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCollectTraceNormalizesKindsWithoutPayload(t *testing.T) {
	observations, limitation := collectTrace(context.Background(), &traceReader{records: []TraceRecord{{Kind: "process-exec", Bytes: 12}, {Kind: "network-attempt", Bytes: 16}}})
	if limitation != "" || len(observations) != 2 {
		t.Fatalf("collectTrace() = (%d observations, %q)", len(observations), limitation)
	}
	if observations[0].Category() != domain.ObservationProcess || observations[0].Subject() != "process-exec" {
		t.Fatalf("first observation = %#v", observations[0])
	}
}

func TestCollectTraceFailsClosedOnUntrustedOrOversizedInput(t *testing.T) {
	tests := []struct {
		name   string
		reader TraceReader
		want   string
	}{
		{"reader failure", &traceReader{err: errors.New("socket failed")}, "M3_DYNAMIC_OBSERVER_FAILED"},
		{"unknown kind", &traceReader{records: []TraceRecord{{Kind: "raw-path:/secret", Bytes: 1}}}, "M3_DYNAMIC_OBSERVER_FAILED"},
		{"byte limit", &traceReader{records: []TraceRecord{{Kind: "process-exec", Bytes: maximumTraceBytes + 1}}}, "M3_DYNAMIC_OBSERVATION_LIMIT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, limitation := collectTrace(context.Background(), test.reader)
			if limitation != test.want {
				t.Fatalf("limitation = %q, want %q", limitation, test.want)
			}
		})
	}
}

func TestCollectTraceDiagnosticIsBoundedAndClassifiesFailure(t *testing.T) {
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{records: []TraceRecord{{Kind: "network-attempt", Bytes: 7}}, errAfter: 1, err: errors.New("transport")})
	if observations != nil || limitation != "M3_DYNAMIC_OBSERVER_FAILED" {
		t.Fatalf("collectTraceDiagnostic() = (%#v, %q)", observations, limitation)
	}
	if got, want := diagnostic.String(), "reason=READER_ERROR events=1 bytes=7 session_complete=false last_kind=network-attempt kinds=network-attempt:1"; got != want {
		t.Fatalf("diagnostic = %q, want %q", got, want)
	}
	if contains := diagnostic.String(); contains == "" || diagnostic.LastKind == "raw-path:/secret" {
		t.Fatalf("diagnostic retained unsafe data: %q", contains)
	}
}

func TestCollectTraceDiagnosticPerKindCountsAndPreservation(t *testing.T) {
	reader := &traceReader{
		budget: &traceBudget{events: 10, bytes: 1024},
		records: []TraceRecord{
			{Kind: "process-exec", Bytes: 10},
			{Kind: "process-exec", Bytes: 10},
			{Kind: "filesystem-open", Bytes: 20},
			{Kind: "filesystem-open", Bytes: 20},
			{Kind: "filesystem-open", Bytes: 20},
			{Kind: "network-attempt", Bytes: 15},
			{Kind: "process-clone", Bytes: 12},
			{Kind: "filesystem-workspace-access", Bytes: 18},
		},
	}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), reader)
	if limitation != "" || len(observations) != 8 {
		t.Fatalf("collectTraceDiagnostic() = (%d observations, %q)", len(observations), limitation)
	}
	if !diagnostic.SessionComplete || diagnostic.Reason != "" {
		t.Fatalf("diagnostic session = complete:%t reason:%q", diagnostic.SessionComplete, diagnostic.Reason)
	}
	if diagnostic.Events != 8 || diagnostic.Bytes != 125 {
		t.Fatalf("diagnostic events=%d bytes=%d, want 8 and 125", diagnostic.Events, diagnostic.Bytes)
	}
	if diagnostic.LastKind != "filesystem-workspace-access" {
		t.Fatalf("last kind = %q, want %q", diagnostic.LastKind, "filesystem-workspace-access")
	}
	expectedCounts := map[string]uint64{
		"process-exec":                2,
		"filesystem-open":             3,
		"network-attempt":             1,
		"process-clone":               1,
		"filesystem-workspace-access": 1,
	}
	for kind, want := range expectedCounts {
		if got := diagnostic.KindCounts[kind]; got != want {
			t.Fatalf("kind %q count = %d, want %d", kind, got, want)
		}
	}
	if diagnostic.KindCounts["honeytoken-access"] != 0 {
		t.Fatalf("honeytoken-access count = %d, want 0", diagnostic.KindCounts["honeytoken-access"])
	}
	const wantString = "reason= events=8 bytes=125 session_complete=true last_kind=filesystem-workspace-access kinds=process-exec:2,process-clone:1,filesystem-open:3,filesystem-workspace-access:1,network-attempt:1"
	if got := diagnostic.String(); got != wantString {
		t.Fatalf("diagnostic.String() = %q, want %q", got, wantString)
	}
}

type testFault string

func (f testFault) Error() string            { return string(f) }
func (f testFault) TraceFaultReason() string { return string(f) }

func TestCollectTraceDiagnosticPreservesCountsOnHelperFault(t *testing.T) {
	reader := &traceReader{
		budget: &traceBudget{events: 10, bytes: 1024},
		records: []TraceRecord{
			{Kind: "process-exec-expected", Bytes: 10},
			{Kind: "filesystem-open", Bytes: 20},
			{Kind: "filesystem-open", Bytes: 20},
		},
		errAfter: 3,
		err:      testFault("EVENT_LIMIT"),
	}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), reader)
	if observations != nil || limitation != "M3_DYNAMIC_OBSERVER_FAILED" {
		t.Fatalf("collectTraceDiagnostic() = (%#v, %q)", observations, limitation)
	}
	if diagnostic.Reason != "EVENT_LIMIT" {
		t.Fatalf("diagnostic.Reason = %q, want %q", diagnostic.Reason, "EVENT_LIMIT")
	}
	if diagnostic.Events != 3 || diagnostic.Bytes != 50 {
		t.Fatalf("diagnostic events=%d bytes=%d, want 3 and 50", diagnostic.Events, diagnostic.Bytes)
	}
	if diagnostic.LastKind != "filesystem-open" {
		t.Fatalf("diagnostic.LastKind = %q, want %q", diagnostic.LastKind, "filesystem-open")
	}
	if diagnostic.KindCounts["process-exec-expected"] != 1 || diagnostic.KindCounts["filesystem-open"] != 2 {
		t.Fatalf("diagnostic.KindCounts = %#v", diagnostic.KindCounts)
	}
	const wantString = "reason=EVENT_LIMIT events=3 bytes=50 session_complete=false last_kind=filesystem-open kinds=process-exec-expected:1,filesystem-open:2"
	if got := diagnostic.String(); got != wantString {
		t.Fatalf("diagnostic.String() = %q, want %q", got, wantString)
	}
}

func TestCollectTraceDiagnosticOnlyCountsValidatedKindsAndFailsClosed(t *testing.T) {
	reader := &traceReader{
		budget: &traceBudget{events: 10, bytes: 1024},
		records: []TraceRecord{
			{Kind: "process-exec", Bytes: 10},
			{Kind: "filesystem-open", Bytes: 20},
			{Kind: "raw-path:/root/.ssh/id_rsa", Bytes: 30},
		},
	}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), reader)
	if observations != nil || limitation != "M3_DYNAMIC_OBSERVER_FAILED" {
		t.Fatalf("collectTraceDiagnostic() = (%#v, %q)", observations, limitation)
	}
	if diagnostic.Reason != "UNKNOWN_EVENT_KIND" {
		t.Fatalf("diagnostic.Reason = %q, want %q", diagnostic.Reason, "UNKNOWN_EVENT_KIND")
	}
	if diagnostic.Events != 2 {
		t.Fatalf("diagnostic.Events = %d, want 2", diagnostic.Events)
	}
	if diagnostic.KindCounts["process-exec"] != 1 || diagnostic.KindCounts["filesystem-open"] != 1 {
		t.Fatalf("diagnostic.KindCounts = %#v", diagnostic.KindCounts)
	}
	if _, exists := diagnostic.KindCounts["raw-path:/root/.ssh/id_rsa"]; exists {
		t.Fatalf("diagnostic retained invalid kind in KindCounts")
	}
	if diagnostic.LastKind == "raw-path:/root/.ssh/id_rsa" {
		t.Fatalf("diagnostic.LastKind retained invalid kind: %q", diagnostic.LastKind)
	}
	if strings.Contains(diagnostic.String(), "raw-path") || strings.Contains(diagnostic.String(), "id_rsa") {
		t.Fatalf("diagnostic.String() leaked unsafe payload: %q", diagnostic.String())
	}
}

func TestCollectTraceDiagnosticByteLimitPreservesCountsAndBehavior(t *testing.T) {
	reader := &traceReader{
		budget: &traceBudget{events: 10, bytes: 50},
		records: []TraceRecord{
			{Kind: "process-exec", Bytes: 20},
			{Kind: "filesystem-open", Bytes: 20},
			{Kind: "network-attempt", Bytes: 20},
		},
	}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), reader)
	if observations != nil || limitation != "M3_DYNAMIC_OBSERVATION_LIMIT" {
		t.Fatalf("collectTraceDiagnostic() = (%#v, %q)", observations, limitation)
	}
	if diagnostic.Reason != "BYTE_LIMIT" {
		t.Fatalf("diagnostic.Reason = %q, want %q", diagnostic.Reason, "BYTE_LIMIT")
	}
	if diagnostic.Events != 2 || diagnostic.Bytes != 40 {
		t.Fatalf("diagnostic events=%d bytes=%d, want 2 and 40", diagnostic.Events, diagnostic.Bytes)
	}
	if diagnostic.KindCounts["process-exec"] != 1 || diagnostic.KindCounts["filesystem-open"] != 1 {
		t.Fatalf("diagnostic.KindCounts = %#v", diagnostic.KindCounts)
	}
	if diagnostic.KindCounts["network-attempt"] != 0 {
		t.Fatalf("network-attempt was counted despite byte limit: %d", diagnostic.KindCounts["network-attempt"])
	}
}

func TestCollectTraceDiagnosticDeterministicFormatting(t *testing.T) {
	emptyDiagnostic := TraceDiagnostic{
		Reason:     "READER_ERROR",
		KindCounts: make(map[string]uint64),
	}
	if got, want := emptyDiagnostic.String(), "reason=READER_ERROR events=0 bytes=0 session_complete=false last_kind= kinds=none"; got != want {
		t.Fatalf("empty diagnostic = %q, want %q", got, want)
	}

	populatedDiagnostic := TraceDiagnostic{
		Reason:          "",
		Events:          5,
		Bytes:           100,
		SessionComplete: true,
		LastKind:        "process-exec",
		KindCounts: map[string]uint64{
			"honeytoken-access":           1,
			"filesystem-open":             2,
			"process-exec":                1,
			"filesystem-workspace-access": 1,
		},
	}
	const wantPopulated = "reason= events=5 bytes=100 session_complete=true last_kind=process-exec kinds=process-exec:1,filesystem-open:2,filesystem-workspace-access:1,honeytoken-access:1"
	if got := populatedDiagnostic.String(); got != wantPopulated {
		t.Fatalf("populated diagnostic = %q, want %q", got, wantPopulated)
	}
}

func TestTraceBudgetsAreProfileBoundedAndFailClosed(t *testing.T) {
	for _, test := range []struct {
		profile string
		events  int
		bytes   uint64
	}{
		{"pypi-wheel", 10_000, 2 << 20}, {"pypi-wheel-pytorch-cpu", 500_000, 128 << 20}, {"pypi-wheel-pytorch-cu126", 100_000, 16 << 20}, {"untrusted", 10_000, 2 << 20},
	} {
		budget := traceBudgetForProfile(test.profile)
		if budget.events != test.events || budget.bytes != test.bytes {
			t.Fatalf("%s budget = %#v", test.profile, budget)
		}
	}
}

func TestPyTorchCPUHelperRecordLimitDoesNotUndercutCollectorBudget(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate collector test source")
	}
	helperSource, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "..", "..", "tools", "gvisor-observer", "observer.cc"))
	if err != nil {
		t.Fatalf("read observer helper source: %v", err)
	}
	const declaration = "constexpr size_t kMaxPyTorchCPURecordsPerConnection = "
	_, remainder, found := strings.Cut(string(helperSource), declaration)
	if !found {
		t.Fatal("PyTorch CPU helper record limit declaration is missing")
	}
	encodedLimit, _, found := strings.Cut(remainder, ";")
	if !found {
		t.Fatal("PyTorch CPU helper record limit declaration is malformed")
	}
	helperLimit, err := strconv.Atoi(strings.TrimSpace(encodedLimit))
	if err != nil {
		t.Fatalf("parse PyTorch CPU helper record limit: %v", err)
	}
	if helperLimit < maximumPyTorchCPUTraceEvents {
		t.Fatalf("PyTorch CPU helper record limit = %d, below collector event budget %d", helperLimit, maximumPyTorchCPUTraceEvents)
	}
}

func TestTraceObservationRejectsKindsTheProductionHelperCannotEmit(t *testing.T) {
	for _, kind := range []string{"process-unexpected", "filesystem-violation", "filesystem-write", "resource-limit"} {
		if _, _, ok := traceObservation(kind); ok {
			t.Fatalf("production trace accepted unsupported kind %q", kind)
		}
	}
}

type traceReader struct {
	records  []TraceRecord
	index    int
	err      error
	errAfter int
	budget   *traceBudget
}

func (r *traceReader) traceBudget() traceBudget {
	if r.budget != nil {
		return *r.budget
	}
	return defaultTraceBudget
}

func (r *traceReader) Next(context.Context) (TraceRecord, error) {
	if r.err != nil && r.index >= r.errAfter {
		return TraceRecord{}, r.err
	}
	if r.index == len(r.records) {
		return TraceRecord{}, io.EOF
	}
	record := r.records[r.index]
	r.index++
	return record, nil
}
