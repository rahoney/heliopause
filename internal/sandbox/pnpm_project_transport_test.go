package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type pnpmControlTransport struct {
	body, input []byte
	args        []string
	calls       int
	err         error
}

func (*pnpmControlTransport) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected command")
}
func (r *pnpmControlTransport) RunInput(_ context.Context, source io.Reader, name string, args ...string) error {
	r.calls++
	r.args = append([]string{name}, args...)
	var err error
	r.input, err = io.ReadAll(source)
	return errors.Join(r.err, err)
}
func (r *pnpmControlTransport) RunOutput(_ context.Context, target io.Writer, name string, args ...string) error {
	r.calls++
	r.args = append([]string{name}, args...)
	_, err := io.Copy(target, bytes.NewReader(r.body))
	return errors.Join(r.err, err)
}

func pnpmTestControl(t *testing.T, name, body string, present bool) domain.ProjectControlFile {
	t.Helper()
	value, err := domain.NewProjectControlFile(name, []byte(body), present)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func pnpmControlArchive(t *testing.T, header tar.Header, body []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	if header.Typeflag == tar.TypeReg {
		header.Size = int64(len(body))
	}
	if err := writer.WriteHeader(&header); err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 {
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestPNPMProjectControlTransportUsesOnlyFixedGuestData(t *testing.T) {
	id := strings.Repeat("a", 64)
	input := []domain.ProjectControlFile{pnpmTestControl(t, "pnpm-lock.yaml", "", false), pnpmTestControl(t, "package.json", `{}`, true)}
	runner := &pnpmControlTransport{}
	if err := copyPNPMProjectControls(context.Background(), runner, id, input); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runner.args, []string{"docker", "cp", "--archive", "-", id + ":" + pnpmProjectRoot}) {
		t.Fatal("input is not fixed data transport")
	}
	reader := tar.NewReader(bytes.NewReader(runner.input))
	for _, name := range []string{"project/", "project/package.json"} {
		h, err := reader.Next()
		if err != nil || h.Name != name || h.Uid != 1000 || h.Gid != 1000 || h.Typeflag != tar.TypeDir && h.Typeflag != tar.TypeReg {
			t.Fatal("input archive boundary changed", err)
		}
	}
	if body, err := io.ReadAll(reader); err != nil || string(body) != `{}` {
		t.Fatal("manifest bytes changed", err)
	}
	if _, err := reader.Next(); err != io.EOF {
		t.Fatal("absent lock became an input")
	}
	runner = &pnpmControlTransport{body: pnpmControlArchive(t, tar.Header{Name: "pnpm-lock.yaml", Typeflag: tar.TypeReg, Mode: 0644, Uid: 1000, Gid: 1000}, []byte("lock"))}
	control, err := readPNPMProjectControl(context.Background(), runner, id, "pnpm-lock.yaml")
	if err != nil || !control.Present() || string(control.Body()) != "lock" || control.Name() != "pnpm-lock.yaml" {
		t.Fatal("control capture differs", err)
	}
	if !reflect.DeepEqual(runner.args, []string{"docker", "cp", id + ":" + pnpmProjectDirectory + "/pnpm-lock.yaml", "-"}) {
		t.Fatal("output reads an unexpected path")
	}
}

func TestPNPMProjectControlTransportRejectsUnknownInputAndKeepsCancellation(t *testing.T) {
	id := strings.Repeat("a", 64)
	manifest := pnpmTestControl(t, "package.json", `{}`, true)
	for _, controls := range [][]domain.ProjectControlFile{
		nil, {manifest, manifest}, {pnpmTestControl(t, "unknown", `{}`, true)},
		{pnpmTestControl(t, "package.json", "", false)}, {pnpmTestControl(t, "package.json", "", true)},
	} {
		runner := &pnpmControlTransport{}
		if err := copyPNPMProjectControls(context.Background(), runner, id, controls); err == nil || runner.calls != 0 {
			t.Fatal("invalid controls reached Docker")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &pnpmControlTransport{}
	if err := copyPNPMProjectControls(ctx, runner, id, []domain.ProjectControlFile{manifest}); !errors.Is(err, context.Canceled) || runner.calls != 0 {
		t.Fatal("cancelled input lost cause or made a request", err)
	}
	if _, err := readPNPMProjectControl(ctx, runner, id, "package.json"); !errors.Is(err, context.Canceled) || runner.calls != 0 {
		t.Fatal("cancelled read lost cause or made a request", err)
	}
	if _, err := readPNPMProjectControl(context.Background(), runner, id, "../package.json"); err == nil || runner.calls != 0 {
		t.Fatal("unknown path reached Docker")
	}
}

func TestPNPMProjectControlCaptureRejectsUntrustedTarMetadataAndOverflow(t *testing.T) {
	id := strings.Repeat("a", 64)
	for _, name := range []string{"uid", "gid", "mode", "name", "link", "extra", "padding", "header-size", "transport-size"} {
		t.Run(name, func(t *testing.T) {
			h := tar.Header{Name: "package.json", Typeflag: tar.TypeReg, Mode: 0600, Uid: 1000, Gid: 1000}
			body := []byte(`{}`)
			switch name {
			case "uid":
				h.Uid = 0
			case "gid":
				h.Gid = 0
			case "mode":
				h.Mode = 0777
			case "name":
				h.Name = "../package.json"
			case "link":
				h.Typeflag = tar.TypeSymlink
				h.Linkname = "/etc/passwd"
				body = nil
			case "header-size":
				body = bytes.Repeat([]byte("x"), pnpmControlLimit+1)
			}
			transport := pnpmControlArchive(t, h, body)
			switch name {
			case "extra":
				transport = append(transport[:len(transport)-1024], pnpmControlArchive(t, h, []byte(`{}`))...)
			case "padding":
				transport = append(transport, byte('x'))
			case "transport-size":
				transport = append(transport, make([]byte, pnpmControlLimit+64<<10)...)
			}
			runner := &pnpmControlTransport{body: transport}
			control, err := readPNPMProjectControl(context.Background(), runner, id, "package.json")
			if err == nil || control.Present() || len(control.Body()) != 0 {
				t.Fatal("untrusted/oversized capture produced a control")
			}
		})
	}
}
