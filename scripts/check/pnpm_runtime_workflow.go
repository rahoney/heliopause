package main

import "strings"

const pnpmRuntimeIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 go test -tags=realcorpus -v -timeout=3m ./internal/sandbox -run '^TestPNPMActual(Isolated(NativeStartup|EmptyLock|NodeQuery)|ReleaseArtifactProbe)$'`

const pnpmProjectIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMActualProductionProjectInspection$'`
const pnpmControlledIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMActualProductionInstanceExecution$/^controlled-(success|exit37|long-name)$'`
const pnpmPublicQuarantineCommand = `HELOX_PNPM_PUBLIC_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMPublicNativeResolverIntegration$'`
const pnpmSandboxTestBuildCommand = `go test -c -tags=realcorpus -o "$RUNNER_TEMP/helox-sandbox-test" ./internal/sandbox`

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
		pnpmProjectIntegrationCommand,
		pnpmControlledIntegrationCommand,
		pnpmPublicQuarantineCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "PNPM runtime helper requires one command: "+command)
		}
	}
	// Same fixed package-subshell convention as the UV/Cargo compiled consumers.
	// Source fixture access belongs to this command, never the supply commands.
	var active []string
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			active = append(active, line)
		}
	}
	scoped := 0
	for i := 2; i+1 < len(active); i++ {
		if active[i] == pnpmControlledIntegrationCommand && active[i-2] == "(" && active[i-1] == "cd internal/sandbox" && active[i+1] == ")" {
			scoped++
		}
	}
	if scoped != 1 {
		findings = append(findings, "PNPM controlled fixture requires one matching package subshell")
	}
	return findings
}

func validatePNPMInspectionClientOrder(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return nil
	}
	sequence := []string{pnpmSandboxTestBuildCommand, `haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"`, "haa_start_policy_helper", "bash scripts/ci-pnpm-runtime.sh"}
	for _, lines := range blocks {
		next := 0
		for _, line := range lines {
			if line == "bash scripts/ci-pnpm-runtime.sh" && next != len(sequence)-1 {
				return []string{"PNPM inspection must follow the tagged installed client and authenticated helper"}
			}
			if next < len(sequence) && line == sequence[next] {
				next++
			}
		}
		if next == len(sequence) {
			return nil
		}
	}
	return []string{"PNPM inspection requires tagged compile, protected installation, helper start and supply in the same active lifecycle block"}
}
