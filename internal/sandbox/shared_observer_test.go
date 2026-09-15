package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func registerTeardownTestSession(t *testing.T, containerID string) *observerSecuritySession {
	t.Helper()
	session := &observerSecuritySession{
		containerID:   containerID,
		generation:    strings.Repeat("a", directExecAdmissionNonceChars),
		lifecycleGate: make(chan struct{}, 1),
	}
	session.lifecycleGate <- struct{}{}
	if !activateObserverSession(session) {
		t.Fatal("activate teardown test session")
	}
	t.Cleanup(func() { forgetObserverSession(session) })
	return session
}

func attachTeardownTestSession(t *testing.T, observer *SharedObserver, session *observerSecuritySession) *sharedTraceReader {
	t.Helper()
	reader := &sharedTraceReader{observer: observer, records: make(chan TraceRecord), done: make(chan struct{}), mountReady: make(chan struct{}), attributionCounts: make(map[string]uint64), session: session}
	observer.mu.Lock()
	observer.streams[session.containerID] = reader
	observer.sessions[session.containerID] = session
	observer.mu.Unlock()
	return reader
}

func awaitTeardownTestSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func assertTeardownRequested(t *testing.T, session *observerSecuritySession) {
	t.Helper()
	session.mu.Lock()
	requested := session.teardownRequested
	session.mu.Unlock()
	if !requested {
		t.Fatal("phase 1 did not record terminal teardown intent")
	}
}

func assertSharedObserverMutexAvailable(t *testing.T, observer *SharedObserver) {
	t.Helper()
	locked := make(chan error, 1)
	go func() {
		observer.mu.Lock()
		fault := observer.fault
		observer.mu.Unlock()
		locked <- fault
	}()
	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SharedObserver.mu after phase 1")
	}
}

func writeTeardownRecord(t *testing.T, endpoint string, record helperRecord) {
	t.Helper()
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	body, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
}

func TestSharedObserverStartProfilePublicationRejectsTerminalObserver(t *testing.T) {
	for _, terminal := range []string{"fail", "close"} {
		t.Run(terminal, func(t *testing.T) {
			const containerID = "0123456789abcdef"
			var profiles, invalidations, arms atomic.Int32
			withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
				switch request.Op {
				case "profile":
					profiles.Add(1)
					_, _ = listener.Write(profileACK(request))
				case "invalidate":
					invalidations.Add(1)
					response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
					_, _ = listener.Write(response)
				case "arm":
					arms.Add(1)
				}
			})
			observer, err := NewSharedObserver(observerEndpoint(t))
			if err != nil {
				t.Fatal(err)
			}
			defer observer.Close()

			registered := make(chan *observerSecuritySession, 1)
			resumePublication := make(chan struct{})
			observer.mu.Lock()
			observer.startProfileBeforePublication = func(session *observerSecuritySession) {
				registered <- session
				<-resumePublication
			}
			observer.mu.Unlock()
			started := make(chan error, 1)
			go func() {
				_, err := observer.StartProfile(context.Background(), containerID, "pypi-wheel")
				started <- err
			}()

			session := <-registered
			if activeObserverSession(containerID) != session || profiles.Load() != 1 {
				t.Fatal("PROFILE did not register the exact session before publication pause")
			}
			observer.mu.Lock()
			_, streamPublished := observer.streams[containerID]
			_, sessionPublished := observer.sessions[containerID]
			observer.mu.Unlock()
			if streamPublished || sessionPublished {
				t.Fatal("StartProfile published state before its publication seam")
			}

			switch terminal {
			case "fail":
				observer.Fail(errors.New("terminal observer fault"))
			case "close":
				if err := observer.Close(); err != nil {
					t.Fatalf("Close() = %v", err)
				}
			}
			observer.mu.Lock()
			_, streamPublished = observer.streams[containerID]
			_, sessionPublished = observer.sessions[containerID]
			terminalObserver := observer.fault != nil || observer.closing
			observer.mu.Unlock()
			if !terminalObserver || streamPublished || sessionPublished {
				t.Fatal("terminal Phase 1 did not detach publication state")
			}

			close(resumePublication)
			if err := <-started; err == nil {
				t.Fatal("StartProfile published after terminal observer state")
			}
			if invalidations.Load() != 1 {
				t.Fatalf("rejected publication invalidations = %d, want 1", invalidations.Load())
			}
			if activeObserverSession(containerID) != nil {
				t.Fatal("rejected publication left an active observer session")
			}
			session.mu.Lock()
			sessionTerminal := session.teardownRequested && session.poisoned && session.invalidated
			session.mu.Unlock()
			if !sessionTerminal {
				t.Fatal("rejected publication did not terminally clean registered session")
			}
			observer.mu.Lock()
			_, streamPublished = observer.streams[containerID]
			_, sessionPublished = observer.sessions[containerID]
			observer.mu.Unlock()
			if streamPublished || sessionPublished {
				t.Fatal("rejected publication reinserted detached observer state")
			}
			runner := &admissionRunner{}
			if _, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/true")...); err == nil {
				t.Fatal("terminally rejected session authorized direct exec")
			}
			if len(runner.calls) != 0 || arms.Load() != 0 {
				t.Fatalf("rejected session emitted ARM or launched runner: arms=%d calls=%#v", arms.Load(), runner.calls)
			}
			if terminal == "close" {
				if _, err := observer.Start(context.Background(), "fedcba9876543210"); err == nil {
					t.Fatal("Start() accepted publication after Close")
				}
				if _, err := observer.StartProfile(context.Background(), "fedcba9876543210", "pypi-wheel"); err == nil {
					t.Fatal("StartProfile() accepted publication after Close")
				}
				if profiles.Load() != 1 {
					t.Fatalf("post-Close StartProfile sent PROFILE: %d", profiles.Load())
				}
			}
		})
	}
}

func TestSharedObserverFailPhaseOneDoesNotDeadlockOwnerFailStop(t *testing.T) {
	observer, err := NewSharedObserver(observerEndpoint(t))
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	session := registerTeardownTestSession(t, "0123456789abcdef")
	attachTeardownTestSession(t, observer, session)
	owner, err := acquireDirectExecLifecycle(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.release()

	phaseOneComplete := make(chan struct{})
	allowPhaseTwo := make(chan struct{})
	phaseTwoWaiting := make(chan struct{})
	phaseTwoComplete := make(chan struct{})
	var once sync.Once
	observer.mu.Lock()
	observer.teardownAfterPhase1 = func() {
		once.Do(func() {
			close(phaseOneComplete)
			<-allowPhaseTwo
		})
	}
	observer.teardownBeforePhase2 = func(*observerSecuritySession) { close(phaseTwoWaiting) }
	observer.teardownAfterPhase2 = func(*observerSecuritySession) { close(phaseTwoComplete) }
	observer.mu.Unlock()
	attachTestFailStop(session, func(context.Context) error {
		observer.Fail(errors.New("exact helper stopped"))
		return nil
	})

	external := make(chan struct{})
	go func() {
		observer.Fail(errors.New("stream failure"))
		close(external)
	}()
	awaitTeardownTestSignal(t, phaseOneComplete, "external teardown phase 1")
	assertTeardownRequested(t, session)
	assertSharedObserverMutexAvailable(t, observer)
	close(allowPhaseTwo)
	awaitTeardownTestSignal(t, phaseTwoWaiting, "external phase 2 lifecycle wait")

	if err := failStopAmbiguousObserverSessionOwned(session, errors.New("owner ambiguity")); !errors.Is(err, errObserverAuthorityAmbiguous) {
		t.Fatalf("owner fail-stop = %v", err)
	}
	owner.release()
	awaitTeardownTestSignal(t, phaseTwoComplete, "external phase 2 completion")
	awaitTeardownTestSignal(t, external, "external teardown completion")
	if activeObserverSession(session.containerID) != nil {
		t.Fatal("terminal session remained active")
	}
}

func TestSharedObserverCloseIntentBlocksWaitingLifecycleBeforeArm(t *testing.T) {
	observer, err := NewSharedObserver(observerEndpoint(t))
	if err != nil {
		t.Fatal(err)
	}
	session := registerTeardownTestSession(t, "0123456789abcdef")
	attachTeardownTestSession(t, observer, session)
	owner, err := acquireDirectExecLifecycle(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.release()

	phaseOneComplete := make(chan struct{})
	allowPhaseTwo := make(chan struct{})
	contended := make(chan struct{})
	var hookOnce sync.Once
	session.mu.Lock()
	session.lifecycleGateContended = func() { hookOnce.Do(func() { close(contended) }) }
	session.mu.Unlock()
	observer.mu.Lock()
	observer.teardownAfterPhase1 = func() {
		close(phaseOneComplete)
		<-allowPhaseTwo
	}
	observer.mu.Unlock()

	runner := &admissionRunner{}
	waiting := make(chan error, 1)
	go func() {
		_, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(session.containerID, boundaryLaunchMode, "/bin/true")...)
		waiting <- err
	}()
	awaitTeardownTestSignal(t, contended, "waiting lifecycle caller")
	closed := make(chan error, 1)
	go func() { closed <- observer.Close() }()
	awaitTeardownTestSignal(t, phaseOneComplete, "Close phase 1")
	assertTeardownRequested(t, session)
	assertSharedObserverMutexAvailable(t, observer)
	if _, err := observer.Start(context.Background(), "fedcba9876543210"); err == nil {
		t.Fatal("Close phase 1 accepted a new observer stream")
	}
	owner.release()
	if err := <-waiting; err == nil {
		t.Fatal("waiting caller reached ARM after teardown intent")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("waiting caller launched runner: %#v", runner.calls)
	}
	close(allowPhaseTwo)
	if err := <-closed; err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if activeObserverSession(session.containerID) != nil {
		t.Fatal("Close left a live observer session")
	}
}

func TestSharedObserverStreamTerminalTeardownIsTwoPhase(t *testing.T) {
	for _, terminal := range []string{"stream-fault", "stream-end"} {
		t.Run(terminal, func(t *testing.T) {
			observer, err := NewSharedObserver(observerEndpoint(t))
			if err != nil {
				t.Fatal(err)
			}
			defer observer.Close()
			session := registerTeardownTestSession(t, "0123456789abcdef")
			reader := attachTeardownTestSession(t, observer, session)
			owner, err := acquireDirectExecLifecycle(context.Background(), session)
			if err != nil {
				t.Fatal(err)
			}
			defer owner.release()
			phaseOneComplete := make(chan struct{})
			phaseTwoWaiting := make(chan struct{})
			phaseTwoComplete := make(chan struct{})
			observer.mu.Lock()
			observer.teardownAfterPhase1 = func() { close(phaseOneComplete) }
			observer.teardownBeforePhase2 = func(*observerSecuritySession) { close(phaseTwoWaiting) }
			observer.teardownAfterPhase2 = func(*observerSecuritySession) { close(phaseTwoComplete) }
			observer.mu.Unlock()

			record := helperRecord{ContainerID: session.containerID, Kind: terminal}
			if terminal == "stream-fault" {
				record.Reason = "STREAM_FAULT"
			}
			writeTeardownRecord(t, observer.endpoint, record)
			awaitTeardownTestSignal(t, phaseOneComplete, terminal+" phase 1")
			assertTeardownRequested(t, session)
			assertSharedObserverMutexAvailable(t, observer)
			select {
			case <-reader.done:
			default:
				t.Fatal("reader was not closed in phase 1")
			}
			awaitTeardownTestSignal(t, phaseTwoWaiting, terminal+" phase 2")
			owner.release()
			awaitTeardownTestSignal(t, phaseTwoComplete, terminal+" terminal cleanup")
			if activeObserverSession(session.containerID) != nil {
				t.Fatal("stream terminal event left a live observer session")
			}
		})
	}
}

func TestSharedObserverFailDeduplicatesReaderAndSessionSnapshot(t *testing.T) {
	observer, err := NewSharedObserver(observerEndpoint(t))
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	session := registerTeardownTestSession(t, "0123456789abcdef")
	attachTeardownTestSession(t, observer, session)
	var cleanupCount atomic.Int32
	observer.mu.Lock()
	observer.teardownAfterPhase2 = func(*observerSecuritySession) { cleanupCount.Add(1) }
	observer.mu.Unlock()
	observer.Fail(errors.New("global failure"))
	if got := cleanupCount.Load(); got != 1 {
		t.Fatalf("terminal cleanup count = %d, want 1", got)
	}
	if activeObserverSession(session.containerID) != nil {
		t.Fatal("deduplicated terminal cleanup left active session")
	}
}

func TestSharedObserverConcurrentCloseAndFailAreIdempotent(t *testing.T) {
	observer, err := NewSharedObserver(observerEndpoint(t))
	if err != nil {
		t.Fatal(err)
	}
	session := registerTeardownTestSession(t, "0123456789abcdef")
	attachTeardownTestSession(t, observer, session)
	phaseTwoEntered := make(chan struct{})
	releasePhaseTwo := make(chan struct{})
	phaseTwoComplete := make(chan struct{})
	var enteredOnce sync.Once
	var cleanupCount atomic.Int32
	observer.mu.Lock()
	observer.teardownBeforePhase2 = func(*observerSecuritySession) {
		enteredOnce.Do(func() {
			close(phaseTwoEntered)
			<-releasePhaseTwo
		})
	}
	observer.teardownAfterPhase2 = func(*observerSecuritySession) {
		cleanupCount.Add(1)
		close(phaseTwoComplete)
	}
	observer.mu.Unlock()
	failDone := make(chan struct{})
	go func() {
		observer.Fail(errors.New("global failure"))
		close(failDone)
	}()
	awaitTeardownTestSignal(t, phaseTwoEntered, "fail phase 2")
	closeDone := make(chan error, 1)
	go func() { closeDone <- observer.Close() }()
	close(releasePhaseTwo)
	awaitTeardownTestSignal(t, failDone, "Fail completion")
	if err := <-closeDone; err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if got := cleanupCount.Load(); got != 1 {
		t.Fatalf("concurrent terminal cleanup count = %d, want 1", got)
	}
	if activeObserverSession(session.containerID) != nil {
		t.Fatal("concurrent terminal operations left active session")
	}
}

func TestSharedObserverConcurrentFailAndStreamFaultAreIdempotent(t *testing.T) {
	observer, err := NewSharedObserver(observerEndpoint(t))
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	session := registerTeardownTestSession(t, "0123456789abcdef")
	attachTeardownTestSession(t, observer, session)
	phaseTwoEntered := make(chan struct{})
	releasePhaseTwo := make(chan struct{})
	phaseTwoComplete := make(chan struct{})
	var enteredOnce sync.Once
	var cleanupCount atomic.Int32
	observer.mu.Lock()
	observer.teardownBeforePhase2 = func(*observerSecuritySession) {
		enteredOnce.Do(func() {
			close(phaseTwoEntered)
			<-releasePhaseTwo
		})
	}
	observer.teardownAfterPhase2 = func(*observerSecuritySession) {
		cleanupCount.Add(1)
		close(phaseTwoComplete)
	}
	observer.mu.Unlock()
	writeTeardownRecord(t, observer.endpoint, helperRecord{ContainerID: session.containerID, Kind: "stream-fault", Reason: "STREAM_FAULT"})
	awaitTeardownTestSignal(t, phaseTwoEntered, "stream-fault phase 2")
	observer.Fail(errors.New("concurrent global failure"))
	close(releasePhaseTwo)
	awaitTeardownTestSignal(t, phaseTwoComplete, "stream-fault terminal cleanup")
	if got := cleanupCount.Load(); got != 1 {
		t.Fatalf("concurrent terminal cleanup count = %d, want 1", got)
	}
	if activeObserverSession(session.containerID) != nil {
		t.Fatal("concurrent stream-fault/fail left active session")
	}
}

func TestSharedObserverAttributesOneContainerAndClosesItsStream(t *testing.T) {
	endpoint := observerEndpoint(t)
	observer, err := NewSharedObserver(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := observer.Start(context.Background(), "0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	var diagnostic bytes.Buffer
	observer.diagnostic = &diagnostic
	for _, record := range []helperRecord{
		{ContainerID: "0123456789abcdef", Kind: "container-start"},
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "SOCKET", Family: "INET", ProcessRelation: "DIRECT_EXEC_SESSION", ProcessClass: "PYTHON"},
		{ContainerID: "0123456789abcdef", Kind: "trusted-control-network", EventSource: "CONNECT", Family: "INET6", ProcessRelation: "DIRECT_EXEC_SESSION", ProcessClass: "PYTHON"},
		{ContainerID: "0123456789abcdef", Kind: "stream-end"},
	} {
		body, _ := json.Marshal(record)
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	record, err := reader.Next(context.Background())
	if err != nil || record.Kind != "network-attempt" {
		t.Fatalf("Next() = (%#v, %v)", record, err)
	}
	if _, err := reader.Next(context.Background()); err != io.EOF {
		t.Fatalf("stream end = %v, want EOF", err)
	}
	if got, want := diagnostic.String(), "observer_attribution sequence=1 profile=default counts=NETWORK/SOCKET/INET/DIRECT_EXEC_SESSION/PYTHON:1,NETWORK/TRUSTED_CONTROL/CONNECT/INET6/DIRECT_EXEC_SESSION/PYTHON:1\n"; got != want {
		t.Fatalf("diagnostic = %q, want %q", got, want)
	}
}

func TestSharedObserverFailsClosedForUnknownContainer(t *testing.T) {
	endpoint := observerEndpoint(t)
	observer, err := NewSharedObserver(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	body, _ := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "CONNECT", Family: "INET6", ProcessRelation: "UNKNOWN", ProcessClass: "OTHER"})
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 100; attempt++ {
		observer.mu.Lock()
		fault := observer.fault
		observer.mu.Unlock()
		if fault != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("unknown container did not fail closed")
}

func TestSharedObserverFailsClosedForLatchedStreamFault(t *testing.T) {
	endpoint := observerEndpoint(t)
	observer, err := NewSharedObserver(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := observer.Start(context.Background(), "0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	body, _ := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: "EVENT_LIMIT"})
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := reader.Next(ctx); err == nil {
		t.Fatal("faulted stream completed without an incomplete error")
	} else {
		fault, ok := err.(traceFault)
		if !ok || fault.TraceFaultReason() != "EVENT_LIMIT" {
			t.Fatalf("fault = %v, want typed EVENT_LIMIT", err)
		}
	}
	if _, err := observer.Start(context.Background(), "fedcba9876543210"); err == nil {
		t.Fatal("faulted observer accepted a subsequent container mapping")
	}
}

func TestSharedObserverFlushesAttributionOnStreamFault(t *testing.T) {
	endpoint := observerEndpoint(t)
	observer, err := NewSharedObserver(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := observer.Start(context.Background(), "0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	var diagnostic bytes.Buffer
	observer.diagnostic = &diagnostic
	for _, record := range []helperRecord{
		{ContainerID: "0123456789abcdef", Kind: "container-start"},
		{ContainerID: "0123456789abcdef", Kind: "process-exec-unexpected", EventSource: "SENTRY_EXEC", ProcessClass: "NODE", ClassificationReason: "CLASS_MISMATCH", ParentRelation: "TRACKED_GROUP"},
		{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: "STREAM_FAULT"},
	} {
		body, _ := json.Marshal(record)
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	record, err := reader.Next(ctx)
	if err != nil || record.Kind != "process-exec-unexpected" {
		t.Fatalf("Next() = (%#v, %v), want process-exec-unexpected", record, err)
	}
	if _, err := reader.Next(ctx); err == nil {
		t.Fatal("stream-fault did not return error")
	} else {
		fault, ok := err.(traceFault)
		if !ok || fault.TraceFaultReason() != "STREAM_FAULT" {
			t.Fatalf("fault = %v, want typed STREAM_FAULT", err)
		}
	}
	got := diagnostic.String()
	want := "observer_attribution sequence=1 profile=default counts=PROCESS/SENTRY_EXEC/NODE/CLASS_MISMATCH/TRACKED_GROUP:1\n"
	if got != want {
		t.Fatalf("diagnostic = %q, want %q", got, want)
	}
	if strings.Count(got, "observer_attribution") != 1 {
		t.Fatalf("attribution emitted %d times, want exactly 1", strings.Count(got, "observer_attribution"))
	}
}

func FuzzDecodeHelperRecord(f *testing.F) {
	f.Add([]byte(`{"container_id":"0123456789abcdef","kind":"network-attempt","event_source":"SOCKET","family":"INET","process_relation":"UNKNOWN","process_class":"OTHER"}`))
	f.Add([]byte(`{"container_id":"invalid","kind":"network-attempt"}`))
	f.Add([]byte(`not-json`))
	f.Fuzz(func(t *testing.T, body []byte) {
		record, err := decodeHelperRecord(body)
		if err == nil && !containerIDPattern.MatchString(record.ContainerID) {
			t.Fatalf("accepted invalid container ID %q", record.ContainerID)
		}
	})
}

func TestDecodeHelperRecordAcceptsOnlyBoundedAttribution(t *testing.T) {
	valid := []helperRecord{
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "CONNECT", Family: "PACKET", ProcessRelation: "TRACKED_EXPECTED_GROUP", ProcessClass: "NPM"},
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "SENDMSG", Family: "INET", ProcessRelation: "CONTROL_GROUP", ProcessClass: "OTHER"},
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "CONNECT", Family: "INET6", ProcessRelation: "ARTIFACT_GROUP", ProcessClass: "PYTHON"},
		{ContainerID: "0123456789abcdef", Kind: "trusted-control-network", EventSource: "CONNECT", Family: "INET6", ProcessRelation: "DIRECT_EXEC_SESSION", ProcessClass: "PYTHON"},
		{ContainerID: "0123456789abcdef", Kind: "process-exec-unexpected", EventSource: "SENTRY_EXEC", ProcessClass: "SHELL", ClassificationReason: "UNMODELED_PARENT", ParentRelation: "UNTRACKED_PARENT"},
		{ContainerID: "0123456789abcdef", Kind: "process-exec-unexpected", EventSource: "SENTRY_EXEC", ProcessClass: "OTHER", ClassificationReason: "ARTIFACT_ROLE", ParentRelation: "ARTIFACT_GROUP"},
	}
	for _, record := range valid {
		payload, _ := json.Marshal(record)
		if _, err := decodeHelperRecord(payload); err != nil {
			t.Fatalf("valid attribution rejected: %#v: %v", record, err)
		}
	}
	invalid := []helperRecord{
		{ContainerID: "0123456789abcdef", Kind: "network-attempt"},
		{ContainerID: "0123456789abcdef", Kind: "network-attempt", EventSource: "SENDTO", Family: "INET", ProcessRelation: "UNKNOWN", ProcessClass: "OTHER", ClassificationReason: "OTHER"},
		{ContainerID: "0123456789abcdef", Kind: "trusted-control-network", EventSource: "SOCKET", Family: "INET", ProcessRelation: "DIRECT_EXEC_SESSION", ProcessClass: "PYTHON"},
		{ContainerID: "0123456789abcdef", Kind: "trusted-control-network", EventSource: "CONNECT", Family: "INET", ProcessRelation: "CONTROL_GROUP", ProcessClass: "PYTHON"},
		{ContainerID: "0123456789abcdef", Kind: "process-exec-unexpected", EventSource: "SYSCALL_EXECVE", ProcessClass: "/usr/bin/sh", ClassificationReason: "OTHER", ParentRelation: "ROOT"},
		{ContainerID: "0123456789abcdef", Kind: "filesystem-open", EventSource: "SOCKET"},
	}
	for _, record := range invalid {
		payload, _ := json.Marshal(record)
		if _, err := decodeHelperRecord(payload); err == nil {
			t.Fatalf("invalid attribution accepted: %#v", record)
		}
	}
}

func TestDecodeHelperRecordValidatesNormalizedCounts(t *testing.T) {
	count := uint64(1153)
	valid := helperRecord{ContainerID: "0123456789abcdef", Kind: "filesystem-workspace-access", Count: &count}
	payload, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHelperRecord(payload); err != nil {
		t.Fatalf("valid aggregate rejected: %v", err)
	}
	for _, payload := range [][]byte{
		[]byte(`{"container_id":"0123456789abcdef","kind":"filesystem-workspace-access"}`),
		[]byte(`{"container_id":"0123456789abcdef","kind":"filesystem-workspace-access","count":0}`),
		[]byte(`{"container_id":"0123456789abcdef","kind":"filesystem-workspace-access","count":-1}`),
		[]byte(`{"container_id":"0123456789abcdef","kind":"filesystem-workspace-access","count":1.5}`),
		[]byte(`{"container_id":"0123456789abcdef","kind":"filesystem-workspace-access","count":10001}`),
		[]byte(`{"container_id":"0123456789abcdef","kind":"stream-end","count":1}`),
		[]byte(`{"container_id":"0123456789abcdef","kind":"network-attempt","event_source":"CONNECT","family":"INET","process_relation":"CONTROL_GROUP","process_class":"OTHER","count":2}`),
	} {
		if _, err := decodeHelperRecord(payload); err == nil {
			t.Fatalf("invalid count payload accepted: %s", payload)
		}
	}
}

func TestSharedObserverAttributionAggregationIsBoundedAndDeterministic(t *testing.T) {
	observer := &SharedObserver{diagnostic: io.Discard}
	counts := make(map[string]uint64)
	for iteration := 0; iteration < maximumPyTorchCPUTraceEvents; iteration++ {
		record := helperRecord{Kind: "network-attempt", EventSource: "SOCKET", Family: "INET", ProcessRelation: "TRACKED_EXPECTED_GROUP", ProcessClass: "PYTHON"}
		counts[attributionKey(record)]++
	}
	if len(counts) != 1 || counts["NETWORK/SOCKET/INET/TRACKED_EXPECTED_GROUP/PYTHON"] != maximumPyTorchCPUTraceEvents {
		t.Fatalf("bounded counts = %#v", counts)
	}
	var output bytes.Buffer
	observer.diagnostic = &output
	observer.writeAttributionDiagnostic(sharedAttributionDiagnostic{sequence: 2, profile: "pypi-wheel-pytorch-cpu", counts: map[string]uint64{
		"PROCESS/SYSCALL_EXECVE/OTHER/UNKNOWN_CLASS/UNTRACKED_PARENT": 2,
		"NETWORK/CONNECT/INET6/DIRECT_EXEC_SESSION/PYTHON":            1,
	}})
	const want = "observer_attribution sequence=2 profile=pypi-wheel-pytorch-cpu counts=NETWORK/CONNECT/INET6/DIRECT_EXEC_SESSION/PYTHON:1,PROCESS/SYSCALL_EXECVE/OTHER/UNKNOWN_CLASS/UNTRACKED_PARENT:2\n"
	if output.String() != want {
		t.Fatalf("diagnostic = %q, want %q", output.String(), want)
	}
}

func TestDecodeHelperRecordRejectsOversizedPayload(t *testing.T) {
	if _, err := decodeHelperRecord(make([]byte, maximumHelperRecordBytes+1)); err == nil {
		t.Fatal("oversized observer record was accepted")
	}
	payload, err := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: strings.Repeat("A", maximumHelperRecordBytes)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHelperRecord(payload); err == nil {
		t.Fatal("oversized observer reason was accepted")
	}
}

func TestDecodeHelperRecordAcceptsBoundedSocketFaultReasons(t *testing.T) {
	for _, reason := range []string{
		"SOCKET_AF_UNSPEC", "SOCKET_AF_NETLINK", "SOCKET_AF_PACKET", "SOCKET_OTHER_FAMILY",
		"CONNECT_ADDRESS_TOO_SHORT", "CONNECT_AF_UNSPEC", "CONNECT_AF_UNIX_INVALID_LENGTH",
		"CONNECT_AF_INET_INVALID_LENGTH", "CONNECT_AF_INET6_INVALID_LENGTH", "CONNECT_AF_NETLINK_INVALID_LENGTH",
		"CONNECT_AF_PACKET_INVALID_LENGTH", "CONNECT_UNKNOWN_FAMILY",
		"PROCESS_PROVENANCE_UNKNOWN", "PROCESS_IDENTITY_REUSED", "CLONE_PROVENANCE_INVALID",
		"PROCESS_STATE_LIMIT", "CONTAINER_ROOT_INVALID", "CONTAINER_ROOT_DUPLICATE",
	} {
		payload, err := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: reason})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeHelperRecord(payload); err != nil {
			t.Fatalf("reason %q rejected: %v", reason, err)
		}
	}
}

func TestDecodeHelperRecordPreservesBoundedFailureReason(t *testing.T) {
	for _, payload := range [][]byte{[]byte(`not-json`), make([]byte, maximumHelperRecordBytes+1)} {
		_, err := decodeHelperRecord(payload)
		var fault observerFault
		if !errors.As(err, &fault) || fault.reason != "ATTRIBUTION_FAILURE" {
			t.Fatalf("error = %#v, want bounded ATTRIBUTION_FAILURE observer fault", err)
		}
	}
}

func TestDecodeHelperRecordRejectsUntrustedFaultReason(t *testing.T) {
	payload, err := json.Marshal(helperRecord{ContainerID: "0123456789abcdef", Kind: "stream-fault", Reason: "SOCKET_AF_123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeHelperRecord(payload); err == nil {
		t.Fatal("untrusted observer reason was accepted")
	}
}

func TestObserverProfileAllowlistRejectsUntrustedBudgetSelection(t *testing.T) {
	if !validObserverProfile("pypi-wheel-pytorch-cpu") || !validObserverProfile("pypi-wheel-pytorch-cu126") {
		t.Fatal("canonical PyTorch profiles rejected")
	}
	for _, profile := range []string{"pytorch:cpu", "pypi-wheel-pytorch-other", "https://example.test"} {
		if validObserverProfile(profile) {
			t.Fatalf("untrusted profile accepted: %s", profile)
		}
	}
}

func TestObserverExpectedTopologyIsProfileBoundedAndArtifactFree(t *testing.T) {
	for _, profile := range []string{"npm-lifecycle", "pypi-wheel", "github-elf"} {
		topology, ok := observerExpectedTopology(profile)
		if !ok || len(topology) < 3 {
			t.Fatalf("expected topology unavailable for %q: %#v", profile, topology)
		}
		encoded, ok := encodeExpectedTopology(topology)
		if !ok || !strings.Contains(encoded, "/|oci-root|/||1|0|0|0") || strings.Contains(encoded, "..") {
			t.Fatalf("topology for %q is not a bounded trusted encoding: %q", profile, encoded)
		}
	}
	if _, ok := observerExpectedTopology("artifact-provided"); ok {
		t.Fatal("untrusted profile supplied expected topology")
	}
	if _, ok := encodeExpectedTopology([]observerMountExpectation{{Mountpoint: "/tmp/../work", Class: "workspace", Parent: "/", FSType: "tmpfs"}}); ok {
		t.Fatal("non-normalized mountpoint was encoded")
	}
}

func TestSharedObserverAwaitMountAnchors(t *testing.T) {
	t.Run("blocks until mount anchors ready", func(t *testing.T) {
		endpoint := observerEndpoint(t)
		observer, err := NewSharedObserver(endpoint)
		if err != nil {
			t.Fatal(err)
		}
		const containerID = "0123456789abcdef"
		_, err = observer.Start(context.Background(), containerID)
		if err != nil {
			t.Fatal(err)
		}
		writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		// Socket ready != mount anchors ready: AwaitMountAnchors must not unblock yet
		shortCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if err := observer.AwaitMountAnchors(shortCtx, containerID); err == nil {
			t.Fatal("AwaitMountAnchors unblocked before mount-anchors-ready record was received")
		}

		// Now send container-start and mount-anchors-ready
		for _, record := range []helperRecord{
			{ContainerID: containerID, Kind: "container-start"},
			{ContainerID: containerID, Kind: "mount-anchors-ready"},
		} {
			body, _ := json.Marshal(record)
			if _, err := writer.Write(body); err != nil {
				t.Fatal(err)
			}
		}

		longCtx, cancelLong := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelLong()
		if err := observer.AwaitMountAnchors(longCtx, containerID); err != nil {
			t.Fatalf("AwaitMountAnchors failed after mount-anchors-ready: %v", err)
		}
	})

	t.Run("duplicate readiness rejected", func(t *testing.T) {
		endpoint := observerEndpoint(t)
		observer, err := NewSharedObserver(endpoint)
		if err != nil {
			t.Fatal(err)
		}
		const containerID = "0123456789abcdef"
		_, err = observer.Start(context.Background(), containerID)
		if err != nil {
			t.Fatal(err)
		}
		writer, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: endpoint, Net: "unixgram"})
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()

		body, _ := json.Marshal(helperRecord{ContainerID: containerID, Kind: "mount-anchors-ready"})
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
		if err := observer.AwaitMountAnchors(context.Background(), containerID); err != nil {
			t.Fatal(err)
		}

		// Second mount-anchors-ready must fail closed as TOPOLOGY_INVALID
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
		for attempt := 0; attempt < 100; attempt++ {
			observer.mu.Lock()
			fault := observer.fault
			observer.mu.Unlock()
			if fault != nil {
				var of observerFault
				if errors.As(fault, &of) && of.reason == "TOPOLOGY_INVALID" {
					return
				}
				t.Fatalf("unexpected fault: %v", fault)
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("duplicate mount-anchors-ready did not fail closed")
	})

	t.Run("unknown container rejected", func(t *testing.T) {
		endpoint := observerEndpoint(t)
		observer, err := NewSharedObserver(endpoint)
		if err != nil {
			t.Fatal(err)
		}
		err = observer.AwaitMountAnchors(context.Background(), "fedcba9876543210")
		if err == nil {
			t.Fatal("AwaitMountAnchors accepted unknown container ID")
		}
		var of observerFault
		if !errors.As(err, &of) || of.reason != "TOPOLOGY_NOT_READY" {
			t.Fatalf("error = %v, want TOPOLOGY_NOT_READY", err)
		}
	})
}

func observerEndpoint(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp("/tmp", "haa-observer-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	return filepath.Join(directory, "o.sock")
}
