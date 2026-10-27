package main

import "strings"

const uvToolNativeCommand = `HELOX_UV_TOOL_RESOLVER_INTEGRATION=1 HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestUVPublicTool(Install|Upgrade)ResolverIntegration$'`
const uvToolBootstrapCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_UV_TOOL_INSTALL_CLI_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_CLI_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE=1 HELOX_UV_TOOL_NATIVE_ADOPTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=12m -test.run '^(TestLinuxUVTool(Install|Upgrade|NativeAdoption)CLIIntegration|TestUVPublicTool(ApprovedUpgrade|UpgradeEvidenceFailure)Integration)$'`

func validateUVToolsScript(contents string) []string {
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
		"set -euo pipefail", `0:|1:bootstrap) ;;`, `*) echo 'unsupported UV tools CI profile' >&2; exit 2 ;;`,
		`test "$(id -u)" != 0`, "trap uv_tools_cleanup EXIT", "go run ./scripts/generate-runtime-lock.go -check",
		`uv_tools_image=$(jq -er '.python_image.reference' scripts/runtimes.lock.json)`,
		`uv_tools_container=$(docker create --pull never --network none --read-only --cap-drop ALL --label "io.heliopause.ci.uv-tools-data=$uv_tools_owner" --entrypoint /bin/false "$uv_tools_image")`,
		`docker cp "$uv_tools_container:/usr/local/bin/python3.14" "$uv_tools_work/python3.14"`,
		`chmod 0400 "$uv_tools_work/python3.14"`, `uv_tools_sha=$(sha256sum "$uv_tools_work/python3.14" | cut -d ' ' -f 1)`,
		uvToolNativeCommand, uvToolBootstrapCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "UV tools helper requires one command: "+command)
		}
	}
	joined := strings.Join(active, "\n")
	for _, scope := range [][]string{{"(", "cd internal/sandbox", uvToolNativeCommand, ")", "exit 0"}, {"uv_tools_owned", "uv_tools_remove", "(", "cd internal/bootstrap", uvToolBootstrapCommand, ")"}} {
		if strings.Count(joined, strings.Join(scope, "\n")) != 1 {
			findings = append(findings, "UV tools consumer requires its package and owned data cleanup scope")
		}
	}
	return findings
}

func validateUVToolsClientOrder(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return nil
	}
	bootstrap := strings.Join([]string{`haa_install_ci_client "$RUNNER_TEMP/helox-pnpm-project-test"`, "haa_start_policy_helper", "bash scripts/ci-uv-tools.sh bootstrap", `haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"`, "haa_start_policy_helper"}, "\n")
	native := strings.Join([]string{"(", "cd internal/sandbox", uvPublicResolverIntegrationCommand, ")", "bash scripts/ci-uv-tools.sh"}, "\n")
	for _, lines := range blocks {
		joined := strings.Join(lines, "\n")
		if strings.Count(joined, bootstrap) == 1 && strings.Count(joined, native) == 1 {
			return nil
		}
	}
	return []string{"UV tools qualification requires its exact protected clients, package scopes, helper start and sandbox restore"}
}
