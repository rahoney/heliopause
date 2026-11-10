package npm

import (
	"strings"
	"testing"
)

func TestPNPMInstanceDirectoryUpstreamRegistryVectors(t *testing.T) {
	// Registry vectors from the pinned dep_path_to_filename tests. The input
	// subset deliberately excludes the upstream file/URL protocol branches.
	for _, test := range []struct{ key, expected string }{
		{"foo@1.0.0", "foo@1.0.0"},
		{"@scope/foo@1.0.0", "@scope+foo@1.0.0"},
		{"foo@1.0.0(bar@2.0.0)(baz@3.0.0)", "foo@1.0.0_bar@2.0.0_baz@3.0.0"},
		{"eslint-plugin-testing-library@7.7.0(eslint@9.35.0(jiti@2.6.1))(typescript@6.0.3)", "eslint-plugin-testing-library@7.7.0_eslint@9.35.0_jiti@2.6.1__typescript@6.0.3"},
		{"foo@1.0.0+build.5", "foo@1.0.0+build.5"},
		{"esbuild@0.0.0-dev+abc(foo@1.0.0)", "esbuild@0.0.0-dev+abc_foo@1.0.0"},
		{"foo@1.0.0(@types/node@24.0.0)", "foo@1.0.0_@types+node@24.0.0"},
	} {
		got, err := PNPMInstanceDirectory(test.key)
		if err != nil || got != test.expected {
			t.Fatalf("key %q: directory=%q expected=%q err=%v", test.key, got, test.expected, err)
		}
	}
}

func TestPNPMInstanceDirectoryShorteningAndInvalidSources(t *testing.T) {
	// Independent SHA256 vectors are generated from the escaped filename,
	// not from a registry digest or the raw parenthesized snapshot key.
	got, err := PNPMInstanceDirectory("foo@1.0.0-RC1")
	if err != nil || got != "foo@1.0.0-RC1_9941fd6db8b6bdee48e265001a59bdbe" {
		t.Fatal("case-sensitive shortening differs from pinned format", got, err)
	}
	key := "foo@1.0.0" + strings.Repeat("(bar@2.0.0)", 20)
	escaped := "foo@1.0.0" + strings.Repeat("_bar@2.0.0", 20)
	got, err = PNPMInstanceDirectory(key)
	if err != nil || got != escaped[:87]+"_1154c61958d1fb5c17df465a0bdd2c1d" || len(got) != PNPMVirtualStoreDirMaxLength {
		t.Fatal("long peer shortening differs from independent vector", got, err)
	}
	for _, key := range []string{"", "a", "/foo@1.0.0", "foo@latest", "file:packages/foo", "foo@https://example.test/a.tgz", "foo@work:1.0.0", "../foo@1.0.0", "foo@1.0.0()", "foo@1.0.0(bar@2.0.0", "foo@1.0.0(bar:2.0.0)"} {
		if _, err := PNPMInstanceDirectory(key); err == nil {
			t.Fatal("unsupported or malformed snapshot produced a directory", key)
		}
	}
}
