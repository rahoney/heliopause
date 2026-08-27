package pypi

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
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
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for name, body := range files {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
