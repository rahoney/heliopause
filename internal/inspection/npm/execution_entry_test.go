package npm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func executionReadFixture(t *testing.T, entries []tarEntry, command string) (*Inspector, artifactnpm.ArchiveInventory, artifactnpm.ExecutionEntry, string) {
	t.Helper()
	root := canonicalRoot(t)
	artifact := measuredInventorySubject(t, root, writeArtifact(t, root, entries))
	inspector, err := NewInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	inventory, report, err := inspector.InspectArchive(context.Background(), artifact)
	if err != nil || !inventory.Valid() || len(report.Findings()) != 0 {
		t.Fatal("fixture inventory rejected", err)
	}
	entry, err := artifactnpm.SelectExecutionEntry(inventory, command)
	if err != nil {
		t.Fatal(err)
	}
	return inspector, inventory, entry, root
}

func nodeReadDigest(t *testing.T, body []byte) domain.ContentDigest {
	t.Helper()
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func TestReadBunEntryMeasuresOriginalAndKeepsNodeBoundary(t *testing.T) {
	for _, test := range []struct{ file, body string }{
		{"cli.ts", "#!/usr/bin/env bun\nconst x: number = 1;\nconsole.log(x);\n"},
		{"cli.ts", "const x: number = 1;\n"},
		{"bin/tool", "#!/usr/bin/env -S bun\nconsole.log('ok');\n"},
	} {
		inspector, inventory, entry, _ := executionReadFixture(t, []tarEntry{
			{name: "package/" + test.file, body: test.body},
			{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","bin":"` + test.file + `"}`},
			{name: "package/other.js", body: "other"},
		}, "")
		proof, err := inspector.ReadBunExecutionEntry(context.Background(), entry)
		if err != nil || !proof.Matches(inventory) || !bytes.Equal(proof.Bytes(), []byte(test.body)) {
			t.Fatal("actual selected original was not measured", err)
		}
		if old, err := inspector.ReadNodeExecutionEntry(context.Background(), entry); err == nil || old.Valid() {
			t.Fatal("Bun-only target entered existing Node proof")
		}
		members := inventory.Entries()
		for i := range members {
			if members[i].Path == "other.js" {
				members[i].Digest = nodeReadDigest(t, []byte("foreign"))
			}
		}
		forged, err := artifactnpm.NewArchiveInventory(inventory.Artifact(), members, inventory.Manifest())
		if err != nil {
			t.Fatal(err)
		}
		foreign, err := artifactnpm.SelectExecutionEntry(forged, "")
		if err != nil {
			t.Fatal(err)
		}
		if result, err := inspector.ReadBunExecutionEntry(context.Background(), foreign); err == nil || result.Valid() {
			t.Fatal("changed unselected member bypassed complete original equality")
		}
		cause := errors.New("stopped Bun entry read")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		if result, err := inspector.ReadBunExecutionEntry(ctx, entry); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || result.Valid() {
			t.Fatal("Bun entry read lost cancellation/cause", err)
		}
	}
}

func TestReadNodeEntryUsesActualSelectedBytesRegardlessOfArchiveOrder(t *testing.T) {
	for _, test := range []struct {
		name, member, body, manifest, command string
		manifestLast                          bool
	}{
		{"manifest first", "cli.js", "console.log('fixture')\n", `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`, "", false},
		{"manifest last", "cli.js", "console.log('fixture')\n", `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`, "", true},
		{"empty JS", "cli.js", "", `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`, "", true},
		{"closed shebang", "bin/tool", "#!/usr/bin/env node\nconsole.log('fixture')\n", `{"name":"tiny","version":"1.2.3","bin":{"tool":"bin/tool"}}`, "tool", true},
		{"explicit alias", "cli.js", "console.log('alias')\n", `{"name":"tiny","version":"1.2.3","bin":{"tiny":"cli.js","alias":"cli.js"}}`, "alias", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			entries := []tarEntry{{name: "package/" + test.member, body: test.body}, {name: "package/other.js", body: "other bytes"}}
			manifest := tarEntry{name: "package/package.json", body: test.manifest}
			if test.manifestLast {
				entries = append(entries, manifest)
			} else {
				entries = append([]tarEntry{manifest}, entries...)
			}
			inspector, inventory, entry, root := executionReadFixture(t, entries, test.command)
			proof, err := inspector.ReadNodeExecutionEntry(context.Background(), entry)
			if err != nil || !proof.Valid() || !proof.Matches(inventory) || !bytes.Equal(proof.Bytes(), []byte(test.body)) || proof.Entry().Target() != entry.Target() || proof.Entry().Command() != entry.Command() {
				t.Fatal("actual bytes or whole original binding lost", err)
			}
			files, err := os.ReadDir(filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa"))
			if err != nil || len(files) != 1 || files[0].Name() != "tarball.tgz" {
				t.Fatal("entry read extracted or published files", err)
			}
		})
	}
}

func TestReadNodeEntryRejectsUnmeasuredInventoryFacts(t *testing.T) {
	for _, change := range []string{"selected digest", "other digest", "mode", "size", "manifest", "acquisition"} {
		t.Run(change, func(t *testing.T) {
			inspector, inventory, _, _ := executionReadFixture(t, []tarEntry{
				{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`},
				{name: "package/cli.js", body: "console.log('fixture')\n"},
				{name: "package/other.js", body: "other bytes"},
			}, "")
			files, manifest, subject := inventory.Entries(), inventory.Manifest(), inventory.Artifact()
			if change == "manifest" {
				manifest = append(manifest, ' ')
			}
			for j := range files {
				member := &files[j]
				if change == "selected digest" && member.Path == "cli.js" || change == "other digest" && member.Path == "other.js" {
					member.Digest = nodeReadDigest(t, []byte("invented"))
				}
				if change == "mode" && member.Path == "other.js" {
					member.Mode ^= 0o111
				}
				if change == "size" && member.Path == "other.js" {
					member.Size++
				}
				if change == "manifest" && member.Path == "package.json" {
					member.Size = int64(len(manifest))
					member.Digest = nodeReadDigest(t, manifest)
				}
			}
			if change == "acquisition" {
				run, err := domain.NewRunID()
				if err != nil {
					t.Fatal(err)
				}
				subject, err = domain.NewAcquiredArtifact(subject.Identity(), subject.Digest(), "intake:"+run.String()+":tarball", subject.SizeBytes())
				if err != nil {
					t.Fatal(err)
				}
			}
			forged, err := artifactnpm.NewArchiveInventory(subject, files, manifest)
			if err != nil {
				t.Fatal("structural fixture invalid", err)
			}
			selected, err := artifactnpm.SelectExecutionEntry(forged, "")
			if err != nil {
				t.Fatal(err)
			}
			if proof, err := inspector.ReadNodeExecutionEntry(context.Background(), selected); err == nil || proof.Valid() {
				t.Fatal("unmeasured facts produced Node proof")
			}
		})
	}
}

func TestReadNodeEntryRejectsSourceMutationAndSymlink(t *testing.T) {
	for _, change := range []string{"crc", "trailing", "symlink"} {
		t.Run(change, func(t *testing.T) {
			inspector, _, entry, root := executionReadFixture(t, []tarEntry{
				{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`},
				{name: "package/cli.js", body: "console.log('fixture')\n"},
			}, "")
			file := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
			if change == "symlink" {
				if err := os.Rename(file, file+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(file+".original", file); err != nil {
					t.Fatal(err)
				}
			} else {
				body, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if change == "crc" {
					body[len(body)-8] ^= 1
				} else {
					body = append(body, 1)
				}
				if err := os.WriteFile(file, body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if proof, err := inspector.ReadNodeExecutionEntry(context.Background(), entry); err == nil || proof.Valid() {
				t.Fatal("mutated or symlinked source produced proof")
			}
		})
	}
}

func TestReadNodeEntryRejectsOtherInterpreterAndPreservesCancellation(t *testing.T) {
	for _, body := range []string{"#!/bin/sh\necho fixture\n", "#!/usr/bin/env python\n", "#!/usr/bin/env node --eval fixture\n"} {
		inspector, _, entry, _ := executionReadFixture(t, []tarEntry{
			{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`},
			{name: "package/cli.js", body: body},
		}, "")
		if proof, err := inspector.ReadNodeExecutionEntry(context.Background(), entry); err == nil || proof.Valid() {
			t.Fatal("other interpreter admitted")
		}
	}
	inspector, _, entry, _ := executionReadFixture(t, []tarEntry{
		{name: "package/package.json", body: `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`},
		{name: "package/cli.js", body: "console.log('fixture')\n"},
	}, "")
	cause := errors.New("caller stopped entry read")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if proof, err := inspector.ReadNodeExecutionEntry(ctx, entry); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || proof.Valid() {
		t.Fatal("entry read lost cancellation or cause", err)
	}
	for _, input := range []struct {
		ctx   context.Context
		entry artifactnpm.ExecutionEntry
	}{
		{nil, entry}, {context.Background(), artifactnpm.ExecutionEntry{}},
	} {
		if proof, err := inspector.ReadNodeExecutionEntry(input.ctx, input.entry); err == nil || proof.Valid() {
			t.Fatal("missing input admitted")
		}
	}
	var absent *Inspector
	if proof, err := absent.ReadNodeExecutionEntry(context.Background(), entry); err == nil || proof.Valid() {
		t.Fatal("nil inspector admitted")
	}
}
