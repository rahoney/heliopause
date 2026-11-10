package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type uvToolTarEntry struct {
	header tar.Header
	body   []byte
}

func uvToolOutputFixture(t *testing.T) map[string]uvToolTarEntry {
	t.Helper()
	env := "tools/haa-tool-fixture"
	site := env + "/lib/python3.14/site-packages"
	entries := map[string]uvToolTarEntry{}
	put := func(name string, kind byte, mode int64, body []byte, target string) {
		entries[name] = uvToolTarEntry{tar.Header{Name: name, Typeflag: kind, Mode: mode, Size: int64(len(body)), Uid: 1000, Gid: 1000, Linkname: target}, body}
	}
	for _, name := range []string{".", "tools", "tool-bin", env, env + "/bin", env + "/lib", env + "/lib/python3.14", site, site + "/haa_tool_fixture-1.0.0.dist-info", site + "/haa_tool_fixture"} {
		put(name, tar.TypeDir, 0700, nil, "")
	}
	for _, name := range []string{"tools/.gitignore", env + "/.gitignore", env + "/CACHEDIR.TAG"} {
		put(name, tar.TypeReg, 0644, []byte("*"), "")
	}
	put("tools/.lock", tar.TypeReg, 0666, nil, "")
	put(env+"/pyvenv.cfg", tar.TypeReg, 0644, []byte(fmt.Sprintf("home = /usr/local/bin\nimplementation = CPython\nuv = %s\nversion_info = %s\ninclude-system-site-packages = false\n", runtimeidentity.UVVersion, runtimeidentity.PythonVersion)), "")
	body, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/uv-explicit-index.toml")
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.ReplaceAll(body, []byte("/work/uv-tools"), []byte(uvNativeToolHome))
	body = bytes.ReplaceAll(body, []byte("/work/uv-bin"), []byte(uvNativeToolBin))
	put(env+"/uv-receipt.toml", tar.TypeReg, 0644, body, "")
	for name, target := range map[string]string{env + "/bin/python": "/usr/local/bin/python", env + "/bin/python3": "python", env + "/bin/python3.14": "python", env + "/lib64": "lib"} {
		put(name, tar.TypeSymlink, 0777, nil, target)
	}
	for _, name := range []string{"haa-alt", "haa-demo"} {
		put(env+"/bin/"+name, tar.TypeReg, 0755, []byte("raise RuntimeError('must never execute')\n"), "")
		put("tool-bin/"+name, tar.TypeSymlink, 0777, nil, uvNativeToolHome+"/haa-tool-fixture/bin/"+name)
	}
	put(site+"/haa_tool_fixture-1.0.0.dist-info/METADATA", tar.TypeReg, 0644, []byte("Metadata-Version: 2.4\nName: haa-tool-fixture\nVersion: 1.0.0\n"), "")
	put(site+"/haa_tool_fixture-1.0.0.dist-info/WHEEL", tar.TypeReg, 0644, []byte("Wheel-Version: 1.0\nTag: py3-none-any\n"), "")
	put(site+"/haa_tool_fixture/__init__.py", tar.TypeReg, 0644, []byte("raise RuntimeError('must never import')\n"), "")
	return entries
}

func uvToolFixtureTar(t *testing.T, entries map[string]uvToolTarEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	var names []string
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry := entries[name]
		if err := writer.WriteHeader(&entry.header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestUVToolNativeOutputReadsDataAndFreezesCompleteInventory(t *testing.T) {
	entries := uvToolOutputFixture(t)
	output, err := scanUVToolNativeOutput(bytes.NewReader(uvToolFixtureTar(t, entries)), "haa-tool-fixture", nil)
	if err != nil || !output.receipt.Valid() || len(output.controls) != 1 || len(output.links) != 2 {
		t.Fatal("native tool controls rejected", err)
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture>=1,<2")
	if err != nil {
		t.Fatal(err)
	}
	frozen, ref, err := uvToolFrozenRequirements(output, requirement)
	version, present, refErr := artifactpypi.RequestedVersion(ref)
	if err != nil || refErr != nil || !present || version != "1.0.0" || frozen != "haa-tool-fixture==1.0.0\n" {
		t.Fatal("exact complete freeze differs", frozen, err, refErr)
	}
	lock := []byte("lock-version='1.0'\n")
	entries["pylock.toml"] = uvToolTarEntry{tar.Header{Name: "pylock.toml", Typeflag: tar.TypeReg, Mode: 0644, Uid: 1000, Gid: 1000, Size: int64(len(lock))}, lock}
	final, err := scanUVToolNativeOutput(bytes.NewReader(uvToolFixtureTar(t, entries)), "haa-tool-fixture", lock)
	if err != nil || !bytes.Equal(final.lock, lock) {
		t.Fatal("actual frozen export rejected", err)
	}
	controls, err := uvToolSelectionControls(requirement, final)
	if err != nil || len(controls) != 4 {
		t.Fatal("native provenance controls missing", err)
	}
	if _, err := scanUVToolNativeOutput(bytes.NewReader(uvToolFixtureTar(t, entries)), "haa-tool-fixture", []byte("changed")); err == nil {
		t.Fatal("changed export accepted")
	}
	wrong, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture>=2")
	if _, _, err := uvToolFrozenRequirements(output, wrong); err == nil {
		t.Fatal("out-of-range native version accepted")
	}
}

func TestUVToolNativeOutputRejectsForeignLayoutAndRuntime(t *testing.T) {
	env := "tools/haa-tool-fixture"
	site := env + "/lib/python3.14/site-packages"
	metadata := site + "/haa_tool_fixture-1.0.0.dist-info/METADATA"
	cases := map[string]func(map[string]uvToolTarEntry){
		"foreign root": func(e map[string]uvToolTarEntry) {
			e["foreign"] = uvToolTarEntry{tar.Header{Name: "foreign", Typeflag: tar.TypeReg, Mode: 0644, Uid: 1000, Gid: 1000}, nil}
		},
		"foreign owner":        func(e map[string]uvToolTarEntry) { v := e[metadata]; v.header.Uid = 0; e[metadata] = v },
		"writable metadata":    func(e map[string]uvToolTarEntry) { v := e[metadata]; v.header.Mode = 0666; e[metadata] = v },
		"missing wheel":        func(e map[string]uvToolTarEntry) { delete(e, site+"/haa_tool_fixture-1.0.0.dist-info/WHEEL") },
		"missing receipt":      func(e map[string]uvToolTarEntry) { delete(e, env+"/uv-receipt.toml") },
		"missing runtime link": func(e map[string]uvToolTarEntry) { delete(e, env+"/bin/python3") },
		"replaced runtime link": func(e map[string]uvToolTarEntry) {
			v := e[env+"/bin/python"]
			v.header.Typeflag = tar.TypeReg
			v.header.Linkname = ""
			v.header.Mode = 0755
			e[env+"/bin/python"] = v
		},
		"different interpreter": func(e map[string]uvToolTarEntry) {
			v := e[env+"/bin/python"]
			v.header.Linkname = "/foreign/python"
			e[env+"/bin/python"] = v
		},
		"alias mode": func(e map[string]uvToolTarEntry) {
			v := e[env+"/bin/python"]
			v.header.Mode = 0755
			e[env+"/bin/python"] = v
		},
		"missing exposed": func(e map[string]uvToolTarEntry) { delete(e, "tool-bin/haa-demo") },
		"exposed foreign target": func(e map[string]uvToolTarEntry) {
			v := e["tool-bin/haa-demo"]
			v.header.Linkname = "/foreign/haa-demo"
			e["tool-bin/haa-demo"] = v
		},
		"extra exposed": func(e map[string]uvToolTarEntry) {
			v := e["tool-bin/haa-demo"]
			v.header.Name = "tool-bin/foreign"
			e[v.header.Name] = v
		},
		"exposed regular": func(e map[string]uvToolTarEntry) {
			v := e["tool-bin/haa-demo"]
			v.header.Typeflag = tar.TypeReg
			v.header.Mode = 0755
			v.header.Linkname = ""
			e[v.header.Name] = v
		},
		"missing command": func(e map[string]uvToolTarEntry) { delete(e, env+"/bin/haa-demo") },
		"payload symlink": func(e map[string]uvToolTarEntry) {
			v := e[site+"/haa_tool_fixture/__init__.py"]
			v.header.Typeflag = tar.TypeSymlink
			v.header.Mode = 0777
			v.header.Linkname = "/foreign/file"
			v.header.Size = 0
			v.body = nil
			e[v.header.Name] = v
		},
		"hardlink": func(e map[string]uvToolTarEntry) {
			v := e[site+"/haa_tool_fixture/__init__.py"]
			v.header.Typeflag = tar.TypeLink
			v.header.Linkname = metadata
			v.header.Size = 0
			v.body = nil
			e[v.header.Name] = v
		},
		"different Python config": func(e map[string]uvToolTarEntry) {
			v := e[env+"/pyvenv.cfg"]
			v.body = bytes.ReplaceAll(v.body, []byte("false"), []byte("true"))
			v.header.Size = int64(len(v.body))
			e[v.header.Name] = v
		},
		"traversal": func(e map[string]uvToolTarEntry) {
			e["tools/../foreign"] = uvToolTarEntry{tar.Header{Name: "tools/../foreign", Typeflag: tar.TypeReg, Mode: 0644, Uid: 1000, Gid: 1000}, nil}
		},
		"normalized duplicate": func(e map[string]uvToolTarEntry) {
			v := e[metadata]
			v.header.Name = "./" + metadata
			e[v.header.Name] = v
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			entries := uvToolOutputFixture(t)
			mutate(entries)
			result, err := scanUVToolNativeOutput(bytes.NewReader(uvToolFixtureTar(t, entries)), "haa-tool-fixture", nil)
			if err == nil || result.receipt.Valid() || len(result.controls) != 0 {
				t.Fatal("foreign native output retained selection")
			}
		})
	}
	body := uvToolFixtureTar(t, uvToolOutputFixture(t))
	for name, input := range map[string][]byte{"truncated": body[:len(body)/2], "trailing": append(append([]byte(nil), body...), 1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := scanUVToolNativeOutput(bytes.NewReader(input), "haa-tool-fixture", nil); err == nil {
				t.Fatal("incomplete native producer accepted")
			}
		})
	}
}

func TestUVToolNativeOutputRequiresProducerAndCleanupSuccess(t *testing.T) {
	body := uvToolFixtureTar(t, uvToolOutputFixture(t))
	id := strings.Repeat("a", 64)
	for _, name := range []string{"complete", "failed producer", "cancel"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cause := errors.New("producer failed after bytes")
			runner := &uvOutputRunner{body: body}
			if name == "failed producer" {
				runner.err = cause
			}
			if name == "cancel" {
				runner.cancel = cancel
			}
			result, err := readUVToolNativeOutput(ctx, runner, id, "haa-tool-fixture", nil)
			if name == "complete" {
				if err != nil || !result.receipt.Valid() {
					t.Fatal(err)
				}
			} else if err == nil || result.receipt.Valid() || len(result.controls) != 0 {
				t.Fatal("unsuccessful producer retained native result")
			}
			if name == "failed producer" && !errors.Is(err, cause) {
				t.Fatal("original failure lost")
			}
		})
	}
}

func TestUVToolSelectionRejectsBeforeRuntime(t *testing.T) {
	called := false
	r := &UVResolver{reader: &PyPIResolver{probe: func(context.Context) (PythonCapability, error) {
		called = true
		return PythonCapability{}, errors.New("must not run")
	}}}
	requirement, _ := artifactpypi.ParseUpgradeRequirement("cowsay==6.1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, test := range []struct {
		ctx         context.Context
		requirement artifactpypi.UpgradeRequirement
	}{{nil, requirement}, {ctx, requirement}, {context.Background(), artifactpypi.UpgradeRequirement{}}} {
		if result, err := r.ResolveToolInstall(test.ctx, test.requirement); err == nil || len(result.Controls()) != 0 {
			t.Fatal("invalid request accepted")
		}
	}
	if called {
		t.Fatal("invalid/cancelled selection probed runtime")
	}
}

func TestUVToolFixedEnvironmentPreservesDirectExecAdmission(t *testing.T) {
	base := pythonResolverCreateArguments("isolated-tool-network", nil, nil, nil)
	before := append([]string(nil), base...)
	tool, err := uvToolCreateArguments(base)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(base, "\x00") != strings.Join(before, "\x00") {
		t.Fatal("shared base arguments mutated")
	}
	for _, value := range []string{"UV_TOOL_DIR=" + uvNativeToolHome, "UV_TOOL_BIN_DIR=" + uvNativeToolBin, "UV_COMPILE_BYTECODE=false", "PYTHONDONTWRITEBYTECODE=1"} {
		count := 0
		for _, arg := range tool {
			if arg == value {
				count++
			}
		}
		if count != 1 {
			t.Fatal("controller tool environment missing/duplicated", value)
		}
	}
	container := strings.Repeat("a", 64)
	args := boundaryExecArguments(container, boundaryLaunchMode, uvToolGuestPath, "tool", "install", "cowsay==6.1")
	id, mode, offset, ok := directExecOrigin(args, "docker")
	if !ok || id != container || mode != boundaryLaunchMode || args[offset] != uvToolGuestPath {
		t.Fatal("native tool exec bypassed existing admission grammar")
	}
	invalid := append([]string{"exec", "--env", "UV_TOOL_DIR=" + uvNativeToolHome}, args[1:]...)
	if _, _, _, ok := directExecOrigin(invalid, "docker"); ok {
		t.Fatal("unsupported first exec shape silently admitted")
	}
	for _, bad := range [][]string{nil, {"create", pythonImageReference, "/bin/true"}, append(append([]string(nil), base...), pythonImageReference)} {
		if _, err := uvToolCreateArguments(bad); err == nil {
			t.Fatal("unknown container shape accepted")
		}
	}
}
