package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type volumeInspectRunner struct {
	volume []byte
	mounts []byte
}

func TestClosureVolumeNameAcceptsOrdinarySessionCharacters(t *testing.T) {
	if name, err := closureVolumeName("sbx_xnr0"); err != nil || !strings.HasPrefix(name, "haa-closure-") {
		t.Fatalf("valid transaction characters rejected: name=%q err=%v", name, err)
	}
}

func (r volumeInspectRunner) Output(_ context.Context, _ string, arguments ...string) ([]byte, error) {
	if len(arguments) > 1 && arguments[0] == "volume" && arguments[1] == "inspect" {
		return r.volume, nil
	}
	if len(arguments) > 0 && arguments[0] == "inspect" {
		return r.mounts, nil
	}
	return nil, errors.New("unexpected Docker command")
}

func TestClosureVolumeRejectsSubstitutedMount(t *testing.T) {
	for _, goBuild := range []bool{false, true} {
		name := "python"
		if goBuild {
			name = "go-build"
		}
		t.Run(name, func(t *testing.T) { testClosureVolumeRejectsSubstitutedMount(t, goBuild) })
	}
}

func testClosureVolumeRejectsSubstitutedMount(t *testing.T, goBuild bool) {
	volume := closureVolume{name: "haa-closure-abc", transaction: "tx", manifestID: strings.Repeat("f", 64), createdAt: "2026-09-29T00:00:00Z", mountpoint: "/var/lib/docker/volumes/haa-closure-abc/_data", capacity: 1024, goBuild: goBuild}
	destination, options := pythonSitePath, "size=1024"+closureTmpfsOptions
	if goBuild {
		destination, options = "/tmp/haa-go-input", options+",noexec"
	}
	volumeJSON, err := json.Marshal(dockerVolumeInspection{
		Name: volume.name, Driver: "local", Scope: "local", Mountpoint: volume.mountpoint,
		CreatedAt: volume.createdAt,
		Options:   map[string]string{"type": "tmpfs", "device": "tmpfs", "o": options},
		Labels:    map[string]string{closureVolumeLabel: "tx", closureManifestLabel: volume.manifestID},
	})
	if err != nil {
		t.Fatal(err)
	}
	container := strings.Repeat("a", 64)
	configured := dockerHostMountInspection{Type: "volume", Source: volume.name, Target: destination, ReadOnly: true}
	configured.VolumeOptions.NoCopy = true
	good := dockerContainerMountInspection{ID: container,
		Mounts:     []dockerMountInspection{{Type: "volume", Name: volume.name, Driver: "local", Source: volume.mountpoint, Destination: destination, RW: false}},
		HostMounts: []dockerHostMountInspection{configured}}
	encode := func(value dockerContainerMountInspection) []byte {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	if err := volume.verifyContainerMount(context.Background(), volumeInspectRunner{volume: volumeJSON, mounts: encode(good)}, container, true); err != nil {
		t.Fatal(err)
	}
	bad := good
	bad.Mounts = []dockerMountInspection{{Type: "bind", Source: volume.mountpoint, Destination: destination, RW: false}}
	if err := volume.verifyContainerMount(context.Background(), volumeInspectRunner{volume: volumeJSON, mounts: encode(bad)}, container, true); err == nil {
		t.Fatal("host bind accepted")
	}
	bad = good
	bad.Mounts = []dockerMountInspection{{Type: "volume", Name: volume.name, Driver: "local", Source: volume.mountpoint, Destination: destination, RW: true}}
	if err := volume.verifyContainerMount(context.Background(), volumeInspectRunner{volume: volumeJSON, mounts: encode(bad)}, container, true); err == nil {
		t.Fatal("writable observation mount accepted")
	}
	bad = good
	bad.HostMounts = []dockerHostMountInspection{{Type: "volume", Source: volume.name, Target: destination, ReadOnly: true}}
	if err := volume.verifyContainerMount(context.Background(), volumeInspectRunner{volume: volumeJSON, mounts: encode(bad)}, container, true); err == nil {
		t.Fatal("missing no-copy rejected")
	}
	bad = good
	bad.HostMounts = append(append([]dockerHostMountInspection{}, good.HostMounts...), dockerHostMountInspection{Type: "bind", Source: "/home/host", Target: "/elsewhere"})
	if err := volume.verifyContainerMount(context.Background(), volumeInspectRunner{volume: volumeJSON, mounts: encode(bad)}, container, true); err == nil {
		t.Fatal("unrelated host bind accepted")
	}
	bad = good
	bad.Mounts = []dockerMountInspection{{Type: "volume", Name: volume.name, Driver: "local", Source: volume.mountpoint, Destination: "/wrong", RW: false}}
	if err := volume.verifyContainerMount(context.Background(), volumeInspectRunner{volume: volumeJSON, mounts: encode(bad)}, container, true); err == nil {
		t.Fatal("wrong mount destination accepted")
	}
	other := volume
	other.createdAt = "2026-09-29T01:00:00Z"
	if err := other.verifyContainerMount(context.Background(), volumeInspectRunner{volume: volumeJSON, mounts: encode(good)}, container, true); err == nil {
		t.Fatal("recreated volume accepted")
	}
}

func TestPythonClosureTopologyIsPhaseBoundAndRejectsUnknownProfiles(t *testing.T) {
	for _, profile := range []string{"pypi-wheel-pytorch-cpu", "pypi-wheel-pytorch-cu126", "pypi-wheel-pytorch-cu130", "pypi-wheel-pytorch-cu132"} {
		for _, phase := range []pythonClosurePhase{pythonClosurePreparation, pythonClosureAnchor, pythonClosureObservation} {
			topology, ok := pythonClosureExpectedTopology(profile, phase)
			if !ok {
				t.Fatalf("%s/%s rejected", profile, phase)
			}
			found := false
			for _, mount := range topology {
				if mount.Mountpoint != pythonSitePath {
					continue
				}
				found = true
				if mount.FSType != "9p" || mount.ReadOnly != (phase != pythonClosurePreparation) {
					t.Fatalf("%s/%s has wrong closure topology: %#v", profile, phase, mount)
				}
			}
			if !found {
				t.Fatalf("%s/%s lost /haa-site", profile, phase)
			}
		}
	}
	if _, ok := pythonClosureExpectedTopology("npm-lifecycle", pythonClosureObservation); ok {
		t.Fatal("npm profile accepted Python volume")
	}
	if _, ok := pythonClosureExpectedTopology("pypi-wheel", "unknown"); ok {
		t.Fatal("unknown phase accepted")
	}
}
