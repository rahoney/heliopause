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
	"path"
	"strconv"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func pipxReplayFixture(t *testing.T, collision bool) (artifactpypi.PipxToolUpgradeInput, []artifactpypi.UVInstalledWheelControls) {
	t.Helper()
	frozen, entries := pipxNativeOutputFixture(t)
	output, err := readPipxNativeOutput(context.Background(), &pipxNativeOutputTransport{body: uvToolFixtureTar(t, entries)}, strings.Repeat("a", 64), "probe", frozen)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := output.receipt.WithPipxLayout("/user/held/probe", "/user/commands", "/user/python")
	if err != nil {
		t.Fatal(err)
	}
	if collision {
		receipt, err = artifactpypi.ParsePipxToolReceipt(bytes.ReplaceAll(receipt.Body(), []byte("haa-demo"), []byte("python")))
		if err != nil {
			t.Fatal(err)
		}
	}
	var metadata []artifactpypi.InstalledMetadataFile
	for _, control := range output.controls {
		metadata = append(metadata, artifactpypi.InstalledMetadataFile{Directory: control.DistInfo, Body: control.Metadata})
	}
	input, err := artifactpypi.NewPipxToolUpgradeInput(receipt, metadata, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	return input, output.controls
}

func TestPipxOriginalReplayContainsOnlyControlledMetadataAndInertScaffold(t *testing.T) {
	input, controls := pipxReplayFixture(t, false)
	original := input.Receipt().Body()
	body, err := pipxToolUpgradeReplayArchive(input, controls)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input.Receipt().Body(), original) || bytes.Contains(body, []byte("/user/held")) || bytes.Contains(body, []byte("/user/commands")) || bytes.Contains(body, []byte("/user/python")) {
		t.Fatal("original changed or Host paths entered guest")
	}
	reader := tar.NewReader(bytes.NewReader(body))
	files := map[string][]byte{}
	headers := map[string]tar.Header{}
	for {
		h, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(h.Name, "/")
		if _, found := headers[name]; found || h.Uid != 1000 || h.Gid != 1000 {
			t.Fatal("duplicate/foreign replay member", name)
		}
		headers[name] = *h
		if h.Typeflag == tar.TypeDir {
			if h.Mode != 0700 {
				t.Fatal("unsafe replay directory")
			}
			continue
		}
		if h.Typeflag == tar.TypeSymlink {
			if h.Mode != 0777 || h.Size != 0 {
				t.Fatal("invalid generated alias")
			}
			continue
		}
		if h.Typeflag != tar.TypeReg {
			t.Fatal("unexpected replay type")
		}
		files[name], err = io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
	}
	env := "pipx/venvs/probe"
	site := env + "/lib/python3.14/site-packages"
	allowed := map[string]bool{"pipx/.cache/CACHEDIR.TAG": true, "pipx/py/CACHEDIR.TAG": true, "pipx/venvs/.probe.lock": true, env + "/.gitignore": true, env + "/CACHEDIR.TAG": true, env + "/.lock": true, env + "/pyvenv.cfg": true, env + "/pipx_metadata.json": true, env + "/bin/haa-alt": true, env + "/bin/haa-demo": true}
	for _, control := range controls {
		dir := site + "/" + control.DistInfo
		for _, file := range []struct {
			name string
			body []byte
		}{{dir + "/METADATA", control.Metadata}, {dir + "/WHEEL", control.Wheel}} {
			allowed[file.name] = true
			if !bytes.Equal(files[file.name], file.body) || headers[file.name].Mode != 0400 {
				t.Fatal("original control or protected mode differs")
			}
		}
		allowed[dir+"/RECORD"] = true
		records, err := csv.NewReader(bytes.NewReader(files[dir+"/RECORD"])).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range records {
			if len(row) != 3 {
				t.Fatal("generated RECORD shape")
			}
			if row[0] == control.DistInfo+"/RECORD" {
				if row[1] != "" || row[2] != "" {
					t.Fatal("RECORD self-hash")
				}
				continue
			}
			name := path.Join(site, row[0])
			content, ok := files[name]
			if !ok {
				t.Fatal("RECORD escaped generated view")
			}
			sum := sha256.Sum256(content)
			if row[1] != "sha256="+base64.RawURLEncoding.EncodeToString(sum[:]) || row[2] != strconv.Itoa(len(content)) {
				t.Fatal("generated RECORD digest differs")
			}
		}
	}
	for name := range files {
		if !allowed[name] {
			t.Fatal("unreviewed original payload", name)
		}
	}
	if len(files) != len(allowed) {
		t.Fatal("missing controlled member")
	}
	for _, name := range []string{"haa-alt", "haa-demo"} {
		if string(files[env+"/bin/"+name]) != uvToolReplayCommand || headers[env+"/bin/"+name].Mode != 0400 || headers["tool-bin/"+name].Linkname != pipxNativeHome+"/venvs/probe/bin/"+name {
			t.Fatal("executable/original command entered replay")
		}
	}
	if headers[env+"/bin/python"].Linkname != "/usr/local/bin/python" || string(files[env+"/pyvenv.cfg"]) != uvToolVenvConfiguration() {
		t.Fatal("fixed interpreter view differs")
	}
	receipt, err := artifactpypi.ParsePipxToolReceipt(files[env+"/pipx_metadata.json"])
	if err != nil || !receipt.Requirement().SameConstraints(input.Receipt().Requirement()) || input.Request().Requirement() != "probe" {
		t.Fatal("original constraint or unconstrained upgrade changed", err)
	}
	if _, err := scanPipxNativeOutput(bytes.NewReader(body), "probe"); err == nil {
		t.Fatal("initial metadata view became completed native output without genuine native log")
	}
}

type pipxReplayCapture struct {
	calls   int
	args    []string
	body    []byte
	failure error
}

func (r *pipxReplayCapture) RunInput(_ context.Context, input io.Reader, name string, args ...string) error {
	r.calls++
	if name != "docker" {
		return errors.New("unexpected command")
	}
	r.args = append([]string(nil), args...)
	var err error
	r.body, err = io.ReadAll(input)
	return errors.Join(r.failure, err)
}

func TestPipxReplayRejectsBeforeTransportAndPreservesProducerFailure(t *testing.T) {
	for _, scenario := range []string{"empty", "missing", "changed", "collision", "container", "cancelled", "context", "runner"} {
		t.Run(scenario, func(t *testing.T) {
			input, controls := pipxReplayFixture(t, scenario == "collision")
			runner := &pipxReplayCapture{}
			var port inputCommandRunner = runner
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			id := strings.Repeat("a", 64)
			switch scenario {
			case "empty":
				input = artifactpypi.PipxToolUpgradeInput{}
			case "missing":
				controls = nil
			case "changed":
				controls[0].Metadata = append(append([]byte(nil), controls[0].Metadata...), []byte("Summary: changed\n")...)
			case "container":
				id = "named"
			case "cancelled":
				cancel()
			case "context":
				ctx = nil
			case "runner":
				port = nil
			}
			err := introducePipxToolUpgradeInput(ctx, port, id, input, controls)
			if err == nil || runner.calls != 0 || len(runner.body) != 0 {
				t.Fatal("invalid original reached transport", err)
			}
			if scenario == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost")
			}
		})
	}
	input, controls := pipxReplayFixture(t, false)
	cause := errors.New("original copy failed")
	runner := &pipxReplayCapture{failure: cause}
	id := strings.Repeat("b", 64)
	if err := introducePipxToolUpgradeInput(context.Background(), runner, id, input, controls); !errors.Is(err, cause) || runner.calls != 1 || len(runner.body) == 0 || strings.Join(runner.args, " ") != "cp --archive - "+id+":/haa-site" {
		t.Fatal("fixed transport/primary failure differs", err)
	}
}

func TestToolReplaySerializerRetainsByteLimit(t *testing.T) {
	files := map[string]uvToolReplayMember{"pipx/oversized": {body: bytes.Repeat([]byte("x"), 2*uvNativeLockLimit+1), mode: 0400}}
	if body, err := serializeToolReplayArchive(files, map[string]bool{"pipx": true}); err == nil || len(body) != 0 {
		t.Fatal("oversized replay returned payload")
	}
}
