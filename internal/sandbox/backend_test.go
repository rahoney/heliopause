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

func TestBackendUsesConstrainedDockerBaseCommand(t *testing.T) {
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	assertConstrainedCreateCommand(t, createArguments(session))
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
	backend, model, request := newEnvironmentBackendModel(t)
	model.traceRecords = []TraceRecord{{Kind: "network-attempt", Bytes: 1}}
	result, err := backend.Execute(context.Background(), request)
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("Execute() = (%q, %v)", result.Status(), err)
	}
	count := uint64(0)
	for _, fact := range result.Observations() {
		if fact.Category() == domain.ObservationNetwork {
			count += fact.Count()
		}
	}
	if count != 4 || len(model.containers) != 0 {
		t.Fatalf("finalized trace not drained: network=%d containers=%d", count, len(model.containers))
	}
}

func TestBackendObserverFailureIsIncompleteAndDisposesContainer(t *testing.T) {
	backend, model, request := newEnvironmentBackendModel(t)
	model.failure = "observer"
	result, err := backend.Execute(context.Background(), request)
	if err == nil || result.Status() != domain.SandboxIncomplete {
		t.Fatalf("Execute() = (%q, %v)", result.Status(), err)
	}
	if code, _ := result.LimitationCode(); code != "M3_DYNAMIC_SETUP_FAILED" {
		t.Fatalf("LimitationCode() = %q", code)
	}
	if model.commands != 0 || model.introductions != 0 || len(model.containers) != 0 || len(model.volumes) != 0 {
		t.Fatal("observer setup failure executed input or retained owned resources")
	}
}

func TestBackendProcessFailureAndCleanupFailureAreIncomplete(t *testing.T) {
	commandFailure, cleanupFailure := errors.New("exit status 1"), errors.New("cleanup")
	tests := []struct {
		name             string
		command, cleanup error
		code             string
	}{
		{"process failure", commandFailure, nil, "M3_DYNAMIC_EXECUTION_FAILED"},
		{"cleanup failure", nil, cleanupFailure, "M3_DYNAMIC_CLEANUP_FAILED"},
		{"primary failure preserved", commandFailure, cleanupFailure, "M3_DYNAMIC_EXECUTION_FAILED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend, model, request := newEnvironmentBackendModel(t)
			model.commandCause, model.closeCause = test.command, test.cleanup
			result, err := backend.Execute(context.Background(), request)
			code, _ := result.LimitationCode()
			if result.Status() != domain.SandboxIncomplete || code != test.code || (test.command != nil && !errors.Is(err, test.command)) || (test.cleanup != nil && !errors.Is(err, test.cleanup)) {
				t.Fatalf("Execute() = (%q/%s, %v)", result.Status(), code, err)
			}
			if len(model.containers) != 0 || len(model.volumes) != 0 {
				t.Fatal("owned resources retained")
			}
		})
	}
}

func TestBackendTimeoutIsIncompleteAndStillDisposed(t *testing.T) {
	backend, model, request := newEnvironmentBackendModel(t)
	model.failure = "timeout"
	backend.wallTimeout = 20 * time.Millisecond
	result, err := backend.Execute(context.Background(), request)
	code, _ := result.LimitationCode()
	if result.Status() != domain.SandboxIncomplete || code != "M3_DYNAMIC_TIMEOUT" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Execute() = (%q/%s, %v)", result.Status(), code, err)
	}
	if model.commands != 1 || len(model.containers) != 0 || len(model.volumes) != 0 {
		t.Fatal("timeout did not stop before next experiment and drain owned resources")
	}
}

func TestBackendArtifactIntroductionBlockedUntilMountAnchorsReady(t *testing.T) {
	t.Run("blocked when observer fails topology reconciliation", func(t *testing.T) {
		backend, model, request := newEnvironmentBackendModel(t)
		model.failure = "topology"
		result, err := backend.Execute(context.Background(), request)
		if err == nil || result.Status() != domain.SandboxIncomplete || model.introductions != 0 || model.commands != 0 || len(model.containers) != 0 {
			t.Fatalf("unready topology allowed input: %s %v", result.Status(), err)
		}
	})
	t.Run("blocked when observer lacks environment topology contract", func(t *testing.T) {
		backend, model, request := newEnvironmentBackendModel(t)
		backend.observer = &plainObserver{}
		result, err := backend.Execute(context.Background(), request)
		if err != nil || result.Status() != domain.SandboxIncomplete || model.creates != 0 || model.introductions != 0 {
			t.Fatal("legacy observer admitted environment experiments")
		}
	})
	t.Run("ordered strictly before artifact introduction", func(t *testing.T) {
		backend, model, request := newEnvironmentBackendModel(t)
		result, err := backend.Execute(context.Background(), request)
		if err != nil || result.Status() != domain.SandboxCompleted || strings.Join(model.order, ",") != "mount-anchors-ready,artifact-introduced,mount-anchors-ready,mount-anchors-ready,artifact-introduced,mount-anchors-ready" {
			t.Fatalf("order=%q result=%s err=%v", model.order, result.Status(), err)
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
}

func (r *recordingRunner) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	_, bounded := ctx.Deadline()
	r.calls = append(r.calls, commandCall{binary: binary, arguments: append([]string(nil), arguments...), bounded: bounded})
	r.timeline = append(r.timeline, commandCall{binary: binary, arguments: append([]string(nil), arguments...), bounded: bounded})

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
	output, err := r.Output(ctx, binary, arguments...)
	if len(r.boundedOutput) > 0 {
		return append([]byte(nil), r.boundedOutput...), err
	}
	if err != nil {
		return output, err
	}
	return output, err
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
