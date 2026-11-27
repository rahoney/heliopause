package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func inventorySubject(t *testing.T) domain.AcquiredArtifact {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 100)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func inventoryDigest(value []byte) domain.ContentDigest {
	h := sha256.Sum256(value)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(h[:]))
	return digest
}

func TestArchiveInventoryBindingAndCopies(t *testing.T) {
	artifact := inventorySubject(t)
	manifest := []byte(`{"name":"tiny","version":"1.2.3"}`)
	entries := []ArchiveEntry{
		{Path: "src/main.js", Size: 3, Mode: 0o755, Digest: inventoryDigest([]byte("abc"))},
		{Path: "package.json", Size: int64(len(manifest)), Mode: 0o644, Digest: inventoryDigest(manifest)},
		{Path: "src", Directory: true, Mode: 0o755},
	}
	inventory, err := NewArchiveInventory(artifact, entries, manifest)
	if err != nil || !inventory.Valid() || !inventory.Matches(artifact) {
		t.Fatal("valid inventory rejected", err)
	}
	entries[0].Path = "changed"
	manifest[0] = '!'
	gotEntries, gotManifest := inventory.Entries(), inventory.Manifest()
	if gotEntries[0].Path != "package.json" || gotManifest[0] != '{' {
		t.Fatal("caller mutation affected inventory or order is nondeterministic")
	}
	gotEntries[0].Path = "changed-again"
	gotManifest[0] = '!'
	if inventory.Entries()[0].Path != "package.json" || inventory.Manifest()[0] != '{' {
		t.Fatal("getter exposed inventory storage")
	}
	foreign, _ := domain.NewAcquiredArtifact(artifact.Identity(), artifact.Digest(), "intake:run_bbbbbbbbbbbbbbbbbbbbbbbbbb:tarball", artifact.SizeBytes())
	if inventory.Matches(foreign) || (ArchiveInventory{}).Matches(artifact) || (ArchiveInventory{}).Valid() {
		t.Fatal("zero or foreign intake subject matched")
	}
}

func TestArchiveInventoryRejectsAmbiguousFiles(t *testing.T) {
	manifest := []byte(`{"name":"tiny","version":"1.2.3"}`)
	base := ArchiveEntry{Path: "package.json", Size: int64(len(manifest)), Mode: 0o644, Digest: inventoryDigest(manifest)}
	for _, item := range []ArchiveEntry{
		base,
		{Path: "../escape", Digest: inventoryDigest(nil)},
		{Path: "/absolute", Digest: inventoryDigest(nil)},
		{Path: "a\\b", Digest: inventoryDigest(nil)},
		{Path: "C:drive", Digest: inventoryDigest(nil)},
		{Path: "a\x00b", Digest: inventoryDigest(nil)},
		{Path: "bad\xff", Digest: inventoryDigest(nil)},
		{Path: "a/../b", Digest: inventoryDigest(nil)},
		{Path: "package.json/child", Digest: inventoryDigest(nil)},
		{Path: "missing-hash"},
		{Path: "negative", Size: -1, Digest: inventoryDigest(nil)},
		{Path: "large", Size: ArchiveFileLimit + 1, Digest: inventoryDigest(nil)},
		{Path: "setuid", Mode: 0o4755, Digest: inventoryDigest(nil)},
		{Path: "negative-mode", Mode: -1, Digest: inventoryDigest(nil)},
		{Path: "dir-size", Directory: true, Size: 1},
		{Path: "dir-hash", Directory: true, Digest: inventoryDigest(nil)},
	} {
		t.Run(strings.ReplaceAll(item.Path, "\x00", "NUL"), func(t *testing.T) {
			if _, err := NewArchiveInventory(inventorySubject(t), []ArchiveEntry{item, base}, manifest); err == nil {
				t.Fatal("accepted ambiguous/unsupported entry", item.Path)
			}
		})
	}
	for _, body := range [][]byte{nil, []byte("{}"), []byte(`{"name":"foreign","version":"1.2.3"}`), []byte(`{"name":"tiny","version":"2.0.0"}`)} {
		if _, err := NewArchiveInventory(inventorySubject(t), []ArchiveEntry{base}, body); err == nil {
			t.Fatal("accepted missing or foreign manifest")
		}
	}
	wrong := base
	wrong.Digest = inventoryDigest([]byte("different"))
	if _, err := NewArchiveInventory(inventorySubject(t), []ArchiveEntry{wrong}, manifest); err == nil {
		t.Fatal("accepted manifest digest drift")
	}
	if _, err := NewArchiveInventory(domain.AcquiredArtifact{}, []ArchiveEntry{base}, manifest); err == nil {
		t.Fatal("accepted zero subject")
	}
}
