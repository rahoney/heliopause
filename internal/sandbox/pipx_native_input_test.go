package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func pipxFrozenInputFixture(t *testing.T, extra map[string][]byte) (*pipxNativeInput, artifactpypi.UVLock, []artifactpypi.UVWheelInput, []artifactpypi.SimpleProject, []byte) {
	t.Helper()
	body := linuxDynamicProjectWheel(t, "probe", "raise RuntimeError('never execute')\n", extra)
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])
	filename := "probe-1.0-py3-none-any.whl"
	url := "https://files.pythonhosted.org/packages/" + filename
	wire := fmt.Sprintf("lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=%s'\n[[packages]]\nname='probe'\nversion='1.0'\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", PinnedPythonRuntime().PythonVersion, url, len(body), digest)
	reference, err := artifactpypi.ParseReference("probe@1.0")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := artifactpypi.ParseUVLock(reference, []byte(wire), PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(map[string]any{"info": map[string]any{"name": "probe", "version": "1.0"}, "urls": []any{map[string]any{"filename": filename, "url": url, "size": len(body), "digests": map[string]string{"sha256": digest}, "requires_python": "", "yanked": false, "packagetype": "bdist_wheel"}}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := artifactpypi.ParseUVReleaseMetadata("probe", "1.0", metadata)
	if err != nil {
		t.Fatal(err)
	}
	inputs := []artifactpypi.UVWheelInput{{URL: url, Reader: bytes.NewReader(body), Size: int64(len(body))}}
	pages := []artifactpypi.SimpleProject{page}
	frozen, err := freezePipxNativeInput(context.Background(), lock, inputs, pages)
	if extra == nil && err != nil {
		t.Fatal(err)
	}
	if extra != nil && (err == nil || !strings.Contains(err.Error(), "startup")) {
		t.Fatal("startup fixture failed outside expected gate", err)
	}
	return frozen, lock, inputs, pages, body
}

type pipxInputTransport struct {
	body  []byte
	calls int
	fail  error
}

func (f *pipxInputTransport) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("process execution forbidden")
}
func (f *pipxInputTransport) RunInput(_ context.Context, r io.Reader, name string, args ...string) error {
	f.calls++
	if name != "docker" || len(args) != 4 || args[0] != "cp" || args[1] != "--archive" || args[2] != "-" || args[3] != strings.Repeat("a", 64)+":/haa-wheels" {
		return errors.New("wrong fixed frozen transport")
	}
	if f.fail != nil {
		return f.fail
	}
	var err error
	f.body, err = io.ReadAll(r)
	return err
}

func TestPipxNativeInputOwnsBoundOriginalBytesAndDeterministicIdentity(t *testing.T) {
	frozen, lock, inputs, pages, body := pipxFrozenInputFixture(t, nil)
	second, err := freezePipxNativeInput(context.Background(), lock, inputs, pages)
	if err != nil || frozen.id != second.id || !frozen.valid() || len(frozen.graph.Nodes()) != 1 {
		t.Fatal("frozen identity or graph differs", err)
	}
	original := bytes.Clone(frozen.files[0].body)
	body[0] ^= 1
	if !frozen.valid() || !bytes.Equal(frozen.files[0].body, original) {
		t.Fatal("caller can mutate frozen input")
	}
	failed, err := freezePipxNativeInput(context.Background(), lock, inputs, pages)
	if err == nil || failed != nil {
		t.Fatal("changed caller archive accepted")
	}
	f := &pipxInputTransport{}
	if err := frozen.introduce(context.Background(), f, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := frozen.scanTar(bytes.NewReader(f.body)); err != nil {
		t.Fatal("frozen tar readback", err)
	}
	cause := errors.New("copy failed")
	f.fail = cause
	if err := frozen.introduce(context.Background(), f, strings.Repeat("a", 64)); !errors.Is(err, cause) {
		t.Fatal("primary copy failure lost", err)
	}
}

func TestPipxNativeInputRejectsBeforeTransportAndBeforeStartup(t *testing.T) {
	frozen, lock, inputs, pages, _ := pipxFrozenInputFixture(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := freezePipxNativeInput(ctx, lock, inputs, pages); err == nil || got != nil {
		t.Fatal("canceled input accepted")
	}
	for _, input := range [][]artifactpypi.UVWheelInput{nil, append(inputs, inputs[0]), {{URL: inputs[0].URL, Reader: inputs[0].Reader, Size: inputs[0].Size + 1}}, {{URL: inputs[0].URL, Reader: nil, Size: inputs[0].Size}}} {
		if got, err := freezePipxNativeInput(context.Background(), lock, input, pages); err == nil || got != nil {
			t.Fatal("invalid original set accepted")
		}
	}
	if got, err := freezePipxNativeInput(context.Background(), lock, inputs, nil); err == nil || got != nil {
		t.Fatal("missing independent metadata accepted")
	}
	bad, _, _, _, _ := pipxFrozenInputFixture(t, map[string][]byte{"startup.pth": []byte("import os\n")})
	if bad != nil {
		t.Fatal("startup hook frozen as native query input")
	}
	f := &pipxInputTransport{}
	frozen.files[0].body[0] ^= 1
	if err := frozen.introduce(context.Background(), f, strings.Repeat("a", 64)); err == nil || f.calls != 0 {
		t.Fatal("changed snapshot reached Docker")
	}
	if err := (*pipxNativeInput)(nil).introduce(context.Background(), f, strings.Repeat("a", 64)); err == nil || f.calls != 0 {
		t.Fatal("nil snapshot reached Docker")
	}
}

func TestPipxNativeInputReadbackRejectsChangedWholeTar(t *testing.T) {
	input, _, _, _, _ := pipxFrozenInputFixture(t, nil)
	for _, fault := range []string{"owner", "mode", "link", "name", "size", "payload", "duplicate", "missing", "trailing"} {
		t.Run(fault, func(t *testing.T) {
			var output bytes.Buffer
			writer := tar.NewWriter(&output)
			file := input.files[0]
			body := bytes.Clone(file.body)
			h := tar.Header{Name: file.name, Typeflag: tar.TypeReg, Mode: 0444, Uid: 0, Gid: 0, Size: int64(len(body))}
			switch fault {
			case "owner":
				h.Uid = 1000
			case "mode":
				h.Mode = 0644
			case "link":
				h.Typeflag = tar.TypeSymlink
				h.Linkname = "/outside"
				h.Size = 0
				body = nil
			case "name":
				h.Name = "../" + file.name
			case "size":
				h.Size--
				body = body[:len(body)-1]
			case "payload":
				body[0] ^= 1
			}
			if fault != "missing" {
				if err := writer.WriteHeader(&h); err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write(body); err != nil {
					t.Fatal(err)
				}
				if fault == "duplicate" {
					if err := writer.WriteHeader(&h); err != nil {
						t.Fatal(err)
					}
					if _, err := writer.Write(body); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			encoded := output.Bytes()
			if fault == "trailing" {
				encoded = append(encoded, 1)
			}
			if err := input.scanTar(bytes.NewReader(encoded)); err == nil {
				t.Fatal("invalid original tar accepted")
			}
		})
	}
}
