//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pipxDataRoundTrip struct {
	body      []byte
	calls     int
	container string
}

func (f *pipxDataRoundTrip) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("DATA qualification forbids process execution")
}
func (f *pipxDataRoundTrip) RunInput(_ context.Context, r io.Reader, name string, args ...string) error {
	if name != "docker" || !reflect.DeepEqual(args, []string{"cp", "--archive", "-", f.container + ":/haa-pipx"}) {
		return errors.New("wrong fixed DATA copy destination")
	}
	b, err := io.ReadAll(io.LimitReader(r, runtimeidentity.PipxToolSize+64<<10+1))
	if err != nil {
		return err
	}
	f.body = b
	f.calls++
	return nil
}
func (f *pipxDataRoundTrip) RunOutput(_ context.Context, w io.Writer, name string, args ...string) error {
	if name != "docker" || !reflect.DeepEqual(args, []string{"cp", f.container + ":" + pipxToolGuestPath, "-"}) {
		return errors.New("wrong fixed DATA readback source")
	}
	f.calls++
	_, err := w.Write(f.body)
	return err
}

func TestPipxProtectedSourceInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_PIPX_SOURCE_NAMESPACE") != "1" {
		t.Skip("explicit protected pipx DATA source qualification")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires Root in disposable namespaces")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		host, other := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || other != nil || self == host {
			t.Fatal("refusing outside disposable namespaces")
		}
	}
	input := os.Getenv("HELOX_PIPX_RUNTIME_ZIP")
	if !filepath.IsAbs(input) {
		t.Fatal("declare immutable actual ZIP input")
	}
	body, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if int64(len(body)) != runtimeidentity.PipxToolSize || hex.EncodeToString(sum[:]) != runtimeidentity.PipxToolSHA256 {
		t.Fatal("actual ZIP differs from selected owner")
	}
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.Chmod(root, 0755))
	version := filepath.Join(root, "versions", "v1.0.0-probe")
	must(os.MkdirAll(version, 0755))
	name := filepath.Join(version, runtimeidentity.PipxRuntimeAssetName)
	must(os.WriteFile(name, body, 0444))
	current := filepath.Join(root, "current")
	must(os.Symlink("versions/v1.0.0-probe", current))
	tool, err := readInstalledPipxTool(context.Background(), root)
	if err != nil || tool == nil {
		t.Fatal("exact protected DATA refused", err)
	}
	f := &pipxDataRoundTrip{container: strings.Repeat("b", 64)}
	must(tool.introduce(context.Background(), f, f.container))
	if f.calls != 2 {
		t.Fatal("copy/readback incomplete")
	}
	id := nativeToolFileIdentity{runtimeidentity.PipxToolSize, runtimeidentity.PipxToolSHA256, 0444}
	got, err := scanNativeToolTar(f.body, id, nativeRuntimePipx)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatal("actual ZIP tar changed", err)
	}
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimePNPM, nativeRuntimeBun, nativeRuntimeToolKind(99)} {
		if _, err := scanNativeToolTar(f.body, id, kind); err == nil {
			t.Fatal("ZIP became another tool")
		}
	}
	for _, fault := range []string{"executable", "writable", "owner", "hardlink", "symlink", "byte", "current", "directory"} {
		t.Run(fault, func(t *testing.T) {
			restore := func() {}
			switch fault {
			case "executable", "writable":
				mode := os.FileMode(0555)
				if fault == "writable" {
					mode = 0644
				}
				must(os.Chmod(name, mode))
				restore = func() { must(os.Chmod(name, 0444)) }
			case "owner":
				must(os.Chown(name, 1000, 1000))
				restore = func() { must(os.Chown(name, 0, 0)) }
			case "hardlink":
				link := name + ".other"
				must(os.Link(name, link))
				restore = func() { must(os.Remove(link)) }
			case "symlink":
				saved := name + ".saved"
				must(os.Rename(name, saved))
				must(os.Symlink(filepath.Base(saved), name))
				restore = func() { must(os.Remove(name)); must(os.Rename(saved, name)) }
			case "byte":
				changed := append([]byte(nil), body...)
				changed[0] ^= 1
				must(os.WriteFile(name, changed, 0444))
				restore = func() { must(os.WriteFile(name, body, 0444)) }
			case "current":
				must(os.Lchown(current, 1000, 1000))
				restore = func() { must(os.Lchown(current, 0, 0)) }
			case "directory":
				must(os.Chmod(version, 0775))
				restore = func() { must(os.Chmod(version, 0755)) }
			}
			defer restore()
			if got, err := readInstalledPipxTool(context.Background(), root); got != nil || err == nil {
				t.Fatal("untrusted protected input admitted")
			}
		})
	}
	for _, fault := range []string{"mode", "owner", "path", "duplicate", "byte", "trailing", "oversize"} {
		t.Run("tar-"+fault, func(t *testing.T) {
			var out bytes.Buffer
			w := tar.NewWriter(&out)
			h := &tar.Header{Name: "pipx-runtime.zip", Typeflag: tar.TypeReg, Mode: 0444, Size: int64(len(body))}
			data := body
			switch fault {
			case "mode":
				h.Mode = 0555
			case "owner":
				h.Uid = 1000
			case "path":
				h.Name = "../pipx-runtime.zip"
			case "byte":
				data = append([]byte(nil), body...)
				data[0] ^= 1
			}
			must(w.WriteHeader(h))
			_, err := w.Write(data)
			must(err)
			if fault == "duplicate" {
				must(w.WriteHeader(h))
				_, err = w.Write(data)
				must(err)
			}
			must(w.Close())
			bad := out.Bytes()
			if fault == "trailing" {
				bad = append(bad, 1)
			}
			if fault == "oversize" {
				bad = append(bad, make([]byte, 64<<10)...)
			}
			if _, err := scanNativeToolTar(bad, id, nativeRuntimePipx); err == nil {
				t.Fatal("invalid transport admitted")
			}
		})
	}
	after, err := readInstalledPipxTool(context.Background(), root)
	if err != nil || !bytes.Equal(after.body, body) {
		t.Fatal("fixture restore failed", err)
	}
	original, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(original, body) {
		t.Fatal("source input changed")
	}
	t.Logf("actual_pipx_data_bytes=%d sha256=%s protected_mode=0444 roundtrip=DATA_ONLY no_Docker_or_Python_execution", len(body), runtimeidentity.PipxToolSHA256)
}
