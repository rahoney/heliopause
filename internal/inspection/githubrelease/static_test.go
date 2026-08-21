package githubrelease

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestStaticInspectorAcceptsBoundedZIP(t *testing.T) {
	root := t.TempDir()
	runID, _ := domain.NewRunID()
	assetPath := filepath.Join(root, runID.String(), "asset")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o700); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	entry, err := writer.Create("bin/tool")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("safe"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := testArtifact(t, runID, "tool.zip", uint64(body.Len()))
	inspector, _ := NewStaticInspector(root)
	report, err := inspector.Inspect(context.Background(), artifact)
	if err != nil || len(report.Findings()) != 0 || report.Evidence()[0].Kind() != "github-release-zip-static" {
		t.Fatalf("Inspect() = %#v, %v", report, err)
	}
}

func TestStaticInspectorRejectsUnsupportedAndMismatchedAssets(t *testing.T) {
	for _, test := range []struct{ name, filename, data, finding string }{
		{"unsupported", "tool.bin", "#!/bin/sh", "M6_FORMAT_UNSUPPORTED"},
		{"mismatch", "tool.zip", "not zip", "M6_FORMAT_UNSUPPORTED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			runID, _ := domain.NewRunID()
			assetPath := filepath.Join(root, runID.String(), "asset")
			if err := os.MkdirAll(filepath.Dir(assetPath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(assetPath, []byte(test.data), 0o600); err != nil {
				t.Fatal(err)
			}
			artifact := testArtifact(t, runID, test.filename, uint64(len(test.data)))
			inspector, _ := NewStaticInspector(root)
			report, err := inspector.Inspect(context.Background(), artifact)
			if err != nil || len(report.Findings()) != 1 || report.Findings()[0].Code() != test.finding {
				t.Fatalf("Inspect() = %#v, %v", report, err)
			}
		})
	}
}

func TestStaticInspectorRejectsZIPSymbolicLink(t *testing.T) {
	root := t.TempDir()
	runID, _ := domain.NewRunID()
	assetPath := filepath.Join(root, runID.String(), "asset")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o700); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	header := &zip.FileHeader{Name: "link"}
	header.SetMode(os.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("target"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	inspector, _ := NewStaticInspector(root)
	report, err := inspector.Inspect(context.Background(), testArtifact(t, runID, "tool.zip", uint64(body.Len())))
	if err != nil || len(report.Findings()) != 1 || report.Findings()[0].Code() != "M6_ARCHIVE_INVALID" {
		t.Fatalf("Inspect() = %#v, %v", report, err)
	}
}

func testArtifact(t *testing.T, runID domain.RunID, variant string, size uint64) domain.AcquiredArtifact {
	t.Helper()
	source, _ := domain.NewSourceID("github-release")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", "v1", variant)
	digest, _ := domain.NewSHA256Digest("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+runID.String()+":github-release", size)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
