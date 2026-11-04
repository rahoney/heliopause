package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestArchiveSourcesShareVerifiedInventoryAndPayload(t *testing.T) {
	root := canonicalRoot(t)
	entries := []tarEntry{
		{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`},
		{name: "package/src", typeflag: tar.TypeDir},
		{name: "package/src/component.tsx", body: "export const x = <div/>;\n"},
		{name: "package/bad.py", body: "\xff\x00unparsed source"},
		{name: "package/README.md", body: "not a parser target"},
		{name: "package/vendor.min.js", body: ""},
		{name: "package/bin/cli", body: "#!/usr/bin/env node\nconsole.log('DATA');\n"},
		{name: "package/helpers/python-tool", body: "#!/usr/bin/python3.15\nraise RuntimeError('DATA');\n"},
	}
	subject := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
	inspector, _ := NewInspector(root)
	legacy, legacyReport, err := inspector.InspectArchive(context.Background(), subject)
	if err != nil {
		t.Fatal(err)
	}
	inventory, sources, report, err := inspector.InspectArchiveSources(context.Background(), subject)
	if err != nil || !reflect.DeepEqual(legacy, inventory) || !reflect.DeepEqual(legacyReport, report) {
		t.Fatal("source capture changed the measured inventory or static report", err)
	}
	if len(sources) != 5 || sources[0].Path != "src/component.tsx" || sources[0].Language != "typescript" || sources[1].Language != "python" || !bytes.Equal(sources[1].Body, []byte(entries[3].body)) || sources[2].Path != "vendor.min.js" || len(sources[2].Body) != 0 || sources[3].Path != "bin/cli" || sources[3].Language != "javascript" || sources[4].Path != "helpers/python-tool" || sources[4].Language != "python" {
		t.Fatal("source selection lost path, language, empty or invalid-encoding input", sources)
	}
	for _, source := range sources {
		sum := sha256.Sum256(source.Body)
		found := false
		for _, member := range inventory.Entries() {
			if member.Path == source.Path {
				found = int64(len(source.Body)) == member.Size && hex.EncodeToString(sum[:]) == member.Digest.String()
			}
		}
		if !found {
			t.Fatal("retained source differs from measured member", source.Path)
		}
	}
	files, err := os.ReadDir(filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa"))
	if err != nil || len(files) != 1 || files[0].Name() != "tarball.tgz" {
		t.Fatal("source collection extracted files", err)
	}
}

func TestArchiveSourcesDiscardCollectedBytesOnLateFailure(t *testing.T) {
	for _, failure := range []string{"crc", "digest", "late-path", "cancelled"} {
		t.Run(failure, func(t *testing.T) {
			root := canonicalRoot(t)
			entries := []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}, {name: "package/source.js", body: "process.env.TOKEN"}}
			if failure == "late-path" {
				entries = append(entries, tarEntry{name: "../escape", body: "bad"})
			}
			subject := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
			if failure == "crc" {
				file := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
				body, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				body[len(body)-8] ^= 1
				if err := os.WriteFile(file, body, 0o600); err != nil {
					t.Fatal(err)
				}
				// Bind the modified compressed bytes: this must still fail the
				// decompressor's independent integrity check after source read.
				subject = measuredInventorySubject(t, root, subject)
			} else if failure == "digest" {
				digest, _ := domain.NewSHA256Digest(strings.Repeat("b", 64))
				subject, _ = domain.NewAcquiredArtifact(subject.Identity(), digest, subject.ContentHandle(), subject.SizeBytes())
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if failure == "cancelled" {
				cancel()
			}
			inspector, _ := NewInspector(root)
			inventory, sources, report, err := inspector.InspectArchiveSources(ctx, subject)
			if inventory.Valid() || len(sources) != 0 || (err == nil && len(report.Findings()) == 0) {
				t.Fatal("unverified or cancelled archive leaked source bytes", err)
			}
		})
	}
}
