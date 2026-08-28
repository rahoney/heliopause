package sandbox

import (
	"context"
	"errors"
	"io"
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
	if got, want := diagnostic.String(), "reason=READER_ERROR events=1 bytes=7 session_complete=false last_kind=network-attempt"; got != want {
		t.Fatalf("diagnostic = %q, want %q", got, want)
	}
	if contains := diagnostic.String(); contains == "" || diagnostic.LastKind == "raw-path:/secret" {
		t.Fatalf("diagnostic retained unsafe data: %q", contains)
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
