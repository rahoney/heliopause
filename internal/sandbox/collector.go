package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

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
	KindCounts      map[string]uint64
}

type traceFault interface{ TraceFaultReason() string }

const (
	maximumTraceEvents           = 10_000
	maximumTraceBytes            = 2 << 20
	maximumPyTorchCPUTraceEvents = 500_000
	maximumPyTorchCPUTraceBytes  = 128 << 20
)

type traceBudget struct {
	events int
	bytes  uint64
}

var defaultTraceBudget = traceBudget{events: maximumTraceEvents, bytes: maximumTraceBytes}

func traceBudgetForProfile(profile string) traceBudget {
	switch profile {
	case "pypi-wheel-pytorch-cpu":
		return traceBudget{events: maximumPyTorchCPUTraceEvents, bytes: maximumPyTorchCPUTraceBytes}
	case "pypi-wheel-pytorch-cu126":
		return traceBudget{events: 100_000, bytes: 16 << 20}
	default:
		return defaultTraceBudget
	}
}

// TraceRecord is an already transport-framed observer event. Payloads are
// intentionally excluded: normal Sandbox results retain only bounded kinds.
type TraceRecord struct {
	Kind  string
	Bytes uint64
	// Count is omitted for an immediate helper event and therefore means one.
	// Aggregated normalized records carry an explicit bounded count.
	Count uint64
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

type mountAnchorReadyObserver interface {
	AwaitMountAnchors(context.Context, string) error
}

func startTrace(ctx context.Context, observer TraceObserver, containerID, profile string) (TraceReader, error) {
	if profiled, ok := observer.(profiledTraceObserver); ok {
		return profiled.StartProfile(ctx, containerID, profile)
	}
	return observer.Start(ctx, containerID)
}

// awaitMountAnchors is a mandatory production gate. Test observers must opt
// in explicitly so a future production observer cannot silently bypass the
// topology-reconciliation boundary.
func awaitMountAnchors(ctx context.Context, observer TraceObserver, containerID string) error {
	ready, ok := observer.(mountAnchorReadyObserver)
	if !ok {
		return observerFault{reason: "TOPOLOGY_NOT_READY"}
	}
	return ready.AwaitMountAnchors(ctx, containerID)
}

// collectTrace normalizes trusted gVisor observer kinds without retaining raw
// paths, argv, environment, file contents, or process output.
func collectTrace(ctx context.Context, reader TraceReader) ([]domain.SandboxObservation, string) {
	observations, limitation, _ := collectTraceDiagnostic(ctx, reader)
	return observations, limitation
}

var recognizedTraceKinds = []string{
	"process-exec",
	"process-exec-expected",
	"process-exec-unexpected",
	"filesystem-workspace-access",
	"filesystem-outside-workspace",
	"network-attempt",
	"honeytoken-access",
}

func cloneKindCounts(counts map[string]uint64) map[string]uint64 {
	if counts == nil {
		return nil
	}
	cloned := make(map[string]uint64, len(counts))
	for k, v := range counts {
		cloned[k] = v
	}
	return cloned
}

func formatKindCounts(counts map[string]uint64) string {
	if len(counts) == 0 {
		return "none"
	}
	var entries []string
	for _, kind := range recognizedTraceKinds {
		if count := counts[kind]; count > 0 {
			entries = append(entries, fmt.Sprintf("%s:%d", kind, count))
		}
	}
	if len(entries) == 0 {
		return "none"
	}
	return strings.Join(entries, ",")
}

func collectTraceDiagnostic(ctx context.Context, reader TraceReader) ([]domain.SandboxObservation, string, TraceDiagnostic) {
	diagnostic := TraceDiagnostic{Reason: "READER_ERROR", KindCounts: make(map[string]uint64)}
	if reader == nil {
		return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
	}
	budget := defaultTraceBudget
	if configured, ok := reader.(interface{ traceBudget() traceBudget }); ok {
		budget = configured.traceBudget()
	}
	var totalBytes uint64
	kindCounts := make(map[string]uint64)
	observations := make([]domain.SandboxObservation, 0)
	indices := make(map[string]int)
	for eventCount := 0; eventCount < budget.events; eventCount++ {
		record, err := reader.Next(ctx)
		if errors.Is(err, io.EOF) {
			diagnostic.Reason, diagnostic.SessionComplete = "", true
			diagnostic.Events, diagnostic.Bytes = uint64(eventCount), totalBytes
			diagnostic.KindCounts = cloneKindCounts(kindCounts)
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
			diagnostic.KindCounts = cloneKindCounts(kindCounts)
			return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
		}
		if record.Bytes > budget.bytes-totalBytes {
			diagnostic.Reason, diagnostic.Events, diagnostic.Bytes = "BYTE_LIMIT", uint64(eventCount), totalBytes
			if _, _, ok := traceObservation(record.Kind); ok {
				diagnostic.LastKind = record.Kind
			}
			diagnostic.KindCounts = cloneKindCounts(kindCounts)
			return nil, "M3_DYNAMIC_OBSERVATION_LIMIT", diagnostic
		}
		totalBytes += record.Bytes
		category, subject, ok := traceObservation(record.Kind)
		if !ok {
			diagnostic.Reason, diagnostic.Events, diagnostic.Bytes = "UNKNOWN_EVENT_KIND", uint64(eventCount), totalBytes
			diagnostic.KindCounts = cloneKindCounts(kindCounts)
			return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
		}
		count := record.Count
		if count == 0 {
			count = 1
		}
		if count > domain.MaximumObservationSummaryCount {
			diagnostic.Reason, diagnostic.Events, diagnostic.Bytes = "INVALID_COUNT", uint64(eventCount), totalBytes
			diagnostic.KindCounts = cloneKindCounts(kindCounts)
			return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
		}
		if domain.MaximumObservationSummaryCount-kindCounts[record.Kind] < count {
			kindCounts[record.Kind] = domain.MaximumObservationSummaryCount
		} else {
			kindCounts[record.Kind] += count
		}
		key := string(category) + ":" + subject
		if existing, exists := indices[key]; exists {
			current := observations[existing].Count()
			if domain.MaximumObservationSummaryCount-current < count {
				count = domain.MaximumObservationSummaryCount
			} else {
				count += current
			}
			observation, err := domain.NewCountedSandboxObservation(category, subject, count)
			if err != nil {
				diagnostic.KindCounts = cloneKindCounts(kindCounts)
				return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
			}
			observations[existing] = observation
			diagnostic.LastKind = record.Kind
			continue
		}
		if len(indices) >= domain.MaximumObservationSummaryUniqueSubjects {
			diagnostic.Reason, diagnostic.Events, diagnostic.Bytes = "UNIQUE_SUBJECT_LIMIT", uint64(eventCount), totalBytes
			diagnostic.KindCounts = cloneKindCounts(kindCounts)
			return nil, "M3_DYNAMIC_OBSERVATION_LIMIT", diagnostic
		}
		observation, err := domain.NewCountedSandboxObservation(category, subject, count)
		if err != nil {
			diagnostic.KindCounts = cloneKindCounts(kindCounts)
			return nil, "M3_DYNAMIC_OBSERVER_FAILED", diagnostic
		}
		indices[key] = len(observations)
		observations = append(observations, observation)
		diagnostic.LastKind = record.Kind
	}
	diagnostic.Reason, diagnostic.Events, diagnostic.Bytes = "EVENT_LIMIT", uint64(budget.events), totalBytes
	diagnostic.KindCounts = cloneKindCounts(kindCounts)
	return nil, "M3_DYNAMIC_OBSERVATION_LIMIT", diagnostic
}

func (d TraceDiagnostic) String() string {
	return fmt.Sprintf("reason=%s events=%d bytes=%d session_complete=%t last_kind=%s kinds=%s", d.Reason, d.Events, d.Bytes, d.SessionComplete, d.LastKind, formatKindCounts(d.KindCounts))
}

func traceObservation(kind string) (domain.ObservationCategory, string, bool) {
	switch kind {
	case "process-exec", "process-exec-expected", "process-exec-unexpected":
		return domain.ObservationProcess, kind, true
	case "filesystem-workspace-access", "filesystem-outside-workspace":
		return domain.ObservationFilesystem, kind, true
	case "network-attempt":
		return domain.ObservationNetwork, kind, true
	case "honeytoken-access":
		return domain.ObservationHoneytoken, kind, true
	default:
		return "", "", false
	}
}
