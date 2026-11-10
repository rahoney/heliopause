package npm

import (
	"archive/tar"
	"context"
	"errors"
	"testing"
)

func TestBunOriginalLayoutProvesWholeHostOwnerWithoutGuestAuthority(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		f := bunLayoutFixture(t, repeated)
		for n := range f.entries {
			f.entries[n].header.Uid = 1234
			f.entries[n].header.Gid = 2345
		}
		body := layoutTarBody(t, f.entries)
		if _, err := ParseBunLayoutArchive(context.Background(), body); err == nil {
			t.Fatal("Host owner admitted by guest parser")
		}
		original, err := ParseBunOriginalLayout(context.Background(), body, 1234, 2345)
		if err != nil {
			t.Fatal(err)
		}
		clear(body)
		if _, err := f.input.BindLayout(context.Background(), f.inventories, original.layout); err == nil {
			t.Fatal("guest binder admitted Host owner")
		}
		proof, err := f.input.ProveOriginalLayout(context.Background(), f.inventories, original)
		if err != nil || proof.String() == "" {
			t.Fatal("whole original source proof", err)
		}
		if _, err := f.input.ProveOriginalLayout(context.Background(), f.inventories[:1], original); err == nil {
			t.Fatal("partial source proof admitted")
		}
		ctx, stop := context.WithCancelCause(context.Background())
		cause := errors.New("original cancellation")
		stop(cause)
		if _, err := f.input.ProveOriginalLayout(ctx, f.inventories, original); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
			t.Fatal("original cause lost", err)
		}
	}
}

func TestBunOriginalLayoutRejectsIncompleteForgedAndUnsafeOwnedTree(t *testing.T) {
	for _, fault := range []string{"missing", "extra", "bytes", "mode", "owner", "bin", "hardlink", "package-link", "metadata"} {
		t.Run(fault, func(t *testing.T) {
			f := bunLayoutFixture(t, false)
			for n := range f.entries {
				e := &f.entries[n]
				if e.header.Name == "node_modules/react/LICENSE" {
					switch fault {
					case "missing":
						f.entries = append(f.entries[:n], f.entries[n+1:]...)
					case "bytes":
						e.body = "X" + e.body[1:]
					case "mode":
						e.header.Mode = 0o755
					case "owner":
						e.header.Uid = 0
					case "hardlink":
						e.header.Typeflag = tar.TypeLink
						e.header.Linkname = "node_modules/js-tokens/LICENSE"
						e.header.Size = 0
						e.body = ""
					}
					break
				}
			}
			if fault == "bin" {
				for n := range f.entries {
					if f.entries[n].header.Name == "node_modules/.bin/loose-envify" {
						f.entries[n].header.Linkname = "../react/package.json"
					}
				}
			}
			if fault == "package-link" {
				for n := range f.entries {
					if f.entries[n].header.Name == "node_modules/react" {
						f.entries[n].header.Typeflag = tar.TypeSymlink
						f.entries[n].header.Mode = 0o777
						f.entries[n].header.Linkname = "js-tokens"
					}
				}
			}
			if fault == "extra" || fault == "metadata" {
				name := "node_modules/react/extra.js"
				if fault == "metadata" {
					name = "node_modules/.modules.yaml"
				}
				f.entries = append(f.entries, layoutTestEntry(name, tar.TypeReg, "unknown bytes", ""))
			}
			for n := range f.entries {
				if fault != "owner" || f.entries[n].header.Uid != 0 {
					f.entries[n].header.Uid = 1234
				}
				f.entries[n].header.Gid = 2345
			}
			layout, err := ParseBunOriginalLayout(context.Background(), layoutTarBody(t, f.entries), 1234, 2345)
			if err != nil {
				if fault != "owner" && fault != "package-link" && fault != "hardlink" {
					t.Fatal("negative failed before binder", err)
				}
				return
			}
			if got, err := f.input.ProveOriginalLayout(context.Background(), f.inventories, layout); err == nil || got.String() != "" {
				t.Fatal("unbound native entry admitted")
			}
		})
	}
}
