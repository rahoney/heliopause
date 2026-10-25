package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const toolTestEntrypoints = "[console_scripts]\nhaa-alt = haa_tool_fixture:main\nhaa-demo = haa_tool_fixture:main\n"

func toolTestWheel(t *testing.T, name, requires, entrypoints string, extras ...wheelTestEntry) []byte {
	t.Helper()
	dist := strings.ReplaceAll(name, "-", "_") + "-1.0.0.dist-info/"
	metadata := "Metadata-Version: 2.4\nName: " + name + "\nVersion: 1.0.0\nRequires-Python: >=3.14.7\n"
	if requires != "" {
		metadata += "Requires-Dist: " + requires + "\n"
	}
	files := map[string][]byte{strings.ReplaceAll(name, "-", "_") + "/__init__.py": []byte("def main(): return 0\n"), dist + "METADATA": []byte(metadata), dist + "WHEEL": []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")}
	if entrypoints != "" {
		files[dist+"entry_points.txt"] = []byte(entrypoints)
	}
	for _, entry := range extras {
		files[entry.name] = entry.body
	}
	var paths []string
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var rows []string
	for _, path := range paths {
		body := files[path]
		sum := sha256.Sum256(body)
		rows = append(rows, fmt.Sprintf("%s,sha256=%s,%d", path, base64.RawURLEncoding.EncodeToString(sum[:]), len(body)))
	}
	rows = append(rows, dist+"RECORD,,")
	files[dist+"RECORD"] = []byte(strings.Join(rows, "\n") + "\n")
	return makeWheel(t, files)
}

func toolTestSelection(t *testing.T, rootWheel, childWheel []byte) (UVLock, []UVWheelInput, []SimpleProject) {
	t.Helper()
	body := "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=3.14.7'\n"
	var inputs []UVWheelInput
	var pages []SimpleProject
	for _, item := range []struct {
		name string
		body []byte
	}{{"haa-tool-fixture", rootWheel}, {"child", childWheel}} {
		sum := sha256.Sum256(item.body)
		hash := hex.EncodeToString(sum[:])
		filename := strings.ReplaceAll(item.name, "-", "_") + "-1.0.0-py3-none-any.whl"
		url := "https://files.pythonhosted.org/packages/" + filename
		body += fmt.Sprintf("[[packages]]\nname=%q\nversion='1.0.0'\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", item.name, url, len(item.body), hash)
		inputs = append(inputs, UVWheelInput{URL: url, Reader: bytes.NewReader(item.body), Size: int64(len(item.body))})
		encoded, err := json.Marshal(map[string]any{"meta": map[string]string{"api-version": "1.0"}, "name": item.name, "files": []any{map[string]any{"filename": filename, "url": url, "hashes": map[string]string{"sha256": hash}, "size": len(item.body), "requires-python": ">=3.14.7", "yanked": false}}})
		if err != nil {
			t.Fatal(err)
		}
		page, err := ParseSimpleProject(item.name, encoded)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
	}
	ref, err := ParseReference("haa-tool-fixture@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := ParseUVLock(ref, []byte(body), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	return lock, inputs, pages
}

func TestToolReceiptBindsIndependentCompleteWheelGraph(t *testing.T) {
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for _, client := range []string{"uv", "pipx"} {
		t.Run(client, func(t *testing.T) {
			parse, extension := ParseUVToolReceipt, ".toml"
			if client == "pipx" {
				parse, extension = ParsePipxToolReceipt, ".json"
			}
			receipt, err := parse(nativeToolFixture(t, client+"-installed"+extension))
			if err != nil {
				t.Fatal(err)
			}
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints), toolTestWheel(t, "child", "", ""))
			graph, err := receipt.BindWheels(lock, inputs, pages, target, DefaultWheelLimits())
			if err != nil {
				t.Fatal(err)
			}
			original, err := lock.BindWheels(inputs, pages, target, DefaultWheelLimits())
			if err != nil || !reflect.DeepEqual(graph, original) || len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 {
				t.Fatal("existing source/closure semantics differ", err)
			}
			if _, err := receipt.BindWheels(lock, inputs, pages[:1], target, DefaultWheelLimits()); err == nil {
				t.Fatal("incomplete independent source accepted")
			}
			foreignPages := append([]SimpleProject(nil), pages...)
			foreignPages[0].files = append([]SimpleFile(nil), pages[0].files...)
			foreignPages[0].files[0].sha256 = strings.Repeat("c", 64)
			if _, err := receipt.BindWheels(lock, inputs, foreignPages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("native lock replaced independent source hash")
			}
			changed := append([]UVWheelInput(nil), inputs...)
			changed[1].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(changed[1].Size)))
			if _, err := receipt.BindWheels(lock, changed, pages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("whole transitive archive authentication skipped")
			}
			wrong := receipt
			wrong.requirement, _ = ParseUpgradeRequirement("haa-tool-fixture>=2")
			if _, err := wrong.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("native constraint mismatch accepted")
			}
			if client == "pipx" {
				wrong = receipt
				wrong.version = "1.1.0"
				if _, err := wrong.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err == nil {
					t.Fatal("native version mismatch accepted")
				}
			}
			wrong = receipt
			wrong.requirement, _ = ParseUpgradeRequirement("other")
			if _, err := wrong.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("foreign root accepted")
			}
		})
	}
}

func TestToolReceiptRejectsCommandAndGraphSubstitution(t *testing.T) {
	r, err := ParseUVToolReceipt(nativeToolFixture(t, "uv-installed.toml"))
	if err != nil {
		t.Fatal(err)
	}
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for name, ep := range map[string]string{
		"missing": "[console_scripts]\nhaa-alt=haa_tool_fixture:main\n",
		"extra":   toolTestEntrypoints + "foreign=haa_tool_fixture:main\n",
		"case":    strings.Replace(toolTestEntrypoints, "haa-demo", "HAA-DEMO", 1),
		"extras":  strings.Replace(toolTestEntrypoints, "haa-demo = haa_tool_fixture:main", "haa-demo = haa_tool_fixture:main [optional]", 1),
		"GUI":     toolTestEntrypoints + "[gui_scripts]\ngui=haa_tool_fixture:main\n",
	} {
		t.Run(name, func(t *testing.T) {
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", ep), toolTestWheel(t, "child", "", toolTestEntrypoints))
			if _, err := r.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("main command set replaced by native/dependency claim")
			}
		})
	}
	for _, req := range []string{"", "child>=2", "absent>=1"} {
		t.Run("dependency-"+req, func(t *testing.T) {
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", req, toolTestEntrypoints), toolTestWheel(t, "child", "", ""))
			if _, err := r.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("orphan/missing/wrong dependency accepted")
			}
		})
	}
	lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints, wheelTestEntry{name: "haa_tool_fixture-1.0.0.data/scripts/foreign", body: []byte("#!/bin/sh\nexit 0\n")}), toolTestWheel(t, "child", "", ""))
	if _, err := r.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("unreviewed data script accepted")
	}
}

func TestPipxMetadataGateCoversWholeInstalledStartupSurface(t *testing.T) {
	pipx, err := ParsePipxToolReceipt(nativeToolFixture(t, "pipx-installed.json"))
	if err != nil {
		t.Fatal(err)
	}
	uv, err := ParseUVToolReceipt(nativeToolFixture(t, "uv-installed.toml"))
	if err != nil {
		t.Fatal(err)
	}
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for _, name := range []string{"startup.pth", "sitecustomize.py", "usercustomize.py", "sitecustomize/__init__.py", "child-1.0.0.data/purelib/sitecustomize.py"} {
		t.Run(name, func(t *testing.T) {
			child := toolTestWheel(t, "child", "", "", wheelTestEntry{name: name, body: []byte("import os\n")})
			lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints), child)
			if _, err := pipx.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err == nil {
				t.Fatal("pipx post-install startup surface in dependency accepted")
			}
			if _, err := uv.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err != nil {
				t.Fatal("pipx-specific metadata gate changed ordinary static graph binding", err)
			}
		})
	}
	child := toolTestWheel(t, "child", "", "", wheelTestEntry{name: "child/sitecustomize.py", body: []byte("# no automatic top-level startup\n")})
	lock, inputs, pages := toolTestSelection(t, toolTestWheel(t, "haa-tool-fixture", "child>=1", toolTestEntrypoints), child)
	if _, err := pipx.BindWheels(lock, inputs, pages, target, DefaultWheelLimits()); err != nil {
		t.Fatal("nested ordinary module mistaken for startup", err)
	}
}
