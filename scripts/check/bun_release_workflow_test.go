package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBunReleaseSupplyRequiresOrderedVerifiedProducer(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", releaseWorkflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(body)
	if findings := validateBunReleaseSupply(baseline); len(findings) != 0 {
		t.Fatal(findings)
	}
	command := `go run ./scripts/update-runtime materialize-bun --root "$GITHUB_WORKSPACE" --archive "$bun_archive" --output "$dist/bun-linux-amd64" --notices-output "$dist/bun-NOTICE.txt"`
	for name, change := range map[string]func(string) string{
		"missing-verification":   func(s string) string { return strings.Replace(s, command, "true", 1) },
		"commented-verification": func(s string) string { return strings.Replace(s, command, "# "+command, 1) },
		"wrong-owner":            func(s string) string { return strings.Replace(s, ".bun_tool.archive.url", ".pnpm_tool.archive.url", 1) },
		"missing-bound":          func(s string) string { return strings.Replace(s, `--max-filesize "$bun_size" `, "", 1) },
		"duplicate-verification": func(s string) string { return strings.Replace(s, command, command+"\n          "+command, 1) },
		"missing-notices-binding": func(s string) string {
			return strings.Replace(s, `--asset "$dist/bun-NOTICE.txt"`, "# omitted Bun notice", 1)
		},
		"conditional-step": func(s string) string {
			return strings.Replace(s, "- name: Build deterministic native release assets", "- name: Build deterministic native release assets\n        if: false", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := change(baseline)
			if candidate == baseline {
				t.Fatal("fixture did not change")
			}
			if len(validateBunReleaseSupply(candidate)) == 0 {
				t.Fatal("unverified or unbound Bun supply accepted")
			}
		})
	}
}
