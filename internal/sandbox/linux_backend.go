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
// composition-root validated Host executor. The caller owns observer.Close.
func NewLinuxBackendWithExecutor(intakeRoot string, executor interface {
	Executor
	CommandRunner
	inputCommandRunner
}) (*Backend, *SharedObserver, error) {
	return newLinuxBackend(intakeRoot, executor)
}

func newLinuxBackend(intakeRoot string, executor interface {
	Executor
	CommandRunner
	inputCommandRunner
}) (*Backend, *SharedObserver, error) {
	if intakeRoot == "" {
		return nil, nil, errors.New("sandbox intake root is required")
	}
	observer, err := NewSharedObserver(ObserverOutputEndpoint)
	if err != nil {
		return nil, nil, err
	}
	introducer, err := NewDockerArtifactIntroducer(intakeRoot, executor)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	capabilityProbe := func(ctx context.Context) (Capability, error) {
		return probe(ctx, runtime.GOOS, executor)
	}
	backend, err := NewBackend(executor, introducer, observer, capabilityProbe)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	return backend, observer, nil
}
