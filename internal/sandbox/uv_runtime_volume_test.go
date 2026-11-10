package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type uvVolumeFake struct {
	volume               dockerVolumeInspection
	container            dockerContainerMountInspection
	listed               string
	fail, stale, removed bool
}

func (f *uvVolumeFake) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if ctx.Err() != nil || name != "docker" || len(args) < 2 {
		return nil, errors.New("invalid request")
	}
	if args[0] == "volume" {
		switch args[1] {
		case "ls":
			if f.fail {
				return nil, errors.New("inventory unavailable")
			}
			if f.stale {
				return []byte("existing"), nil
			}
			return nil, nil
		case "create":
			v := dockerVolumeInspection{Name: args[len(args)-1], Driver: "local", Scope: "local", Mountpoint: "/owned", CreatedAt: "fixed", Options: map[string]string{"type": "tmpfs", "device": "tmpfs", "o": uvRuntimeVolumeOptions}, Labels: map[string]string{}}
			for _, a := range args {
				if options, ok := strings.CutPrefix(a, "o="); ok {
					v.Options["o"] = options
				}
				if label, value, ok := strings.Cut(a, "="); ok && strings.HasPrefix(label, "io.heliopause.") {
					v.Labels[label] = value
				}
			}
			f.volume = v
			return []byte(v.Name), nil
		case "inspect":
			if f.fail {
				return nil, errors.New("identity unavailable")
			}
			return json.Marshal(f.volume)
		case "rm":
			f.removed = true
			return []byte(f.volume.Name), nil
		}
	}
	if args[0] == "ps" {
		if f.fail {
			return nil, errors.New("attachment query unavailable")
		}
		return []byte(f.listed), nil
	}
	if args[0] == "inspect" {
		return json.Marshal(f.container)
	}
	return nil, errors.New("unexpected request")
}

func nativeVolumeFixture(t *testing.T, kind nativeRuntimeToolKind) (*uvVolumeFake, nativeRuntimeVolume, string) {
	t.Helper()
	f := &uvVolumeFake{}
	v, err := createNativeRuntimeVolume(context.Background(), f, kind)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 64)
	f.listed = id
	f.container.ID = id
	f.container.Mounts = []dockerMountInspection{{Type: "volume", Name: v.name, Source: v.mountpoint, Destination: "/haa-runtime", Driver: "local", RW: true}}
	f.container.HostMounts = []dockerHostMountInspection{{Type: "volume", Source: v.name, Target: "/haa-runtime"}}
	f.container.HostMounts[0].VolumeOptions.NoCopy = true
	return f, v, id
}

func TestUVRuntimeVolumeRejectsSubstitutionAndUnknownCleanup(t *testing.T) {
	verifyNativeVolumeBoundary(t, nativeRuntimeUV)
}

func TestPNPMRuntimeVolumeRejectsSubstitutionAndUnknownCleanup(t *testing.T) {
	verifyNativeVolumeBoundary(t, nativeRuntimePNPM)
}

func TestBunRuntimeVolumeRejectsSubstitutionAndUnknownCleanup(t *testing.T) {
	verifyNativeVolumeBoundary(t, nativeRuntimeBun)
}

func verifyNativeVolumeBoundary(t *testing.T, kind nativeRuntimeToolKind) {
	t.Helper()
	spec, known := nativeRuntimeToolIdentity(kind)
	if !known {
		t.Fatal("unknown fixture tool")
	}
	for _, fault := range []string{"none", "driver", "scope", "options", "owner", "hash", "generation", "source", "missing", "foreign", "duplicate", "mount-source", "read-only", "copy-up", "bind", "shadow", "query"} {
		t.Run(fault, func(t *testing.T) {
			f, v, id := nativeVolumeFixture(t, kind)
			switch fault {
			case "driver":
				f.volume.Driver = "other"
			case "scope":
				f.volume.Scope = "global"
			case "options":
				f.volume.Options["o"] = "size=67108864,uid=1000"
			case "owner":
				f.volume.Labels[spec.volumeLabel] = "foreign"
			case "hash":
				f.volume.Labels[spec.hashLabel] = strings.Repeat("0", 64)
			case "generation":
				f.volume.CreatedAt = "replaced"
			case "source":
				f.volume.Mountpoint = "/other"
			case "missing":
				f.listed = ""
			case "foreign":
				f.listed = id + "\n" + strings.Repeat("b", 64)
			case "duplicate":
				f.container.Mounts = append(f.container.Mounts, f.container.Mounts[0])
			case "mount-source":
				f.container.Mounts[0].Source = "/foreign"
			case "read-only":
				f.container.Mounts[0].RW = false
			case "copy-up":
				f.container.HostMounts[0].VolumeOptions.NoCopy = false
			case "bind":
				f.container.Binds = []string{"/host:/haa-runtime"}
			case "shadow":
				f.container.HostMounts = append(f.container.HostMounts, dockerHostMountInspection{Type: "volume", Source: "foreign", Target: "/haa-runtime/uv"})
			case "query":
				f.fail = true
			}
			if err := v.verifyTarget(context.Background(), f, id); (err == nil) != (fault == "none") {
				t.Fatal("target attribution disagrees", err)
			}
			if fault != "missing" {
				if err := v.close(context.Background(), f); err == nil || f.removed {
					t.Fatal("attached/unknown volume removed")
				}
			}
		})
	}
	f, v, _ := nativeVolumeFixture(t, kind)
	f.listed = ""
	if err := v.close(context.Background(), f); err != nil || !f.removed {
		t.Fatal("owned unattached volume cleanup failed", err)
	}
	for _, f := range []*uvVolumeFake{{fail: true}, {stale: true}} {
		if _, err := createNativeRuntimeVolume(context.Background(), f, kind); err == nil || f.removed {
			t.Fatal("unknown/existing volume adopted")
		}
	}
}
