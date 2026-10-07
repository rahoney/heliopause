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
	Reason            string
	FaultSite         string
	FaultImageLocator uint64
	FaultOpen         FaultOpenDiagnostic
	FaultBudget       FaultBudgetDiagnostic
	FaultRaw          FaultRawDiagnostic
	FaultLedger       FaultLedgerDiagnostic
	Events            uint64
	Bytes             uint64
	SessionComplete   bool
	LastKind          string
	KindCounts        map[string]uint64
}

// FaultOpenDiagnostic contains only bounded kernel-derived classifications.
// It explains a rejected open and never participates in Policy or admission.
type FaultOpenDiagnostic struct {
	BuildTargetImage     bool   `json:"build_target_image,omitempty"`
	Image                string `json:"image"`
	KernelImage          string `json:"kernel_image,omitempty"`
	Role                 string `json:"role"`
	Provenance           string `json:"provenance"`
	Subject              string `json:"subject"`
	Mount                string `json:"mount"`
	Flags                uint32 `json:"flags"`
	PathLocator          uint64 `json:"path_locator,omitempty"`
	ExecutableLocator    uint64 `json:"executable_locator,omitempty"`
	MountpointLocator    uint64 `json:"mountpoint_locator,omitempty"`
	ExecutablePinned     bool   `json:"executable_pinned,omitempty"`
	GoDriverCreator      bool   `json:"go_driver_creator,omitempty"`
	CargoDriverCreator   bool   `json:"cargo_driver_creator,omitempty"`
	CargoCompilerCreator bool   `json:"cargo_compiler_creator,omitempty"`
	CargoProgramCreator  bool   `json:"cargo_program_creator,omitempty"`
	LldSameGroup         bool   `json:"lld_same_group,omitempty"`
	NativeCCInvocation   bool   `json:"native_cc_invocation,omitempty"`
	RustcVersionQuery    bool   `json:"rustc_version_query,omitempty"`
	RustcMetadataQuery   bool   `json:"rustc_metadata_query,omitempty"`
	RustcArgc            uint32 `json:"rustc_argc,omitempty"`
	RustcArgvLocator     uint64 `json:"rustc_argv_locator,omitempty"`
}

// FaultBudgetDiagnostic explains the existing helper limit using only fixed
// counters. It cannot increase a budget or repair an incomplete observation.
type FaultBudgetDiagnostic struct {
	Charged   uint64 `json:"charged"`
	Limit     uint64 `json:"limit"`
	Close     uint64 `json:"close"`
	Fcntl     uint64 `json:"fcntl"`
	Raw       uint64 `json:"raw"`
	Other     uint64 `json:"other"`
	Workspace uint64 `json:"workspace"`
}

// FaultRawDiagnostic explains a rejected send syscall without preserving its
// payload, address, path or mutable process name. It confers no permission.
type FaultRawDiagnostic struct {
	Check              string `json:"check"`
	Sysno              uint64 `json:"sysno"`
	FD                 uint64 `json:"fd"`
	KernelImage        string `json:"kernel_image"`
	Role               string `json:"role"`
	Provenance         string `json:"provenance"`
	ExecutableLocator  uint64 `json:"executable_locator"`
	SocketpairObserved bool   `json:"socketpair_observed,omitempty"`
	SocketpairFDMatch  bool   `json:"socketpair_fd_match,omitempty"`
	SocketpairDomain   int32  `json:"socketpair_domain,omitempty"`
	SocketpairType     uint32 `json:"socketpair_type,omitempty"`
	SocketpairSource   string `json:"socketpair_source,omitempty"`
}

// FaultLedgerDiagnostic describes the host-owned aggregate ledger, whose
// normalized counts are distinct from the helper's charged syscall records.
// These scalars explain a rejection without changing the ledger's decision.
type FaultLedgerDiagnostic struct {
	Events, EventLimit, Bytes, ByteLimit uint64
	RequestedEvents, RequestedBytes      uint64
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
	case "pypi-wheel-pytorch-cu126", "pypi-wheel-pytorch-cu130", "pypi-wheel-pytorch-cu132":
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
				var site interface{ TraceFaultSite() string }
				if errors.As(err, &site) {
					diagnostic.FaultSite = site.TraceFaultSite()
				}
				var locator interface{ TraceFaultImageLocator() uint64 }
				if errors.As(err, &locator) {
					diagnostic.FaultImageLocator = locator.TraceFaultImageLocator()
				}
				var open interface{ TraceFaultOpen() FaultOpenDiagnostic }
				if errors.As(err, &open) {
					diagnostic.FaultOpen = open.TraceFaultOpen()
				}
				var counters interface{ TraceFaultBudget() FaultBudgetDiagnostic }
				if errors.As(err, &counters) {
					diagnostic.FaultBudget = counters.TraceFaultBudget()
				}
				var raw interface{ TraceFaultRaw() FaultRawDiagnostic }
				if errors.As(err, &raw) {
					diagnostic.FaultRaw = raw.TraceFaultRaw()
				}
				var ledger interface{ TraceFaultLedger() FaultLedgerDiagnostic }
				if errors.As(err, &ledger) {
					diagnostic.FaultLedger = ledger.TraceFaultLedger()
				}
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
	site := ""
	if d.FaultSite != "" {
		site = " fault_site=" + d.FaultSite
	}
	if d.FaultImageLocator != 0 {
		site += fmt.Sprintf(" image_locator_fnv1a64=%016x", d.FaultImageLocator)
	}
	if d.FaultOpen.Subject != "" {
		site += fmt.Sprintf(" open_image=%s open_role=%s open_provenance=%s open_subject=%s open_mount=%s open_flags=%d", d.FaultOpen.Image, d.FaultOpen.Role, d.FaultOpen.Provenance, d.FaultOpen.Subject, d.FaultOpen.Mount, d.FaultOpen.Flags)
	}
	if d.FaultOpen.PathLocator != 0 {
		site += fmt.Sprintf(" open_locator_fnv1a64=%016x", d.FaultOpen.PathLocator)
	}
	if d.FaultOpen.KernelImage != "" {
		site += " open_kernel_image=" + d.FaultOpen.KernelImage
	}
	if d.FaultOpen.ExecutableLocator != 0 {
		site += fmt.Sprintf(" open_executable_fnv1a64=%016x", d.FaultOpen.ExecutableLocator)
		site += fmt.Sprintf(" open_executable_pinned=%t open_go_driver_creator=%t", d.FaultOpen.ExecutablePinned, d.FaultOpen.GoDriverCreator)
		site += fmt.Sprintf(" open_cargo_driver_creator=%t open_rustc_version_query=%t", d.FaultOpen.CargoDriverCreator, d.FaultOpen.RustcVersionQuery)
		if d.FaultOpen.KernelImage == "GCC" || d.FaultOpen.KernelImage == "COLLECT2" || d.FaultOpen.KernelImage == "LLD_LAUNCHER" || d.FaultOpen.KernelImage == "RUST_LLD" {
			site += fmt.Sprintf(" open_cargo_compiler_creator=%t open_native_cc_invocation=%t", d.FaultOpen.CargoCompilerCreator, d.FaultOpen.NativeCCInvocation)
		}
		if d.FaultOpen.KernelImage == "RUST_LLD" {
			site += fmt.Sprintf(" open_lld_same_group=%t", d.FaultOpen.LldSameGroup)
		}
		if d.FaultOpen.CargoProgramCreator {
			site += " open_cargo_program_creator=true"
		}
		if d.FaultOpen.BuildTargetImage {
			site += " open_build_target_image=true"
		}
		site += fmt.Sprintf(" open_rustc_metadata_query=%t open_rustc_argc=%d open_rustc_argv_fnv1a64=%016x", d.FaultOpen.RustcMetadataQuery, d.FaultOpen.RustcArgc, d.FaultOpen.RustcArgvLocator)
	}
	if d.FaultOpen.MountpointLocator != 0 {
		site += fmt.Sprintf(" open_mountpoint_fnv1a64=%016x", d.FaultOpen.MountpointLocator)
	}
	if d.FaultBudget.Limit != 0 {
		b := d.FaultBudget
		site += fmt.Sprintf(" budget_charged=%d budget_limit=%d budget_close=%d budget_fcntl=%d budget_raw=%d budget_other=%d budget_workspace=%d", b.Charged, b.Limit, b.Close, b.Fcntl, b.Raw, b.Other, b.Workspace)
	}
	if d.FaultRaw.Check != "" {
		r := d.FaultRaw
		site += fmt.Sprintf(" raw_check=%s raw_sysno=%d raw_fd=%d raw_kernel_image=%s raw_role=%s raw_provenance=%s raw_executable_fnv1a64=%016x", r.Check, r.Sysno, r.FD, r.KernelImage, r.Role, r.Provenance, r.ExecutableLocator)
		if r.SocketpairObserved {
			site += fmt.Sprintf(" raw_socketpair_observed=true raw_socketpair_fd_match=%t raw_socketpair_domain=%d raw_socketpair_type=%d raw_socketpair_source=%s", r.SocketpairFDMatch, r.SocketpairDomain, r.SocketpairType, r.SocketpairSource)
		}
	}
	if d.FaultLedger.EventLimit != 0 {
		l := d.FaultLedger
		site += fmt.Sprintf(" ledger_events=%d ledger_event_limit=%d ledger_bytes=%d ledger_byte_limit=%d ledger_requested_events=%d ledger_requested_bytes=%d", l.Events, l.EventLimit, l.Bytes, l.ByteLimit, l.RequestedEvents, l.RequestedBytes)
	}
	return fmt.Sprintf("reason=%s events=%d bytes=%d session_complete=%t last_kind=%s kinds=%s", d.Reason, d.Events, d.Bytes, d.SessionComplete, d.LastKind, formatKindCounts(d.KindCounts)) + site
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
