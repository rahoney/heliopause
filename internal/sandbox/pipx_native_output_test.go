package sandbox

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pipxNativeOutputFixture(t *testing.T, additional ...map[string][]byte) (*pipxNativeInput, map[string]uvToolTarEntry) {
	t.Helper()
	extra := map[string][]byte{"probe-1.0.dist-info/entry_points.txt": []byte("[console_scripts]\nhaa-alt = probe:main\nhaa-demo = probe:main\n")}
	for _, files := range additional {
		for name, body := range files {
			if _, exists := extra[name]; exists {
				t.Fatal("duplicate fixture file")
			}
			extra[name] = body
		}
	}
	frozen, _, _, _, _ := pipxFrozenInputFixture(t, extra)
	entries := map[string]uvToolTarEntry{}
	env := "pipx/venvs/probe"
	site := env + "/lib/python3.14/site-packages"
	put := func(name string, kind byte, mode int64, body []byte, target string) {
		entries[name] = uvToolTarEntry{tar.Header{Name: name, Typeflag: kind, Mode: mode, Size: int64(len(body)), Uid: 1000, Gid: 1000, Linkname: target}, body}
	}
	for _, name := range []string{".", "pipx", "pipx/.cache", "pipx/logs", "pipx/py", "pipx/venvs", env, env + "/bin", env + "/lib", env + "/lib/python3.14", site, "tool-bin", "tool-man"} {
		put(name, tar.TypeDir, 0755, nil, "")
	}
	for _, name := range []string{"pipx/.cache/CACHEDIR.TAG", "pipx/py/CACHEDIR.TAG", env + "/.gitignore", env + "/CACHEDIR.TAG"} {
		put(name, tar.TypeReg, 0644, []byte("*"), "")
	}
	put(env+"/.lock", tar.TypeReg, 0666, nil, "")
	put("pipx/venvs/.probe.lock", tar.TypeReg, 0644, nil, "")
	for _, name := range []string{"activate", "activate.bat", "activate.csh", "activate.fish", "activate.nu", "activate.ps1", "activate.xsh", "activate_this.py", "deactivate.bat", "pydoc.bat"} {
		put(env+"/bin/"+name, tar.TypeReg, 0644, []byte("must never execute"), "")
	}
	put(env+"/pyvenv.cfg", tar.TypeReg, 0644, []byte(fmt.Sprintf("home = /usr/local/bin\nimplementation = CPython\nuv = %s\nversion_info = %s\ninclude-system-site-packages = false\n", runtimeidentity.UVVersion, runtimeidentity.PythonVersion)), "")
	put("pipx/logs/cmd_2026-10-28_06.00.00.log", tar.TypeReg, 0644, []byte("untrusted data, no completion authority"), "")
	for name, target := range map[string]string{env + "/bin/python": "/usr/local/bin/python", env + "/bin/python3": "python", env + "/bin/python3.14": "python", env + "/lib64": "lib"} {
		put(name, tar.TypeSymlink, 0777, nil, target)
	}
	for _, name := range []string{"haa-alt", "haa-demo"} {
		put(env+"/bin/"+name, tar.TypeReg, 0755, []byte("raise RuntimeError('never execute')\n"), "")
		put("tool-bin/"+name, tar.TypeSymlink, 0777, nil, pipxNativeHome+"/venvs/probe/bin/"+name)
	}
	body, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/pipx-installed.json")
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.ReplaceAll(body, []byte("haa-tool-fixture"), []byte("probe"))
	body = bytes.ReplaceAll(body, []byte("/work/pipx"), []byte(pipxNativeHome))
	body = bytes.ReplaceAll(body, []byte("1.0.0"), []byte("1.0"))
	var wire map[string]any
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	wire["main_package"].(map[string]any)["pip_args"] = []any{}
	body, err = json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	put(env+"/pipx_metadata.json", tar.TypeReg, 0600, body, "")
	original := frozen.files[0].body
	archive, err := zip.NewReader(bytes.NewReader(original), int64(len(original)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		input, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(input)
		closeErr := input.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		name := site + "/" + file.Name
		for parent := path.Dir(name); parent != site && parent != "."; parent = path.Dir(parent) {
			if _, ok := entries[parent]; !ok {
				put(parent, tar.TypeDir, 0755, nil, "")
			}
		}
		put(name, tar.TypeReg, 0644, body, "")
	}
	return frozen, entries
}

type pipxNativeOutputTransport struct {
	body  []byte
	cause error
	calls int
}

func (r *pipxNativeOutputTransport) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("process execution forbidden")
}
func (r *pipxNativeOutputTransport) RunOutput(_ context.Context, w io.Writer, name string, args ...string) error {
	r.calls++
	if name != "docker" || len(args) != 3 || args[0] != "cp" || args[1] != strings.Repeat("a", 64)+":"+pythonSitePath+"/." || args[2] != "-" {
		return errors.New("unexpected complete output transport")
	}
	if _, err := w.Write(r.body); err != nil {
		return err
	}
	return r.cause
}
func TestPipxNativeOutputAuthenticatesWholeOriginalControlsAndCompleteLayout(t *testing.T) {
	frozen, entries := pipxNativeOutputFixture(t)
	encoded := uvToolFixtureTar(t, entries)
	r := &pipxNativeOutputTransport{body: encoded}
	output, err := readPipxNativeOutput(context.Background(), r, strings.Repeat("a", 64), "probe", frozen)
	if err != nil || output.receipt.Client() != "pipx" || len(output.controls) != 1 || len(output.links) != 2 || len(output.graph.Nodes()) != 1 {
		t.Fatal("complete native output refused", err)
	}
	// Native output is never an archive source. Even a parseable altered header
	// with the same project/version must fail its original wheel size/digest.
	output.controls[0].Metadata = append(output.controls[0].Metadata, []byte("Summary: forged\n")...)
	if _, err := frozen.bindNativeOutput(output); err == nil {
		t.Fatal("parseable modified native metadata accepted")
	}
	cause := errors.New("primary producer failure")
	r.cause = cause
	if result, err := readPipxNativeOutput(context.Background(), r, strings.Repeat("a", 64), "probe", frozen); !errors.Is(err, cause) || result.receipt.Valid() {
		t.Fatal("producer failure lost or output adopted", err)
	}
	r.calls = 0
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readPipxNativeOutput(ctx, r, strings.Repeat("a", 64), "probe", frozen); !errors.Is(err, context.Canceled) || r.calls != 0 {
		t.Fatal("cancel reached transport", err)
	}
	if _, err := readPipxNativeOutput(context.Background(), r, strings.Repeat("a", 64), "probe", nil); err == nil || r.calls != 0 {
		t.Fatal("invalid original reached transport")
	}
}
func TestPipxNativeOutputRejectsForeignIncompleteAndChangedTree(t *testing.T) {
	env := "pipx/venvs/probe"
	metadata := env + "/lib/python3.14/site-packages/probe-1.0.dist-info/METADATA"
	cases := map[string]func(map[string]uvToolTarEntry){
		"foreign environment": func(e map[string]uvToolTarEntry) {
			e["pipx/venvs/foreign"] = uvToolTarEntry{tar.Header{Name: "pipx/venvs/foreign", Typeflag: tar.TypeDir, Mode: 0755, Uid: 1000, Gid: 1000}, nil}
		},
		"resource exposure": func(e map[string]uvToolTarEntry) {
			e["tool-man/foreign"] = uvToolTarEntry{tar.Header{Name: "tool-man/foreign", Typeflag: tar.TypeReg, Mode: 0644, Uid: 1000, Gid: 1000}, nil}
		},
		"extra script": func(e map[string]uvToolTarEntry) {
			e[env+"/bin/foreign"] = uvToolTarEntry{tar.Header{Name: env + "/bin/foreign", Typeflag: tar.TypeReg, Mode: 0755, Uid: 1000, Gid: 1000}, nil}
		},
		"missing runtime":      func(e map[string]uvToolTarEntry) { delete(e, env+"/bin/python3") },
		"missing parent":       func(e map[string]uvToolTarEntry) { delete(e, env+"/lib") },
		"missing metadata":     func(e map[string]uvToolTarEntry) { delete(e, metadata) },
		"missing receipt":      func(e map[string]uvToolTarEntry) { delete(e, env+"/pipx_metadata.json") },
		"missing exposed link": func(e map[string]uvToolTarEntry) { delete(e, "tool-bin/haa-demo") },
		"foreign runtime alias": func(e map[string]uvToolTarEntry) {
			v := e[env+"/bin/python"]
			v.header.Linkname = "/tmp/python"
			e[v.header.Name] = v
		},
		"foreign command alias": func(e map[string]uvToolTarEntry) {
			v := e["tool-bin/haa-demo"]
			v.header.Linkname = "/tmp/program"
			e[v.header.Name] = v
		},
		"writable metadata": func(e map[string]uvToolTarEntry) { v := e[metadata]; v.header.Mode = 0666; e[metadata] = v },
		"foreign owner":     func(e map[string]uvToolTarEntry) { v := e[metadata]; v.header.Uid = 0; e[metadata] = v },
		"hard link": func(e map[string]uvToolTarEntry) {
			v := e[metadata]
			v.header.Typeflag = tar.TypeLink
			v.header.Linkname = "/outside"
			v.header.Size = 0
			v.body = nil
			e[metadata] = v
		},
		"changed metadata": func(e map[string]uvToolTarEntry) {
			v := e[metadata]
			v.body = append(v.body, []byte("Summary: forged\n")...)
			v.header.Size = int64(len(v.body))
			e[metadata] = v
		},
		"changed runtime config": func(e map[string]uvToolTarEntry) {
			v := e[env+"/pyvenv.cfg"]
			v.body = bytes.ReplaceAll(v.body, []byte("false"), []byte("true"))
			v.header.Size = int64(len(v.body))
			e[v.header.Name] = v
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			frozen, entries := pipxNativeOutputFixture(t)
			change(entries)
			r := &pipxNativeOutputTransport{body: uvToolFixtureTar(t, entries)}
			result, err := readPipxNativeOutput(context.Background(), r, strings.Repeat("a", 64), "probe", frozen)
			if err == nil || result.receipt.Valid() {
				t.Fatal("invalid complete output accepted")
			}
		})
	}
	frozen, entries := pipxNativeOutputFixture(t)
	body := uvToolFixtureTar(t, entries)
	for name, encoded := range map[string][]byte{"duplicate": append(bytes.Clone(body[:512]), body...), "trailing": append(bytes.Clone(body), 1), "truncated": body[:len(body)/2]} {
		t.Run(name, func(t *testing.T) {
			r := &pipxNativeOutputTransport{body: encoded}
			if _, err := readPipxNativeOutput(context.Background(), r, strings.Repeat("a", 64), "probe", frozen); err == nil {
				t.Fatal("invalid tar accepted")
			}
		})
	}
	var oversized bytes.Buffer
	writer := tar.NewWriter(&oversized)
	if err := writer.WriteHeader(&tar.Header{Name: env + "/pipx_metadata.json", Typeflag: tar.TypeReg, Mode: 0600, Uid: 1000, Gid: 1000, Size: uvNativeLockLimit + 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := scanPipxNativeOutput(bytes.NewReader(oversized.Bytes()), "probe"); err == nil {
		t.Fatal("oversized control accepted")
	}
}

// This tests authentic whole archives, not mutation of a previously admitted
// snapshot. The existing primary dist-info identity guard must reject the
// collision before a second wheel can reach native execution or output binding.
func TestPipxNativeInputRejectsCrossWheelControlCollisionBeforeOutput(t *testing.T) {
	first, entries := pipxNativeOutputFixture(t)
	if _, err := scanPipxNativeOutput(bytes.NewReader(uvToolFixtureTar(t, entries)), "probe"); err != nil {
		t.Fatal(err)
	}
	unpack := func(body []byte) map[string][]byte {
		archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			t.Fatal(err)
		}
		files := map[string][]byte{}
		for _, file := range archive.File {
			r, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(r)
			closeErr := r.Close()
			if err != nil || closeErr != nil {
				t.Fatal(err, closeErr)
			}
			files[file.Name] = b
		}
		return files
	}
	pack := func(files map[string][]byte, record string) []byte {
		var names []string
		for name := range files {
			if name != record {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		var rows strings.Builder
		for _, name := range names {
			sum := sha256.Sum256(files[name])
			fmt.Fprintf(&rows, "%s,sha256=%s,%d\n", name, base64.RawURLEncoding.EncodeToString(sum[:]), len(files[name]))
		}
		rows.WriteString(record + ",,\n")
		files[record] = []byte(rows.String())
		names = append(names, record)
		var result bytes.Buffer
		writer := zip.NewWriter(&result)
		for _, name := range names {
			entry, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write(files[name]); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return result.Bytes()
	}
	root := unpack(first.files[0].body)
	root["probe-1.0.dist-info/METADATA"] = append(root["probe-1.0.dist-info/METADATA"], []byte("Requires-Dist: zzzchild>=1\n")...)
	rootBody := pack(root, "probe-1.0.dist-info/RECORD")
	forged := append(bytes.Clone(root["probe-1.0.dist-info/METADATA"]), []byte("Summary: shadowed by another wheel\n")...)
	childBody := linuxDynamicProjectWheel(t, "zzzchild", "raise RuntimeError('never execute')\n", map[string][]byte{"probe-1.0.dist-info/METADATA": forged, "probe-1.0.dist-info/WHEEL": root["probe-1.0.dist-info/WHEEL"]})
	lockBody := "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=" + runtimeidentity.PythonVersion + "'\n"
	var inputs []artifactpypi.UVWheelInput
	var pages []artifactpypi.SimpleProject
	for _, item := range []struct {
		name string
		body []byte
	}{{"probe", rootBody}, {"zzzchild", childBody}} {
		sum := sha256.Sum256(item.body)
		digest := hex.EncodeToString(sum[:])
		filename := item.name + "-1.0-py3-none-any.whl"
		url := "https://files.pythonhosted.org/packages/" + filename
		lockBody += fmt.Sprintf("[[packages]]\nname=%q\nversion='1.0'\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", item.name, url, len(item.body), digest)
		body, err := json.Marshal(map[string]any{"info": map[string]any{"name": item.name, "version": "1.0"}, "urls": []any{map[string]any{"filename": filename, "url": url, "size": len(item.body), "digests": map[string]string{"sha256": digest}, "requires_python": "", "yanked": false, "packagetype": "bdist_wheel"}}})
		if err != nil {
			t.Fatal(err)
		}
		page, err := artifactpypi.ParseUVReleaseMetadata(item.name, "1.0", body)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
		inputs = append(inputs, artifactpypi.UVWheelInput{URL: url, Reader: bytes.NewReader(item.body), Size: int64(len(item.body))})
	}
	reference, err := artifactpypi.ParseReference("probe@1.0")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := artifactpypi.ParseUVLock(reference, []byte(lockBody), runtimeidentity.PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(childBody)
	runtime := PinnedPythonRuntime()
	_, inspectErr := artifactpypi.InspectWheel(bytes.NewReader(childBody), int64(len(childBody)), "zzzchild-1.0-py3-none-any.whl", hex.EncodeToString(sum[:]), artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}, artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits())
	stage, known := artifactpypi.WheelValidationStageOf(inspectErr)
	if !known || stage != artifactpypi.WheelValidationDistInfoIdentity {
		t.Fatal("foreign primary metadata was not rejected by its existing owner", stage, inspectErr)
	}
	if frozen, err := freezePipxNativeInput(context.Background(), lock, inputs, pages); err == nil || frozen != nil {
		t.Fatal("foreign primary metadata reached the native input/output boundary")
	}

}

func TestPipxNativeOutputPreservesNestedDistributionPayload(t *testing.T) {
	frozen, entries := pipxNativeOutputFixture(t, map[string][]byte{"probe/_vendor/foreign-1.0.dist-info/METADATA": []byte("nested inert payload"), "probe/_vendor/foreign-1.0.dist-info/WHEEL": []byte("nested inert payload")})
	runner := &pipxNativeOutputTransport{body: uvToolFixtureTar(t, entries)}
	if _, err := readPipxNativeOutput(context.Background(), runner, strings.Repeat("a", 64), "probe", frozen); err != nil {
		t.Fatal("nested payload became primary distribution metadata", err)
	}
}
