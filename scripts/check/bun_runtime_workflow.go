package main

import "strings"

const bunNativeIntegrationCommand = `HELOX_BUN_NATIVE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^(TestBunPublicNativeResolverIntegration|TestBunActualNativeProjectSelectionBoundary)$'`
const bunProjectQuarantineCommand = `HELOX_BUN_PROJECT_QUARANTINE=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunPublicProjectQuarantineIntegration$'`

// Exact concrete supply/consumer binding; actual Bash execution is separate.
func validateBunRuntimeScript(contents string) []string {
	counts := map[string]int{}
	var active []string
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		counts[line]++
		if line != "" && !strings.HasPrefix(line, "#") {
			active = append(active, line)
		}
	}
	var findings []string
	for _, command := range []string{
		"set -euo pipefail", `test "$#" -eq 0`, `test "$(id -u)" != 0`,
		`test -n "${RUNNER_TEMP:-}"`, `test "${RUNNER_TEMP:0:1}" = /`,
		"test ! -e /usr/lib/heliopause", "test ! -L /usr/lib/heliopause", "trap bun_cleanup EXIT",
		"go run ./scripts/generate-runtime-lock.go -check",
		`bun_url=$(jq -er '.bun_tool.archive.url' scripts/runtimes.lock.json)`,
		`bun_size=$(jq -er '.bun_tool.archive.size' scripts/runtimes.lock.json)`,
		`curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 120 --max-filesize "$bun_size" --output "$bun_work/archive.zip" "$bun_url"`,
		`go run ./scripts/update-runtime materialize-bun --root "$PWD" --archive "$bun_work/archive.zip" --output "$bun_work/bun-linux-amd64" --notices-output "$bun_work/bun-NOTICE.txt"`,
		"sudo -n mkdir -m 0755 -- /usr/lib/heliopause",
		`sudo -n install -m 0555 "$bun_work/bun-linux-amd64" /usr/lib/heliopause/versions/v1.0.0-bun-ci/bun-linux-amd64`,
		"sudo -n ln -s versions/v1.0.0-bun-ci /usr/lib/heliopause/current",
		bunNativeIntegrationCommand, bunProjectQuarantineCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "Bun runtime helper requires one command: "+command)
		}
	}
	scope := strings.Join([]string{"(", "cd internal/sandbox", bunNativeIntegrationCommand, bunProjectQuarantineCommand, ")"}, "\n")
	if strings.Count(strings.Join(active, "\n"), scope) != 1 {
		findings = append(findings, "Bun actual consumers require one matching package subshell")
	}
	return findings
}

func validateBunInspectionClientOrder(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return nil
	}
	sequence := strings.Join([]string{"bash scripts/ci-pnpm-runtime.sh project", `haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"`, "haa_start_policy_helper", "bash scripts/ci-bun-runtime.sh"}, "\n")
	for _, lines := range blocks {
		if strings.Count(strings.Join(lines, "\n"), sequence) == 1 {
			return nil
		}
	}
	return []string{"Bun actual requires restored tagged sandbox client and authenticated helper in the owning lifecycle"}
}
