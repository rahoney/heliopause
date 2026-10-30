package main

import "strings"

const uvRunSourceCommand = `HELOX_UV_NATIVE_RUNTIME=1 HELOX_UV_TOOL_RUN_RESOLVER_INTEGRATION=1 HELOX_UV_TOOL_RUN_PUBLIC_BINDING_INTEGRATION=1 HELOX_UV_TOOL_RUN_NATIVE_CENSUS_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^(TestUVActualPublicToolRun(SourceBinding|NativeCensus)|TestUVPublicToolRunResolverIntegration)$'`

const uvRunWorkflowCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_UV_RUN_EXECUTION_WORKFLOW_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonUVRunExecutionWorkflowIntegration$'`

const pipxRunWorkflowCommand = `HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_PIPX_RUN_EXECUTION_WORKFLOW_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonPipxRunExecutionWorkflowIntegration$'`

const uvRunCLICommand = `HELOX_UV_RUN_EXECUTION_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=8m -test.run '^TestLinuxUVRunExecutionCLIIntegration$'`

const pipxRunCLICommand = `HELOX_PIPX_RUN_EXECUTION_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=8m -test.run '^TestLinuxPipxRunExecutionCLIIntegration$'`

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
		"set -euo pipefail", `0:|1:bootstrap|1:pipx-bootstrap|1:python-execution-preparation|1:python-execution-inputs|1:python-execution|1:python-execution-boundary|1:python-execution-hostile|1:python-execution-workflow|1:uv-run-source|1:uv-run-workflow|1:uv-run-cli|1:pipx-run-workflow|1:pipx-run-cli) ;;`, `if [[ ${1-} != bootstrap && ${1-} != pipx-bootstrap && ${1-} != python-execution-preparation && ${1-} != python-execution-inputs && ${1-} != python-execution && ${1-} != python-execution-boundary && ${1-} != python-execution-hostile && ${1-} != python-execution-workflow && ${1-} != uv-run-source && ${1-} != uv-run-workflow && ${1-} != uv-run-cli && ${1-} != pipx-run-workflow && ${1-} != pipx-run-cli ]]; then`, `*) echo 'unsupported UV tools CI profile' >&2; exit 2 ;;`,
		`test "$(id -u)" != 0`, "trap uv_tools_cleanup EXIT", "go run ./scripts/generate-runtime-lock.go -check",
		`uv_tools_image=$(jq -er '.python_image.reference' scripts/runtimes.lock.json)`,
		`uv_tools_container=$(docker create --pull never --network none --read-only --cap-drop ALL --label "io.heliopause.ci.uv-tools-data=$uv_tools_owner" --entrypoint /bin/false "$uv_tools_image")`,
		`docker cp "$uv_tools_container:/usr/local/bin/python3.14" "$uv_tools_work/python3.14"`,
		`chmod 0400 "$uv_tools_work/python3.14"`, `uv_tools_sha=$(sha256sum "$uv_tools_work/python3.14" | cut -d ' ' -f 1)`,
		uvRunSourceCommand, uvRunWorkflowCommand, uvRunCLICommand, pipxRunWorkflowCommand, pipxRunCLICommand, uvToolNativeCommand, uvToolBootstrapCommand, pipxToolBootstrapCommand, pythonExecutionPreparationCommand, pythonExecutionInputsCommand, pythonConsoleExecutionCommand, pythonExecutionBoundaryCommand, pythonExecutionHostileCommand, pythonExecutionWorkflowCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "UV tools helper requires one command: "+command)
		}
	}
	joined := strings.Join(active, "\n")
	for _, scope := range [][]string{{"test \"$(id -u)\" != 0", "if [[ ${1-} == uv-run-source ]]; then", "(", "cd internal/sandbox", uvRunSourceCommand, ")", "exit 0", "fi", "if [[ ${1-} == uv-run-cli || ${1-} == pipx-run-cli ]]; then", "(", "cd internal/bootstrap", "if [[ ${1-} == pipx-run-cli ]]; then", pipxRunCLICommand, "else", uvRunCLICommand, "fi", ")", "exit 0", "fi", "if [[ ${1-} == python-execution-boundary ]]; then", "(", "cd internal/bootstrap", pythonExecutionBoundaryCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution-hostile ]]; then", "(", "cd internal/bootstrap", pythonExecutionHostileCommand, ")", "exit 0", "fi"}, {"(", "cd internal/sandbox", uvToolNativeCommand, ")", "exit 0"}, {"uv_tools_owned", "uv_tools_remove", "if [[ ${1-} == python-execution-preparation ]]; then", "(", "cd internal/bootstrap", pythonExecutionPreparationCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution-inputs ]]; then", "(", "cd internal/bootstrap", pythonExecutionInputsCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution ]]; then", "(", "cd internal/bootstrap", pythonConsoleExecutionCommand, ")", "exit 0", "fi", "if [[ ${1-} == python-execution-workflow ]]; then", "(", "cd internal/bootstrap", pythonExecutionWorkflowCommand, ")", "exit 0", "fi", "if [[ ${1-} == uv-run-workflow || ${1-} == pipx-run-workflow ]]; then", "(", "cd internal/bootstrap", "if [[ ${1-} == pipx-run-workflow ]]; then", pipxRunWorkflowCommand, "else", uvRunWorkflowCommand, "fi", ")", "exit 0", "fi", "if [[ ${1-} == pipx-bootstrap ]]; then"}, {"if [[ ${1-} == pipx-bootstrap ]]; then", "(", "cd internal/bootstrap", pipxToolBootstrapCommand, ")", "else", "(", "cd internal/bootstrap", uvToolBootstrapCommand, ")", "fi"}} {
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

uv_run_workflow_log="$RUNNER_TEMP/haa-uv-run-workflow.log"
uv_run_workflow_status=0
bash scripts/ci-uv-tools.sh uv-run-workflow >"$uv_run_workflow_log" 2>&1 || uv_run_workflow_status=$?
uv_run_workflow_log_status=0
cat -- "$uv_run_workflow_log" || uv_run_workflow_log_status=$?
if ((uv_run_workflow_status)); then exit "$uv_run_workflow_status"; fi
if ((uv_run_workflow_log_status)); then exit "$uv_run_workflow_log_status"; fi
awk '
BEGIN { expected["1"]="true true"; expected["2"]="true true"; expected["3"]="false false" }
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 != "TestLinuxPythonUVRunExecutionWorkflowIntegration" || parent++) bad=1 }
/python_uv_run_application case=/ {
  if ($0 !~ /^[[:space:]]+python_uv_run_execution_integration_test\.go:[0-9]+: python_uv_run_application case=[1-3] current_policy=ALLOW selector=uv-tool-run-isolated runtime_attempted=(true|false) complete=(true|false) source_sha256=274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a$/) bad=1
  id=$3; sub(/^case=/,"",id); attempted=$6; sub(/^runtime_attempted=/,"",attempted); complete=$7; sub(/^complete=/,"",complete)
  if (!(id in expected) || seen[id]++ || expected[id] != attempted " " complete) bad=1
  cases++
}
END { exit (bad || parent != 1 || cases != 3) }
' "$uv_run_workflow_log"
rm -- "$uv_run_workflow_log"

uv_run_cli_log="$RUNNER_TEMP/haa-uv-run-cli.log"
uv_run_cli_status=0
bash scripts/ci-uv-tools.sh uv-run-cli >"$uv_run_cli_log" 2>&1 || uv_run_cli_status=$?
uv_run_cli_log_status=0
cat -- "$uv_run_cli_log" || uv_run_cli_log_status=$?
if ((uv_run_cli_status)); then exit "$uv_run_cli_status"; fi
if ((uv_run_cli_log_status)); then exit "$uv_run_cli_log_status"; fi
awk '
BEGIN {
  expected["1"]="alias=uvx format=json language=en show_output=false exit_code=0 current_policy=ALLOW nodes=1 guest_cleanup=CONFIRMED"
  expected["2"]="alias=run format=json language=ko show_output=true exit_code=2 current_policy=ALLOW nodes=1 guest_cleanup=CONFIRMED"
  expected["3"]="alias=uvx format=human language=ko show_output=true exit_code=0 current_policy=ALLOW nodes=1 guest_cleanup=CONFIRMED"
  expected["4"]="alias=run format=json language=en show_output=true exit_code=0 current_policy=ALLOW nodes=2 guest_cleanup=CONFIRMED"
  expected["5"]="alias=uvx format=json language=en show_output=false exit_code=1 current_policy=NOT_AVAILABLE runtime_attempted=false stored_manifest=0"
}
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 != "TestLinuxUVRunExecutionCLIIntegration" || parent++) bad=1 }
/production_uv_run_cli case=/ {
  id=$3; sub(/^case=/,"",id)
  if (!(id in expected) || seen[id]++ || $0 !~ ("^[[:space:]]+python_uv_run_cli_integration_test\\.go:[0-9]+: production_uv_run_cli case=" id " " expected[id] "$")) bad=1
  cases++
}
/production_uv_run_cli_source case=/ {
  if ($0 !~ /^[[:space:]]+python_uv_run_cli_integration_test\.go:[0-9]+: production_uv_run_cli_source case=[1-4] name=[a-z0-9-]+ version=[0-9.]+ sha256=[0-9a-f]+ evidence=[0-9]+$/) bad=1
  id=$3; sub(/^case=/,"",id); name=$4; sub(/^name=/,"",name); version=$5; sub(/^version=/,"",version); hash=$6; sub(/^sha256=/,"",hash); evidence=$7; sub(/^evidence=/,"",evidence)
  if (length(hash)!=64 || evidence+0<3 || source[id " " name]++) bad=1
  if (id ~ /^[1-3]$/) { if (name!="cowsay" || version!="6.1" || hash!="274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a") bad=1 }
  else if (id=="4") { if (name=="python-slugify") { if (version!="8.0.4" || hash!="276540b79961052b66b7d116620b36518847f52d5fd9e3a70164fc8c50faa6b8") bad=1 } else if (name!="text-unidecode") bad=1 }
  else bad=1
  sources++
}
END { exit (bad || parent!=1 || cases!=5 || sources!=5 || !source["1 cowsay"] || !source["2 cowsay"] || !source["3 cowsay"] || !source["4 python-slugify"] || !source["4 text-unidecode"]) }
' "$uv_run_cli_log"
rm -- "$uv_run_cli_log"

pipx_run_log="$RUNNER_TEMP/haa-pipx-run.log"
pipx_run_status=0
bash scripts/ci-pipx-runtime.sh run >"$pipx_run_log" 2>&1 || pipx_run_status=$?
pipx_run_log_status=0
cat -- "$pipx_run_log" || pipx_run_log_status=$?
if ((pipx_run_status)); then exit "$pipx_run_status"; fi
if ((pipx_run_log_status)); then exit "$pipx_run_log_status"; fi
awk '
BEGIN { expected["1"]="true true"; expected["2"]="true true"; expected["3"]="false false" }
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 == "TestLinuxPythonPipxRunExecutionWorkflowIntegration") { if (parent++) bad=1 } else if ($3 != "TestLinuxPipxRunExecutionCLIIntegration") bad=1 }
/python_pipx_run_application case=/ {
  if ($0 !~ /^[[:space:]]+python_uv_run_execution_integration_test\.go:[0-9]+: python_pipx_run_application case=[1-3] current_policy=ALLOW selector=pipx-run-isolated runtime_attempted=(true|false) complete=(true|false) source_sha256=274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a$/) bad=1
  id=$3; sub(/^case=/,"",id); attempted=$6; sub(/^runtime_attempted=/,"",attempted); complete=$7; sub(/^complete=/,"",complete)
  if (!(id in expected) || seen[id]++ || expected[id] != attempted " " complete) bad=1
  cases++
}
END { exit (bad || parent != 1 || cases != 3) }
' "$pipx_run_log"
awk '
BEGIN {
  expected["1"]="alias=run format=json language=en show_output=false exit_code=0 current_policy=ALLOW nodes=1 guest_cleanup=CONFIRMED"
  expected["2"]="alias=run format=json language=ko show_output=true exit_code=2 current_policy=ALLOW nodes=1 guest_cleanup=CONFIRMED"
  expected["3"]="alias=run format=human language=ko show_output=true exit_code=0 current_policy=ALLOW nodes=1 guest_cleanup=CONFIRMED"
  expected["4"]="alias=run format=json language=en show_output=true exit_code=0 current_policy=ALLOW nodes=2 guest_cleanup=CONFIRMED"
  expected["5"]="alias=run format=json language=en show_output=false exit_code=1 current_policy=NOT_AVAILABLE runtime_attempted=false stored_manifest=0"
  expected["6"]="alias=run format=json language=en show_output=true exit_code=0 current_policy=ALLOW nodes=2 guest_cleanup=CONFIRMED"
}
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 == "TestLinuxPipxRunExecutionCLIIntegration") { if (parent++) bad=1 } else if ($3 != "TestLinuxPythonPipxRunExecutionWorkflowIntegration") bad=1 }
/production_pipx_run_cli case=/ {
  id=$3; sub(/^case=/,"",id)
  if (!(id in expected) || seen[id]++ || $0 !~ ("^[[:space:]]+python_uv_run_cli_integration_test\\.go:[0-9]+: production_pipx_run_cli case=" id " " expected[id] "$")) bad=1
  cases++
}
/production_pipx_run_cli_source case=/ {
  if ($0 !~ /^[[:space:]]+python_uv_run_cli_integration_test\.go:[0-9]+: production_pipx_run_cli_source case=[1-46] name=[a-z0-9-]+ version=[0-9.]+ sha256=[0-9a-f]+ evidence=[0-9]+$/) bad=1
  id=$3; sub(/^case=/,"",id); name=$4; sub(/^name=/,"",name); version=$5; sub(/^version=/,"",version); hash=$6; sub(/^sha256=/,"",hash); evidence=$7; sub(/^evidence=/,"",evidence)
  if (length(hash)!=64 || evidence+0<3 || source[id " " name]++) bad=1
  if (id ~ /^[1-3]$/) { if (name!="cowsay" || version!="6.1" || hash!="274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a") bad=1 }
  else if (id=="4" || id=="6") { if (name=="python-slugify") { if (version!="8.0.4" || hash!="276540b79961052b66b7d116620b36518847f52d5fd9e3a70164fc8c50faa6b8") bad=1 } else if (name!="text-unidecode") bad=1 }
  else bad=1
  sources++
}
END { exit (bad || parent!=1 || cases!=6 || sources!=7 || !source["1 cowsay"] || !source["2 cowsay"] || !source["3 cowsay"] || !source["4 python-slugify"] || !source["4 text-unidecode"] || !source["6 python-slugify"] || !source["6 text-unidecode"]) }
' "$pipx_run_log"
rm -- "$pipx_run_log"

bash scripts/ci-pnpm-runtime.sh run

haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"
haa_start_policy_helper
uv_run_source_log="$RUNNER_TEMP/haa-uv-run-source.log"
uv_run_source_status=0
bash scripts/ci-uv-tools.sh uv-run-source >"$uv_run_source_log" 2>&1 || uv_run_source_status=$?
uv_run_source_log_status=0
cat -- "$uv_run_source_log" || uv_run_source_log_status=$?
if ((uv_run_source_status)); then exit "$uv_run_source_status"; fi
if ((uv_run_source_log_status)); then exit "$uv_run_source_log_status"; fi
awk '
BEGIN { expected["TestUVActualPublicToolRunSourceBinding"]=1; expected["TestUVActualPublicToolRunNativeCensus"]=1; expected["TestUVPublicToolRunResolverIntegration"]=1 }
/--- SKIP:/ { bad=1 }
/^--- PASS:/ {
  if (!($3 in expected) || accepted[$3]++) bad=1
  parents++
}
/uv_public_query_binding phase=/ {
  if ($0 !~ /^[[:space:]]+uv_tool_run_public_real_test\.go:[0-9]+: uv_public_query_binding phase=prepared source=ACTUAL_PUBLIC_INDEPENDENT whole_original_startup=BOUND nodes=1 wheels=1 readonly_source=SEALED selected_program=NOT_RUN Evidence=NOT_RUN Policy=NOT_RUN native_terminal=EACCES wheel_sha256=274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a$/ || source++) bad=1
}
/uv_native_source_console phase=/ {
  if ($0 !~ /^[[:space:]]+uv_tool_run_census_real_test\.go:[0-9]+: uv_native_source_console phase=bound original=WHOLE source=INDEPENDENT command=cowsay metadata_sha256=780e250c9ff0a37fe32ec832874a5ebb6c9692915baf899a94e549b4c195262c native_record=UNAPPROVED_DATA Policy=NOT_RUN program=NOT_RUN$/ || console++) bad=1
}
/uv_public_native_census phase=/ {
  if ($0 !~ /^[[:space:]]+uv_tool_run_census_real_test\.go:[0-9]+: uv_public_native_census phase=collected source=ACTUAL_PUBLIC_INDEPENDENT whole_original_startup=BOUND readonly_source=SEALED environments=1 facts=84 selected_program=NOT_RUN Evidence=NOT_RUN Policy=NOT_RUN selection_equivalence=NOT_QUALIFIED native_terminal=EACCES$/ || census++) bad=1
}
/uv_run_resolver_trace case=/ {
  if ($0 !~ /^[[:space:]]+uv_run_resolver_real_test\.go:[0-9]+: uv_run_resolver_trace case=(normal|incomplete-refused) phase=(uv-pip-resolver|pypi-wheel|frozen-offline) raw_complete=true delivered_complete=(true|false) injected_incomplete=(true|false) events=[0-9]+ bytes=[0-9]+ kinds=map\[.*\]$/) bad=1
  id=$3; sub(/^case=/,"",id); phase=$4; sub(/^phase=/,"",phase)
  if (traces[id " " phase]++) bad=1
  rejected=(id=="incomplete-refused" && phase=="frozen-offline")
  if ($6 != "delivered_complete=" (rejected ? "false" : "true") || $7 != "injected_incomplete=" (rejected ? "true" : "false")) bad=1
  trace_count++
}
/uv_run_resolver case=/ {
  if ($0 !~ /^[[:space:]]+uv_run_resolver_real_test\.go:[0-9]+: uv_run_resolver case=(normal|incomplete-refused) source=ACTUAL_PUBLIC_INDEPENDENT original=WHOLE readonly_source=SEALED native=ISOLATED_OFFLINE source_console=BOUND record=UNAPPROVED_DATA native_terminal=EACCES Policy=NOT_RUN program=NOT_RUN selection_returned=(true|false)$/) bad=1
  id=$3; sub(/^case=/,"",id)
  if (resolver[id]++ || $13 != "selection_returned=" (id=="normal" ? "true" : "false")) bad=1
  resolver_count++
}
END { exit (bad || parents != 3 || source != 1 || console != 1 || census != 1 || trace_count != 6 || resolver_count != 2) }
' "$uv_run_source_log"
rm -- "$uv_run_source_log"
`

func validateCurrentExecutionPreparationQualification(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", currentExecutionPreparationStep)
	if err != nil {
		return nil
	}
	if len(blocks) == 1 && kernelQualificationLines(strings.Join(blocks[0], "\n")) == kernelQualificationLines(currentExecutionPreparationRun) {
		return nil
	}
	return []string{"current execution preparation/refusal qualification requires one unconditional protected-client lifecycle, exact parents/readonly4/console2/boundary8/hostile5/workflow3/native-app3/native-cli5/pipx-app3/pipx-cli6/source3+markers3+resolver2+trace6/unique cases/no-SKIP and original-error guards"}
}
