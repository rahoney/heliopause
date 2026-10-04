package gomodule

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestModuleStaticInspectionDoesNotExecuteSourceAndRejectsMalformedSubject(t *testing.T) {
	for _, tc := range []struct {
		name, mod, file, body string
		duplicate, link       bool
		want                  string
	}{
		{name: "normal", mod: "example.com/module", file: "module.go", body: "package module\nfunc init() { panic(\"must never execute\") }\n"},
		{name: "module identity", mod: "example.com/other", file: "module.go", body: "package module", want: "M12_GO_MODULE_IDENTITY_MISMATCH"},
		{name: "syntax", mod: "example.com/module", file: "module.go", body: "package !!!", want: "M12_GO_SOURCE_INVALID"},
		{name: "traversal", mod: "example.com/module", file: "../escape.go", body: "package module", want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "duplicate", mod: "example.com/module", file: "module.go", body: "package module", duplicate: true, want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "symlink", mod: "example.com/module", file: "module.go", body: "outside", link: true, want: "M12_GO_MODULE_ARCHIVE_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var z bytes.Buffer
			w := zip.NewWriter(&z)
			name := "example.com/module@v1.0.0/" + tc.file
			h := &zip.FileHeader{Name: name, Method: zip.Store}
			if tc.link {
				h.SetMode(os.ModeSymlink | 0o777)
			}
			f, err := w.CreateHeader(h)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(f, tc.body); err != nil {
				t.Fatal(err)
			}
			if tc.duplicate {
				f, err := w.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.WriteString(f, tc.body); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			mod := []byte("module " + tc.mod + "\ngo 1.26.0\n")
			body := append([]byte("HAA-GO-MODULE-1\n"), make([]byte, 8)...)
			binary.BigEndian.PutUint64(body[len(body)-8:], uint64(len(mod)))
			body = append(body, mod...)
			body = append(body, z.Bytes()...)
			root := t.TempDir()
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(root, run.String()), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, run.String(), "module.bundle"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
			identity, _ := domain.NewResolvedArtifactIdentity(artifactgo.Source(), "example.com/module", "v1.0.0", "module")
			acquired, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":module", uint64(len(body)))
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := NewStaticInspector(root)
			if err != nil {
				t.Fatal(err)
			}
			report, err := inspector.Inspect(context.Background(), acquired)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if len(report.Findings()) != 0 {
					t.Fatal(report.Findings())
				}
			} else if len(report.Findings()) != 1 || report.Findings()[0].Code() != tc.want {
				t.Fatalf("findings=%v", report.Findings())
			}
		})
	}
}
