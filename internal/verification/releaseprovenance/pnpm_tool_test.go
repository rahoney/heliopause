package releaseprovenance

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"testing"
)

// A minimal structural fixture with one GNU interpreter and a libc DT_NEEDED.
// It is not executable or a signed runtime qualification.
func syntheticPNPMELF() []byte {
	body := make([]byte, 768)
	copy(body, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	put16 := func(at int, value uint16) { binary.LittleEndian.PutUint16(body[at:], value) }
	put32 := func(at int, value uint32) { binary.LittleEndian.PutUint32(body[at:], value) }
	put64 := func(at int, value uint64) { binary.LittleEndian.PutUint64(body[at:], value) }
	put16(16, 3)
	put16(18, 62)
	put32(20, 1)
	put64(24, 0x100)
	put64(32, 64)
	put64(40, 512)
	put16(52, 64)
	put16(54, 56)
	put16(56, 2)
	put16(58, 64)
	put16(60, 4)
	put16(62, 3)
	put32(64, 1)
	put32(68, 5)
	put64(96, uint64(len(body)))
	put64(104, uint64(len(body)))
	put64(112, 4096)
	interp := []byte(gnuAMD64Interpreter + "\x00")
	put32(120, 3)
	put32(124, 4)
	put64(128, 176)
	put64(136, 176)
	put64(152, uint64(len(interp)))
	put64(160, uint64(len(interp)))
	put64(168, 1)
	copy(body[176:], interp)
	copy(body[224:], "\x00libc.so.6\x00")
	put64(256, 1)
	put64(264, 1)
	names := "\x00.dynstr\x00.dynamic\x00.shstrtab\x00"
	copy(body[304:], names)
	section := func(index int, name, kind uint32, offset, size uint64, link uint32, entry uint64) {
		start := 512 + index*64
		put32(start, name)
		put32(start+4, kind)
		put64(start+24, offset)
		put64(start+32, size)
		put32(start+40, link)
		put64(start+48, 1)
		put64(start+56, entry)
	}
	section(1, 1, 3, 224, uint64(len("\x00libc.so.6\x00")), 0, 0)
	section(2, 9, 6, 256, 32, 1, 16)
	section(3, 18, 3, 304, uint64(len(names)), 0, 0)
	return body
}

type pnpmTestFile struct {
	header tar.Header
	body   []byte
}

func pnpmToolTestFiles() []pnpmTestFile {
	body := syntheticPNPMELF()
	return []pnpmTestFile{
		{tar.Header{Name: "pnpm", Typeflag: tar.TypeReg, Mode: 0755, Uid: 1001, Gid: 1001, Size: int64(len(body))}, body},
		{tar.Header{Name: "dist", Typeflag: tar.TypeDir, Mode: 0755, Uid: 1001, Gid: 1001}, nil},
		{tar.Header{Name: "dist/data", Typeflag: tar.TypeReg, Mode: 0644, Uid: 1001, Gid: 1001, Size: 4}, []byte("data")},
		{tar.Header{Name: "dist/link", Typeflag: tar.TypeLink, Mode: 0644, Uid: 1001, Gid: 1001, Linkname: "dist/data"}, nil},
	}
}

func syntheticPNPMArchive(t testing.TB, files []pnpmTestFile, trailer []byte) (PNPMReleaseReceipt, []byte) {
	t.Helper()
	var expanded bytes.Buffer
	w := tar.NewWriter(&expanded)
	for _, file := range files {
		if err := w.WriteHeader(&file.header); err != nil {
			t.Fatal(err)
		}
		if len(file.body) > 0 {
			if _, err := w.Write(file.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	expanded.Write(trailer)
	var compressed bytes.Buffer
	z := gzip.NewWriter(&compressed)
	if _, err := z.Write(expanded.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	// Package-private structural coverage only, not publisher authentication.
	r := PNPMReleaseReceipt{identity: PNPMReleaseIdentity{Version: "12.15.0", ArchiveSHA256: uvDigest(compressed.Bytes())}, bundleSHA256: "synthetic-structural-only"}
	return r, compressed.Bytes()
}

func TestPNPMNativeArchiveStructuralBinding(t *testing.T) {
	r, archive := syntheticPNPMArchive(t, pnpmToolTestFiles(), nil)
	tool, err := r.VerifyToolArchive(context.Background(), archive)
	if err != nil {
		t.Fatal(err)
	}
	if tool.Release() != r.Identity() || tool.Size() != 768 || tool.SHA256() != uvDigest(syntheticPNPMELF()) || len(tool.Libraries()) != 1 || tool.Libraries()[0] != "libc.so.6" {
		t.Fatalf("bad native binding: %+v", tool)
	}
	libraries := tool.Libraries()
	libraries[0] = "mutated"
	if tool.Libraries()[0] != "libc.so.6" {
		t.Fatal("receipt libraries are mutable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.VerifyToolArchive(ctx, archive); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel cause lost: %v", err)
	}
	if _, err := (PNPMReleaseReceipt{}).VerifyToolArchive(context.Background(), archive); err == nil {
		t.Fatal("zero receipt accepted")
	}
	changed := append([]byte(nil), archive...)
	changed[len(changed)/2] ^= 1
	if _, err := r.VerifyToolArchive(context.Background(), changed); err == nil {
		t.Fatal("changed authenticated archive accepted")
	}
}

func TestPNPMReadToolRequiresCompleteArchiveBeforeReturningBytes(t *testing.T) {
	r, archive := syntheticPNPMArchive(t, pnpmToolTestFiles(), nil)
	tool, body, err := r.ReadToolArchive(context.Background(), archive)
	if err != nil || !bytes.Equal(body, syntheticPNPMELF()) || tool.SHA256() != uvDigest(body) {
		t.Fatalf("native bytes not bound to the archive: %v", err)
	}
	files := pnpmToolTestFiles()
	files = append(files, pnpmTestFile{header: tar.Header{Name: "late-symlink", Typeflag: tar.TypeSymlink, Linkname: "/foreign"}})
	r, archive = syntheticPNPMArchive(t, files, nil)
	if _, body, err := r.ReadToolArchive(context.Background(), archive); err == nil || body != nil {
		t.Fatal("native bytes escaped before a later invalid member was rejected")
	}
	r, archive = syntheticPNPMArchive(t, pnpmToolTestFiles(), []byte("unexpected trailing data"))
	if _, body, err := r.ReadToolArchive(context.Background(), archive); err == nil || body != nil {
		t.Fatal("native bytes escaped before trailing content was rejected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, body, err := r.ReadToolArchive(ctx, archive); !errors.Is(err, context.Canceled) || body != nil {
		t.Fatal("cancelled extraction exposed native bytes or lost the cause")
	}
}

func TestPNPMNativeArchiveRefusals(t *testing.T) {
	for _, name := range []string{"missing", "duplicate", "traversal", "absolute", "unexpected-root", "symlink", "hardlink-escape", "hardlink-forward", "hardlink-to-native", "native-mode", "data-mode", "elf", "machine", "loader", "load-bounds", "writable-exec", "trailing", "entry-budget"} {
		t.Run(name, func(t *testing.T) {
			files := pnpmToolTestFiles()
			var trailer []byte
			switch name {
			case "missing":
				files = files[1:]
			case "duplicate":
				files = append(files, files[0])
			case "traversal":
				files[2].header.Name = "dist/../../escape"
			case "absolute":
				files[2].header.Name = "/tmp/escape"
			case "unexpected-root":
				files[2].header.Name = "unexpected"
			case "symlink":
				files[3].header.Typeflag = tar.TypeSymlink
			case "hardlink-escape":
				files[3].header.Linkname = "../../escape"
			case "hardlink-forward":
				files[3].header.Linkname = "dist/missing"
			case "hardlink-to-native":
				files[3].header.Linkname = "pnpm"
			case "native-mode":
				files[0].header.Mode = 0644
			case "data-mode":
				files[2].header.Mode = 0666
			case "elf":
				files[0].body[0] = 0
			case "machine":
				binary.LittleEndian.PutUint16(files[0].body[18:], 183)
			case "loader":
				files[0].body[177] = 'x'
			case "load-bounds":
				binary.LittleEndian.PutUint64(files[0].body[96:], 9999999)
			case "writable-exec":
				binary.LittleEndian.PutUint32(files[0].body[68:], 7)
			case "trailing":
				trailer = []byte("not tar padding")
			case "entry-budget":
				for i := 0; i < 1024; i++ {
					files = append(files, pnpmTestFile{header: tar.Header{Name: fmt.Sprintf("dist/d%d", i), Typeflag: tar.TypeDir, Mode: 0755}})
				}
			}
			r, archive := syntheticPNPMArchive(t, files, trailer)
			if _, err := r.VerifyToolArchive(context.Background(), archive); err == nil {
				t.Fatal("invalid native supply accepted")
			}
		})
	}
}

func TestPNPMNativeArchiveCompressionAndTransportBounds(t *testing.T) {
	r, archive := syntheticPNPMArchive(t, pnpmToolTestFiles(), nil)
	for _, body := range [][]byte{nil, []byte("not gzip"), archive[:len(archive)-1], make([]byte, maxPNPMArchiveBytes+1)} {
		r.identity.ArchiveSHA256 = uvDigest(body)
		if _, err := r.VerifyToolArchive(context.Background(), body); err == nil {
			t.Fatal("invalid transport accepted")
		}
	}
}

func TestPNPMNativeArchiveRejectsDeclaredExpansionBeforeReading(t *testing.T) {
	_, compressed := syntheticPNPMArchive(t, pnpmToolTestFiles(), nil)
	for _, size := range []int64{maxPNPMToolBytes + 1, maxPNPMExpandedBytes + 1} {
		z, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(z)
		if err != nil {
			t.Fatal(err)
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		copy(body[124:136], fmt.Sprintf("%011o\x00", size))
		copy(body[148:156], "        ")
		var sum int
		for _, value := range body[:512] {
			sum += int(value)
		}
		copy(body[148:156], fmt.Sprintf("%06o\x00 ", sum))
		var changed bytes.Buffer
		writer := gzip.NewWriter(&changed)
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		r := PNPMReleaseReceipt{identity: PNPMReleaseIdentity{ArchiveSHA256: uvDigest(changed.Bytes())}, bundleSHA256: "synthetic-structural-only"}
		if _, err := r.VerifyToolArchive(context.Background(), changed.Bytes()); err == nil {
			t.Fatal("declared expansion accepted")
		}
	}
}
