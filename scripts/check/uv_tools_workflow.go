package main

import "strings"

const uvToolNativeCommand = `HELOX_UV_TOOL_RESOLVER_INTEGRATION=1 HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestUVPublicTool(Install|Upgrade)ResolverIntegration$'`
const uvToolBootstrapCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_UV_TOOL_INSTALL_CLI_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_CLI_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE=1 HELOX_UV_TOOL_NATIVE_ADOPTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=12m -test.run '^(TestLinuxUVTool(Install|Upgrade|NativeAdoption)CLIIntegration|TestUVPublicTool(ApprovedUpgrade|UpgradeEvidenceFailure)Integration)$'`

const pipxToolBootstrapCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PIPX_TOOL_INSTALL_CLI_INTEGRATION=1 HELOX_PIPX_TOOL_UPDATE_CLI_INTEGRATION=1 HELOX_PIPX_TOOL_NATIVE_ADOPTION_INTEGRATION=1 HELOX_PIPX_TOOL_UPDATE_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=12m -test.run '^(TestLinuxPipxTool(Install|Upgrade|NativeAdoption)CLIIntegration|TestPipxPublicToolApprovedUpgradeIntegration)$'`

const pythonExecutionPreparationCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_EXECUTION_PREPARATION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonExecutionPreparationIntegration$'`

const pythonExecutionInputsCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_EXECUTION_INPUT_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonExecutionInputsIntegration$'`

const pythonConsoleExecutionCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_EXECUTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonExecutionIntegration$'`

const pythonExecutionBoundaryCommand = `HELOX_PYTHON_EXECUTION_BOUNDARY_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=8m -test.run '^TestLinuxPythonExecutionBoundaryIntegration$'`

const pythonExecutionHostileCommand = `HELOX_PYTHON_EXECUTION_HOSTILE_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=8m -test.run '^TestLinuxPythonExecutionHostileIntegration$'`

const pythonExecutionWorkflowCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_EXECUTION_WORKFLOW_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonExecutionWorkflowIntegration$'`

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
		"set -euo pipefail", `0:|1:bootstrap|1:pipx-bootstrap|1:python-execution-preparation|1:python-execution-inputs|1:python-execution|1:python-execution-boundary|1:python-execution-hostile|1:python-execution-workflow) ;;`, `if [[ ${1-} != bootstrap && ${1-} != pipx-bootstrap && ${1-} != python-execution-preparation && ${1-} != python-execution-inputs && ${1-} != python-execution && ${1-} != python-execution-boundary && ${1-} != python-execution-hostile && ${1-} != python-execution-workflow ]]; then`, `*) echo 'unsupported UV tools CI profile' >&2; exit 2 ;;`,
		`test "$(id -u)" != 0`, "trap uv_tools_cleanup EXIT", "go run ./scripts/generate-runtime-lock.go -check",
		`uv_tools_image=$(jq -er '.python_image.reference' scripts/runtimes.lock.json)`,
		`uv_tools_container=$(docker create --pull never --network none --read-only --cap-drop ALL --label "io.heliopause.ci.uv-tools-data=$uv_tools_owner" --entrypoint /bin/false "$uv_tools_image")`,
		`docker cp "$uv_tools_container:/usr/local/bin/python3.14" "$uv_tools_work/python3.14"`,
		`chmod 0400 "$uv_tools_work/python3.14"`, `uv_tools_sha=$(sha256sum "$uv_tools_work/python3.14" | cut -d ' ' -f 1)`,
		uvToolNativeCommand, uvToolBootstrapCommand, pipxToolBootstrapCommand, pythonExecutionPreparationCommand, pythonExecutionInputsCommand, pythonConsoleExecutionCommand, pythonExecutionBoundaryCommand, pythonExecutionHostileCommand, pythonExecutionWorkflowCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "UV tools helper requires one command: "+command)
		}
	}
	joined := strings.Join(active, "\n")
	for _, scope := range [][]string{{"test \"$(id -u)\" != 0", "if [[ ${1-} == python-execution-boundary ]]; then", "(", "cd internal/bootstrap", pythonExecutionBoundaryCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution-hostile ]]; then", "(", "cd internal/bootstrap", pythonExecutionHostileCommand, ")", "exit 0", "fi"}, {"(", "cd internal/sandbox", uvToolNativeCommand, ")", "exit 0"}, {"uv_tools_owned", "uv_tools_remove", "if [[ ${1-} == python-execution-preparation ]]; then", "(", "cd internal/bootstrap", pythonExecutionPreparationCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution-inputs ]]; then", "(", "cd internal/bootstrap", pythonExecutionInputsCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution ]]; then", "(", "cd internal/bootstrap", pythonConsoleExecutionCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution-workflow ]]; then", "(", "cd internal/bootstrap", pythonExecutionWorkflowCommand, ")", "exit 0", "fi", "if [[ ${1-} == pipx-bootstrap ]]; then"}, {"if [[ ${1-} == pipx-bootstrap ]]; then", "(", "cd internal/bootstrap", pipxToolBootstrapCommand, ")", "else", "(", "cd internal/bootstrap", uvToolBootstrapCommand, ")", "fi"}} {
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

const currentExecutionPreparationStep = "Qualify current execution workflows and boundaries"

const currentExecutionPreparationRun = `set -euo pipefail
source scripts/ci-integration-cleanup.sh
haa_ci_initialize
haa_install_ci_client "$RUNNER_TEMP/helox-pnpm-project-test"
haa_start_policy_helper
umask 077
node_refusal_log="$RUNNER_TEMP/haa-node-refusal.log"
node_refusal_status=0
(
  cd internal/bootstrap
  HELOX_NPM_EXECUTION_CLI_REFUSAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxNPMExecutionCLIRefusalIntegration$'
) >"$node_refusal_log" 2>&1 || node_refusal_status=$?
node_refusal_log_status=0
cat -- "$node_refusal_log" || node_refusal_log_status=$?
if ((node_refusal_status)); then exit "$node_refusal_status"; fi
if ((node_refusal_log_status)); then exit "$node_refusal_log_status"; fi
awk '
BEGIN { n=split("1 2", names, " "); for (i=1; i<=n; i++) expected[names[i]]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS: TestLinuxNPMExecutionCLIRefusalIntegration \(/ { seen++ }
/^[[:space:]]+npm_execution_refusal_integration_test\.go:[0-9]+: production_npm_execution_refusal case=/ {
  case_name=$0
  sub(/^.* production_npm_execution_refusal case=/, "", case_name)
  sub(/ .*/, "", case_name)
  if (!(case_name in expected) || accepted[case_name]++) bad=1
  cases++
}
END { exit (bad || seen != 1 || cases != 2) }
' "$node_refusal_log"
rm -- "$node_refusal_log"

python_preparation_log="$RUNNER_TEMP/haa-python-preparation.log"
python_preparation_status=0
bash scripts/ci-uv-tools.sh python-execution-preparation >"$python_preparation_log" 2>&1 || python_preparation_status=$?
python_preparation_log_status=0
cat -- "$python_preparation_log" || python_preparation_log_status=$?
if ((python_preparation_status)); then exit "$python_preparation_status"; fi
if ((python_preparation_log_status)); then exit "$python_preparation_log_status"; fi
awk '
BEGIN { n=split("1 2 3", names, " "); for (i=1; i<=n; i++) expected[names[i]]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS: TestLinuxPythonExecutionPreparationIntegration \(/ { seen++ }
/^[[:space:]]+python_execution_prepare_integration_test\.go:[0-9]+: python_execution_preparation case=/ {
  case_name=$0
  sub(/^.* python_execution_preparation case=/, "", case_name)
  sub(/ .*/, "", case_name)
  if (!(case_name in expected) || accepted[case_name]++) bad=1
  cases++
}
END { exit (bad || seen != 1 || cases != 3) }
' "$python_preparation_log"
rm -- "$python_preparation_log"

python_inputs_log="$RUNNER_TEMP/haa-python-inputs.log"
python_inputs_status=0
bash scripts/ci-uv-tools.sh python-execution-inputs >"$python_inputs_log" 2>&1 || python_inputs_status=$?
python_inputs_log_status=0
cat -- "$python_inputs_log" || python_inputs_log_status=$?
if ((python_inputs_status)); then exit "$python_inputs_status"; fi
if ((python_inputs_log_status)); then exit "$python_inputs_log_status"; fi
awk '
BEGIN { n=split("1 2 3 4", names, " "); for (i=1; i<=n; i++) expected[names[i]]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS: TestLinuxPythonExecutionInputsIntegration \(/ { seen++ }
/^[[:space:]]+python_execution_inputs_integration_test\.go:[0-9]+: python_execution_inputs case=/ {
  case_name=$0
  sub(/^.* python_execution_inputs case=/, "", case_name)
  sub(/ .*/, "", case_name)
  if (!(case_name in expected) || accepted[case_name]++) bad=1
  cases++
}
END { exit (bad || seen != 1 || cases != 4) }
' "$python_inputs_log"
rm -- "$python_inputs_log"

python_runtime_log="$RUNNER_TEMP/haa-python-runtime.log"
python_runtime_status=0
bash scripts/ci-uv-tools.sh python-execution >"$python_runtime_log" 2>&1 || python_runtime_status=$?
python_runtime_log_status=0
cat -- "$python_runtime_log" || python_runtime_log_status=$?
if ((python_runtime_status)); then exit "$python_runtime_status"; fi
if ((python_runtime_log_status)); then exit "$python_runtime_log_status"; fi
awk '
BEGIN { n=split("1 2", names, " "); for (i=1; i<=n; i++) expected[names[i]]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS: TestLinuxPythonExecutionIntegration \(/ { seen++ }
/^[[:space:]]+python_execution_integration_test\.go:[0-9]+: python_execution case=/ {
  case_name=$0
  sub(/^.* python_execution case=/, "", case_name)
  sub(/ .*/, "", case_name)
  if (!(case_name in expected) || accepted[case_name]++) bad=1
  cases++
}
END { exit (bad || seen != 1 || cases != 2) }
' "$python_runtime_log"
rm -- "$python_runtime_log"

python_boundary_log="$RUNNER_TEMP/haa-python-boundary.log"
python_boundary_status=0
bash scripts/ci-uv-tools.sh python-execution-boundary >"$python_boundary_log" 2>&1 || python_boundary_status=$?
python_boundary_log_status=0
cat -- "$python_boundary_log" || python_boundary_log_status=$?
if ((python_boundary_status)); then exit "$python_boundary_status"; fi
if ((python_boundary_log_status)); then exit "$python_boundary_log_status"; fi
awk '
BEGIN { n=split("argv exit37 term kill cancel stdout-overflow stderr-overflow deadline", names, " "); for (i=1; i<=n; i++) expected[names[i]]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS: TestLinuxPythonExecutionBoundaryIntegration \(/ { seen++ }
/^[[:space:]]+python_execution_boundary_integration_test\.go:[0-9]+: python_boundary case=/ {
  case_name=$0
  sub(/^.* python_boundary case=/, "", case_name)
  sub(/ .*/, "", case_name)
  if (!(case_name in expected) || accepted[case_name]++) bad=1
  cases++
}
END { exit (bad || seen != 1 || cases != 8) }
' "$python_boundary_log"
rm -- "$python_boundary_log"

python_hostile_log="$RUNNER_TEMP/haa-python-hostile.log"
python_hostile_status=0
bash scripts/ci-uv-tools.sh python-execution-hostile >"$python_hostile_log" 2>&1 || python_hostile_status=$?
python_hostile_log_status=0
cat -- "$python_hostile_log" || python_hostile_log_status=$?
if ((python_hostile_status)); then exit "$python_hostile_status"; fi
if ((python_hostile_log_status)); then exit "$python_hostile_log_status"; fi
awk '
BEGIN { n=split("outside-read network clone-exec comm-spoof readonly", names, " "); for (i=1; i<=n; i++) expected[names[i]]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS: TestLinuxPythonExecutionHostileIntegration \(/ { seen++ }
/^[[:space:]]+python_execution_hostile_integration_test\.go:[0-9]+: python_hostile case=/ {
  case_name=$0
  sub(/^.* python_hostile case=/, "", case_name)
  sub(/ .*/, "", case_name)
  if (!(case_name in expected) || accepted[case_name]++) bad=1
  cases++
}
END { exit (bad || seen != 1 || cases != 5) }
' "$python_hostile_log"
rm -- "$python_hostile_log"

python_workflow_log="$RUNNER_TEMP/haa-python-workflow.log"
python_workflow_status=0
bash scripts/ci-uv-tools.sh python-execution-workflow >"$python_workflow_log" 2>&1 || python_workflow_status=$?
python_workflow_log_status=0
cat -- "$python_workflow_log" || python_workflow_log_status=$?
if ((python_workflow_status)); then exit "$python_workflow_status"; fi
if ((python_workflow_log_status)); then exit "$python_workflow_log_status"; fi
awk '
BEGIN { n=split("1 2 3", names, " "); for (i=1; i<=n; i++) expected[names[i]]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS: TestLinuxPythonExecutionWorkflowIntegration \(/ { seen++ }
/^[[:space:]]+python_execution_run_integration_test\.go:[0-9]+: python_application case=/ {
  case_name=$0
  sub(/^.* python_application case=/, "", case_name)
  sub(/ .*/, "", case_name)
  if (!(case_name in expected) || accepted[case_name]++) bad=1
  cases++
}
END { exit (bad || seen != 1 || cases != 3) }
' "$python_workflow_log"
rm -- "$python_workflow_log"
`

func validateCurrentExecutionPreparationQualification(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", currentExecutionPreparationStep)
	if err != nil {
		return nil
	}
	if len(blocks) == 1 && kernelQualificationLines(strings.Join(blocks[0], "\n")) == kernelQualificationLines(currentExecutionPreparationRun) {
		return nil
	}
	return []string{"current execution preparation/refusal qualification requires one unconditional protected-client lifecycle, exact parents/readonly4/console2/boundary8/hostile5/workflow3/unique cases/no-SKIP and original-error guards"}
}
