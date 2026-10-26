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
		"skip native":          strings.Replace(original, "HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION=1", "HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION=0", 1),
		"skip current failure": strings.Replace(original, "HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE=1", "HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE=0", 1),
		"ignore bootstrap":     strings.Replace(original, uvToolBootstrapCommand, uvToolBootstrapCommand+" || true", 1),
		"comment native":       strings.Replace(original, uvToolNativeCommand, "# "+uvToolNativeCommand, 1),
		"wrong package":        strings.Replace(original, "cd internal/bootstrap", "cd internal/sandbox", 1),
		"omit data removal":    strings.Replace(original, "uv_tools_owned\nuv_tools_remove", "uv_tools_owned\necho kept", 1),
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
