package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// This includes fixed transport/filesystem overhead, not extra artifact budget.
const pipxWheelInputCapacity int64 = 65 << 20

type pipxWheelVolume struct {
	volume         closureVolume
	input          *pipxNativeInput
	writer, anchor string
	seeded, sealed bool
}

func createPipxWheelVolume(ctx context.Context, runner CommandRunner, transaction string, input *pipxNativeInput) (*pipxWheelVolume, error) {
	if !input.valid() {
		return nil, errPipxNativeInput
	}
	v, err := createInputVolume(ctx, runner, transaction+"-pipx-wheels", input.id, pipxWheelInputCapacity, false, projectBuildVolumeConfig{destination: pipxNativeWheels})
	if v.name == "" {
		return nil, err
	}
	return &pipxWheelVolume{volume: v, input: input}, err
}

func (v *pipxWheelVolume) valid() bool {
	return v != nil && v.input.valid() && v.volume.destination() == pipxNativeWheels && v.volume.manifestID == v.input.id && v.volume.capacity == pipxWheelInputCapacity && !v.volume.goBuild && !v.volume.goBuildOutput && !v.volume.projectBuildExecutable
}

func (v *pipxWheelVolume) preparationMount() (string, error) {
	if !v.valid() || v.writer != "" || v.sealed {
		return "", errPipxNativeInput
	}
	return v.volume.mountArgument(false)
}

func (v *pipxWheelVolume) readerMount() (string, error) {
	if !v.valid() || !v.seeded {
		return "", errPipxNativeInput
	}
	return v.volume.mountArgument(true)
}

func (v *pipxWheelVolume) seed(ctx context.Context, runner CommandRunner, writer string) error {
	if !v.valid() || v.writer != "" || v.seeded || v.sealed {
		return errPipxNativeInput
	}
	if err := v.verifyAttachments(ctx, runner, map[string]bool{writer: false}); err != nil {
		return err
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, writer); err != nil {
		return err
	}
	v.writer = writer // A failed or partial copy is never adopted or retried.
	if err := v.input.introduce(ctx, runner, writer); err != nil {
		return err
	}
	if err := v.readback(ctx, runner, writer); err != nil {
		return err
	}
	v.seeded = true
	return nil
}

func (v *pipxWheelVolume) bindAnchor(ctx context.Context, runner CommandRunner, anchor string) error {
	if !v.valid() || !v.seeded || v.sealed || v.anchor != "" || anchor == v.writer {
		return errPipxNativeInput
	}
	if err := v.verifyAttachments(ctx, runner, map[string]bool{v.writer: false, anchor: true}); err != nil {
		return err
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, v.writer); err != nil {
		return err
	}
	if err := v.readback(ctx, runner, anchor); err != nil {
		return err
	}
	v.anchor = anchor
	return nil
}

func (v *pipxWheelVolume) seal(ctx context.Context, runner CommandRunner) error {
	if !v.valid() || !v.seeded || v.anchor == "" || v.sealed {
		return errPipxNativeInput
	}
	if err := v.verifyAttachments(ctx, runner, map[string]bool{v.anchor: true}); err != nil {
		return err
	}
	if err := v.readback(ctx, runner, v.anchor); err != nil {
		return err
	}
	v.sealed = true
	return nil
}

func (v *pipxWheelVolume) verifyReader(ctx context.Context, runner CommandRunner, reader string) error {
	if !v.valid() || !v.sealed || reader == v.writer {
		return errPipxNativeInput
	}
	if err := v.verifyAttachments(ctx, runner, map[string]bool{v.anchor: true, reader: true}); err != nil {
		return err
	}
	if err := v.readback(ctx, runner, v.anchor); err != nil {
		return err
	}
	if reader != v.anchor {
		return v.readback(ctx, runner, reader)
	}
	return nil
}

func (v *pipxWheelVolume) verifyAttachments(ctx context.Context, runner CommandRunner, authorized map[string]bool) error {
	if !v.valid() {
		return errPipxNativeInput
	}
	if err := v.volume.verifyAttachments(ctx, runner, authorized); err != nil {
		return err
	}
	for id := range authorized {
		body, err := runner.Output(ctx, "docker", "inspect", "--format", containerMountInspectTemplate, id)
		var actual dockerContainerMountInspection
		if err != nil || len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &actual) != nil || actual.ID != id || len(actual.Binds) != 0 {
			return errPipxNativeInput
		}
		shadow := func(target, source string) bool {
			return target == "/" || strings.HasPrefix(target, pipxNativeWheels+"/") || source == v.volume.name && target != pipxNativeWheels
		}
		for _, m := range actual.Mounts {
			if m.Type == "bind" || shadow(m.Destination, m.Name) {
				return errPipxNativeInput
			}
		}
		for _, m := range actual.HostMounts {
			if m.Type == "bind" || shadow(m.Target, m.Source) {
				return errPipxNativeInput
			}
		}
	}
	return nil
}

func (v *pipxWheelVolume) readback(ctx context.Context, runner CommandRunner, container string) error {
	output, ok := runner.(streamingDockerOutput)
	if !v.valid() || !ok || !exactObservationContainerID(container) {
		return errPipxNativeInput
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, container); err != nil {
		return err
	}
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := output.RunOutput(ctx, writer, "docker", "cp", container+":"+pipxNativeWheels+"/.", "-")
		_ = writer.CloseWithError(err)
		done <- err
	}()
	err := v.input.scanTar(reader)
	_ = reader.CloseWithError(err)
	return errors.Join(err, <-done, ctx.Err())
}

func (p *pipxNativeInput) scanTar(input io.Reader) error {
	if !p.valid() || input == nil {
		return errPipxNativeInput
	}
	expected := map[string]pipxNativeInputFile{}
	var total int64
	for _, f := range p.files {
		expected[f.name] = f
		total += int64(len(f.body))
	}
	bounded := &io.LimitedReader{R: input, N: total + int64(len(expected)+1)*2048 + 10240}
	archive := tar.NewReader(bounded)
	seen := map[string]bool{}
	root := false
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errPipxNativeInput
		}
		name := strings.TrimPrefix(h.Name, "./")
		if h.Typeflag == tar.TypeDir && (name == "" || name == ".") && !root && h.Uid == 0 && h.Gid == 0 && h.Mode == 0755 && h.Size == 0 {
			root = true
			continue
		}
		f, ok := expected[name]
		if !ok || seen[name] || h.Typeflag != tar.TypeReg || h.Linkname != "" || h.Uid != 0 || h.Gid != 0 || h.Mode != 0444 || h.Size != int64(len(f.body)) {
			return errPipxNativeInput
		}
		hash := sha256.New()
		if n, err := io.CopyN(hash, archive, h.Size); err != nil || n != h.Size || hex.EncodeToString(hash.Sum(nil)) != f.digest {
			return errPipxNativeInput
		}
		seen[name] = true
	}
	if len(seen) != len(expected) || bounded.N == 0 {
		return errPipxNativeInput
	}
	return drainUVTarPadding(bounded)
}

func (v *pipxWheelVolume) close(ctx context.Context, runner CommandRunner) error {
	if v == nil {
		return errPipxNativeInput
	}
	if err := closeWritableOutputVolume(ctx, runner, &v.volume, pipxNativeWheels); err != nil {
		return err
	}
	*v = pipxWheelVolume{}
	return nil
}
