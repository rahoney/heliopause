package sandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

const (
	directExecAdmissionNonceBytes = 32
	directExecAdmissionNonceChars = directExecAdmissionNonceBytes * 2
	maximumControlDatagramBytes   = 4096
)

var directExecAdmissionTimeout = 2 * time.Second

var errObserverAuthorityAmbiguous = errors.New("observer direct-exec authority is ambiguous")

// sessionEntropy is kept behind this narrow seam solely to prove that an
// entropy failure fails closed. Production always uses crypto/rand.
var sessionEntropy = rand.Read

type directExecAdmissionRequired interface{ RequiresDirectExecAdmission() }

type admissionAwareCommandRunner struct{ runner CommandRunner }

func admissionAwareRunner(runner CommandRunner) CommandRunner {
	if runner == nil {
		return nil
	}
	if _, ok := runner.(*admissionAwareCommandRunner); ok {
		return runner
	}
	if _, ok := runner.(directExecAdmissionRequired); !ok {
		return runner
	}
	return &admissionAwareCommandRunner{runner: runner}
}

func (r *admissionAwareCommandRunner) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	prepared, finish, err := prepareDirectExecAdmission(ctx, binary, arguments)
	if err != nil {
		return nil, err
	}
	output, commandErr := r.runner.Output(ctx, binary, prepared...)
	if finish != nil {
		if finishErr := finish(ctx, commandErr == nil); finishErr != nil {
			return output, finishErr
		}
	}
	return output, commandErr
}

func (r *admissionAwareCommandRunner) RunInput(ctx context.Context, input io.Reader, binary string, arguments ...string) error {
	runner, ok := r.runner.(inputCommandRunner)
	if !ok {
		return errors.New("sandbox input stream runner is unavailable")
	}
	prepared, finish, err := prepareDirectExecAdmission(ctx, binary, arguments)
	if err != nil {
		return err
	}
	commandErr := runner.RunInput(ctx, input, binary, prepared...)
	if finish != nil {
		if finishErr := finish(ctx, commandErr == nil); finishErr != nil {
			return finishErr
		}
	}
	return commandErr
}

func (r *admissionAwareCommandRunner) RunDiscard(ctx context.Context, binary string, arguments ...string) error {
	runner, ok := r.runner.(discardCommandRunner)
	if !ok {
		return errors.New("sandbox discard runner is unavailable")
	}
	prepared, finish, err := prepareDirectExecAdmission(ctx, binary, arguments)
	if err != nil {
		return err
	}
	commandErr := runner.RunDiscard(ctx, binary, prepared...)
	if finish != nil {
		if finishErr := finish(ctx, commandErr == nil); finishErr != nil {
			return finishErr
		}
	}
	return commandErr
}

func (r *admissionAwareCommandRunner) RunBounded(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	runner, ok := r.runner.(boundedCommandRunner)
	if !ok {
		return nil, errors.New("sandbox bounded runner is unavailable")
	}
	prepared, finish, err := prepareDirectExecAdmission(ctx, binary, arguments)
	if err != nil {
		return nil, err
	}
	output, commandErr := runner.RunBounded(ctx, binary, prepared...)
	if finish != nil {
		if finishErr := finish(ctx, commandErr == nil); finishErr != nil {
			return output, finishErr
		}
	}
	return output, commandErr
}

func (r *admissionAwareCommandRunner) RunOutput(ctx context.Context, output io.Writer, binary string, arguments ...string) error {
	runner, ok := r.runner.(interface {
		RunOutput(context.Context, io.Writer, string, ...string) error
	})
	if !ok {
		return errors.New("sandbox output stream runner is unavailable")
	}
	prepared, finish, err := prepareDirectExecAdmission(ctx, binary, arguments)
	if err != nil {
		return err
	}
	commandErr := runner.RunOutput(ctx, output, binary, prepared...)
	if finish != nil {
		if finishErr := finish(ctx, commandErr == nil); finishErr != nil {
			return finishErr
		}
	}
	return commandErr
}

type directExecAdmission struct {
	containerID, generation, mode, nonce string
	session                              *observerSecuritySession
}

func prepareDirectExecAdmission(ctx context.Context, binary string, arguments []string) ([]string, func(context.Context, bool) error, error) {
	containerID, mode, nonceIndex, ok := directExecOrigin(arguments, binary)
	if !ok {
		return arguments, nil, nil
	}
	session := activeObserverSession(containerID)
	if session == nil {
		return nil, nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	nonce, err := newDirectExecAdmissionNonce()
	if err != nil {
		return nil, nil, errors.New("generate direct-exec admission")
	}
	admission := directExecAdmission{containerID: containerID, generation: session.generation, mode: mode, nonce: nonce, session: session}
	if err := controlDirectExecAdmission(ctx, "arm", admission); err != nil {
		return nil, nil, err
	}
	prepared := make([]string, 0, len(arguments)+1)
	prepared = append(prepared, arguments[:nonceIndex]...)
	prepared = append(prepared, nonce)
	prepared = append(prepared, arguments[nonceIndex:]...)
	return prepared, func(releaseCtx context.Context, runnerSucceeded bool) error {
		return finalizeDirectExecAdmission(releaseCtx, admission, runnerSucceeded)
	}, nil
}

func directExecOrigin(arguments []string, binary string) (string, string, int, bool) {
	if binary != "docker" || len(arguments) < 7 || arguments[0] != "exec" {
		return "", "", 0, false
	}
	index := 1
	if index < len(arguments) && arguments[index] == "-i" {
		index++
	}
	if index+4 >= len(arguments) || arguments[index] != "--user" || arguments[index+1] != boundaryBootstrapUser || !containerIDPattern.MatchString(arguments[index+2]) || arguments[index+3] != boundaryHelperPath {
		return "", "", 0, false
	}
	switch arguments[index+4] {
	case boundaryOriginLaunchMode:
		return arguments[index+2], boundaryLaunchMode, index + 5, true
	case boundaryOriginPythonHandoffMode:
		return arguments[index+2], boundaryPythonHandoffMode, index + 5, true
	case boundaryOriginELFHandoffMode:
		return arguments[index+2], boundaryELFHandoffMode, index + 5, true
	default:
		return "", "", 0, false
	}
}

func newDirectExecAdmissionNonce() (string, error) { return newObserverSecurityGeneration() }

func newObserverSecurityGeneration() (string, error) {
	bytes := make([]byte, directExecAdmissionNonceBytes)
	count, err := sessionEntropy(bytes)
	if err != nil {
		return "", err
	}
	if count != len(bytes) {
		return "", errors.New("short cryptographic random read")
	}
	return hex.EncodeToString(bytes), nil
}

type controlRequest struct {
	Op        string `json:"op"`
	Container string `json:"container_id"`
	Profile   string `json:"profile,omitempty"`
	Topology  string `json:"expected_topology,omitempty"`
	Session   string `json:"session_generation"`
	Mode      string `json:"mode,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}
type strictControlAck map[string]string
type observerControlPeer struct {
	connection *net.UnixConn
	mu         sync.Mutex
	closeOnce  sync.Once
	// normalPeerLockContended and normalPeerLockAcquired are unexported test
	// synchronization seams. They are nil in production and only observe the
	// actual ordered-peer mutex acquisition path.
	normalPeerLockContended func()
	normalPeerLockAcquired  func()
}

func openObserverControlPeer(ctx context.Context, label string) (*observerControlPeer, error) {
	if ctx == nil || label == "" {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if _, err := verifyOwnedObserverSocket(observerControlEndpoint); err != nil {
		return nil, observerFault{reason: "HELPER_UNAVAILABLE"}
	}
	// unixpacket gives the observer one accepted server connection whose FD is
	// the session's authority-bearing peer identity.  Unlike unixgram sender
	// addresses, that identity cannot be replaced by another client pathname.
	connection, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: observerControlEndpoint, Net: "unixpacket"})
	if err != nil {
		return nil, observerFault{reason: "HELPER_UNAVAILABLE"}
	}
	return &observerControlPeer{connection: connection}, nil
}

func (p *observerControlPeer) Close() {
	if p != nil && p.connection != nil {
		p.closeOnce.Do(func() { _ = p.connection.Close() })
	}
}

func (p *observerControlPeer) exchange(ctx context.Context, request controlRequest, required map[string]string) error {
	acknowledgement, err := p.exchangeAck(ctx, request)
	if err != nil || !exactControlAck(acknowledgement, required) {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return nil
}

// exchangeAck owns the one request/one response turn on the persistent
// SOCK_SEQPACKET peer. The mutex makes a response unambiguously belong to the
// caller's request, while callers retain the ACK to validate operation-specific
// terminal state.
func (p *observerControlPeer) exchangeAck(ctx context.Context, request controlRequest) (strictControlAck, error) {
	return p.exchangeAckAuthorized(ctx, request, nil, nil)
}

// exchangeAckAuthorized is the normal-authority variant of exchangeAck.  It
// keeps the ordered peer lock through the complete authoritative exchange.
// An ambiguous result poisons the session before that peer lock is released,
// so a normal operation already waiting for the next turn cannot send with
// stale authority. Teardown never takes the peer lock while holding the
// session lock.
func (p *observerControlPeer) exchangeAckAuthorized(ctx context.Context, request controlRequest, session *observerSecuritySession, validate func(strictControlAck) bool) (strictControlAck, error) {
	if p == nil || p.connection == nil || ctx == nil {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	body, err := json.Marshal(request)
	if err != nil || len(body) == 0 || len(body) > maximumControlDatagramBytes {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if session != nil && p.normalPeerLockContended != nil {
		if !p.mu.TryLock() {
			p.normalPeerLockContended()
			p.mu.Lock()
		}
	} else {
		p.mu.Lock()
	}
	defer p.mu.Unlock()
	if session != nil && p.normalPeerLockAcquired != nil {
		p.normalPeerLockAcquired()
	}
	poisonAmbiguous := func() {
		if session == nil {
			return
		}
		session.mu.Lock()
		newlyPoisoned := !session.poisoned
		session.poisoned = true
		hook := session.normalAfterAmbiguityPoisonBeforePeerUnlock
		session.mu.Unlock()
		if newlyPoisoned && hook != nil {
			hook()
		}
	}
	deadline := time.Now().Add(directExecAdmissionTimeout)
	if fromContext, ok := ctx.Deadline(); ok && fromContext.Before(deadline) {
		deadline = fromContext
	}
	if err := p.connection.SetDeadline(deadline); err != nil {
		poisonAmbiguous()
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if session != nil {
		session.mu.Lock()
		usable := !session.poisoned && !session.invalidated &&
			session.containerID == request.Container && session.generation == request.Session
		if !usable {
			session.mu.Unlock()
			return nil, errNormalAuthorityUnavailable
		}
	}
	_, writeErr := p.connection.Write(body)
	if session != nil {
		session.mu.Unlock()
	}
	if writeErr != nil {
		poisonAmbiguous()
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	response := make([]byte, maximumControlDatagramBytes)
	size, _, flags, _, err := p.connection.ReadMsgUnix(response, nil)
	if err != nil || size <= 0 || flags&syscall.MSG_TRUNC != 0 {
		poisonAmbiguous()
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	acknowledgement, err := decodeStrictControlAck(response[:size])
	if err != nil {
		poisonAmbiguous()
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if validate != nil && !validate(acknowledgement) {
		poisonAmbiguous()
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return acknowledgement, nil
}

func exactControlAck(acknowledgement strictControlAck, required map[string]string) bool {
	if len(acknowledgement) != len(required) {
		return false
	}
	for field, expected := range required {
		if acknowledgement[field] != expected {
			return false
		}
	}
	return true
}

// decodeStrictControlAck accepts only the small scalar-string ACK object. The
// token walk detects duplicate keys and the final Token call rejects trailing JSON.
func decodeStrictControlAck(payload []byte) (strictControlAck, error) {
	if len(payload) == 0 || len(payload) > maximumControlDatagramBytes {
		return nil, errors.New("control acknowledgement size")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return nil, errors.New("control acknowledgement is not an object")
	}
	allowed := map[string]bool{"op": true, "ack_op": true, "status": true, "container_id": true, "profile": true, "expected_topology": true, "session_generation": true, "mode": true, "nonce": true}
	values := make(strictControlAck)
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || !allowed[key] {
			return nil, errors.New("invalid control acknowledgement field")
		}
		if _, duplicate := values[key]; duplicate {
			return nil, errors.New("duplicate control acknowledgement field")
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil || len(raw) < 2 || raw[0] != '"' {
			return nil, errors.New("invalid control acknowledgement value")
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("invalid control acknowledgement string")
		}
		values[key] = value
	}
	last, err := decoder.Token()
	if err != nil || last != json.Delim('}') {
		return nil, errors.New("unterminated control acknowledgement")
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing control acknowledgement JSON")
	}
	return values, nil
}

type observerSecuritySession struct {
	containerID, generation, profile, topology string
	peer                                       *observerControlPeer
	mu                                         sync.Mutex
	poisoned                                   bool
	invalidated                                bool
	failStop                                   func(context.Context) error
	// normalBeforePeerLock is an unexported test synchronization seam. It is
	// called only after the ordinary fast-path validation and cannot alter the
	// authorization decision or the request bytes.
	normalBeforePeerLock func()
	// normalAfterAmbiguityPoisonBeforePeerUnlock is an unexported test
	// synchronization seam. Production leaves it nil; it can delay only after
	// production has made poisoning authoritative and while the peer lock is
	// still held.
	normalAfterAmbiguityPoisonBeforePeerUnlock func()
}

var observerSessions = struct {
	sync.Mutex
	byContainer map[string]*observerSecuritySession
}{byContainer: make(map[string]*observerSecuritySession)}

func activeObserverSession(containerID string) *observerSecuritySession {
	observerSessions.Lock()
	session := observerSessions.byContainer[containerID]
	observerSessions.Unlock()
	if session == nil {
		return nil
	}
	session.mu.Lock()
	valid := !session.poisoned && !session.invalidated
	session.mu.Unlock()
	if !valid {
		return nil
	}
	return session
}
func activateObserverSession(session *observerSecuritySession) bool {
	observerSessions.Lock()
	defer observerSessions.Unlock()
	if _, exists := observerSessions.byContainer[session.containerID]; exists {
		return false
	}
	observerSessions.byContainer[session.containerID] = session
	return true
}
func forgetObserverSession(session *observerSecuritySession) {
	if session == nil {
		return
	}
	session.mu.Lock()
	session.poisoned = true
	session.invalidated = true
	session.mu.Unlock()
	observerSessions.Lock()
	if observerSessions.byContainer[session.containerID] == session {
		delete(observerSessions.byContainer, session.containerID)
	}
	observerSessions.Unlock()
	session.peer.Close()
}

func registerObserverProfile(ctx context.Context, containerID, profile string) (*observerSecuritySession, error) {
	topology, ok := observerExpectedTopology(profile)
	encodedTopology, encoded := encodeExpectedTopology(topology)
	if ctx == nil || !containerIDPattern.MatchString(containerID) || !validObserverProfile(profile) || !ok || !encoded || activeObserverSession(containerID) != nil {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	generation, err := newObserverSecurityGeneration()
	if err != nil {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	peer, err := openObserverControlPeer(ctx, "profile")
	if err != nil {
		return nil, err
	}
	session := &observerSecuritySession{containerID: containerID, generation: generation, profile: profile, topology: encodedTopology, peer: peer}
	required := map[string]string{"op": "ack", "ack_op": "profile", "status": "registered", "container_id": containerID, "profile": profile, "expected_topology": encodedTopology, "session_generation": generation}
	if err := peer.exchange(ctx, controlRequest{Op: "profile", Container: containerID, Profile: profile, Topology: encodedTopology, Session: generation}, required); err != nil {
		peer.Close()
		return nil, err
	}
	if !activateObserverSession(session) {
		peer.Close()
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return session, nil
}

func invalidateObserverSession(ctx context.Context, session *observerSecuritySession) error {
	if session == nil || ctx == nil {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	session.mu.Lock()
	invalidated := session.invalidated
	if !invalidated {
		session.poisoned = true
	}
	session.mu.Unlock()
	if invalidated {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return invalidatePoisonedObserverSession(ctx, session)
}

func invalidatePoisonedObserverSession(ctx context.Context, session *observerSecuritySession) error {
	if session == nil || ctx == nil {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	required := map[string]string{"op": "ack", "ack_op": "invalidate", "status": "invalidated", "container_id": session.containerID, "session_generation": session.generation}
	if err := session.peer.exchange(ctx, controlRequest{Op: "invalidate", Container: session.containerID, Session: session.generation}, required); err != nil {
		return failStopAmbiguousObserverSession(session, err)
	}
	forgetObserverSession(session)
	return nil
}

// failStopAmbiguousObserverSession is the terminal authority resolution path:
// an exact INVALIDATE ACK proves cleanup; otherwise only termination of the
// exact helper that owns the in-memory admission state can prove it.
func failStopAmbiguousObserverSession(session *observerSecuritySession, cause error) error {
	if session == nil {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	session.mu.Lock()
	session.poisoned = true
	stop := session.failStop
	session.mu.Unlock()
	if stop == nil {
		return errors.Join(cause, errObserverAuthorityAmbiguous, observerFault{reason: "LIFECYCLE_ERROR"})
	}
	stopContext, cancel := context.WithTimeout(context.Background(), supervisorStopTimeout)
	stopErr := stop(stopContext)
	cancel()
	if stopErr != nil {
		return errors.Join(cause, errObserverAuthorityAmbiguous, stopErr)
	}
	forgetObserverSession(session)
	return errors.Join(cause, errObserverAuthorityAmbiguous)
}

func controlDirectExecAdmission(ctx context.Context, operation string, admission directExecAdmission) error {
	if operation != "arm" && operation != "cancel" {
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
	_, err := exchangeDirectExecAdmission(ctx, operation, admission, nil)
	return err
}

// finalizeDirectExecAdmission resolves runner completion against the observer's
// retained, exact slot. A target's non-zero exit after PENDING->CONSUMED is not
// security ambiguity and therefore returns to the caller unchanged.
func finalizeDirectExecAdmission(ctx context.Context, admission directExecAdmission, runnerSucceeded bool) error {
	state, err := directExecAdmissionStatus(ctx, admission)
	if err != nil {
		return err
	}
	switch state {
	case "consumed":
		return completeDirectExecAdmission(ctx, admission)
	case "pending":
		state, err = cancelDirectExecAdmission(ctx, admission, runnerSucceeded)
		if err != nil {
			return err
		}
		if state == "consumed" {
			return completeDirectExecAdmission(ctx, admission)
		}
		// cancelDirectExecAdmission's locked classifier leaves only this
		// legitimate terminal state here.
		return nil
	default:
		return observerFault{reason: "LIFECYCLE_ERROR"}
	}
}

func directExecAdmissionStatus(ctx context.Context, admission directExecAdmission) (string, error) {
	acknowledgement, err := exchangeDirectExecAdmission(ctx, "status", admission, nil)
	if err != nil {
		return "", err
	}
	return acknowledgement["status"], nil
}

func cancelDirectExecAdmission(ctx context.Context, admission directExecAdmission, runnerSucceeded bool) (string, error) {
	acknowledgement, err := exchangeDirectExecAdmission(ctx, "cancel", admission, func(status string) bool {
		// A cancelled admission is valid only if the command itself failed. A
		// successful command without the matching SENTRY_EXEC is a lifecycle
		// ambiguity, so this classifier runs under the ordered peer lock.
		return !runnerSucceeded || status != "cancelled"
	})
	if err != nil {
		return "", err
	}
	return acknowledgement["status"], nil
}

func completeDirectExecAdmission(ctx context.Context, admission directExecAdmission) error {
	_, err := exchangeDirectExecAdmission(ctx, "complete", admission, nil)
	if err != nil {
		return err
	}
	return nil
}

func exchangeDirectExecAdmission(ctx context.Context, operation string, admission directExecAdmission, semanticStatusValid func(string) bool) (strictControlAck, error) {
	if ctx == nil || admission.session == nil || !containerIDPattern.MatchString(admission.containerID) || !validObserverSecurityGeneration(admission.generation) || !validBoundaryMode(admission.mode) || !validDirectExecAdmissionNonce(admission.nonce) {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	admission.session.mu.Lock()
	valid := !admission.session.poisoned && !admission.session.invalidated && admission.session.containerID == admission.containerID && admission.session.generation == admission.generation
	admission.session.mu.Unlock()
	if !valid || activeObserverSession(admission.containerID) != admission.session {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	// This is a diagnostic-only test seam.  The decision that authorizes the
	// request remains below, under the ordered peer lock.
	admission.session.mu.Lock()
	hook := admission.session.normalBeforePeerLock
	admission.session.mu.Unlock()
	if hook != nil {
		hook()
	}
	// The peer lock orders the request/response turn. Recheck session authority
	// while that lock is held and retain the session lock through Write: a poison
	// which became authoritative while this caller waited cannot permit a stale
	// ARM/STATUS/CANCEL/COMPLETE request onto the control transport.
	// INVALIDATE deliberately uses the unguarded exchange path, because teardown
	// must remain available after poisoning.
	request := controlRequest{Op: operation, Container: admission.containerID, Session: admission.generation, Mode: admission.mode, Nonce: admission.nonce}
	required := map[string]string{"op": "ack", "ack_op": operation, "container_id": admission.containerID, "session_generation": admission.generation, "mode": admission.mode, "nonce": admission.nonce}
	validate := func(acknowledgement strictControlAck) bool {
		if !exactControlAckExceptStatus(acknowledgement, required) {
			return false
		}
		switch operation {
		case "arm":
			return acknowledgement["status"] == "armed" && (semanticStatusValid == nil || semanticStatusValid(acknowledgement["status"]))
		case "status":
			return (acknowledgement["status"] == "pending" || acknowledgement["status"] == "consumed") &&
				(semanticStatusValid == nil || semanticStatusValid(acknowledgement["status"]))
		case "cancel":
			return (acknowledgement["status"] == "cancelled" || acknowledgement["status"] == "consumed") &&
				(semanticStatusValid == nil || semanticStatusValid(acknowledgement["status"]))
		case "complete":
			return acknowledgement["status"] == "completed" && (semanticStatusValid == nil || semanticStatusValid(acknowledgement["status"]))
		default:
			return false
		}
	}
	acknowledgement, err := admission.session.peer.exchangeAckAuthorized(ctx, request, admission.session, validate)
	if err != nil {
		if errors.Is(err, errNormalAuthorityUnavailable) {
			return nil, observerFault{reason: "LIFECYCLE_ERROR"}
		}
		return nil, ambiguousDirectExecAdmission(ctx, admission, err)
	}
	return acknowledgement, nil
}

var errNormalAuthorityUnavailable = errors.New("normal observer authority unavailable")

func exactControlAckExceptStatus(acknowledgement strictControlAck, required map[string]string) bool {
	if len(acknowledgement) != len(required)+1 || acknowledgement["status"] == "" {
		return false
	}
	for field, expected := range required {
		if acknowledgement[field] != expected {
			return false
		}
	}
	return true
}

func ambiguousDirectExecAdmission(ctx context.Context, admission directExecAdmission, cause error) error {
	if cleanupErr := invalidateObserverSession(ctx, admission.session); cleanupErr != nil {
		return errors.Join(cause, errObserverAuthorityAmbiguous, cleanupErr)
	}
	return errors.Join(cause, errObserverAuthorityAmbiguous)
}

func controlOperationStatus(operation string) string {
	switch operation {
	case "cancel":
		return "cancelled"
	case "arm":
		return "armed"
	default:
		return ""
	}
}

func validBoundaryMode(mode string) bool {
	return mode == boundaryLaunchMode || mode == boundaryPythonHandoffMode || mode == boundaryELFHandoffMode
}
func validObserverSecurityGeneration(generation string) bool {
	return validDirectExecAdmissionNonce(generation)
}
func validDirectExecAdmissionNonce(nonce string) bool {
	if len(nonce) != directExecAdmissionNonceChars {
		return false
	}
	for _, character := range nonce {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
