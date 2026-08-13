package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckMarkdownTreeAcceptsHealthyFixture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "README.md"), "[guide](docs/guide.md#usage)\n\n```text\n[ignored](missing.md)\n```\n")
	writeTestFile(t, filepath.Join(root, "docs", "guide.md"), "# Usage\n\n[external](https://example.com)\n\n[local]: ../README.md\n")
	if err := checkMarkdownTree(root); err != nil {
		t.Fatalf("checkMarkdownTree error: %v", err)
	}
}

func TestCheckMarkdownTreeRejectsBrokenFixtures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		contents string
		want     string
	}{
		"broken link":    {contents: "[missing](missing.md)\n", want: "does not exist"},
		"unclosed fence": {contents: "```go\npackage example\n", want: "unclosed fenced code block"},
		"absolute Unix":  {contents: "[local](/Users/example/secret.md)\n", want: "absolute local link"},
		"absolute file":  {contents: "[local](file:///tmp/secret.md)\n", want: "absolute local link"},
		"absolute Win":   {contents: "[local](C:\\Users\\example\\secret.md)\n", want: "absolute local link"},
		"outside tree":   {contents: "[outside](../outside.md)\n", want: "leaves the source tree"},
		"reference link": {contents: "[missing]: missing.md\n", want: "does not exist"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "README.md"), test.contents)
			err := checkMarkdownTree(root)
			var failure *checkFailure
			if !errors.As(err, &failure) || failure.class != findingFailure {
				t.Fatalf("checkMarkdownTree error = %v, want finding failure", err)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("checkMarkdownTree error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestFenceMarkerRequiresMatchingClosure(t *testing.T) {
	t.Parallel()

	marker, length, closing := fenceMarker("````go", 0, 0)
	if marker != '`' || length != 4 || closing {
		t.Fatalf("opening fence = %q, %d, %t", marker, length, closing)
	}
	if _, _, closing = fenceMarker("```", marker, length); closing {
		t.Fatal("shorter fence closed the block")
	}
	if _, _, closing = fenceMarker("````", marker, length); !closing {
		t.Fatal("matching fence did not close the block")
	}
}
