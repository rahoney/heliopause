package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipxClosureRequiresExplicitFrozenInputs(t *testing.T) {
	for _, mode := range []string{"relative", "missing-projects", "symlink-root", "extra-argument"} {
		t.Run(mode, func(t *testing.T) {
			input := t.TempDir()
			args := []string{"verify-pipx-closure", "--input", input}
			switch mode {
			case "relative":
				args[2] = "."
			case "symlink-root":
				link := filepath.Join(t.TempDir(), "supply")
				if err := os.Symlink(input, link); err != nil {
					t.Fatal(err)
				}
				args[2] = link
			case "extra-argument":
				args = append(args, "extra")
			}
			var out bytes.Buffer
			if err := run(args, &out); err == nil || out.Len() != 0 {
				t.Fatal("incomplete closure produced facts")
			}
		})
	}
}

func TestPipxUnsignedIdentityDoesNotGainPublisherClaims(t *testing.T) {
	digest := strings.Repeat("a", 64)
	body := `{"version":"1.9.2","wheel_sha256":"` + digest + `"}`
	v, d, err := readPipxUnsignedIdentity([]byte(body))
	if err != nil || v != "1.9.2" || d != digest {
		t.Fatal("bounded unsigned identity differs")
	}
	for _, bad := range []string{
		`null`, body + `{}`, strings.Replace(body, `"version":`, `"version":"1.0.0","version":`, 1),
		strings.Replace(body, `"version"`, `"Version"`, 1), strings.Replace(body, `"1.9.2"`, `"../escape"`, 1),
		strings.Replace(body, `"1.9.2"`, `"1.9.2-rc1"`, 1), strings.Replace(body, `"wheel_sha256"`, `"source_commit"`, 1),
		strings.Replace(body, `{`, `{"verified":true,`, 1), strings.Repeat(" ", 4097),
	} {
		if _, _, err := readPipxUnsignedIdentity([]byte(bad)); err == nil {
			t.Fatal("invalid unsigned identity accepted")
		}
	}
}
