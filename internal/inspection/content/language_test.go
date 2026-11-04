package content

import (
	"strings"
	"testing"
)

func TestSourceLanguageIsOnlyAParserSelection(t *testing.T) {
	for name, want := range map[string]string{
		"src/a.js": "javascript", "a.mjs": "javascript", "a.cjs": "javascript", "a.jsx": "javascript",
		"a.ts": "typescript", "a.mts": "typescript", "a.cts": "typescript", "a.tsx": "typescript",
		"a.py": "python", "a.pyi": "python", "a.go": "go", "a.rs": "rust", "a.tf": "hcl", "a.hcl": "hcl",
		"a.tf.json": "", "a.py.gz": "", "a.js.map": "", "a.so": "", "a.sh": "", "a": "",
	} {
		if got := SourceLanguage(name); got != want {
			t.Errorf("%q: got %q, want %q", name, got, want)
		}
	}
}

func TestUnsupportedSourceDoesNotBecomeAbsentSource(t *testing.T) {
	for _, test := range []struct {
		name, prefix string
		mode         uint32
		unsupported  bool
	}{
		{"helper.sh", "echo DATA", 0, true}, {"helper", "#!/bin/sh\n", 0, true},
		{"program", "opaque text", 0o755, true}, {"native.c", "int x;", 0, true},
		{"source.JS", "", 0, true}, {"README.md", "DATA", 0, false},
		{"source.js", "", 0o755, false}, {"program", "#!/usr/bin/env node\n", 0o755, false},
	} {
		if UnsupportedSource(test.name, []byte(test.prefix), test.mode) != test.unsupported {
			t.Fatal("source limitation classification was lost", test.name)
		}
	}
}

func TestSourceLanguageRecognizesOnlyBoundedInterpreterHints(t *testing.T) {
	for prefix, language := range map[string]string{
		"#!/usr/bin/env node\nconsole.log('x')":        "javascript",
		"#!/usr/bin/env -S node --no-warnings\n":       "javascript",
		"#!/usr/bin/env /usr/bin/node\n":               "javascript",
		"#!/usr/bin/nodejs\r\n":                        "javascript",
		"#!python\npass":                               "python",
		"#!/usr/bin/python3.15\npass":                  "python",
		"#!/usr/bin/env python3\npass":                 "python",
		"#!/usr/bin/python-malicious\n":                "",
		"#!/usr/bin/python.\n":                         "",
		"#!/usr/bin/env\n":                             "",
		"#!/usr/bin/env -S\n":                          "",
		"#!/bin/sh\n":                                  "",
		"#!\n":                                         "",
		"plain data\n#!/usr/bin/node":                  "",
		strings.Repeat(" ", 512) + "#!/usr/bin/node\n": "",
	} {
		if got := SourceLanguage("bin/tool", []byte(prefix)); got != language {
			t.Errorf("prefix=%q: got %q, want %q", prefix, got, language)
		}
	}
	if SourceLanguage("component.tsx", []byte("#!/usr/bin/env node\n")) != "typescript" {
		t.Fatal("interpreter hint erased the existing syntax selection")
	}
}
