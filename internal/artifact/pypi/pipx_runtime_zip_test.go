package pypi

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func runtimeZIPFixture(t *testing.T, headers []zip.FileHeader) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, header := range headers {
		member, err := writer.CreateHeader(&header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write([]byte("original member\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func runtimeZIPHeader(name string) zip.FileHeader {
	header := zip.FileHeader{Name: name, Method: zip.Store, Modified: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)}
	header.SetMode(0444)
	return header
}

func TestPipxRuntimeZIPMemberFactsAndRefusals(t *testing.T) {
	for _, mode := range []string{"valid", "duplicate", "unsorted", "traversal", "absolute", "backslash", "nul", "collision", "symlink", "writable", "setuid", "compressed", "unsupported-method", "expanded", "timestamp", "crc", "truncated", "empty", "bound"} {
		t.Run(mode, func(t *testing.T) {
			headers := []zip.FileHeader{runtimeZIPHeader("a.py"), runtimeZIPHeader("b/c.py")}
			switch mode {
			case "duplicate":
				headers[1].Name = headers[0].Name
			case "unsorted":
				headers[0], headers[1] = headers[1], headers[0]
			case "traversal":
				headers[0].Name = "../a.py"
			case "absolute":
				headers[0].Name = "/a.py"
			case "backslash":
				headers[0].Name = "a\\b.py"
			case "nul":
				headers[0].Name = "a\x00.py"
			case "collision":
				headers[1].Name = "a.py/b.py"
			case "symlink":
				headers[0].SetMode(os.ModeSymlink | 0444)
			case "writable":
				headers[0].SetMode(0644)
			case "setuid":
				headers[0].SetMode(os.ModeSetuid | 0444)
			case "compressed":
				headers[0].Method = zip.Deflate
			case "timestamp":
				headers[0].Modified = headers[0].Modified.Add(time.Second * 2)
			}
			body := runtimeZIPFixture(t, headers)
			switch mode {
			case "unsupported-method":
				index := bytes.Index(body, []byte{'P', 'K', 1, 2})
				if index < 0 {
					t.Fatal("central header missing")
				}
				binary.LittleEndian.PutUint16(body[index+10:], 99)
			case "expanded":
				index := bytes.Index(body, []byte{'P', 'K', 1, 2})
				if index < 0 {
					t.Fatal("central header missing")
				}
				binary.LittleEndian.PutUint32(body[index+24:], uint32(PublicPyPIProfile().ResourcePolicy().MaxGraphUncompressed()+1))
			case "crc":
				at := bytes.Index(body, []byte("original member"))
				if at < 0 {
					t.Fatal("fixture body missing")
				}
				body[at] ^= 1
			case "truncated":
				body = body[:len(body)-1]
			case "empty":
				body = nil
			case "bound":
				body = make([]byte, pipxSupplyZIPLimit+1)
			}
			facts, err := pipxRuntimeZIPFiles(body)
			if mode != "valid" && mode != "compressed" {
				if err == nil || facts != nil {
					t.Fatal("invalid ZIP accepted")
				}
				return
			}
			if err != nil || len(facts) != 2 || facts[0].Path != "a.py" || facts[0].Size != 16 || facts[0].SHA256 != pipxSupplyDigest([]byte("original member\n")) {
				t.Fatal("wrong original facts", facts, err)
			}
		})
	}
}

func TestPipxRuntimeZIPRequiresSelectedAsset(t *testing.T) {
	if (PipxRuntimeZIP{}).Valid() {
		t.Fatal("zero value accepted")
	}
	for _, body := range [][]byte{nil, make([]byte, runtimeidentity.PipxToolSize-1), make([]byte, runtimeidentity.PipxToolSize), make([]byte, runtimeidentity.PipxToolSize+1)} {
		value, err := ParsePipxRuntimeZIP(body)
		if err == nil || value.Valid() || len(value.Bytes()) != 0 || len(value.Files()) != 0 {
			t.Fatal("unselected asset accepted")
		}
	}
}
