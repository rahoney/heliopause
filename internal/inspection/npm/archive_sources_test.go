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
	"strconv"
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

func TestArchiveContentSelectsDeclaredExtensionlessMainAndBins(t *testing.T) {
	for _, manifestLast := range []bool{false, true} {
		root := canonicalRoot(t)
		manifest := tarEntry{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","main":"./lib/entry","bin":{"tiny":"./bin/tool","alias":"bin/tool"}}`}
		entries := []tarEntry{
			{name: "package/lib/entry", body: "module.exports = 1;\n"},
			{name: "package/bin/tool", body: "console.log('DATA');\n"},
			{name: "package/other.js", body: "module.exports = 2;\n"},
			{name: "package/README", body: "unselected DATA"},
		}
		if manifestLast {
			entries = append(entries, manifest)
		} else {
			entries = append([]tarEntry{manifest}, entries...)
		}
		subject := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
		inspector, _ := NewInspector(root)
		archive, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
		if err != nil || !archive.Matches(subject) || len(sources) != 3 || len(report.Executions()) != 4 {
			t.Fatal("declared entry source coverage incomplete", manifestLast, len(sources), err)
		}
		seen := map[string]bool{}
		for _, source := range sources {
			if seen[source.Path] || source.Language != "javascript" || len(source.Body) == 0 {
				t.Fatal("source role was duplicated or lost", source.Path)
			}
			seen[source.Path] = true
		}
		if !seen["lib/entry"] || !seen["bin/tool"] || !seen["other.js"] || seen["README"] {
			t.Fatal("metadata targets were omitted or DATA was guessed as source", seen)
		}
	}
}

func TestArchiveContentDeclaredJSONAndUnknownInterpreterKeepTheirRoles(t *testing.T) {
	root := canonicalRoot(t)
	subject := measuredInventorySubject(t, root, writeArtifact(t, root, []tarEntry{
		{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","main":"./data.json","bin":"bin/tool"}`},
		{name: "package/data.json", body: `{"value":1}`},
		{name: "package/bin/tool", body: "#!/bin/sh\necho DATA\n"},
	}))
	inspector, _ := NewInspector(root)
	_, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
	if err != nil || len(sources) != 0 {
		t.Fatal("JSON DATA or unsupported shell was relabeled as JavaScript", err)
	}
	for _, fact := range report.Evidence() {
		if fact.Kind() == "content-inventory" && !strings.Contains(fact.Summary(), `"unsupported_sources":1`) {
			t.Fatal("unsupported declared interpreter disappeared", fact)
		}
	}
}

func TestArchiveContentRejectsUnsafeOrMissingDeclaredTargetsWithoutFacts(t *testing.T) {
	for _, extra := range []string{
		`"main":"../escape"`,
		`"main":123`,
		`"bin":"missing"`,
		`"bin":"../escape"`,
		`"bin":{"bad/name":"entry"}`,
		`"bin":["entry"]`,
		`"bin":"entry","bin":"other"`,
	} {
		root := canonicalRoot(t)
		subject := measuredInventorySubject(t, root, writeArtifact(t, root, []tarEntry{
			{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3",` + extra + `}`},
			{name: "package/entry", body: "console.log('DATA');\n"},
		}))
		inspector, _ := NewInspector(root)
		archive, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
		if err == nil || archive.Valid() || len(sources) != 0 || len(report.Executions()) > 1 {
			t.Fatal("invalid metadata produced trusted content facts", extra, err)
		}
	}
}

func TestArchiveContentDeclaredNativeKeepsByteFormatPriority(t *testing.T) {
	root := canonicalRoot(t)
	subject := measuredInventorySubject(t, root, writeArtifact(t, root, []tarEntry{
		{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","main":"native","bin":"native"}`},
		{name: "package/native", body: "MZ"},
	}))
	inspector, _ := NewInspector(root)
	archive, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
	if err == nil || !archive.Matches(subject) || len(sources) != 0 || len(report.Executions()) != 4 || report.Executions()[1].Status() != domain.ExecutionIncomplete {
		t.Fatal("metadata disguised an unsupported native format as source", err)
	}
}

func TestArchiveContentSelectsConditionalAndNestedMetadataSources(t *testing.T) {
	root := canonicalRoot(t)
	entries := []tarEntry{
		{name: "package/export-node", body: "module.exports=1;\n"},
		{name: "package/export-default", body: "module.exports=2;\n"},
		{name: "package/src/a/b/entry", body: "module.exports=3;\n"},
		{name: "package/import-local", body: "module.exports=4;\n"},
		{name: "package/module-entry", body: "module.exports=5;\n"},
		{name: "package/browser-entry", body: "module.exports=6;\n"},
		{name: "package/bin/tool", body: "console.log('DATA');\n"},
		{name: "package/bin/deep/tool", body: "console.log('DATA');\n"},
		{name: "package/nested-entry", body: "module.exports=7;\n"},
		{name: "package/nested/child", body: "module.exports=8;\n"},
		{name: "package/external-name", body: "not local source"},
		{name: "package/data.json", body: `{"value":1}`},
		{name: "package/nested/package.json", body: `{"main":"../nested-entry","exports":"./child"}`},
		{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","exports":{".":{"node":"./export-node?mode=test","default":"./export-default"},"./feature/*":"./src/*/entry","./data":"./data.json"},"imports":{"#local":["external-package","./import-local"],"#external":"external-name"},"module":"./module-entry","browser":{"./server":"./browser-entry","fs":false},"directories":{"bin":"bin"}}`},
	}
	subject := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
	inspector, _ := NewInspector(root)
	archive, sources, _, err := inspector.InspectArchiveContent(context.Background(), subject)
	if err != nil || !archive.Matches(subject) {
		t.Fatal("metadata source collection failed", err)
	}
	wanted := map[string]bool{"export-node": true, "export-default": true, "src/a/b/entry": true, "import-local": true, "module-entry": true, "browser-entry": true, "bin/tool": true, "bin/deep/tool": true, "nested-entry": true, "nested/child": true}
	if len(sources) != len(wanted) {
		t.Fatal("metadata source targets omitted or DATA guessed as source", len(sources))
	}
	for _, source := range sources {
		if !wanted[source.Path] || source.Language != "javascript" {
			t.Fatal("wrong source role", source.Path)
		}
		delete(wanted, source.Path)
	}
	if len(wanted) != 0 {
		t.Fatal("metadata sources omitted or duplicated", wanted)
	}
}

func TestDeclaredMetadataRequiresEveryMeasuredBody(t *testing.T) {
	root := canonicalRoot(t)
	manifest := `{"name":"tiny","version":"1.2.3"}`
	nested := `{"main":"../entry"}`
	subject := measuredInventorySubject(t, root, writeArtifact(t, root, []tarEntry{
		{name: "package/package.json", body: manifest},
		{name: "package/nested/package.json", body: nested},
		{name: "package/entry", body: "module.exports=1;\n"},
	}))
	inspector, _ := NewInspector(root)
	archive, _, err := inspector.InspectArchive(context.Background(), subject)
	if err != nil {
		t.Fatal(err)
	}
	for _, failure := range []string{"none", "missing", "extra", "substituted"} {
		metadata := map[string][]byte{"package.json": []byte(manifest), "nested/package.json": []byte(nested)}
		switch failure {
		case "missing":
			delete(metadata, "nested/package.json")
		case "extra":
			metadata["foreign/package.json"] = []byte(`{}`)
		case "substituted":
			metadata["nested/package.json"] = []byte(`{"main":"../other"}`)
		}
		targets, err := archive.DeclaredSourcePaths(metadata)
		if failure == "none" {
			if err != nil || !targets["entry"] || len(targets) != 1 {
				t.Fatal("measured metadata did not bind", targets, err)
			}
		} else if err == nil || targets != nil {
			t.Fatal("unmeasured metadata became source authority", failure)
		}
	}
}

func TestArchiveContentRejectsUnsafeNestedMetadataAndTargets(t *testing.T) {
	for _, nested := range []string{
		`{"main":"../../outside"}`,
		`{"exports":"../../outside"}`,
		`{"exports":"./../../outside?query"}`,
		`{"imports":{"#local":"./%2e%2e/%2e%2e/outside"}}`,
		`{"directories":{"bin":"../../outside"}}`,
		`{"exports":123}`,
		`{"exports":"./entry","exports":"./other"}`,
		`{`,
	} {
		root := canonicalRoot(t)
		subject := measuredInventorySubject(t, root, writeArtifact(t, root, []tarEntry{
			{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`},
			{name: "package/nested/package.json", body: nested},
		}))
		inspector, _ := NewInspector(root)
		archive, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
		if err == nil || archive.Valid() || len(sources) != 0 || len(report.Executions()) > 1 {
			t.Fatal("unsafe nested metadata produced content facts", nested, err)
		}
	}
}

func TestArchiveContentBoundsCombinedMetadataPatterns(t *testing.T) {
	for _, bound := range []string{"count", "bytes"} {
		root := canonicalRoot(t)
		entries := []tarEntry{{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3"}`}}
		count, width := 6000, 0
		if bound == "bytes" {
			count, width = 1000, 2200
		}
		for document := 0; document < 2; document++ {
			targets := make([]string, count)
			for i := range targets {
				targets[i] = "./" + strings.Repeat("a", width) + strconv.Itoa(i) + "*"
			}
			body, err := json.Marshal(map[string]any{"exports": targets})
			if err != nil {
				t.Fatal(err)
			}
			entries = append(entries, tarEntry{name: "package/nested" + strconv.Itoa(document) + "/package.json", body: string(body)})
		}
		subject := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
		inspector, _ := NewInspector(root)
		archive, sources, report, err := inspector.InspectArchiveContent(context.Background(), subject)
		if err == nil || archive.Valid() || len(sources) != 0 || len(report.Executions()) > 1 {
			t.Fatal("metadata expression budget was bypassed", bound, err)
		}
	}
}
