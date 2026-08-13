package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestBackendExecutesOneShotSandboxWithConstrainedDockerCommand(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef\n"), nil, []byte("0\n"), nil}}
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
	if len(runner.calls) != 4 {
		t.Fatalf("command calls = %d, want 4", len(runner.calls))
	}
	assertConstrainedCreateCommand(t, runner.calls[0].arguments)
	if got := runner.calls[1]; got.binary != "docker" || !sameStrings(got.arguments, []string{"start", "0123456789abcdef"}) {
		t.Fatalf("start command = %q %q", got.binary, got.arguments)
	}
	if got := runner.calls[2]; got.binary != "docker" || !sameStrings(got.arguments, []string{"wait", "0123456789abcdef"}) {
		t.Fatalf("wait command = %q %q", got.binary, got.arguments)
	}
	if got := runner.calls[3]; got.binary != "docker" || !sameStrings(got.arguments, []string{"rm", "--force", "0123456789abcdef"}) {
		t.Fatalf("cleanup command = %q %q", got.binary, got.arguments)
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

func TestBackendCollectsTrustedObservationBeforeDisposal(t *testing.T) {
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, []byte("0\n"), nil}}
	observer := &recordingObserver{reader: &traceReader{records: []TraceRecord{{Kind: "network-attempt", Bytes: 1}}}}
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
		{"process failure", [][]byte{[]byte("0123456789abcdef"), nil, []byte("1\n"), nil}, nil, "M3_DYNAMIC_EXECUTION_FAILED"},
		{"cleanup failure", [][]byte{[]byte("0123456789abcdef"), nil, []byte("0\n"), nil}, []error{nil, nil, nil, errors.New("cleanup")}, "M3_DYNAMIC_CLEANUP_FAILED"},
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
	runner := &recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), nil, nil, nil}, waitForContext: true, waitForContextAt: 2}
	backend := newTestBackend(t, runner, &recordingIntroducer{}, &emptyObserver{}, availableProbe)
	backend.wallTimeout = time.Millisecond

	result, err := backend.Execute(context.Background(), sandboxRequest(t))
	if err != nil || result.Status() != domain.SandboxIncomplete {
		t.Fatalf("Execute() = (%q, %v), want incomplete", result.Status(), err)
	}
	if code, _ := result.LimitationCode(); code != "M3_DYNAMIC_TIMEOUT" {
		t.Fatalf("LimitationCode() = %q, want timeout", code)
	}
	if len(runner.calls) != 4 || !sameStrings(runner.calls[3].arguments, []string{"rm", "--force", "0123456789abcdef"}) {
		t.Fatalf("cleanup command missing after timeout: %#v", runner.calls)
	}
}

func assertConstrainedCreateCommand(t *testing.T, arguments []string) {
	t.Helper()
	joined := strings.Join(arguments, " ")
	for _, required := range []string{"--runtime " + gVisorRuntimeName, "--user 1000:1000", "--network none", "--read-only", "--cap-drop ALL", "no-new-privileges", "--pids-limit 64", "--memory 512m", "--cpus 1", "--ulimit cpu=30:30", nodeImageReference} {
		if !strings.Contains(joined, required) {
			t.Errorf("create command missing %q: %q", required, joined)
		}
	}
	for _, forbidden := range []string{"--mount", "--volume", "-v ", "--env", "--privileged", "--pid host", "--network host", "/var/run/docker.sock"} {
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
}
type recordingRunner struct {
	calls            []commandCall
	responses        [][]byte
	errors           []error
	waitForContext   bool
	waitForContextAt int
}

func (r *recordingRunner) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	r.calls = append(r.calls, commandCall{binary: binary, arguments: append([]string(nil), arguments...)})
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

type recordingIntroducer struct {
	calls       int
	containerID string
}

type emptyObserver struct{}

func (emptyObserver) Start(context.Context, string) (TraceReader, error) { return &traceReader{}, nil }

type recordingObserver struct {
	containerID string
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
