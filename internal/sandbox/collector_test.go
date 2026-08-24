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

func TestTraceObservationRejectsKindsTheProductionHelperCannotEmit(t *testing.T) {
	for _, kind := range []string{"honeytoken-access", "process-unexpected", "filesystem-violation", "filesystem-write", "resource-limit"} {
		if _, _, ok := traceObservation(kind); ok {
			t.Fatalf("production trace accepted unsupported kind %q", kind)
		}
	}
}

type traceReader struct {
	records []TraceRecord
	index   int
	err     error
}

func (r *traceReader) Next(context.Context) (TraceRecord, error) {
	if r.err != nil {
		return TraceRecord{}, r.err
	}
	if r.index == len(r.records) {
		return TraceRecord{}, io.EOF
	}
	record := r.records[r.index]
	r.index++
	return record, nil
}
