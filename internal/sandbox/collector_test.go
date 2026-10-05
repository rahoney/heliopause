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

func TestAggregateLedgerDiagnosticSeparatesNormalizedCountFromRecords(t *testing.T) {
	ledger, err := newObservationTraceLedger(goBuildProfile)
	if err != nil {
		t.Fatal(err)
	}
	// Retain the exact former Host ledger boundary as a diagnostic regression.
	ledger.maxEvents = 10000
	for range 588 {
		if err := ledger.charge(helperRecord{Kind: "process-exec-expected"}, 114); err != nil {
			t.Fatal(err)
		}
	}
	count := uint64(10000)
	failure := ledger.charge(helperRecord{Kind: "filesystem-workspace-access", Count: &count}, 110)
	if failure == nil || ledger.events != 588 || ledger.bytes != 67032 {
		t.Fatal("rejected aggregate count changed the budget or became success")
	}
	_, limitation, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{err: failure})
	if limitation == "" || diagnostic.Reason != "EVENT_LIMIT" || diagnostic.FaultBudget.Limit != 0 {
		t.Fatal("host ledger failure was confused with a helper charged-record limit")
	}
	want := FaultLedgerDiagnostic{588, 10000, 67032, 2 << 20, 10000, 110}
	if diagnostic.FaultLedger != want || !strings.Contains(diagnostic.String(), "ledger_requested_events=10000") {
		t.Fatalf("aggregate ledger diagnostic = %+v", diagnostic.FaultLedger)
	}
	if err := ledger.charge(helperRecord{Kind: "process-exec-expected"}, 1); err != nil || ledger.events != 589 {
		t.Fatal("diagnostic changed the remaining authorization")
	}
	ledger = &observationTraceLedger{maxEvents: 3, maxBytes: 8, events: 1, bytes: 7}
	if err := ledger.charge(helperRecord{Kind: "process-exec-expected"}, 2); err == nil {
		t.Fatal("byte overflow became success")
	} else {
		_, _, diagnostic := collectTraceDiagnostic(context.Background(), &traceReader{err: err})
		if diagnostic.Reason != "BYTE_LIMIT" || diagnostic.FaultLedger.Bytes != 7 || diagnostic.FaultLedger.RequestedBytes != 2 || ledger.events != 1 || ledger.bytes != 7 {
			t.Fatal("byte ledger rejection or diagnostic changed")
		}
	}
}

func TestGoBuildAggregateLedgerNeverReplenishesAcrossStreams(t *testing.T) {
	ledger, err := newObservationTraceLedger(goBuildProfile)
	if err != nil || ledger.maxEvents != 20000 || ledger.maxBytes != 2<<20 {
		t.Fatalf("build ledger = %+v, %v", ledger, err)
	}
	// Preparation and build share one authorization; summary counts consume it.
	for _, events := range []uint64{619, 10000, 9381} {
		if err := ledger.charge(helperRecord{Kind: "filesystem-workspace-access", Count: &events}, 134); err != nil {
			t.Fatal(err)
		}
	}
	if ledger.events != 20000 {
		t.Fatal("exact aggregate bound was not charged")
	}
	err = ledger.charge(helperRecord{Kind: "process-exec-expected"}, 134)
	var fault observerFault
	if !errors.As(err, &fault) || fault.reason != "EVENT_LIMIT" || ledger.events != 20000 || ledger.bytes != 402 {
		t.Fatal("overflow succeeded or changed the remaining authorization")
	}
	for _, profile := range []string{"pypi-wheel", "pypi-wheel-pytorch-cpu", "pypi-wheel-pytorch-cu126", "pypi-wheel-pytorch-cu130", "pypi-wheel-pytorch-cu132"} {
		other, err := newObservationTraceLedger(profile)
		if err != nil || other.maxEvents != uint64(traceBudgetForProfile(profile).events) {
			t.Fatalf("existing profile budget changed: %q", profile)
		}
	}
	if traceBudgetForProfile(goBuildProfile).events != 10000 {
		t.Fatal("physical Go build collector budget changed")
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
			{Kind: "network-attempt", Bytes: 15},
			{Kind: "filesystem-workspace-access", Bytes: 18, Count: 1153},
		},
	}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), reader)
	if limitation != "" || len(observations) != 3 {
		t.Fatalf("collectTraceDiagnostic() = (%d observations, %q)", len(observations), limitation)
	}
	if !diagnostic.SessionComplete || diagnostic.Reason != "" {
		t.Fatalf("diagnostic session = complete:%t reason:%q", diagnostic.SessionComplete, diagnostic.Reason)
	}
	if diagnostic.Events != 4 || diagnostic.Bytes != 53 {
		t.Fatalf("diagnostic events=%d bytes=%d, want 4 and 53", diagnostic.Events, diagnostic.Bytes)
	}
	if diagnostic.LastKind != "filesystem-workspace-access" {
		t.Fatalf("last kind = %q, want %q", diagnostic.LastKind, "filesystem-workspace-access")
	}
	expectedCounts := map[string]uint64{
		"process-exec":                2,
		"network-attempt":             1,
		"filesystem-workspace-access": 1153,
	}
	for kind, want := range expectedCounts {
		if got := diagnostic.KindCounts[kind]; got != want {
			t.Fatalf("kind %q count = %d, want %d", kind, got, want)
		}
	}
	if diagnostic.KindCounts["honeytoken-access"] != 0 {
		t.Fatalf("honeytoken-access count = %d, want 0", diagnostic.KindCounts["honeytoken-access"])
	}
	if observations[2].Count() != 1153 {
		t.Fatalf("workspace count = %d, want 1153", observations[2].Count())
	}
	const wantString = "reason= events=4 bytes=53 session_complete=true last_kind=filesystem-workspace-access kinds=process-exec:2,filesystem-workspace-access:1153,network-attempt:1"
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
			{Kind: "filesystem-workspace-access", Bytes: 20, Count: 2},
		},
		errAfter: 2,
		err:      testFault("EVENT_LIMIT"),
	}
	observations, limitation, diagnostic := collectTraceDiagnostic(context.Background(), reader)
	if observations != nil || limitation != "M3_DYNAMIC_OBSERVER_FAILED" {
		t.Fatalf("collectTraceDiagnostic() = (%#v, %q)", observations, limitation)
	}
	if diagnostic.Reason != "EVENT_LIMIT" {
		t.Fatalf("diagnostic.Reason = %q, want %q", diagnostic.Reason, "EVENT_LIMIT")
	}
	if diagnostic.Events != 2 || diagnostic.Bytes != 30 {
		t.Fatalf("diagnostic events=%d bytes=%d, want 2 and 30", diagnostic.Events, diagnostic.Bytes)
	}
	if diagnostic.LastKind != "filesystem-workspace-access" {
		t.Fatalf("diagnostic.LastKind = %q, want %q", diagnostic.LastKind, "filesystem-workspace-access")
	}
	if diagnostic.KindCounts["process-exec-expected"] != 1 || diagnostic.KindCounts["filesystem-workspace-access"] != 2 {
		t.Fatalf("diagnostic.KindCounts = %#v", diagnostic.KindCounts)
	}
	const wantString = "reason=EVENT_LIMIT events=2 bytes=30 session_complete=false last_kind=filesystem-workspace-access kinds=process-exec-expected:1,filesystem-workspace-access:2"
	if got := diagnostic.String(); got != wantString {
		t.Fatalf("diagnostic.String() = %q, want %q", got, wantString)
	}
}

func TestCollectTraceDiagnosticOnlyCountsValidatedKindsAndFailsClosed(t *testing.T) {
	reader := &traceReader{
		budget: &traceBudget{events: 10, bytes: 1024},
		records: []TraceRecord{
			{Kind: "process-exec", Bytes: 10},
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
	if diagnostic.Events != 1 {
		t.Fatalf("diagnostic.Events = %d, want 1", diagnostic.Events)
	}
	if diagnostic.KindCounts["process-exec"] != 1 {
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
			{Kind: "filesystem-workspace-access", Bytes: 20},
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
	if diagnostic.KindCounts["process-exec"] != 1 || diagnostic.KindCounts["filesystem-workspace-access"] != 1 {
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
			"process-exec":                1,
			"filesystem-workspace-access": 1,
		},
	}
	const wantPopulated = "reason= events=5 bytes=100 session_complete=true last_kind=process-exec kinds=process-exec:1,filesystem-workspace-access:1,honeytoken-access:1"
	if got := populatedDiagnostic.String(); got != wantPopulated {
		t.Fatalf("populated diagnostic = %q, want %q", got, wantPopulated)
	}
}

func TestCollectTraceRetainsUniqueSubjectsWithSaturatingCounts(t *testing.T) {
	observations, limitation := collectTrace(context.Background(), &traceReader{records: []TraceRecord{
		{Kind: "filesystem-workspace-access", Bytes: 10, Count: 6000},
		{Kind: "filesystem-workspace-access", Bytes: 10, Count: 6000},
		{Kind: "network-attempt", Bytes: 10},
	}})
	if limitation != "" || len(observations) != 2 {
		t.Fatalf("collectTrace() = (%#v, %q)", observations, limitation)
	}
	if observations[0].Subject() != "filesystem-workspace-access" || observations[0].Count() != domain.MaximumObservationSummaryCount {
		t.Fatalf("workspace observation = %#v", observations[0])
	}
	if observations[1].Subject() != "network-attempt" || observations[1].Count() != 1 {
		t.Fatalf("network observation = %#v", observations[1])
	}
}

func TestCollectTraceFailsClosedOnInvalidAggregateCount(t *testing.T) {
	for _, count := range []uint64{domain.MaximumObservationSummaryCount + 1, ^uint64(0)} {
		_, limitation := collectTrace(context.Background(), &traceReader{records: []TraceRecord{{Kind: "filesystem-workspace-access", Bytes: 1, Count: count}}})
		if limitation != "M3_DYNAMIC_OBSERVER_FAILED" {
			t.Fatalf("count %d limitation = %q", count, limitation)
		}
	}
}

func TestTraceBudgetsAreProfileBoundedAndFailClosed(t *testing.T) {
	for _, test := range []struct {
		profile string
		events  int
		bytes   uint64
	}{
		{"pypi-wheel", 10_000, 2 << 20}, {"pypi-wheel-pytorch-cpu", 500_000, 128 << 20}, {"pypi-wheel-pytorch-cu126", 100_000, 16 << 20}, {"pypi-wheel-pytorch-cu130", 100_000, 16 << 20}, {"pypi-wheel-pytorch-cu132", 100_000, 16 << 20}, {"go-module-resolver", 10_000, 2 << 20}, {"untrusted", 10_000, 2 << 20},
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
