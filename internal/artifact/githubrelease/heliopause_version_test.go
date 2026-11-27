package githubrelease

import (
	"strings"
	"testing"
)

func TestHeliopauseVersionRejectsMutableAliasesAndNonExactInputs(t *testing.T) {
	for _, version := range []string{"v0.1.0", "v1.2.3", "v1.2.3-rc.1", "v1.2.3+build.1"} {
		if err := ValidateHeliopauseVersion(version); err != nil {
			t.Fatal(version, err)
		}
	}
	for _, version := range []string{"", "latest", "main", "v1", "v1.2", "1.2.3", "V1.2.3", " v1.2.3", "v1.2.3\n", "https://example.test/v1.2.3", "v1.2.3#asset", "v1.2.3-" + strings.Repeat("a", 128)} {
		if err := ValidateHeliopauseVersion(version); err == nil {
			t.Fatal("non-exact input accepted", version)
		}
	}
}
