package pypi

import (
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestPipxNativeEnrollmentStaticRecipeIsNarrowerThanMetadataReplay(t *testing.T) {
	for _, tc := range []struct {
		arguments []string
		supported bool
	}{
		{nil, false},
		{[]string{"--no-build", "--no-sources"}, false},
		{[]string{"--no-build", "--link-mode=copy"}, false},
		{[]string{"--no-sources", "--link-mode", "copy"}, false},
		{[]string{"--no-build", "--no-sources", "--link-mode=copy"}, true},
		{[]string{"--no-build", "--no-sources", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--link-mode", "copy", "--index-url", "https://pypi.org/simple/"}, true},
	} {
		receipt, files := pipxToolUpgradeFixture(t, tc.arguments)
		input, err := NewPipxToolUpgradeInput(receipt, files, runtimeidentity.PythonVersion)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := input.Control()
		if input.NativeUVEnrollmentRecipeSupported() != tc.supported {
			t.Fatal("native recipe format widened", tc.arguments)
		}
		after, _ := input.Control()
		if before.Digest() != after.Digest() {
			t.Fatal("static recipe gate rewrote original")
		}
	}
}
