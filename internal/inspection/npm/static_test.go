package npm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestInspector(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		entries []tarEntry
		finding string
	}{
		{"safe", []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}}, ""},
		{"link", []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}, {name: "package/link", typeflag: tar.TypeSymlink}}, "M2_ARCHIVE_TYPE_INVALID"},
		{"path escape", []tarEntry{{name: "../package.json", body: `{}`}}, "M2_ARCHIVE_PATH_INVALID"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := canonicalRoot(t)
			artifact := writeArtifact(t, root, test.entries)
			inspector, err := NewInspector(root)
			if err != nil {
				t.Fatal(err)
			}
			report, err := inspector.Inspect(context.Background(), artifact)
			if err != nil {
				t.Fatalf("Inspect() error = %v", err)
			}
			if report.Execution().Status() != domain.ExecutionCompleted {
				t.Fatalf("Execution() = %#v", report.Execution())
			}
			findings := report.Findings()
			if test.finding == "" && len(findings) != 0 {
				t.Fatalf("Findings() = %#v", findings)
			}
			if test.finding != "" && (len(findings) != 1 || findings[0].Code() != test.finding) {
				t.Fatalf("Findings() = %#v", findings)
			}
		})
	}
}

type tarEntry struct {
	name, body string
	typeflag   byte
}

func writeArtifact(t *testing.T, root string, entries []tarEntry) domain.AcquiredArtifact {
	t.Helper()
	runID, _ := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	directory := filepath.Join(root, runID.String())
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(directory, "tarball.tgz"))
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	writer := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		flag := entry.typeflag
		if flag == 0 {
			flag = tar.TypeReg
		}
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o600, Size: int64(len(entry.body)), Typeflag: flag}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+runID.String()+":tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func canonicalRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}
