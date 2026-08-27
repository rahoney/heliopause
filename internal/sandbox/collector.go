package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// TraceDiagnostic is a bounded, payload-free account of a trace collection.
// It is retained only to explain a fail-closed decision; observations remain
// the sole policy input.
type TraceDiagnostic struct {
	Reason          string
	Events          uint64
	Bytes           uint64
	SessionComplete bool
	LastKind        string
}

type traceFault interface{ TraceFaultReason() string }

const (
	maximumTraceEvents = 10_000
	maximumTraceBytes  = 2 << 20
)

// TraceRecord is an already transport-framed observer event. Payloads are
// intentionally excluded: normal Sandbox results retain only bounded kinds.
type TraceRecord struct {
	Kind  string
	Bytes uint64
}

// TraceReader belongs to the trusted observer boundary, never to the Artifact.
type TraceReader interface {
	Next(context.Context) (TraceRecord, error)
}

// TraceObserver starts collecting before the Artifact is introduced.
type TraceObserver interface {
	Start(context.Context, string) (TraceReader, error)
}

type profiledTraceObserver interface {
	StartProfile(context.Context, string, string) (TraceReader, error)
}

func startTrace(ctx context.Context, observer TraceObserver, containerID, profile string) (TraceReader, error) {
	if profiled, ok := observer.(profiledTraceObserver); ok {
		return profiled.StartProfile(ctx, containerID, profile)
	}
	return observer.Start(ctx, containerID)
}

// collectTrace normalizes trusted gVisor observer kinds without retaining raw
// paths, argv, environment, file contents, or process output.
func collectTrace(ctx context.Context, reader TraceReader) ([]domain.SandboxObservation, string) {
	observations, limitation, _ := collectTraceDiagnostic(ctx, reader)
	return observations, limitation
}

func collectTraceDiagnostic(ctx context.Context, reader TraceReader) ([]domain.SandboxObservation, string, TraceDiagnostic) {
	diagnostic := TraceDiagnostic{Reason: "READER_ERROR"}
	if reader == nil {
		return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
	}
	var totalBytes uint64
	observations := make([]domain.SandboxObservation, 0)
	for eventCount := 0; eventCount < maximumTraceEvents; eventCount++ {
		record, err := reader.Next(ctx)
		if errors.Is(err, io.EOF) {
			diagnostic.Reason, diagnostic.SessionComplete = "", true
			diagnostic.Events, diagnostic.Bytes = uint64(eventCount), totalBytes
			return observations, "", diagnostic
		}
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				diagnostic.Reason = "READER_TIMEOUT"
			}
			var fault traceFault
			if errors.As(err, &fault) {
				diagnostic.Reason = fault.TraceFaultReason()
			}
			diagnostic.Events, diagnostic.Bytes = uint64(eventCount), totalBytes
			return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
		}
		if record.Bytes > maximumTraceBytes-totalBytes {
			diagnostic.Reason, diagnostic.Events, diagnostic.Bytes, diagnostic.LastKind = "BYTE_LIMIT", uint64(eventCount), totalBytes, record.Kind
			return nil, "M3_DYNAMIC_OBSERVATION_LIMIT", diagnostic
		}
		totalBytes += record.Bytes
		category, subject, ok := traceObservation(record.Kind)
		if !ok {
			diagnostic.Reason, diagnostic.Events, diagnostic.Bytes = "UNKNOWN_EVENT_KIND", uint64(eventCount), totalBytes
			return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
		}
		observation, err := domain.NewSandboxObservation(category, subject)
		if err != nil {
			return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
		}
		observations = append(observations, observation)
		diagnostic.LastKind = record.Kind
	}
	diagnostic.Reason, diagnostic.Events, diagnostic.Bytes = "EVENT_LIMIT", maximumTraceEvents, totalBytes
	return nil, "M3_DYNAMIC_OBSERVATION_LIMIT", diagnostic
}

func (d TraceDiagnostic) String() string {
	return fmt.Sprintf("reason=%s events=%d bytes=%d session_complete=%t last_kind=%s", d.Reason, d.Events, d.Bytes, d.SessionComplete, d.LastKind)
}

func traceObservation(kind string) (domain.ObservationCategory, string, bool) {
	switch kind {
	case "process-exec", "process-exec-expected", "process-exec-unexpected", "process-clone":
		return domain.ObservationProcess, kind, true
	case "filesystem-open", "filesystem-workspace-access", "filesystem-outside-workspace":
		return domain.ObservationFilesystem, kind, true
	case "network-attempt":
		return domain.ObservationNetwork, kind, true
	case "honeytoken-access":
		return domain.ObservationHoneytoken, kind, true
	default:
		return "", "", false
	}
}
