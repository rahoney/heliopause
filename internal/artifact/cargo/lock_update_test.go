package cargo

import (
	"strings"
	"testing"
)

func TestLockUpdateSelectorUsesOneOriginalPublicIdentity(t *testing.T) {
	_, lock := cargoActualMetadataFixture(t)
	for _, selector := range []string{"", "itoa", "itoa@1.0.17"} {
		if err := ValidateUpdateSelector(selector, lock); err != nil {
			t.Fatalf("valid selector %q: %v", selector, err)
		}
	}
	for _, selector := range []string{"missing", "itoa@1.0.16", "haa_schema_fixture", "../itoa", "https://example.invalid/itoa", "itoa@1.0", "itoa@1.0.17@foreign", "--precise", "itoa\x00", "itoa,serde"} {
		if err := ValidateUpdateSelector(selector, lock); err == nil {
			t.Fatalf("unsupported selector %q accepted", selector)
		}
	}
	ambiguous := "version=4\n[[package]]\nname='itoa'\nversion='1.0.16'\nsource='" + registrySourceURL + "'\nchecksum='" + strings.Repeat("a", 64) + "'\n[[package]]\nname='itoa'\nversion='1.0.17'\nsource='" + registrySourceURL + "'\nchecksum='" + strings.Repeat("b", 64) + "'\n"
	if ValidateUpdateSelector("itoa", []byte(ambiguous)) == nil || ValidateUpdateSelector("itoa@1.0.17", []byte(ambiguous)) != nil {
		t.Fatal("ambiguous name or exact old-version disambiguation was lost")
	}
	localCollision := ambiguous + "[[package]]\nname='itoa'\nversion='1.0.17'\n"
	if ValidateUpdateSelector("itoa@1.0.17", []byte(localCollision)) == nil {
		t.Fatal("local/public package-id ambiguity accepted")
	}
	for _, invalid := range [][]byte{nil, []byte("invalid [["), []byte(strings.ReplaceAll(ambiguous, registrySourceURL, "git+https://example.invalid/repository")), []byte(strings.ReplaceAll(ambiguous, strings.Repeat("a", 64), ""))} {
		if ValidateUpdateSelector("", invalid) == nil {
			t.Fatal("complete update accepted an uncertain original lock")
		}
	}
}
