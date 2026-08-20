package pypi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"testing"
)

func TestInspectSdistAcceptsBoundedPEP517Source(t *testing.T) {
	archive := makeSdist(t, map[string][]byte{
		"example-1.0/PKG-INFO":            []byte("Metadata-Version: 2.4\nName: example\nVersion: 1.0\nRequires-Dist: runtime>=1\n"),
		"example-1.0/pyproject.toml":      []byte("[build-system]\nrequires = [\"setuptools>=70\", \"wheel\"]\nbuild-backend = \"setuptools.build_meta\"\n"),
		"example-1.0/example/__init__.py": []byte(""),
	})
	sum := sha256.Sum256(archive)
	inspection, err := InspectSdist(bytes.NewReader(archive), "example-1.0.tar.gz", hex.EncodeToString(sum[:]), DefaultSdistLimits())
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Project != "example" || inspection.Version != "1.0" || inspection.BuildBackend != "setuptools.build_meta" || inspection.BuildConfigSHA256 == "" || !slices.Equal(inspection.BuildRequirements, []string{"setuptools", "wheel"}) || !slices.Equal(inspection.BuildRequirementSpecs, []string{"setuptools>=70", "wheel"}) {
		t.Fatalf("inspection = %#v", inspection)
	}
}

func TestInspectSdistRejectsUnsafeAndUnsupportedBuildMetadata(t *testing.T) {
	tests := []struct {
		name  string
		files map[string][]byte
	}{
		{"path escape", map[string][]byte{"../PKG-INFO": []byte("bad")}},
		{"legacy build", map[string][]byte{"example-1.0/PKG-INFO": []byte("Name: example\nVersion: 1.0\n"), "example-1.0/pyproject.toml": []byte("[build-system]\nrequires = [\"wheel\"]\n")}},
		{"in tree backend", map[string][]byte{"example-1.0/PKG-INFO": []byte("Name: example\nVersion: 1.0\n"), "example-1.0/pyproject.toml": []byte("[build-system]\nrequires = [\"wheel\"]\nbuild-backend = \"backend\"\nbackend-path = [\"build\"]\n")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := makeSdist(t, test.files)
			sum := sha256.Sum256(archive)
			if _, err := InspectSdist(bytes.NewReader(archive), "example-1.0.tar.gz", hex.EncodeToString(sum[:]), DefaultSdistLimits()); err == nil {
				t.Fatal("unsafe sdist accepted")
			}
		})
	}
}

func TestInspectSdistRejectsNonPAXArchive(t *testing.T) {
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	tw := tar.NewWriter(gz)
	for name, body := range map[string][]byte{
		"example-1.0/PKG-INFO":       []byte("Name: example\nVersion: 1.0\n"),
		"example-1.0/pyproject.toml": []byte("[build-system]\nrequires = [\"wheel\"]\nbuild-backend = \"setuptools.build_meta\"\n"),
	} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(out.Bytes())
	if _, err := InspectSdist(bytes.NewReader(out.Bytes()), "example-1.0.tar.gz", hex.EncodeToString(sum[:]), DefaultSdistLimits()); err == nil {
		t.Fatal("non-PAX sdist accepted")
	}
}

func FuzzInspectSdistNoPanic(f *testing.F) {
	f.Add([]byte("not an sdist"))
	f.Fuzz(func(t *testing.T, body []byte) {
		sum := sha256.Sum256(body)
		_, _ = InspectSdist(bytes.NewReader(body), "example-1.0.tar.gz", hex.EncodeToString(sum[:]), DefaultSdistLimits())
	})
}

func makeSdist(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg, Format: tar.FormatPAX, PAXRecords: map[string]string{"comment": "heliopause"}}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
