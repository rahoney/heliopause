package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestSemgrepTransportBindsSelectedRolesAndRejectsCallerIdentity(t *testing.T) {
	pin := runtimeidentity.SelectedSemgrepSupply()
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeSemgrep, nativeRuntimeSemgrepRules} {
		spec, known := nativeRuntimeToolIdentity(kind)
		selected, name, mode := pin.Tool, pin.Tool.Path, int64(0555)
		if kind == nativeRuntimeSemgrepRules {
			selected, name, mode = pin.Rules, runtimeidentity.SemgrepRulesAssetName, 0444
		}
		if !known || spec.installedName != name || spec.guestPath != "/haa-semgrep/"+name || spec.size != selected.Size || spec.sha256 != selected.SHA256 || spec.fileMode() != mode || spec.maxBytes != runtimeidentity.SemgrepAssetMaximum(name) {
			t.Fatal("fixed role drift", kind)
		}
		identity := nativeToolFileIdentity{spec.size, spec.sha256, mode}
		if !validNativeTransportIdentity(identity, kind) {
			t.Fatal("selected identity rejected")
		}
		for _, fault := range []string{"hash", "size", "mode"} {
			bad := identity
			switch fault {
			case "hash":
				bad.sha256 = strings.Repeat("0", 64)
			case "size":
				bad.size--
			case "mode":
				bad.mode ^= 0111
			}
			runner := &uvSupplyFake{}
			if _, err := readNativeToolFromContainer(context.Background(), runner, strings.Repeat("b", 64), spec.guestPath, bad, kind); err == nil || len(runner.calls) != 0 {
				t.Fatal("caller identity reached Docker", kind, fault)
			}
		}
		runner := &uvVolumeFake{}
		if _, err := createNativeRuntimeVolume(context.Background(), runner, kind); err == nil || runner.volume.Name != "" {
			t.Fatal("scanner acquired mutable native volume")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if body, err := readInstalledNativeTool(ctx, filepath.Join(t.TempDir(), "absent"), kind); body != nil || !errors.Is(err, context.Canceled) {
			t.Fatal("cancelled supply returned bytes", err)
		}
	}
}

func TestSemgrepRulesTarAuthenticatesBytesAndPassiveFileMetadata(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", runtimeidentity.SelectedSemgrepSupply().Rules.Path))
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := nativeRuntimeToolIdentity(nativeRuntimeSemgrepRules)
	identity := nativeToolFileIdentity{spec.size, spec.sha256, spec.fileMode()}
	for _, fault := range []string{"none", "name", "mode", "owner", "group", "link", "changed", "duplicate", "truncated", "trailing"} {
		t.Run(fault, func(t *testing.T) {
			var stream bytes.Buffer
			archive := tar.NewWriter(&stream)
			header := &tar.Header{Name: spec.name, Typeflag: tar.TypeReg, Mode: 0444, Size: int64(len(body))}
			switch fault {
			case "name":
				header.Name = "../" + spec.name
			case "mode":
				header.Mode = 0555
			case "owner":
				header.Uid = 1000
			case "group":
				header.Gid = 1000
			case "link":
				header.Typeflag = tar.TypeSymlink
				header.Linkname = "/elsewhere"
				header.Size = 0
			}
			count := 1
			if fault == "duplicate" {
				count = 2
			}
			for index := 0; index < count; index++ {
				if err := archive.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
				if header.Typeflag == tar.TypeReg {
					data := bytes.Clone(body)
					if fault == "changed" {
						data[0] ^= 1
					}
					if _, err := archive.Write(data); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			data := stream.Bytes()
			if fault == "truncated" {
				data = data[:512+len(body)-1]
			}
			if fault == "trailing" {
				data = append(data, '!')
			}
			actual, err := scanNativeToolTar(data, identity, nativeRuntimeSemgrepRules)
			if fault == "none" {
				if err != nil || !bytes.Equal(actual, body) {
					t.Fatal("selected rules rejected", err)
				}
			} else if err == nil {
				t.Fatal("foreign rules transport accepted")
			}
		})
	}
}

func TestSemgrepReadonlySupplyReusesSealingAndUnknownCleanup(t *testing.T) {
	transaction := "semgrep-supply-test-semgrep-supply"
	name, _ := closureVolumeName(transaction)
	volume := closureVolume{name: name, transaction: transaction, manifestID: semgrepSupplyDigest(), createdAt: "2026-11-04T00:00:00Z", mountpoint: "/var/lib/docker/volumes/" + name + "/_data", capacity: semgrepSupplyVolumeCapacity, projectBuildDestination: semgrepRuntimeRoot, projectBuildExecutable: true}
	runner := &pipxVolumeRunner{volume: volume, mounts: make(map[string]dockerContainerMountInspection)}
	ctx := context.Background()
	v, err := createSemgrepSupplyVolume(ctx, runner, "semgrep-supply-test")
	if err != nil || !v.valid() {
		t.Fatal("fixed scanner supply refused", err)
	}
	if v.volume.tmpfsOptions() != "size=269484032,nosuid,nodev,uid=0,gid=0,mode=0755" {
		t.Fatal("tool supply topology changed", v.volume.tmpfsOptions())
	}
	preparation, err := v.preparationMount()
	if err != nil || strings.Contains(preparation, "readonly") {
		t.Fatal("preparation mount", err)
	}
	if _, err := v.readerMount(); err == nil {
		t.Fatal("unseeded input admitted")
	}
	if err := v.seal(ctx, runner); err == nil {
		t.Fatal("unprepared input sealed")
	}
	for _, fault := range []string{"capacity", "pin", "noexec"} {
		candidate := volume
		switch fault {
		case "capacity":
			candidate.capacity--
		case "pin":
			candidate.manifestID = strings.Repeat("a", 64)
		case "noexec":
			candidate.projectBuildExecutable = false
		}
		if _, err := createInputVolume(ctx, runner, "bad", candidate.manifestID, candidate.capacity, false, projectBuildVolumeConfig{destination: semgrepRuntimeRoot, executable: candidate.projectBuildExecutable}); err == nil {
			t.Fatal("supply override admitted", fault)
		}
	}
	// Structural mount tests do not assert actual scanner bytes or admission.
	v.seeded = true
	readOnly, err := v.readerMount()
	if err != nil || readOnly != preparation+",readonly" {
		t.Fatal("mutable reader", err)
	}
	id := strings.Repeat("b", 64)
	runner.attach(id, true)
	if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{id: true}); err != nil {
		t.Fatal(err)
	}
	original := runner.mounts[id]
	for _, destination := range []string{"/", semgrepRuntimeRoot + "/nested", "/alias"} {
		changed := original
		changed.Mounts = append(append([]dockerMountInspection{}, original.Mounts...), dockerMountInspection{Type: "volume", Name: volume.name, Destination: destination})
		runner.mounts[id] = changed
		if err := v.volume.verifyAttachments(ctx, runner, map[string]bool{id: true}); err == nil {
			t.Fatal("shadow or alias mount admitted", destination)
		}
	}
	runner.mounts[id] = original
	runner.fail = "volume inspect"
	if err := v.close(ctx, runner); err == nil || runner.removed {
		t.Fatal("UNKNOWN supply deleted")
	}
	candidate, err := createSemgrepSupplyVolume(ctx, runner, "semgrep-supply-test")
	if err == nil || candidate == nil || runner.removed {
		t.Fatal("uncertain CREATE discarded or deleted its identity", err)
	}
	runner.fail = ""
	runner.inventory = nil
	if err := v.close(ctx, runner); err != nil || !runner.removed {
		t.Fatal("exact idle supply not cleaned", err)
	}
}

func TestSemgrepUnpreparedSupplyNeverReachesDocker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if prepared, err := readInstalledSemgrepTool(ctx, filepath.Join(t.TempDir(), "absent")); prepared != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled source wrapper returned bytes", err)
	}
	for _, tool := range []*semgrepPreparedTool{nil, {}, {core: []byte("changed"), rules: []byte("changed")}} {
		runner := &uvSupplyFake{}
		if err := tool.introduce(context.Background(), runner, strings.Repeat("b", 64)); err == nil || len(runner.calls) != 0 {
			t.Fatal("unprepared supply reached Docker")
		}
	}
}
