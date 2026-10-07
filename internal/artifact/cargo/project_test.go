package cargo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCargoSelectedManifestsFollowWorkspaceAndPathDependencies(t *testing.T) {
	root := []byte("[workspace]\nmembers=['crates/**']\nexclude=['crates/excluded']\n")
	member := []byte("[package]\nname='member'\nversion='0.1.0'\n[dev-dependencies]\nlocal={path='../../../local'}\n")
	local := []byte("[package]\nname='local'\nversion='0.1.0'\n")
	files := map[string][]byte{
		"Cargo.toml": root, "crates/nested/member/Cargo.toml": member, "local/Cargo.toml": local,
		"crates/excluded/Cargo.toml": []byte("invalid [["), "tests/fixture/Cargo.toml": []byte("invalid [["),
	}
	selected, err := SelectedLocalManifests(files)
	if err != nil || !reflect.DeepEqual(selected, []string{"Cargo.toml", "crates/nested/member/Cargo.toml", "local/Cargo.toml"}) {
		t.Fatalf("selected local manifests = %v, %v", selected, err)
	}
	for _, body := range [][]byte{[]byte("invalid [["), []byte("[package]\nname='local'\nversion='0.1.0'\n[build-dependencies]\nevil={git='https://example.invalid/repo'}\n")} {
		files["local/Cargo.toml"] = body
		if _, err := SelectedLocalManifests(files); err == nil {
			t.Fatal("selected invalid/source-substituted dependency accepted")
		}
	}
	delete(files, "local/Cargo.toml")
	if _, err := SelectedLocalManifests(files); err == nil {
		t.Fatal("missing local dependency accepted")
	}
}

func TestCargoMemberExpansionHasFiniteWorkBound(t *testing.T) {
	files := map[string][]byte{}
	for index := 0; index < 4096; index++ {
		files[fmt.Sprintf("crates/member-%d/Cargo.toml", index)] = []byte("unselected data")
	}
	files["Cargo.toml"] = []byte("[workspace]\nmembers=[" + strings.Repeat("'absent/*',", 4096) + "]\n")
	if _, err := SelectedLocalManifests(files); err == nil || !strings.Contains(err.Error(), "work bound") {
		t.Fatalf("expansion work was not bounded: %v", err)
	}
}

func TestCargoManifestRejectsSourceSubstitutionAndExternalPaths(t *testing.T) {
	base := "[package]\nname=\"fixture\"\nversion=\"1.2.3\"\n"
	for _, suffix := range []string{
		"[dependencies]\nnormal=\"1\"\n",
		"[dependencies]\nlocal={path=\"nested\"}\n",
		"[dependencies]\nrenamed={package=\"normal\",version=\"1\",registry=\"crates-io\"}\n",
		"[target.'cfg(unix)'.dependencies]\nnormal={version=\"1\"}\n",
		"[workspace]\nmembers=[\"members/*\"]\n[workspace.dependencies]\nnormal=\"1\"\n",
		"[package.metadata]\ngit=\"opaque package metadata\"\n",
		"[lib]\npath=\"source/library.rs\"\n[[bin]]\nname=\"tool\"\npath=\"source/tool.rs\"\n",
	} {
		if err := ValidateProjectManifest([]byte(base+suffix), "Cargo.toml"); err != nil {
			t.Fatalf("normal manifest rejected: %v", err)
		}
	}
	for _, suffix := range []string{
		"[dependencies]\nx={git=\"https://example.invalid/repo\"}\n",
		"[dependencies]\nx={registry=\"private\",version=\"1\"}\n",
		"[dependencies]\nx={registry-index=\"https://index.crates.io/\",version=\"1\"}\n",
		"[dev-dependencies]\nx={path=\"../outside\"}\n",
		"[build-dependencies]\nx={path=\"/tmp/outside\"}\n",
		"[target.'cfg(unix)'.dependencies]\nx={git=\"https://example.invalid/repo\"}\n",
		"[workspace]\nmembers=[\"../outside\"]\n",
		"[workspace.dependencies]\nx={path=\"../outside\"}\n",
		"[patch.crates-io]\nx={path=\"inside\"}\n",
		"[replace]\n'x:1.2.3'={path=\"inside\"}\n",
		"[lib]\npath=\"../../outside.rs\"\n",
		"[[bin]]\nname=\"tool\"\npath=\"/tmp/outside.rs\"\n",
		"[[test]]\nname=\"case\"\npath=\"../outside.rs\"\n",
		"[[example]]\nname=\"demo\"\npath=\"../outside.rs\"\n",
		"[[bench]]\nname=\"benchmark\"\npath=\"../outside.rs\"\n",
	} {
		if ValidateProjectManifest([]byte(base+suffix), "Cargo.toml") == nil {
			t.Fatal("unsafe manifest accepted")
		}
	}
	for _, field := range []string{"build=\"../outside.rs\"\n", "workspace=\"../outside\"\n", "build=\"C:/outside.rs\"\n"} {
		if ValidateProjectManifest([]byte(base+field), "Cargo.toml") == nil {
			t.Fatal("external package path accepted")
		}
	}
	if err := ValidateProjectManifest([]byte(base+"[dependencies]\nx={path=\"../sibling\"}\n"), "members/one/Cargo.toml"); err != nil {
		t.Fatal("inside-project parent path rejected", err)
	}
	for _, body := range [][]byte{[]byte("syntax invalid [["), bytes.Repeat([]byte(" "), MaxProjectControlBytes+1), []byte(base + "name=\"duplicate\"\n")} {
		if ValidateProjectManifest(body, "Cargo.toml") == nil {
			t.Fatal("invalid/excessive manifest accepted")
		}
	}
}

func TestCargoCompleteProjectSnapshotBindsLocalManifestsFeaturesAndRuntime(t *testing.T) {
	body, lock := cargoActualMetadataFixture(t)
	manifest := []byte("[package]\nname=\"haa_schema_fixture\"\nversion=\"0.1.0\"\nedition=\"2021\"\n[dependencies]\nitoa=\"=1.0.17\"\n")
	files := map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": lock}
	target, _ := domain.NewInstallTarget("/project")
	install, _ := domain.NewInstallContext(target)
	snapshot, err := BuildProjectSnapshot(install, body, manifest, lock, "/fixture", "fixed-runtime", files)
	if err != nil || !snapshot.Valid() || len(snapshot.Dependencies()) != 1 || len(snapshot.ControlDigests()) != 2 {
		t.Fatalf("complete snapshot %v", err)
	}
	for _, fault := range []string{"runtime", "features", "root control", "local inventory"} {
		t.Run(fault, func(t *testing.T) {
			metadata := append([]byte(nil), body...)
			m := append([]byte(nil), manifest...)
			runtime := "fixed-runtime"
			inventory := map[string][]byte{"Cargo.toml": m, "Cargo.lock": lock}
			switch fault {
			case "runtime":
				runtime = "other-runtime"
			case "features":
				var d projectMetadata
				if err := json.Unmarshal(metadata, &d); err != nil {
					t.Fatal(err)
				}
				d.Resolve.Nodes[1].Features = []string{"no-panic"}
				metadata, _ = json.Marshal(d)
			case "root control":
				m = append(m, []byte("\n# changed manifest bytes\n")...)
				inventory["Cargo.toml"] = m
			case "local inventory":
				delete(inventory, "Cargo.toml")
			}
			changed, err := BuildProjectSnapshot(install, metadata, m, lock, "/fixture", runtime, inventory)
			if fault == "local inventory" {
				if err == nil || changed.Valid() {
					t.Fatal("missing copied manifest accepted")
				}
				return
			}
			if err != nil || changed.GraphDigest() == snapshot.GraphDigest() {
				t.Fatalf("changed state not bound: %v", err)
			}
		})
	}
	// Add an unrelated direct registry dependency: the complete snapshot must
	// retain it even though the requested itoa primary graph would omit it.
	var d projectMetadata
	if err := json.Unmarshal(body, &d); err != nil {
		t.Fatal(err)
	}
	source := registrySourceURL
	d.Packages = append(d.Packages, projectPackage{ID: "second", Name: "second", Version: "1.2.3", Source: &source})
	d.Resolve.Nodes = append(d.Resolve.Nodes, projectNode{ID: "second"})
	secondEdge := d.Resolve.Nodes[0].Deps[0]
	secondEdge.Name, secondEdge.Pkg = "second", "second"
	d.Resolve.Nodes[0].Deps = append(d.Resolve.Nodes[0].Deps, secondEdge)
	extraLock := []byte(strings.Replace(string(lock), `["itoa"]`, `["itoa", "second"]`, 1))
	extraLock = append(extraLock, []byte("\n[[package]]\nname=\"second\"\nversion=\"1.2.3\"\nsource=\""+registrySourceURL+"\"\nchecksum=\""+strings.Repeat("a", 64)+"\"\n")...)
	manifest = append(append([]byte(nil), manifest...), []byte("second=\"=1.2.3\"\n")...)
	files["Cargo.toml"] = manifest
	metadata, _ := json.Marshal(d)
	files["Cargo.lock"] = extraLock
	complete, err := BuildProjectSnapshot(install, metadata, manifest, extraLock, "/fixture", "fixed-runtime", files)
	if err != nil || len(complete.Dependencies()) != 2 {
		t.Fatalf("complete project dropped unrelated package %v", err)
	}
}
