package content

import "testing"

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
