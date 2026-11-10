package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

type layoutTarTestEntry struct {
	header tar.Header
	body   string
}

func layoutTestEntry(name string, kind byte, body, link string) layoutTarTestEntry {
	mode := int64(0o644)
	if kind == tar.TypeDir {
		mode = 0o755
	} else if kind == tar.TypeSymlink {
		mode = 0o777
	}
	return layoutTarTestEntry{tar.Header{Name: name, Typeflag: kind, Mode: mode, Uid: 1000, Gid: 1000, Size: int64(len(body)), Linkname: link}, body}
}

func layoutTarBody(t *testing.T, entries []layoutTarTestEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	for _, item := range entries {
		if err := writer.WriteHeader(&item.header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(item.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestPNPMLayoutArchivePreservesRegularAndAliasBytes(t *testing.T) {
	// Docker may encode the same LICENSE bytes in different packages as a
	// hardlink. Forward aliases are also parsed without Host extraction.
	entries := []layoutTarTestEntry{
		layoutTestEntry("node_modules/", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/first", tar.TypeLink, "", "node_modules/second"),
		layoutTestEntry("node_modules/second", tar.TypeLink, "", "node_modules/LICENSE"),
		layoutTestEntry("node_modules/LICENSE", tar.TypeReg, "license payload\n", ""),
		layoutTestEntry("node_modules/current", tar.TypeSymlink, "", "LICENSE"),
	}
	body := layoutTarBody(t, entries)
	parsed, err := ParsePNPMLayoutArchive(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	clear(body)
	got := parsed.Entries()
	got[0].Name = "changed"
	if parsed.Entries()[0].Name != "node_modules" {
		t.Fatal("entry getter exposed internal storage")
	}
	digest := sha256.Sum256([]byte("license payload\n"))
	for _, name := range []string{"node_modules/LICENSE", "node_modules/first", "node_modules/second"} {
		file, err := parsed.ReadFile(name)
		if err != nil || string(file) != "license payload\n" {
			t.Fatal("regular/alias bytes changed", name, err)
		}
		file[0] = '!'
		again, _ := parsed.ReadFile(name)
		if string(again) != "license payload\n" {
			t.Fatal("file getter exposed internal storage")
		}
		found := false
		for _, item := range parsed.Entries() {
			if item.Name == name {
				found = item.Digest.String() == hex.EncodeToString(digest[:])
			}
		}
		if !found {
			t.Fatal("regular/alias digest lost", name)
		}
	}
	for _, name := range []string{"node_modules", "node_modules/current", "missing"} {
		if _, err := parsed.ReadFile(name); err == nil {
			t.Fatal("non-regular entry was followed", name)
		}
	}
	if _, err := (PNPMLayoutArchive{}).ReadFile("node_modules/LICENSE"); err == nil {
		t.Fatal("zero layout yielded a file")
	}
}

func TestPNPMLayoutArchiveRejectsUnsafeEnvelopes(t *testing.T) {
	for _, name := range []string{"wrong-root", "duplicate", "escape", "backslash", "noncanonical", "missing-parent", "file-parent", "symlink-parent", "symlink-outside", "hardlink-outside", "hardlink-missing", "hardlink-cycle", "wrong-owner", "wrong-group", "setuid", "negative-mode", "special", "unknown-pax", "xattr", "nonlink-target", "truncated", "missing-trailer", "nonzero-tail"} {
		t.Run(name, func(t *testing.T) {
			entries := []layoutTarTestEntry{layoutTestEntry("node_modules/", tar.TypeDir, "", ""), layoutTestEntry("node_modules/file", tar.TypeReg, "payload", "")}
			switch name {
			case "wrong-root":
				entries[0].header.Name = "foreign/"
			case "duplicate":
				entries = append(entries, entries[1])
			case "escape":
				entries[1].header.Name = "../outside"
			case "backslash":
				entries[1].header.Name = "node_modules/a\\b"
			case "noncanonical":
				entries[1].header.Name = "node_modules/a/../file"
			case "missing-parent":
				entries[1].header.Name = "node_modules/missing/file"
			case "file-parent":
				entries = append(entries, layoutTestEntry("node_modules/file/child", tar.TypeReg, "x", ""))
			case "symlink-parent":
				entries = append(entries, layoutTestEntry("node_modules/link", tar.TypeSymlink, "", "file"), layoutTestEntry("node_modules/link/child", tar.TypeReg, "x", ""))
			case "symlink-outside":
				entries = append(entries, layoutTestEntry("node_modules/link", tar.TypeSymlink, "", "../../outside"))
			case "hardlink-outside":
				entries = append(entries, layoutTestEntry("node_modules/link", tar.TypeLink, "", "../outside"))
			case "hardlink-missing":
				entries = append(entries, layoutTestEntry("node_modules/link", tar.TypeLink, "", "node_modules/missing"))
			case "hardlink-cycle":
				entries = append(entries, layoutTestEntry("node_modules/first", tar.TypeLink, "", "node_modules/second"), layoutTestEntry("node_modules/second", tar.TypeLink, "", "node_modules/first"))
			case "wrong-owner":
				entries[1].header.Uid = 0
			case "wrong-group":
				entries[1].header.Gid = 0
			case "setuid":
				entries[1].header.Mode = 0o4755
			case "negative-mode":
				entries[1].header.Mode = -1
			case "special":
				entries = append(entries, layoutTestEntry("node_modules/device", tar.TypeChar, "", ""))
			case "unknown-pax":
				entries[1].header.PAXRecords = map[string]string{"vendor.extra": "unreviewed"}
			case "xattr":
				entries[1].header.PAXRecords = map[string]string{"SCHILY.xattr.security.capability": "unreviewed"}
			case "nonlink-target":
				entries[1].header.Linkname = "unexpected"
			}
			body := layoutTarBody(t, entries)
			switch name {
			case "truncated":
				body = body[:1200]
			case "missing-trailer":
				body = body[:len(body)-1024]
			case "nonzero-tail":
				body = append(body, []byte("hidden")...)
			}
			parsed, err := ParsePNPMLayoutArchive(context.Background(), body)
			if err == nil || len(parsed.Entries()) != 0 {
				t.Fatal("unsafe envelope yielded parsed data")
			}
		})
	}
}

func TestPNPMLayoutArchiveBoundsAndCancellation(t *testing.T) {
	root := layoutTestEntry("node_modules/", tar.TypeDir, "", "")
	longPath := layoutTestEntry("node_modules/"+strings.Repeat("a", 4096), tar.TypeReg, "x", "")
	if _, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, []layoutTarTestEntry{root, longPath})); err == nil {
		t.Fatal("oversized path accepted")
	}
	header := longPath.header
	header.Name, header.Size = "node_modules/large", ArchiveFileLimit+1
	if validPNPMLayoutHeader(&header) {
		t.Fatal("oversized declared file accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if parsed, err := ParsePNPMLayoutArchive(ctx, layoutTarBody(t, []layoutTarTestEntry{root})); !errors.Is(err, context.Canceled) || len(parsed.Entries()) != 0 {
		t.Fatal("cancellation lost", err)
	}
}
