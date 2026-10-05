package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestGoBuildMaterializationRequiresCompleteContentAndFixedOwnership(t *testing.T) {
	hash := sha256.Sum256([]byte("approved"))
	manifest := goBuildInputManifest{members: map[string]goBuildInputMember{"project/main.go": {kind: tar.TypeReg, size: 8, sha256: hex.EncodeToString(hash[:])}}}
	for _, failure := range []string{"", "content", "missing", "extra", "writable", "owner", "link", "duplicate", "root"} {
		t.Run(failure, func(t *testing.T) {
			var body bytes.Buffer
			writer := tar.NewWriter(&body)
			add := func(header *tar.Header, data string) {
				if err := writer.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write([]byte(data)); err != nil {
					t.Fatal(err)
				}
			}
			for _, root := range []string{".", "project", "cache"} {
				mode := int64(0o500)
				if root == "." {
					mode = 0o700
				}
				if root == "cache" && failure == "root" {
					continue
				}
				add(&tar.Header{Name: root, Typeflag: tar.TypeDir, Mode: mode, Uid: 1000, Gid: 1000}, "")
			}
			if failure != "missing" {
				header := &tar.Header{Name: "project/main.go", Typeflag: tar.TypeReg, Mode: 0o400, Size: 8, Uid: 1000, Gid: 1000}
				data := "approved"
				switch failure {
				case "content":
					data = "poisoned"
				case "writable":
					header.Mode = 0o600
				case "owner":
					header.Uid = 0
				case "link":
					header.Typeflag, header.Linkname, header.Size, data = tar.TypeLink, "project/other", 0, ""
				}
				add(header, data)
				if failure == "duplicate" {
					add(header, data)
				}
			}
			if failure == "extra" {
				add(&tar.Header{Name: "cache/extra", Typeflag: tar.TypeReg, Mode: 0o400, Uid: 1000, Gid: 1000, Size: 6}, "poison")
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			err := scanGoBuildMaterialization(context.Background(), &body, manifest)
			if (err == nil) != (failure == "") {
				t.Fatalf("materialization %s: %v", failure, err)
			}
		})
	}
}

func TestGoBuildOutputMountCannotSubstituteReadOnlyInput(t *testing.T) {
	input := closureVolume{name: "haa-input", transaction: "tx", createdAt: "created", mountpoint: "/volume/input", goBuild: true}
	output := closureVolume{name: "haa-output", transaction: "tx-output", createdAt: "created", mountpoint: "/volume/output", goBuild: true, goBuildOutput: true}
	for _, readOnly := range []bool{false, true} {
		arguments, err := goBuildVolumesCreateArguments(input, &output, readOnly)
		if err != nil {
			t.Fatal(err)
		}
		var mounts []string
		for i, arg := range arguments {
			if arg == "--mount" {
				mounts = append(mounts, arguments[i+1])
			}
		}
		inputMount, _ := input.mountArgument(readOnly)
		outputMount, _ := output.mountArgument(false)
		if !reflect.DeepEqual(mounts, []string{inputMount, outputMount}) || strings.Contains(outputMount, "readonly") {
			t.Fatal("output changed fixed input/output permissions")
		}
		topology, ok := goBuildOutputExpectedTopology(readOnly)
		if !ok || len(topology) != 5 {
			t.Fatal("input/output topology is incomplete")
		}
		inputFound, outputFound := false, false
		for _, mount := range topology {
			if mount.Mountpoint == goBuildGuestInput {
				inputFound = true
				if mount.ReadOnly != readOnly {
					t.Fatal("input lost phase-bound read-only setting")
				}
			}
			if mount.Mountpoint == goBuildGuestOutput {
				outputFound = true
				if mount.ReadOnly || mount.Class != "workspace" || mount.Parent != "/tmp" || mount.FSType != "9p" {
					t.Fatal("output topology drifted")
				}
			}
		}
		if !inputFound || !outputFound {
			t.Fatal("input/output mount missing")
		}
	}
	for _, failure := range []string{"input-role", "output-role", "alias", "transaction", "identity"} {
		t.Run(failure, func(t *testing.T) {
			i, o := input, output
			switch failure {
			case "input-role":
				i.goBuildOutput = true
			case "output-role":
				o.goBuildOutput = false
			case "alias":
				o.name = i.name
			case "transaction":
				o.transaction = "foreign-output"
			case "identity":
				o.createdAt = ""
			}
			if _, err := goBuildVolumesCreateArguments(i, &o, true); err == nil {
				t.Fatal("substituted output volume accepted")
			}
		})
	}
}

func TestGoBuildInputTopologyKeepsReadOnlyInputSeparateFromScratch(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		topology, ok := goBuildInputExpectedTopology(readonly)
		if !ok || len(topology) != 4 {
			t.Fatal("go input topology is incomplete")
		}
		for _, mount := range topology {
			switch mount.Mountpoint {
			case "/tmp/haa-go-input":
				if mount.Class != "workspace" || mount.FSType != "9p" || mount.Parent != "/tmp" || mount.ReadOnly != readonly || mount.NoExec {
					t.Fatal("go input guest topology drifted")
				}
			case "/tmp":
				if mount.ReadOnly || !mount.NoExec {
					t.Fatal("go scratch topology drifted")
				}
			case "/":
				if !mount.ReadOnly {
					t.Fatal("go runtime is writable")
				}
			}
		}
	}
	if _, err := newObservationTraceLedger("artifact-selected"); err == nil {
		t.Fatal("unknown profile accepted a phase budget")
	}
}
