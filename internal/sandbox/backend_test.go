package sandbox

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestBackendExecutesOneShotSandboxWithConstrainedDockerCommand(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef\n"), nil, nil, nil, nil}}
	introducer := &recordingIntroducer{}
	backend := newTestBackend(t, runner, introducer, &emptyObserver{}, availableProbe)

	result, err := backend.Execute(context.Background(), sandboxRequest(t))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status() != domain.SandboxCompleted {
		t.Fatalf("Status() = %q, want COMPLETED", result.Status())
	}
	if introducer.calls != 1 || introducer.containerID != "0123456789abcdef" {
		t.Fatalf("Introduce calls = %d, container = %q", introducer.calls, introducer.containerID)
	}
	if len(runner.calls) != 5 {
		t.Fatalf("command calls = %d, want 5", len(runner.calls))
	}
	assertConstrainedCreateCommand(t, runner.calls[0].arguments)
	if got := runner.calls[1]; got.binary != "docker" || !sameStrings(got.arguments, []string{"start", "0123456789abcdef"}) {
		t.Fatalf("start command = %q %q", got.binary, got.arguments)
	}
	if got := runner.calls[2]; got.binary != "docker" || !sameStrings(got.arguments, boundaryReadinessArguments("0123456789abcdef")) {
		t.Fatalf("helper readiness command = %q %q", got.binary, got.arguments)
	}
	if got := runner.calls[3]; got.binary != "docker" || !sameStrings(got.arguments, boundaryExecArguments("0123456789abcdef", boundaryLaunchMode, "/bin/sh", "-ceu", npmLifecycleCommand)) {
		t.Fatalf("npm command = %q %q", got.binary, got.arguments)
	}
	if got := runner.calls[4]; got.binary != "docker" || !sameStrings(got.arguments, []string{"rm", "--force", "0123456789abcdef"}) {
		t.Fatalf("cleanup command = %q %q", got.binary, got.arguments)
	}
}

func TestNPMLifecycleCommandIsExactOfflineNonNotifyingInvocation(t *testing.T) {
	const want = "mkdir -p /tmp/package /tmp/.npm; cd /tmp/package; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm install --ignore-scripts=false --no-audit --no-fund --offline --no-update-notifier /tmp/artifact.tgz"
	if npmLifecycleCommand != want {
		t.Fatalf("npm lifecycle command = %q, want %q", npmLifecycleCommand, want)
	}
	for _, forbidden := range []string{"--online", "--prefer-online", "npm_config_registry=", "CI=true"} {
		if strings.Contains(npmLifecycleCommand, forbidden) {
			t.Fatalf("npm lifecycle command must not contain %q: %q", forbidden, npmLifecycleCommand)
		}
	}
}

func TestBackendDoesNotExecuteWhenCapabilityIsUnavailable(t *testing.T) {
	runner := &recordingRunner{}
	backend := newTestBackend(t, runner, &recordingIntroducer{}, &emptyObserver{}, func(context.Context) (Capability, error) {
		return Capability{LimitationCode: "M3_LINUX_ONLY"}, nil
	})
	result, err := backend.Execute(context.Background(), sandboxRequest(t))
	if err != nil || result.Status() != domain.SandboxIncomplete {
		t.Fatalf("Execute() = (%q, %v), want incomplete result", result.Status(), err)
	}
	if code, _ := result.LimitationCode(); code != "M3_LINUX_ONLY" {
		t.Fatalf("LimitationCode() = %q", code)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runtime commands = %d, want 0", len(runner.calls))
	}
}

func TestBackendTerminatesContainerBeforeCollectingFinalizedTrace(t *testing.T) {
	cleanup := make(chan struct{})
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}}
	runner.cleanupSignal = cleanup
	observer := &recordingObserver{reader: &cleanupGatedTraceReader{
		cleanup: cleanup,
		records: []TraceRecord{{Kind: "network-attempt", Bytes: 1}},
	}}
	backend := newTestBackend(t, runner, &recordingIntroducer{}, observer, availableProbe)

	result, err := backend.Execute(context.Background(), sandboxRequest(t))
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("Execute() = (%q, %v)", result.Status(), err)
	}
	if observer.containerID != "0123456789abcdef" {
		t.Fatalf("observer container ID = %q", observer.containerID)
	}
	observations := result.Observations()
	if len(observations) != 2 || observations[0].Category() != domain.ObservationNetwork {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestBackendObserverFailureIsIncompleteAndDisposesContainer(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil}}
	backend := newTestBackend(t, runner, &recordingIntroducer{}, &recordingObserver{err: errors.New("observer unavailable")}, availableProbe)

	result, err := backend.Execute(context.Background(), sandboxRequest(t))
	if err != nil || result.Status() != domain.SandboxIncomplete {
		t.Fatalf("Execute() = (%q, %v)", result.Status(), err)
	}
	if code, _ := result.LimitationCode(); code != "M3_DYNAMIC_OBSERVER_FAILED" {
		t.Fatalf("LimitationCode() = %q", code)
	}
	if len(runner.calls) != 2 || !sameStrings(runner.calls[1].arguments, []string{"rm", "--force", "0123456789abcdef"}) {
		t.Fatalf("container not disposed: %#v", runner.calls)
	}
}

func TestBackendProcessFailureAndCleanupFailureAreIncomplete(t *testing.T) {
	tests := []struct {
		name       string
		responses  [][]byte
		errors     []error
		limitation string
	}{
		{"process failure", [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}, []error{nil, nil, nil, errors.New("exit status 1")}, "M3_DYNAMIC_EXECUTION_FAILED"},
		{"cleanup failure", [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}, []error{nil, nil, nil, nil, errors.New("cleanup")}, "M3_DYNAMIC_CLEANUP_FAILED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend := newTestBackend(t, &recordingRunner{responses: test.responses, errors: test.errors}, &recordingIntroducer{}, &emptyObserver{}, availableProbe)
			result, err := backend.Execute(context.Background(), sandboxRequest(t))
			if err != nil || result.Status() != domain.SandboxIncomplete {
				t.Fatalf("Execute() = (%q, %v), want incomplete", result.Status(), err)
			}
			if code, _ := result.LimitationCode(); code != test.limitation {
				t.Fatalf("LimitationCode() = %q, want %q", code, test.limitation)
			}
		})
	}
}

func TestBackendTimeoutIsIncompleteAndStillDisposed(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}, waitForContext: true, waitForContextAt: 3}
	backend := newTestBackend(t, runner, &recordingIntroducer{}, &emptyObserver{}, availableProbe)
	backend.wallTimeout = time.Millisecond

	result, err := backend.Execute(context.Background(), sandboxRequest(t))
	if err != nil || result.Status() != domain.SandboxIncomplete {
		t.Fatalf("Execute() = (%q, %v), want incomplete", result.Status(), err)
	}
	if code, _ := result.LimitationCode(); code != "M3_DYNAMIC_TIMEOUT" {
		t.Fatalf("LimitationCode() = %q, want timeout", code)
	}
	if len(runner.calls) != 5 || !sameStrings(runner.calls[4].arguments, []string{"rm", "--force", "0123456789abcdef"}) {
		t.Fatalf("cleanup command missing after timeout: %#v", runner.calls)
	}
}

func TestBackendArtifactIntroductionBlockedUntilMountAnchorsReady(t *testing.T) {
	t.Run("blocked when observer fails topology reconciliation", func(t *testing.T) {
		runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil}}
		introducer := &recordingIntroducer{}
		observer := &recordingObserver{err: errors.New("topology reconciliation failed")}
		backend := newTestBackend(t, runner, introducer, observer, availableProbe)

		result, err := backend.Execute(context.Background(), sandboxRequest(t))
		if err != nil || result.Status() != domain.SandboxIncomplete {
			t.Fatalf("Execute() = (%q, %v), want Incomplete", result.Status(), err)
		}
		if introducer.calls != 0 {
			t.Fatalf("introducer called %d times, want 0 when mount anchors not ready", introducer.calls)
		}
	})

	t.Run("blocked when observer does not implement mountAnchorReadyObserver", func(t *testing.T) {
		runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil}}
		introducer := &recordingIntroducer{}
		observer := &plainObserver{}
		backend := newTestBackend(t, runner, introducer, observer, availableProbe)

		result, err := backend.Execute(context.Background(), sandboxRequest(t))
		if err != nil || result.Status() != domain.SandboxIncomplete {
			t.Fatalf("Execute() = (%q, %v), want Incomplete", result.Status(), err)
		}
		if introducer.calls != 0 {
			t.Fatalf("introducer called %d times, want 0 when observer lacks topology readiness", introducer.calls)
		}
	})

	t.Run("ordered strictly before artifact introduction", func(t *testing.T) {
		runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil, nil}}
		introducer := &sequencedIntroducer{}
		var order []string
		observer := &sequencedObserver{
			onAwait: func() { order = append(order, "mount-anchors-ready") },
		}
		introducer.onIntroduce = func() { order = append(order, "artifact-introduced") }
		backend := newTestBackend(t, runner, introducer, observer, availableProbe)

		result, err := backend.Execute(context.Background(), sandboxRequest(t))
		if err != nil || result.Status() != domain.SandboxCompleted {
			t.Fatalf("Execute() = (%q, %v)", result.Status(), err)
		}
		if len(order) < 2 || order[0] != "mount-anchors-ready" || order[1] != "artifact-introduced" {
			t.Fatalf("execution order = %#v, want [mount-anchors-ready, artifact-introduced]", order)
		}
	})
}

func assertConstrainedCreateCommand(t *testing.T, arguments []string) {
	t.Helper()
	joined := strings.Join(arguments, " ")
	for _, required := range []string{"--runtime " + gVisorRuntimeName, "--network none", "--read-only", "--cap-drop ALL", "--cap-add SETUID", "--cap-add SETGID", "--cap-add SETPCAP", "no-new-privileges", "--pids-limit 64", "--memory 512m", "--cpus 1", "--ulimit cpu=30:30", "--tmpfs " + boundaryHelperMount, nodeImageReference, boundaryContainerCommand()} {
		if !strings.Contains(joined, required) {
			t.Errorf("create command missing %q: %q", required, joined)
		}
	}
	if strings.Contains(joined, "--user 1000:1000") {
		t.Errorf("OCI helper initializer must run as root: %q", joined)
	}
	for _, forbidden := range []string{"--mount", "--volume", "-v ", "--privileged", "--pid host", "--network host", "/var/run/docker.sock"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("create command contains forbidden %q: %q", forbidden, joined)
		}
	}
}

func newTestBackend(t *testing.T, runner CommandRunner, introducer ArtifactIntroducer, observer TraceObserver, probe CapabilityProbe) *Backend {
	t.Helper()
	backend, err := NewBackend(runner, introducer, observer, probe)
	if err != nil {
		t.Fatalf("NewBackend() error = %v", err)
	}
	return backend
}

func availableProbe(context.Context) (Capability, error) { return Capability{Available: true}, nil }

type commandCall struct {
	binary    string
	arguments []string
	bounded   bool
}
type recordingRunner struct {
	calls            []commandCall
	inputCalls       []commandCall
	timeline         []commandCall
	input            []byte
	responses        [][]byte
	errors           []error
	boundedOutput    []byte
	waitForContext   bool
	waitForContextAt int
	cleanupSignal    chan struct{}
}

func (r *recordingRunner) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	_, bounded := ctx.Deadline()
	r.calls = append(r.calls, commandCall{binary: binary, arguments: append([]string(nil), arguments...), bounded: bounded})
	r.timeline = append(r.timeline, commandCall{binary: binary, arguments: append([]string(nil), arguments...), bounded: bounded})
	if r.cleanupSignal != nil && binary == "docker" && len(arguments) == 3 &&
		arguments[0] == "rm" && arguments[1] == "--force" {
		close(r.cleanupSignal)
		r.cleanupSignal = nil
	}
	index := len(r.calls) - 1
	if r.waitForContext && index == r.waitForContextAt {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if index < len(r.errors) && r.errors[index] != nil {
		return nil, r.errors[index]
	}
	if index < len(r.responses) {
		return r.responses[index], nil
	}
	return nil, nil
}

func (r *recordingRunner) RunDiscard(ctx context.Context, binary string, arguments ...string) error {
	_, err := r.Output(ctx, binary, arguments...)
	return err
}

func (r *recordingRunner) RunBounded(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	_, err := r.Output(ctx, binary, arguments...)
	return append([]byte(nil), r.boundedOutput...), err
}

type recordingIntroducer struct {
	calls       int
	containerID string
}

type emptyObserver struct{}

func (emptyObserver) Start(context.Context, string) (TraceReader, error) { return &traceReader{}, nil }
func (emptyObserver) AwaitMountAnchors(context.Context, string) error    { return nil }

type plainObserver struct{}

func (plainObserver) Start(context.Context, string) (TraceReader, error) { return &traceReader{}, nil }

type sequencedObserver struct {
	onAwait func()
}

func (s *sequencedObserver) Start(context.Context, string) (TraceReader, error) {
	return &traceReader{}, nil
}
func (s *sequencedObserver) AwaitMountAnchors(context.Context, string) error {
	if s.onAwait != nil {
		s.onAwait()
	}
	return nil
}

type sequencedIntroducer struct {
	onIntroduce func()
}

func (s *sequencedIntroducer) Introduce(context.Context, string, domain.AcquiredArtifact) error {
	if s.onIntroduce != nil {
		s.onIntroduce()
	}
	return nil
}

type cleanupGatedTraceReader struct {
	cleanup <-chan struct{}
	records []TraceRecord
}

func (r *cleanupGatedTraceReader) Next(ctx context.Context) (TraceRecord, error) {
	select {
	case <-r.cleanup:
	case <-ctx.Done():
		return TraceRecord{}, ctx.Err()
	}
	if len(r.records) == 0 {
		return TraceRecord{}, io.EOF
	}
	record := r.records[0]
	r.records = r.records[1:]
	return record, nil
}

type recordingObserver struct {
	containerID string
	profile     string
	reader      TraceReader
	err         error
}

func (o *recordingObserver) Start(_ context.Context, containerID string) (TraceReader, error) {
	o.containerID = containerID
	if o.err != nil {
		return nil, o.err
	}
	return o.reader, nil
}

func (o *recordingObserver) StartProfile(_ context.Context, containerID, profile string) (TraceReader, error) {
	o.containerID = containerID
	o.profile = profile
	if o.err != nil {
		return nil, o.err
	}
	return o.reader, nil
}

func (o *recordingObserver) AwaitMountAnchors(_ context.Context, _ string) error { return o.err }

func (r *recordingIntroducer) Introduce(_ context.Context, containerID string, _ domain.AcquiredArtifact) error {
	r.calls++
	r.containerID = containerID
	return nil
}

func sandboxRequest(t *testing.T) domain.SandboxRequest {
	t.Helper()
	source, err := domain.NewSourceID("npm")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(source, "example", "1.0.0", "tarball")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func sameStrings(actual, expected []string) bool {
	return strings.Join(actual, "\x00") == strings.Join(expected, "\x00")
}
