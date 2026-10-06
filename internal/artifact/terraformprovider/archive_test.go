package terraformprovider

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/sumdb/dirhash"
)

func providerELFFixture() []byte {
	b := make([]byte, 128)
	copy(b, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(b[16:], 2)
	binary.LittleEndian.PutUint16(b[18:], 62)
	binary.LittleEndian.PutUint32(b[20:], 1)
	binary.LittleEndian.PutUint64(b[32:], 64)
	binary.LittleEndian.PutUint16(b[52:], 64)
	binary.LittleEndian.PutUint16(b[54:], 56)
	binary.LittleEndian.PutUint16(b[56:], 1)
	binary.LittleEndian.PutUint16(b[58:], 64)
	binary.LittleEndian.PutUint32(b[64:], 1)
	binary.LittleEndian.PutUint64(b[96:], 128)
	binary.LittleEndian.PutUint64(b[104:], 128)
	return b
}

func packageFixture(t *testing.T, names []string, modes []os.FileMode, payloads [][]byte) Bundle {
	t.Helper()
	var body bytes.Buffer
	w := zip.NewWriter(&body)
	for i, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Store}
		h.SetMode(modes[i])
		f, e := w.CreateHeader(h)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.Write(payloads[i]); e != nil {
			t.Fatal(e)
		}
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	return Bundle{parts: [][]byte{nil, nil, nil, nil, nil, body.Bytes()}}
}

func TestInspectPackageStandardHashAndExecutionSurfaces(t *testing.T) {
	exe := "terraform-provider-random_v3.7.2_x5"
	elf := providerELFFixture()
	b := packageFixture(t, []string{"LICENSE.txt", exe}, []os.FileMode{0o644, 0o755}, [][]byte{[]byte("license\n"), elf})
	contents, e := InspectPackage(context.Background(), b, "hashicorp/random@3.7.2")
	if e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(t.TempDir(), "package.zip")
	if e = os.WriteFile(p, b.Archive(), 0o600); e != nil {
		t.Fatal(e)
	}
	want, e := dirhash.HashZip(p, dirhash.Hash1)
	if e != nil || want != contents.H1 || contents.Executable != exe || contents.ExecutableDigest != sha256Hex(elf) {
		t.Fatalf("hash/ELF binding mismatch: %+v %v", contents, e)
	}
	for _, c := range []struct {
		name   string
		names  []string
		modes  []os.FileMode
		bodies [][]byte
	}{
		{"traversal", []string{"../" + exe}, []os.FileMode{0o755}, [][]byte{elf}},
		{"symlink", []string{exe}, []os.FileMode{os.ModeSymlink | 0o755}, [][]byte{elf}},
		{"setuid", []string{exe}, []os.FileMode{os.ModeSetuid | 0o755}, [][]byte{elf}},
		{"wrong-version", []string{"terraform-provider-random_v3.7.1_x5"}, []os.FileMode{0o755}, [][]byte{elf}},
		{"duplicate-executable", []string{exe, "terraform-provider-random_v3.7.2_x6"}, []os.FileMode{0o755, 0o755}, [][]byte{elf, elf}},
		{"case-alias", []string{exe, exe}, []os.FileMode{0o755, 0o755}, [][]byte{elf, elf}},
		{"unmarked-ELF", []string{exe, "readme"}, []os.FileMode{0o755, 0o644}, [][]byte{elf, elf}},
		{"script", []string{exe, "install.sh"}, []os.FileMode{0o755, 0o755}, [][]byte{elf, []byte("#!/bin/sh\n")}},
		{"wrong-platform", []string{exe}, []os.FileMode{0o755}, [][]byte{append(append([]byte(nil), elf[:18]...), append([]byte{183, 0}, elf[20:]...)...)}},
		{"truncated", []string{exe}, []os.FileMode{0o755}, [][]byte{elf[:65]}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, e := InspectPackage(context.Background(), packageFixture(t, c.names, c.modes, c.bodies), "hashicorp/random@3.7.2"); e == nil {
				t.Fatal("unsafe package accepted")
			}
		})
	}
	bad := b.Archive()
	at := bytes.Index(bad, []byte("license\n"))
	bad[at] = 'X'
	b.parts[5] = bad
	if _, e := InspectPackage(context.Background(), b, "hashicorp/random@3.7.2"); e == nil {
		t.Fatal("CRC mismatch accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := InspectPackage(ctx, b, "hashicorp/random@3.7.2"); e == nil {
		t.Fatal("cancelled inspection accepted")
	}
}

func TestEmptyProviderBundleFailsWithoutPanic(t *testing.T) {
	if _, e := InspectPackage(context.Background(), Bundle{}, "hashicorp/random@3.7.2"); e == nil {
		t.Fatal("empty bundle accepted")
	}
	var bundle Bundle
	if bundle.RegistryDigest() != "" || bundle.ArchiveDigest() != "" || len(bundle.Archive()) != 0 || len(bundle.Signature()) != 0 || len(bundle.Checksums()) != 0 {
		t.Fatal("zero bundle claimed content")
	}
	if _, e := bundle.ZipReader(); e == nil {
		t.Fatal("zero bundle ZIP accepted")
	}
}
