package githubrelease

import (
	"strings"
	"testing"
)

func TestParseReferenceNormalizesExactPublicSelector(t *testing.T) {
	t.Parallel()
	reference, err := ParseReference("OpenAI/Heliopause@v1.2.3#heliopause-linux-amd64.tar.gz")
	if err != nil || reference.Source().String() != SourceName || reference.Locator() != "openai/heliopause@v1.2.3#heliopause-linux-amd64.tar.gz" {
		t.Fatalf("ParseReference() = %#v, %v", reference, err)
	}
}

func TestParseReferenceRejectsAmbiguousOrUnsafeSelector(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"owner/repo", "owner/repo@latest#asset", "owner/repo@v1#", "owner/repo@v1#/asset", "owner/repo@v1#../asset", "owner/repo@v1#asset?download=1", "owner/repo@v1#asset name", "https://github.com/o/r@v1#a", "owner/repo@v1#asset#other", "owner/repo@v1@other#asset", "owner/repo@v1/#asset", "owner/repo@v1#asset\\name"} {
		if _, err := ParseReference(value); err == nil {
			t.Fatalf("ParseReference(%q) error = nil", value)
		}
	}
}

func FuzzParseReference(f *testing.F) {
	for _, seed := range []string{"owner/repo@v1.0.0#tool", "OpenAI/Heliopause@release/2026#tool.tar.gz", "owner/repo@v1#asset"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		reference, err := ParseReference(value)
		if err != nil {
			return
		}
		if _, err := ParseReference(reference.Locator()); err != nil || strings.Contains(reference.Locator(), " ") {
			t.Fatalf("normalized reference %q is not stable: %v", reference.Locator(), err)
		}
	})
}
