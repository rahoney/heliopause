package pypi

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"testing"
)

func TestInspectWheelValidatesIntegrityAndStaticSurface(t *testing.T) {
	data := []byte("print('safe')\n")
	sum := sha256.Sum256(data)
	metadata := []byte("Metadata-Version: 2.4\nName: packaging\nVersion: 25.0\nRequires-Python: >=3.8\nImport-Name: packaging\n")
	wheel := []byte("Wheel-Version: 1.0\nGenerator: test\nRoot-Is-Purelib: true\nTag: cp314-cp314-manylinux_2_36_x86_64\n")
	metadataSum := sha256.Sum256(metadata)
	wheelSum := sha256.Sum256(wheel)
	record := "pkg/__init__.py,sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]) + ",14\n" +
		"packaging-25.0.dist-info/METADATA,sha256=" + base64.RawURLEncoding.EncodeToString(metadataSum[:]) + "," + itoa(len(metadata)) + "\n" +
		"packaging-25.0.dist-info/WHEEL,sha256=" + base64.RawURLEncoding.EncodeToString(wheelSum[:]) + "," + itoa(len(wheel)) + "\n" +
		"packaging-25.0.dist-info/RECORD,,\n"
	archive := makeWheel(t, map[string][]byte{
		"pkg/__init__.py":                   data,
		"packaging-25.0.dist-info/METADATA": metadata,
		"packaging-25.0.dist-info/WHEEL":    wheel,
		"packaging-25.0.dist-info/RECORD":   []byte(record),
	})
	digest := sha256.Sum256(archive)
	inspection, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "packaging-25.0-cp314-cp314-manylinux_2_36_x86_64.whl", hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Project != "packaging" || inspection.Version != "25.0" || len(inspection.Files) != 4 || inspection.ObservedSHA256 == "" {
		t.Fatalf("inspection = %#v", inspection)
	}
}

func TestInspectWheelDerivesImportNameWhenMetadataOmitsIt(t *testing.T) {
	data := []byte("value = 1\n")
	sum := sha256.Sum256(data)
	metadata := []byte("Metadata-Version: 2.4\nName: packaging\nVersion: 25.0\n")
	wheel := []byte("Wheel-Version: 1.0\nTag: cp314-cp314-manylinux_2_36_x86_64\n")
	metadataSum, wheelSum := sha256.Sum256(metadata), sha256.Sum256(wheel)
	record := "packaging/__init__.py,sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]) + ",10\n" +
		"packaging-25.0.dist-info/METADATA,sha256=" + base64.RawURLEncoding.EncodeToString(metadataSum[:]) + "," + itoa(len(metadata)) + "\n" +
		"packaging-25.0.dist-info/WHEEL,sha256=" + base64.RawURLEncoding.EncodeToString(wheelSum[:]) + "," + itoa(len(wheel)) + "\n" +
		"packaging-25.0.dist-info/RECORD,,\n"
	archive := makeWheel(t, map[string][]byte{"packaging/__init__.py": data, "packaging-25.0.dist-info/METADATA": metadata, "packaging-25.0.dist-info/WHEEL": wheel, "packaging-25.0.dist-info/RECORD": []byte(record)})
	digest := sha256.Sum256(archive)
	inspection, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "packaging-25.0-cp314-cp314-manylinux_2_36_x86_64.whl", hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err != nil || len(inspection.ImportNames) != 1 || inspection.ImportNames[0] != "packaging" {
		t.Fatalf("fallback imports=%q error=%v", inspection.ImportNames, err)
	}
}

func itoa(value int) string { return fmt.Sprintf("%d", value) }

func TestInspectWheelRejectsUnsafeAndMismatchedInputs(t *testing.T) {
	data := makeWheel(t, map[string][]byte{"../escape": []byte("bad")})
	digest := sha256.Sum256(data)
	if _, err := InspectWheel(bytes.NewReader(data), int64(len(data)), "x-1.0-py3-none-any.whl", hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits()); err == nil {
		t.Fatal("unsafe wheel accepted")
	}
}

func TestInspectWheelAcceptsNormalizedEquivalentIdentity(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		metadataName string
		contentPath  string
		distInfo     string
	}{
		{name: "Jinja2 and jinja2", filename: "jinja2-1.0-cp314-cp314-manylinux_2_36_x86_64.whl", metadataName: "Jinja2", contentPath: "jinja2/__init__.py", distInfo: "jinja2-1.0.dist-info"},
		{name: "typing_extensions and typing-extensions", filename: "typing-extensions-1.0-cp314-cp314-manylinux_2_36_x86_64.whl", metadataName: "typing_extensions", contentPath: "typing_extensions/__init__.py", distInfo: "typing_extensions-1.0.dist-info"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := validWheelArchive(t, test.metadataName, "1.0", test.contentPath, test.distInfo)
			inspection := inspectTestWheel(t, archive, test.filename)
			if inspection.Project == "" || inspection.Version != "1.0" {
				t.Fatalf("inspection identity = %#v", inspection)
			}
		})
	}
}

func TestInspectWheelAcceptsSafeExplicitDirectories(t *testing.T) {
	archive := validWheelArchive(t, "packaging", "25.0", "packaging/__init__.py", "packaging-25.0.dist-info",
		wheelTestEntry{name: "packaging/", mode: os.ModeDir | 0o755},
		wheelTestEntry{name: "packaging-25.0.dist-info/", mode: os.ModeDir | 0o755},
		wheelTestEntry{name: "packaging-25.0.dist-info/licenses/", mode: os.ModeDir | 0o755},
	)
	inspection := inspectTestWheel(t, archive, "packaging-25.0-cp314-cp314-manylinux_2_36_x86_64.whl")
	if len(inspection.Files) != 4 {
		t.Fatalf("inspection files = %d, want 4 regular files", len(inspection.Files))
	}
}

func TestInspectWheelIgnoresNestedDistributionMetadata(t *testing.T) {
	archive := validWheelArchive(t, "setuptools", "84.0", "setuptools/__init__.py", "setuptools-84.0.dist-info",
		wheelTestEntry{name: "setuptools/_vendor/example-1.0.dist-info/METADATA", body: []byte("nested metadata")},
		wheelTestEntry{name: "setuptools/_vendor/example-1.0.dist-info/WHEEL", body: []byte("nested wheel")},
		wheelTestEntry{name: "setuptools/_vendor/example-1.0.dist-info/RECORD", body: []byte("nested record")},
	)
	inspection := inspectTestWheel(t, archive, "setuptools-84.0-cp314-cp314-manylinux_2_36_x86_64.whl")
	if len(inspection.Files) != 7 {
		t.Fatalf("inspection files = %d, want 7 regular files", len(inspection.Files))
	}
}

func TestInspectWheelRejectsIdentityAndUnsafeEntries(t *testing.T) {
	tests := []struct {
		name      string
		metadata  string
		version   string
		distInfo  string
		mutate    func([]wheelTestEntry) []wheelTestEntry
		wantStage WheelValidationStage
	}{
		{name: "different project after normalization", metadata: "other", version: "1.0", distInfo: "packaging-1.0.dist-info", wantStage: WheelValidationMetadataIdentity},
		{name: "different version", metadata: "packaging", version: "2.0", distInfo: "packaging-1.0.dist-info", wantStage: WheelValidationMetadataIdentity},
		{name: "wrong primary dist-info", metadata: "packaging", version: "1.0", distInfo: "other-1.0.dist-info", wantStage: WheelValidationMetadataInvalid},
		{name: "path traversal directory", metadata: "packaging", version: "1.0", distInfo: "packaging-1.0.dist-info", mutate: func(entries []wheelTestEntry) []wheelTestEntry {
			return append(entries, wheelTestEntry{name: "../escape/", mode: os.ModeDir | 0o755})
		}, wantStage: WheelValidationFileType},
		{name: "symlink", metadata: "packaging", version: "1.0", distInfo: "packaging-1.0.dist-info", mutate: func(entries []wheelTestEntry) []wheelTestEntry {
			return append(entries, wheelTestEntry{name: "link", body: []byte("target"), mode: os.ModeSymlink | 0o777})
		}, wantStage: WheelValidationFileType},
		{name: "file-directory collision", metadata: "packaging", version: "1.0", distInfo: "packaging-1.0.dist-info", mutate: func(entries []wheelTestEntry) []wheelTestEntry {
			return append(entries, wheelTestEntry{name: "collision", body: []byte("x")}, wheelTestEntry{name: "collision/", mode: os.ModeDir | 0o755})
		}, wantStage: WheelValidationFileType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := validWheelEntries(t, test.metadata, test.version, "packaging/__init__.py", test.distInfo)
			if test.mutate != nil {
				entries = test.mutate(entries)
			}
			archive := makeWheelEntries(t, entries...)
			filename := "packaging-1.0-cp314-cp314-manylinux_2_36_x86_64.whl"
			digest := sha256.Sum256(archive)
			_, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), filename, hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
			if err == nil {
				t.Fatal("unsafe or mismatched wheel accepted")
			}
			stage, ok := WheelValidationStageOf(err)
			if !ok || stage != test.wantStage {
				t.Fatalf("WheelValidationStageOf() = %q, %v; want %q", stage, ok, test.wantStage)
			}
		})
	}
}

func TestWheelEntryPathRejectsInvalidDirectories(t *testing.T) {
	tests := []struct {
		name string
		file zip.File
	}{
		{name: "nonzero directory", file: zip.File{FileHeader: zip.FileHeader{Name: "nonzero/", UncompressedSize64: 1}}},
		{name: "directory without trailing slash", file: zip.File{FileHeader: zip.FileHeader{Name: "invalid"}}},
	}
	tests[0].file.FileHeader.SetMode(os.ModeDir | 0o755)
	tests[1].file.FileHeader.SetMode(os.ModeDir | 0o755)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, ok := wheelEntryPath(&test.file); ok {
				t.Fatal("invalid directory accepted")
			}
		})
	}
}

func TestInspectWheelRejectsMissingOrDuplicatePrimaryMetadata(t *testing.T) {
	for _, kind := range []string{"METADATA", "WHEEL", "RECORD"} {
		for _, operation := range []string{"missing", "duplicate"} {
			t.Run(operation+" "+kind, func(t *testing.T) {
				name := "packaging-1.0.dist-info/" + kind
				entries := validWheelEntries(t, "packaging", "1.0", "packaging/__init__.py", "packaging-1.0.dist-info")
				if operation == "missing" {
					entries = removeWheelEntry(entries, name)
				} else {
					for _, entry := range entries {
						if entry.name == name {
							entries = append(entries, entry)
							break
						}
					}
				}
				archive := makeWheelEntries(t, entries...)
				digest := sha256.Sum256(archive)
				_, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "packaging-1.0-cp314-cp314-manylinux_2_36_x86_64.whl", hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
				if err == nil {
					t.Fatal("invalid primary metadata accepted")
				}
			})
		}
	}
}

func TestWheelTagsCompatibleManylinuxSemantics(t *testing.T) {
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	tests := []struct {
		name     string
		platform string
		target   WheelTarget
		want     bool
	}{
		{name: "exact", platform: "manylinux_2_36_x86_64", target: target, want: true},
		{name: "older glibc baseline", platform: "manylinux_2_28_x86_64", target: target, want: true},
		{name: "newer wheel baseline", platform: "manylinux_2_36_x86_64", target: WheelTarget{"cp314", "cp314", "manylinux_2_28_x86_64"}, want: false},
		{name: "architecture mismatch", platform: "manylinux_2_28_aarch64", target: target, want: false},
		{name: "malformed", platform: "manylinux_x86_64", target: target, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := wheelTagsCompatible([]string{"cp314"}, []string{"cp314"}, []string{test.platform}, test.target); got != test.want {
				t.Fatalf("wheelTagsCompatible(%q, %q) = %v, want %v", test.platform, test.target.Platform, got, test.want)
			}
		})
	}
}

func FuzzInspectWheelNoPanic(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Fuzz(func(t *testing.T, body []byte) {
		digest := sha256.Sum256(body)
		_, _ = InspectWheel(bytes.NewReader(body), int64(len(body)), "x-1.0-py3-none-any.whl", hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	})
}

func makeWheel(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]wheelTestEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, wheelTestEntry{name: name, body: files[name]})
	}
	return makeWheelEntries(t, entries...)
}

type wheelTestEntry struct {
	name string
	body []byte
	mode os.FileMode
}

func makeWheelEntries(t *testing.T, entries ...wheelTestEntry) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, testEntry := range entries {
		header := &zip.FileHeader{Name: testEntry.name, Method: zip.Store}
		if testEntry.mode == 0 {
			header.SetMode(0o644)
		} else {
			header.SetMode(testEntry.mode)
		}
		entry, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(testEntry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func validWheelEntries(t *testing.T, metadataName, metadataVersion, contentPath, distInfo string, extras ...wheelTestEntry) []wheelTestEntry {
	t.Helper()
	data := []byte("print('safe')\n")
	metadata := []byte("Metadata-Version: 2.4\nName: " + metadataName + "\nVersion: " + metadataVersion + "\n")
	wheel := []byte("Wheel-Version: 1.0\nTag: cp314-cp314-manylinux_2_36_x86_64\n")
	dataSum, metadataSum, wheelSum := sha256.Sum256(data), sha256.Sum256(metadata), sha256.Sum256(wheel)
	regular := append([]wheelTestEntry{{name: contentPath, body: data}}, extras...)
	record := contentPath + ",sha256=" + base64.RawURLEncoding.EncodeToString(dataSum[:]) + "," + itoa(len(data)) + "\n"
	for _, entry := range extras {
		if entry.mode != 0 {
			continue
		}
		sum := sha256.Sum256(entry.body)
		record += entry.name + ",sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]) + "," + itoa(len(entry.body)) + "\n"
	}
	record +=
		distInfo + "/METADATA,sha256=" + base64.RawURLEncoding.EncodeToString(metadataSum[:]) + "," + itoa(len(metadata)) + "\n" +
			distInfo + "/WHEEL,sha256=" + base64.RawURLEncoding.EncodeToString(wheelSum[:]) + "," + itoa(len(wheel)) + "\n" +
			distInfo + "/RECORD,,\n"
	return append(regular,
		wheelTestEntry{name: distInfo + "/METADATA", body: metadata},
		wheelTestEntry{name: distInfo + "/WHEEL", body: wheel},
		wheelTestEntry{name: distInfo + "/RECORD", body: []byte(record)},
	)
}

func validWheelArchive(t *testing.T, metadataName, version, contentPath, distInfo string, extras ...wheelTestEntry) []byte {
	t.Helper()
	entries := validWheelEntries(t, metadataName, version, contentPath, distInfo, extras...)
	return makeWheelEntries(t, entries...)
}

func inspectTestWheel(t *testing.T, archive []byte, filename string) WheelInspection {
	t.Helper()
	digest := sha256.Sum256(archive)
	inspection, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), filename, hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	return inspection
}

func removeWheelEntry(entries []wheelTestEntry, name string) []wheelTestEntry {
	filtered := make([]wheelTestEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.name != name {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
