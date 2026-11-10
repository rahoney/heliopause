package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// This mirror holds authenticated wheel DATA only. It introduces no pipx
// runtime, receipt, Host target, execution admission or Policy authority.
type uvPublicRawMirror struct {
	session pipxNativeSession
	volume  *pipxWheelVolume
}

func (m *uvPublicRawMirror) prepare(ctx context.Context, input *pipxNativeInput) error {
	id, err := domain.NewSandboxSessionID()
	if err != nil {
		return err
	}
	m.session.transaction = id.String()
	m.volume, err = createPipxWheelVolume(ctx, m.session.runner, id.String(), input)
	if err != nil {
		return err
	}
	rw, err := m.volume.preparationMount()
	if err != nil {
		return err
	}
	writer, err := m.session.dataKeeper(ctx, "0:0", "--mount", rw)
	if err != nil {
		return err
	}
	if err := m.volume.seed(ctx, m.session.runner, writer.id); err != nil {
		return err
	}
	ro, err := m.volume.readerMount()
	if err != nil {
		return err
	}
	anchor, err := m.session.dataKeeper(ctx, "1000:1000", "--mount", ro)
	if err != nil {
		return err
	}
	if err := m.volume.bindAnchor(ctx, m.session.runner, anchor.id); err != nil {
		return err
	}
	if err := m.session.removeKeeper(ctx, writer); err != nil {
		return err
	}
	return m.volume.seal(ctx, m.session.runner)
}

func (m *uvPublicRawMirror) close() error {
	ctx, cancel := resolverCleanupContext()
	defer cancel()
	var result error
	for i := len(m.session.keepers) - 1; i >= 0; i-- {
		result = errors.Join(result, m.session.removeKeeper(ctx, m.session.keepers[i]))
	}
	if m.volume != nil {
		result = errors.Join(result, m.volume.close(ctx, m.session.runner))
	}
	return result
}

// These are the same mounts already qualified by the public native fixture.
func uvFrozenRunCreateArguments(runtime uvRuntimeVolume, output closureVolume, wheels *pipxWheelVolume) ([]string, error) {
	if wheels == nil || output.destination() != pythonSitePath {
		return nil, errPipxNativeInput
	}
	runtimeMount, err := runtime.mountArgument()
	if err != nil {
		return nil, err
	}
	outputMount, err := output.mountArgument(false)
	if err != nil {
		return nil, err
	}
	wheelMount, err := wheels.readerMount()
	if err != nil {
		return nil, err
	}
	return pythonResolverCreateArguments("none", nil, []string{"--mount", runtimeMount}, []string{"--mount", outputMount, "--mount", wheelMount}), nil
}

func startUVFrozenRunTrace(ctx context.Context, observer TraceObserver, container string) (TraceReader, error) {
	if ctx == nil || observer == nil || !exactObservationContainerID(container) {
		return nil, errPipxNativeInput
	}
	topology, ok := observerExpectedTopology(uvResolverProfile)
	if !ok {
		return nil, errors.New("UV base topology unavailable")
	}
	topology = append(topology, observerMountExpectation{pipxNativeWheels, "workspace", "/", "9p", true, false, false, false})
	bound, ok := observer.(interface {
		startProfile(context.Context, string, string, []observerMountExpectation) (TraceReader, error)
	})
	if !ok {
		return nil, errors.New("UV readonly input topology binding unavailable")
	}
	return bound.startProfile(ctx, container, uvResolverProfile, topology)
}
