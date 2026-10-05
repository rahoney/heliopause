package sandbox

import (
	"context"
	"errors"
	"fmt"
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
	stopping    chan struct{}
	stopOnce    sync.Once
}

func startObservationCPUWatch(ctx context.Context, client ObservationResourceClient, transaction string, ledger *observationTransaction, cancel context.CancelFunc, abort func(context.Context) error) (*observationCPUWatch, error) {
	if ctx == nil || client == nil || transaction == "" || ledger == nil || cancel == nil || abort == nil {
		return nil, errors.New("python observation CPU watch is unavailable")
	}
	watch := &observationCPUWatch{client: client, transaction: transaction, ledger: ledger, cancel: cancel, abort: abort, done: make(chan struct{}), stopping: make(chan struct{})}
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
	return w.sampleLocked(ctx, false)
}

func (w *observationCPUWatch) sampleLocked(ctx context.Context, termination bool) error {
	return w.sampleAccountingLocked(ctx, termination, false)
}

func (w *observationCPUWatch) sampleAccountingLocked(ctx context.Context, termination, periodic bool) error {
	readContext, cancel := context.WithTimeout(ctx, w.ledger.policy.pollInterval)
	defer cancel()
	lease, err := w.client.Read(readContext, w.transaction)
	if err != nil {
		return fmt.Errorf("python observation CPU counter is unavailable: request_context=%s read_context=%s resource=%s", pythonResourceErrorReason(ctx.Err()), pythonResourceErrorReason(readContext.Err()), pythonResourceErrorReason(err))
	}
	w.ledger.mu.Lock()
	defer w.ledger.mu.Unlock()
	// A rejected external terminal outcome is already a sticky failure. The
	// periodic watcher must keep accounting its teardown, rather than invent a
	// second admission failure while the controller enters deferred cleanup.
	// Explicit launch samples still fail, and all timing/counter/CPU limits apply.
	if periodic && w.ledger.terminalFailure {
		termination = true
	}
	return w.ledger.accountCPULocked(lease.UsageUsec, time.Now(), termination)
}

// beginTransition and endTransition bracket a trusted Docker create/start or
// drain operation. The watcher cannot mistake an admitted but not yet
// registered cgroup member for an unrelated process. The parent counter is
// sampled on both sides, and the reserved stop bound limits the gap.
func (w *observationCPUWatch) beginTransition(ctx context.Context) error {
	return w.beginLifecycle(ctx, false)
}

func (w *observationCPUWatch) beginTermination(ctx context.Context) error {
	return w.beginLifecycle(ctx, true)
}

func (w *observationCPUWatch) beginLifecycle(ctx context.Context, termination bool) error {
	if w == nil || ctx == nil {
		return errors.New("python observation lifecycle accounting is unavailable")
	}
	w.sampleMu.Lock()
	defer w.sampleMu.Unlock()
	if !w.transition.IsZero() {
		return errors.New("python observation lifecycle transition overlaps another")
	}
	if err := w.sampleLocked(ctx, termination); err != nil {
		return err
	}
	w.transition = time.Now()
	return nil
}

func (w *observationCPUWatch) endTransition(ctx context.Context) error {
	return w.endLifecycle(ctx, false)
}

func (w *observationCPUWatch) endTermination(ctx context.Context) error {
	return w.endLifecycle(ctx, true)
}

func (w *observationCPUWatch) endLifecycle(ctx context.Context, termination bool) error {
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
	return w.ledger.accountCPUAfterTransition(lease.UsageUsec, started, time.Now(), termination)
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
		case <-w.stopping:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sampleMu.Lock()
			// Stop cannot cancel an already pending trusted read. A queued tick
			// must also not start another read after ordinary cleanup requests stop.
			select {
			case <-w.stopping:
				w.sampleMu.Unlock()
				return
			default:
			}
			var err error
			if !w.transition.IsZero() {
				if time.Since(w.transition) > w.ledger.policy.stopBound {
					err = errors.New("python observation lifecycle exceeded its reserved stop bound")
				}
			} else {
				err = w.sampleAccountingLocked(ctx, false, true)
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
	// Let an in-flight sample finish under its existing poll deadline, including
	// any genuine fault and bounded emergency abort. Canceling first would turn
	// ordinary cleanup into a missing-counter failure in the trusted reader.
	w.stopOnce.Do(func() { close(w.stopping) })
	<-w.done
	w.cancel()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.firstErr
}
