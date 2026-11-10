package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func uvSyntheticTar(t *testing.T, edit func(*tar.Header), duplicate bool) ([]byte, uvToolFileIdentity) {
	t.Helper()
	native := []byte("synthetic structural data; not native qualification")
	digest := sha256.Sum256(native)
	identity := uvToolFileIdentity{int64(len(native)), hex.EncodeToString(digest[:]), 0755}
	var out bytes.Buffer
	archive := tar.NewWriter(&out)
	count := 1
	if duplicate {
		count = 2
	}
	for i := 0; i < count; i++ {
		h := &tar.Header{Name: "uv", Typeflag: tar.TypeReg, Mode: 0755, Uid: 0, Gid: 0, Size: int64(len(native))}
		if edit != nil {
			edit(h)
		}
		if err := archive.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := archive.Write(native); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes(), identity
}

func TestUVSupplyTarRequiresOneExactRootFileAndCompleteStream(t *testing.T) {
	body, identity := uvSyntheticTar(t, nil, false)
	if native, err := scanUVToolTar(body, identity); err != nil || int64(len(native)) != identity.size {
		t.Fatal("structural tool rejected", err)
	}
	for _, name := range []string{"path", "link", "owner", "group", "mode", "duplicate", "hash", "trailing", "truncated", "oversize"} {
		t.Run(name, func(t *testing.T) {
			body, id := uvSyntheticTar(t, func(h *tar.Header) {
				switch name {
				case "path":
					h.Name = "../uv"
				case "link":
					h.Typeflag = tar.TypeSymlink
					h.Linkname = "/elsewhere"
					h.Size = 0
				case "owner":
					h.Uid = 1000
				case "group":
					h.Gid = 1000
				case "mode":
					h.Mode = 0777
				}
			}, name == "duplicate")
			switch name {
			case "hash":
				id.sha256 = strings.Repeat("0", 64)
			case "trailing":
				body = append(body, 1)
			case "truncated":
				body = body[:520]
			case "oversize":
				id.size = 64<<20 + 1
			}
			if _, err := scanUVToolTar(body, id); err == nil {
				t.Fatal("unbound supply accepted")
			}
		})
	}
}

func TestUVSupplyArchiveWriterCannotBypassWholeTransportBound(t *testing.T) {
	out := uvToolArchiveOutput{limit: 4}
	if _, err := io.WriteString(&out, "12345"); err == nil || !out.failed || len(out.body) != 0 {
		t.Fatal("StringWriter bypassed bound")
	}
	if _, err := out.Write([]byte("1")); err == nil {
		t.Fatal("failed writer became usable")
	}
}

type uvSupplyFake struct {
	calls       [][]string
	name, owner string
	fault       string
	cleanupErr  error
}

func (f *uvSupplyFake) Output(ctx context.Context, _ string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch args[0] {
	case "image":
		if f.fault == "image" {
			return []byte("sha256:wrong amd64"), nil
		}
		return []byte("sha256:" + runtimeidentity.UVConfigSHA256 + " " + runtimeidentity.UVArchitecture), nil
	case "create":
		for i, a := range args {
			if a == "--name" {
				f.name = args[i+1]
			}
			if a == "--label" {
				f.owner = strings.TrimPrefix(args[i+1], uvSupplyOwnerLabel+"=")
			}
		}
		if f.fault == "create" {
			return nil, errors.New("create reply unavailable")
		}
		return []byte(strings.Repeat("a", 64)), nil
	case "container":
		owner := f.owner
		state := "created"
		image := "sha256:" + runtimeidentity.UVConfigSHA256
		if f.fault == "owner" {
			owner = "foreign"
		}
		if f.fault == "state" {
			state = "running"
		}
		if f.fault == "config" {
			image = "sha256:foreign"
		}
		body, err := json.Marshal(map[string]any{"Id": strings.Repeat("a", 64), "Name": "/" + f.name, "Image": image, "Config": map[string]any{"Image": runtimeidentity.UVImageReference, "Labels": map[string]string{uvSupplyOwnerLabel: owner}}, "State": map[string]any{"Status": state, "Running": state == "running", "Pid": 0, "StartedAt": "0001-01-01T00:00:00Z"}})
		return body, err
	case "rm":
		return nil, f.cleanupErr
	default:
		return nil, errors.New("unexpected command")
	}
}
func (f *uvSupplyFake) RunOutput(_ context.Context, w io.Writer, _ string, args ...string) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	_, err := w.Write([]byte("unverified tool bytes"))
	return err
}
func (f *uvSupplyFake) RunInput(_ context.Context, r io.Reader, _ string, args ...string) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	_, err := io.Copy(io.Discard, r)
	return err
}

func TestUVSupplyNeverStartsAndOnlyRemovesVerifiedOwnedCreatedContainer(t *testing.T) {
	for _, fault := range []string{"image", "create", "owner", "state", "config", "bytes", "cleanup"} {
		t.Run(fault, func(t *testing.T) {
			f := &uvSupplyFake{fault: fault}
			sentinel := errors.New("cleanup denied")
			if fault == "cleanup" {
				f.cleanupErr = sentinel
			}
			prepared, err := prepareUVTool(context.Background(), f)
			if err == nil || prepared != nil {
				t.Fatal("unverified preparation succeeded")
			}
			removed := 0
			created := 0
			for _, args := range f.calls {
				switch args[0] {
				case "start", "run", "exec":
					t.Fatal("supply image executed", args)
				case "rm":
					removed++
					if args[1] != strings.Repeat("a", 64) {
						t.Fatal("foreign remove", args)
					}
				case "create":
					created++
					if !strings.Contains(strings.Join(args, " "), "--pull never") || !strings.Contains(strings.Join(args, " "), "--network none") {
						t.Fatal("mutable or networked supply", args)
					}
				}
			}
			wantRemove := fault == "create" || fault == "bytes" || fault == "cleanup"
			if (removed == 1) != wantRemove || fault == "image" && created != 0 {
				t.Fatal("ownership cleanup mismatch", fault, f.calls)
			}
			if fault == "cleanup" && !errors.Is(err, sentinel) {
				t.Fatal("cleanup failure lost", err)
			}
			if fault == "owner" || fault == "state" || fault == "config" {
				if !strings.Contains(err.Error(), "cleanup unproven") {
					t.Fatal("unknown became stopped", err)
				}
			}
		})
	}
}

func TestUVSupplyCancellationAndPrivateToolSubstitutionRefuseBeforeCopy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &uvSupplyFake{}
	if _, err := prepareUVTool(ctx, f); err == nil || len(f.calls) != 1 {
		t.Fatal("cancelled supply proceeded", err, f.calls)
	}
	if err := (&uvPreparedTool{body: []byte("substituted")}).introduce(context.Background(), f, strings.Repeat("b", 64)); err == nil {
		t.Fatal("unverified target tool copied")
	}
}
