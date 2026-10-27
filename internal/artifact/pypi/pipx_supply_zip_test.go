package pypi

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func TestPipxSupplyZIPPreservesOriginalBytesDeterministically(t *testing.T) {
	packages, inputs, pages := pipxSupplyFixture(t, "click>=1.0.0", pipxSupplyEntry, wheelTestEntry{name: "pipx-1.0.0.data/data/share/man/man1/pipx.1", body: []byte("manual\n")})
	closure, err := BindPipxSupplyClosure(packages, inputs, pages)
	if err != nil {
		t.Fatal(err)
	}
	body, err := closure.ZIP(inputs)
	if err != nil {
		t.Fatal(err)
	}
	reversed := []UVWheelInput{inputs[1], inputs[0]}
	again, err := closure.ZIP(reversed)
	if err != nil || !bytes.Equal(body, again) {
		t.Fatal("input order changed candidate bytes", err)
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != len(closure.Files()) {
		t.Fatal("candidate file set differs")
	}
	for i, member := range archive.File {
		fact := closure.Files()[i]
		reader, err := member.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil || closeErr != nil || member.Name != fact.Path || int64(len(data)) != fact.Size || pipxSupplyDigest(data) != fact.SHA256 {
			t.Fatal("original member changed", fact.Path)
		}
		if member.Method != zip.Store || !member.Mode().IsRegular() || member.Mode().Perm() != 0444 || member.Modified.Year() != 1980 {
			t.Fatal("ZIP metadata is not canonical")
		}
	}
}

func TestPipxSupplyZIPRefusesChangedInputsAndUnsupportedLayout(t *testing.T) {
	for _, mode := range []string{"zero", "missing", "extra", "duplicate", "url", "size", "nil", "archive-bytes", "member-digest", "missing-member", "relocation", "file-collision", "expansion"} {
		t.Run(mode, func(t *testing.T) {
			packages, inputs, pages := pipxSupplyFixture(t, "click>=1.0.0", pipxSupplyEntry)
			closure, err := BindPipxSupplyClosure(packages, inputs, pages)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "zero":
				closure = PipxSupplyClosure{}
			case "missing":
				inputs = inputs[:1]
			case "extra":
				inputs = append(inputs, inputs[0])
			case "duplicate":
				inputs[1] = inputs[0]
			case "url":
				inputs[0].URL += "different"
			case "size":
				inputs[0].Size--
			case "nil":
				inputs[0].Reader = nil
			case "archive-bytes":
				body, err := io.ReadAll(io.NewSectionReader(inputs[0].Reader, 0, inputs[0].Size))
				if err != nil {
					t.Fatal(err)
				}
				body[len(body)-1] ^= 1
				inputs[0].Reader = bytes.NewReader(body)
			case "member-digest":
				closure.files[0].SHA256 = "changed"
			case "missing-member":
				closure.files = closure.files[1:]
			case "relocation":
				closure.files[0].Destination = "relocated/path"
			case "file-collision":
				closure.files[1].Path = closure.files[0].Path
			case "expansion":
				closure.files[0].Size = pipxSupplyZIPLimit + 1
			}
			if body, err := closure.ZIP(inputs); err == nil || body != nil {
				t.Fatal("unsupported/changed supply produced candidate")
			}
		})
	}
}
