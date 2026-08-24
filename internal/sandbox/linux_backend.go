package sandbox

import (
	"context"
	"errors"
	"runtime"
)

// ObserverOutputEndpoint is the HAA-only normalized-record endpoint. The
// separately installed gVisor observer helper sends to it; it is never mounted
// into an Artifact container.
const ObserverOutputEndpoint = "/run/heliopause-observer/haa-output.sock"

// NewLinuxBackendWithExecutor composes the production Docker backend from a
// composition-root validated Host executor and its process-scoped observer.
func NewLinuxBackendWithExecutor(intakeRoot string, executor interface {
	Executor
	CommandRunner
	inputCommandRunner
}, observer TraceObserver) (*Backend, error) {
	return newLinuxBackend(intakeRoot, executor, observer)
}

func newLinuxBackend(intakeRoot string, executor interface {
	Executor
	CommandRunner
	inputCommandRunner
}, observer TraceObserver) (*Backend, error) {
	if intakeRoot == "" || observer == nil {
		return nil, errors.New("sandbox intake root and process-scoped observer are required")
	}
	introducer, err := NewDockerArtifactIntroducer(intakeRoot, executor)
	if err != nil {
		return nil, err
	}
	capabilityProbe := func(ctx context.Context) (Capability, error) {
		return probe(ctx, runtime.GOOS, executor)
	}
	backend, err := NewBackend(executor, introducer, observer, capabilityProbe)
	if err != nil {
		return nil, err
	}
	return backend, nil
}
