package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
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
	fault      error
	closeOnce  sync.Once
	closeErr   error
}

type helperRecord struct {
	ContainerID string `json:"container_id"`
	Kind        string `json:"kind"`
}

// decodeHelperRecord validates only the fixed, normalized helper envelope. It
// intentionally does not expose or retain remote-sink protobuf payloads.
func decodeHelperRecord(payload []byte) (helperRecord, error) {
	var record helperRecord
	if err := json.Unmarshal(payload, &record); err != nil || !containerIDPattern.MatchString(record.ContainerID) {
		return helperRecord{}, errors.New("observer record is invalid")
	}
	return record, nil
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
	observer := &SharedObserver{listener: listener, endpoint: endpoint, endpointID: endpointID, streams: make(map[string]*sharedTraceReader)}
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
		return nil, errors.New("shared observer is not configured")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.fault != nil {
		return nil, o.fault
	}
	if _, exists := o.streams[containerID]; exists {
		return nil, errors.New("duplicate Sandbox observer mapping")
	}
	reader := &sharedTraceReader{observer: o, records: make(chan TraceRecord, maximumTraceEvents+1), done: make(chan struct{})}
	o.streams[containerID] = reader
	return reader, nil
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
			o.fail(errors.New("observer mapping is unconfirmed"))
			return
		}
		if record.Kind == "stream-fault" {
			o.mu.Unlock()
			o.fail(errors.New("observer stream is incomplete"))
			return
		}
		if record.Kind == "stream-end" {
			delete(o.streams, record.ContainerID)
			close(reader.done)
			o.mu.Unlock()
			continue
		}
		if record.Kind == "container-start" {
			o.mu.Unlock()
			continue
		}
		if _, _, ok := traceObservation(record.Kind); !ok {
			o.mu.Unlock()
			o.fail(errors.New("observer record kind is invalid"))
			return
		}
		select {
		case reader.records <- TraceRecord{Kind: record.Kind, Bytes: uint64(size)}:
		default:
			o.mu.Unlock()
			o.fail(errors.New("observer event limit exceeded"))
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
	observer *SharedObserver
	records  chan TraceRecord
	done     chan struct{}
}

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
