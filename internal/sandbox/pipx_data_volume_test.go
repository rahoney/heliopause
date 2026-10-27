package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pipxVolumeRunner struct {
	volume    closureVolume
	mounts    map[string]dockerContainerMountInspection
	inventory []string
	fail      string
	removed   bool
}

func (r *pipxVolumeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if name != "docker" || len(args) == 0 {
		return nil, errors.New("unexpected command")
	}
	command := strings.Join(args, " ")
	if r.fail != "" && strings.Contains(command, r.fail) {
		return nil, errors.New("query unavailable")
	}
	switch {
	case args[0] == "volume" && args[1] == "ls":
		return nil, nil
	case args[0] == "volume" && args[1] == "create":
		return []byte(args[len(args)-1]), nil
	case args[0] == "volume" && args[1] == "inspect":
		v := r.volume
		return json.Marshal(dockerVolumeInspection{Name: v.name, Driver: "local", Scope: "local", Mountpoint: v.mountpoint, CreatedAt: v.createdAt,
			Options: map[string]string{"type": "tmpfs", "device": "tmpfs", "o": v.tmpfsOptions()}, Labels: map[string]string{closureVolumeLabel: v.transaction, closureManifestLabel: v.manifestID}})
	case args[0] == "volume" && args[1] == "rm":
		r.removed = true
		return []byte(r.volume.name), nil
	case args[0] == "ps":
		return []byte(strings.Join(r.inventory, "\n")), nil
	case args[0] == "inspect" && args[2] == containerMountInspectTemplate:
		return json.Marshal(r.mounts[args[3]])
	case args[0] == "inspect":
		return []byte(args[3] + "|true|42"), nil
	default:
		return nil, errors.New("unexpected command " + command)
	}
}

func pipxVolumeFixture() (*pipxDataVolume, *pipxVolumeRunner) {
	transaction := "pipx-data-test-pipx-data"
	name, _ := closureVolumeName(transaction)
	v := closureVolume{name: name, transaction: transaction, manifestID: runtimeidentity.PipxToolSHA256,
		createdAt: "2026-10-27T00:00:00Z", mountpoint: "/var/lib/docker/volumes/" + name + "/_data",
		capacity: pipxDataVolumeCapacity, projectBuildDestination: pipxRuntimeRoot}
	return &pipxDataVolume{volume: v}, &pipxVolumeRunner{volume: v, mounts: make(map[string]dockerContainerMountInspection)}
}
func (r *pipxVolumeRunner) attach(id string, readOnly bool) {
	host := dockerHostMountInspection{Type: "volume", Source: r.volume.name, Target: pipxRuntimeRoot, ReadOnly: readOnly}
	host.VolumeOptions.NoCopy = true
	r.mounts[id] = dockerContainerMountInspection{ID: id, Mounts: []dockerMountInspection{{Type: "volume", Name: r.volume.name, Source: r.volume.mountpoint, Destination: pipxRuntimeRoot, Driver: "local", RW: !readOnly}}, HostMounts: []dockerHostMountInspection{host}}
	r.inventory = append(r.inventory, id)
}

func TestPipxDataVolumeFixedCreationAndModes(t *testing.T) {
	ctx := context.Background()
	_, r := pipxVolumeFixture()
	v, err := createPipxDataVolume(ctx, r, "pipx-data-test")
	if err != nil {
		t.Fatal(err)
	}
	rw, err := v.preparationMount()
	if err != nil || strings.Contains(rw, "readonly") {
		t.Fatal(rw, err)
	}
	if _, err := v.readerMount(); err == nil {
		t.Fatal("unseeded reader accepted")
	}
	v.seeded = true
	ro, err := v.readerMount()
	if err != nil || ro != rw+",readonly" {
		t.Fatal(ro, err)
	}
	if got := v.volume.tmpfsOptions(); got != "size=8388608,nosuid,nodev,uid=0,gid=0,mode=0755,noexec" {
		t.Fatal(got)
	}
	for _, capacity := range []int64{0, 1024, pipxDataVolumeCapacity + 1} {
		if _, err := createInputVolume(ctx, r, "bad", runtimeidentity.PipxToolSHA256, capacity, false, projectBuildVolumeConfig{destination: pipxRuntimeRoot}); err == nil {
			t.Fatal("capacity override")
		}
	}
	if _, err := createInputVolume(ctx, r, "bad", strings.Repeat("a", 64), pipxDataVolumeCapacity, false, projectBuildVolumeConfig{destination: pipxRuntimeRoot}); err == nil {
		t.Fatal("pin override")
	}
	r.fail = "volume inspect"
	r.removed = false
	uncertain, err := createPipxDataVolume(ctx, r, "pipx-data-test")
	if err == nil || uncertain == nil || r.removed {
		t.Fatal("uncertain volume removed or adopted")
	}
	if err := uncertain.close(ctx, r); err == nil || r.removed {
		t.Fatal("unknown cleanup accepted")
	}
}

func TestPipxDataVolumeRejectsShadowAndAliasMounts(t *testing.T) {
	id := strings.Repeat("a", 64)
	ctx := context.Background()
	for _, target := range []string{"/", pipxRuntimeRoot + "/pipx-runtime.zip", pipxRuntimeRoot + "/nested", "/alias"} {
		for _, configured := range []bool{false, true} {
			t.Run(target+map[bool]string{true: "/configured", false: "/actual"}[configured], func(t *testing.T) {
				v, r := pipxVolumeFixture()
				r.attach(id, true)
				if err := v.volume.verifyAttachments(ctx, r, map[string]bool{id: true}); err != nil {
					t.Fatal(err)
				}
				m := r.mounts[id]
				if configured {
					m.HostMounts = append(m.HostMounts, dockerHostMountInspection{Type: "volume", Source: v.volume.name, Target: target})
				} else {
					m.Mounts = append(m.Mounts, dockerMountInspection{Type: "volume", Name: v.volume.name, Destination: target})
				}
				r.mounts[id] = m
				if err := v.volume.verifyAttachments(ctx, r, map[string]bool{id: true}); err == nil {
					t.Fatal("shadow/alias accepted")
				}
			})
		}
	}
}

func TestPipxDataVolumeRejectsIncompleteLifecycleAndUnknownCleanup(t *testing.T) {
	ctx := context.Background()
	writer := strings.Repeat("a", 64)
	anchor := strings.Repeat("b", 64)
	v, r := pipxVolumeFixture()
	if err := v.seal(ctx, r); err == nil {
		t.Fatal("unseeded seal")
	}
	if err := v.bindAnchor(ctx, r, anchor); err == nil {
		t.Fatal("unseeded anchor")
	}
	if err := v.verifyReader(ctx, r, anchor); err == nil {
		t.Fatal("unsealed read")
	}
	r.attach(writer, false)
	// Invalid bytes fail before streaming; the write attempt remains consumed.
	if err := v.seed(ctx, r, &pipxPreparedTool{body: []byte("wrong")}, writer); err == nil {
		t.Fatal("wrong supply")
	}
	if v.writer != writer || v.seeded {
		t.Fatal("partial preparation adopted")
	}
	if err := v.seed(ctx, r, &pipxPreparedTool{}, writer); err == nil {
		t.Fatal("partial supply retry")
	}
	if _, err := v.preparationMount(); err == nil {
		t.Fatal("second writer mount")
	}
	v.seeded = true
	v.anchor = anchor
	r.attach(anchor, true)
	if err := v.seal(ctx, r); err == nil || v.sealed {
		t.Fatal("attached writer accepted")
	}
	r.inventory = []string{anchor}
	r.fail = "ps "
	if err := v.seal(ctx, r); err == nil || v.sealed {
		t.Fatal("unknown absence accepted")
	}
	if err := v.close(ctx, r); err == nil || r.removed {
		t.Fatal("unknown cleanup")
	}
	r.fail = ""
	if err := v.close(ctx, r); err == nil || r.removed {
		t.Fatal("live attachment cleanup")
	}
	r.inventory = nil
	if err := v.close(ctx, r); err != nil || !r.removed || v.valid() {
		t.Fatal("empty cleanup", err)
	}
}
