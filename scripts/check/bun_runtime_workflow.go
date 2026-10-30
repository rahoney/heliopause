package main

import "strings"

const bunNativeIntegrationCommand = `HELOX_BUN_NATIVE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^(TestBunPublicNativeResolverIntegration|TestBunActualNativeProjectSelectionBoundary)$'`
const bunProjectQuarantineCommand = `HELOX_BUN_PROJECT_QUARANTINE=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunPublicProjectQuarantineIntegration$'`
const bunArtifactExecutionCommand = `HELOX_BUN_ARTIFACT_EXECUTION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^TestBunActualArtifactExecutionBoundary$'`

const bunApprovedProjectCommand = `HELOX_BUN_CACHE_INTEGRATION=1 HELOX_BUN_PROJECT_TRANSACTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunPublic(ApprovedCache|ProjectTransaction)Integration$'`
const bunProjectCLICommand = `HELOX_BUN_PROJECT_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunProjectCLIIntegration$'`
const bunOriginalProjectCommand = `HELOX_BUN_NATIVE_ORIGINAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunNativeOriginalIntegration$'`

const bunColdIntegrationBlock = `bun_cold_log="$bun_work/cold.log"
bun_cold_status=0
(
  cd internal/sandbox
  HELOX_BUN_TOOL_RUN_SELECTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^TestBunPublicColdToolRunSelectionIntegration$'
) >"$bun_cold_log" 2>&1 || bun_cold_status=$?
bun_cold_log_status=0
cat -- "$bun_cold_log" || bun_cold_log_status=$?
if ((bun_cold_status)); then exit "$bun_cold_status"; fi
if ((bun_cold_log_status)); then exit "$bun_cold_log_status"; fi
awk '
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 != "TestBunPublicColdToolRunSelectionIntegration" || parents++) bad=1 }
/bun_cold_run / {
  if ($0 !~ /^[[:space:]]+bun_run_resolver_real_test\.go:[0-9]+: bun_cold_run request=typescript@5\.9\.3 roots=1 instances=1 cache_entries=[1-9][0-9]* source_artifacts=1 command=SEALED_MISSING_PATH native_DATA=UNAPPROVED Policy\/program=NOT_RUN$/ || cases++) bad=1
}
END { exit (bad || parents!=1 || cases!=1) }
' "$bun_cold_log"`

const bunRunIntegrationBlock = `bun_run_log="$bun_work/run.log"
bun_run_status=0
(
  cd internal/bootstrap
  HELOX_BUN_NATIVE_RUNTIME=1 HELOX_BUN_RUN_EXECUTION_WORKFLOW_INTEGRATION=1 HELOX_BUN_RUN_EXECUTION_CLI_INTEGRATION=1 HELOX_BUN_RUN_EXECUTION_CLI_REFUSAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=10m -test.run '^TestLinuxBunRunExecution(Workflow|CLI(Refusal)?)Integration$'
) >"$bun_run_log" 2>&1 || bun_run_status=$?
bun_run_log_status=0
cat -- "$bun_run_log" || bun_run_log_status=$?
if ((bun_run_status)); then exit "$bun_run_status"; fi
if ((bun_run_log_status)); then exit "$bun_run_log_status"; fi
awk '
BEGIN {
expected["TestLinuxBunRunExecutionWorkflowIntegration"]=1
expected["TestLinuxBunRunExecutionCLIIntegration"]=1
expected["TestLinuxBunRunExecutionCLIRefusalIntegration"]=1
cli[1]="alias=bunx format=json language=en show_output=false exit_code=0"
cli[2]="alias=x format=json language=ko show_output=true exit_code=1"
cli[3]="alias=bunx format=human language=ko show_output=true exit_code=0"
}
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if (!($3 in expected) || parents[$3]++) bad=1; parent_count++ }
/production_bun_run_application attempt=/ {
if ($0 !~ /^[[:space:]]+node_execution_run_integration_test\.go:[0-9]+: production_bun_run_application attempt=[12] artifact_role=true target_loaded=true exit_code=[01] stdout_bytes=[1-9][0-9]* stderr_bytes=0 user_program_run=true$/) bad=1
id=$3; sub(/^attempt=/,"",id)
if (apps[id]++ || $6 != "exit_code=" (id-1)) bad=1
app_count++
}
/production_bun_run_cli case=/ {
id=$3; sub(/^case=/,"",id)
if (!(id in cli) || cases[id]++ || $0 !~ ("^[[:space:]]+npm_execution_integration_test\\.go:[0-9]+: production_bun_run_cli case=" id " " cli[id] " current_policy=ALLOW guest_cleanup=CONFIRMED$")) bad=1
case_count++
}
/production_bun_run_cli_graph / {
if ($0 !~ /^[[:space:]]+npm_execution_integration_test\.go:[0-9]+: production_bun_run_cli_graph nodes=[1-9][0-9]* edges=[1-9][0-9]* primary=rimraf current_policy=ALLOW$/ || graph++) bad=1
nodes=$3; sub(/^nodes=/,"",nodes); if (nodes<=1) bad=1
}
/production_bun_run_refusal case=/ {
id=$3; sub(/^case=/,"",id)
reason=(id==1 ? "ambiguous" : "missing")
if (id !~ /^[12]$/ || refused[id]++ || $0 !~ ("^[[:space:]]+npm_execution_refusal_integration_test\\.go:[0-9]+: production_bun_run_refusal case=" id " reason=" reason " current_policy=ALLOW preparation=INCOMPLETE runtime_attempted=false exit_absent=true output_bytes=0 stored_manifest=0$")) bad=1
refusal_count++
}
END { exit (bad || parent_count!=3 || app_count!=2 || case_count!=3 || refusal_count!=2 || graph!=1) }
' "$bun_run_log"`

const bunProgramIntegrationBlock = `bun_program_log="$bun_work/program.log"
bun_program_status=0
(
  cd internal/bootstrap
  HELOX_BUN_NATIVE_RUNTIME=1 HELOX_BUN_PROGRAM_EXECUTION_WORKFLOW_INTEGRATION=1 HELOX_BUN_PROGRAM_EXECUTION_CLI_INTEGRATION=1 HELOX_BUN_PROGRAM_EXECUTION_BOUNDARY_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=10m -test.run '^TestLinuxBunProgramExecution(Workflow|CLI|Boundary)Integration$'
) >"$bun_program_log" 2>&1 || bun_program_status=$?
bun_program_log_status=0
cat -- "$bun_program_log" || bun_program_log_status=$?
if ((bun_program_status)); then exit "$bun_program_status"; fi
if ((bun_program_log_status)); then exit "$bun_program_log_status"; fi
awk '
BEGIN {
split("argv term kill readonly", complete, " "); for (i in complete) boundary[complete[i]]="true"
split("cancel stdout-overflow stderr-overflow outside-read network child comm-spoof deadline", incomplete, " "); for (i in incomplete) boundary[incomplete[i]]="false"
expected["TestLinuxBunProgramExecutionWorkflowIntegration"]=1
expected["TestLinuxBunProgramExecutionCLIIntegration"]=1
expected["TestLinuxBunProgramExecutionBoundaryIntegration"]=1
cli[1]="alias=bunx format=json language=en show_output=false exit_code=0"
cli[2]="alias=x format=json language=ko show_output=true exit_code=1"
cli[3]="alias=bunx format=human language=ko show_output=true exit_code=0"
}
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if (!($3 in expected) || parents[$3]++) bad=1; parent_count++ }
/production_bun_program_application attempt=/ {
if ($0 !~ /^[[:space:]]+node_execution_run_integration_test\.go:[0-9]+: production_bun_program_application attempt=[12] artifact_role=true target_loaded=true exit_code=[01] stdout_bytes=[1-9][0-9]* stderr_bytes=0 user_program_run=true$/) bad=1
id=$3; sub(/^attempt=/,"",id)
if (apps[id]++ || $6 != "exit_code=" (id-1)) bad=1
app_count++
}
/production_bun_program_cli case=/ {
id=$3; sub(/^case=/,"",id)
if (!(id in cli) || cases[id]++ || $0 !~ ("^[[:space:]]+npm_execution_integration_test\\.go:[0-9]+: production_bun_program_cli case=" id " " cli[id] " current_policy=ALLOW guest_cleanup=CONFIRMED$")) bad=1
case_count++
}
/production_bun_program_cli_graph / {
if ($0 !~ /^[[:space:]]+npm_execution_integration_test\.go:[0-9]+: production_bun_program_cli_graph nodes=[1-9][0-9]* edges=[1-9][0-9]* primary=rimraf current_policy=ALLOW$/ || graph++) bad=1
nodes=$3; sub(/^nodes=/,"",nodes); if (nodes<=1) bad=1
}
/bun_program_boundary case=/ {
if ($0 !~ /^[[:space:]]+node_execution_boundary_integration_test\.go:[0-9]+: bun_program_boundary case=[a-z-]+ current_policy=ALLOW fixture_registry=CONTROLLED_TEST native_solver=NOT_RUN complete=(true|false) stdout_bytes=[0-9]+ stderr_bytes=[0-9]+$/) bad=1
id=$3; sub(/^case=/,"",id)
if (!(id in boundary) || boundaries[id]++ || $7 != "complete=" boundary[id]) bad=1
boundary_count++
}
END { exit (bad || parent_count!=3 || app_count!=2 || case_count!=3 || boundary_count!=12 || graph!=1) }
' "$bun_program_log"`

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
		"set -euo pipefail", `case "$#:${1-}" in`, `0:|1:project) ;;`, `*) echo 'unsupported Bun CI profile' >&2; exit 2 ;;`, "esac", `bun_profile=${1:-sandbox}`, `test "$(id -u)" != 0`,
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
		bunNativeIntegrationCommand, bunProjectQuarantineCommand, bunArtifactExecutionCommand, bunApprovedProjectCommand, bunProjectCLICommand, bunOriginalProjectCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "Bun runtime helper requires one command: "+command)
		}
	}
	scope := strings.Join([]string{"(", "cd internal/sandbox", bunNativeIntegrationCommand, bunProjectQuarantineCommand, bunArtifactExecutionCommand, ")"}, "\n")
	if strings.Count(strings.Join(active, "\n"), scope) != 1 {
		findings = append(findings, "Bun actual consumers require one matching package subshell")
	}
	projectScope := strings.Join([]string{`if [[ "$bun_profile" == project ]]; then`, "(", "cd internal/bootstrap", bunApprovedProjectCommand, bunProjectCLICommand, bunOriginalProjectCommand, ")", kernelQualificationLines(bunRunIntegrationBlock), kernelQualificationLines(bunProgramIntegrationBlock), "exit 0", "fi"}, "\n")
	if strings.Count(strings.Join(active, "\n"), projectScope) != 1 {
		findings = append(findings, "Bun project consumers require one guarded bootstrap subshell")
	}
	for _, block := range []string{bunColdIntegrationBlock, bunRunIntegrationBlock, bunProgramIntegrationBlock} {
		if strings.Count(kernelQualificationLines(contents), kernelQualificationLines(block)) != 1 {
			findings = append(findings, "Bun one-off qualification requires exact scopes, unique complete cohorts, no-SKIP and original-error guards")
		}
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

func validateBunProjectClientOrder(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return nil
	}
	sequence := strings.Join([]string{"bash scripts/ci-bun-runtime.sh", `haa_install_ci_client "$RUNNER_TEMP/helox-pnpm-project-test"`, "haa_start_policy_helper", "bash scripts/ci-bun-runtime.sh project", `haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"`, "haa_start_policy_helper"}, "\n")
	for _, lines := range blocks {
		if strings.Count(strings.Join(lines, "\n"), sequence) == 1 {
			return nil
		}
	}
	return []string{"Bun project actual requires bootstrap client/helper and sandbox restoration in the owning lifecycle"}
}
