package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func pipxSupplyFixture(t *testing.T, requirement, entries string, extras ...wheelTestEntry) ([]UVLockedPackage, []UVWheelInput, []SimpleProject) {
	t.Helper()
	mainExtras := append([]wheelTestEntry{{name: "pipx/__main__.py", body: []byte("def cli(): return 0\n")}}, extras...)
	main := toolTestWheel(t, "pipx", requirement, entries, mainExtras...)
	child := toolTestWheel(t, "click", "", "")
	var packages []UVLockedPackage
	var inputs []UVWheelInput
	var pages []SimpleProject
	for _, item := range []struct {
		name string
		body []byte
	}{{"pipx", main}, {"click", child}} {
		sum := sha256.Sum256(item.body)
		digest := hex.EncodeToString(sum[:])
		filename := item.name + "-1.0.0-py3-none-any.whl"
		url := "https://files.pythonhosted.org/packages/" + filename
		packages = append(packages, UVLockedPackage{item.name, "1.0.0", []UVLockedWheel{{url, filename, digest, int64(len(item.body))}}})
		inputs = append(inputs, UVWheelInput{url, bytes.NewReader(item.body), int64(len(item.body))})
		wire := map[string]any{"meta": map[string]string{"api-version": "1.0"}, "name": item.name, "files": []any{map[string]any{"filename": filename, "url": url, "hashes": map[string]string{"sha256": digest}, "size": len(item.body), "requires-python": ">=3.14.7", "yanked": false}}}
		body, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		page, err := ParseSimpleProject(item.name, body)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
	}
	return packages, inputs, pages
}

const pipxSupplyEntry = "[console_scripts]\npipx = pipx.__main__:cli\n"

func TestPipxSupplyClosureReusesWholeArchiveMetadataAndMarkers(t *testing.T) {
	requirement := "click!=0.9,>=1.0.0\nRequires-Dist: colorama>=0.4.6; sys_platform == 'win32'\nRequires-Dist: uv>=0.12.22; extra == 'uv'"
	packages, inputs, pages := pipxSupplyFixture(t, requirement, pipxSupplyEntry, wheelTestEntry{name: "pipx-1.0.0.data/data/share/man/man1/pipx.1", body: []byte("bounded manual\n")})
	closure, err := BindPipxSupplyClosure(packages, inputs, pages)
	if err != nil {
		t.Fatal(err)
	}
	if !closure.Valid() || len(closure.Wheels()) != 2 || len(closure.Files()) == 0 {
		t.Fatal("closure facts missing")
	}
	manual := false
	records := 0
	for _, file := range closure.Files() {
		if len(file.SHA256) != 64 {
			t.Fatal("supply manifest has an unhashed file")
		}
		if strings.HasSuffix(file.Path, "/RECORD") {
			records++
			if file.Size <= 0 {
				t.Fatal("RECORD self-hash exemption was mistaken for empty supply bytes")
			}
		}
		if file.Scheme == SchemeData {
			manual = file.Role == RuntimeRoleResource && file.Destination == "share/man/man1/pipx.1"
		}
	}
	if !manual || records != 2 {
		t.Fatal("resource scheme or immutable RECORD facts missing")
	}
	if _, err := ParseBoundedRequirement("click!=0.9"); err == nil {
		t.Fatal("supply-only exclusion expanded the general requirement language")
	}
	wheels := closure.Wheels()
	wheels[0].SHA256 = "tampered"
	files := closure.Files()
	files[0].Path = "escape"
	if closure.Wheels()[0].SHA256 == "tampered" || closure.Files()[0].Path == "escape" {
		t.Fatal("closure can be changed through getters")
	}
	if (PipxSupplyClosure{}).Valid() {
		t.Fatal("zero closure is valid")
	}
}

func TestPipxSupplyClosureRejectsIncompleteOrSubstitutedInputs(t *testing.T) {
	for _, mode := range []string{"no-main", "no-wheel", "missing-page", "duplicate-package", "wrong-version", "wrong-filename", "wrong-digest", "wrong-url", "wrong-size", "nonpure", "unreachable", "unsatisfied", "excluded", "exclusion-wildcard", "exclusion-prerelease", "exclusion-overflow", "exclusion-empty", "active-missing", "unknown-marker", "startup", "outside-root", "shadow-child", "bytecode", "entrypoint", "data-script"} {
		t.Run(mode, func(t *testing.T) {
			requirement := "click>=1.0.0"
			entry := pipxSupplyEntry
			var extras []wheelTestEntry
			switch mode {
			case "unreachable":
				requirement = ""
			case "unsatisfied":
				requirement = "click>=2.0.0"
			case "excluded":
				requirement = "click!=1,>=1.0.0"
			case "exclusion-wildcard":
				requirement = "click!=0.9.*"
			case "exclusion-prerelease":
				requirement = "click!=0.9rc1"
			case "exclusion-overflow":
				requirement = "click!=" + strings.Repeat("9", 30)
			case "exclusion-empty":
				requirement = "click!=0.9,"
			case "active-missing":
				requirement = "userpath>=1.0.0"
			case "unknown-marker":
				requirement = "click; unknown == 'x'"
			case "startup":
				extras = []wheelTestEntry{{name: "startup.pth", body: []byte("import os\n")}}
			case "outside-root":
				extras = []wheelTestEntry{{name: "unowned.py", body: []byte("pass\n")}}
			case "shadow-child":
				extras = []wheelTestEntry{{name: "click/__init__.py", body: []byte("pass\n")}}
			case "bytecode":
				extras = []wheelTestEntry{{name: "pipx/cached.pyc", body: []byte("bad bytecode")}}
			case "entrypoint":
				entry = strings.Replace(entry, "pipx.__main__:cli", "pipx.__main__:other", 1)
			case "data-script":
				extras = []wheelTestEntry{{name: "pipx-1.0.0.data/scripts/run", body: []byte("#!/bin/sh\nexit 0\n")}}
			}
			packages, inputs, pages := pipxSupplyFixture(t, requirement, entry, extras...)
			switch mode {
			case "no-main":
				packages = packages[1:]
				inputs = inputs[1:]
				pages = pages[1:]
			case "no-wheel":
				inputs = inputs[:1]
			case "missing-page":
				pages = pages[:1]
			case "duplicate-package":
				packages[1] = packages[0]
			case "wrong-version":
				packages[0].Version = "2.0.0"
			case "wrong-filename":
				packages[0].Wheels[0].Filename = "other-1.0.0-py3-none-any.whl"
			case "wrong-digest":
				packages[0].Wheels[0].SHA256 = strings.Repeat("0", 64)
			case "wrong-url":
				packages[0].Wheels[0].URL = "https://files.pythonhosted.org/packages/other.whl"
			case "wrong-size":
				packages[0].Wheels[0].Size++
			case "nonpure":
				packages[0].Wheels[0].Filename = "pipx-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl"
			}
			if c, err := BindPipxSupplyClosure(packages, inputs, pages); err == nil || c.Valid() {
				t.Fatal("incomplete or substituted closure accepted")
			}
		})
	}
}
