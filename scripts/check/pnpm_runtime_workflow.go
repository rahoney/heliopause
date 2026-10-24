package main

import "strings"

const pnpmRuntimeIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 go test -tags=realcorpus -v -timeout=3m ./internal/sandbox -run '^TestPNPMActual(Isolated(NativeStartup|EmptyLock|NodeQuery)|ReleaseArtifactProbe)$'`

// Check the helper's concrete command binding, not arbitrary Bash semantics.
// Bash syntax and the executed helper/cleanup are qualified separately.
func validatePNPMRuntimeScript(contents string) []string {
	counts := map[string]int{}
	for _, line := range strings.Split(contents, "\n") {
		counts[strings.TrimSpace(line)]++
	}
	var findings []string
	for _, command := range []string{
		"set -euo pipefail",
		"test ! -e /usr/lib/heliopause",
		"test ! -L /usr/lib/heliopause",
		"trap pnpm_cleanup EXIT",
		"go run ./scripts/generate-runtime-lock.go -check",
		`pnpm_url=$(jq -er '.pnpm_tool.archive.url' scripts/runtimes.lock.json)`,
		`pnpm_size=$(jq -er '.pnpm_tool.archive.size' scripts/runtimes.lock.json)`,
		`curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 120 --max-filesize "$pnpm_size" --output "$pnpm_work/archive.tar.gz" "$pnpm_url"`,
		`go run ./scripts/update-runtime materialize-pnpm --root "$PWD" --archive "$pnpm_work/archive.tar.gz" --output "$pnpm_work/pnpm-linux-amd64" --notices-output "$pnpm_work/pnpm-NOTICE.txt"`,
		"sudo -n mkdir -m 0755 -- /usr/lib/heliopause",
		pnpmRuntimeIntegrationCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "PNPM runtime helper requires one command: "+command)
		}
	}
	return findings
}
