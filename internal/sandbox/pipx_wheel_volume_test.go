package sandbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type pipxWheelVolumeRunner struct {
	*pipxVolumeRunner
	archive              []byte
	copyError, readError error
	copies               int
}

func (r *pipxWheelVolumeRunner) RunInput(_ context.Context, input io.Reader, name string, args ...string) error {
	r.copies++
	if name != "docker" || len(args) != 4 || args[0] != "cp" || args[1] != "--archive" || args[2] != "-" || !strings.HasSuffix(args[3], ":"+pipxNativeWheels) {
		return errors.New("unexpected wheel copy")
	}
	if r.copyError != nil {
		return r.copyError
	}
	var err error
	r.archive, err = io.ReadAll(input)
	return err
}
func (r *pipxWheelVolumeRunner) RunOutput(_ context.Context, output io.Writer, name string, args ...string) error {
	if name != "docker" || len(args) != 3 || args[0] != "cp" || !strings.HasSuffix(args[1], ":"+pipxNativeWheels+"/.") || args[2] != "-" {
		return errors.New("unexpected wheel readback")
	}
	if r.readError != nil {
		return r.readError
	}
	_, err := io.Copy(output, bytes.NewReader(r.archive))
	return err
}
func pipxWheelVolumeFixture(t *testing.T) (*pipxWheelVolume, *pipxWheelVolumeRunner) {
	t.Helper()
	input, _, _, _, _ := pipxFrozenInputFixture(t, nil)
	transaction := "pipx-wheel-test-pipx-wheels"
	name, _ := closureVolumeName(transaction)
	v := closureVolume{name: name, transaction: transaction, manifestID: input.id, createdAt: "2026-10-28T00:00:00Z", mountpoint: "/var/lib/docker/volumes/" + name + "/_data", capacity: pipxWheelInputCapacity, projectBuildDestination: pipxNativeWheels}
	return &pipxWheelVolume{volume: v, input: input}, &pipxWheelVolumeRunner{pipxVolumeRunner: &pipxVolumeRunner{volume: v, mounts: map[string]dockerContainerMountInspection{}}}
}
func (r *pipxWheelVolumeRunner) attachWheel(id string, ro bool) {
	r.attach(id, ro)
	m := r.mounts[id]
	m.Mounts[0].Destination = pipxNativeWheels
	m.HostMounts[0].Target = pipxNativeWheels
	r.mounts[id] = m
}
func TestPipxWheelVolumeSealedLifetimeAndFixedCapacity(t *testing.T) {
	ctx := context.Background()
	v, r := pipxWheelVolumeFixture(t)
	created, err := createPipxWheelVolume(ctx, r, "pipx-wheel-test", v.input)
	if err != nil {
		t.Fatal(err)
	}
	v = created
	rw, err := v.preparationMount()
	if err != nil || strings.Contains(rw, "readonly") {
		t.Fatal(rw, err)
	}
	if _, err := v.readerMount(); err == nil {
		t.Fatal("unseeded reader")
	}
	if got := v.volume.tmpfsOptions(); got != "size=68157440,nosuid,nodev,uid=0,gid=0,mode=0755,noexec" {
		t.Fatal(got)
	}
	for _, size := range []int64{0, pipxWheelInputCapacity - 1, pipxWheelInputCapacity + 1} {
		if _, err := createInputVolume(ctx, r, "bad", v.input.id, size, false, projectBuildVolumeConfig{destination: pipxNativeWheels}); err == nil {
			t.Fatal("capacity override")
		}
	}
	writer, anchor, reader := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	r.attachWheel(writer, false)
	if err := v.seed(ctx, r, writer); err != nil {
		t.Fatal(err)
	}
	if _, err := v.preparationMount(); err == nil {
		t.Fatal("second preparation mount")
	}
	if err := v.seed(ctx, r, writer); err == nil || r.copies != 1 {
		t.Fatal("reseed")
	}
	ro, err := v.readerMount()
	if err != nil || ro != rw+",readonly" {
		t.Fatal(ro, err)
	}
	r.attachWheel(anchor, true)
	if err := v.bindAnchor(ctx, r, anchor); err != nil {
		t.Fatal(err)
	}
	if err := v.seal(ctx, r); err == nil || v.sealed {
		t.Fatal("live writer sealed")
	}
	r.inventory = []string{anchor}
	if err := v.seal(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.attachWheel(reader, true)
	if err := v.verifyReader(ctx, r, reader); err != nil {
		t.Fatal(err)
	}
	foreign := strings.Repeat("d", 64)
	r.attachWheel(foreign, false)
	if err := v.verifyReader(ctx, r, reader); err == nil {
		t.Fatal("foreign writer")
	}
	r.inventory = []string{anchor, reader}
	if err := v.verifyReader(ctx, r, reader); err != nil {
		t.Fatal(err)
	}
	r.fail = "ps "
	if err := v.verifyReader(ctx, r, reader); err == nil {
		t.Fatal("unknown inventory")
	}
	if err := v.close(ctx, r); err == nil || r.removed {
		t.Fatal("unknown cleanup")
	}
	r.fail = ""
	if err := v.close(ctx, r); err == nil || r.removed {
		t.Fatal("attached cleanup")
	}
	r.inventory = nil
	if err := v.close(ctx, r); err != nil || !r.removed || v.valid() {
		t.Fatal("idle cleanup", err)
	}
}
func TestPipxWheelVolumeNeverAdoptsPartialCopyOrReadback(t *testing.T) {
	for _, phase := range []string{"copy", "readback", "changed"} {
		t.Run(phase, func(t *testing.T) {
			v, r := pipxWheelVolumeFixture(t)
			ctx := context.Background()
			writer := strings.Repeat("a", 64)
			r.attachWheel(writer, false)
			cause := errors.New("primary transport error")
			if phase == "copy" {
				r.copyError = cause
			} else if phase == "readback" {
				r.readError = cause
			} else {
				v.input.files[0].body[0] ^= 1
			}
			err := v.seed(ctx, r, writer)
			if err == nil || v.seeded || v.sealed {
				t.Fatal("partial input adopted")
			}
			if phase != "changed" && !errors.Is(err, cause) {
				t.Fatal("transport cause lost", err)
			}
			r.copyError = nil
			r.readError = nil
			if err := v.seed(ctx, r, writer); err == nil {
				t.Fatal("partial input retry")
			}
			if _, err := v.readerMount(); err == nil {
				t.Fatal("partial input reader")
			}
		})
	}
}
func TestPipxWheelVolumeRejectsShadowAliasBindAndChangedReadback(t *testing.T) {
	ctx := context.Background()
	id := strings.Repeat("a", 64)
	for _, target := range []string{"/", pipxNativeWheels + "/shadow", "/alias"} {
		for _, configured := range []bool{false, true} {
			v, r := pipxWheelVolumeFixture(t)
			r.attachWheel(id, true)
			m := r.mounts[id]
			if configured {
				m.HostMounts = append(m.HostMounts, dockerHostMountInspection{Type: "volume", Source: v.volume.name, Target: target})
			} else {
				m.Mounts = append(m.Mounts, dockerMountInspection{Type: "volume", Name: v.volume.name, Destination: target})
			}
			r.mounts[id] = m
			if err := v.verifyAttachments(ctx, r, map[string]bool{id: true}); err == nil {
				t.Fatal("shadow/alias", target, configured)
			}
		}
	}
	v, r := pipxWheelVolumeFixture(t)
	r.attachWheel(id, true)
	m := r.mounts[id]
	m.Binds = []string{"/host:/guest:ro"}
	r.mounts[id] = m
	if err := v.verifyAttachments(ctx, r, map[string]bool{id: true}); err == nil {
		t.Fatal("bind")
	}
	v, r = pipxWheelVolumeFixture(t)
	r.attachWheel(id, false)
	if err := v.seed(ctx, r, id); err != nil {
		t.Fatal(err)
	}
	r.archive[512] ^= 1
	if err := v.readback(ctx, r, id); err == nil {
		t.Fatal("changed original readback")
	}
}
