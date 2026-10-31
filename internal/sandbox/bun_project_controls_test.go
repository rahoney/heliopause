package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
	"io"
	"strings"
	"testing"
)

func bunControlDirectory(t *testing.T, lock bool, edit func(*tar.Header), duplicate bool) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := tar.NewWriter(&body)
	root := tar.Header{Name: "project/", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}
	if err := writer.WriteHeader(&root); err != nil {
		t.Fatal(err)
	}
	names := []string{"package.json"}
	if lock {
		names = append(names, "bun.lock")
	}
	if duplicate {
		names = append(names, "package.json")
	}
	for _, name := range names {
		value := []byte(`{"fixture":"unapproved data"}`)
		h := tar.Header{Name: "project/" + name, Typeflag: tar.TypeReg, Mode: 0644, Uid: 1000, Gid: 1000, Size: int64(len(value))}
		if edit != nil {
			edit(&h)
		}
		if h.Typeflag != tar.TypeReg {
			h.Size = 0
		}
		if err := writer.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := writer.Write(value); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestBunControlCensusRequiresCompleteOwnedProjectAndExplicitAbsence(t *testing.T) {
	for _, lock := range []bool{false, true} {
		controls, err := parseBunProjectControls(context.Background(), bunControlDirectory(t, lock, nil, false))
		if err != nil || len(controls) != 2 || !controls[0].Present() || controls[0].Name() != "package.json" || controls[1].Name() != "bun.lock" || controls[1].Present() != lock {
			t.Fatal("complete data census differs", err)
		}
	}
	for _, fault := range []string{"path", "foreign-lock", "owner", "group", "mode", "symlink", "hardlink", "duplicate", "trailing", "truncated"} {
		t.Run(fault, func(t *testing.T) {
			body := bunControlDirectory(t, true, func(h *tar.Header) {
				switch fault {
				case "path":
					h.Name = "project/../package.json"
				case "foreign-lock":
					h.Name = "project/pnpm-lock.yaml"
				case "owner":
					h.Uid = 0
				case "group":
					h.Gid = 0
				case "mode":
					h.Mode = 0755
				case "symlink":
					h.Typeflag = tar.TypeSymlink
					h.Linkname = "elsewhere"
				case "hardlink":
					h.Typeflag = tar.TypeLink
					h.Linkname = "project/other"
				}
			}, fault == "duplicate")
			if fault == "trailing" {
				body = append(body, 1)
			}
			if fault == "truncated" {
				body = body[:600]
			}
			if controls, err := parseBunProjectControls(context.Background(), body); err == nil || controls != nil {
				t.Fatal("uncertain census became selected controls")
			}
		})
	}
	runner := &pnpmControlTransport{err: errors.New("No such file"), body: bunControlDirectory(t, false, nil, false)}
	if controls, err := readBunProjectControls(context.Background(), runner, strings.Repeat("a", 64)); err == nil || controls != nil {
		t.Fatal("transport error became absent lock")
	}
}

func TestBunControlInputKeepsPrivateCacheAndPNPMLockSeparate(t *testing.T) {
	runner := &pnpmControlTransport{}
	id := strings.Repeat("a", 64)
	if err := copyBunProjectControls(context.Background(), runner, id, nil); err == nil {
		t.Fatal("empty controls accepted")
	}
	if err := copyBunProjectControls(context.Background(), runner, id, []domain.ProjectControlFile{pnpmTestControl(t, "package.json", `{}`, true), pnpmTestControl(t, "bun.lock", "", false)}); err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(bytes.NewReader(runner.input))
	names := []string{}
	for {
		h, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, h.Name)
		if h.Uid != 1000 || h.Gid != 1000 {
			t.Fatal("Bun data ownership changed")
		}
	}
	if strings.Join(names, ",") != "project/,project/package.json,cache/,tmp/" || runner.args[len(runner.args)-1] != id+":"+bunProjectRoot {
		t.Fatal("Bun data boundary differs", names, runner.args)
	}
}
