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
	if len(validateCurrentExecutionPreparationQualification(original[:start]+original[end:])) == 0 {
		t.Fatal("absent current qualification admitted")
	}
}
