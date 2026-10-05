package cargo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"strconv"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func fixtureArchive(t *testing.T, headers []*tar.Header, content []byte) []byte {
	t.Helper()
	var body bytes.Buffer
	gz := gzip.NewWriter(&body)
	writer := tar.NewWriter(gz)
	for _, h := range headers {
		if err := writer.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg && h.Size > 0 {
			if _, err := writer.Write(content[:int(h.Size)]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestCargoArchiveActualPublicCrate(t *testing.T) {
	body, err := os.ReadFile("testdata/itoa-1.0.17.crate")
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := domain.NewResolvedArtifactIdentity(Source(), "itoa", "1.0.17", "crate")
	files, err := ArchiveFiles(body, identity)
	if err != nil || len(files) != 14 || len(files["src/lib.rs"]) == 0 {
		t.Fatalf("actual crate: files=%d err=%v", len(files), err)
	}
	for _, fault := range []string{"crc", "second gzip member", "truncated", "identity"} {
		t.Run(fault, func(t *testing.T) {
			changed := append([]byte(nil), body...)
			id := identity
			switch fault {
			case "crc":
				changed[len(changed)-8] ^= 1
			case "second gzip member":
				changed = append(changed, body...)
			case "truncated":
				changed = changed[:len(changed)-2]
			case "identity":
				id, _ = domain.NewResolvedArtifactIdentity(Source(), "other", "1.0.17", "crate")
			}
			if files, err := ArchiveFiles(changed, id); err == nil || files != nil {
				t.Fatal("uninspected or invalid archive accepted")
			}
		})
	}
}

func TestCargoArchiveRejectsDangerousPathsTypesAndBounds(t *testing.T) {
	identity, _ := domain.NewResolvedArtifactIdentity(Source(), "fixture", "1.2.3", "crate")
	manifest := []byte("[package]\nname=\"fixture\"\nversion=\"1.2.3\"\n")
	base := &tar.Header{Name: "fixture-1.2.3/Cargo.toml", Mode: 0o644, Size: int64(len(manifest)), Typeflag: tar.TypeReg}
	for _, fault := range []string{"escape", "absolute", "dot", "backslash", "symlink", "hardlink", "fifo", "setid", "duplicate", "case collision", "ancestor collision", "ancestor case collision", "wrong manifest", "file cap", "count cap"} {
		t.Run(fault, func(t *testing.T) {
			h := &tar.Header{Name: "fixture-1.2.3/data", Mode: 0o644, Typeflag: tar.TypeReg}
			first := *base
			headers := []*tar.Header{&first, h}
			content := append([]byte(nil), manifest...)
			switch fault {
			case "escape":
				h.Name = "fixture-1.2.3/../escape"
			case "absolute":
				h.Name = "/tmp/escape"
			case "dot":
				h.Name = "fixture-1.2.3/."
			case "backslash":
				h.Name = "fixture-1.2.3/path\\file"
			case "symlink":
				h.Typeflag = tar.TypeSymlink
				h.Linkname = "Cargo.toml"
			case "hardlink":
				h.Typeflag = tar.TypeLink
				h.Linkname = "Cargo.toml"
			case "fifo":
				h.Typeflag = tar.TypeFifo
			case "setid":
				h.Mode = 0o4644
			case "duplicate":
				h.Name = first.Name
			case "case collision":
				h.Name = "fixture-1.2.3/cargo.toml"
			case "ancestor collision":
				headers = append(headers, &tar.Header{Name: h.Name + "/child", Mode: 0o644, Typeflag: tar.TypeReg})
			case "ancestor case collision":
				h.Name = "fixture-1.2.3/Dir/one"
				headers = append(headers, &tar.Header{Name: "fixture-1.2.3/dir/two", Mode: 0o644, Typeflag: tar.TypeReg})
			case "wrong manifest":
				content = bytes.ReplaceAll(content, []byte("fixture"), []byte("unknown"))
			case "file cap":
				// Direct header injection avoids creating an oversized fixture body.
				var raw bytes.Buffer
				gz := gzip.NewWriter(&raw)
				tw := tar.NewWriter(gz)
				h.Size = MaxCrateFileBytes + 1
				if err := tw.WriteHeader(h); err != nil {
					t.Fatal(err)
				}
				if err := gz.Close(); err != nil {
					t.Fatal(err)
				}
				if _, err := ArchiveFiles(raw.Bytes(), identity); err == nil {
					t.Fatal("oversized file accepted")
				}
				return
			case "count cap":
				for i := 0; i < MaxCrateFiles; i++ {
					headers = append(headers, &tar.Header{Name: "fixture-1.2.3/entry" + strconv.Itoa(i), Mode: 0o644, Typeflag: tar.TypeReg})
				}
			}
			body := fixtureArchive(t, headers, content)
			if files, err := ArchiveFiles(body, identity); err == nil || files != nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}
