package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipxReleaseSupplyRequiresFrozenVerifiedDataProducer(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", releaseWorkflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(body)
	if findings := validatePipxReleaseSupply(baseline); len(findings) != 0 {
		t.Fatal(findings)
	}
	fetch := `go run ./scripts/update-runtime fetch-pipx-supply --root "$GITHUB_WORKSPACE" --output "$pipx_input"`
	materialize := `go run ./scripts/update-runtime materialize-pipx --root "$GITHUB_WORKSPACE" --input "$pipx_input" --output "$dist/pipx-runtime-linux-amd64.zip" --notices-output "$dist/pipx-NOTICE.txt"`
	for name, change := range map[string]func(string) string{
		"missing-fetch":   func(s string) string { return strings.Replace(s, fetch, "true", 1) },
		"commented-fetch": func(s string) string { return strings.Replace(s, fetch, "# "+fetch, 1) },
		"duplicate-fetch": func(s string) string { return strings.Replace(s, fetch, fetch+"\n          "+fetch, 1) },
		"swapped-producers": func(s string) string {
			s = strings.Replace(s, fetch, "PLACEHOLDER", 1)
			s = strings.Replace(s, materialize, fetch, 1)
			return strings.Replace(s, "PLACEHOLDER", materialize, 1)
		},
		"missing-materializer": func(s string) string { return strings.Replace(s, materialize, "true", 1) },
		"candidate-materializer": func(s string) string {
			return strings.Replace(s, materialize, strings.Replace(materialize, "materialize-pipx ", "materialize-pipx-candidate ", 1), 1)
		},
		"source-output": func(s string) string {
			return strings.Replace(s, `pipx_input="$RUNNER_TEMP/helox-pipx-supply"`, `pipx_input="$GITHUB_WORKSPACE/pipx-supply"`, 1)
		},
		"other-input": func(s string) string { return strings.Replace(s, `--input "$pipx_input"`, `--input "$other_input"`, 1) },
		"missing-zip-binding": func(s string) string {
			return strings.Replace(s, `--asset "$dist/pipx-runtime-linux-amd64.zip"`, "# omitted ZIP", 1)
		},
		"missing-notices-binding": func(s string) string {
			return strings.Replace(s, `--asset "$dist/pipx-NOTICE.txt"`, "# omitted notice", 1)
		},
		"conditional-step": func(s string) string {
			return strings.Replace(s, "- name: Build deterministic native release assets", "- name: Build deterministic native release assets\n        if: false", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := change(baseline)
			if candidate == baseline {
				t.Fatal("fixture unchanged")
			}
			if len(validatePipxReleaseSupply(candidate)) == 0 {
				t.Fatal("unverified/unbound supply accepted")
			}
		})
	}
}
