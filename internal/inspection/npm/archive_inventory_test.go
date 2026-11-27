package npm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

	"github.com/rahoney/heliopause/internal/core/domain"
)

func measuredInventorySubject(t *testing.T, root string, original domain.AcquiredArtifact) domain.AcquiredArtifact {
	t.Helper()
	file := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
	body, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(body)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(h[:]))
	artifact, err := domain.NewAcquiredArtifact(original.Identity(), digest, original.ContentHandle(), uint64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func TestInspectArchivePreservesStaticReportAndMeasuresFiles(t *testing.T) {
	root := canonicalRoot(t)
	manifest := `{"name":"tiny","version":"1.2.3","scripts":{"install":"exit 0"}}`
	artifact := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: manifest}, {name: "package/src", typeflag: tar.TypeDir}, {name: "package/src/main.js", body: "\x00binary\xff\n"}, {name: "package/empty"}})
	artifact = measuredInventorySubject(t, root, artifact)
	inspector, _ := NewInspector(root)
	legacy, err := inspector.Inspect(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	inventory, report, err := inspector.InspectArchive(context.Background(), artifact)
	if err != nil || !inventory.Matches(artifact) || !reflect.DeepEqual(legacy, report) {
		t.Fatal("inventory changed the original report or subject", err)
	}
	entries := inventory.Entries()
	if len(entries) != 4 || !entries[2].Directory || entries[2].Path != "src" || !bytes.Equal(inventory.Manifest(), []byte(manifest)) {
		t.Fatal("inventory lost files/directory/manifest", entries)
	}
	h := sha256.Sum256([]byte("\x00binary\xff\n"))
	if entries[3].Path != "src/main.js" || entries[3].Size != 9 || entries[3].Mode != 0o600 || entries[3].Digest.String() != hex.EncodeToString(h[:]) {
		t.Fatal("non-manifest payload was not measured exactly", entries[3])
	}
	files, _ := os.ReadDir(filepath.Dir(filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")))
	if len(files) != 1 || files[0].Name() != "tarball.tgz" {
		t.Fatal("inspection extracted files")
	}
}

func TestInspectArchiveNoInventoryForStaticViolations(t *testing.T) {
	for _, entries := range [][]tarEntry{
		{{name: "package/package.json", body: `{"name":"foreign","version":"1.2.3"}`}},
		{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}, {name: "package/link", typeflag: tar.TypeSymlink}},
		{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}, {name: "package/link", typeflag: tar.TypeLink}},
		{{name: "../escape", body: "hidden"}},
		{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}, {name: "package/package.json", body: `{}`}},
	} {
		root := canonicalRoot(t)
		artifact := writeArtifact(t, root, entries)
		artifact = measuredInventorySubject(t, root, artifact)
		inspector, _ := NewInspector(root)
		legacy, _ := inspector.Inspect(context.Background(), artifact)
		inventory, report, err := inspector.InspectArchive(context.Background(), artifact)
		if err != nil || inventory.Valid() || len(report.Findings()) != 1 || !reflect.DeepEqual(legacy, report) {
			t.Fatal("static violation lost its original report or produced inventory", err)
		}
	}
}

func TestInspectArchiveRejectsSourceAndContainerAmbiguity(t *testing.T) {
	for _, test := range []string{"digest", "size", "truncated", "crc", "second-member", "trailing-byte", "nonzero-tar-tail", "ancestor-file", "backslash", "decoded-limit", "symlink-source"} {
		t.Run(test, func(t *testing.T) {
			root := canonicalRoot(t)
			entries := []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}}
			if test == "ancestor-file" {
				entries = append(entries, tarEntry{name: "package/a", body: "x"}, tarEntry{name: "package/a/b", body: "y"})
			}
			if test == "backslash" {
				entries = append(entries, tarEntry{name: "package/a\\b", body: "x"})
			}
			artifact := writeArtifact(t, root, entries)
			file := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
			body, _ := os.ReadFile(file)
			switch test {
			case "truncated":
				body = body[:len(body)-5]
			case "crc":
				body[len(body)-8] ^= 1
			case "second-member":
				body = append(body, body...)
			case "trailing-byte":
				body = append(body, 1)
			case "nonzero-tar-tail", "decoded-limit":
				reader, _ := gzip.NewReader(bytes.NewReader(body))
				decoded, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				_ = reader.Close()
				var output bytes.Buffer
				writer := gzip.NewWriter(&output)
				_, _ = writer.Write(decoded)
				if test == "nonzero-tar-tail" {
					_, _ = writer.Write([]byte("hidden"))
				} else if _, err := io.CopyN(writer, inventoryZeroReader{}, decodedLimit+1); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				body = output.Bytes()
			}
			if err := os.WriteFile(file, body, 0o600); err != nil {
				t.Fatal(err)
			}
			artifact = measuredInventorySubject(t, root, artifact)
			if test == "digest" {
				digest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
				artifact, _ = domain.NewAcquiredArtifact(artifact.Identity(), digest, artifact.ContentHandle(), artifact.SizeBytes())
			}
			if test == "size" {
				artifact, _ = domain.NewAcquiredArtifact(artifact.Identity(), artifact.Digest(), artifact.ContentHandle(), artifact.SizeBytes()+1)
			}
			if test == "symlink-source" {
				if err := os.Rename(file, file+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(file+".original", file); err != nil {
					t.Fatal(err)
				}
			}
			inspector, _ := NewInspector(root)
			inventory, report, err := inspector.InspectArchive(context.Background(), artifact)
			if inventory.Valid() || (err == nil && len(report.Findings()) == 0) {
				t.Fatal("ambiguous/unbound archive produced a normal inventory")
			}
		})
	}
}

type inventoryZeroReader struct{}

func (inventoryZeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestInventoryCancellationPreservesCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("inventory caller stopped")
	reader := &inventoryCancellingReader{cancel: func() { cancel(cause) }}
	err := copyArchivePayload(ctx, io.Discard, reader, 128<<10)
	if !errors.Is(err, context.Canceled) || !errors.Is(context.Cause(ctx), cause) || reader.reads != 1 {
		t.Fatal("inventory read continued after cancellation", err, reader.reads)
	}
	root := canonicalRoot(t)
	artifact := writeArtifact(t, root, []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}})
	inspector, _ := NewInspector(root)
	inventory, _, err := inspector.InspectArchive(ctx, artifact)
	if !errors.Is(err, context.Canceled) || inventory.Valid() {
		t.Fatal("cancelled public inspection produced inventory", err)
	}
}

type inventoryCancellingReader struct {
	cancel func()
	reads  int
}

func (r *inventoryCancellingReader) Read(p []byte) (int, error) {
	r.reads++
	r.cancel()
	clear(p)
	return len(p), nil
}
