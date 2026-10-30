package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUVToolsConcreteConsumersAndImageDataBinding(t *testing.T) {
	path := filepath.Join("..", "ci-uv-tools.sh")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validateUVToolsScript(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	for name, candidate := range map[string]string{
		"skip current readonly inputs":      strings.Replace(original, "HELOX_PYTHON_EXECUTION_INPUT_INTEGRATION=1", "HELOX_PYTHON_EXECUTION_INPUT_INTEGRATION=0", 1),
		"ignore current readonly inputs":    strings.Replace(original, pythonExecutionInputsCommand, pythonExecutionInputsCommand+" || true", 1),
		"wrong current readonly parent":     strings.Replace(original, "^TestLinuxPythonExecutionInputsIntegration$", "^TestLinuxNodeExecutionInputsIntegration$", 1),
		"readonly enters native":            strings.Replace(original, " && ${1-} != python-execution-inputs", "", 1),
		"skip selected Python runtime":      strings.Replace(original, "HELOX_PYTHON_EXECUTION_INTEGRATION=1", "HELOX_PYTHON_EXECUTION_INTEGRATION=0", 1),
		"wrong selected Python parent":      strings.Replace(original, "^TestLinuxPythonExecutionIntegration$", "^TestLinuxPythonExecutionInputsIntegration$", 1),
		"runtime enters native":             strings.Replace(original, " && ${1-} != python-execution &&", " &&", 1),
		"ignore selected Python runtime":    strings.Replace(original, pythonConsoleExecutionCommand, pythonConsoleExecutionCommand+" || true", 1),
		"skip current Python preparation":   strings.Replace(original, "HELOX_PYTHON_EXECUTION_PREPARATION_INTEGRATION=1", "HELOX_PYTHON_EXECUTION_PREPARATION_INTEGRATION=0", 1),
		"ignore current Python preparation": strings.Replace(original, pythonExecutionPreparationCommand, pythonExecutionPreparationCommand+" || true", 1),
		"wrong current Python parent":       strings.Replace(original, "^TestLinuxPythonExecutionPreparationIntegration$", "^TestLinuxNodeExecutionPreparationIntegration$", 1),
		"skip pipx native":                  strings.Replace(original, "HELOX_PIPX_TOOL_NATIVE_ADOPTION_INTEGRATION=1", "HELOX_PIPX_TOOL_NATIVE_ADOPTION_INTEGRATION=0", 1),
		"omit pipx native":                  strings.Replace(original, "TestLinuxPipxTool(Install|Upgrade|NativeAdoption)CLIIntegration", "TestLinuxPipxTool(Install|Upgrade)CLIIntegration", 1),
		"ignore pipx":                       strings.Replace(original, pipxToolBootstrapCommand, pipxToolBootstrapCommand+" || true", 1),
		"skip native":                       strings.Replace(original, "HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION=1", "HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION=0", 1),
		"skip current failure":              strings.Replace(original, "HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE=1", "HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE=0", 1),
		"skip native adoption":              strings.Replace(original, "HELOX_UV_TOOL_NATIVE_ADOPTION_INTEGRATION=1", "HELOX_UV_TOOL_NATIVE_ADOPTION_INTEGRATION=0", 1),
		"omit native adoption":              strings.Replace(original, "TestLinuxUVTool(Install|Upgrade|NativeAdoption)CLIIntegration", "TestLinuxUVTool(Install|Upgrade)CLIIntegration", 1),
		"ignore bootstrap":                  strings.Replace(original, uvToolBootstrapCommand, uvToolBootstrapCommand+" || true", 1),
		"comment native":                    strings.Replace(original, uvToolNativeCommand, "# "+uvToolNativeCommand, 1),
		"wrong package":                     strings.Replace(original, "cd internal/bootstrap", "cd internal/sandbox", 1),
		"omit data removal":                 strings.Replace(original, "uv_tools_owned\nuv_tools_remove", "uv_tools_owned\necho kept", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if candidate == original {
				t.Fatal("negative mutation missed its source")
			}
			if len(validateUVToolsScript(candidate)) == 0 {
				t.Fatal("invalid script admitted")
			}
		})
	}
	if output, err := exec.Command("bash", "-n", path).CombinedOutput(); err != nil {
		t.Fatal(string(output), err)
	}
}

func TestUVToolsProtectedClientAndHelperScopes(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "heliopause-ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validateUVToolsClientOrder(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	call := "          bash scripts/ci-uv-tools.sh bootstrap\n"
	for name, candidate := range map[string]string{
		"commented":       strings.Replace(original, call, "          # bash scripts/ci-uv-tools.sh bootstrap\n", 1),
		"ignored failure": strings.Replace(original, call, "          bash scripts/ci-uv-tools.sh bootstrap || true\n", 1),
		"wrong client":    strings.Replace(original, "          haa_install_ci_client \"$RUNNER_TEMP/helox-pnpm-project-test\"\n          haa_start_policy_helper\n"+call, "          haa_install_ci_client \"$RUNNER_TEMP/helox-bootstrap-test\"\n          haa_start_policy_helper\n"+call, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validateUVToolsClientOrder(candidate)) == 0 {
				t.Fatal("invalid owning scope admitted")
			}
		})
	}
}

func TestCurrentExecutionPreparationQualificationRejectsMissingOrWeakenedScope(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "heliopause-ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validateCurrentExecutionPreparationQualification(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	start := strings.Index(original, "      - name: "+currentExecutionPreparationStep+"\n")
	if start < 0 {
		t.Fatal("current step absent")
	}
	tail := strings.Index(original[start+1:], "      - name:")
	if tail < 0 {
		t.Fatal("following step absent")
	}
	end := start + 1 + tail
	owned := original[start:end]
	for name, changed := range map[string]string{
		"ignored readonly error": strings.Replace(owned, "if ((python_inputs_status)); then exit", "if ((0)); then exit", 1),
		"wrong readonly profile": strings.Replace(owned, "ci-uv-tools.sh python-execution-inputs", "ci-uv-tools.sh bootstrap", 1),
		"wrong readonly count":   strings.Replace(owned, "cases != 4", "cases != 3", 1),
		"wrong readonly IDs":     strings.Replace(owned, "1 2 3 4", "1 2 3 3", 1),
		"conditional":            strings.Replace(owned, "        run: |", "        if: false\n        run: |", 1),
		"wrong protected client": strings.Replace(owned, "helox-pnpm-project-test", "helox-bootstrap-test", 1),
		"missing helper":         strings.Replace(owned, "          haa_start_policy_helper\n", "", 1),
		"missing Node flag":      strings.Replace(owned, "HELOX_NPM_EXECUTION_CLI_REFUSAL_INTEGRATION=1", "HELOX_NPM_EXECUTION_CLI_REFUSAL_INTEGRATION=0", 1),
		"ignored Node error":     strings.Replace(owned, "if ((node_refusal_status)); then exit", "if ((0)); then exit", 1),
		"ignored Python error":   strings.Replace(owned, "if ((python_preparation_status)); then exit", "if ((0)); then exit", 1),
		"wrong Python profile":   strings.Replace(owned, "ci-uv-tools.sh python-execution-preparation", "ci-uv-tools.sh bootstrap", 1),
		"missing noSkip":         strings.Replace(owned, "/--- SKIP:/ { bad=1 }", "", 1),
		"missing uniqueness":     strings.Replace(owned, " || accepted[case_name]++", "", 1),
		"wrong case count":       strings.Replace(owned, "cases != 3", "cases != 2", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if changed == owned {
				t.Fatal("negative mutation missed its source")
			}
			candidate := original[:start] + changed + original[end:]
			if len(validateCurrentExecutionPreparationQualification(candidate)) == 0 {
				t.Fatal("weakened execution qualification admitted")
			}
		})
	}
	runtimeStart := strings.Index(owned, "          python_runtime_log=")
	if runtimeStart < 0 {
		t.Fatal("selected runtime scope absent")
	}
	runtimeOwned := owned[runtimeStart:]
	for name, changed := range map[string]string{
		"ignored runtime command error": strings.Replace(runtimeOwned, "if ((python_runtime_status)); then exit", "if ((0)); then exit", 1),
		"ignored runtime cat error":     strings.Replace(runtimeOwned, "if ((python_runtime_log_status)); then exit", "if ((0)); then exit", 1),
		"wrong runtime profile":         strings.Replace(runtimeOwned, "ci-uv-tools.sh python-execution >", "ci-uv-tools.sh python-execution-inputs >", 1),
		"wrong runtime count":           strings.Replace(runtimeOwned, "cases != 2", "cases != 1", 1),
		"duplicate runtime IDs":         strings.Replace(runtimeOwned, "1 2", "1 1", 1),
		"missing runtime noSkip":        strings.Replace(runtimeOwned, "/--- SKIP:/ { bad=1 }", "", 1),
		"missing runtime uniqueness":    strings.Replace(runtimeOwned, " || accepted[case_name]++", "", 1),
		"wrong runtime parent":          strings.Replace(runtimeOwned, "TestLinuxPythonExecutionIntegration", "TestLinuxPythonExecutionInputsIntegration", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if changed == runtimeOwned {
				t.Fatal("runtime negative mutation missed its source")
			}
			candidate := original[:start] + owned[:runtimeStart] + changed + original[end:]
			if len(validateCurrentExecutionPreparationQualification(candidate)) == 0 {
				t.Fatal("weakened selected runtime qualification admitted")
			}
		})
	}
	if len(validateCurrentExecutionPreparationQualification(original[:start]+owned[:runtimeStart]+original[end:])) == 0 {
		t.Fatal("missing selected runtime parent admitted")
	}
	if len(validateCurrentExecutionPreparationQualification(original[:start]+original[end:])) == 0 {
		t.Fatal("absent current qualification admitted")
	}
}

func TestUVToolsCurrentPythonCohortDispatch(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-uv-tools.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for _, profile := range []struct{ name, flag, parent, command string }{
		{"boundary", "HELOX_PYTHON_EXECUTION_BOUNDARY_INTEGRATION=1", "^TestLinuxPythonExecutionBoundaryIntegration$", pythonExecutionBoundaryCommand},
		{"hostile", "HELOX_PYTHON_EXECUTION_HOSTILE_INTEGRATION=1", "^TestLinuxPythonExecutionHostileIntegration$", pythonExecutionHostileCommand},
		{"workflow", "HELOX_PYTHON_EXECUTION_WORKFLOW_INTEGRATION=1", "^TestLinuxPythonExecutionWorkflowIntegration$", pythonExecutionWorkflowCommand},
	} {
		t.Run(profile.name, func(t *testing.T) {
			for name, candidate := range map[string]string{
				"disabled":        strings.Replace(original, profile.flag, strings.TrimSuffix(profile.flag, "1")+"0", 1),
				"wrong parent":    strings.Replace(original, profile.parent, "^TestLinuxPythonExecutionIntegration$", 1),
				"ignored error":   strings.Replace(original, profile.command, profile.command+" || true", 1),
				"missing branch":  strings.Replace(original, "if [[ ${1-} == python-execution-"+profile.name+" ]]; then", "if false; then", 1),
				"native fallback": strings.Replace(original, " && ${1-} != python-execution-"+profile.name, "", 1),
			} {
				t.Run(name, func(t *testing.T) {
					if candidate == original {
						t.Fatal("mutation missed source")
					}
					if len(validateUVToolsScript(candidate)) == 0 {
						t.Fatal("invalid cohort dispatch admitted")
					}
				})
			}
		})
	}
}

func TestCurrentExecutionPythonCohortRejectsWeakenedLogs(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "heliopause-ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for _, profile := range []struct{ name, parent, ids, count string }{
		{"boundary", "TestLinuxPythonExecutionBoundaryIntegration", "argv exit37 term kill cancel stdout-overflow stderr-overflow deadline", "8"},
		{"hostile", "TestLinuxPythonExecutionHostileIntegration", "outside-read network clone-exec comm-spoof readonly", "5"},
		{"workflow", "TestLinuxPythonExecutionWorkflowIntegration", "1 2 3", "3"},
	} {
		t.Run(profile.name, func(t *testing.T) {
			variable := "python_" + profile.name
			start := strings.Index(original, "          "+variable+"_log=")
			if start < 0 {
				t.Fatal("log scope absent")
			}
			closeLine := "          rm -- \"$" + variable + "_log\"\n"
			length := strings.Index(original[start:], closeLine)
			if length < 0 {
				t.Fatal("log end absent")
			}
			end := start + length + len(closeLine)
			owned := original[start:end]
			for name, changed := range map[string]string{
				"command error":      strings.Replace(owned, "if (("+variable+"_status)); then exit", "if ((0)); then exit", 1),
				"cat error":          strings.Replace(owned, "if (("+variable+"_log_status)); then exit", "if ((0)); then exit", 1),
				"wrong profile":      strings.Replace(owned, "ci-uv-tools.sh python-execution-"+profile.name, "ci-uv-tools.sh bootstrap", 1),
				"wrong count":        strings.Replace(owned, "cases != "+profile.count, "cases != 0", 1),
				"wrong IDs":          strings.Replace(owned, profile.ids, "foreign", 1),
				"SKIP accepted":      strings.Replace(owned, "/--- SKIP:/ { bad=1 }", "", 1),
				"duplicate accepted": strings.Replace(owned, " || accepted[case_name]++", "", 1),
				"child parent":       strings.Replace(owned, profile.parent+" ", profile.parent+"/child ", 1),
			} {
				t.Run(name, func(t *testing.T) {
					if changed == owned {
						t.Fatal("mutation missed source")
					}
					if len(validateCurrentExecutionPreparationQualification(original[:start]+changed+original[end:])) == 0 {
						t.Fatal("weakened cohort log admitted")
					}
				})
			}
			if len(validateCurrentExecutionPreparationQualification(original[:start]+original[end:])) == 0 {
				t.Fatal("missing cohort admitted")
			}
		})
	}
}

func TestUVRunSourceHostedDispatchAndLogs(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-uv-tools.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(body)
	for name, changed := range map[string]string{
		"disabled native": strings.Replace(script, "HELOX_UV_NATIVE_RUNTIME=1", "HELOX_UV_NATIVE_RUNTIME=0", 1),
		"disabled source": strings.Replace(script, "HELOX_UV_TOOL_RUN_PUBLIC_BINDING_INTEGRATION=1", "HELOX_UV_TOOL_RUN_PUBLIC_BINDING_INTEGRATION=0", 1),
		"disabled census": strings.Replace(script, "HELOX_UV_TOOL_RUN_NATIVE_CENSUS_INTEGRATION=1", "HELOX_UV_TOOL_RUN_NATIVE_CENSUS_INTEGRATION=0", 1),
		"wrong parent":    strings.Replace(script, "^(TestUVActualPublicToolRun(SourceBinding|NativeCensus)|TestUVPublicToolRunResolverIntegration)$", "^TestUVActualPublicToolRunSourceBinding$", 1),
		"wrong package":   strings.Replace(script, "cd internal/sandbox", "cd internal/bootstrap", 1),
		"ignored error":   strings.Replace(script, uvRunSourceCommand, uvRunSourceCommand+" || true", 1),
		"missing branch":  strings.Replace(script, "if [[ ${1-} == uv-run-source ]]; then", "if false; then", 1),
		"native fallback": strings.Replace(script, " && ${1-} != uv-run-source", "", 1),
		"unknown profile": strings.Replace(script, "|1:uv-run-source|1:uv-run-workflow|1:uv-run-cli|1:pipx-run-workflow|1:pipx-run-cli) ;;", "|1:anything) ;;", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if changed == script {
				t.Fatal("mutation missed source")
			}
			if len(validateUVToolsScript(changed)) == 0 {
				t.Fatal("weakened native dispatch admitted")
			}
		})
	}
	body, err = os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "heliopause-ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	start := strings.Index(original, "          uv_run_source_log=")
	closeLine := "          rm -- \"$uv_run_source_log\"\n"
	if start < 0 {
		t.Fatal("native scope absent")
	}
	length := strings.Index(original[start:], closeLine)
	if length < 0 {
		t.Fatal("native scope end absent")
	}
	end := start + length + len(closeLine)
	owned := original[start:end]
	for name, changed := range map[string]string{
		"command error":         strings.Replace(owned, "if ((uv_run_source_status)); then exit", "if ((0)); then exit", 1),
		"cat error":             strings.Replace(owned, "if ((uv_run_source_log_status)); then exit", "if ((0)); then exit", 1),
		"wrong profile":         strings.Replace(owned, "ci-uv-tools.sh uv-run-source", "ci-uv-tools.sh bootstrap", 1),
		"missing source parent": strings.Replace(owned, "expected[\"TestUVActualPublicToolRunSourceBinding\"]=1; ", "", 1),
		"missing census parent": strings.Replace(owned, "expected[\"TestUVActualPublicToolRunNativeCensus\"]=1", "", 1),
		"wrong count":           strings.Replace(owned, "parents != 3", "parents != 1", 1),
		"duplicate parent":      strings.Replace(owned, " || accepted[$3]++", "", 1),
		"skip accepted":         strings.Replace(owned, "/--- SKIP:/ { bad=1 }", "", 1),
		"source missing":        strings.Replace(owned, "source != 1", "source != 0", 1),
		"console missing":       strings.Replace(owned, "console != 1", "console != 0", 1),
		"census missing":        strings.Replace(owned, "census != 1", "census != 0", 1),
		"source hash":           strings.Replace(owned, "274b1e6fc1b966d53976333eb90ac94cb07a450a700b455af9fbdf882244b30a", "foreign", 1),
		"console hash":          strings.Replace(owned, "780e250c9ff0a37fe32ec832874a5ebb6c9692915baf899a94e549b4c195262c", "foreign", 1),
		"wrong census":          strings.Replace(owned, "environments=1 facts=84", "environments=0 facts=0", 1),
		"promoted record":       strings.Replace(owned, "native_record=UNAPPROVED_DATA", "native_record=APPROVED", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if changed == owned {
				t.Fatal("mutation missed source")
			}
			if len(validateCurrentExecutionPreparationQualification(original[:start]+changed+original[end:])) == 0 {
				t.Fatal("weakened native qualification admitted")
			}
		})
	}
	if len(validateCurrentExecutionPreparationQualification(original[:start]+original[end:])) == 0 {
		t.Fatal("missing native consumer admitted")
	}
	rebind := "haa_install_ci_client \"$RUNNER_TEMP/helox-sandbox-test\"\n          haa_start_policy_helper\n          uv_run_source_log="
	for _, changed := range []string{strings.Replace(original, rebind, strings.Replace(rebind, "helox-sandbox-test", "helox-pnpm-project-test", 1), 1), strings.Replace(original, rebind, strings.Replace(rebind, "          haa_start_policy_helper\n", "", 1), 1)} {
		if changed == original {
			t.Fatal("rebind mutation missed source")
		}
		if len(validateCurrentExecutionPreparationQualification(changed)) == 0 {
			t.Fatal("wrong native client/helper admitted")
		}
	}
}

func TestUVRunCurrentDispatchAndCohortGuards(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-uv-tools.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for name, changed := range map[string]string{
		"disabled selector":         strings.Replace(original, "HELOX_UV_TOOL_RUN_RESOLVER_INTEGRATION=1", "HELOX_UV_TOOL_RUN_RESOLVER_INTEGRATION=0", 1),
		"selector parent absent":    strings.Replace(original, "|TestUVPublicToolRunResolverIntegration)", ")", 1),
		"disabled current app":      strings.Replace(original, "HELOX_PYTHON_UV_RUN_EXECUTION_WORKFLOW_INTEGRATION=1", "HELOX_PYTHON_UV_RUN_EXECUTION_WORKFLOW_INTEGRATION=0", 1),
		"disabled CLI":              strings.Replace(original, "HELOX_UV_RUN_EXECUTION_CLI_INTEGRATION=1", "HELOX_UV_RUN_EXECUTION_CLI_INTEGRATION=0", 1),
		"CLI error ignored":         strings.Replace(original, uvRunCLICommand, uvRunCLICommand+" || true", 1),
		"current app error ignored": strings.Replace(original, uvRunWorkflowCommand, uvRunWorkflowCommand+" || true", 1),
		"CLI branch absent":         strings.Replace(original, "if [[ ${1-} == uv-run-cli || ${1-} == pipx-run-cli ]]; then", "if false; then", 1),
		"current app branch absent": strings.Replace(original, "if [[ ${1-} == uv-run-workflow || ${1-} == pipx-run-workflow ]]; then", "if false; then", 1),
		"app enters native":         strings.Replace(original, " && ${1-} != uv-run-workflow", "", 1),
		"pipx app enters native":    strings.Replace(original, " && ${1-} != pipx-run-workflow", "", 1),
		"pipx CLI ignored":          strings.Replace(original, pipxRunCLICommand, pipxRunCLICommand+" || true", 1),
		"pipx app disabled":         strings.Replace(original, "HELOX_PYTHON_PIPX_RUN_EXECUTION_WORKFLOW_INTEGRATION=1", "HELOX_PYTHON_PIPX_RUN_EXECUTION_WORKFLOW_INTEGRATION=0", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if changed == original {
				t.Fatal("mutation missed source")
			}
			if len(validateUVToolsScript(changed)) == 0 {
				t.Fatal("weakened native current dispatch admitted")
			}
		})
	}
	body, err = os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "heliopause-ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	original = string(body)
	for _, key := range []string{"uv_run_workflow", "uv_run_cli", "pipx_run"} {
		t.Run(key, func(t *testing.T) {
			start := strings.Index(original, "          "+key+"_log=")
			if start < 0 {
				t.Fatal("cohort absent")
			}
			closeLine := "          rm -- \"$" + key + "_log\"\n"
			length := strings.Index(original[start:], closeLine)
			if length < 0 {
				t.Fatal("cohort end absent")
			}
			end := start + length + len(closeLine)
			owned := original[start:end]
			mutations := map[string]string{
				"command failure ignored":  strings.Replace(owned, "if (("+key+"_status)); then exit", "if ((0)); then exit", 1),
				"log read failure ignored": strings.Replace(owned, "if (("+key+"_log_status)); then exit", "if ((0)); then exit", 1),
				"SKIP admitted":            strings.Replace(owned, "/--- SKIP:/ { bad=1 }", "", 1),
				"wrong parent":             strings.Replace(owned, "/^--- PASS:/", "/^--- FAKE:/", 1),
			}
			for name, changed := range mutations {
				t.Run(name, func(t *testing.T) {
					if changed == owned {
						t.Fatal("mutation missed source")
					}
					if len(validateCurrentExecutionPreparationQualification(original[:start]+changed+original[end:])) == 0 {
						t.Fatal("weakened native current cohort admitted")
					}
				})
			}
			if len(validateCurrentExecutionPreparationQualification(original[:start]+original[end:])) == 0 {
				t.Fatal("missing native current cohort admitted")
			}
		})
	}
}
