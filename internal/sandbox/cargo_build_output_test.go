package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestCargoBuildOutputMetadataDiagnosticPreservesRejection(t *testing.T) {
	var input, output bytes.Buffer
	w := tar.NewWriter(&input)
	header := &tar.Header{Name: "artifact-controlled-secret-name", Linkname: "artifact-controlled-link-target", Typeflag: tar.TypeLink, Mode: 0o700, Uid: 1000, Gid: 1000, Format: tar.FormatUSTAR}
	if err := w.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	err := filterCargoBuildOutputs(context.Background(), &input, &output)
	var boundary *cargoOutputArchiveFailure
	if !errors.As(err, &boundary) || boundary.kind != tar.TypeLink || !boundary.link || output.Len() != 0 {
		t.Fatalf("diagnostic changed rejected archive: %v", err)
	}
	text := err.Error()
	if len(text) > 256 || strings.Contains(text, "artifact-controlled") || !strings.Contains(text, "ARCHIVE_BOUNDARY member=1 kind=49 uid=1000 gid=1000 mode=700 link=true pax=0") {
		t.Fatalf("unbounded or raw diagnostic: %q", text)
	}
}

func cargoOutputTestArchive(t *testing.T, headers []*tar.Header, bodies []string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for i, h := range headers {
		if err := writer.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(bodies[i])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestCargoBuildOutputRootHeaderIsDataOnly(t *testing.T) {
	for _, name := range []string{".", "./"} {
		t.Run(name, func(t *testing.T) {
			headers := []*tar.Header{{Name: name, Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}, {Name: "product", Typeflag: tar.TypeReg, Mode: 0o700, Uid: 1000, Gid: 1000, Size: 4}}
			var output bytes.Buffer
			if err := filterCargoBuildOutputs(context.Background(), bytes.NewReader(cargoOutputTestArchive(t, headers, []string{"", "ELF!"})), &output); err != nil {
				t.Fatal(err)
			}
			r := tar.NewReader(&output)
			h, err := r.Next()
			if err != nil || h.Name != "product" || h.Typeflag != tar.TypeReg || h.Mode != 0o500 || h.Uid != 1000 || h.Gid != 1000 || h.Size != 4 {
				t.Fatal("output data or metadata binding changed")
			}
		})
	}
	for _, bad := range []string{"regular-root", "duplicate-root", "late-root", "root-owner", "root-write", "root-traversal"} {
		t.Run(bad, func(t *testing.T) {
			root := &tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}
			product := &tar.Header{Name: "product", Typeflag: tar.TypeReg, Mode: 0o700, Uid: 1000, Gid: 1000, Size: 4}
			headers := []*tar.Header{root, product}
			bodies := []string{"", "ELF!"}
			switch bad {
			case "regular-root":
				root.Name = "."
				root.Typeflag = tar.TypeReg
			case "duplicate-root":
				headers = append(headers, root)
				bodies = append(bodies, "")
			case "late-root":
				root.Name = "."
				headers = []*tar.Header{product, root}
				bodies = []string{"ELF!", ""}
			case "root-owner":
				root.Uid = 0
			case "root-write":
				root.Mode = 0o777
			case "root-traversal":
				root.Name = "../"
			}
			var output bytes.Buffer
			if err := filterCargoBuildOutputs(context.Background(), bytes.NewReader(cargoOutputTestArchive(t, headers, bodies)), &output); err == nil {
				t.Fatal("unsafe root archive accepted")
			}
		})
	}
}

func TestCargoBuildOutputArchiveAliasesAreReconstructedAsRegularData(t *testing.T) {
	for _, forward := range []bool{false, true} {
		t.Run(map[bool]string{false: "prior-target", true: "forward-target"}[forward], func(t *testing.T) {
			root := &tar.Header{Name: "./", Typeflag: tar.TypeDir, Uid: 1000, Gid: 1000, Mode: 0o755}
			dir := &tar.Header{Name: "deps/", Typeflag: tar.TypeDir, Uid: 1000, Gid: 1000, Mode: 0o755}
			target := &tar.Header{Name: "deps/program-hash", Typeflag: tar.TypeReg, Uid: 1000, Gid: 1000, Mode: 0o755, Size: 4}
			alias := &tar.Header{Name: "program", Typeflag: tar.TypeLink, Linkname: "./deps/program-hash", Uid: 1000, Gid: 1000, Mode: 0o755}
			headers := []*tar.Header{root, dir, target, alias}
			bodies := []string{"", "", "ELF!", ""}
			if forward {
				headers = []*tar.Header{root, dir, alias, target}
				bodies = []string{"", "", "", "ELF!"}
			}
			var output bytes.Buffer
			if err := filterCargoBuildOutputs(context.Background(), bytes.NewReader(cargoOutputTestArchive(t, headers, bodies)), &output); err != nil {
				t.Fatal(err)
			}
			reader := tar.NewReader(&output)
			h, err := reader.Next()
			if err != nil || h.Name != "program" || h.Typeflag != tar.TypeReg || h.Linkname != "" || h.Size != 4 || h.Mode != 0o500 {
				t.Fatal("archive link reached host output")
			}
			data := make([]byte, 4)
			if _, err := io.ReadFull(reader, data); err != nil || string(data) != "ELF!" {
				t.Fatal("linked output bytes differ")
			}
		})
	}
	for _, bad := range []string{"missing", "escape", "absolute", "symlink", "directory", "chain", "owner", "mode", "duplicate", "unselected-escape", "regular-linkname"} {
		t.Run(bad, func(t *testing.T) {
			target := &tar.Header{Name: "deps/program-hash", Typeflag: tar.TypeReg, Uid: 1000, Gid: 1000, Mode: 0o755, Size: 4}
			alias := &tar.Header{Name: "program", Typeflag: tar.TypeLink, Linkname: "deps/program-hash", Uid: 1000, Gid: 1000, Mode: 0o755}
			headers := []*tar.Header{target, alias}
			bodies := []string{"ELF!", ""}
			switch bad {
			case "missing":
				alias.Linkname = "deps/missing"
			case "escape":
				alias.Linkname = "../outside"
			case "absolute":
				alias.Linkname = "/etc/shadow"
			case "symlink":
				target.Typeflag = tar.TypeSymlink
				target.Linkname = "/etc/shadow"
				target.Size = 0
				bodies[0] = ""
			case "directory":
				target.Typeflag = tar.TypeDir
				target.Size = 0
				bodies[0] = ""
			case "chain":
				target.Typeflag = tar.TypeLink
				target.Linkname = "elsewhere"
				target.Size = 0
				bodies[0] = ""
			case "owner":
				alias.Uid = 0
			case "mode":
				alias.Mode = 0o700
			case "duplicate":
				alias.Name = target.Name
			case "unselected-escape":
				alias.Name = "deps/unselected"
				alias.Linkname = "../outside"
			case "regular-linkname":
				target.Linkname = "outside"
			}
			var output bytes.Buffer
			if err := filterCargoBuildOutputs(context.Background(), bytes.NewReader(cargoOutputTestArchive(t, headers, bodies)), &output); err == nil {
				t.Fatal("unsafe alias archive accepted")
			}
		})
	}
}

func TestCargoBuildOutputAliasExpansionIsBoundedBeforeEmission(t *testing.T) {
	body := strings.Repeat("x", 8<<20)
	headers := []*tar.Header{{Name: "deps/backing", Typeflag: tar.TypeReg, Uid: 1000, Gid: 1000, Mode: 0o755, Size: int64(len(body))}}
	bodies := []string{body}
	for i := 0; i < 26; i++ {
		headers = append(headers, &tar.Header{Name: fmt.Sprintf("product%d", i), Typeflag: tar.TypeLink, Linkname: "deps/backing", Uid: 1000, Gid: 1000, Mode: 0o755})
		bodies = append(bodies, "")
	}
	archive := cargoOutputTestArchive(t, headers, bodies)
	for attempt := 0; attempt < 3; attempt++ {
		var output bytes.Buffer
		err := filterCargoBuildOutputs(context.Background(), bytes.NewReader(archive), &output)
		var boundary *cargoOutputArchiveFailure
		if !errors.As(err, &boundary) || boundary.member != 26 || output.Len() != 0 {
			t.Fatalf("alias amplification/first failure changed: %v", err)
		}
	}
}
