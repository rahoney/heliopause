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

	"github.com/rahoney/heliopause/internal/core/domain"
)

// SharedObserver receives only the helper's normalized records over a trusted
// Host-only datagram socket and attributes each record to exactly one container.
type SharedObserver struct {
	listener   *net.UnixConn
	endpoint   string
	endpointID os.FileInfo
	mu         sync.Mutex
	streams    map[string]*sharedTraceReader
	sessions   map[string]*observerSecuritySession
	failStop   func(context.Context) error
	diagnostic io.Writer
	sequence   uint64
	fault      error
	closing    bool
	closeOnce  sync.Once
	closeErr   error
	// teardownAfterPhase1, teardownBeforePhase2, and teardownAfterPhase2 are
	// unexported test seams.
	// Production leaves them nil. All run after SharedObserver.mu is released.
	teardownAfterPhase1  func()
	teardownBeforePhase2 func(*observerSecuritySession)
	teardownAfterPhase2  func(*observerSecuritySession)
	// startProfileBeforePublication pauses only a fully constructed but not yet
	// published profile session. Production leaves it nil; it never runs while
	// SharedObserver.mu is held.
	startProfileBeforePublication func(*observerSecuritySession)
}

// observerMountExpectation is trusted backend configuration. It is serialized
// only across the existing Host-only control socket and is never derived from
// trace traffic or Artifact input.
type observerMountExpectation struct {
	Mountpoint string `json:"mountpoint"`
	Class      string `json:"class"`
	Parent     string `json:"parent"`
	FSType     string `json:"filesystem_type"`
	ReadOnly   bool   `json:"read_only"`
	NoExec     bool   `json:"noexec"`
	NoSUID     bool   `json:"nosuid"`
	NoDev      bool   `json:"nodev"`
}

func observerExpectedTopology(profile string) ([]observerMountExpectation, bool) {
	root := observerMountExpectation{"/", "oci-root", "/", "", true, false, false, false}
	tmp := observerMountExpectation{"/tmp", "workspace", "/", "tmpfs", false, true, true, false}
	runtime := observerMountExpectation{"/haa-runtime", "helper", "/", "tmpfs", false, false, true, false}
	switch profile {
	case "npm-lifecycle":
		return []observerMountExpectation{root, tmp, runtime}, true
	case "pypi-wheel", "pypi-wheel-pytorch-cpu", "pypi-wheel-pytorch-cu126":
		site := observerMountExpectation{"/haa-site", "workspace", "/", "tmpfs", false, false, true, false}
		return []observerMountExpectation{root, tmp, site, runtime}, true
	case "github-elf":
		work := observerMountExpectation{"/work", "workspace", "/", "tmpfs", false, false, true, false}
		return []observerMountExpectation{root, tmp, work, runtime}, true
	default:
		return nil, false
	}
}

func encodeExpectedTopology(topology []observerMountExpectation) (string, bool) {
	if len(topology) == 0 || len(topology) > 64 {
		return "", false
	}
	entries := make([]string, 0, len(topology))
	for _, mount := range topology {
		if !normalizedObserverMountpoint(mount.Mountpoint) || !normalizedObserverMountpoint(mount.Parent) ||
			strings.ContainsAny(mount.Mountpoint+mount.Class+mount.Parent+mount.FSType, ",;|\"") {
			return "", false
		}
		bools := func(value bool) string {
			if value {
				return "1"
			}
			return "0"
		}
		entries = append(entries, strings.Join([]string{mount.Mountpoint, mount.Class, mount.Parent, mount.FSType,
			bools(mount.ReadOnly), bools(mount.NoExec), bools(mount.NoSUID), bools(mount.NoDev)}, "|"))
	}
	encoded := strings.Join(entries, ";")
	if len(encoded) == 0 || len(encoded) > 4096 {
		return "", false
	}
	return encoded, true
}

func normalizedObserverMountpoint(path string) bool {
	if path == "/" {
		return true
	}
	if !strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") || strings.Contains(path, "//") {
		return false
	}
	for _, component := range strings.Split(path[1:], "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

const ObserverControlEndpoint = "/run/heliopause-observer/haa-control.sock"

// observerControlEndpoint is immutable in production. Tests substitute a
// private Unixgram endpoint to exercise the request/ack protocol without
// exposing any test seam outside this package.
var observerControlEndpoint = ObserverControlEndpoint

type helperRecord struct {
	ContainerID          string  `json:"container_id"`
	Kind                 string  `json:"kind"`
	Reason               string  `json:"reason,omitempty"`
	EventSource          string  `json:"event_source,omitempty"`
	Family               string  `json:"family,omitempty"`
	ProcessRelation      string  `json:"process_relation,omitempty"`
	ProcessClass         string  `json:"process_class,omitempty"`
	ClassificationReason string  `json:"classification_reason,omitempty"`
	ParentRelation       string  `json:"parent_relation,omitempty"`
	Count                *uint64 `json:"count,omitempty"`
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
	if err := json.Unmarshal(payload, &record); err != nil || !containerIDPattern.MatchString(record.ContainerID) || (record.Reason != "" && !validObserverReason(record.Reason)) || !validRecordCount(record) || !validAttribution(record) {
		return helperRecord{}, observerFault{reason: "ATTRIBUTION_FAILURE"}
	}
	return record, nil
}

func validRecordCount(record helperRecord) bool {
	switch record.Kind {
	case "container-start", "stream-end", "stream-fault", "trusted-control-network":
		return record.Count == nil
	case "mount-anchors-ready":
		return record.Count == nil
	case "filesystem-workspace-access":
		return record.Count != nil && *record.Count >= 1 && *record.Count <= domain.MaximumObservationSummaryCount
	case "process-exec", "process-exec-expected", "process-exec-unexpected", "filesystem-outside-workspace", "network-attempt", "honeytoken-access":
		return record.Count == nil || (*record.Count == 1)
	default:
		return false
	}
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
	case "EVENT_LIMIT", "BYTE_LIMIT", "READER_ERROR", "READER_TIMEOUT", "STREAM_FAULT", "ATTRIBUTION_FAILURE", "FINALIZATION_TIMEOUT", "UNKNOWN_EVENT_KIND", "CHANNEL_OVERFLOW", "CONTAINER_MISMATCH", "PROFILE_LOOKUP_FAILURE", "TOPOLOGY_INVALID", "TOPOLOGY_MISMATCH", "TOPOLOGY_MUTATION", "TOPOLOGY_NOT_READY", "SOCKET_AF_UNSPEC", "SOCKET_AF_NETLINK", "SOCKET_AF_PACKET", "SOCKET_OTHER_FAMILY", "CONNECT_ADDRESS_TOO_SHORT", "CONNECT_AF_UNSPEC", "CONNECT_AF_UNIX_INVALID_LENGTH", "CONNECT_AF_INET_INVALID_LENGTH", "CONNECT_AF_INET6_INVALID_LENGTH", "CONNECT_AF_NETLINK_INVALID_LENGTH", "CONNECT_AF_PACKET_INVALID_LENGTH", "CONNECT_UNKNOWN_FAMILY", "RAW_SYSCALL_INVALID", "FD_STATE_UNKNOWN", "FD_STATE_LIMIT", "EXEC_CORRELATION_INVALID", "PROCESS_PROVENANCE_UNKNOWN", "PROCESS_IDENTITY_REUSED", "CLONE_PROVENANCE_INVALID", "PROCESS_STATE_LIMIT", "CONTAINER_ROOT_INVALID", "CONTAINER_ROOT_DUPLICATE":
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
	observer := &SharedObserver{listener: listener, endpoint: endpoint, endpointID: endpointID, streams: make(map[string]*sharedTraceReader), sessions: make(map[string]*observerSecuritySession), diagnostic: os.Stderr}
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

// setFailStop binds the observer to its owning supervisor's exact helper
// process handle. It is configured only during trusted composition.
func (o *SharedObserver) setFailStop(failStop func(context.Context) error) {
	if o == nil || failStop == nil {
		return
	}
	o.mu.Lock()
	o.failStop = failStop
	o.mu.Unlock()
}

func (o *SharedObserver) failStopHelper(ctx context.Context) error {
	if o == nil || ctx == nil {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	o.mu.Lock()
	stop := o.failStop
	o.mu.Unlock()
	if stop == nil {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return stop(ctx)
}

// Close stops the trusted listener and removes its local endpoint. It is safe
// to call more than once.
func (o *SharedObserver) Close() error {
	if o == nil || o.listener == nil {
		return nil
	}
	o.closeOnce.Do(func() {
		o.mu.Lock()
		o.closing = true
		batch := o.beginTeardownLocked()
		for _, reader := range o.streams {
			closeSharedTraceReaderDone(reader)
			batch.add(reader.session)
		}
		o.streams = make(map[string]*sharedTraceReader)
		o.sessions = make(map[string]*observerSecuritySession)
		o.mu.Unlock()
		o.completeTeardown(batch)
		o.closeErr = o.listener.Close()
	})
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
	if o.closing {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if _, exists := o.streams[containerID]; exists {
		return nil, observerFault{reason: "PREVIOUS_TRACE_NOT_FINALIZED"}
	}
	reader := &sharedTraceReader{observer: o, records: make(chan TraceRecord, defaultTraceBudget.events+1), done: make(chan struct{}), mountReady: make(chan struct{}), budget: defaultTraceBudget, profile: "default", attributionCounts: make(map[string]uint64)}
	o.streams[containerID] = reader
	return reader, nil
}

func (o *SharedObserver) StartProfile(ctx context.Context, containerID, profile string) (TraceReader, error) {
	if ctx == nil || profile == "" {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	o.mu.Lock()
	active := o.fault == nil && !o.closing
	o.mu.Unlock()
	if !active {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	session, err := registerObserverProfile(ctx, containerID, profile)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	session.failStop = o.failStopHelper
	session.mu.Unlock()
	budget := traceBudgetForProfile(profile)
	shared := &sharedTraceReader{
		observer:          o,
		records:           make(chan TraceRecord, budget.events+1),
		done:              make(chan struct{}),
		mountReady:        make(chan struct{}),
		budget:            budget,
		profile:           profile,
		attributionCounts: make(map[string]uint64),
		session:           session,
	}
	// The reader/session relationship is complete before either is reachable
	// through SharedObserver maps. This closes the profile-publication race with
	// Phase 1 teardown.
	o.mu.Lock()
	hook := o.startProfileBeforePublication
	o.mu.Unlock()
	if hook != nil {
		hook(session)
	}

	var publicationErr error
	o.mu.Lock()
	switch {
	case o.fault != nil:
		publicationErr = o.fault
	case o.closing:
		publicationErr = observerFault{reason: "LIFECYCLE_ERROR"}
	case o.streams[containerID] != nil || o.sessions[containerID] != nil:
		publicationErr = observerFault{reason: "PREVIOUS_TRACE_NOT_FINALIZED"}
	default:
		// This is the sole publication point for a profiled reader. Phase 1
		// teardown either observes this complete relationship or rejects this
		// publication after it has detached the observer maps.
		o.streams[containerID] = shared
		o.sessions[containerID] = session
	}
	o.mu.Unlock()
	if publicationErr != nil {
		// registerObserverProfile already created exact observer authority. It
		// must be terminally invalidated outside SharedObserver.mu, because this
		// cleanup can wait for lifecycleGate and use the control peer.
		if cleanupErr := invalidateObserverSession(ctx, session); cleanupErr != nil {
			return nil, errors.Join(publicationErr, cleanupErr)
		}
		return nil, publicationErr
	}
	return shared, nil
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
			o.sequence++
			o.writeAttributionDiagnostic(sharedAttributionDiagnostic{o.sequence, reader.profile, reader.attributionCounts})
			reason := record.Reason
			if reason == "" {
				reason = "STREAM_FAULT"
			}
			if o.fault == nil {
				o.fault = observerFault{reason: reason}
			}
			batch := o.beginTeardownLocked(reader.session)
			closeSharedTraceReaderDone(reader)
			for _, remaining := range o.streams {
				closeSharedTraceReaderDone(remaining)
				batch.add(remaining.session)
			}
			o.streams = make(map[string]*sharedTraceReader)
			o.sessions = make(map[string]*observerSecuritySession)
			o.mu.Unlock()
			o.completeTeardown(batch)
			return
		}
		if record.Kind == "stream-end" {
			delete(o.streams, record.ContainerID)
			delete(o.sessions, record.ContainerID)
			o.sequence++
			o.writeAttributionDiagnostic(sharedAttributionDiagnostic{o.sequence, reader.profile, reader.attributionCounts})
			batch := o.beginTeardownLocked(reader.session)
			closeSharedTraceReaderDone(reader)
			o.mu.Unlock()
			o.completeTeardown(batch)
			continue
		}
		if record.Kind == "container-start" {
			o.mu.Unlock()
			continue
		}
		if record.Kind == "mount-anchors-ready" {
			select {
			case <-reader.mountReady:
				o.mu.Unlock()
				o.fail(observerFault{reason: "TOPOLOGY_INVALID"})
				return
			default:
				close(reader.mountReady)
			}
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
		count := uint64(1)
		if record.Count != nil {
			count = *record.Count
		}
		select {
		case reader.records <- TraceRecord{Kind: record.Kind, Bytes: uint64(size), Count: count}:
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
	if o.fault == nil {
		o.fault = err
	}
	batch := o.beginTeardownLocked()
	for _, reader := range o.streams {
		closeSharedTraceReaderDone(reader)
		batch.add(reader.session)
	}
	for _, session := range o.sessions {
		batch.add(session)
	}
	o.streams = make(map[string]*sharedTraceReader)
	o.sessions = make(map[string]*observerSecuritySession)
	o.mu.Unlock()
	o.completeTeardown(batch)
}

type sharedObserverTeardownBatch struct {
	sessions     []*observerSecuritySession
	seen         map[*observerSecuritySession]struct{}
	afterPhase1  func()
	beforePhase2 func(*observerSecuritySession)
	afterPhase2  func(*observerSecuritySession)
}

// beginTeardownLocked is Phase 1. SharedObserver.mu is held. It records
// irreversible session intent but never waits for a lifecycle gate or peer.
func (o *SharedObserver) beginTeardownLocked(initial ...*observerSecuritySession) sharedObserverTeardownBatch {
	batch := sharedObserverTeardownBatch{
		seen:         make(map[*observerSecuritySession]struct{}),
		afterPhase1:  o.teardownAfterPhase1,
		beforePhase2: o.teardownBeforePhase2,
		afterPhase2:  o.teardownAfterPhase2,
	}
	for _, session := range initial {
		batch.add(session)
	}
	for _, session := range o.sessions {
		batch.add(session)
	}
	return batch
}

func (b *sharedObserverTeardownBatch) add(session *observerSecuritySession) {
	if session == nil {
		return
	}
	if _, exists := b.seen[session]; exists {
		return
	}
	b.seen[session] = struct{}{}
	requestObserverSessionTeardown(session)
	b.sessions = append(b.sessions, session)
}

// completeTeardown is Phase 2. SharedObserver.mu is not held, so an external
// teardown can wait for an owner without creating a SharedObserver.mu to
// lifecycleGate lock edge.
func (o *SharedObserver) completeTeardown(batch sharedObserverTeardownBatch) {
	if len(batch.sessions) == 0 {
		return
	}
	if batch.afterPhase1 != nil {
		batch.afterPhase1()
	}
	for _, session := range batch.sessions {
		if batch.beforePhase2 != nil {
			batch.beforePhase2(session)
		}
		forgetObserverSession(session)
		if batch.afterPhase2 != nil {
			batch.afterPhase2(session)
		}
	}
}

func closeSharedTraceReaderDone(reader *sharedTraceReader) {
	if reader == nil {
		return
	}
	select {
	case <-reader.done:
	default:
		close(reader.done)
	}
}

type sharedTraceReader struct {
	observer          *SharedObserver
	records           chan TraceRecord
	done              chan struct{}
	mountReady        chan struct{}
	budget            traceBudget
	profile           string
	attributionCounts map[string]uint64
	session           *observerSecuritySession
}

// AwaitMountAnchors is intentionally available only on the trusted shared
// observer implementation. A socket-ready helper is not sufficient: this
// waits for its authoritative topology reconciliation acknowledgement.
func (o *SharedObserver) AwaitMountAnchors(ctx context.Context, containerID string) error {
	if o == nil || ctx == nil || !containerIDPattern.MatchString(containerID) {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	o.mu.Lock()
	reader := o.streams[containerID]
	fault := o.fault
	o.mu.Unlock()
	if fault != nil || reader == nil {
		if fault != nil {
			return fault
		}
		return observerFault{reason: "TOPOLOGY_NOT_READY"}
	}
	select {
	case <-reader.mountReady:
		return nil
	case <-reader.done:
		o.mu.Lock()
		err := o.fault
		o.mu.Unlock()
		if err != nil {
			return err
		}
		return observerFault{reason: "TOPOLOGY_NOT_READY"}
	case <-ctx.Done():
		return observerFault{reason: "TOPOLOGY_NOT_READY"}
	}
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
