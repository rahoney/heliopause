package npm

import (
	"archive/tar"
	"context"
	"errors"
	"strings"
	"testing"
)

func originalLayoutFixture(t *testing.T) inspectionLayoutFixture {
	t.Helper()
	f := fullInspectionLayoutFixture(t, true)
	for index := range f.entries {
		f.entries[index].header.Uid, f.entries[index].header.Gid = 1234, 2345
	}
	return f
}

func TestPNPMOriginalLayoutProvesWholeSourceWithoutGuestAuthority(t *testing.T) {
	for _, cache := range []bool{false, true} {
		f := originalLayoutFixture(t)
		if !cache {
			var entries []layoutTarTestEntry
			for _, entry := range f.entries {
				if entry.header.Name != pnpmWorkspaceCachePath {
					entries = append(entries, entry)
				}
			}
			f.entries = entries
		}
		body := layoutTarBody(t, f.entries)
		if _, err := ParsePNPMLayoutArchive(context.Background(), body); err == nil {
			t.Fatal("Host ownership admitted by guest parser")
		}
		original, err := ParsePNPMOriginalLayout(context.Background(), body, 1234, 2345)
		if err != nil {
			t.Fatal(err)
		}
		clear(body)
		proof, err := f.input.ProveOriginalLayout(context.Background(), f.inventories, original)
		if err != nil || proof.String() == "" {
			t.Fatal("complete old peer tree rejected", err)
		}
		if _, err := f.input.ProveOriginalLayout(context.Background(), f.inventories[:1], original); err == nil {
			t.Fatal("partial old source coverage admitted")
		}
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("original cancellation")
		cancel(cause)
		if _, err := f.input.ProveOriginalLayout(ctx, f.inventories, original); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
			t.Fatal("original cause lost", err)
		}
	}
	guest := fullInspectionLayoutFixture(t, false)
	if _, err := guest.input.BindLayout(context.Background(), guest.inventories, guest.parse(t), "/tmp/foreign-store"); err == nil {
		t.Fatal("guest store binding weakened")
	}
}

func TestPNPMOriginalLayoutRejectsIncompleteOrForgedOwnership(t *testing.T) {
	for _, change := range []string{"extra", "missing", "payload", "peer-link", "bin", "metadata", "store", "cache", "hardlink", "owner", "mode", "escape"} {
		t.Run(change, func(t *testing.T) {
			f := originalLayoutFixture(t)
			changed := false
			for index := range f.entries {
				entry := &f.entries[index]
				name := entry.header.Name
				switch {
				case change == "missing" && strings.HasSuffix(name, "/LICENSE"):
					f.entries = append(f.entries[:index], f.entries[index+1:]...)
					changed = true
				case change == "payload" && strings.HasSuffix(name, "/LICENSE"):
					entry.body = "changed old code"
					changed = true
				case change == "peer-link" && entry.header.Typeflag == tar.TypeSymlink:
					entry.header.Linkname = ".pnpm"
					changed = true
				case change == "bin" && strings.Contains(name, "/.bin/"):
					entry.body = "#!/bin/sh\nexit 0\n"
					changed = true
				case change == "metadata" && name == "node_modules/.modules.yaml":
					entry.body = strings.ReplaceAll(entry.body, `"isolated"`, `"hoisted"`)
					changed = true
				case change == "store" && name == "node_modules/.modules.yaml":
					entry.body = strings.ReplaceAll(entry.body, "/tmp/haa-store/v11", "/tmp/../foreign/v11")
					changed = true
				case change == "cache" && name == pnpmWorkspaceCachePath:
					entry.body = `{"duplicate":1,"duplicate":2}`
					changed = true
				case change == "hardlink" && strings.HasSuffix(name, "/LICENSE"):
					entry.header.Typeflag, entry.header.Linkname, entry.body = tar.TypeLink, "node_modules/.modules.yaml", ""
					changed = true
				case change == "owner":
					entry.header.Uid++
					changed = true
				case change == "mode" && entry.header.Typeflag == tar.TypeDir:
					entry.header.Mode = 0o777
					changed = true
				case change == "escape" && entry.header.Typeflag == tar.TypeSymlink:
					entry.header.Linkname = "../../../outside"
					changed = true
				}
				if changed {
					break
				}
			}
			if change == "extra" {
				entry := layoutTestEntry("node_modules/foreign.js", tar.TypeReg, "foreign", "")
				entry.header.Uid, entry.header.Gid = 1234, 2345
				f.entries = append(f.entries, entry)
				changed = true
			}
			if !changed {
				t.Fatal("negative did not alter fixture")
			}
			for index := range f.entries {
				f.entries[index].header.Size = int64(len(f.entries[index].body))
			}
			original, err := ParsePNPMOriginalLayout(context.Background(), layoutTarBody(t, f.entries), 1234, 2345)
			if err == nil {
				_, err = f.input.ProveOriginalLayout(context.Background(), f.inventories, original)
			}
			if err == nil {
				t.Fatal("forged original ownership admitted")
			}
		})
	}
}
