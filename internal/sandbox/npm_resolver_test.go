package sandbox

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestNPMCreateArgumentsConstrainedAndBoundaryConfigured(t *testing.T) {
	t.Parallel()

	network := "haa-resolver-test-net"
	hostArguments := []string{"--add-host", "registry.npmjs.org:1.1.1.1"}
	arguments := npmCreateArguments(network, hostArguments)
	joined := strings.Join(arguments, " ")

	for _, required := range []string{
		"--pull never",
		"--runtime " + gVisorRuntimeName,
		"--network " + network,
		"--read-only",
		"--cap-drop ALL",
		"--cap-add SETUID",
		"--cap-add SETGID",
		"--cap-add SETPCAP",
		"--security-opt no-new-privileges",
		"--pids-limit 64",
		"--memory 512m",
		"--cpus 1",
		"--tmpfs /tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700",
		"--tmpfs " + boundaryHelperMount,
		"--add-host registry.npmjs.org:1.1.1.1",
		runtimeidentity.NodeImageReference,
		boundaryContainerCommand(),
	} {
		if !strings.Contains(joined, required) {
			t.Errorf("npmCreateArguments missing %q: %q", required, joined)
		}
	}

	if strings.Contains(joined, "--user 1000:1000") {
		t.Errorf("npmCreateArguments must not specify user 1000:1000 (helper initializes as root): %q", joined)
	}
	lastArg := arguments[len(arguments)-1]
	if lastArg == "sleep infinity" || lastArg != boundaryContainerCommand() {
		t.Errorf("npmCreateArguments PID1 command must be boundaryContainerCommand(), got %q", lastArg)
	}

	for _, forbidden := range []string{
		"--mount",
		"--volume",
		"-v ",
		"--privileged",
		"--pid host",
		"--network host",
		"/var/run/docker.sock",
	} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("npmCreateArguments contains forbidden %q: %q", forbidden, joined)
		}
	}
}

func TestNPMResolverExecutionSequenceAndBoundaryWrapping(t *testing.T) {
	t.Parallel()

	lock := resolverLockJSON()
	runner := &recordingRunner{responses: [][]byte{
		[]byte("0123456789abcdef"), // network create
		[]byte("172.30.0.0/24"),    // network inspect
		[]byte("0123456789ab"),     // docker create
		nil,                        // docker start
		nil,                        // awaitBoundaryHelper (boundaryReadinessArguments)
		[]byte(resolverNPMVersion), // npm --version
		nil,                        // npm install
		[]byte(lock),               // cat package-lock.json
	}}

	var sequence []string
	observer := &sequencedObserver{
		onAwait: func() {
			sequence = append(sequence, "observer-mount-anchors-ready")
		},
	}

	service := &recordingResolverPolicyService{}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, observer, service)
	if err != nil {
		t.Fatal(err)
	}

	reference, err := artifactnpm.ParseReference("primary@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/target")
	installContext, _ := domain.NewInstallContext(target)

	resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext)
	if err != nil {
		t.Fatalf("ResolveDependencies() error = %v", err)
	}
	if len(resolution.Graph().Nodes()) != 1 || resolution.RuntimeIdentity() != resolverRuntimeIdentity || resolution.LockfileDigest().String() == "" {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}

	// Verify all exec calls (both runner.calls and runner.inputCalls) use boundary wrapping.
	execCount := 0
	for _, call := range runner.calls {
		if call.binary != "docker" || len(call.arguments) < 2 {
			continue
		}
		if call.arguments[0] == "exec" {
			execCount++
			assertBoundaryExecArguments(t, call.arguments, "0123456789ab")
		}
	}
	if execCount < 4 { // readiness check, version check, npm install, cat lockfile
		t.Fatalf("expected at least 4 boundary exec calls, got %d", execCount)
	}

	if len(runner.inputCalls) != 1 {
		t.Fatalf("expected 1 input call for package.json, got %d", len(runner.inputCalls))
	}
	inputCall := runner.inputCalls[0]
	if inputCall.binary != "docker" || inputCall.arguments[0] != "exec" || inputCall.arguments[1] != "-i" {
		t.Fatalf("input call is not docker exec -i: %#v", inputCall)
	}
	assertBoundaryInputExecArguments(t, inputCall.arguments, "0123456789ab")

	// Verify readiness sequence: start -> awaitBoundaryHelper -> awaitMountAnchors -> npm exec
	var startIdx, helperIdx, npmIdx int
	for idx, call := range runner.calls {
		if call.binary == "docker" && len(call.arguments) >= 2 {
			if call.arguments[0] == "start" {
				startIdx = idx
			}
			if call.arguments[0] == "exec" && strings.Contains(strings.Join(call.arguments, " "), boundaryReadinessScript) {
				helperIdx = idx
			}
			if call.arguments[0] == "exec" && strings.Contains(strings.Join(call.arguments, " "), "npm") {
				if npmIdx == 0 {
					npmIdx = idx
				}
			}
		}
	}
	if startIdx == 0 || helperIdx == 0 || npmIdx == 0 {
		t.Fatalf("missing expected calls: start=%d, helper=%d, npm=%d", startIdx, helperIdx, npmIdx)
	}
	if helperIdx <= startIdx {
		t.Fatalf("awaitBoundaryHelper (%d) must occur after docker start (%d)", helperIdx, startIdx)
	}
	if npmIdx <= helperIdx {
		t.Fatalf("npm exec (%d) must occur after awaitBoundaryHelper (%d)", npmIdx, helperIdx)
	}
	if len(sequence) != 1 || sequence[0] != "observer-mount-anchors-ready" {
		t.Fatalf("observer mount anchors was not awaited: %#v", sequence)
	}
}

func TestNPMResolverObserverFaultRetainsDiagnosticAndFailsClosed(t *testing.T) {
	t.Parallel()

	lock := resolverLockJSON()
	runner := &recordingRunner{responses: [][]byte{
		[]byte("0123456789abcdef"), // network create
		[]byte("172.30.0.0/24"),    // network inspect
		[]byte("0123456789ab"),     // docker create
		nil,                        // docker start
		nil,                        // awaitBoundaryHelper
		[]byte(resolverNPMVersion), // npm --version
		nil,                        // npm install
		[]byte(lock),               // cat package-lock.json
	}}

	// Observer emits fault with TOPOLOGY_MISMATCH
	observer := &recordingObserver{
		reader: &traceReader{err: observerFault{reason: "TOPOLOGY_MISMATCH"}},
	}

	service := &recordingResolverPolicyService{}
	resolver, err := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, observer, service)
	if err != nil {
		t.Fatal(err)
	}

	reference, _ := artifactnpm.ParseReference("primary@1.0.0")
	target, _ := domain.NewInstallTarget("/tmp/target")
	installContext, _ := domain.NewInstallContext(target)

	resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext)
	if err == nil {
		t.Fatal("ResolveDependencies() expected error on observer fault, got nil")
	}

	// Must retain the TraceDiagnostic string with TOPOLOGY_MISMATCH
	if !strings.Contains(err.Error(), "npm resolver observation is incomplete: reason=TOPOLOGY_MISMATCH") {
		t.Fatalf("expected error to contain TraceDiagnostic reason=TOPOLOGY_MISMATCH, got %q", err.Error())
	}

	// Resolution must be zeroed
	if len(resolution.Graph().Nodes()) != 0 || resolution.LockfileDigest().String() != "" || resolution.RuntimeIdentity() != "" {
		t.Fatalf("resolution must be empty on observer fault, got: %#v", resolution)
	}

	// Container and network cleanup must be performed
	cleanedUpContainer := false
	for _, call := range runner.calls {
		if call.binary == "docker" && len(call.arguments) >= 3 && call.arguments[0] == "rm" && call.arguments[1] == "--force" && call.arguments[2] == "0123456789ab" {
			cleanedUpContainer = true
		}
	}
	if !cleanedUpContainer {
		t.Fatal("container was not cleaned up after observer fault")
	}
	if service.remove != 1 {
		t.Fatalf("network policy was not cleaned up: service.remove = %d", service.remove)
	}
}

func TestNPMResolverFailsClosedOnReadinessFailure(t *testing.T) {
	t.Parallel()

	t.Run("boundary helper failure", func(t *testing.T) {
		runner := &recordingRunner{
			responses: [][]byte{
				[]byte("0123456789abcdef"), // network create
				[]byte("172.30.0.0/24"),    // network inspect
				[]byte("0123456789ab"),     // docker create
				nil,                        // docker start
			},
			errors: []error{
				nil, nil, nil, nil,
				errors.New("helper not ready"), // awaitBoundaryHelper attempts fail
			},
		}
		observer := &recordingObserver{reader: &traceReader{}}
		service := &recordingResolverPolicyService{}
		resolver, _ := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, observer, service)

		reference, _ := artifactnpm.ParseReference("primary@1.0.0")
		target, _ := domain.NewInstallTarget("/tmp/target")
		installContext, _ := domain.NewInstallContext(target)

		resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext)
		if err == nil {
			t.Fatal("expected error on boundary helper readiness failure")
		}
		if len(resolution.Graph().Nodes()) != 0 {
			t.Fatalf("resolution must be empty: %#v", resolution)
		}
		if service.remove != 1 {
			t.Fatalf("network policy was not cleaned up: service.remove = %d", service.remove)
		}
	})

	t.Run("mount anchors failure", func(t *testing.T) {
		runner := &recordingRunner{
			responses: [][]byte{
				[]byte("0123456789abcdef"), // network create
				[]byte("172.30.0.0/24"),    // network inspect
				[]byte("0123456789ab"),     // docker create
				nil,                        // docker start
				nil,                        // awaitBoundaryHelper succeeds
			},
		}
		observer := &recordingObserver{err: errors.New("mount anchors not reconciled")}
		service := &recordingResolverPolicyService{}
		resolver, _ := NewNPMResolverWithObserver(runner, staticEndpoints{addresses: []netip.Addr{netip.MustParseAddr("1.1.1.1")}}, observer, service)

		reference, _ := artifactnpm.ParseReference("primary@1.0.0")
		target, _ := domain.NewInstallTarget("/tmp/target")
		installContext, _ := domain.NewInstallContext(target)

		resolution, err := resolver.ResolveDependencies(context.Background(), reference, installContext)
		if err == nil {
			t.Fatal("expected error on mount anchors failure")
		}
		if len(resolution.Graph().Nodes()) != 0 {
			t.Fatalf("resolution must be empty: %#v", resolution)
		}
		if service.remove != 1 {
			t.Fatalf("network policy was not cleaned up: service.remove = %d", service.remove)
		}
	})
}

func assertBoundaryExecArguments(t *testing.T, arguments []string, containerID string) {
	t.Helper()
	// Boundary exec format: exec --user 0:0 <containerID> /haa-runtime/haa-boundary --origin-launch <cmd...>
	if len(arguments) < 6 {
		t.Fatalf("exec command too short: %#v", arguments)
	}
	if arguments[0] != "exec" || arguments[1] != "--user" || arguments[2] != boundaryBootstrapUser || arguments[3] != containerID || arguments[4] != boundaryHelperPath || arguments[5] != boundaryOriginLaunchMode {
		t.Fatalf("exec arguments do not use canonical boundary helper wrapping: %#v", arguments)
	}
}

func assertBoundaryInputExecArguments(t *testing.T, arguments []string, containerID string) {
	t.Helper()
	// Boundary input exec format: exec -i --user 0:0 <containerID> /haa-runtime/haa-boundary --origin-launch <cmd...>
	if len(arguments) < 7 {
		t.Fatalf("input exec command too short: %#v", arguments)
	}
	if arguments[0] != "exec" || arguments[1] != "-i" || arguments[2] != "--user" || arguments[3] != boundaryBootstrapUser || arguments[4] != containerID || arguments[5] != boundaryHelperPath || arguments[6] != boundaryOriginLaunchMode {
		t.Fatalf("input exec arguments do not use canonical boundary helper wrapping: %#v", arguments)
	}
}
