package sandbox

import (
	"context"
	"errors"
	"io"

	"github.com/rahoney/heliopause/internal/core/domain"
)

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
	if reader == nil {
		return nil, "M3_DYNAMIC_OBSERVER_FAILED"
	}
	var totalBytes uint64
	observations := make([]domain.SandboxObservation, 0)
	for eventCount := 0; eventCount < maximumTraceEvents; eventCount++ {
		record, err := reader.Next(ctx)
		if errors.Is(err, io.EOF) {
			return observations, ""
		}
		if err != nil {
			return nil, "M3_DYNAMIC_OBSERVER_FAILED"
		}
		if record.Bytes > maximumTraceBytes-totalBytes {
			return nil, "M3_DYNAMIC_OBSERVATION_LIMIT"
		}
		totalBytes += record.Bytes
		category, subject, ok := traceObservation(record.Kind)
		if !ok {
			return nil, "M3_DYNAMIC_OBSERVER_FAILED"
		}
		observation, err := domain.NewSandboxObservation(category, subject)
		if err != nil {
			return nil, "M3_DYNAMIC_OBSERVER_FAILED"
		}
		observations = append(observations, observation)
	}
	return nil, "M3_DYNAMIC_OBSERVATION_LIMIT"
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
