package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const pipxRuntimeRoot = "/haa-pipx"
const pipxDataVolumeCapacity int64 = 8 << 20

// This owner attests DATA transport and lifetime only. It cannot authorize
// Python execution or supply external observation completion evidence.
type readonlyToolVolume struct {
	source  *sourceAnalysisPreparedInput
	semgrep bool
	volume  closureVolume
	writer  string
	anchor  string
	seeded  bool
	sealed  bool
}

// Preserve the existing qualified Pipx surface while sharing its state machine.
type pipxDataVolume = readonlyToolVolume

func createPipxDataVolume(ctx context.Context, runner CommandRunner, transaction string) (*pipxDataVolume, error) {
	volume, err := createInputVolume(ctx, runner, transaction+"-pipx-data", runtimeidentity.PipxToolSHA256,
		pipxDataVolumeCapacity, false, projectBuildVolumeConfig{destination: pipxRuntimeRoot})
	if volume.name == "" {
		return nil, err
	}
	return &pipxDataVolume{volume: volume}, err
}

func (v *readonlyToolVolume) valid() bool {
	if v != nil && v.source != nil {
		return !v.semgrep && len(v.source.manifestID) == 64 && v.volume.destination() == sourceAnalysisRoot && v.volume.manifestID == v.source.manifestID && v.volume.capacity == sourceAnalysisVolumeCapacity && !v.volume.goBuild && !v.volume.goBuildOutput && !v.volume.projectBuildExecutable
	}
	if v != nil && v.semgrep {
		return v.volume.destination() == semgrepRuntimeRoot && v.volume.manifestID == semgrepSupplyDigest() && v.volume.capacity == semgrepSupplyVolumeCapacity && !v.volume.goBuild && !v.volume.goBuildOutput && v.volume.projectBuildExecutable
	}
	return v != nil && v.volume.destination() == pipxRuntimeRoot &&
		v.volume.manifestID == runtimeidentity.PipxToolSHA256 && v.volume.capacity == pipxDataVolumeCapacity &&
		!v.volume.goBuild && !v.volume.goBuildOutput && !v.volume.projectBuildExecutable
}

func (v *readonlyToolVolume) preparationMount() (string, error) {
	if !v.valid() || v.writer != "" || v.sealed {
		return "", errPipxToolSupply
	}
	return v.volume.mountArgument(false)
}

func (v *readonlyToolVolume) readerMount() (string, error) {
	if !v.valid() || !v.seeded {
		return "", errPipxToolSupply
	}
	return v.volume.mountArgument(true)
}

func (v *readonlyToolVolume) seed(ctx context.Context, runner CommandRunner, tool interface {
	introduce(context.Context, CommandRunner, string) error
}, writer string) error {
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

func (v *readonlyToolVolume) bindAnchor(ctx context.Context, runner CommandRunner, anchor string) error {
	if !v.valid() || !v.seeded || v.sealed || v.anchor != "" || anchor == v.writer {
		return errPipxToolSupply
	}
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{v.writer: false, anchor: true}); err != nil {
		return err
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, v.writer); err != nil {
		return err
	}
	if err := v.verifyReaderBytes(ctx, runner, anchor); err != nil {
		return err
	}
	v.anchor = anchor
	return nil
}

// The trusted lifecycle controller must remove preparation before sealing.
// An exited but still attached writer, or a failed inventory, is not absence.
func (v *readonlyToolVolume) seal(ctx context.Context, runner CommandRunner) error {
	if !v.valid() || !v.seeded || v.anchor == "" || v.sealed {
		return errPipxToolSupply
	}
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{v.anchor: true}); err != nil {
		return err
	}
	if err := v.verifyReaderBytes(ctx, runner, v.anchor); err != nil {
		return err
	}
	v.sealed = true
	return nil
}

func (v *readonlyToolVolume) verifyReader(ctx context.Context, runner CommandRunner, reader string) error {
	if !v.valid() || !v.sealed || reader == v.writer {
		return errPipxToolSupply
	}
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{v.anchor: true, reader: true}); err != nil {
		return err
	}
	if err := v.verifyReaderBytes(ctx, runner, v.anchor); err != nil {
		return err
	}
	if reader != v.anchor {
		return v.verifyReaderBytes(ctx, runner, reader)
	}
	return nil
}

func (v *readonlyToolVolume) verifyReaderBytes(ctx context.Context, runner CommandRunner, container string) error {
	if !v.valid() {
		return errPipxToolSupply
	}
	if v.source != nil {
		return v.source.readback(ctx, runner, container)
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, container); err != nil {
		return err
	}
	output, ok := runner.(streamingDockerOutput)
	if !ok {
		return errors.New("pipx DATA reader requires bounded streaming transport")
	}
	kinds := []nativeRuntimeToolKind{nativeRuntimePipx}
	if v.semgrep {
		kinds = []nativeRuntimeToolKind{nativeRuntimeSemgrep, nativeRuntimeSemgrepRules}
	}
	for _, kind := range kinds {
		spec, ok := nativeRuntimeToolIdentity(kind)
		if !ok {
			return errPipxToolSupply
		}
		if _, err := readNativeToolFromContainer(ctx, output, container, spec.guestPath, nativeToolFileIdentity{spec.size, spec.sha256, spec.fileMode()}, kind); err != nil {
			return err
		}
	}
	return nil
}

func (v *readonlyToolVolume) close(ctx context.Context, runner CommandRunner) error {
	if !v.valid() {
		return errPipxToolSupply
	}
	if err := closeWritableOutputVolume(ctx, runner, &v.volume, v.volume.destination()); err != nil {
		return err
	}
	*v = readonlyToolVolume{}
	return nil
}
