package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func toolReplayFixture(t *testing.T, collision bool) (artifactpypi.UVToolUpgradeInput, []artifactpypi.UVInstalledWheelControls) {
	t.Helper()
	body, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/uv-installed.toml")
	if err != nil {
		t.Fatal(err)
	}
	if collision {
		body = bytes.ReplaceAll(body, []byte("haa-demo"), []byte("python"))
	}
	var wire map[string]any
	if err := toml.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	options := wire["tool"].(map[string]any)["options"].(map[string]any)
	delete(options, "no-index")
	delete(options, "find-links")
	body, err = toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := artifactpypi.ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = receipt.WithUVLayout("/user/private/haa-tool-fixture", "/user/commands")
	if err != nil {
		t.Fatal(err)
	}
	metadata := []byte("Metadata-Version: 2.4\nName: haa-tool-fixture\nVersion: 1.0.0\n")
	directory := "haa_tool_fixture-1.0.0.dist-info"
	request, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture>=1.1,<2")
	input, err := artifactpypi.NewUVToolUpgradeInput(receipt, request, []artifactpypi.InstalledMetadataFile{{Directory: directory, Body: metadata}}, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	return input, []artifactpypi.UVInstalledWheelControls{{DistInfo: directory, Metadata: metadata, Wheel: []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")}}
}

func TestUVToolReplayArchiveContainsOnlyControlledMetadataAndInertCommands(t *testing.T) {
	input, controls := toolReplayFixture(t, false)
	body, err := uvToolUpgradeReplayArchive(input, controls)
	if err != nil {
		t.Fatal(err)
	}
	original := input.Receipt().Body()
	if bytes.Contains(body, []byte("/user/private")) || bytes.Contains(body, []byte("/user/commands")) {
		t.Fatal("Host paths leaked into native replay")
	}
	if !bytes.Equal(input.Receipt().Body(), original) {
		t.Fatal("original receipt changed")
	}
	scanned, err := scanUVToolNativeOutput(bytes.NewReader(body), "haa-tool-fixture", nil)
	if err != nil || len(scanned.controls) != 1 {
		t.Fatal("existing whole native reader rejected controlled replay", err)
	}
	reader := tar.NewReader(bytes.NewReader(body))
	headers := map[string]tar.Header{}
	files := map[string][]byte{}
	for {
		h, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(h.Name, "/")
		if _, exists := headers[name]; exists {
			t.Fatal("duplicate tar member")
		}
		headers[name] = *h
		if h.Uid != 1000 || h.Gid != 1000 {
			t.Fatal("replay owner differs")
		}
		if h.Typeflag == tar.TypeReg {
			b, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			files[name] = b
			if strings.HasPrefix(name, "tools/haa-tool-fixture/bin/") && (h.Mode&0111 != 0 || string(b) != uvToolReplayCommand) {
				t.Fatal("original or executable command transported")
			}
		}
	}
	env := "tools/haa-tool-fixture"
	site := env + "/lib/python3.14/site-packages"
	for _, name := range []string{"tools", "tool-bin", env, env + "/bin", site} {
		if h := headers[name]; h.Typeflag != tar.TypeDir || h.Mode != 0700 {
			t.Fatal("missing private replay directory", name)
		}
	}
	allowed := map[string]bool{"tools/.gitignore": true, "tools/.lock": true, env + "/.gitignore": true, env + "/CACHEDIR.TAG": true, env + "/pyvenv.cfg": true, env + "/uv-receipt.toml": true, env + "/bin/haa-alt": true, env + "/bin/haa-demo": true, site + "/haa_tool_fixture-1.0.0.dist-info/METADATA": true, site + "/haa_tool_fixture-1.0.0.dist-info/WHEEL": true, site + "/haa_tool_fixture-1.0.0.dist-info/RECORD": true}
	for name := range files {
		if !allowed[name] {
			t.Fatal("program/startup/unknown file entered replay", name)
		}
	}
	if string(files[env+"/pyvenv.cfg"]) != uvToolVenvConfiguration() {
		t.Fatal("runtime configuration changed")
	}
	for name, target := range map[string]string{env + "/bin/python": "/usr/local/bin/python", env + "/bin/python3": "python", env + "/bin/python3.14": "python", "tool-bin/haa-demo": uvNativeToolHome + "/haa-tool-fixture/bin/haa-demo"} {
		if h := headers[name]; h.Typeflag != tar.TypeSymlink || h.Linkname != target || h.Mode != 0777 {
			t.Fatal("runtime/exposure alias differs", name)
		}
	}
	record := files[site+"/haa_tool_fixture-1.0.0.dist-info/RECORD"]
	rows, err := csv.NewReader(bytes.NewReader(record)).ReadAll()
	if err != nil || len(rows) != 5 {
		t.Fatal("generated RECORD differs", err)
	}
	for _, row := range rows {
		if len(row) != 3 {
			t.Fatal("record width")
		}
		name := path.Clean(site + "/" + row[0])
		if strings.HasSuffix(name, "/RECORD") {
			if row[1] != "" || row[2] != "" {
				t.Fatal("RECORD self hash")
			}
			continue
		}
		data, ok := files[name]
		if !ok {
			t.Fatal("RECORD points outside generated files", name)
		}
		sum := sha256.Sum256(data)
		if row[1] != "sha256="+base64.RawURLEncoding.EncodeToString(sum[:]) {
			t.Fatal("generated RECORD digest differs")
		}
	}
	repeated, err := uvToolUpgradeReplayArchive(input, controls)
	if err != nil || !bytes.Equal(repeated, body) {
		t.Fatal("replay archive is not deterministic", err)
	}
}

type toolReplayInputRunner struct {
	calls int
	name  string
	args  []string
	body  []byte
	err   error
}

func (r *toolReplayInputRunner) RunInput(_ context.Context, input io.Reader, name string, args ...string) error {
	r.calls++
	r.name = name
	r.args = append([]string(nil), args...)
	r.body, _ = io.ReadAll(input)
	return r.err
}

func TestUVToolReplayTransportIsClosedAndPreservesPrimaryFailure(t *testing.T) {
	input, controls := toolReplayFixture(t, false)
	container := strings.Repeat("a", 64)
	primary := errors.New("input transport stopped")
	runner := &toolReplayInputRunner{err: primary}
	if err := introduceUVToolUpgradeInput(context.Background(), runner, container, input, controls); !errors.Is(err, primary) {
		t.Fatal("primary failure replaced", err)
	}
	if runner.calls != 1 || runner.name != "docker" || !reflect.DeepEqual(runner.args, []string{"cp", "--archive", "-", container + ":" + pythonSitePath}) || len(runner.body) == 0 {
		t.Fatal("arbitrary transport shape")
	}
	for _, mode := range []string{"nil-context", "cancelled", "invalid-container", "missing-WHEEL", "alias-collision"} {
		t.Run(mode, func(t *testing.T) {
			input, controls := toolReplayFixture(t, mode == "alias-collision")
			ctx := context.Background()
			id := container
			if mode == "nil-context" {
				ctx = nil
			}
			if mode == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if mode == "invalid-container" {
				id = "not-owned"
			}
			if mode == "missing-WHEEL" {
				controls[0].Wheel = nil
			}
			runner := &toolReplayInputRunner{}
			if err := introduceUVToolUpgradeInput(ctx, runner, id, input, controls); err == nil || runner.calls != 0 {
				t.Fatal("invalid replay reached transport")
			}
		})
	}
}
