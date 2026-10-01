package sandbox

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ObservationResourceLease is controller-operational evidence supplied by the
// privileged typed resource port. No artifact-controlled value can create it.
type ObservationResourceLease struct {
	CgroupParent string
	CPU          int
	UsageUsec    uint64
}

// ObservationResourceClient is supplied by the composition root. The sandbox
// never imports the privileged Host helper implementation.
type ObservationResourceClient interface {
	Create(context.Context, string, string) (ObservationResourceLease, error)
	Register(context.Context, string, string, string) (ObservationResourceLease, error)
	Read(context.Context, string) (ObservationResourceLease, error)
	Terminate(context.Context, string, string) (ObservationResourceLease, error)
	Close(context.Context, string) (ObservationResourceLease, error)
}

// observationCPUWatch runs in the trusted controller, outside every artifact
// runtime. The authorization is the transaction parent cpu.stat, not a Docker
// bandwidth limit or per-process RLIMIT. Any late/missing sample terminates
// qualification and requests exact runtime cleanup.
type observationCPUWatch struct {
	client      ObservationResourceClient
	transaction string
	ledger      *observationTransaction
	cancel      context.CancelFunc
	abort       func(context.Context) error
	mu          sync.Mutex
	sampleMu    sync.Mutex
	transition  time.Time
	firstErr    error
	done        chan struct{}
}

func startObservationCPUWatch(ctx context.Context, client ObservationResourceClient, transaction string, ledger *observationTransaction, cancel context.CancelFunc, abort func(context.Context) error) (*observationCPUWatch, error) {
	if ctx == nil || client == nil || transaction == "" || ledger == nil || cancel == nil || abort == nil {
		return nil, errors.New("python observation CPU watch is unavailable")
	}
	watch := &observationCPUWatch{client: client, transaction: transaction, ledger: ledger, cancel: cancel, abort: abort, done: make(chan struct{})}
	if err := watch.sample(ctx); err != nil {
		return nil, err
	}
	go watch.run(ctx)
	return watch, nil
}

func (w *observationCPUWatch) sample(ctx context.Context) error {
	if w == nil || ctx == nil {
		return errors.New("python observation CPU sample is unavailable")
	}
	w.sampleMu.Lock()
	defer w.sampleMu.Unlock()
	if !w.transition.IsZero() {
		return errors.New("python observation CPU sample crossed a lifecycle transition")
	}
	return w.sampleLocked(ctx)
}

func (w *observationCPUWatch) sampleLocked(ctx context.Context) error {
	readContext, cancel := context.WithTimeout(ctx, w.ledger.policy.pollInterval)
	defer cancel()
	lease, err := w.client.Read(readContext, w.transaction)
	if err != nil {
		return errors.New("python observation CPU counter is unavailable")
	}
	if err := w.ledger.sampleCPU(lease.UsageUsec, time.Now()); err != nil {
		return err
	}
	return nil
}

// beginTransition and endTransition bracket a trusted Docker create/start or
// drain operation. The watcher cannot mistake an admitted but not yet
// registered cgroup member for an unrelated process. The parent counter is
// sampled on both sides, and the reserved stop bound limits the gap.
func (w *observationCPUWatch) beginTransition(ctx context.Context) error {
	if w == nil || ctx == nil {
		return errors.New("python observation lifecycle accounting is unavailable")
	}
	w.sampleMu.Lock()
	defer w.sampleMu.Unlock()
	if !w.transition.IsZero() {
		return errors.New("python observation lifecycle transition overlaps another")
	}
	if err := w.sampleLocked(ctx); err != nil {
		return err
	}
	w.transition = time.Now()
	return nil
}

func (w *observationCPUWatch) endTransition(ctx context.Context) error {
	if w == nil || ctx == nil {
		return errors.New("python observation lifecycle accounting is unavailable")
	}
	w.sampleMu.Lock()
	defer w.sampleMu.Unlock()
	started := w.transition
	w.transition = time.Time{}
	if started.IsZero() || time.Since(started) > w.ledger.policy.stopBound {
		return errors.New("python observation lifecycle exceeded its reserved stop bound")
	}
	readContext, cancel := context.WithTimeout(ctx, w.ledger.policy.stopBound-time.Since(started))
	defer cancel()
	lease, err := w.client.Read(readContext, w.transaction)
	if err != nil {
		return errors.New("python observation CPU counter after lifecycle is unavailable")
	}
	return w.ledger.sampleCPUAfterTransition(lease.UsageUsec, started, time.Now())
}

func (w *observationCPUWatch) run(ctx context.Context) {
	defer close(w.done)
	// Sample before the maximum permitted gap. A ticker at the exact gap
	// would always exceed it by the trusted helper's read latency.
	frequency := w.ledger.policy.pollInterval / 2
	if frequency <= 0 {
		frequency = w.ledger.policy.pollInterval
	}
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sampleMu.Lock()
			var err error
			if !w.transition.IsZero() {
				if time.Since(w.transition) > w.ledger.policy.stopBound {
					err = errors.New("python observation lifecycle exceeded its reserved stop bound")
				}
			} else {
				err = w.sampleLocked(ctx)
			}
			w.sampleMu.Unlock()
			if err != nil {
				w.mu.Lock()
				if w.firstErr == nil {
					w.firstErr = err
				}
				w.mu.Unlock()
				w.cancel()
				stopContext, stopCancel := context.WithTimeout(context.Background(), w.ledger.policy.stopBound)
				stopErr := w.abort(stopContext)
				stopCancel()
				if stopErr != nil {
					w.mu.Lock()
					w.firstErr = errors.Join(w.firstErr, errors.New("python observation runtime termination is unproven"))
					w.mu.Unlock()
				}
				return
			}
		}
	}
}

func (w *observationCPUWatch) stop() error {
	if w == nil {
		return errors.New("python observation CPU watch is unavailable")
	}
	w.cancel()
	<-w.done
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.firstErr
}
