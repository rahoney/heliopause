package pypi

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
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
