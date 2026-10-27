package sandbox

import (
	"context"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const pipxRuntimeRoot = "/haa-pipx"
const pipxDataVolumeCapacity int64 = 8 << 20

// This owner attests DATA transport and lifetime only. It cannot authorize
// Python execution or supply external observation completion evidence.
type pipxDataVolume struct {
	volume closureVolume
	writer string
	anchor string
	seeded bool
	sealed bool
}

func createPipxDataVolume(ctx context.Context, runner CommandRunner, transaction string) (*pipxDataVolume, error) {
	volume, err := createInputVolume(ctx, runner, transaction+"-pipx-data", runtimeidentity.PipxToolSHA256,
		pipxDataVolumeCapacity, false, projectBuildVolumeConfig{destination: pipxRuntimeRoot})
	if volume.name == "" {
		return nil, err
	}
	return &pipxDataVolume{volume: volume}, err
}

func pipxShadowMount(destination string) bool {
	return destination == "/" || strings.HasPrefix(destination, pipxRuntimeRoot+"/")
}

func (v *pipxDataVolume) valid() bool {
	return v != nil && v.volume.destination() == pipxRuntimeRoot &&
		v.volume.manifestID == runtimeidentity.PipxToolSHA256 && v.volume.capacity == pipxDataVolumeCapacity &&
		!v.volume.goBuild && !v.volume.goBuildOutput && !v.volume.projectBuildExecutable
}

func (v *pipxDataVolume) preparationMount() (string, error) {
	if !v.valid() || v.writer != "" || v.sealed {
		return "", errPipxToolSupply
	}
	return v.volume.mountArgument(false)
}

func (v *pipxDataVolume) readerMount() (string, error) {
	if !v.valid() || !v.seeded {
		return "", errPipxToolSupply
	}
	return v.volume.mountArgument(true)
}

func (v *pipxDataVolume) seed(ctx context.Context, runner CommandRunner, tool *pipxPreparedTool, writer string) error {
	if !v.valid() || v.writer != "" || v.seeded || v.sealed || tool == nil {
		return errPipxToolSupply
	}
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{writer: false}); err != nil {
		return err
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, writer); err != nil {
		return err
	}
	// A partially copied supply is never retried or adopted by this handle.
	v.writer = writer
	if err := tool.introduce(ctx, runner, writer); err != nil {
		return err
	}
	v.seeded = true
	return nil
}

func (v *pipxDataVolume) bindAnchor(ctx context.Context, runner CommandRunner, anchor string) error {
	if !v.valid() || !v.seeded || v.sealed || v.anchor != "" || anchor == v.writer {
		return errPipxToolSupply
	}
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{v.writer: false, anchor: true}); err != nil {
		return err
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, v.writer); err != nil {
		return err
	}
	if err := verifyPipxDataReader(ctx, runner, anchor); err != nil {
		return err
	}
	v.anchor = anchor
	return nil
}

// The trusted lifecycle controller must remove preparation before sealing.
// An exited but still attached writer, or a failed inventory, is not absence.
func (v *pipxDataVolume) seal(ctx context.Context, runner CommandRunner) error {
	if !v.valid() || !v.seeded || v.anchor == "" || v.sealed {
		return errPipxToolSupply
	}
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{v.anchor: true}); err != nil {
		return err
	}
	if err := verifyPipxDataReader(ctx, runner, v.anchor); err != nil {
		return err
	}
	v.sealed = true
	return nil
}

func (v *pipxDataVolume) verifyReader(ctx context.Context, runner CommandRunner, reader string) error {
	if !v.valid() || !v.sealed || reader == v.writer {
		return errPipxToolSupply
	}
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{v.anchor: true, reader: true}); err != nil {
		return err
	}
	if err := verifyPipxDataReader(ctx, runner, v.anchor); err != nil {
		return err
	}
	if reader != v.anchor {
		return verifyPipxDataReader(ctx, runner, reader)
	}
	return nil
}

func verifyPipxDataReader(ctx context.Context, runner CommandRunner, container string) error {
	if err := verifyObservationRuntimeAlive(ctx, runner, container); err != nil {
		return err
	}
	output, ok := runner.(streamingDockerOutput)
	if !ok {
		return errors.New("pipx DATA reader requires bounded streaming transport")
	}
	spec, ok := nativeRuntimeToolIdentity(nativeRuntimePipx)
	if !ok {
		return errPipxToolSupply
	}
	_, err := readNativeToolFromContainer(ctx, output, container, spec.guestPath,
		nativeToolFileIdentity{spec.size, spec.sha256, spec.fileMode()}, nativeRuntimePipx)
	return err
}

func (v *pipxDataVolume) close(ctx context.Context, runner CommandRunner) error {
	if !v.valid() {
		return errPipxToolSupply
	}
	if err := closeWritableOutputVolume(ctx, runner, &v.volume, pipxRuntimeRoot); err != nil {
		return err
	}
	*v = pipxDataVolume{}
	return nil
}
