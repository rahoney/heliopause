package gomodule

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestModuleStaticInspectionDoesNotExecuteSourceAndRejectsMalformedSubject(t *testing.T) {
	for _, tc := range []struct {
		name, mod, identity, file, body string
		duplicate, link, corrupt        bool
		requiredMod                     string
		want                            string
		contentFinding                  string
	}{
		{name: "normal", mod: "example.com/module", file: "module.go", body: "package module\nfunc init() { panic(\"must never execute\") }\n"},
		{name: "python helper is content DATA", mod: "example.com/module", file: "tools/helper.py", body: "raise RuntimeError('must never execute')\n"},
		{name: "extensionless helper", mod: "example.com/module", file: "tools/helper", body: "#!/usr/bin/env python3\nraise RuntimeError('must never execute')\n"},
		{name: "extensionless helper CRC", mod: "example.com/module", file: "tools/helper", body: "#!/usr/bin/env python3\nraise RuntimeError('must never execute')\n", corrupt: true, contentFinding: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "other source CRC", mod: "example.com/module", file: "tools/helper.py", body: "raise RuntimeError('must never execute')\n", corrupt: true, contentFinding: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "required old control", mod: "example.com/module", file: "module.go", body: "package module\n", requiredMod: "module example.com/module\ngo 1.25\n"},
		{name: "foreign required control", mod: "example.com/module", file: "module.go", body: "package module\n", requiredMod: "module example.com/foreign\ngo 1.25\n", want: "M12_GO_MODULE_IDENTITY_MISMATCH"},
		{name: "malformed required control", mod: "example.com/module", file: "module.go", body: "package module\n", requiredMod: "module (\n", want: "M12_GO_MODULE_IDENTITY_MISMATCH"},
		{name: "module identity", mod: "example.com/other", file: "module.go", body: "package module", want: "M12_GO_MODULE_IDENTITY_MISMATCH"},
		{name: "syntax", mod: "example.com/module", file: "module.go", body: "package !!!", want: "M12_GO_SOURCE_INVALID"},
		{name: "negative test asset", mod: "example.com/module", file: "testdata/bad.go", body: "package !!!"},
		{name: "nested negative test asset", mod: "example.com/module", file: "parser/testdata/src/bad/bad.go", body: "// deliberately no package declaration\n"},
		{name: "testdata file is source", mod: "example.com/module", file: "testdata.go", body: "package !!!", want: "M12_GO_SOURCE_INVALID"},
		{name: "similar directory is source", mod: "example.com/module", file: "testdatabase/bad.go", body: "package !!!", want: "M12_GO_SOURCE_INVALID"},
		{name: "case sensitive directory is source", mod: "example.com/module", file: "Testdata/bad.go", body: "package !!!", want: "M12_GO_SOURCE_INVALID"},
		{name: "module name is not data", mod: "example.com/testdata", identity: "example.com/testdata", file: "bad.go", body: "package !!!", want: "M12_GO_SOURCE_INVALID"},
		{name: "test asset CRC", mod: "example.com/module", file: "testdata/bad.go", body: "package !!!", corrupt: true, want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "test asset traversal", mod: "example.com/module", file: "testdata/../bad.go", body: "package !!!", want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "test asset duplicate", mod: "example.com/module", file: "testdata/bad.go", body: "package !!!", duplicate: true, want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "test asset symlink", mod: "example.com/module", file: "testdata/bad.go", body: "outside", link: true, want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "traversal", mod: "example.com/module", file: "../escape.go", body: "package module", want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "duplicate", mod: "example.com/module", file: "module.go", body: "package module", duplicate: true, want: "M12_GO_MODULE_ARCHIVE_INVALID"},
		{name: "symlink", mod: "example.com/module", file: "module.go", body: "outside", link: true, want: "M12_GO_MODULE_ARCHIVE_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var z bytes.Buffer
			w := zip.NewWriter(&z)
			modulePath := tc.identity
			if modulePath == "" {
				modulePath = "example.com/module"
			}
			name := modulePath + "@v1.0.0/" + tc.file
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
			if tc.corrupt {
				pos := bytes.Index(z.Bytes(), []byte(tc.body))
				if pos < 0 {
					t.Fatal("fixture payload missing")
				}
				z.Bytes()[pos] ^= 1
			}
			mod := []byte("module " + tc.mod + "\ngo 1.26.0\n")
			control, header := mod, "HAA-GO-MODULE-1\n"
			if tc.requiredMod != "" {
				var err error
				control, err = json.Marshal(struct {
					Mod      []byte                          `json:"mod"`
					Required []artifactgo.RequiredModControl `json:"required_mods"`
				}{mod, []artifactgo.RequiredModControl{{Version: "v0.9.0", Body: []byte(tc.requiredMod)}}})
				if err != nil {
					t.Fatal(err)
				}
				header = "HAA-GO-MODULE-2\n"
			}
			body := append([]byte(header), make([]byte, 8)...)
			binary.BigEndian.PutUint64(body[len(body)-8:], uint64(len(control)))
			body = append(body, control...)
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
			identity, _ := domain.NewResolvedArtifactIdentity(artifactgo.Source(), modulePath, "v1.0.0", "module")
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
			sources, measured, sourceErr := inspector.InspectSources(context.Background(), acquired)
			if sourceErr != nil {
				t.Fatal(sourceErr)
			}
			if tc.contentFinding != "" {
				if len(report.Findings()) != 0 || len(sources) != 0 || len(measured.Findings()) != 1 || measured.Findings()[0].Code() != tc.contentFinding {
					t.Fatal("newly selected helper bytes did not receive independent ZIP integrity validation")
				}
				return
			}
			if !reflect.DeepEqual(report, measured) {
				t.Fatal("source selection changed the existing static report")
			}
			if tc.want == "" {
				if len(sources) != 1 || sources[0].Path != tc.file || !bytes.Equal(sources[0].Body, []byte(tc.body)) {
					t.Fatal("validated source/testdata bytes were dropped or changed", sources)
				}
			} else if len(sources) != 0 {
				t.Fatal("invalid module leaked partial sources")
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
