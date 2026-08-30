package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
)

// SharedObserver receives only the helper's normalized records over a trusted
// Host-only datagram socket and attributes each record to exactly one container.
type SharedObserver struct {
	listener   *net.UnixConn
	endpoint   string
	endpointID os.FileInfo
	mu         sync.Mutex
	streams    map[string]*sharedTraceReader
	diagnostic io.Writer
	sequence   uint64
	fault      error
	closeOnce  sync.Once
	closeErr   error
}

const ObserverControlEndpoint = "/run/heliopause-observer/haa-control.sock"

type helperRecord struct {
	ContainerID          string `json:"container_id"`
	Kind                 string `json:"kind"`
	Reason               string `json:"reason,omitempty"`
	EventSource          string `json:"event_source,omitempty"`
	Family               string `json:"family,omitempty"`
	ProcessRelation      string `json:"process_relation,omitempty"`
	ProcessClass         string `json:"process_class,omitempty"`
	ClassificationReason string `json:"classification_reason,omitempty"`
	ParentRelation       string `json:"parent_relation,omitempty"`
}

type observerFault struct{ reason string }

func (e observerFault) Error() string            { return "observer stream is incomplete" }
func (e observerFault) TraceFaultReason() string { return e.reason }

const maximumHelperRecordBytes = 1024

// decodeHelperRecord validates only the fixed, normalized helper envelope. It
// intentionally does not expose or retain remote-sink protobuf payloads.
func decodeHelperRecord(payload []byte) (helperRecord, error) {
	if len(payload) == 0 || len(payload) > maximumHelperRecordBytes {
		return helperRecord{}, observerFault{reason: "ATTRIBUTION_FAILURE"}
	}
	var record helperRecord
	if err := json.Unmarshal(payload, &record); err != nil || !containerIDPattern.MatchString(record.ContainerID) || (record.Reason != "" && !validObserverReason(record.Reason)) || !validAttribution(record) {
		return helperRecord{}, observerFault{reason: "ATTRIBUTION_FAILURE"}
	}
	return record, nil
}

func validAttribution(record helperRecord) bool {
	hasAttribution := record.EventSource != "" || record.Family != "" || record.ProcessRelation != "" || record.ProcessClass != "" || record.ClassificationReason != "" || record.ParentRelation != ""
	switch record.Kind {
	case "network-attempt":
		return validFixed(record.EventSource, "SOCKET", "CONNECT", "SENDTO", "SENDMSG", "SENDMMSG") && validFixed(record.Family, "INET", "INET6", "PACKET") &&
			validFixed(record.ProcessRelation, "BOOTSTRAP_ROOT", "BOOTSTRAP_CHILD", "DIRECT_EXEC_SESSION", "TRACKED_EXPECTED_GROUP", "TRACKED_UNEXPECTED_GROUP", "CONTROL_GROUP", "ARTIFACT_GROUP", "UNKNOWN") &&
			validFixed(record.ProcessClass, "SHELL", "PYTHON", "PIP", "NODE", "NPM", "ARTIFACT", "OTHER") && record.ClassificationReason == "" && record.ParentRelation == ""
	case "trusted-control-network":
		return validFixed(record.EventSource, "CONNECT", "SENDTO", "SENDMSG", "SENDMMSG") && validFixed(record.Family, "INET", "INET6") &&
			record.ProcessRelation == "DIRECT_EXEC_SESSION" && validFixed(record.ProcessClass, "SHELL", "PYTHON", "PIP", "NODE", "NPM", "ARTIFACT", "OTHER") && record.ClassificationReason == "" && record.ParentRelation == ""
	case "process-exec-unexpected":
		return record.EventSource == "SENTRY_EXEC" && record.Family == "" && record.ProcessRelation == "" &&
			validFixed(record.ProcessClass, "SHELL", "PYTHON", "PIP", "NODE", "NPM", "ARTIFACT", "SLEEP", "MKDIR", "CAT", "CHMOD", "OTHER") &&
			validFixed(record.ClassificationReason, "INVALID_PROCESS_IDENTITY", "START_TIME_MISMATCH", "CLASS_MISMATCH", "UNMODELED_PARENT", "BOOTSTRAP_ENDED", "DIRECT_EXEC_NOT_ALLOWED", "TRACKING_LIMIT", "UNKNOWN_CLASS", "ARTIFACT_ROLE", "OTHER") &&
			validFixed(record.ParentRelation, "BOOTSTRAP_ROOT", "BOOTSTRAP_CHILD", "TRACKED_PARENT", "TRACKED_GROUP", "ROOT", "UNTRACKED_PARENT", "ARTIFACT_GROUP", "UNKNOWN")
	default:
		return !hasAttribution
	}
}

func validFixed(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validObserverReason(reason string) bool {
	switch reason {
	case "EVENT_LIMIT", "BYTE_LIMIT", "READER_ERROR", "READER_TIMEOUT", "STREAM_FAULT", "ATTRIBUTION_FAILURE", "FINALIZATION_TIMEOUT", "UNKNOWN_EVENT_KIND", "CHANNEL_OVERFLOW", "CONTAINER_MISMATCH", "PROFILE_LOOKUP_FAILURE", "SOCKET_AF_UNSPEC", "SOCKET_AF_NETLINK", "SOCKET_AF_PACKET", "SOCKET_OTHER_FAMILY", "CONNECT_ADDRESS_TOO_SHORT", "CONNECT_AF_UNSPEC", "CONNECT_AF_UNIX_INVALID_LENGTH", "CONNECT_AF_INET_INVALID_LENGTH", "CONNECT_AF_INET6_INVALID_LENGTH", "CONNECT_AF_NETLINK_INVALID_LENGTH", "CONNECT_AF_PACKET_INVALID_LENGTH", "CONNECT_UNKNOWN_FAMILY", "RAW_SYSCALL_INVALID", "FD_STATE_UNKNOWN", "FD_STATE_LIMIT", "EXEC_CORRELATION_INVALID", "PROCESS_PROVENANCE_UNKNOWN", "PROCESS_IDENTITY_REUSED", "CLONE_PROVENANCE_INVALID", "PROCESS_STATE_LIMIT", "CONTAINER_ROOT_INVALID", "CONTAINER_ROOT_DUPLICATE":
		return true
	default:
		return false
	}
}

// NewSharedObserver binds the HAA-only output endpoint used by the pinned helper.
func NewSharedObserver(endpoint string) (*SharedObserver, error) {
	return newSharedObserver(endpoint, true)
}

// newExclusiveSharedObserver is reserved for the process-scoped production
// supervisor. It never removes an existing pathname because that pathname may
// belong to another supervisor or a prior failed run.
func newExclusiveSharedObserver(endpoint string) (*SharedObserver, error) {
	return newSharedObserver(endpoint, false)
}

func newSharedObserver(endpoint string, removeStale bool) (*SharedObserver, error) {
	if endpoint == "" {
		return nil, errors.New("observer output endpoint is required")
	}
	if _, err := os.Lstat(endpoint); err == nil && !removeStale {
		return nil, errors.New("observer output endpoint is already owned")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("inspect observer output endpoint")
	}
	if removeStale {
		if err := os.Remove(endpoint); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("remove stale observer endpoint")
		}
	}
	listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		return nil, err
	}
	endpointID, err := os.Lstat(endpoint)
	if err != nil {
		_ = listener.Close()
		return nil, errors.New("capture observer output endpoint identity")
	}
	observer := &SharedObserver{listener: listener, endpoint: endpoint, endpointID: endpointID, streams: make(map[string]*sharedTraceReader), diagnostic: os.Stderr}
	go observer.receive()
	return observer, nil
}

func (o *SharedObserver) endpointIdentity() (string, os.FileInfo) {
	if o == nil {
		return "", nil
	}
	return o.endpoint, o.endpointID
}

func (o *SharedObserver) Fail(err error) {
	if err == nil {
		err = errors.New("observer failed")
	}
	o.fail(err)
}

// Close stops the trusted listener and removes its local endpoint. It is safe
// to call more than once.
func (o *SharedObserver) Close() error {
	if o == nil || o.listener == nil {
		return nil
	}
	o.closeOnce.Do(func() { o.closeErr = o.listener.Close() })
	return o.closeErr
}

func (o *SharedObserver) Start(_ context.Context, containerID string) (TraceReader, error) {
	if o == nil || o.listener == nil || !containerIDPattern.MatchString(containerID) {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.fault != nil {
		return nil, o.fault
	}
	if _, exists := o.streams[containerID]; exists {
		return nil, observerFault{reason: "PREVIOUS_TRACE_NOT_FINALIZED"}
	}
	reader := &sharedTraceReader{observer: o, records: make(chan TraceRecord, defaultTraceBudget.events+1), done: make(chan struct{}), budget: defaultTraceBudget, profile: "default", attributionCounts: make(map[string]uint64)}
	o.streams[containerID] = reader
	return reader, nil
}

func (o *SharedObserver) StartProfile(ctx context.Context, containerID, profile string) (TraceReader, error) {
	if ctx == nil || profile == "" {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if err := registerObserverProfile(ctx, containerID, profile); err != nil {
		return nil, err
	}
	reader, err := o.Start(ctx, containerID)
	if err != nil {
		return nil, err
	}
	shared := reader.(*sharedTraceReader)
	shared.profile = profile
	budget := traceBudgetForProfile(profile)
	if budget != defaultTraceBudget {
		shared.records = make(chan TraceRecord, budget.events+1)
		shared.budget = budget
	}
	return shared, nil
}

func registerObserverProfile(ctx context.Context, containerID, profile string) error {
	if !containerIDPattern.MatchString(containerID) || !validObserverProfile(profile) {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	body, err := json.Marshal(struct {
		ContainerID string `json:"container_id"`
		Profile     string `json:"profile"`
	}{containerID, profile})
	if err != nil {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	connection, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: ObserverControlEndpoint, Net: "unixgram"})
	if err != nil {
		return observerFault{reason: "HELPER_UNAVAILABLE"}
	}
	defer connection.Close()
	if _, err := connection.Write(body); err != nil {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return nil
}

func validObserverProfile(profile string) bool {
	return profile == "npm-lifecycle" || profile == "pypi-wheel" || profile == "pypi-wheel-pytorch-cpu" || profile == "pypi-wheel-pytorch-cu126" || profile == "github-elf"
}

func (o *SharedObserver) receive() {
	buffer := make([]byte, 1024)
	for {
		size, _, err := o.listener.ReadFromUnix(buffer)
		if err != nil {
			o.fail(errors.New("observer stream failed"))
			return
		}
		record, err := decodeHelperRecord(buffer[:size])
		if err != nil {
			o.fail(err)
			return
		}
		o.mu.Lock()
		reader := o.streams[record.ContainerID]
		if reader == nil {
			o.mu.Unlock()
			o.fail(observerFault{reason: "ATTRIBUTION_FAILURE"})
			return
		}
		if record.Kind == "stream-fault" {
			o.mu.Unlock()
			reason := record.Reason
			if reason == "" {
				reason = "STREAM_FAULT"
			}
			o.fail(observerFault{reason: reason})
			return
		}
		if record.Kind == "stream-end" {
			delete(o.streams, record.ContainerID)
			o.sequence++
			o.writeAttributionDiagnostic(sharedAttributionDiagnostic{o.sequence, reader.profile, reader.attributionCounts})
			close(reader.done)
			o.mu.Unlock()
			continue
		}
		if record.Kind == "container-start" {
			o.mu.Unlock()
			continue
		}
		if record.Kind == "trusted-control-network" {
			if key := attributionKey(record); key != "" {
				reader.attributionCounts[key]++
			}
			o.mu.Unlock()
			continue
		}
		if _, _, ok := traceObservation(record.Kind); !ok {
			o.mu.Unlock()
			o.fail(observerFault{reason: "UNKNOWN_EVENT_KIND"})
			return
		}
		if key := attributionKey(record); key != "" {
			reader.attributionCounts[key]++
		}
		select {
		case reader.records <- TraceRecord{Kind: record.Kind, Bytes: uint64(size)}:
		default:
			o.mu.Unlock()
			o.fail(observerFault{reason: "CHANNEL_OVERFLOW"})
			return
		}
		o.mu.Unlock()
	}
}

func (o *SharedObserver) fail(err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.fault == nil {
		o.fault = err
	}
	for _, reader := range o.streams {
		select {
		case <-reader.done:
		default:
			close(reader.done)
		}
	}
}

type sharedTraceReader struct {
	observer          *SharedObserver
	records           chan TraceRecord
	done              chan struct{}
	budget            traceBudget
	profile           string
	attributionCounts map[string]uint64
}

type sharedAttributionDiagnostic struct {
	sequence uint64
	profile  string
	counts   map[string]uint64
}

func attributionKey(record helperRecord) string {
	switch record.Kind {
	case "network-attempt":
		return strings.Join([]string{"NETWORK", record.EventSource, record.Family, record.ProcessRelation, record.ProcessClass}, "/")
	case "trusted-control-network":
		return strings.Join([]string{"NETWORK", "TRUSTED_CONTROL", record.EventSource, record.Family, record.ProcessRelation, record.ProcessClass}, "/")
	case "process-exec-unexpected":
		return strings.Join([]string{"PROCESS", record.EventSource, record.ProcessClass, record.ClassificationReason, record.ParentRelation}, "/")
	default:
		return ""
	}
}

func (o *SharedObserver) writeAttributionDiagnostic(d sharedAttributionDiagnostic) {
	if o.diagnostic == nil || len(d.counts) == 0 {
		return
	}
	keys := make([]string, 0, len(d.counts))
	for key := range d.counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, d.counts[key]))
	}
	_, _ = fmt.Fprintf(o.diagnostic, "observer_attribution sequence=%d profile=%s counts=%s\n", d.sequence, d.profile, strings.Join(parts, ","))
}

func (r *sharedTraceReader) traceBudget() traceBudget { return r.budget }

func (r *sharedTraceReader) Next(ctx context.Context) (TraceRecord, error) {
	// A stream-end datagram can arrive immediately after a final observation.
	// Preserve already accepted observations before reporting normal EOF.
	select {
	case record := <-r.records:
		return record, nil
	default:
	}
	select {
	case record := <-r.records:
		return record, nil
	case <-r.done:
		select {
		case record := <-r.records:
			return record, nil
		default:
		}
		r.observer.mu.Lock()
		err := r.observer.fault
		r.observer.mu.Unlock()
		if err != nil {
			return TraceRecord{}, err
		}
		return TraceRecord{}, io.EOF
	case <-ctx.Done():
		return TraceRecord{}, ctx.Err()
	}
}
