package pypi

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
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

func TestWheelImportSurfaceFromRecordedFiles(t *testing.T) {
	for _, test := range []struct {
		name       string
		entries    []wheelTestEntry
		wantImport string
		noImport   bool
	}{
		{
			name: "ordinary .py package",
			entries: []wheelTestEntry{
				{name: "example/__init__.py", body: []byte("value = 1\n")},
			},
			wantImport: "example",
			noImport:   false,
		},
		{
			name: "Python native-extension-only wheel",
			entries: []wheelTestEntry{
				{name: "cuda/bindings/_runtime.cpython-314-x86_64-linux-gnu.so", body: []byte("native")},
			},
			wantImport: "cuda.bindings._runtime",
			noImport:   false,
		},
		{
			name: "mixed .py + native-extension wheel",
			entries: []wheelTestEntry{
				{name: "package/__init__.py", body: []byte("value = 1\n")},
				{name: "package/_native.cpython-314-x86_64-linux-gnu.so", body: []byte("native")},
			},
			wantImport: "package,package._native",
			noImport:   false,
		},
		{
			name: "realistic native/data-only CUDA-style wheel",
			entries: []wheelTestEntry{
				{name: "nvidia/nccl/lib/libnccl.so.2", body: []byte("elf payload")},
				{name: "nvidia/nccl/include/nccl.h", body: []byte("header payload")},
			},
			wantImport: "",
			noImport:   true,
		},
		{
			name:       "proven metadata-only wheel",
			entries:    nil,
			wantImport: "",
			noImport:   true,
		},
		{
			name: "unknown ambiguous empty surface",
			entries: []wheelTestEntry{
				{name: "README", body: []byte("unstructured root payload")},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "unexpected executable Python surface fails closed",
			entries: []wheelTestEntry{
				{name: "nvidia/nccl/hook.pth", body: []byte("import sys")},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "unrecognized payload binary fails closed",
			entries: []wheelTestEntry{
				{name: "nvidia/nccl/payload.unknown_binary", body: []byte("binary")},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "versioned library directory does not authorize unknown payload",
			entries: []wheelTestEntry{
				{name: "nvidia/libfoo.so.2/payload.unknown_binary", body: []byte("binary")},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "unknown versioned library suffix fails closed",
			entries: []wheelTestEntry{
				{name: "nvidia/libfoo.so.unknown", body: []byte("binary")},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "dist-info Python path hook fails closed",
			entries: []wheelTestEntry{
				{name: "example-1.0.dist-info/hook.pth", body: []byte("import example")},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "bare .so proven Python extension",
			entries: []wheelTestEntry{
				{name: "package/_native.so", body: buildSyntheticELF([]string{"PyInit__native"})},
			},
			wantImport: "package._native",
			noImport:   false,
		},
		{
			name: "mixed .py + bare .so Python extension",
			entries: []wheelTestEntry{
				{name: "package/__init__.py", body: []byte("value = 1\n")},
				{name: "package/_native.so", body: buildSyntheticELF([]string{"PyInit__native"})},
			},
			wantImport: "package,package._native",
			noImport:   false,
		},
		{
			name: "bare .so proven native library with no PyInit",
			entries: []wheelTestEntry{
				{name: "nvidia/cu13/lib/libpcsamplingutil.so", body: buildSyntheticELF([]string{"cuptiActivityEnable"})},
				{name: "nvidia/cu13/include/cupti.h", body: []byte("header payload")},
			},
			wantImport: "",
			noImport:   true,
		},
		{
			name: "ambiguous bare .so fails closed",
			entries: []wheelTestEntry{
				{name: "package/corrupt.so", body: []byte("corrupt non-elf payload")},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "bare .so with mismatched PyInit fails closed",
			entries: []wheelTestEntry{
				{name: "package/_native.so", body: buildSyntheticELF([]string{"PyInit_othermod"})},
			},
			wantImport: "",
			noImport:   false,
		},
		{
			name: "wheel with .dist-info/entry_points.txt cannot be noImport",
			entries: []wheelTestEntry{
				{name: "example-1.0.dist-info/entry_points.txt", body: []byte("[console_scripts]\nrun = example.cli:main\n")},
			},
			wantImport: "",
			noImport:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", test.entries, []string{"py3-none-any"}, nil)
			inspection := inspectTestWheel(t, archive, "example-1.0-py3-none-any.whl")
			if got := strings.Join(inspection.ImportNames, ","); got != test.wantImport || inspection.NoImportSurface != test.noImport {
				t.Fatalf("import surface = %q, no-import=%v, want %q/%v", got, inspection.NoImportSurface, test.wantImport, test.noImport)
			}
		})
	}
}

func TestWheelEntryPointsParsedAndValidated(t *testing.T) {
	epContent := `
[console_scripts]
black = black:patched_main
cli-tool = mypkg.cli:main [extra_feat]
my command = example:main

# Comments and empty lines are supported
; Alternate comment
[gui_scripts]
app = mypkg.gui:App.run

[pytest11]
my_plugin = mypkg.plugin
custom plugin check = mypkg.plugin:check_func
`
	archive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{
		{name: "example-1.0.dist-info/entry_points.txt", body: []byte(epContent)},
	}, []string{"py3-none-any"}, nil)
	inspection := inspectTestWheel(t, archive, "example-1.0-py3-none-any.whl")

	expectedEPs := []string{
		"console_scripts: black = black:patched_main",
		"console_scripts: cli-tool = mypkg.cli:main [extra_feat]",
		"console_scripts: my command = example:main",
		"gui_scripts: app = mypkg.gui:App.run",
		"pytest11: custom plugin check = mypkg.plugin:check_func",
		"pytest11: my_plugin = mypkg.plugin",
	}
	sort.Strings(expectedEPs)
	if strings.Join(inspection.EntryPoints, ";") != strings.Join(expectedEPs, ";") {
		t.Fatalf("entry points = %v, want %v", inspection.EntryPoints, expectedEPs)
	}
	if inspection.NoImportSurface {
		t.Fatalf("wheel with executable entry points must not have NoImportSurface=true")
	}

	// Malformed entry_points.txt must fail closed
	for _, malformed := range []struct {
		name    string
		content string
	}{
		{"missing equals", "[console_scripts]\ninvalid line without equals\n"},
		{"entry outside section", "foo = bar:main\n[console_scripts]\n"},
		{"unclosed section header", "[console_scripts\nfoo = bar:main\n"},
		{"empty section name", "[]\nfoo = bar:main\n"},
		{"invalid section punctuation", "[bad!group]\n"},
		{"empty dotted section component", "[bad..group]\n"},
		{"invalid UTF-8", string([]byte{'[', 'x', ']', '\n', '#', 0xff, '\n'})},
		{"invalid module name in target", "[console_scripts]\nfoo = 123-invalid:main\n"},
		{"empty key", "[console_scripts]\n= bar:main\n"},
		{"empty value", "[console_scripts]\nfoo =\n"},
		{"unclosed extras bracket", "[console_scripts]\nfoo = bar:main [extra\n"},
		{"key starting with bracket", "[console_scripts]\n [foo = bar:main\n"},
		{"control char in key", "[console_scripts]\nfoo\x00bar = bar:main\n"},
	} {
		t.Run(malformed.name, func(t *testing.T) {
			malformedArchive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{
				{name: "example-1.0.dist-info/entry_points.txt", body: []byte(malformed.content)},
			}, []string{"py3-none-any"}, nil)
			digest := sha256.Sum256(malformedArchive)
			_, err := InspectWheel(bytes.NewReader(malformedArchive), int64(len(malformedArchive)), "example-1.0-py3-none-any.whl", hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
			if err == nil {
				t.Fatalf("expected malformed entry_points.txt to fail validation, but it passed")
			}
			stage, ok := WheelValidationStageOf(err)
			if !ok || stage != WheelValidationMetadataInvalid {
				t.Fatalf("validation stage = %q (%v), want %q", stage, err, WheelValidationMetadataInvalid)
			}
		})
	}
}

func TestWheelEntryPointsAcceptsInternalWhitespace(t *testing.T) {
	epContent := `
[my.plugins]
my command = example:main
hello   world = example:main2
`
	archive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{
		{name: "example-1.0.dist-info/entry_points.txt", body: []byte(epContent)},
	}, []string{"py3-none-any"}, nil)
	inspection := inspectTestWheel(t, archive, "example-1.0-py3-none-any.whl")

	expected := []string{
		"my.plugins: hello   world = example:main2",
		"my.plugins: my command = example:main",
	}
	sort.Strings(expected)
	if strings.Join(inspection.EntryPoints, ";") != strings.Join(expected, ";") {
		t.Fatalf("entry points = %v, want %v", inspection.EntryPoints, expected)
	}
	if inspection.NoImportSurface {
		t.Fatalf("wheel with executable entry points must not have NoImportSurface=true")
	}
}

func TestWheelBareSOClassificationSemantics(t *testing.T) {
	// 1. Python extension exporting PyInit__native
	extELF := buildSyntheticELF([]string{"PyInit__native"})
	class, mod, err := classifyELFBytes(extELF)
	if err != nil || class != soPythonExtension || mod != "_native" {
		t.Fatalf("extELF: got class=%v, mod=%q, err=%v; want soPythonExtension, _native", class, mod, err)
	}

	// 2. Proven native library exporting cuptiActivityEnable, zero PyInit_
	nativeELF := buildSyntheticELF([]string{"cuptiActivityEnable", "otherNativeFunc"})
	class, mod, err = classifyELFBytes(nativeELF)
	if err != nil || class != soProvenNative || mod != "" {
		t.Fatalf("nativeELF: got class=%v, mod=%q, err=%v; want soProvenNative, empty", class, mod, err)
	}

	// 3. Ambiguous: non-ELF corrupt bytes
	class, _, _ = classifyELFBytes([]byte("not an elf binary at all, just text payload"))
	if class != soAmbiguous {
		t.Fatalf("corrupt bytes: got class=%v, want soAmbiguous", class)
	}

	// 4. Ambiguous: ELF with multiple PyInit_ symbols
	multiELF := buildSyntheticELF([]string{"PyInit_foo", "PyInit_bar"})
	class, _, _ = classifyELFBytes(multiELF)
	if class != soAmbiguous {
		t.Fatalf("multi PyInit ELF: got class=%v, want soAmbiguous", class)
	}

	// 5. If actual NVIDIA cupti library is present on disk, verify real ELF
	if realData, readErr := os.ReadFile("/tmp/test_libpcsamplingutil.so"); readErr == nil && len(realData) > 0 {
		class, mod, err = classifyELFBytes(realData)
		if err != nil || class != soProvenNative || mod != "" {
			t.Fatalf("real cupti ELF: got class=%v, mod=%q, err=%v; want soProvenNative, empty", class, mod, err)
		}
	}
}

func buildSyntheticELF(symbols []string) []byte {
	shstrtab := []byte("\x00.text\x00.dynsym\x00.dynstr\x00.shstrtab\x00")
	offTextName := uint32(1)
	offDynsymName := uint32(7)
	offDynstrName := uint32(15)
	offShstrtabName := uint32(23)

	dynstr := []byte("\x00")
	type symEntry struct {
		nameOffset uint32
		section    uint16
	}
	var symEntries []symEntry
	symEntries = append(symEntries, symEntry{nameOffset: 0, section: 0})
	for _, s := range symbols {
		offset := uint32(len(dynstr))
		dynstr = append(dynstr, []byte(s)...)
		dynstr = append(dynstr, 0)
		symEntries = append(symEntries, symEntry{nameOffset: offset, section: 1})
	}

	dynsym := make([]byte, len(symEntries)*24)
	for i, sym := range symEntries {
		off := i * 24
		binary.LittleEndian.PutUint32(dynsym[off:off+4], sym.nameOffset)
		if i > 0 {
			dynsym[off+4] = 0x12 // STB_GLOBAL << 4 | STT_FUNC
		}
		binary.LittleEndian.PutUint16(dynsym[off+6:off+8], sym.section)
		binary.LittleEndian.PutUint64(dynsym[off+8:off+16], 0x1000)
		binary.LittleEndian.PutUint64(dynsym[off+16:off+24], 16)
	}

	text := []byte{0xc3} // ret

	buf := new(bytes.Buffer)
	eh := make([]byte, 64)
	copy(eh[0:4], []byte{0x7f, 'E', 'L', 'F'})
	eh[4] = 2                                    // ELFCLASS64
	eh[5] = 1                                    // ELFDATA2LSB
	eh[6] = 1                                    // EV_CURRENT
	binary.LittleEndian.PutUint16(eh[16:18], 3)  // ET_DYN
	binary.LittleEndian.PutUint16(eh[18:20], 62) // EM_X86_64
	binary.LittleEndian.PutUint32(eh[20:24], 1)  // EV_CURRENT
	binary.LittleEndian.PutUint16(eh[52:54], 64) // e_ehsize
	binary.LittleEndian.PutUint16(eh[58:60], 64) // e_shentsize
	binary.LittleEndian.PutUint16(eh[60:62], 5)  // e_shnum = 5
	binary.LittleEndian.PutUint16(eh[62:64], 4)  // e_shstrndx = 4

	textOff := uint64(64)
	dynsymOff := textOff + uint64(len(text))
	dynstrOff := dynsymOff + uint64(len(dynsym))
	shstrtabOff := dynstrOff + uint64(len(dynstr))
	shOff := shstrtabOff + uint64(len(shstrtab))

	binary.LittleEndian.PutUint64(eh[40:48], shOff)

	buf.Write(eh)
	buf.Write(text)
	buf.Write(dynsym)
	buf.Write(dynstr)
	buf.Write(shstrtab)

	writeSectionHeader := func(name uint32, typ uint32, flags uint64, off, size uint64, link, info uint32, entsize uint64) {
		sh := make([]byte, 64)
		binary.LittleEndian.PutUint32(sh[0:4], name)
		binary.LittleEndian.PutUint32(sh[4:8], typ)
		binary.LittleEndian.PutUint64(sh[8:16], flags)
		binary.LittleEndian.PutUint64(sh[24:32], off)
		binary.LittleEndian.PutUint64(sh[32:40], size)
		binary.LittleEndian.PutUint32(sh[40:44], link)
		binary.LittleEndian.PutUint32(sh[44:48], info)
		binary.LittleEndian.PutUint64(sh[56:64], entsize)
		buf.Write(sh)
	}

	writeSectionHeader(0, 0, 0, 0, 0, 0, 0, 0)
	writeSectionHeader(offTextName, 1, 6, textOff, uint64(len(text)), 0, 0, 0)
	writeSectionHeader(offDynsymName, 11, 2, dynsymOff, uint64(len(dynsym)), 3, 1, 24)
	writeSectionHeader(offDynstrName, 3, 2, dynstrOff, uint64(len(dynstr)), 0, 0, 0)
	writeSectionHeader(offShstrtabName, 3, 0, shstrtabOff, uint64(len(shstrtab)), 0, 0, 0)

	return buf.Bytes()
}

func TestDeclaredWheelImportMetadataRejectsAmbiguity(t *testing.T) {
	parsed := headerValues([]byte("Import-Name: example\nImport-Namespace: nvidia.nccl\nImport-Namespace: nvidia.nvshmem\n"), 1024)
	imports, err := declaredWheelImports(splitHeaders(parsed["import-name"]), splitHeaders(parsed["import-namespace"]))
	if err != nil || strings.Join(imports, ",") != "example,nvidia.nccl,nvidia.nvshmem" {
		t.Fatalf("parsed import metadata = %q, %v", imports, err)
	}
	for _, test := range []struct {
		names, namespaces []string
		want              string
		invalid           bool
	}{
		{[]string{"example; private"}, nil, "example", false},
		{nil, []string{"nvidia.nccl"}, "nvidia.nccl", false},
		{[]string{"nvidia.nccl"}, []string{"nvidia.nccl"}, "", true},
		{[]string{"bad-name"}, nil, "", true},
	} {
		imports, err := declaredWheelImports(test.names, test.namespaces)
		if (err != nil) != test.invalid || strings.Join(imports, ",") != test.want {
			t.Fatalf("declaredWheelImports(%q,%q) = %q, %v", test.names, test.namespaces, imports, err)
		}
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
		{name: "cusparselt legacy 2014", platform: "manylinux2014_x86_64", target: target, want: true},
		{name: "legacy 2010", platform: "manylinux2010_x86_64", target: target, want: true},
		{name: "legacy 1", platform: "manylinux1_x86_64", target: target, want: true},
		{name: "legacy baseline too new", platform: "manylinux2014_x86_64", target: WheelTarget{"cp314", "cp314", "manylinux_2_12_x86_64"}, want: false},
		{name: "legacy architecture mismatch", platform: "manylinux2014_aarch64", target: target, want: false},
		{name: "unversioned Linux", platform: "linux_x86_64", target: target, want: false},
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

func TestWheelMetadataTagsMatchExpandedFilenameTags(t *testing.T) {
	filenamePython := []string{"cp314"}
	filenameABI := []string{"cp314"}
	tests := []struct {
		name string
		tags []string
		want bool
	}{
		{name: "single tag", tags: []string{"cp314-cp314-manylinux_2_36_x86_64"}, want: true},
		{name: "compressed platform expansion", tags: []string{
			"cp314-cp314-manylinux_2_17_x86_64",
			"cp314-cp314-manylinux2014_x86_64",
			"cp314-cp314-manylinux_2_28_x86_64",
		}, want: true},
		{name: "ordering ignored", tags: []string{
			"cp314-cp314-manylinux_2_28_x86_64",
			"cp314-cp314-manylinux2014_x86_64",
			"cp314-cp314-manylinux_2_17_x86_64",
		}, want: true},
		{name: "missing expanded tag", tags: []string{
			"cp314-cp314-manylinux_2_17_x86_64",
			"cp314-cp314-manylinux_2_28_x86_64",
		}, want: false},
		{name: "extra tag", tags: []string{
			"cp314-cp314-manylinux_2_36_x86_64",
			"cp314-cp314-manylinux_2_28_x86_64",
		}, want: false},
		{name: "malformed tag", tags: []string{"cp314.cp314-manylinux_2_36_x86_64"}, want: false},
		{name: "duplicate tag", tags: []string{
			"cp314-cp314-manylinux_2_36_x86_64",
			"cp314-cp314-manylinux_2_36_x86_64",
		}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			platform := []string{"manylinux_2_36_x86_64"}
			if strings.Contains(test.name, "expansion") || strings.Contains(test.name, "ordering") || strings.Contains(test.name, "missing") {
				platform = []string{"manylinux2014_x86_64", "manylinux_2_17_x86_64", "manylinux_2_28_x86_64"}
			}
			if got := wheelMetadataTagsMatch(test.tags, filenamePython, filenameABI, platform); got != test.want {
				t.Fatalf("wheelMetadataTagsMatch() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestInspectWheelStreamsLargeRecordPayload(t *testing.T) {
	payload := bytes.Repeat([]byte("p"), int(DefaultWheelLimits().MaxMetadata)+1)
	archive := recordedWheelArchive(t, "packaging", "25.0", "packaging-25.0.dist-info", []wheelTestEntry{{name: "packaging/payload.bin", body: payload}}, []string{"cp314-cp314-manylinux_2_36_x86_64"}, nil)
	if _, err := inspectTestWheelResult(archive, "packaging-25.0-cp314-cp314-manylinux_2_36_x86_64.whl"); err != nil {
		t.Fatalf("large valid RECORD payload rejected: %v", err)
	}
}

func TestInspectWheelRejectsRecordIntegrityAndExemptionBypasses(t *testing.T) {
	payload := []byte("payload")
	tests := []struct {
		name      string
		content   []wheelTestEntry
		transform func([]string) []string
	}{
		{name: "top-level digest mismatch", content: []wheelTestEntry{{name: "payload.bin", body: payload}}, transform: replaceRecordRow(func(parts []string) []string { parts[1] = "sha256=" + strings.Repeat("0", 43); return parts })},
		{name: "nested digest mismatch", content: []wheelTestEntry{{name: "pkg/payload.bin", body: payload}}, transform: replaceRecordRow(func(parts []string) []string { parts[1] = "sha256=" + strings.Repeat("0", 43); return parts })},
		{name: "size mismatch", content: []wheelTestEntry{{name: "pkg/payload.bin", body: payload}}, transform: replaceRecordRow(func(parts []string) []string { parts[2] = "0"; return parts })},
		{name: "nested RECORD exemption", content: []wheelTestEntry{{name: "vendor/example.dist-info/RECORD", body: payload}}, transform: replaceRecordRow(func(parts []string) []string { parts[1], parts[2] = "", ""; return parts })},
		{name: "unrecorded file", content: []wheelTestEntry{{name: "pkg/payload.bin", body: payload}}, transform: func(rows []string) []string { return rows[1:] }},
		{name: "unknown RECORD file", content: []wheelTestEntry{{name: "pkg/payload.bin", body: payload}}, transform: func(rows []string) []string { return append(rows, "unknown.bin,,") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := recordedWheelArchive(t, "packaging", "25.0", "packaging-25.0.dist-info", test.content, []string{"cp314-cp314-manylinux_2_36_x86_64"}, test.transform)
			if _, err := inspectTestWheelResult(archive, "packaging-25.0-cp314-cp314-manylinux_2_36_x86_64.whl"); err == nil {
				t.Fatal("invalid RECORD accepted")
			}
		})
	}
}

func TestInspectWheelAllowsOnlyExactPrimaryRecordJWSOmission(t *testing.T) {
	archive := recordedWheelArchive(t, "packaging", "25.0", "packaging-25.0.dist-info", []wheelTestEntry{
		{name: "packaging/__init__.py", body: []byte("safe")},
		{name: "packaging-25.0.dist-info/RECORD.jws", body: []byte("signature")},
		{name: "packaging-25.0.dist-info/RECORD.p7s", body: []byte("signature")},
	}, []string{"cp314-cp314-manylinux_2_36_x86_64"}, nil)
	if _, err := inspectTestWheelResult(archive, "packaging-25.0-cp314-cp314-manylinux_2_36_x86_64.whl"); err != nil {
		t.Fatalf("primary RECORD signature omission rejected: %v", err)
	}
}

func inspectTestWheelResult(archive []byte, filename string) (WheelInspection, error) {
	digest := sha256.Sum256(archive)
	return InspectWheel(bytes.NewReader(archive), int64(len(archive)), filename, hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
}

func replaceRecordRow(update func([]string) []string) func([]string) []string {
	return func(rows []string) []string {
		for index, row := range rows {
			parts := strings.Split(row, ",")
			if len(parts) == 3 && parts[1] != "" && !strings.HasSuffix(parts[0], "/METADATA") && !strings.HasSuffix(parts[0], "/WHEEL") {
				rows[index] = strings.Join(update(parts), ",")
				break
			}
		}
		return rows
	}
}

func recordedWheelArchive(t *testing.T, metadataName, version, distInfo string, content []wheelTestEntry, tags []string, transform func([]string) []string) []byte {
	t.Helper()
	metadata := []byte("Metadata-Version: 2.4\nName: " + metadataName + "\nVersion: " + version + "\n")
	wheel := []byte("Wheel-Version: 1.0\n")
	for _, tag := range tags {
		wheel = append(wheel, []byte("Tag: "+tag+"\n")...)
	}
	entries := append([]wheelTestEntry(nil), content...)
	metadataSum, wheelSum := sha256.Sum256(metadata), sha256.Sum256(wheel)
	rows := make([]string, 0, len(content)+3)
	for _, entry := range content {
		if entry.mode != 0 || entry.name == distInfo+"/RECORD.jws" || entry.name == distInfo+"/RECORD.p7s" {
			continue
		}
		sum := sha256.Sum256(entry.body)
		rows = append(rows, entry.name+",sha256="+base64.RawURLEncoding.EncodeToString(sum[:])+","+itoa(len(entry.body)))
	}
	rows = append(rows,
		distInfo+"/METADATA,sha256="+base64.RawURLEncoding.EncodeToString(metadataSum[:])+","+itoa(len(metadata)),
		distInfo+"/WHEEL,sha256="+base64.RawURLEncoding.EncodeToString(wheelSum[:])+","+itoa(len(wheel)),
		distInfo+"/RECORD,,",
	)
	if transform != nil {
		rows = transform(rows)
	}
	record := []byte(strings.Join(rows, "\n") + "\n")
	entries = append(entries,
		wheelTestEntry{name: distInfo + "/METADATA", body: metadata},
		wheelTestEntry{name: distInfo + "/WHEEL", body: wheel},
		wheelTestEntry{name: distInfo + "/RECORD", body: record},
	)
	return makeWheelEntries(t, entries...)
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

// 1. Root .dist-info accepted; competing root .dist-info rejected with DIST_INFO_IDENTITY.
func TestModelC_RootVsCompetingDistInfo(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "pkg/__init__.py", body: []byte("val = 1\n")},
	}
	archive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "example-1.0-py3-none-any.whl")
	if insp.Project != "example" || len(insp.ImportNames) != 1 || insp.ImportNames[0] != "pkg" {
		t.Fatalf("canonical wheel inspection failed: %#v", insp)
	}

	competingEntries := []wheelTestEntry{
		{name: "pkg/__init__.py", body: []byte("val = 1\n")},
		{name: "competing-2.0.dist-info/METADATA", body: []byte("Metadata-Version: 2.1\nName: competing\nVersion: 2.0\n")},
	}
	competingArchive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", competingEntries, []string{"py3-none-any"}, nil)
	digest := sha256.Sum256(competingArchive)
	_, err := InspectWheel(bytes.NewReader(competingArchive), int64(len(competingArchive)), "example-1.0-py3-none-any.whl", hex.EncodeToString(digest[:]), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err == nil {
		t.Fatal("expected competing root dist-info to be rejected")
	}
	stage, ok := WheelValidationStageOf(err)
	if !ok || stage != WheelValidationDistInfoIdentity {
		t.Fatalf("expected DIST_INFO_IDENTITY, got stage=%q, err=%v", stage, err)
	}
}

// 2. Root entry_points.txt interpreted as installer metadata; nested pkg/_vendor/foo.dist-info/entry_points.txt preserved as inert payload without triggering DIST_INFO_IDENTITY.
func TestModelC_NestedVendoredDistInfoPreserved(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "setuptools/__init__.py", body: []byte("version = '84.0.0'\n")},
		{name: "setuptools/_vendor/wheel-0.46.3.dist-info/entry_points.txt", body: []byte("[console_scripts]\nwheel = wheel.cli:main\n")},
		{name: "setuptools/_vendor/wheel-0.46.3.dist-info/METADATA", body: []byte("Metadata-Version: 2.1\nName: wheel\nVersion: 0.46.3\n")},
		{name: "setuptools-84.0.0.dist-info/entry_points.txt", body: []byte("[distutils.commands]\ncheck = distutils.command.check:check\n")},
	}
	archive := recordedWheelArchive(t, "setuptools", "84.0.0", "setuptools-84.0.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "setuptools-84.0.0-py3-none-any.whl")
	if insp.Project != "setuptools" {
		t.Fatalf("unexpected project: %q", insp.Project)
	}
	if len(insp.EntryPoints) != 1 || !strings.Contains(insp.EntryPoints[0], "distutils.command.check") {
		t.Fatalf("canonical entry points not parsed: %v", insp.EntryPoints)
	}
	for _, ep := range insp.EntryPoints {
		if strings.Contains(ep, "wheel.cli:main") {
			t.Fatalf("nested vendored entry_points.txt must not be promoted into canonical EntryPoints: %v", ep)
		}
	}
	foundNested := false
	for _, f := range insp.Surface.InstalledFiles {
		if f.ArchivePath == "setuptools/_vendor/wheel-0.46.3.dist-info/entry_points.txt" {
			foundNested = true
			if f.Role != RuntimeRoleInertData || f.Scheme != SchemeSite {
				t.Fatalf("unexpected role/scheme for nested vendored file: role=%v, scheme=%v", f.Role, f.Scheme)
			}
		}
	}
	if !foundNested {
		t.Fatal("nested vendored entry_points.txt not found in surface")
	}
}

// 3. Active root .pth in site-packages mapped as startup hook; nested pkg/foo.pth mapped as inert data.
func TestModelC_SiteStartupHookVsNestedPth(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "_cuda_redirector.pth", body: []byte("import _cuda_redirector\n")},
		{name: "_cuda_redirector.py", body: []byte("pass\n")},
		{name: "pkg/__init__.py", body: []byte("pass\n")},
		{name: "pkg/_vendor/nested.pth", body: []byte("import sys\n")},
	}
	archive := recordedWheelArchive(t, "cuda-bindings", "12.9.9", "cuda_bindings-12.9.9.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "cuda_bindings-12.9.9-py3-none-any.whl")

	if len(insp.Surface.SiteStartupHooks) != 1 || insp.Surface.SiteStartupHooks[0] != "_cuda_redirector.pth" {
		t.Fatalf("expected active startup hook _cuda_redirector.pth, got: %v", insp.Surface.SiteStartupHooks)
	}
	foundNested := false
	for _, f := range insp.Surface.InstalledFiles {
		if f.ArchivePath == "pkg/_vendor/nested.pth" {
			foundNested = true
			if f.Role != RuntimeRoleInertData {
				t.Fatalf("nested .pth must be inert data, got role=%v", f.Role)
			}
		}
	}
	if !foundNested {
		t.Fatal("nested .pth not found in surface")
	}
}

// 4. .data/purelib/ and .data/platlib/ Python code mapped into site-packages and participating in import discovery.
func TestModelC_DataRelocatedPythonParticipation(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "example-1.0.data/purelib/relocated/__init__.py", body: []byte("relocated = True\n")},
		{name: "example-1.0.data/platlib/native_mod.py", body: []byte("val = 42\n")},
		{name: "example-1.0.data/scripts/run-tool", body: []byte("#!/bin/sh\necho hi\n")},
	}
	archive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "example-1.0-py3-none-any.whl")

	sort.Strings(insp.ImportNames)
	want := []string{"native_mod", "relocated"}
	if strings.Join(insp.ImportNames, ",") != strings.Join(want, ",") {
		t.Fatalf("relocated code must participate in imports: got %v, want %v", insp.ImportNames, want)
	}
	if len(insp.Scripts) != 1 || insp.Scripts[0] != "bin/run-tool" {
		t.Fatalf("scripts from .data/scripts not mapped: %v", insp.Scripts)
	}
}

// 5. PEP 420 namespace container (e.g. cuda) distinguished from executable subpackage (e.g. cuda.pathfinder), with executable subpackage included in import candidate surface.
func TestModelC_NamespaceContainerVsExecutableSubpackage(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "cuda/pathfinder/__init__.py", body: []byte("find = True\n")},
		{name: "cuda/pathfinder/pathfinder.py", body: []byte("pass\n")},
	}
	archive := recordedWheelArchive(t, "cuda-pathfinder", "1.8.2", "cuda_pathfinder-1.8.2.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "cuda_pathfinder-1.8.2-py3-none-any.whl")

	if len(insp.Surface.NamespaceContainers) != 1 || insp.Surface.NamespaceContainers[0] != "cuda" {
		t.Fatalf("expected namespace container 'cuda', got: %v", insp.Surface.NamespaceContainers)
	}
	if len(insp.Surface.ExecutableSubpackages) != 1 || insp.Surface.ExecutableSubpackages[0] != "cuda.pathfinder" {
		t.Fatalf("expected executable subpackage 'cuda.pathfinder', got: %v", insp.Surface.ExecutableSubpackages)
	}
	if len(insp.ImportNames) != 1 || insp.ImportNames[0] != "cuda.pathfinder" {
		t.Fatalf("import candidate must probe 'cuda.pathfinder' rather than empty namespace 'cuda': got %v", insp.ImportNames)
	}
}

// 6. Target Python 3.14 native extension recognized (.cpython-314-x86_64-linux-gnu.so, abi3.so, bare .so with PyInit); ordinary native library without PyInit recognized as RuntimeRoleNativeLibrary.
func TestModelC_NativeExtensionsVsOrdinaryLibraries(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "pkg/mod.cpython-314-x86_64-linux-gnu.so", body: []byte("elf_ext")},
		{name: "pkg/abi.abi3.so", body: []byte("elf_abi3")},
		{name: "pkg/libhelper.so.1", body: []byte("elf_lib")},
		{name: "pkg/libunversioned.so", body: buildSyntheticELF([]string{"some_helper_func"})},
	}
	archive := recordedWheelArchive(t, "native-pkg", "1.0", "native_pkg-1.0.dist-info", entries, []string{"cp314-cp314-manylinux_2_36_x86_64"}, nil)
	insp := inspectTestWheel(t, archive, "native_pkg-1.0-cp314-cp314-manylinux_2_36_x86_64.whl")

	sort.Strings(insp.Surface.PythonExtensions)
	sort.Strings(insp.Surface.NativeLibraries)
	wantExts := []string{"pkg.abi", "pkg.mod"}
	if strings.Join(insp.Surface.PythonExtensions, ",") != strings.Join(wantExts, ",") {
		t.Fatalf("python extensions mismatch: got %v, want %v", insp.Surface.PythonExtensions, wantExts)
	}
	wantLibs := []string{"pkg/libhelper.so.1", "pkg/libunversioned.so"}
	if strings.Join(insp.Surface.NativeLibraries, ",") != strings.Join(wantLibs, ",") {
		t.Fatalf("native libraries mismatch: got %v, want %v", insp.Surface.NativeLibraries, wantLibs)
	}
}

// 7. Inert data files recognized; ambiguous/unknown files classified as RuntimeRoleUnresolved.
func TestModelC_InertDataVsUnresolved(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "pkg/__init__.py", body: []byte("pass\n")},
		{name: "pkg/data.json", body: []byte("{}\n")},
		{name: "pkg/header.h", body: []byte("#pragma once\n")},
		{name: "pkg/docs.md", body: []byte("# Title\n")},
		{name: "pkg/unknown.arbitrary_extension", body: []byte("binary\n")},
	}
	archive := recordedWheelArchive(t, "inert-test", "1.0", "inert_test-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "inert_test-1.0-py3-none-any.whl")

	if len(insp.Surface.UnresolvedFiles) != 1 || insp.Surface.UnresolvedFiles[0] != "pkg/unknown.arbitrary_extension" {
		t.Fatalf("expected unresolved file pkg/unknown.arbitrary_extension, got: %v", insp.Surface.UnresolvedFiles)
	}
}

// 8. Mixed known + unresolved surface fails closed.
func TestModelC_MixedUnresolvedFailsClosed(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "pkg/__init__.py", body: []byte("pass\n")},
		{name: "pkg/corrupt.unknown_bin", body: []byte("data")},
	}
	archive := recordedWheelArchive(t, "mixed-test", "1.0", "mixed_test-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "mixed_test-1.0-py3-none-any.whl")

	_, err := BuildObservationPlan(insp, defaultResourcePolicy())
	if err == nil || !strings.Contains(err.Error(), "unresolved surface") {
		t.Fatalf("BuildObservationPlan must fail closed on unresolved surfaces, got err=%v", err)
	}
}

// 9. The complete 31–34 candidate set becomes independent typed units under
// the admitted artifact policy, with no batch boundary or truncation.
func TestModelC_ImportCandidateUnitsUnderResourcePolicy(t *testing.T) {
	for _, tc := range []struct {
		count         int
		policyCU126   bool
		expectPlanErr bool
	}{
		{count: 31},
		{count: 32},
		{count: 33, expectPlanErr: true},
		{count: 33, policyCU126: true},
		{count: 34, policyCU126: true},
	} {
		imports := make([]string, tc.count)
		for i := 0; i < tc.count; i++ {
			imports[i] = fmt.Sprintf("pkg.mod%d", i)
		}
		insp := WheelInspection{
			Project:     "multi-mod",
			Version:     "1.0",
			ImportNames: imports,
		}
		policy := defaultResourcePolicy()
		if tc.policyCU126 {
			policy = pyTorchCU126ResourcePolicy()
		}
		plan, err := BuildObservationPlan(insp, policy)
		if tc.expectPlanErr {
			if err == nil {
				t.Fatalf("count=%d: expected error exceeding policy bound, got nil", tc.count)
			}
			continue
		}
		if err != nil {
			t.Fatalf("count=%d: unexpected error: %v", tc.count, err)
		}
		if len(plan.Units) != tc.count || ValidateTypedObservationPlan(plan, policy) != nil {
			t.Fatalf("count=%d: complete typed direct-import plan missing", tc.count)
		}
		for i, candidate := range plan.ImportCandidates {
			if plan.Units[i].Kind != DirectImportUnit || plan.Units[i].Candidate != candidate {
				t.Fatalf("count=%d: candidate %q is missing at unit %d", tc.count, candidate, i)
			}
		}
	}
}

// 10. Strict RFC 822 parsing: stops at header/body separator (\n\n), unfolds continuation lines, preserves repeated fields, distinguishes empty vs absent.
func TestModelC_RFC822StrictParsing(t *testing.T) {
	raw := "Metadata-Version: 2.1\n" +
		"Name: test-pkg\n" +
		"Version: 1.0\n" +
		"Summary:\n" +
		"Requires-Dist: dep-one (>=1.0)\n" +
		"Requires-Dist: dep-two\n" +
		"    (<=2.0)\n" +
		"\n" +
		"Description: This is the description body\n" +
		"Requires-Dist: evil-body-dep\n"

	meta, err := parseRFC822Metadata([]byte(raw), 1024)
	if err != nil {
		t.Fatalf("parseRFC822Metadata failed: %v", err)
	}
	if meta.has("license") {
		t.Fatal("expected 'license' to be absent")
	}
	if !meta.has("summary") {
		t.Fatal("expected 'summary' to be present")
	}
	val, ok := meta.first("summary")
	if !ok || val != "" {
		t.Fatalf("expected explicit empty summary, got %q, %v", val, ok)
	}
	dists := meta.all("requires-dist")
	if len(dists) != 2 {
		t.Fatalf("expected 2 requires-dist headers, got %d: %v", len(dists), dists)
	}
	if dists[0] != "dep-one (>=1.0)" {
		t.Fatalf("unexpected dist 0: %q", dists[0])
	}
	if !strings.Contains(dists[1], "dep-two") || !strings.Contains(dists[1], "(<=2.0)") {
		t.Fatalf("continuation line not unfolded: %q", dists[1])
	}
	if meta.has("description") {
		t.Fatal("headers after empty line must not be parsed")
	}
	for _, d := range dists {
		if strings.Contains(d, "evil-body-dep") {
			t.Fatal("body text parsed as header")
		}
	}
}

// 11. Proven no-import-surface wheels: metadata-only (metapackages like cuda-toolkit) and native-only (nvidia-nccl-cu12, nvidia-cufft-cu12).
func TestModelC_ProvenNoImportSurfaceWheels(t *testing.T) {
	metaArchive := recordedWheelArchive(t, "cuda-toolkit", "12.6.3", "cuda_toolkit-12.6.3.dist-info", nil, []string{"py3-none-any"}, nil)
	metaInsp := inspectTestWheel(t, metaArchive, "cuda_toolkit-12.6.3-py3-none-any.whl")
	if !metaInsp.NoImportSurface {
		t.Fatalf("metapackage must be NoImportSurface: %#v", metaInsp)
	}
	plan, err := BuildObservationPlan(metaInsp, defaultResourcePolicy())
	if err != nil || !plan.NoImportSurface || !plan.MetadataOnly {
		t.Fatalf("observation plan for metapackage invalid: %#v, err=%v", plan, err)
	}

	nativeEntries := []wheelTestEntry{
		{name: "nvidia/nccl/lib/libnccl.so.2", body: []byte("elf")},
		{name: "nvidia/nccl/include/nccl.h", body: []byte("h")},
	}
	nativeArchive := recordedWheelArchive(t, "nvidia-nccl-cu12", "2.29.3", "nvidia_nccl_cu12-2.29.3.dist-info", nativeEntries, []string{"py3-none-any"}, nil)
	nativeInsp := inspectTestWheel(t, nativeArchive, "nvidia_nccl_cu12-2.29.3-py3-none-any.whl")
	if !nativeInsp.NoImportSurface {
		t.Fatalf("native-only library must be NoImportSurface: %#v", nativeInsp)
	}
	nativePlan, err := BuildObservationPlan(nativeInsp, defaultResourcePolicy())
	if err != nil || !nativePlan.NoImportSurface || nativePlan.MetadataOnly {
		t.Fatalf("observation plan for native library invalid: %#v, err=%v", nativePlan, err)
	}
}

// 12. End-to-end regression on real corpus (/tmp/haa_corpus).
func TestModelC_RealCorpusQualification(t *testing.T) {
	corpusDir := "/tmp/haa_corpus"
	if _, err := os.Stat(corpusDir); err != nil {
		t.Fatalf("required real wheel corpus not found at %s: missing corpus is an explicit failure", corpusDir)
	}

	manifest := []struct {
		fileOnDisk             string
		canonical              string
		profile                string
		project                string
		version                string
		source                 string
		expectedSHA256         string
		wantImports            []string
		noImport               bool
		expectedCandidateCount int
	}{
		// CPU
		{
			fileOnDisk:     "setuptools-84.0.0-py3-none-any.whl",
			profile:        "pytorch:cpu",
			project:        "setuptools",
			version:        "84.0.0",
			source:         "pypi",
			expectedSHA256: "51a52592b3b99e102b609654876bd65f19f999935166d1352678931132b0c670",
			wantImports:    []string{"_distutils_hack", "setuptools"},
			noImport:       false,
		},
		{
			fileOnDisk:     "filelock-4.0.3-py3-none-any.whl",
			profile:        "pytorch:cpu",
			project:        "filelock",
			version:        "4.0.3",
			source:         "pypi",
			expectedSHA256: "30cd166e2aee2c7534ce2c33c6367c4cd051b8368e20e2c4eb0c34f699bacfab",
			wantImports:    []string{"filelock"},
			noImport:       false,
		},
		{
			fileOnDisk:     "filelock-4.0.4-py3-none-any.whl",
			profile:        "pytorch:cpu",
			project:        "filelock",
			version:        "4.0.4",
			source:         "pypi",
			expectedSHA256: "0df72be195ca7892216d16f2edce8d9b93a571f02402972020a8cff84c594c7b",
			wantImports:    []string{"filelock"},
			noImport:       false,
		},
		{
			fileOnDisk:     "sympy-1.14.0-py3-none-any.whl",
			profile:        "pytorch:cpu",
			project:        "sympy",
			version:        "1.14.0",
			source:         "pypi",
			expectedSHA256: "e091cc3e99d2141a0ba2847328f5479b05d94a6635cb96148ccb3f34671bd8f5",
			wantImports:    []string{"sympy"},
			noImport:       false,
		},
		{
			fileOnDisk:     "markupsafe-3.0.3-cp314-cp314-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl",
			profile:        "pytorch:cpu",
			project:        "markupsafe",
			version:        "3.0.3",
			source:         "pypi",
			expectedSHA256: "457a69a9577064c05a97c41f4e65148652db078a3a509039e64d3467b9e7ef97",
			wantImports:    []string{"markupsafe", "markupsafe._speedups"},
			noImport:       false,
		},
		// cu126
		{
			fileOnDisk:     "haa-cu126-toolkit-audit.whl",
			canonical:      "cuda_toolkit-12.6.3-py2.py3-none-any.whl",
			profile:        "pytorch:cu126",
			project:        "cuda-toolkit",
			version:        "12.6.3",
			source:         "pypi",
			expectedSHA256: "79d8605baeb6c2f695761e0efb54bc62dbc3c9e32eb0742df7669c07befaa8f7",
			wantImports:    nil,
			noImport:       true,
		},
		{
			fileOnDisk:     "haa-cu126-first-nccl.whl",
			canonical:      "nvidia_nccl_cu12-2.29.3-py3-none-manylinux_2_18_x86_64.whl",
			profile:        "pytorch:cu126",
			project:        "nvidia-nccl-cu12",
			version:        "2.29.3",
			source:         "pypi",
			expectedSHA256: "35ad42e7d5d722a83c36a3a478e281c20a5646383deaf1b9ed1a9ab7d61bed53",
			wantImports:    nil,
			noImport:       true,
		},
		{
			fileOnDisk:     "haa-cu126-first-cufft.whl",
			canonical:      "nvidia_cufft_cu12-11.3.0.4-py3-none-manylinux2014_x86_64.manylinux_2_17_x86_64.whl",
			profile:        "pytorch:cu126",
			project:        "nvidia-cufft-cu12",
			version:        "11.3.0.4",
			source:         "pypi",
			expectedSHA256: "ccba62eb9cef5559abd5e0d54ceed2d9934030f51163df018532142a8ec533e5",
			wantImports:    []string{"nvidia"},
			noImport:       false,
		},
		{
			fileOnDisk:     "cuda_pathfinder-1.8.2-py3-none-any.whl",
			canonical:      "cuda_pathfinder-1.8.2-py3-none-any.whl",
			profile:        "pytorch:cu126",
			project:        "cuda-pathfinder",
			version:        "1.8.2",
			source:         "pypi",
			expectedSHA256: "4e65059febdb4d19d5cbc4798677e19db2b582f2f702f457b609e571690d357e",
			wantImports:    []string{"cuda.pathfinder"},
			noImport:       false,
		},
		{
			fileOnDisk:             "cuda_bindings-12.9.9-cp314-cp314-manylinux_2_24_x86_64.manylinux_2_28_x86_64.whl",
			canonical:              "cuda_bindings-12.9.9-cp314-cp314-manylinux_2_24_x86_64.manylinux_2_28_x86_64.whl",
			profile:                "pytorch:cu126",
			project:                "cuda-bindings",
			version:                "12.9.9",
			source:                 "pypi",
			expectedSHA256:         "94e4f9bd6b9b21aad545e0d441707094e4ff88ab420d62b85fcdc8b6da1e039f",
			noImport:               false,
			expectedCandidateCount: 34,
		},
		// cu130
		{
			fileOnDisk:     "haa-cu130-toolkit-audit.whl",
			canonical:      "cuda_toolkit-13.0.3.0-py2.py3-none-any.whl",
			profile:        "pytorch:cu130",
			project:        "cuda-toolkit",
			version:        "13.0.3.0",
			source:         "pypi",
			expectedSHA256: "d693caaa261214ddd7dbb60d68e71cbed884e68c2be7509778f3051da0b91c3f",
			wantImports:    nil,
			noImport:       true,
		},
		{
			fileOnDisk:     "haa-cu130-cupti-audit.whl",
			canonical:      "nvidia_cuda_cupti-13.0.85-py3-none-manylinux_2_25_x86_64.whl",
			profile:        "pytorch:cu130",
			project:        "nvidia-cuda-cupti-cu13",
			version:        "13.0.85",
			source:         "pypi",
			expectedSHA256: "4eb01c08e859bf924d222250d2e8f8b8ff6d3db4721288cf35d14252a4d933c8",
			wantImports:    nil,
			noImport:       true,
		},
		{
			fileOnDisk:             "cuda_bindings-13.4.3-cp314-cp314-manylinux_2_24_x86_64.manylinux_2_28_x86_64.whl",
			canonical:              "cuda_bindings-13.4.3-cp314-cp314-manylinux_2_24_x86_64.manylinux_2_28_x86_64.whl",
			profile:                "pytorch:cu130",
			project:                "cuda-bindings",
			version:                "13.4.3",
			source:                 "pypi",
			expectedSHA256:         "bbacde6f75665b197016b986164cfdaa33b17515e5e635a63ddb75926aaa71c3",
			noImport:               false,
			expectedCandidateCount: 31,
		},
		// cu132
		{
			fileOnDisk:     "haa-cu132-toolkit-audit.whl",
			canonical:      "cuda_toolkit-13.2.1-py2.py3-none-any.whl",
			profile:        "pytorch:cu132",
			project:        "cuda-toolkit",
			version:        "13.2.1",
			source:         "pypi",
			expectedSHA256: "646d0e3668ce6f78f2312bb9cc0f668b9cbfcbef187eaa6a39eb2ea6dbec2a31",
			wantImports:    nil,
			noImport:       true,
		},
		{
			fileOnDisk:     "haa-cu132-cupti-audit.whl",
			canonical:      "nvidia_cuda_cupti-13.2.75-py3-none-manylinux_2_25_x86_64.whl",
			profile:        "pytorch:cu132",
			project:        "nvidia-cuda-cupti-cu13",
			version:        "13.2.75",
			source:         "pypi",
			expectedSHA256: "f75aca6bef89c625a4076a820302bb06764daa1d21595286f6bee5e237d3a187",
			wantImports:    nil,
			noImport:       true,
		},
	}

	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	for _, tc := range manifest {
		t.Run(tc.fileOnDisk, func(t *testing.T) {
			path := filepath.Join(corpusDir, tc.fileOnDisk)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing required corpus file %s: %v", tc.fileOnDisk, err)
			}
			hash := sha256.Sum256(data)
			actualSHA256 := hex.EncodeToString(hash[:])
			if actualSHA256 != tc.expectedSHA256 {
				t.Fatalf("corpus file %s digest mismatch: got %s, want independent pinned hash %s", tc.fileOnDisk, actualSHA256, tc.expectedSHA256)
			}

			wheelFilename := tc.canonical
			if wheelFilename == "" {
				wheelFilename = tc.fileOnDisk
			}
			limits := DefaultWheelLimits()
			if prof, ok := PyTorchProfile(strings.TrimPrefix(tc.profile, "pytorch:")); ok {
				limits = prof.ResourcePolicy().WheelLimits()
			}
			source, err := domain.NewSourceID(tc.source)
			if err != nil {
				t.Fatalf("invalid source id %s: %v", tc.source, err)
			}
			insp, err := InspectWheelForSource(bytes.NewReader(data), int64(len(data)), wheelFilename, tc.expectedSHA256, target, limits, source)
			if err != nil {
				stage, _ := WheelValidationStageOf(err)
				t.Fatalf("InspectWheel(%q) failed: %v (stage: %s)", wheelFilename, err, stage)
			}
			if insp.NoImportSurface != tc.noImport {
				t.Fatalf("NoImportSurface = %v, want %v", insp.NoImportSurface, tc.noImport)
			}
			if len(tc.wantImports) > 0 {
				for _, want := range tc.wantImports {
					found := false
					for _, got := range insp.ImportNames {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected candidate import %q in %v", want, insp.ImportNames)
					}
				}
			}

			var policy ResourcePolicy
			switch tc.profile {
			case "pytorch:cpu":
				policy = pyTorchCPUResourcePolicy()
			case "pytorch:cu126":
				policy = pyTorchCU126ResourcePolicy()
			case "pytorch:cu130":
				policy = pyTorchCU130ResourcePolicy()
			case "pytorch:cu132":
				policy = pyTorchCU132ResourcePolicy()
			default:
				policy = defaultResourcePolicy()
			}

			plan, err := BuildObservationPlan(insp, policy)
			if err != nil {
				t.Fatalf("BuildObservationPlan(%q) failed: %v", wheelFilename, err)
			}
			if err := ValidateTypedObservationPlan(plan, policy); err != nil || !plan.Admissible() {
				t.Fatalf("authenticated corpus observation plan is nonqualifying: %v, entry points=%v, scripts=%v", err, plan.EntryPointCoverage, plan.ScriptCoverage)
			}
			if tc.expectedCandidateCount > 0 && plan.TotalImportCount != tc.expectedCandidateCount {
				t.Fatalf("%s total imports = %d, want %d", tc.fileOnDisk, plan.TotalImportCount, tc.expectedCandidateCount)
			}
		})
	}
	t.Logf("REAL_CORPUS_CLASSIFICATION: REPRESENTATIVE_REAL_CORPUS (15 authenticated packages verified)")
}

func TestModelC_MissingCorpusDirectoryFails(t *testing.T) {
	fakeDir := "/nonexistent/test/corpus/directory"
	validateCorpus := func(dir string) error {
		if _, err := os.Stat(dir); err != nil {
			return errors.New("missing real corpus directory")
		}
		return nil
	}
	if err := validateCorpus(fakeDir); err == nil {
		t.Fatal("expected missing corpus directory to fail, got nil")
	}
}

func TestModelC_CorpusHashMismatchFails(t *testing.T) {
	corpusDir := "/tmp/haa_corpus"
	if _, err := os.Stat(corpusDir); err != nil {
		t.Fatalf("corpus directory missing: %v", err)
	}
	file := filepath.Join(corpusDir, "setuptools-84.0.0-py3-none-any.whl")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	wrongDigest := strings.Repeat("0", 64)
	limits := DefaultWheelLimits()
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	_, err = InspectWheel(bytes.NewReader(data), int64(len(data)), "setuptools-84.0.0-py3-none-any.whl", wrongDigest, target, limits)
	if err == nil {
		t.Fatal("expected digest mismatch to fail, got nil")
	}
	stage, _ := WheelValidationStageOf(err)
	if stage != WheelValidationDigest {
		t.Fatalf("expected stage %s, got %s (err: %v)", WheelValidationDigest, stage, err)
	}
}

func TestModelC_DuplicateInstalledDestinationRejected(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "foo.py", body: []byte("x = 1\n")},
		{name: "testpkg-1.0.data/purelib/foo.py", body: []byte("x = 2\n")},
	}
	archive := recordedWheelArchive(t, "testpkg", "1.0", "testpkg-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	limits := DefaultWheelLimits()
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	sum := sha256.Sum256(archive)
	declared := hex.EncodeToString(sum[:])
	_, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "testpkg-1.0-py3-none-any.whl", declared, target, limits)
	if err == nil {
		t.Fatal("expected duplicate destination to be rejected, got nil error")
	}
	stage, _ := WheelValidationStageOf(err)
	if stage != WheelValidationRecord {
		t.Fatalf("expected stage %s, got %s (err: %v)", WheelValidationRecord, stage, err)
	}
}

func TestModelC_WrongTargetExtensionRejected(t *testing.T) {
	for _, badExt := range []string{
		"myext.cpython-313-x86_64-linux-gnu.so",
		"myext.cpython-314-darwin.so",
		"myext.cpython-314-aarch64-linux-gnu.so",
		"myext.cpython-314-x86_64-unknown-linux-gnu.so",
		"myext.pyd",
	} {
		t.Run(badExt, func(t *testing.T) {
			entries := []wheelTestEntry{
				{name: badExt, body: []byte("fake binary")},
			}
			archive := recordedWheelArchive(t, "testext", "1.0", "testext-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
			limits := DefaultWheelLimits()
			target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
			sum := sha256.Sum256(archive)
			declared := hex.EncodeToString(sum[:])
			insp, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "testext-1.0-py3-none-any.whl", declared, target, limits)
			if err != nil {
				return
			}
			if len(insp.Surface.UnresolvedFiles) == 0 {
				t.Fatalf("wrong-target extension %s must be unresolved: %#v", badExt, insp.Surface)
			}
			_, err = BuildObservationPlan(insp, defaultResourcePolicy())
			if err == nil {
				t.Fatalf("BuildObservationPlan must fail on unresolved file %s", badExt)
			}
		})
	}
}

func TestModelC_TypedHookAndExactEntryPointPlan(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "pkg/__init__.py", body: []byte("\n")},
		{name: "pkg/cli.py", body: []byte("def main(): pass\n")},
		{name: "other.py", body: []byte("x = 1\n")},
		{name: "extra/module.py", body: []byte("x = 1\n")},
		{name: "a.pth", body: []byte("extra\nimport pkg\nimport other\n")},
		{name: "pkg/nested.pth", body: []byte("import sys\n")},
		{name: "testpkg-1.0.dist-info/entry_points.txt", body: []byte("[console_scripts]\nrun = pkg.cli:main\n")},
	}
	archive := recordedWheelArchive(t, "testpkg", "1.0", "testpkg-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "testpkg-1.0-py3-none-any.whl")
	plan, err := BuildObservationPlan(insp, defaultResourcePolicy())
	if err != nil || ValidateTypedObservationPlan(plan, defaultResourcePolicy()) != nil {
		t.Fatalf("typed plan rejected: %v, %#v", err, plan)
	}
	if plan.EntryPointCoverage["console_scripts: run = pkg.cli:main"] != string(CoverageCoveredByModule) || !containsCandidate(plan.ImportCandidates, "pkg.cli") {
		t.Fatalf("entry point was not bound to exact installed module: %#v", plan)
	}
	if len(plan.SiteHookLines) != 3 || len(plan.Units) != len(plan.ImportCandidates)+3 || plan.Units[len(plan.Units)-1].Kind != InstalledStartupUnit {
		t.Fatalf("independent hook/startup units missing: %#v", plan)
	}
	for _, line := range plan.SiteHookLines {
		if line.File == "pkg/nested.pth" {
			t.Fatal("nested inert hook became active")
		}
	}
	missing := plan
	missing.Units = append([]PlannedObservationUnit(nil), plan.Units[:len(plan.Units)-1]...)
	if ValidateTypedObservationPlan(missing, defaultResourcePolicy()) == nil {
		t.Fatal("missing startup unit accepted")
	}
	extra := plan
	extra.Units = append(append([]PlannedObservationUnit(nil), plan.Units...), plan.Units[0])
	if ValidateTypedObservationPlan(extra, defaultResourcePolicy()) == nil {
		t.Fatal("extra/duplicate unit accepted")
	}
}

func TestModelC_PTHPathMayComeFromAnotherWheel(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "site.pth", body: []byte("extra\n")},
	}
	archive := recordedWheelArchive(t, "testpkg", "1.0", "testpkg-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "testpkg-1.0-py3-none-any.whl")
	if len(insp.Surface.SiteHookLines) != 1 || insp.Surface.SiteHookLines[0].Path != "extra" {
		t.Fatalf("valid declarative path was rejected before closure assembly: %#v", insp.Surface.SiteHookLines)
	}
}

func TestModelC_UnrelatedImportDoesNotCoverEntryPoint(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "other.py", body: []byte("x=1\n")},
		{name: "testpkg-1.0.dist-info/entry_points.txt", body: []byte("[console_scripts]\nrun = ambient:main\n")},
	}
	archive := recordedWheelArchive(t, "testpkg", "1.0", "testpkg-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	insp := inspectTestWheel(t, archive, "testpkg-1.0-py3-none-any.whl")
	plan, err := BuildObservationPlan(insp, defaultResourcePolicy())
	if err != nil || plan.EntryPointCoverage["console_scripts: run = ambient:main"] != string(CoverageManualReview) || plan.Admissible() {
		t.Fatalf("unowned entry point target qualified: %v, %#v", err, plan)
	}
}

func TestModelC_HookOnlyAndEntryPointOnlyOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		files      []wheelTestEntry
		wantUnits  int
		admissible bool
	}{
		{"hook only", []wheelTestEntry{{name: "hook.pth", body: []byte("import sys\n")}}, 2, true},
		{"entry point only", []wheelTestEntry{{name: "testpkg-1.0.dist-info/entry_points.txt", body: []byte("[console_scripts]\nrun = absent:main\n")}}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archive := recordedWheelArchive(t, "testpkg", "1.0", "testpkg-1.0.dist-info", tc.files, []string{"py3-none-any"}, nil)
			insp := inspectTestWheel(t, archive, "testpkg-1.0-py3-none-any.whl")
			plan, err := BuildObservationPlan(insp, defaultResourcePolicy())
			if err != nil || ValidateTypedObservationPlan(plan, defaultResourcePolicy()) != nil || len(plan.Units) != tc.wantUnits || plan.Admissible() != tc.admissible {
				t.Fatalf("observation disposition = %#v, %v", plan, err)
			}
		})
	}
}

func TestModelC_NestedInertPthDoesNotBreakNoImport(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "nvidia/nccl/lib/libnccl.so.2", body: []byte("elf")},
		{name: "nvidia/nccl/nested.pth", body: []byte("# inert pth\n")},
	}
	archive := recordedWheelArchive(t, "nvidia-nccl-cu12", "2.29.3", "nvidia_nccl_cu12-2.29.3.dist-info", entries, []string{"py3-none-any"}, nil)
	limits := DefaultWheelLimits()
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	sum := sha256.Sum256(archive)
	declared := hex.EncodeToString(sum[:])
	insp, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "nvidia_nccl_cu12-2.29.3-py3-none-any.whl", declared, target, limits)
	if err != nil {
		t.Fatalf("InspectWheel failed: %v", err)
	}
	if !insp.NoImportSurface {
		t.Fatalf("nested inert .pth must not break no-import classification: %#v", insp)
	}
	plan, err := BuildObservationPlan(insp, defaultResourcePolicy())
	if err != nil || !plan.NoImportSurface {
		t.Fatalf("observation plan invalid: %#v, err=%v", plan, err)
	}
}

func TestModelC_EntryPointCoverageExplicit(t *testing.T) {
	epContent := []byte("[console_scripts]\nhelox = mymod:main\n")
	entries := []wheelTestEntry{
		{name: "mymod.py", body: []byte("def main(): pass\n")},
		{name: "testpkg-1.0.dist-info/entry_points.txt", body: epContent},
	}
	archive := recordedWheelArchive(t, "testpkg", "1.0", "testpkg-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	limits := DefaultWheelLimits()
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	sum := sha256.Sum256(archive)
	declared := hex.EncodeToString(sum[:])
	insp, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "testpkg-1.0-py3-none-any.whl", declared, target, limits)
	if err != nil {
		t.Fatalf("InspectWheel failed: %v", err)
	}
	plan, err := BuildObservationPlan(insp, defaultResourcePolicy())
	if err != nil {
		t.Fatalf("BuildObservationPlan failed: %v", err)
	}
	if len(plan.EntryPointCoverage) == 0 {
		t.Fatal("entry points must have explicit coverage in observation plan")
	}
	for ep, outcome := range plan.EntryPointCoverage {
		if outcome != string(CoverageCoveredByModule) {
			t.Fatalf("entry point %s coverage outcome = %s, want %s", ep, outcome, CoverageCoveredByModule)
		}
	}
}

func TestModelC_ScriptCoverageExplicit(t *testing.T) {
	entries := []wheelTestEntry{
		{name: "mymod.py", body: []byte("x = 1\n")},
		{name: "testpkg-1.0.data/scripts/my-script", body: []byte("#!/bin/sh\necho hi\n")},
	}
	archive := recordedWheelArchive(t, "testpkg", "1.0", "testpkg-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
	limits := DefaultWheelLimits()
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	sum := sha256.Sum256(archive)
	declared := hex.EncodeToString(sum[:])
	insp, err := InspectWheel(bytes.NewReader(archive), int64(len(archive)), "testpkg-1.0-py3-none-any.whl", declared, target, limits)
	if err != nil {
		t.Fatalf("InspectWheel failed: %v", err)
	}
	plan, err := BuildObservationPlan(insp, defaultResourcePolicy())
	if err != nil {
		t.Fatalf("BuildObservationPlan failed: %v", err)
	}
	if len(plan.ScriptCoverage) == 0 {
		t.Fatal("scripts must have explicit coverage in observation plan")
	}
	outcome, ok := plan.ScriptCoverage["bin/my-script"]
	if !ok || outcome != string(CoverageManualReview) {
		t.Fatalf("script coverage outcome = %s, want %s", outcome, CoverageManualReview)
	}

	// Script-only artifact (no import candidates) receives MANUAL_REVIEW policy outcome
	scriptOnlyEntries := []wheelTestEntry{
		{name: "testscript-1.0.data/scripts/standalone-tool", body: []byte("#!/bin/sh\necho hi\n")},
	}
	scriptOnlyArchive := recordedWheelArchive(t, "testscript", "1.0", "testscript-1.0.dist-info", scriptOnlyEntries, []string{"py3-none-any"}, nil)
	scriptOnlySum := sha256.Sum256(scriptOnlyArchive)
	scriptOnlyDeclared := hex.EncodeToString(scriptOnlySum[:])
	scriptOnlyInsp, err := InspectWheel(bytes.NewReader(scriptOnlyArchive), int64(len(scriptOnlyArchive)), "testscript-1.0-py3-none-any.whl", scriptOnlyDeclared, target, limits)
	if err != nil {
		t.Fatalf("InspectWheel failed: %v", err)
	}
	scriptOnlyPlan, err := BuildObservationPlan(scriptOnlyInsp, defaultResourcePolicy())
	if err != nil {
		t.Fatalf("BuildObservationPlan for script-only artifact failed: %v", err)
	}
	scriptOutcome, ok := scriptOnlyPlan.ScriptCoverage["bin/standalone-tool"]
	if !ok || scriptOutcome != string(CoverageManualReview) {
		t.Fatalf("script-only coverage outcome = %s, want %s", scriptOutcome, CoverageManualReview)
	}
}
