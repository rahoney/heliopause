package main

import "strings"

const pnpmRuntimeIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 go test -tags=realcorpus -v -timeout=3m ./internal/sandbox -run '^TestPNPMActual(Isolated(NativeStartup|EmptyLock|NodeQuery)|ReleaseArtifactProbe)$'`

const pnpmProjectIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMActualProductionProjectInspection$'`
const pnpmControlledIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMActualProductionInstanceExecution$/^controlled-(success|exit37|long-name)$'`
const pnpmPublicQuarantineCommand = `HELOX_PNPM_PUBLIC_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMPublicNativeResolverIntegration$'`
const pnpmSandboxTestBuildCommand = `go test -c -tags=realcorpus -o "$RUNNER_TEMP/helox-sandbox-test" ./internal/sandbox`
const pnpmBootstrapTestBuildCommand = `go test -c -tags=realcorpus -o "$RUNNER_TEMP/helox-pnpm-project-test" ./internal/bootstrap`
const pnpmBootstrapIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 HELOX_PNPM_CACHE_INTEGRATION=1 HELOX_PNPM_PROJECT_TRANSACTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMPublic(ApprovedCache|ProjectTransaction)Integration$'`
const pnpmCLIIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 HELOX_PNPM_PROJECT_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMProjectCLIIntegration$'`
const pnpmOriginalIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 HELOX_PNPM_NATIVE_ORIGINAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMNativeOriginalIntegration$'`

const pnpmRunIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_RUN_EXECUTION_WORKFLOW_INTEGRATION=1 HELOX_PNPM_RUN_EXECUTION_CLI_INTEGRATION=1 HELOX_PNPM_RUN_EXECUTION_CLI_REFUSAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=10m -test.run '^TestLinuxPNPMRunExecution(Workflow|CLI(Refusal)?)Integration$'`
const pnpmColdIntegrationCommand = `HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_TOOL_RUN_SELECTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^TestPNPMPublicColdToolRunSelectionIntegration$'`

const pnpmRunIntegrationBlock = `if [[ "$pnpm_profile" == run ]]; then
  pnpm_run_log="$pnpm_work/run.log"
  pnpm_run_status=0
  (
    cd internal/bootstrap
    HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_RUN_EXECUTION_WORKFLOW_INTEGRATION=1 HELOX_PNPM_RUN_EXECUTION_CLI_INTEGRATION=1 HELOX_PNPM_RUN_EXECUTION_CLI_REFUSAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=10m -test.run '^TestLinuxPNPMRunExecution(Workflow|CLI(Refusal)?)Integration$'
  ) >"$pnpm_run_log" 2>&1 || pnpm_run_status=$?
  pnpm_run_log_status=0
  cat -- "$pnpm_run_log" || pnpm_run_log_status=$?
  if ((pnpm_run_status)); then exit "$pnpm_run_status"; fi
  if ((pnpm_run_log_status)); then exit "$pnpm_run_log_status"; fi
  awk '
BEGIN {
  expected["TestLinuxPNPMRunExecutionWorkflowIntegration"]=1
  expected["TestLinuxPNPMRunExecutionCLIIntegration"]=1
  expected["TestLinuxPNPMRunExecutionCLIRefusalIntegration"]=1
  cli[1]="alias=dlx format=json language=en show_output=false exit_code=0"
  cli[2]="alias=pnx format=json language=ko show_output=true exit_code=1"
  cli[3]="alias=pnpx format=human language=ko show_output=true exit_code=0"
}
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if (!($3 in expected) || parents[$3]++) bad=1; parent_count++ }
/production_pnpm_run_application attempt=/ {
  if ($0 !~ /^[[:space:]]+node_execution_run_integration_test\.go:[0-9]+: production_pnpm_run_application attempt=[12] artifact_role=true target_loaded=true exit_code=[01] stdout_bytes=[1-9][0-9]* stderr_bytes=0 user_program_run=true$/) bad=1
  id=$3; sub(/^attempt=/,"",id)
  if (apps[id]++ || $6 != "exit_code=" (id-1)) bad=1
  app_count++
}
/production_pnpm_run_cli case=/ {
  id=$3; sub(/^case=/,"",id)
  if (!(id in cli) || cases[id]++ || $0 !~ ("^[[:space:]]+npm_execution_integration_test\\.go:[0-9]+: production_pnpm_run_cli case=" id " " cli[id] " current_policy=ALLOW guest_cleanup=CONFIRMED$")) bad=1
  case_count++
}
/production_pnpm_run_cli_graph / {
  if ($0 !~ /^[[:space:]]+npm_execution_integration_test\.go:[0-9]+: production_pnpm_run_cli_graph nodes=[1-9][0-9]* edges=[1-9][0-9]* primary=rimraf current_policy=ALLOW$/ || graph++) bad=1
  nodes=$3; sub(/^nodes=/,"",nodes); if (nodes<=1) bad=1
}
/production_pnpm_run_refusal case=/ {
  id=$3; sub(/^case=/,"",id)
  reason=(id==1 ? "ambiguous" : "missing")
  if (id !~ /^[12]$/ || refused[id]++ || $0 !~ ("^[[:space:]]+npm_execution_refusal_integration_test\\.go:[0-9]+: production_pnpm_run_refusal case=" id " reason=" reason " current_policy=ALLOW preparation=INCOMPLETE runtime_attempted=false exit_absent=true output_bytes=0 stored_manifest=0$")) bad=1
  refusal_count++
}
END { exit (bad || parent_count!=3 || app_count!=2 || case_count!=3 || refusal_count!=2 || graph!=1) }
' "$pnpm_run_log"
  exit 0
fi`

const pnpmColdIntegrationBlock = `pnpm_cold_log="$pnpm_work/cold.log"
pnpm_cold_status=0
(
  cd internal/sandbox
  HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_TOOL_RUN_SELECTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^TestPNPMPublicColdToolRunSelectionIntegration$'
) >"$pnpm_cold_log" 2>&1 || pnpm_cold_status=$?
pnpm_cold_log_status=0
cat -- "$pnpm_cold_log" || pnpm_cold_log_status=$?
if ((pnpm_cold_status)); then exit "$pnpm_cold_status"; fi
if ((pnpm_cold_log_status)); then exit "$pnpm_cold_log_status"; fi
awk '
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 != "TestPNPMPublicColdToolRunSelectionIntegration" || parents++) bad=1 }
/pnpm_cold_run / {
  if ($0 !~ /^[[:space:]]+pnpm_run_resolver_real_test\.go:[0-9]+: pnpm_cold_run roots=1 packages=1 instances=1 cache_entries=[1-9][0-9]* source_artifacts=1 selected=typescript@5\.9\.3 command=SEALED_MISSING_PATH native_DATA=UNAPPROVED$/ || cases++) bad=1
}
END { exit (bad || parents!=1 || cases!=1) }
' "$pnpm_cold_log"`

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
		pnpmBootstrapIntegrationCommand,
		pnpmCLIIntegrationCommand,
		pnpmOriginalIntegrationCommand,
		pnpmRunIntegrationCommand,
		pnpmColdIntegrationCommand,
		`test "$#" -le 1`,
		`pnpm_profile=${1:-sandbox}`,
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
	profileGuard := "case \"$pnpm_profile\" in\nsandbox|project|run) ;;\n*) echo 'unsupported PNPM CI profile' >&2; exit 2 ;;\nesac"
	projectScope := "if [[ \"$pnpm_profile\" == project ]]; then\n" + pnpmBootstrapIntegrationCommand + "\n" + pnpmCLIIntegrationCommand + "\n" + pnpmOriginalIntegrationCommand + "\nexit 0\nfi"
	if strings.Count(strings.Join(active, "\n"), profileGuard) != 1 || strings.Count(strings.Join(active, "\n"), projectScope) != 1 {
		findings = append(findings, "PNPM project profile requires its fixed argument guard and exclusive installed-client command")
	}
	for _, block := range []string{pnpmRunIntegrationBlock, pnpmColdIntegrationBlock} {
		if strings.Count(kernelQualificationLines(contents), kernelQualificationLines(block)) != 1 {
			findings = append(findings, "PNPM one-off qualification requires exact scopes, complete unique cohorts, no-SKIP and original-error guards")
		}
	}
	return findings
}

func validatePNPMProjectClientOrder(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return nil
	}
	sequence := strings.Join([]string{
		"bash scripts/ci-pnpm-runtime.sh", pnpmBootstrapTestBuildCommand,
		`haa_install_ci_client "$RUNNER_TEMP/helox-pnpm-project-test"`, "haa_start_policy_helper",
		"bash scripts/ci-pnpm-runtime.sh project", `haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"`, "haa_start_policy_helper",
	}, "\n")
	for _, lines := range blocks {
		if strings.Count(strings.Join(lines, "\n"), sequence) == 1 {
			return nil
		}
	}
	return []string{"PNPM project actual requires tagged bootstrap compile, protected client/helper and sandbox restoration in the owning lifecycle"}
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
