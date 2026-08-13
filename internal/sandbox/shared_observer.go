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
	listener  *net.UnixConn
	mu        sync.Mutex
	streams   map[string]*sharedTraceReader
	fault     error
	closeOnce sync.Once
	closeErr  error
}

type helperRecord struct {
	ContainerID string `json:"container_id"`
	Kind        string `json:"kind"`
}

// NewSharedObserver binds the HAA-only output endpoint used by the pinned helper.
func NewSharedObserver(endpoint string) (*SharedObserver, error) {
	if endpoint == "" {
		return nil, errors.New("observer output endpoint is required")
	}
	if err := os.Remove(endpoint); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("remove stale observer endpoint")
	}
	listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		return nil, err
	}
	observer := &SharedObserver{listener: listener, streams: make(map[string]*sharedTraceReader)}
	go observer.receive()
	return observer, nil
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
		var record helperRecord
		if err := json.Unmarshal(buffer[:size], &record); err != nil || !containerIDPattern.MatchString(record.ContainerID) {
			o.fail(errors.New("observer record is invalid"))
			return
		}
		o.mu.Lock()
		reader := o.streams[record.ContainerID]
		if reader == nil {
			o.mu.Unlock()
			o.fail(errors.New("observer mapping is unconfirmed"))
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
	select {
	case record := <-r.records:
		return record, nil
	case <-r.done:
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
