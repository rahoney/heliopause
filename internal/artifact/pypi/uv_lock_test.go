package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestUVNativeLockPreservesWheelAlternatives(t *testing.T) {
	body, err := os.ReadFile("testdata/uv-0.13.4-requests.pylock.toml")
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("requests@2.32.5")
	lock, err := ParseUVLock(reference, body, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	packages := lock.Packages()
	if len(packages) != 5 || packages[1].Name != "charset-normalizer" || len(packages[1].Wheels) != 3 {
		t.Fatal("lost native graph or alternatives", packages)
	}
	packages[0].Name = "changed"
	packages[1].Wheels[0].URL = "changed"
	if lock.Packages()[0].Name == "changed" || lock.Packages()[1].Wheels[0].URL == "changed" {
		t.Fatal("mutable exported selection authority")
	}
	for name, body := range map[string][]byte{
		"empty":                     nil,
		"oversize":                  bytes.Repeat([]byte(" "), maxReportBytes+1),
		"future schema":             bytes.ReplaceAll(body, []byte(`lock-version = "1.0"`), []byte(`lock-version = "2.0"`)),
		"unknown source":            append(append([]byte(nil), body...), []byte("\n[packages.directory]\npath='.'\n")...),
		"target":                    bytes.ReplaceAll(body, []byte(`>=3.14.7`), []byte(`>=4.0`)),
		"creator":                   bytes.ReplaceAll(body, []byte(`created-by = "uv"`), []byte(`created-by = "pip"`)),
		"missing root":              bytes.ReplaceAll(body, []byte(`name = "requests"`), []byte(`name = "other"`)),
		"root version":              bytes.ReplaceAll(body, []byte(`version = "2.32.5"`), []byte(`version = "2.32.4"`)),
		"aggregate download budget": bytes.Replace(body, []byte("size = 136464"), []byte("size = 67108864"), 1),
		"foreign URL":               bytes.ReplaceAll(body, []byte("files.pythonhosted.org"), []byte("example.org")),
		"option injection":          append(append([]byte(nil), body...), []byte("\n[tool.uv]\nindex-url='http://example.org'\n")...),
		"duplicate table key":       append([]byte("lock-version='1.0'\n"), body...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVLock(reference, body, "3.14.7"); err == nil {
				t.Fatal("accepted malformed native lock")
			}
		})
	}
}

func uvTestWheel(t *testing.T, name, requirement string) []byte {
	t.Helper()
	dist := name + "-1.0.dist-info/"
	metadata := "Metadata-Version: 2.4\nName: " + name + "\nVersion: 1.0\nRequires-Python: >=3.8\n"
	if requirement != "" {
		metadata += "Requires-Dist: " + requirement + "\n"
	}
	files := map[string][]byte{name + "/__init__.py": []byte("# inert fixture\n"), dist + "METADATA": []byte(metadata), dist + "WHEEL": []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")}
	var rows []string
	for path, body := range files {
		sum := sha256.Sum256(body)
		rows = append(rows, fmt.Sprintf("%s,sha256=%s,%d", path, base64.RawURLEncoding.EncodeToString(sum[:]), len(body)))
	}
	rows = append(rows, dist+"RECORD,,")
	files[dist+"RECORD"] = []byte(strings.Join(rows, "\n") + "\n")
	return makeWheel(t, files)
}

func uvTestSelection(t *testing.T, requirement string) (UVLock, []UVWheelInput, []SimpleProject) {
	t.Helper()
	lock := `lock-version="1.0"
created-by="uv"
requires-python=">=3.14.7"
`
	var inputs []UVWheelInput
	var pages []SimpleProject
	for _, name := range []string{"parent", "child"} {
		req := ""
		if name == "parent" {
			req = requirement
		}
		body := uvTestWheel(t, name, req)
		sum := sha256.Sum256(body)
		hash := hex.EncodeToString(sum[:])
		filename := name + "-1.0-py3-none-any.whl"
		url := "https://files.pythonhosted.org/packages/" + filename
		lock += fmt.Sprintf("\n[[packages]]\nname=%q\nversion='1.0'\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", name, url, len(body), hash)
		inputs = append(inputs, UVWheelInput{URL: url, Reader: bytes.NewReader(body), Size: int64(len(body))})
		wire := map[string]any{"meta": map[string]string{"api-version": "1.0"}, "name": name, "files": []any{map[string]any{"filename": filename, "url": url, "hashes": map[string]string{"sha256": hash}, "requires-python": ">=3.8", "yanked": false, "size": len(body)}}}
		encoded, _ := json.Marshal(wire)
		page, err := ParseSimpleProject(name, encoded)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
	}
	reference, _ := ParseReference("parent@1.0")
	parsed, err := ParseUVLock(reference, []byte(lock), "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	return parsed, inputs, pages
}

func TestUVWheelBindingUsesAuthenticatedMetadataAndCompleteGraph(t *testing.T) {
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	lock, inputs, pages := uvTestSelection(t, "child>=1.0")
	graph, err := lock.BindWheels(inputs, pages, target, DefaultWheelLimits())
	if err != nil || len(graph.Nodes()) != 2 || len(graph.Edges()) != 1 {
		t.Fatal("native lock did not bind exact wheel graph", err)
	}
	for _, requirement := range []string{"", "child>=2.0", "missing>=1.0", "child; sys_platform == 'win32'"} {
		broken, files, metadata := uvTestSelection(t, requirement)
		if _, err := broken.BindWheels(files, metadata, target, DefaultWheelLimits()); err == nil {
			t.Fatal("accepted incomplete, orphan or unsatisfied graph", requirement)
		}
	}
	for name, mutate := range map[string]func([]UVWheelInput, []SimpleProject){
		"bytes": func(files []UVWheelInput, _ []SimpleProject) {
			files[0].Reader = bytes.NewReader(bytes.Repeat([]byte("x"), int(files[0].Size)))
		},
		"foreign URL":          func(files []UVWheelInput, _ []SimpleProject) { files[0].URL = "https://example.org/x.whl" },
		"size":                 func(files []UVWheelInput, _ []SimpleProject) { files[0].Size++ },
		"duplicate":            func(files []UVWheelInput, _ []SimpleProject) { files[0] = files[1] },
		"independent metadata": func(_ []UVWheelInput, metadata []SimpleProject) { metadata[0] = metadata[1] },
	} {
		t.Run(name, func(t *testing.T) {
			a, files, metadata := uvTestSelection(t, "child>=1.0")
			mutate(files, metadata)
			if _, err := a.BindWheels(files, metadata, target, DefaultWheelLimits()); err == nil {
				t.Fatal("accepted substituted selection")
			}
		})
	}
	if _, err := lock.BindWheels(inputs[:1], pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("accepted partial materialization")
	}
	if _, err := (UVLock{}).BindWheels(nil, nil, target, DefaultWheelLimits()); err == nil {
		t.Fatal("accepted empty lock")
	}
	larger := DefaultWheelLimits()
	larger.MaxCompressed++
	if _, err := lock.BindWheels(inputs, pages, target, larger); err == nil {
		t.Fatal("caller weakened public source limits")
	}
	target.Python, target.ABI = "cp313", "cp313"
	if _, err := lock.BindWheels(inputs, pages, target, DefaultWheelLimits()); err == nil {
		t.Fatal("changed target after resolution")
	}
}

func TestUVNativeMaterializationSelectsOneWheelWithoutApproval(t *testing.T) {
	body, err := os.ReadFile("testdata/uv-0.13.4-requests.pylock.toml")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := os.ReadFile("testdata/uv-0.13.4-requests-installed.json")
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("requests@2.32.5")
	lock, err := ParseUVLock(reference, body, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	files, err := lock.SelectInstalledWheels(snapshot, target)
	if err != nil || len(files) != 5 {
		t.Fatal("native uv materialization did not identify exact alternatives", err)
	}
	found := false
	for _, file := range files {
		if strings.HasPrefix(file.Filename, "charset_normalizer-") {
			found = true
			if !strings.Contains(file.Filename, "cp314-cp314-") {
				t.Fatal("selected a different ABI from the native tool")
			}
		}
	}
	if !found {
		t.Fatal("missing native extension wheel")
	}
	for name, input := range map[string][]byte{
		"unknown":         bytes.Replace(snapshot, []byte(`"name":`), []byte(`"other":`), 1),
		"different ABI":   bytes.ReplaceAll(snapshot, []byte("cp314"), []byte("cp313")),
		"missing tag":     bytes.Replace(snapshot, []byte("py3-none-any"), []byte(""), 1),
		"duplicate name":  bytes.Replace(snapshot, []byte(`"charset-normalizer"`), []byte(`"certifi"`), 1),
		"changed version": bytes.Replace(snapshot, []byte(`"2.32.5"`), []byte(`"2.32.4"`), 1),
		"trailing":        append(append([]byte(nil), snapshot...), []byte("null")...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := lock.SelectInstalledWheels(input, target); err == nil {
				t.Fatal("accepted altered native selection")
			}
		})
	}
	// Equal standard tags do not identify a URL. Refuse alternate mirrors/builds
	// with identical tags instead of guessing from order or human log messages.
	for i, entry := range lock.packages {
		if entry.Name == "certifi" {
			copy := entry.Wheels[0]
			copy.URL = strings.Replace(copy.URL, "/packages/", "/packages/duplicate/", 1)
			lock.packages[i].Wheels = append(lock.packages[i].Wheels, copy)
		}
	}
	if _, err := lock.SelectInstalledWheels(snapshot, target); err == nil {
		t.Fatal("guessed between ambiguous native files")
	}
}

func FuzzParseUVLockNoPanic(f *testing.F) {
	body, err := os.ReadFile("testdata/uv-0.13.4-requests.pylock.toml")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(body)
	f.Add([]byte("lock-version='future'"))
	f.Add(uvProjectBody(nil))
	reference, _ := ParseReference("requests@2.32.5")
	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = ParseUVLock(reference, input, "3.14.7")
		_, _ = ParseUVProjectLock(input, "3.14.7")
	})
}
