package npm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func executionEntryInventory(t *testing.T, name, fields string) ArchiveInventory {
	t.Helper()
	base := inventorySubject(t)
	identity, err := domain.NewResolvedArtifactIdentity(base.Identity().Source(), name, "1.2.3", "tarball")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifact(identity, base.Digest(), base.ContentHandle(), base.SizeBytes())
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte(fmt.Sprintf(`{"name":%q,"version":"1.2.3"%s}`, name, fields))
	entries := []ArchiveEntry{
		{Path: "cli.js", Size: 3, Mode: 0o644, Digest: inventoryDigest([]byte("abc"))},
		{Path: "other.js", Size: 3, Mode: 0o644, Digest: inventoryDigest([]byte("xyz"))},
		{Path: "dir", Directory: true, Mode: 0o755},
		{Path: "package.json", Size: int64(len(manifest)), Mode: 0o644, Digest: inventoryDigest(manifest)},
	}
	inventory, err := NewArchiveInventory(artifact, entries, manifest)
	if err != nil {
		t.Fatal(err)
	}
	return inventory
}

func TestExecutionEntryRequiresExactDeclaredCommand(t *testing.T) {
	for _, test := range []struct {
		name, pkg, fields, command, want string
	}{
		{"bare string", "tiny", `,"bin":"./cli.js"`, "", "tiny"},
		{"scoped string", "@scope/tiny", `,"bin":"cli.js"`, "", "tiny"},
		{"single matching map", "tiny", `,"bin":{"tiny":"cli.js"}`, "", "tiny"},
		{"different command explicit", "tiny", `,"bin":{"tool":"cli.js"}`, "tool", "tool"},
		{"different command needs selection", "tiny", `,"bin":{"tool":"cli.js"}`, "", ""},
		{"multiple bins need selection", "tiny", `,"bin":{"tiny":"cli.js","other":"other.js"}`, "", ""},
		{"multiple explicit", "tiny", `,"bin":{"tiny":"cli.js","other":"other.js"}`, "other", "other"},
		{"same target aliases need selection", "tiny", `,"bin":{"tiny":"cli.js","alias":"cli.js"}`, "", ""},
		{"alias explicit", "tiny", `,"bin":{"tiny":"cli.js","alias":"cli.js"}`, "alias", "alias"},
		{"unknown command", "tiny", `,"bin":"cli.js"`, "other", ""},
		{"command string not evaluated", "tiny", `,"bin":"cli.js"`, "tiny;other", ""},
		{"absent bin", "tiny", "", "", ""},
		{"null bin", "tiny", `,"bin":null`, "", ""},
		{"missing member", "tiny", `,"bin":"missing.js"`, "", ""},
		{"directory member", "tiny", `,"bin":"dir"`, "", ""},
		{"escape", "tiny", `,"bin":"../cli.js"`, "", ""},
		{"duplicate bin key", "tiny", `,"bin":"cli.js","bin":"other.js"`, "", ""},
		{"directory discovery", "tiny", `,"directories":{"bin":"dir"}`, "", ""},
		{"bundled dependency", "tiny", `,"bin":"cli.js","bundledDependencies":["foreign"]`, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			inventory := executionEntryInventory(t, test.pkg, test.fields)
			entry, err := SelectExecutionEntry(inventory, test.command)
			if test.want == "" {
				if err == nil || entry.Valid() || entry.Matches(inventory) {
					t.Fatal("ambiguous or invalid selection accepted")
				}
				return
			}
			if err != nil || !entry.Valid() || entry.Command() != test.want || entry.Artifact() != inventory.Artifact() || !entry.Matches(inventory) {
				t.Fatal("exact original entry selection failed", err)
			}
			want := "cli.js"
			if test.command == "other" {
				want = "other.js"
			}
			if entry.Target().Path != want || entry.Target().Directory || entry.Target().Mode != 0o644 {
				t.Fatal("original member facts changed")
			}
		})
	}
}

func TestExecutionEntryRejectsInventorySubstitution(t *testing.T) {
	original := executionEntryInventory(t, "tiny", `,"bin":"cli.js"`)
	entry, err := SelectExecutionEntry(original, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"handle", "archive digest", "archive size", "manifest", "target digest", "target size", "target mode", "other member"} {
		t.Run(change, func(t *testing.T) {
			artifact, entries, manifest := original.Artifact(), original.Entries(), original.Manifest()
			if change == "handle" || change == "archive digest" || change == "archive size" {
				handle, digest, size := artifact.ContentHandle(), artifact.Digest(), artifact.SizeBytes()
				switch change {
				case "handle":
					run, runErr := domain.NewRunID()
					if runErr != nil {
						t.Fatal(runErr)
					}
					handle = "intake:" + run.String() + ":tarball"
				case "archive digest":
					digest, _ = domain.NewSHA256Digest(strings.Repeat("b", 64))
				case "archive size":
					size++
				}
				artifact, err = domain.NewAcquiredArtifact(artifact.Identity(), digest, handle, size)
				if err != nil {
					t.Fatal(err)
				}
			}
			if change == "manifest" {
				manifest = []byte(strings.Replace(string(manifest), `"bin":"cli.js"`, `"bin":{"tiny":"cli.js"}`, 1))
			}
			for i := range entries {
				file := &entries[i]
				if change == "manifest" && file.Path == "package.json" {
					file.Size, file.Digest = int64(len(manifest)), inventoryDigest(manifest)
				}
				if file.Path == "cli.js" {
					switch change {
					case "target digest":
						file.Digest = inventoryDigest([]byte("new"))
					case "target size":
						file.Size++
					case "target mode":
						file.Mode = 0o755
					}
				}
				if change == "other member" && file.Path == "other.js" {
					file.Digest = inventoryDigest([]byte("new"))
				}
			}
			changed, err := NewArchiveInventory(artifact, entries, manifest)
			if err != nil {
				t.Fatal("substitution fixture invalid", err)
			}
			if entry.Matches(changed) {
				t.Fatal("different acquisition or full inventory substituted")
			}
		})
	}
	manifest, entries := original.Manifest(), original.Entries()
	manifest[0], entries[0].Mode = '!', 0o777
	target := entry.Target()
	target.Path = "changed"
	if !entry.Matches(original) || entry.Target().Path != "cli.js" {
		t.Fatal("returned copies changed original binding")
	}
	var zero ExecutionEntry
	if zero.Valid() || zero.Matches(original) || entry.Matches(ArchiveInventory{}) {
		t.Fatal("zero binding accepted")
	}
	if selected, err := SelectExecutionEntry(ArchiveInventory{}, "tiny"); err == nil || selected.Valid() {
		t.Fatal("zero inventory accepted")
	}
}
