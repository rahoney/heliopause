package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	contentInventory, contentSources, contentReport, contentErr := inspector.InspectArchiveContent(context.Background(), subject)
	if contentErr != nil || !reflect.DeepEqual(contentInventory, inventory) || !reflect.DeepEqual(contentSources, sources) || len(contentReport.Executions()) != 4 || contentReport.Executions()[1].Status() != domain.ExecutionCompleted {
		t.Fatal("content selection changed measured source facts or lost bound binary applicability", contentErr)
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
			entries := []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}, {name: "package/source.js", body: "process.env.TOKEN"}, {name: "package/native", body: "\x7fELF"}}
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
			contentInventory, contentSources, contentReport, contentErr := inspector.InspectArchiveContent(ctx, subject)
			if contentInventory.Valid() || len(contentSources) != 0 || len(contentReport.Executions()) > 1 || contentErr == nil && len(contentReport.Findings()) == 0 {
				t.Fatal("late archive failure leaked provisional native facts", contentErr)
			}
		})
	}
}

func TestArchiveContentPreservesNativeMemberInventoryWithoutSourceDuplication(t *testing.T) {
	root := canonicalRoot(t)
	entries := []tarEntry{
		{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`},
		{name: "package/disguised.js", body: "MZ"},
		{name: "package/native", body: "\x7fELF"},
		{name: "package/source.js", body: "console.log('DATA');\n"},
	}
	subject := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
	inspector, _ := NewInspector(root)
	inventory, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
	if err == nil || !inventory.Valid() || len(sources) != 1 || sources[0].Path != "source.js" || len(report.Executions()) != 4 || report.Executions()[1].Status() != domain.ExecutionIncomplete {
		t.Fatal("native selection lost verified inventory or overstated coverage", err)
	}
	coverageFacts := 0
	for _, fact := range report.Evidence() {
		if fact.Kind() == "binary-content-coverage" {
			coverageFacts++
			if !strings.Contains(fact.Summary(), "member_sha256") {
				t.Fatal("native fact lost member digest")
			}
		}
	}
	if coverageFacts != 2 {
		t.Fatal("native census does not match measured entries", coverageFacts)
	}
}

func TestArchiveContentCensusPreservesUnsupportedSourceAndNativeRoles(t *testing.T) {
	root := canonicalRoot(t)
	entries := []tarEntry{
		{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`},
		{name: "package/directory", typeflag: tar.TypeDir},
		{name: "package/tool", body: "#!/bin/sh\necho DATA\n"},
		{name: "package/UPPER.JS", body: "console.log('DATA');\n"},
		{name: "package/disguised.js", body: "MZ"},
		{name: "package/source.js", body: "console.log('DATA');\n"},
		{name: "package/README.md", body: "DATA"},
	}
	subject := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
	inspector, _ := NewInspector(root)
	archive, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
	if err == nil || !archive.Matches(subject) || len(sources) != 1 || sources[0].Path != "source.js" {
		t.Fatal("unsupported native member was completed or source roles changed", err)
	}
	count := 0
	for _, fact := range report.Evidence() {
		if fact.Kind() != "content-inventory" {
			continue
		}
		count++
		var census struct {
			Members     int    `json:"members"`
			Sources     int    `json:"sources"`
			Binaries    int    `json:"binaries"`
			Unsupported int    `json:"unsupported_sources"`
			Digest      string `json:"inventory_sha256"`
		}
		if json.Unmarshal([]byte(fact.Summary()), &census) != nil || census.Members != 6 || census.Sources != 1 || census.Binaries != 1 || census.Unsupported != 2 || len(census.Digest) != 64 || fact.Identity() != subject.Identity() || fact.Digest() != subject.Digest() {
			t.Fatal("complete regular-member census lost source/native/unsupported/subject binding", fact)
		}
	}
	if count != 1 {
		t.Fatal("complete archive did not publish exactly one census", count)
	}
}
