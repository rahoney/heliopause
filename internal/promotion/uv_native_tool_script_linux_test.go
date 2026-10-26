package promotion

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

type nativeToolScriptFixture struct {
	proof   *uvNativeToolScriptProof
	project *pypiNativeDistribution
	wheel   artifactpypi.WheelInspection
	digest  string
	root    string
	close   func() error
}

func makeNativeToolScriptFixture(t *testing.T, mode string) nativeToolScriptFixture {
	t.Helper()
	var fixture struct {
		Filename, SHA256 string
		Body             []byte
	}
	data, err := os.ReadFile("testdata/uv-native-tool-wheel.json")
	if err != nil || json.Unmarshal(data, &fixture) != nil {
		t.Fatal("read synthetic native producer input", err)
	}
	sum := sha256.Sum256(fixture.Body)
	if hex.EncodeToString(sum[:]) != fixture.SHA256 {
		t.Fatal("native producer archive fixture changed")
	}
	wheel, err := artifactpypi.InspectWheel(bytes.NewReader(fixture.Body), int64(len(fixture.Body)), fixture.Filename,
		fixture.SHA256, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, artifactpypi.DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	parent, err := os.MkdirTemp("", "haa-n2a-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	root := filepath.Join(parent, "haa-tool-fixture")
	site := filepath.Join(root, "lib/python3.14/site-packages")
	for _, p := range []string{site, filepath.Join(root, "bin")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p string, b []byte, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, mode); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "pyvenv.cfg"), []byte("implementation = CPython\nversion_info = 3.14.7\nuv = 0.13.4\ninclude-system-site-packages = false\n"), 0600)
	write(filepath.Join(root, "bin/python3.14"), []byte("inert original interpreter DATA; never execute"), 0700)
	if err := os.Symlink("python3.14", filepath.Join(root, "bin/python")); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(fixture.Body), int64(len(fixture.Body)))
	if err != nil {
		t.Fatal(err)
	}
	installed := map[string][]byte{}
	const dist = "haa_tool_fixture-1.0.0.dist-info"
	for _, f := range archive.File {
		if f.Name == dist+"/RECORD" {
			continue
		}
		reader, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		installed[f.Name] = body
	}
	installed[dist+"/INSTALLER"], installed[dist+"/REQUESTED"] = []byte("uv"), []byte{}
	var launchers struct {
		WheelSHA256 string `json:"wheel_sha256"`
		Cases       []struct {
			Name    string
			Scripts []struct {
				Name, Python, SHA256 string
				Body                 []byte
			}
		}
	}
	data, err = os.ReadFile("../artifact/pypi/testdata/persistent-tool/uv-native-launchers.json")
	if err != nil || json.Unmarshal(data, &launchers) != nil || launchers.WheelSHA256 != fixture.SHA256 ||
		len(launchers.Cases) != 5 || launchers.Cases[0].Name != "short" || len(launchers.Cases[0].Scripts) != 2 {
		t.Fatal("independent native wrapper fixture differs", err)
	}
	python := filepath.Join(root, "bin/python")
	if len(python)+3 > 127 || strings.ContainsAny(python, " '\r\n") {
		t.Fatal("short isolated fixture path is unavailable")
	}
	for _, s := range launchers.Cases[0].Scripts {
		sum := sha256.Sum256(s.Body)
		if hex.EncodeToString(sum[:]) != s.SHA256 || !bytes.HasPrefix(s.Body, []byte("#!"+s.Python+"\n")) {
			t.Fatal("native generated fixture changed")
		}
		body := bytes.Replace(s.Body, []byte("#!"+s.Python+"\n"), []byte("#!"+python+"\n"), 1)
		installed["../../../bin/"+s.Name] = body
	}
	switch mode {
	case "script-code":
		installed["../../../bin/haa-demo"] = bytes.Replace(installed["../../../bin/haa-demo"], []byte("sys.exit(main())"), []byte("sys.exit(0)"), 1)
	case "script-interpreter":
		installed["../../../bin/haa-demo"] = bytes.Replace(installed["../../../bin/haa-demo"], []byte("#!"+python+"\n"), []byte("#!"+python+"3.14\n"), 1)
	case "missing-script":
		delete(installed, "../../../bin/haa-alt")
	case "extra-script":
		installed["../../../bin/unclaimed"] = []byte("inert unclaimed script")
	case "extra-cache":
		installed[dist+"/uv_cache.json"] = []byte("{}")
	case "installer":
		installed[dist+"/INSTALLER"] = []byte("uv\n")
	}
	keys := make([]string, 0, len(installed))
	for key := range installed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var record bytes.Buffer
	w := csv.NewWriter(&record)
	for _, key := range keys {
		body := installed[key]
		sum := sha256.Sum256(body)
		if err := w.Write([]string{key, "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), strconv.Itoa(len(body))}); err != nil {
			t.Fatal(err)
		}
		permission := os.FileMode(0600)
		if strings.HasPrefix(key, "../../../bin/") {
			permission = 0700
		}
		if mode == "nonexecutable" && key == "../../../bin/haa-demo" {
			permission = 0600
		}
		write(filepath.Clean(filepath.Join(site, filepath.FromSlash(key))), body, permission)
	}
	if err := w.Write([]string{dist + "/RECORD", "", ""}); err != nil {
		t.Fatal(err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(site, dist, "RECORD"), record.Bytes(), 0600)
	plan, err := discoverUVPythonVenv(root)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginPyPIVenvTransactionWithDiscovery(plan, discoverUVPythonVenv)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	close := func() error {
		if closed {
			return nil
		}
		closed = true
		return tx.close()
	}
	t.Cleanup(func() {
		if err := close(); err != nil {
			t.Error(err)
		}
	})
	inventory, err := capturePyPINativeInventoryFromMetadata(context.Background(), tx,
		[]artifactpypi.InstalledMetadataFile{{Directory: dist, Body: installed[dist+"/METADATA"]}})
	if err != nil {
		t.Fatal(err)
	}
	proof, err := newUVNativeToolScriptProof(context.Background(), tx)
	if err != nil {
		t.Fatal(err)
	}
	return nativeToolScriptFixture{proof, inventory.projects["haa-tool-fixture"], wheel, fixture.SHA256, root, close}
}

func TestNativeUVToolScriptFootprintBindsWholeArchiveAndHeldInterpreter(t *testing.T) {
	f := makeNativeToolScriptFixture(t, "valid")
	destinations, files, err := f.proof.prove(context.Background(), f.project, f.wheel, f.digest)
	if err != nil || len(destinations) != len(f.project.files) || len(files) != len(destinations) {
		t.Fatal("complete footprint proof failed", err)
	}
	scripts := 0
	for _, file := range files {
		if file.Scheme == "scripts" {
			scripts++
			if file.Role != "pinned-uv-console-script" || (file.Relative != "haa-demo" && file.Relative != "haa-alt") {
				t.Fatal("script role/source binding differs", file)
			}
		}
	}
	if scripts != 2 || len(f.proof.tx.current.Files) != 0 {
		t.Fatal("proof did not cover two scripts or enrolled ownership")
	}
	if _, err := os.Lstat(filepath.Join(f.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("proof published ownership metadata", err)
	}
	for _, client := range []pypiNativeClient{nativeUVClient, nativePipClient} {
		if _, _, err := proveNativePythonWheelFootprint(client, f.project, f.wheel, f.digest, nil); err == nil {
			t.Fatal("ordinary native footprint admitted generated scripts")
		}
	}
}

func TestNativeUVToolScriptFootprintRejectsRecordOnlyClaimsAndIncompleteArchive(t *testing.T) {
	for _, mode := range []string{"script-code", "script-interpreter", "missing-script", "extra-script", "nonexecutable", "extra-cache", "installer", "wheel-digest", "wheel-project", "wheel-entrypoint", "gui", "extras", "duplicate-entrypoint"} {
		t.Run(mode, func(t *testing.T) {
			f := makeNativeToolScriptFixture(t, mode)
			switch mode {
			case "wheel-digest":
				f.digest = strings.Repeat("f", 64)
			case "wheel-project":
				f.wheel.Project = "other"
			case "wheel-entrypoint":
				f.wheel.Surface.EntryPointDetails[0].Attr = "other"
			case "gui":
				f.wheel.Surface.EntryPointDetails[0].Group = "gui_scripts"
			case "extras":
				f.wheel.Surface.EntryPointDetails[0].Extras = []string{"other"}
			case "duplicate-entrypoint":
				f.wheel.Surface.EntryPointDetails = append(f.wheel.Surface.EntryPointDetails, f.wheel.Surface.EntryPointDetails[0])
			}
			if _, _, err := f.proof.prove(context.Background(), f.project, f.wheel, f.digest); err == nil || len(f.proof.tx.current.Files) != 0 {
				t.Fatal("unproven script/original footprint accepted or ownership mutated")
			}
		})
	}
}

func TestNativeUVToolScriptProofRejectsInterpreterAndGuardDrift(t *testing.T) {
	for _, mode := range []string{"python-bytes", "python-alias", "config", "closed", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			f := makeNativeToolScriptFixture(t, "valid")
			ctx := context.Background()
			switch mode {
			case "python-bytes":
				if err := os.WriteFile(filepath.Join(f.root, "bin/python3.14"), []byte("changed inert bytes"), 0700); err != nil {
					t.Fatal(err)
				}
			case "python-alias":
				before, err := os.Lstat(filepath.Join(f.root, "bin/python"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(f.root, "bin/python"), filepath.Join(f.root, "bin/python-original-link")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("python3.14", filepath.Join(f.root, "bin/python")); err != nil {
					t.Fatal(err)
				}
				after, err := os.Lstat(filepath.Join(f.root, "bin/python"))
				if err != nil || os.SameFile(before, after) {
					t.Fatal("replacement alias must have a distinct retained inode", err)
				}
			case "config":
				if err := os.WriteFile(filepath.Join(f.root, "pyvenv.cfg"), []byte("version = 3.14.7\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "closed":
				if err := f.close(); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, _, err := f.proof.prove(ctx, f.project, f.wheel, f.digest); err == nil {
				t.Fatal("drifted original interpreter/guard accepted")
			}
		})
	}
}
