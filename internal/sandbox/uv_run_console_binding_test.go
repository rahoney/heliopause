package sandbox

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func uvRunBindingFixture(t *testing.T) (*pipxNativeInput, []uvRunCensusFact, string) {
	return pythonRunBindingFixture(t, false)
}

func pythonRunBindingFixture(t *testing.T, pipx bool) (*pipxNativeInput, []uvRunCensusFact, string) {
	t.Helper()
	frozen, _, _, _, body := pipxFrozenInputFixture(t, map[string][]byte{"probe-1.0.dist-info/entry_points.txt": []byte("[console_scripts]\nprobe = probe:main\n")})
	_, infos, err := uvRunConsoleOriginal(frozen, "probe")
	if err != nil || len(infos) != 1 {
		t.Fatal("source-shaped whole fixture", err)
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string][]byte{}
	for _, f := range archive.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		closeErr := r.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		payload[f.Name] = b
	}
	env := "uv-run-cache/archive-v0/0123456789abcdef"
	if pipx {
		env = "pipx/.cache/0123456789abcde"
	}
	site := env + "/lib/python3.14/site-packages/"
	var facts []uvRunCensusFact
	for _, name := range []string{env, env + "/bin", env + "/lib", env + "/lib/python3.14", strings.TrimSuffix(site, "/")} {
		facts = append(facts, uvRunCensusFact{Name: name, Kind: tar.TypeDir, UID: 1000, GID: 1000, Mode: 0755})
	}
	for name, target := range map[string]string{"bin/python": "/usr/local/bin/python", "bin/python3": "python", "bin/python3.14": "python", "lib64": "lib"} {
		facts = append(facts, uvRunCensusFact{Name: env + "/" + name, Kind: tar.TypeSymlink, UID: 1000, GID: 1000, Mode: 0777, Link: target})
	}
	regular := func(name string, b []byte, mode int64, capture bool) {
		sum := sha256.Sum256(b)
		f := uvRunCensusFact{Name: name, Kind: tar.TypeReg, UID: 1000, GID: 1000, Mode: mode, Size: int64(len(b)), SHA256: hex.EncodeToString(sum[:])}
		if capture {
			f.Body = bytes.Clone(b)
		}
		facts = append(facts, f)
	}
	configuration := uvToolVenvConfiguration() + "relocatable = true\n"
	if pipx {
		configuration = uvToolVenvConfiguration()
		regular(env+"/pipx_metadata.json", []byte(`{"unapproved":"DATA"}`), 0600, false)
		regular(env+"/.lock", nil, 0666, false)
	}
	regular(env+"/pyvenv.cfg", []byte(configuration), 0644, true)
	for _, f := range infos[0].Surface.InstalledFiles {
		b := payload[f.ArchivePath]
		if strings.HasSuffix(f.Destination, ".dist-info/RECORD") {
			b = []byte("native RECORD is unapproved DATA, not ownership\n")
		}
		regular(site+f.Destination, b, 0644, strings.HasSuffix(f.Destination, "/METADATA") || strings.HasSuffix(f.Destination, "/WHEEL"))
	}
	regular(site+"probe-1.0.dist-info/INSTALLER", []byte("uv"), 0644, false)
	regular(site+"probe-1.0.dist-info/REQUESTED", nil, 0644, false)
	regular(site+"probe-1.0.dist-info/uv_cache.json", []byte(`{"not_source_authority":true}`), 0644, false)
	fixture, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/uv-native-relocatable-launcher.json")
	if err != nil {
		t.Fatal(err)
	}
	var native struct{ Body []byte }
	if json.Unmarshal(fixture, &native) != nil {
		t.Fatal("native wrapper fixture")
	}
	script := bytes.Replace(native.Body, []byte("from cowsay.__main__ import cli"), []byte("from probe import main"), 1)
	script = bytes.Replace(script, []byte("sys.exit(cli())"), []byte("sys.exit(main())"), 1)
	if pipx {
		original, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/uv-native-launchers.json")
		if err != nil {
			t.Fatal(err)
		}
		var native struct {
			Cases []struct {
				Scripts []struct {
					Python string
					Body   []byte
				}
			}
		}
		if json.Unmarshal(original, &native) != nil || len(native.Cases) == 0 || len(native.Cases[0].Scripts) == 0 {
			t.Fatal("native nonrelocatable fixture")
		}
		recorded := native.Cases[0].Scripts[0]
		script = bytes.Replace(recorded.Body, []byte(recorded.Python), []byte(pythonSitePath+"/"+env+"/bin/python"), 1)
		script = bytes.Replace(script, []byte("from haa_tool_fixture import main"), []byte("from probe import main"), 1)
	}
	regular(env+"/bin/probe", script, 0755, true)
	return frozen, facts, env
}

func cloneUVRunFacts(facts []uvRunCensusFact) []uvRunCensusFact {
	result := append([]uvRunCensusFact(nil), facts...)
	for i := range result {
		result[i].Body = bytes.Clone(result[i].Body)
	}
	return result
}

func TestUVRunConsoleDataBindsWholeOriginalAndCanonicalEntryWithoutNativeOwnership(t *testing.T) {
	frozen, facts, _ := uvRunBindingFixture(t)
	entry, err := bindUVRunConsoleData(frozen, "probe", facts)
	if err != nil || !entry.Valid() || entry.Command() != "probe" || entry.EntryPoint().Module != "probe" || entry.EntryPoint().Attr != "main" || entry.Metadata().Path != "probe-1.0.dist-info/entry_points.txt" {
		t.Fatal("source-shaped native binding", err)
	}
	if _, err := bindUVRunConsoleData(frozen, "missing", facts); err == nil {
		t.Fatal("native filename replaced source console selection")
	}
}

func TestUVRunConsoleDataRejectsIncompleteChangedForeignAndUnsafeNativeClaims(t *testing.T) {
	testPythonRunConsoleDataRejectsChangedClaims(t, false)
}

func TestPipxRunConsoleDataRejectsIncompleteChangedForeignAndUnsafeNativeClaims(t *testing.T) {
	testPythonRunConsoleDataRejectsChangedClaims(t, true)
}

func testPythonRunConsoleDataRejectsChangedClaims(t *testing.T, pipx bool) {
	frozen, original, env := pythonRunBindingFixture(t, pipx)
	bind := bindUVRunConsoleData
	if pipx {
		bind = bindPipxRunConsoleData
	}
	change := func(facts []uvRunCensusFact, suffix string, edit func(*uvRunCensusFact)) []uvRunCensusFact {
		for i := range facts {
			if strings.HasSuffix(facts[i].Name, suffix) {
				edit(&facts[i])
				return facts
			}
		}
		t.Fatal("mutation target absent", suffix)
		return nil
	}
	mutations := map[string]func([]uvRunCensusFact) []uvRunCensusFact{
		"duplicate": func(f []uvRunCensusFact) []uvRunCensusFact { return append(f, f[0]) },
		"missing source": func(f []uvRunCensusFact) []uvRunCensusFact {
			for i := range f {
				if strings.HasSuffix(f[i].Name, "/probe/__init__.py") {
					return append(f[:i], f[i+1:]...)
				}
			}
			t.Fatal("source target absent")
			return nil
		},
		"payload digest": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/probe/__init__.py", func(x *uvRunCensusFact) { x.SHA256 = strings.Repeat("a", 64) })
		},
		"changed metadata": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/METADATA", func(x *uvRunCensusFact) { x.Body = append(x.Body, 'x') })
		},
		"body hash mismatch": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/bin/probe", func(x *uvRunCensusFact) { x.Body[0] = '!' })
		},
		"changed wrapper": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/bin/probe", func(x *uvRunCensusFact) {
				x.Body = bytes.Replace(x.Body, []byte("sys.exit(main())"), []byte("sys.exit(other())"), 1)
				x.Size = int64(len(x.Body))
				sum := sha256.Sum256(x.Body)
				x.SHA256 = hex.EncodeToString(sum[:])
			})
		},
		"foreign runtime": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/bin/python", func(x *uvRunCensusFact) { x.Link = "/other/bin/python" })
		},
		"writable wrapper": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/bin/probe", func(x *uvRunCensusFact) { x.Mode = 0777 })
		},
		"wrong owner": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/bin/probe", func(x *uvRunCensusFact) { x.UID = 0 })
		},
		"changed cfg": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/pyvenv.cfg", func(x *uvRunCensusFact) {
				x.Body = []byte("include-system-site-packages = true\n")
				x.Size = int64(len(x.Body))
				sum := sha256.Sum256(x.Body)
				x.SHA256 = hex.EncodeToString(sum[:])
			})
		},
		"foreign helper path": func(f []uvRunCensusFact) []uvRunCensusFact {
			return change(f, "/bin/probe", func(x *uvRunCensusFact) { x.Name = "../bin/probe" })
		},
		"extra package body": func(f []uvRunCensusFact) []uvRunCensusFact {
			return append(f, uvRunCensusFact{Name: env + "/lib/python3.14/site-packages/evil.py", Kind: tar.TypeReg, UID: 1000, GID: 1000, Mode: 0644, Size: 1, SHA256: strings.Repeat("a", 64)})
		},
		"extra console": func(f []uvRunCensusFact) []uvRunCensusFact {
			return append(f, uvRunCensusFact{Name: env + "/bin/evil", Kind: tar.TypeReg, UID: 1000, GID: 1000, Mode: 0755, Size: 1, SHA256: strings.Repeat("a", 64)})
		},
		"foreign link": func(f []uvRunCensusFact) []uvRunCensusFact {
			return append(f, uvRunCensusFact{Name: env + "/lib/python3.14/site-packages/evil.py", Kind: tar.TypeSymlink, UID: 1000, GID: 1000, Mode: 0777, Link: "/etc/passwd"})
		},
		"extra distribution": func(f []uvRunCensusFact) []uvRunCensusFact {
			x := f[0]
			for _, row := range f {
				if strings.HasSuffix(row.Name, "/METADATA") {
					x = row
					break
				}
			}
			x.Name = env + "/lib/python3.14/site-packages/other-1.0.dist-info/METADATA"
			return append(f, x)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			if entry, err := bind(frozen, "probe", mutate(cloneUVRunFacts(original))); err == nil || entry.Valid() {
				t.Fatal("changed DATA accepted")
			}
		})
	}
	if _, err := bind(frozen, "probe", nil); err == nil {
		t.Fatal("empty census accepted")
	}
	frozen.files[0].body[0] ^= 1
	if _, err := bind(frozen, "probe", original); err == nil {
		t.Fatal("stale frozen original accepted")
	}
}

func TestPipxRunConsoleDataBindsOriginalWithoutInstalledToolOwnership(t *testing.T) {
	frozen, facts, _ := pythonRunBindingFixture(t, true)
	entry, err := bindPipxRunConsoleData(frozen, "probe", facts)
	if err != nil || !entry.Valid() || entry.Command() != "probe" {
		t.Fatal("pipx native DATA binding", err)
	}
	if _, err := bindUVRunConsoleData(frozen, "probe", facts); err == nil {
		t.Fatal("pipx cache was accepted as UV relocatable geometry")
	}
	for _, fault := range []string{"lock content", "lock owner", "metadata absent"} {
		changed := cloneUVRunFacts(facts)
		for i := range changed {
			if fault == "metadata absent" && strings.HasSuffix(changed[i].Name, "/pipx_metadata.json") {
				changed = append(changed[:i], changed[i+1:]...)
				break
			}
			if strings.HasSuffix(changed[i].Name, "/.lock") {
				if fault == "lock content" {
					changed[i].Size = 1
				} else if fault == "lock owner" {
					changed[i].UID = 0
				}
			}
		}
		if result, err := bindPipxRunConsoleData(frozen, "probe", changed); err == nil || result.Valid() {
			t.Fatal("native lock/metadata guard failed", fault)
		}
	}
	for _, name := range []string{"pipx/.cache/foreign", "tool-bin/foreign"} {
		changed := append(cloneUVRunFacts(facts), uvRunCensusFact{Name: name, Kind: tar.TypeReg, UID: 1000, GID: 1000, Mode: 0644, Size: 1})
		if result, err := bindPipxRunConsoleData(frozen, "probe", changed); err == nil || result.Valid() {
			t.Fatal("foreign outer output accepted")
		}
	}
}
