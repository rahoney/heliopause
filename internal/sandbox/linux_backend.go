package sandbox

import "errors"

// ObserverOutputEndpoint is the HAA-only normalized-record endpoint. The
// separately installed gVisor observer helper sends to it; it is never mounted
// into an Artifact container.
const ObserverOutputEndpoint = "/run/heliopause-observer/haa-output.sock"

// NewLinuxBackend composes the production Docker backend with the fixed,
// trusted observer endpoint mandated by the installed runsc-trace runtime.
// The caller owns observer.Close and must keep it alive for the inspection.
func NewLinuxBackend(intakeRoot string) (*Backend, *SharedObserver, error) {
	if intakeRoot == "" {
		return nil, nil, errors.New("sandbox intake root is required")
	}
	observer, err := NewSharedObserver(ObserverOutputEndpoint)
	if err != nil {
		return nil, nil, err
	}
	introducer, err := NewDockerArtifactIntroducer(intakeRoot, systemExecutor{})
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	backend, err := NewBackend(systemExecutor{}, introducer, observer, Probe)
	if err != nil {
		_ = observer.Close()
		return nil, nil, err
	}
	return backend, observer, nil
}
