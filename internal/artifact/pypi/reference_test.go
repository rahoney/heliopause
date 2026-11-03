package pypi

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestComparePublicVersionsUsesPEP440Ordering(t *testing.T) {
	ordered := []string{"1.dev0", "1.0.dev456", "1.0a1", "1.0a2.dev456", "1.0a12.dev456", "1.0a12", "1.0b1.dev456", "1.0b2", "1.0b2.post345.dev456", "1.0b2.post345", "1.0rc1.dev456", "1.0rc1", "1.0", "1.0.post456.dev34", "1.0.post456", "1.0.15", "1.1.dev1", "2!0.0"}
	for i, left := range ordered {
		for j, right := range ordered {
			order, err := ComparePublicVersions(left, right)
			if err != nil || i < j && order >= 0 || i == j && order != 0 || i > j && order <= 0 {
				t.Fatalf("incorrect public version order %q vs %q: %d %v", left, right, order, err)
			}
		}
	}
	for _, pair := range [][2]string{{"v01.0.0", "1"}, {"0!1.0", "1.0.0"}, {"1.0-c2", "1.0rc2"}, {"1.0-2", "1.0.post2"}, {"1.0dev", "1.0.dev0"}} {
		if order, err := ComparePublicVersions(pair[0], pair[1]); err != nil || order != 0 {
			t.Fatal("normalization equality lost", pair, order, err)
		}
	}
	if order, err := ComparePublicVersions("1!1.0", "99999999999999999999999999999999999999999999999999.0"); err != nil || order <= 0 {
		t.Fatal("epoch did not dominate release", order, err)
	}
	if order, err := ComparePublicVersions("1.0.post9999999999999999999999999999999999999999", "1.0.post10000000000000000000000000000000000000000"); err != nil || order >= 0 {
		t.Fatal("numeric version overflow or lexical comparison", order, err)
	}
	for _, invalid := range []string{"", "1.0+cpu", ">=1.0", "1.0.*", " 1.0", "1.0\n", strings.Repeat("1", 257)} {
		if _, err := ComparePublicVersions(invalid, "1.0"); err == nil {
			t.Fatal("unsupported public version accepted", invalid)
		}
		if _, err := ComparePublicVersions("1.0", invalid); err == nil {
			t.Fatal("unsupported right public version accepted", invalid)
		}
	}
}

func TestParseReferenceNormalizesPublicProjectAndVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		locator string
	}{
		{input: "Django", locator: "django"},
		{input: "zope.interface@v05.001RC", locator: "zope-interface@5.1rc0"},
		{input: "Example_Package@1.0-r004.dev", locator: "example-package@1.0.post4.dev0"},
		{input: "project@1!001.00b.02.post-003.dev004", locator: "project@1!1.0b2.post3.dev4"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			reference, err := ParseReference(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if reference.Source().String() != "pypi" || reference.Locator() != test.locator {
				t.Fatalf("reference = %q/%q", reference.Source().String(), reference.Locator())
			}
		})
	}
}

func TestParseReferenceRejectsUnsupportedOrAmbiguousInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"", " project", "project ", "project[extra]", "project@>=1", "project@1.0+local",
		"project; python_version >= '3.14'", "project@https://example.test/project.whl", "project@1.0@2",
		"project/child", "project\\child", "project@1.0-",
	} {
		if _, err := ParseReference(input); err == nil {
			t.Fatalf("ParseReference(%q) error = nil", input)
		}
	}
}

func TestRequestedReferenceFieldsRequireValidatedPyPIReference(t *testing.T) {
	t.Parallel()

	reference, err := ParseReference("Example_Package@v1.0RC")
	if err != nil {
		t.Fatal(err)
	}
	project, err := RequestedProject(reference)
	if err != nil || project != "example-package" {
		t.Fatalf("RequestedProject() = %q, %v", project, err)
	}
	version, present, err := RequestedVersion(reference)
	if err != nil || !present || version != "1.0rc0" {
		t.Fatalf("RequestedVersion() = %q, %v, %v", version, present, err)
	}

	source, err := domain.NewSourceID("npm")
	if err != nil {
		t.Fatal(err)
	}
	nonPyPI, err := domain.NewArtifactReference(source, "example")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RequestedProject(nonPyPI); err == nil {
		t.Fatal("RequestedProject(non-PyPI) error = nil")
	}
}

func FuzzParseReference(f *testing.F) {
	for _, input := range []string{"Django", "Example_Package@v1.0RC", "bad@>=1", "name@1.0+local"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		reference, err := ParseReference(input)
		if err != nil {
			return
		}
		if _, err := ParseReference(reference.Locator()); err != nil {
			t.Fatalf("canonical locator %q no longer parses: %v", reference.Locator(), err)
		}
	})
}

func TestCompareInstalledVersionsPreservesLocalOrdering(t *testing.T) {
	ordered := []string{"1.0rc1+cpu", "1.0", "1.0+abc", "1.0+abc.1", "1.0+abc.2", "1.0+abc.10", "1.0+cpu", "1.0+1", "1.0+2", "1.0.post1", "1.1"}
	for i, left := range ordered {
		for j, right := range ordered {
			order, err := CompareInstalledVersions(left, right)
			if err != nil || i < j && order >= 0 || i == j && order != 0 || i > j && order <= 0 {
				t.Fatal(left, right, order, err)
			}
		}
	}
	for _, pair := range [][2]string{{"1.0+ABC-001", "1+abc.1"}, {"1.0+CPU", "1.0.0+cpu"}} {
		if order, err := CompareInstalledVersions(pair[0], pair[1]); err != nil || order != 0 {
			t.Fatal(pair, order, err)
		}
	}
	for _, invalid := range []string{"1.0+cpu.", "1.0+cpu..1", "1.0+cpu-", "1.0+cpu+1", "1.0+?", "1.0+cpu\n", strings.Repeat("1", 257)} {
		if _, err := CompareInstalledVersions(invalid, "1"); err == nil {
			t.Fatal("invalid local version accepted", invalid)
		}
		if _, err := CompareInstalledVersions("1", invalid); err == nil {
			t.Fatal("invalid right local version accepted", invalid)
		}
	}
	if _, err := ComparePublicVersions("1+cpu", "1"); err == nil {
		t.Fatal("public-only comparison contract widened")
	}
}
