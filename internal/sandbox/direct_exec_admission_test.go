package sandbox

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type admissionRunner struct {
	calls [][]string
	err   error
}

type sequenceAdmissionRunner struct {
	calls  [][]string
	errors []error
}

func (*sequenceAdmissionRunner) RequiresDirectExecAdmission() {}

func (r *sequenceAdmissionRunner) Output(_ context.Context, _ string, arguments ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), arguments...))
	if len(r.errors) == 0 {
		return nil, nil
	}
	err := r.errors[0]
	r.errors = r.errors[1:]
	return nil, err
}

func (*admissionRunner) RequiresDirectExecAdmission() {}

func (r *admissionRunner) Output(_ context.Context, _ string, arguments ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), arguments...))
	return nil, r.err
}

type admissionRequest struct {
	Op          string `json:"op"`
	ContainerID string `json:"container_id"`
	Profile     string `json:"profile"`
	Topology    string `json:"expected_topology"`
	Generation  string `json:"session_generation"`
	Mode        string `json:"mode"`
	Nonce       string `json:"nonce"`
}

func admissionACK(request admissionRequest, status string) []byte {
	response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": request.Op, "status": status,
		"container_id": request.ContainerID, "session_generation": request.Generation, "mode": request.Mode, "nonce": request.Nonce})
	return response
}

func profileACK(request admissionRequest) []byte {
	response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "profile", "status": "registered",
		"container_id": request.ContainerID, "profile": request.Profile, "expected_topology": request.Topology, "session_generation": request.Generation})
	return response
}

func registerAdmissionSession(t *testing.T, containerID string) *observerSecuritySession {
	t.Helper()
	session, err := registerObserverProfile(context.Background(), containerID, "pypi-wheel")
	if err != nil {
		t.Fatalf("registerObserverProfile() error = %v", err)
	}
	t.Cleanup(func() { forgetObserverSession(session) })
	return session
}

func withShortAdmissionTimeout(t *testing.T) {
	t.Helper()
	previous := directExecAdmissionTimeout
	directExecAdmissionTimeout = 100 * time.Millisecond
	t.Cleanup(func() { directExecAdmissionTimeout = previous })
}

func attachTestFailStop(session *observerSecuritySession, failStop func(context.Context) error) {
	session.mu.Lock()
	session.failStop = failStop
	session.mu.Unlock()
}

func withAdmissionControl(t *testing.T, respond func(*net.UnixConn, admissionRequest)) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("SOCK_SEQPACKET test transport requires Linux")
	}
	endpoint := filepath.Join(t.TempDir(), "control.sock")
	listener, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: endpoint, Net: "unixpacket"})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(endpoint, 0o700)
	previous := observerControlEndpoint
	observerControlEndpoint = endpoint
	t.Cleanup(func() {
		observerControlEndpoint = previous
		_ = listener.Close()
	})
	go func() {
		for {
			connection, acceptErr := listener.AcceptUnix()
			if acceptErr != nil {
				return
			}
			go func(connection *net.UnixConn) {
				defer connection.Close()
				buffer := make([]byte, maximumControlDatagramBytes)
				for {
					size, readErr := connection.Read(buffer)
					if readErr != nil {
						return
					}
					var request admissionRequest
					if json.Unmarshal(buffer[:size], &request) == nil {
						respond(connection, request)
					}
				}
			}(connection)
		}
	}()
}

func encodeVarint(val uint64) []byte {
	var buf []byte
	for val >= 0x80 {
		buf = append(buf, byte(val&0x7f)|0x80)
		val >>= 7
	}
	buf = append(buf, byte(val&0x7f))
	return buf
}

func encodeField(fieldNum int, wireType int, data []byte) []byte {
	tag := uint64((fieldNum << 3) | wireType)
	var buf []byte
	buf = append(buf, encodeVarint(tag)...)
	if wireType == 2 {
		buf = append(buf, encodeVarint(uint64(len(data)))...)
		buf = append(buf, data...)
	}
	return buf
}

func encodeVarintField(fieldNum int, val uint64) []byte {
	tag := uint64((fieldNum << 3) | 0)
	var buf []byte
	buf = append(buf, encodeVarint(tag)...)
	buf = append(buf, encodeVarint(val)...)
	return buf
}

func buildObserverPacket(msgType uint16, payload []byte) []byte {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint16(header[0:2], 8)
	binary.LittleEndian.PutUint16(header[2:4], msgType)
	binary.LittleEndian.PutUint32(header[4:8], 0)
	return append(header, payload...)
}

func buildContainerStartPayload(containerID string) []byte {
	var ctxData []byte
	ctxData = append(ctxData, encodeVarintField(4, 1)...)
	ctxData = append(ctxData, encodeVarintField(5, 1)...)
	ctxData = append(ctxData, encodeField(6, 2, []byte(containerID))...)
	ctxData = append(ctxData, encodeVarintField(10, 0)...)
	ctxData = append(ctxData, encodeVarintField(11, 0)...)

	var startMsg []byte
	startMsg = append(startMsg, encodeField(1, 2, ctxData)...)
	startMsg = append(startMsg, encodeField(2, 2, []byte(containerID))...)
	return startMsg
}

var testExecThreadGroupSeq uint32 = 7000

func buildSentryExecPayload(containerID, mode, nonce string) []byte {
	tgid := atomic.AddUint32(&testExecThreadGroupSeq, 1)
	var ctxData []byte
	ctxData = append(ctxData, encodeVarintField(4, uint64(tgid))...)
	ctxData = append(ctxData, encodeVarintField(5, uint64(tgid)*10000)...)
	ctxData = append(ctxData, encodeField(6, 2, []byte(containerID))...)
	ctxData = append(ctxData, encodeField(9, 2, []byte("haa-boundary"))...)
	ctxData = append(ctxData, encodeVarintField(10, 0)...)
	ctxData = append(ctxData, encodeVarintField(11, 1)...)

	bPath := []byte("/haa-runtime/haa-boundary")
	var execMsg []byte
	execMsg = append(execMsg, encodeField(1, 2, ctxData)...)
	execMsg = append(execMsg, encodeField(2, 2, bPath)...)
	execMsg = append(execMsg, encodeField(3, 2, bPath)...)
	execMsg = append(execMsg, encodeField(3, 2, []byte(mode))...)
	execMsg = append(execMsg, encodeField(3, 2, []byte(nonce))...)
	execMsg = append(execMsg, encodeField(3, 2, []byte("/bin/true"))...)
	execMsg = append(execMsg, encodeField(17, 2, bPath)...)
	return execMsg
}

func findObserverBinary(t *testing.T) string {
	t.Helper()
	candidate := "/usr/libexec/heliopause/haa_gvisor_observer"
	if st, err := os.Stat(candidate); err == nil && st.Mode().IsRegular() && st.Mode()&0o111 != 0 {
		return candidate
	}
	t.Skip("observer binary not found")
	return ""
}

type realObserverTestFixture struct {
	cmd         *exec.Cmd
	dir         string
	remotePath  string
	outputPath  string
	controlPath string
	remoteConn  *net.UnixConn
	outputConn  *net.UnixConn
}

func startRealObserverForTest(t *testing.T, containerID string) *realObserverTestFixture {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("real gVisor observer tests require Linux")
	}
	binary := findObserverBinary(t)
	dir, err := os.MkdirTemp("/tmp", "haa-real-obs-")
	if err != nil {
		dir, err = os.MkdirTemp("", "haa-real-obs-")
		if err != nil {
			t.Fatal(err)
		}
	}
	_ = os.Chmod(dir, 0o700)
	remotePath := filepath.Join(dir, "remote.sock")
	outputPath := filepath.Join(dir, "output.sock")
	controlPath := filepath.Join(dir, "control.sock")

	outConn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: outputPath, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(outputPath, 0o700)

	r, w, err := os.Pipe()
	if err != nil {
		_ = outConn.Close()
		t.Fatal(err)
	}

	cmd := exec.Command(binary, remotePath, outputPath, controlPath, "--ready-fd=3")
	cmd.ExtraFiles = []*os.File{w}
	if err := cmd.Start(); err != nil {
		_ = r.Close()
		_ = w.Close()
		_ = outConn.Close()
		t.Fatal(err)
	}
	_ = w.Close()

	ready := make([]byte, 1)
	if _, err := r.Read(ready); err != nil || ready[0] != 'R' {
		_ = r.Close()
		_ = cmd.Process.Kill()
		_ = outConn.Close()
		t.Fatalf("observer ready signal failed: %v", err)
	}
	_ = r.Close()

	_ = os.Chmod(remotePath, 0o700)
	_ = os.Chmod(controlPath, 0o700)

	remConn, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: remotePath, Net: "unixpacket"})
	if err != nil {
		_ = cmd.Process.Kill()
		_ = outConn.Close()
		t.Fatal(err)
	}

	if _, err := remConn.Write([]byte{0x08, 0x01}); err != nil {
		_ = remConn.Close()
		_ = cmd.Process.Kill()
		_ = outConn.Close()
		t.Fatal(err)
	}
	hsBuf := make([]byte, 16)
	if _, err := remConn.Read(hsBuf); err != nil {
		_ = remConn.Close()
		_ = cmd.Process.Kill()
		_ = outConn.Close()
		t.Fatal(err)
	}

	startMsg := buildContainerStartPayload(containerID)
	if _, err := remConn.Write(buildObserverPacket(1, startMsg)); err != nil {
		_ = remConn.Close()
		_ = cmd.Process.Kill()
		_ = outConn.Close()
		t.Fatal(err)
	}
	outBuf := make([]byte, 4096)
	if _, err := outConn.Read(outBuf); err != nil {
		_ = remConn.Close()
		_ = cmd.Process.Kill()
		_ = outConn.Close()
		t.Fatal(err)
	}

	prevEndpoint := observerControlEndpoint
	observerControlEndpoint = controlPath

	fixture := &realObserverTestFixture{
		cmd:         cmd,
		dir:         dir,
		remotePath:  remotePath,
		outputPath:  outputPath,
		controlPath: controlPath,
		remoteConn:  remConn,
		outputConn:  outConn,
	}

	t.Cleanup(func() {
		observerControlEndpoint = prevEndpoint
		_ = remConn.Close()
		_ = outConn.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		_ = os.Remove(remotePath)
		_ = os.Remove(outputPath)
		_ = os.Remove(controlPath)
		_ = os.RemoveAll(dir)
	})

	return fixture
}

func (f *realObserverTestFixture) sendSentryExec(t *testing.T, containerID, mode, nonce string) {
	t.Helper()
	execMsg := buildSentryExecPayload(containerID, mode, nonce)
	if _, err := f.remoteConn.Write(buildObserverPacket(3, execMsg)); err != nil {
		t.Fatalf("send sentry exec: %v", err)
	}
	outBuf := make([]byte, 4096)
	n, err := f.outputConn.Read(outBuf)
	if err != nil {
		t.Fatalf("read output record: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(outBuf[:n], &record); err != nil || record["kind"] != "process-exec-expected" {
		t.Fatalf("unexpected record: %s", string(outBuf[:n]))
	}
}

type testRealExecRunner struct {
	onExec func(nonce string)
	err    error
	calls  [][]string
}

func (*testRealExecRunner) RequiresDirectExecAdmission() {}

func (r *testRealExecRunner) Output(_ context.Context, _ string, arguments ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), arguments...))
	for i := 0; i+1 < len(arguments); i++ {
		if arguments[i] == boundaryLaunchMode || arguments[i] == boundaryPythonHandoffMode || arguments[i] == boundaryELFHandoffMode ||
			arguments[i] == boundaryOriginLaunchMode || arguments[i] == boundaryOriginPythonHandoffMode || arguments[i] == boundaryOriginELFHandoffMode {
			if r.onExec != nil {
				r.onExec(arguments[i+1])
			}
			break
		}
	}
	return nil, r.err
}

func TestDirectExecOriginRequiresCanonicalBoundaryInvocation(t *testing.T) {
	arguments := boundaryExecArguments("0123456789abcdef", boundaryLaunchMode, "python", "-I")
	containerID, mode, nonceIndex, ok := directExecOrigin(arguments, "docker")
	if !ok || containerID != "0123456789abcdef" || mode != boundaryLaunchMode || nonceIndex != 6 {
		t.Fatalf("directExecOrigin() = (%q, %q, %d, %t)", containerID, mode, nonceIndex, ok)
	}
	for _, invalid := range []struct {
		binary string
		args   []string
	}{
		{"podman", arguments},
		{"docker", append([]string{"exec", "--user", "1000:1000"}, arguments[3:]...)},
		{"docker", []string{"exec", "--user", boundaryBootstrapUser, "0123456789abcdef", boundaryHelperPath, boundaryLaunchMode}},
		{"docker", []string{"exec", "--user", boundaryBootstrapUser, "0123456789abcdef", boundaryHelperPath, "--launch-anything", "python"}},
	} {
		if _, _, _, ok := directExecOrigin(invalid.args, invalid.binary); ok {
			t.Fatalf("directExecOrigin(%q, %q) unexpectedly accepted", invalid.binary, invalid.args)
		}
	}
}

func TestDirectExecAdmissionNonceIsCanonicalCryptographicCapability(t *testing.T) {
	nonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		t.Fatalf("newDirectExecAdmissionNonce() error = %v", err)
	}
	if !validDirectExecAdmissionNonce(nonce) || len(nonce) != directExecAdmissionNonceChars {
		t.Fatalf("nonce = %q is not canonical", nonce)
	}
	second, err := newDirectExecAdmissionNonce()
	if err != nil || nonce == second {
		t.Fatalf("second nonce = %q, err = %v", second, err)
	}
	for _, invalid := range []string{"", nonce[:63], "A" + nonce[1:], "g" + nonce[1:]} {
		if validDirectExecAdmissionNonce(invalid) {
			t.Fatalf("invalid nonce accepted: %q", invalid)
		}
	}
}

func TestObserverSecurityGenerationIsFreshCanonicalAndFailsClosedOnEntropyError(t *testing.T) {
	first, err := newObserverSecurityGeneration()
	if err != nil || !validObserverSecurityGeneration(first) {
		t.Fatalf("newObserverSecurityGeneration() = %q, %v", first, err)
	}
	second, err := newObserverSecurityGeneration()
	if err != nil || first == second {
		t.Fatalf("fresh generation = %q, %v", second, err)
	}
	original := sessionEntropy
	sessionEntropy = func([]byte) (int, error) { return 0, errors.New("entropy unavailable") }
	t.Cleanup(func() { sessionEntropy = original })
	if generation, err := newObserverSecurityGeneration(); err == nil || generation != "" {
		t.Fatalf("entropy failure generation = %q, %v", generation, err)
	}
}

func TestStrictControlACKDecoderRejectsAmbiguousJSON(t *testing.T) {
	valid := []byte(`{"op":"ack","ack_op":"arm","status":"armed","container_id":"0123456789abcdef","session_generation":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","mode":"--launch","nonce":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`)
	ack, err := decodeStrictControlAck(valid)
	if err != nil {
		t.Fatalf("decodeStrictControlAck(valid) error = %v", err)
	}
	required := map[string]string{"op": "ack", "ack_op": "arm", "status": "armed", "container_id": "0123456789abcdef", "session_generation": strings.Repeat("a", 64), "mode": "--launch", "nonce": strings.Repeat("b", 64)}
	if !exactControlAck(ack, required) {
		t.Fatal("valid exact ACK did not correlate")
	}
	for _, payload := range [][]byte{
		[]byte(`{"op":"ack","unknown":"value"}`),
		[]byte(`{"op":"ack","op":"ack"}`),
		[]byte(`{"op":{"nested":"value"}}`),
		[]byte(`{"op":"ack"}{"op":"ack"}`),
		append(append([]byte(nil), valid...), make([]byte, maximumControlDatagramBytes)...),
	} {
		if _, err := decodeStrictControlAck(payload); err == nil {
			t.Fatalf("ambiguous ACK accepted: %q", payload)
		}
	}
	if exactControlAck(strictControlAck{"op": "ack"}, required) {
		t.Fatal("missing ACK fields correlated")
	}
}

func TestSessionBindsProfileAdmissionAndInvalidation(t *testing.T) {
	var mu sync.Mutex
	registrations := make(map[string]string)
	admissions := make(map[string]bool)
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		mu.Lock()
		defer mu.Unlock()
		switch request.Op {
		case "profile":
			registrations[request.ContainerID] = request.Generation
			_, _ = listener.Write(profileACK(request))
		case "arm":
			status := "rejected"
			if registrations[request.ContainerID] == request.Generation {
				admissions[request.Generation+request.Nonce] = true
				status = "armed"
			}
			_, _ = listener.Write(admissionACK(request, status))
		case "cancel":
			status := "rejected"
			if registrations[request.ContainerID] == request.Generation && admissions[request.Generation+request.Nonce] {
				delete(admissions, request.Generation+request.Nonce)
				status = "cancelled"
			}
			_, _ = listener.Write(admissionACK(request, status))
		case "invalidate":
			status := "rejected"
			if registrations[request.ContainerID] == request.Generation {
				delete(registrations, request.ContainerID)
				for identity := range admissions {
					if strings.HasPrefix(identity, request.Generation) {
						delete(admissions, identity)
					}
				}
				status = "invalidated"
			}
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": status,
				"container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	first := registerAdmissionSession(t, containerID)
	nonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		t.Fatal(err)
	}
	firstAdmission := directExecAdmission{containerID: containerID, generation: first.generation, mode: boundaryLaunchMode, nonce: nonce, session: first}
	if err := controlDirectExecAdmission(context.Background(), "arm", firstAdmission); err != nil {
		t.Fatalf("arm first session: %v", err)
	}
	// Production invalidation poisons before exercising terminal cleanup.
	if err := invalidateObserverSession(context.Background(), first); err != nil {
		t.Fatalf("invalidate first session: %v", err)
	}
	if err := controlDirectExecAdmission(context.Background(), "cancel", firstAdmission); err == nil {
		t.Fatal("old generation cancelled an invalidated admission")
	}
	second := registerAdmissionSession(t, containerID)
	if second.generation == first.generation {
		t.Fatal("fresh registration reused its security generation")
	}
	if activeObserverSession(containerID) != second {
		t.Fatal("new registration did not replace invalidated security session")
	}
	if err := controlDirectExecAdmission(context.Background(), "arm", firstAdmission); err == nil {
		t.Fatal("old generation armed after new registration")
	}
	mu.Lock()
	if len(admissions) != 0 {
		mu.Unlock()
		t.Fatalf("invalidation retained admissions: %#v", admissions)
	}
	mu.Unlock()
}

func TestPersistentControlPeerSerializesConcurrentAcknowledgements(t *testing.T) {
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		if request.Op == "profile" {
			_, _ = listener.Write(profileACK(request))
			return
		}
		// The peer receives two concurrent callers; an ACK is deliberately
		// delayed so an unsynchronized shared reader could consume the wrong one.
		time.Sleep(10 * time.Millisecond)
		_, _ = listener.Write(admissionACK(request, "armed"))
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	var wait sync.WaitGroup
	errorsByCall := make(chan error, 2)
	for index := 0; index < 2; index++ {
		nonce, err := newDirectExecAdmissionNonce()
		if err != nil {
			t.Fatal(err)
		}
		wait.Add(1)
		go func(nonce string) {
			defer wait.Done()
			errorsByCall <- controlDirectExecAdmission(context.Background(), "arm", directExecAdmission{
				containerID: containerID, generation: session.generation, mode: boundaryLaunchMode, nonce: nonce, session: session,
			})
		}(nonce)
	}
	wait.Wait()
	close(errorsByCall)
	for err := range errorsByCall {
		if err != nil {
			t.Fatalf("concurrent control exchange failed: %v", err)
		}
	}
}

func TestWaitingNormalOperationRechecksPoisonedSessionBeforeControlWrite(t *testing.T) {
	var mu sync.Mutex
	var operations []string
	firstArmReceived := make(chan struct{})
	var firstArm sync.Once
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		mu.Lock()
		operations = append(operations, request.Op)
		mu.Unlock()
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			firstArm.Do(func() {
				close(firstArmReceived)
				// This structurally valid but uncorrelated ACK is an actual control
				// exchange ambiguity; production validation must poison before the
				// ordered peer mutex is released.
				_, _ = listener.Write([]byte(`{"op":"ack"}`))
			})
		case "invalidate":
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	attachTestFailStop(session, func(context.Context) error {
		t.Fatal("acknowledged INVALIDATE unexpectedly fail-stopped helper")
		return nil
	})
	firstNonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		t.Fatal(err)
	}
	secondNonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		t.Fatal(err)
	}
	firstAdmission := directExecAdmission{containerID: containerID, generation: session.generation, mode: boundaryLaunchMode, nonce: firstNonce, session: session}
	secondAdmission := directExecAdmission{containerID: containerID, generation: session.generation, mode: boundaryLaunchMode, nonce: secondNonce, session: session}

	// Operation B pauses after acquiring the actual ordered peer mutex.
	// Operation A then attempts that mutex and observes a real contention before
	// it can enter the protected exchange.
	firstOwnsPeerLock := make(chan struct{})
	releaseFirstExchange := make(chan struct{})
	secondContendedPeerLock := make(chan struct{})
	releaseSecondIntoPeerLock := make(chan struct{})
	poisonedBeforePeerUnlock := make(chan struct{})
	releaseAmbiguousExchange := make(chan struct{})
	var peerAcquisitions atomic.Int32
	session.peer.normalPeerLockAcquired = func() {
		if peerAcquisitions.Add(1) == 1 {
			close(firstOwnsPeerLock)
			<-releaseFirstExchange
		}
	}
	session.peer.normalPeerLockContended = func() {
		close(secondContendedPeerLock)
		<-releaseSecondIntoPeerLock
	}
	session.mu.Lock()
	session.normalAfterAmbiguityPoisonBeforePeerUnlock = func() {
		close(poisonedBeforePeerUnlock)
		<-releaseAmbiguousExchange
	}
	session.mu.Unlock()
	firstResult := make(chan error, 1)
	go func() { firstResult <- controlDirectExecAdmission(context.Background(), "arm", firstAdmission) }()
	<-firstOwnsPeerLock
	secondResult := make(chan error, 1)
	go func() { secondResult <- controlDirectExecAdmission(context.Background(), "arm", secondAdmission) }()
	<-secondContendedPeerLock
	close(releaseSecondIntoPeerLock)
	close(releaseFirstExchange)
	<-firstArmReceived
	<-poisonedBeforePeerUnlock
	close(releaseAmbiguousExchange)

	if err := <-firstResult; err == nil {
		t.Fatal("ambiguous ARM unexpectedly succeeded")
	}
	if err := <-secondResult; err == nil {
		t.Fatal("waiting ARM sent after production ambiguity")
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(operations, ",") != "profile,arm,invalidate" {
		t.Fatalf("control operations = %v; waiting ARM was emitted or teardown was skipped", operations)
	}
	if activeObserverSession(containerID) != nil {
		t.Fatal("invalidated session remained usable")
	}
}

func TestWaitingArmRejectsLifecycleSemanticAmbiguityBeforePeerUnlock(t *testing.T) {
	var mu sync.Mutex
	var operations []string
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		mu.Lock()
		operations = append(operations, request.Op)
		mu.Unlock()
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			_, _ = listener.Write(admissionACK(request, "armed"))
		case "status":
			_, _ = listener.Write(admissionACK(request, "pending"))
		case "cancel":
			_, _ = listener.Write(admissionACK(request, "cancelled"))
		case "invalidate":
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	attachTestFailStop(session, func(context.Context) error {
		t.Fatal("acknowledged INVALIDATE unexpectedly fail-stopped helper")
		return nil
	})
	semanticNonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		t.Fatal(err)
	}
	waitingNonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		t.Fatal(err)
	}
	semanticAdmission := directExecAdmission{containerID: containerID, generation: session.generation, mode: boundaryLaunchMode, nonce: semanticNonce, session: session}
	waitingAdmission := directExecAdmission{containerID: containerID, generation: session.generation, mode: boundaryLaunchMode, nonce: waitingNonce, session: session}
	if err := controlDirectExecAdmission(context.Background(), "arm", semanticAdmission); err != nil {
		t.Fatalf("arm semantic admission: %v", err)
	}

	// STATUS is the first normal exchange. Pause the second (CANCEL) only after
	// it owns the real peer mutex, then prove ARM's TryLock observed contention.
	cancelOwnsPeerLock := make(chan struct{})
	releaseCancelExchange := make(chan struct{})
	armContendedPeerLock := make(chan struct{})
	releaseArmIntoPeerLock := make(chan struct{})
	poisonedBeforePeerUnlock := make(chan struct{})
	releaseAmbiguousExchange := make(chan struct{})
	var peerAcquisitions atomic.Int32
	session.peer.normalPeerLockAcquired = func() {
		if peerAcquisitions.Add(1) == 2 {
			close(cancelOwnsPeerLock)
			<-releaseCancelExchange
		}
	}
	session.peer.normalPeerLockContended = func() {
		close(armContendedPeerLock)
		<-releaseArmIntoPeerLock
	}
	session.mu.Lock()
	session.normalAfterAmbiguityPoisonBeforePeerUnlock = func() {
		close(poisonedBeforePeerUnlock)
		<-releaseAmbiguousExchange
	}
	session.mu.Unlock()

	semanticResult := make(chan error, 1)
	go func() { semanticResult <- finalizeDirectExecAdmission(context.Background(), semanticAdmission, true) }()
	<-cancelOwnsPeerLock
	armResult := make(chan error, 1)
	go func() { armResult <- controlDirectExecAdmission(context.Background(), "arm", waitingAdmission) }()
	<-armContendedPeerLock
	close(releaseArmIntoPeerLock)
	close(releaseCancelExchange)
	<-poisonedBeforePeerUnlock
	close(releaseAmbiguousExchange)

	if err := <-semanticResult; err == nil {
		t.Fatal("successful runner with cancelled admission did not fail closed")
	}
	if err := <-armResult; err == nil {
		t.Fatal("waiting ARM sent after lifecycle-semantic ambiguity")
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(operations, ",") != "profile,arm,status,cancel,invalidate" {
		t.Fatalf("control operations = %v; waiting ARM was emitted or teardown was skipped", operations)
	}
	if activeObserverSession(containerID) != nil {
		t.Fatal("invalidated session remained usable")
	}
}

func TestAmbiguousArmInvalidatesSessionAndPreventsLaunch(t *testing.T) {
	withShortAdmissionTimeout(t)
	var mu sync.Mutex
	pending := make(map[string]bool)
	invalidations := 0
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		mu.Lock()
		defer mu.Unlock()
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			pending[request.Generation+request.Nonce] = true // accepted, ACK deliberately lost
		case "invalidate":
			for key := range pending {
				if strings.HasPrefix(key, request.Generation) {
					delete(pending, key)
				}
			}
			invalidations++
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	attachTestFailStop(session, func(context.Context) error {
		t.Fatal("acknowledged invalidation unexpectedly fail-stopped helper")
		return nil
	})
	runner := &admissionRunner{}
	if _, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/true")...); err == nil {
		t.Fatal("lost ARM ACK launched direct exec")
	}
	if len(runner.calls) != 0 || activeObserverSession(containerID) != nil {
		t.Fatalf("ambiguous ARM retained authority: calls=%#v session=%#v", runner.calls, activeObserverSession(containerID))
	}
	mu.Lock()
	defer mu.Unlock()
	if invalidations != 1 || len(pending) != 0 {
		t.Fatalf("invalidation = %d, pending = %#v", invalidations, pending)
	}
}

type testControlledProcess struct {
	cmd      *exec.Cmd
	listener io.Closer
	done     chan struct{}
	err      error
	once     sync.Once
}

func (p *testControlledProcess) Done() <-chan struct{} { return p.done }
func (p *testControlledProcess) ExitError() error      { return p.err }
func (p *testControlledProcess) Stop(ctx context.Context) error {
	p.once.Do(func() {
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
		if p.listener != nil {
			_ = p.listener.Close()
		}
	})
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestInvalidateAckLossFailStopsExactSessionOwner(t *testing.T) {
	withShortAdmissionTimeout(t)
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		if request.Op == "profile" {
			_, _ = listener.Write(profileACK(request))
		}
		// ARM is accepted without an ACK and INVALIDATE is also silent.
	})

	paths := supervisorPaths(t)
	var proc *testControlledProcess
	launcher := func(_ context.Context, remote, _ string) (ObserverProcess, error) {
		listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: remote, Net: "unixgram"})
		if err != nil {
			return nil, err
		}
		_ = os.Chmod(remote, 0o700)
		cmd := exec.Command("sleep", "60")
		if err := cmd.Start(); err != nil {
			_ = listener.Close()
			return nil, err
		}
		proc = &testControlledProcess{
			cmd:      cmd,
			listener: listener,
			done:     make(chan struct{}),
		}
		go func() {
			proc.err = cmd.Wait()
			_ = listener.Close()
			close(proc.done)
		}()
		return proc, nil
	}

	supervisor, err := newObserverSupervisor(context.Background(), launcher, paths.remote, paths.output, paths.lock)
	if err != nil {
		t.Fatalf("newObserverSupervisor() error = %v", err)
	}
	t.Cleanup(func() { _ = supervisor.Close() })

	const containerID = "0123456789abcdef"
	const otherContainerID = "fedcba9876543210"

	if _, err := startTrace(context.Background(), supervisor.Observer(), containerID, "pypi-wheel"); err != nil {
		t.Fatalf("startTrace(containerID) error = %v", err)
	}
	if _, err := startTrace(context.Background(), supervisor.Observer(), otherContainerID, "pypi-wheel"); err != nil {
		t.Fatalf("startTrace(otherContainerID) error = %v", err)
	}

	session := activeObserverSession(containerID)
	if session == nil {
		t.Fatal("session not active after StartProfile")
	}
	otherSession := activeObserverSession(otherContainerID)
	if otherSession == nil {
		t.Fatal("other session not active after StartProfile")
	}

	runner := &admissionRunner{}
	if _, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/true")...); err == nil {
		t.Fatal("ambiguous invalidation launched direct exec")
	}

	if len(runner.calls) != 0 {
		t.Fatalf("runner was called: %#v", runner.calls)
	}

	// Verify exact helper process was killed by fail-stop
	select {
	case <-proc.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("exact helper process was not terminated by fail-stop")
	}

	// Verify all sessions were forgotten and poisoned
	if activeObserverSession(containerID) != nil {
		t.Fatalf("session %q remained active", containerID)
	}
	if activeObserverSession(otherContainerID) != nil {
		t.Fatalf("session %q remained active", otherContainerID)
	}
	session.mu.Lock()
	sessionPoisoned := session.poisoned
	session.mu.Unlock()
	if !sessionPoisoned {
		t.Fatal("targeted session was not poisoned")
	}
	otherSession.mu.Lock()
	otherPoisoned := otherSession.poisoned
	otherSession.mu.Unlock()
	if !otherPoisoned {
		t.Fatal("peer session was not poisoned by supervisor fail-stop")
	}

	// Verify supervisor observer rejects new traces and profiles
	if _, err := supervisor.Observer().Start(context.Background(), "2222333344445555"); err == nil {
		t.Fatal("supervisor observer accepted new trace after fail-stop")
	}
	if _, err := startTrace(context.Background(), supervisor.Observer(), "2222333344445555", "pypi-wheel"); err == nil {
		t.Fatal("supervisor observer accepted new profile after fail-stop")
	}
}

func TestAmbiguousCancelPoisonsSessionAndInvalidatesIt(t *testing.T) {
	withShortAdmissionTimeout(t)
	invalidations := 0
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			_, _ = listener.Write(admissionACK(request, "armed"))
		case "invalidate":
			invalidations++
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	nonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		t.Fatal(err)
	}
	admission := directExecAdmission{containerID: containerID, generation: session.generation, mode: boundaryLaunchMode, nonce: nonce, session: session}
	if err := controlDirectExecAdmission(context.Background(), "arm", admission); err != nil {
		t.Fatalf("arm: %v", err)
	}
	if err := controlDirectExecAdmission(context.Background(), "cancel", admission); err == nil {
		t.Fatal("lost CANCEL ACK left session usable")
	}
	if invalidations != 1 || activeObserverSession(containerID) != nil {
		t.Fatalf("cancel ambiguity did not invalidate session: invalidations=%d session=%#v", invalidations, activeObserverSession(containerID))
	}
}

func TestReadinessAuthorityAmbiguityDoesNotRetryPoisonedSession(t *testing.T) {
	withShortAdmissionTimeout(t)
	arms := 0
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			arms++ // accepted, but ACK deliberately lost
		case "invalidate":
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	attachTestFailStop(session, func(context.Context) error {
		t.Fatal("acknowledged invalidation unexpectedly fail-stopped helper")
		return nil
	})
	runner := &admissionRunner{}
	if err := awaitBoundaryHelper(context.Background(), admissionAwareRunner(runner), containerID); err == nil {
		t.Fatal("readiness accepted ambiguous authority")
	}
	if arms != 1 || len(runner.calls) != 0 || activeObserverSession(containerID) != nil {
		t.Fatalf("readiness retried poisoned session: arms=%d calls=%#v session=%#v", arms, runner.calls, activeObserverSession(containerID))
	}
}

func TestReadinessRetriesOrdinaryFailureWithFreshAdmissionNonce(t *testing.T) {
	var nonces []string
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			nonces = append(nonces, request.Nonce)
			_, _ = listener.Write(admissionACK(request, "armed"))
		case "cancel":
			_, _ = listener.Write(admissionACK(request, "cancelled"))
		case "status":
			// The first readiness probe never reached the observer; the second
			// reached its root and can complete normally.
			status := "pending"
			if len(nonces) == 2 && request.Nonce == nonces[1] {
				status = "consumed"
			}
			_, _ = listener.Write(admissionACK(request, status))
		case "complete":
			_, _ = listener.Write(admissionACK(request, "completed"))
		}
	})
	const containerID = "0123456789abcdef"
	registerAdmissionSession(t, containerID)
	runner := &sequenceAdmissionRunner{errors: []error{errors.New("not ready"), nil}}
	if err := awaitBoundaryHelper(context.Background(), admissionAwareRunner(runner), containerID); err != nil {
		t.Fatalf("ordinary readiness retry: %v", err)
	}
	if len(runner.calls) != 2 || len(nonces) != 2 || nonces[0] == nonces[1] {
		t.Fatalf("readiness admissions = %#v, calls=%#v", nonces, runner.calls)
	}
}

func TestAdmissionRequiresExactPositiveAckBeforeDockerExec(t *testing.T) {
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		if request.Op == "profile" {
			_, _ = listener.Write(profileACK(request))
			return
		}
		status := map[string]string{"arm": "armed", "status": "consumed", "complete": "completed"}[request.Op]
		_, _ = listener.Write(admissionACK(request, status))
	})
	registerAdmissionSession(t, "0123456789abcdef")
	runner := &admissionRunner{}
	wrapped := admissionAwareRunner(runner)
	arguments := boundaryExecArguments("0123456789abcdef", boundaryLaunchMode, "python", "-I")
	if _, err := wrapped.Output(context.Background(), "docker", arguments...); err != nil {
		t.Fatalf("wrapped Output() error = %v", err)
	}
	if len(runner.calls) != 1 || len(runner.calls[0]) != len(arguments)+1 ||
		!validDirectExecAdmissionNonce(runner.calls[0][6]) || runner.calls[0][7] != "python" {
		t.Fatalf("admission-wrapped command = %#v", runner.calls)
	}
}

func TestAdmissionAckFailurePreventsDockerExecAndCancellationIsExact(t *testing.T) {
	requests := make(chan admissionRequest, 3)
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		if request.Op == "profile" {
			_, _ = listener.Write(profileACK(request))
			return
		}
		requests <- request
		if request.Op == "arm" {
			_, _ = listener.Write(admissionACK(request, "armed"))
			return
		}
		if request.Op == "status" {
			_, _ = listener.Write(admissionACK(request, "pending"))
			return
		}
		_, _ = listener.Write(admissionACK(request, "cancelled"))
	})
	session := registerAdmissionSession(t, "0123456789abcdef")
	runner := &admissionRunner{err: errors.New("launch failed")}
	wrapped := admissionAwareRunner(runner)
	arguments := boundaryExecArguments("0123456789abcdef", boundaryLaunchMode, "python")
	if _, err := wrapped.Output(context.Background(), "docker", arguments...); err == nil {
		t.Fatal("wrapped Output() succeeded after launch failure")
	}
	first := <-requests
	second := <-requests
	third := <-requests
	if first.Op != "arm" || second.Op != "status" || third.Op != "cancel" || first.Nonce != second.Nonce || first.Nonce != third.Nonce ||
		len(runner.calls) != 1 {
		t.Fatalf("admission lifecycle = %#v, %#v, %#v; calls=%#v", first, second, third, runner.calls)
	}

	// A write without a correlated ACK must not invoke Docker.
	observerControlEndpoint = filepath.Join(t.TempDir(), "no-ack.sock")
	forgetObserverSession(session)
	listener, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: observerControlEndpoint, Net: "unixpacket"})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(observerControlEndpoint, 0o700)
	defer listener.Close()
	go func() {
		for {
			connection, acceptErr := listener.AcceptUnix()
			if acceptErr != nil {
				return
			}
			go func(connection *net.UnixConn) {
				defer connection.Close()
				buffer := make([]byte, maximumControlDatagramBytes)
				for {
					size, readErr := connection.Read(buffer)
					if readErr != nil {
						return
					}
					var request admissionRequest
					if json.Unmarshal(buffer[:size], &request) == nil && request.Op == "profile" {
						_, _ = connection.Write(profileACK(request))
					}
				}
			}(connection)
		}
	}()
	registerAdmissionSession(t, "0123456789abcdef")
	noAckRunner := &admissionRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := admissionAwareRunner(noAckRunner).Output(ctx, "docker", arguments...); err == nil || len(noAckRunner.calls) != 0 {
		t.Fatalf("missing ACK invoked docker: err=%v calls=%#v", err, noAckRunner.calls)
	}
}

func TestConsumedAdmissionPreservesOriginalTargetFailureAndSession(t *testing.T) {
	const containerID = "0123456789abcdef"
	obs := startRealObserverForTest(t, containerID)
	session := registerAdmissionSession(t, containerID)
	targetErr := errors.New("target exited 17")
	runner := &testRealExecRunner{
		onExec: func(nonce string) {
			obs.sendSentryExec(t, containerID, boundaryLaunchMode, nonce)
		},
		err: targetErr,
	}
	_, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/false")...)
	if !errors.Is(err, targetErr) {
		t.Fatalf("target error was not preserved: %v", err)
	}
	if activeObserverSession(containerID) != session {
		t.Fatal("normal consumed target failure poisoned session")
	}

	// Verify a second command on the same session succeeds.
	successRunner := &testRealExecRunner{
		onExec: func(nonce string) {
			obs.sendSentryExec(t, containerID, boundaryLaunchMode, nonce)
		},
	}
	if _, err := admissionAwareRunner(successRunner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/true")...); err != nil {
		t.Fatalf("second command on session failed: %v", err)
	}
	if activeObserverSession(containerID) != session {
		t.Fatal("session became inactive after second command")
	}
}

func TestCancelConsumptionRaceCompletesAndPreservesRunnerFailure(t *testing.T) {
	const containerID = "0123456789abcdef"
	obs := startRealObserverForTest(t, containerID)
	session := registerAdmissionSession(t, containerID)

	targetErr := errors.New("target exited 17")
	var capturedNonce string
	runner := &testRealExecRunner{
		onExec: func(nonce string) {
			capturedNonce = nonce
		},
		err: targetErr,
	}

	callCount := 0
	session.mu.Lock()
	session.normalBeforePeerLock = func() {
		callCount++
		// Call 1: ARM
		// Call 2: STATUS (returns "pending")
		// Call 3: CANCEL (transition to consumed before CANCEL lock is acquired)
		if callCount == 3 {
			obs.sendSentryExec(t, containerID, boundaryLaunchMode, capturedNonce)
		}
	}
	session.mu.Unlock()

	_, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/false")...)
	if !errors.Is(err, targetErr) {
		t.Fatalf("error was not preserved: %v", err)
	}
	if activeObserverSession(containerID) != session {
		t.Fatal("session became inactive after cancel-consumption race")
	}
	if callCount != 4 {
		t.Fatalf("expected 4 control calls (arm, status, cancel, complete), got %d", callCount)
	}

	// Disarm hook and verify subsequent execution succeeds on the same session
	session.mu.Lock()
	session.normalBeforePeerLock = nil
	session.mu.Unlock()

	successRunner := &testRealExecRunner{
		onExec: func(nonce string) {
			obs.sendSentryExec(t, containerID, boundaryLaunchMode, nonce)
		},
	}
	if _, err := admissionAwareRunner(successRunner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/true")...); err != nil {
		t.Fatalf("subsequent command on session failed: %v", err)
	}
	if activeObserverSession(containerID) != session {
		t.Fatal("session became inactive after subsequent command")
	}
}

func TestAmbiguousStatusPoisonsSession(t *testing.T) {
	withShortAdmissionTimeout(t)
	invalidations := 0
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			_, _ = listener.Write(admissionACK(request, "armed"))
		case "invalidate":
			invalidations++
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	runner := &admissionRunner{err: errors.New("launch failed")}
	if _, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/false")...); err == nil {
		t.Fatal("lost STATUS ACK did not fail closed")
	}
	if invalidations != 1 || activeObserverSession(containerID) != nil || session.poisoned == false {
		t.Fatalf("invalidations=%d session=%#v poisoned=%t", invalidations, activeObserverSession(containerID), session.poisoned)
	}
}

func TestAmbiguousCompletePoisonsSession(t *testing.T) {
	withShortAdmissionTimeout(t)
	invalidations := 0
	withAdmissionControl(t, func(listener *net.UnixConn, request admissionRequest) {
		switch request.Op {
		case "profile":
			_, _ = listener.Write(profileACK(request))
		case "arm":
			_, _ = listener.Write(admissionACK(request, "armed"))
		case "status":
			_, _ = listener.Write(admissionACK(request, "consumed"))
		case "invalidate":
			invalidations++
			response, _ := json.Marshal(map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": request.ContainerID, "session_generation": request.Generation})
			_, _ = listener.Write(response)
		}
	})
	const containerID = "0123456789abcdef"
	session := registerAdmissionSession(t, containerID)
	runner := &admissionRunner{}
	if _, err := admissionAwareRunner(runner).Output(context.Background(), "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/true")...); err == nil {
		t.Fatal("lost COMPLETE ACK did not fail closed")
	}
	if invalidations != 1 || activeObserverSession(containerID) != nil || !session.poisoned {
		t.Fatalf("invalidations=%d session=%#v poisoned=%t", invalidations, activeObserverSession(containerID), session.poisoned)
	}
}
