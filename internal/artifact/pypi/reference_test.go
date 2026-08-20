package pypi

import (
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

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
