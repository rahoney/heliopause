package releaseprovenance

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

type bunZIPTestEntry struct {
	header zip.FileHeader
	body   []byte
}

// Structural data only; these fake receipts do not exercise publisher trust.
// Real vendor checksums/bytes are checked by the separate offline command.
func bunZIPFixture(t *testing.T, entries []bunZIPTestEntry) (BunChecksumReceipt, []byte) {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, entry := range entries {
		h := entry.header
		f, err := w.CreateHeader(&h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	body := out.Bytes()
	return BunChecksumReceipt{archive: uvDigest(body), document: "synthetic structural only", signer: "synthetic structural only"}, body
}

func bunZIPEntries() []bunZIPTestEntry {
	dir := zip.FileHeader{Name: "bun-linux-x64/", Method: zip.Store}
	dir.SetMode(os.ModeDir | 0o755)
	file := zip.FileHeader{Name: "bun-linux-x64/bun", Method: zip.Store}
	file.SetMode(0o755)
	return []bunZIPTestEntry{{dir, nil}, {file, syntheticPNPMELF()}}
}

func TestBunNativeZIPStructuralBindingPreservesPNPMELFRules(t *testing.T) {
	r, body := bunZIPFixture(t, bunZIPEntries())
	tool, native, err := r.ReadToolArchive(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(native, syntheticPNPMELF()) || tool.SHA256() != uvDigest(native) || tool.Size() != int64(len(native)) || tool.ArchiveSHA256() != uvDigest(body) || len(tool.Libraries()) != 1 || tool.Libraries()[0] != "libc.so.6" {
		t.Fatal("native data binding differs")
	}
	libraries := tool.Libraries()
	libraries[0] = "changed"
	if tool.Libraries()[0] != "libc.so.6" {
		t.Fatal("native data receipt leaked mutable libraries")
	}
	if _, native, err := (BunChecksumReceipt{}).ReadToolArchive(context.Background(), body); err == nil || native != nil {
		t.Fatal("zero receipt exposed native bytes")
	}
	changed := bytes.Clone(body)
	changed[len(changed)-1] ^= 1
	if _, native, err := r.ReadToolArchive(context.Background(), changed); err == nil || native != nil {
		t.Fatal("changed authenticated transport exposed native bytes")
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("cancel Bun native read")
	cancel(cause)
	if _, native, err := r.ReadToolArchive(ctx, body); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || native != nil {
		t.Fatal("native cancellation lost cause or exposed data", err)
	}
}

func TestBunNativeZIPRejectsUnsafeAndIncompleteData(t *testing.T) {
	for _, change := range []string{"missing", "duplicate", "extra", "path", "file-mode", "directory-mode", "symlink", "unknown-extra", "bad-elf", "wrong-cpu", "wx-load", "wrong-loader", "crc", "declared-expansion", "encrypted"} {
		t.Run(change, func(t *testing.T) {
			entries := bunZIPEntries()
			switch change {
			case "missing":
				entries = entries[:1]
			case "duplicate":
				entries[0] = entries[1]
			case "extra":
				entries = append(entries, entries[1])
			case "path":
				entries[1].header.Name = "../outside"
			case "file-mode":
				entries[1].header.SetMode(0o777)
			case "directory-mode":
				entries[0].header.SetMode(os.ModeDir | 0o777)
			case "symlink":
				entries[1].header.SetMode(os.ModeSymlink | 0o777)
			case "unknown-extra":
				entries[1].header.Extra = []byte{0xfe, 0xca, 0, 0}
			case "bad-elf":
				entries[1].body = []byte("not ELF")
			case "wrong-cpu":
				binary.LittleEndian.PutUint16(entries[1].body[18:], 183)
			case "wx-load":
				binary.LittleEndian.PutUint32(entries[1].body[68:], 7)
			case "wrong-loader":
				entries[1].body = bytes.Replace(entries[1].body, []byte("ld-linux"), []byte("ld-other"), 1)
			}
			_, body := bunZIPFixture(t, entries)
			if change == "crc" {
				at := bytes.Index(body, []byte{0x7f, 'E', 'L', 'F'})
				if at < 0 {
					t.Fatal("fixture data absent")
				}
				body[at+24] ^= 1
			}
			if change == "declared-expansion" || change == "encrypted" {
				pos := 0
				found := false
				for {
					at := bytes.Index(body[pos:], []byte{'P', 'K', 1, 2})
					if at < 0 {
						break
					}
					at += pos
					nameLength := int(binary.LittleEndian.Uint16(body[at+28:]))
					name := string(body[at+46 : at+46+nameLength])
					if name == "bun-linux-x64/bun" {
						if change == "declared-expansion" {
							binary.LittleEndian.PutUint32(body[at+24:], maxBunNativeBytes+1)
						} else {
							binary.LittleEndian.PutUint16(body[at+8:], 1)
						}
						found = true
						break
					}
					pos = at + 46 + nameLength
				}
				if !found {
					t.Fatal("fixture central entry absent")
				}
			}
			if native, _, err := readBunNativeZIP(context.Background(), body); err == nil || native != nil {
				t.Fatal("unsafe Bun ZIP exposed native bytes")
			}
		})
	}
}

func TestBunZIPMetadataAndCancelledReadAreBounded(t *testing.T) {
	for _, extra := range [][]byte{{1}, {0x55, 0x54, 1, 0, 8}, {0x75, 0x78, 5, 0, 1, 4, 0, 0, 0}, {0x55, 0x54, 1, 0, 0, 0x55, 0x54, 1, 0, 0}} {
		if bunZIPExtra(extra) {
			t.Fatal("malformed/duplicate ZIP metadata accepted")
		}
	}
	if !bunZIPExtra([]byte{0x55, 0x54, 5, 0, 1, 0, 0, 0, 0}) {
		t.Fatal("bounded standard mtime rejected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (bunZIPContextReader{ctx, strings.NewReader("data")}).Read(make([]byte, 8)); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled expansion continued", err)
	}
	if _, _, err := readBunNativeZIP(context.Background(), nil); err == nil {
		t.Fatal("empty transport accepted")
	}
	var _ io.Reader = bunZIPContextReader{}
}

func TestBunNativeZIPRejectsUnsignedTrailer(t *testing.T) {
	_, body := bunZIPFixture(t, bunZIPEntries())
	body = append(body, []byte("unknown trailing transport")...)
	if native, _, err := readBunNativeZIP(context.Background(), body); err == nil || native != nil {
		t.Fatal("Bun structural ZIP reader accepted unaccounted trailing bytes")
	}
}
