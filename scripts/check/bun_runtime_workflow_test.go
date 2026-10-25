package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBunRuntimeSupplyAndActualConsumerBinding(t *testing.T) {
	path := filepath.Join("..", "ci-bun-runtime.sh")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validateBunRuntimeScript(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	for name, candidate := range map[string]string{
		"missing-Bun-JS-env":         strings.Replace(original, "HELOX_BUN_ARTIFACT_EXECUTION=1 ", "", 1),
		"commented-Bun-JS":           strings.Replace(original, bunArtifactExecutionCommand, "# "+bunArtifactExecutionCommand, 1),
		"echoed-Bun-JS":              strings.Replace(original, bunArtifactExecutionCommand, "echo skipped", 1),
		"ignored-Bun-JS-failure":     strings.Replace(original, bunArtifactExecutionCommand, bunArtifactExecutionCommand+" || true", 1),
		"duplicate-Bun-JS":           strings.Replace(original, bunArtifactExecutionCommand, bunArtifactExecutionCommand+"\n"+bunArtifactExecutionCommand, 1),
		"missing-native-env":         strings.Replace(original, "HELOX_BUN_NATIVE_RESOLVER_INTEGRATION=1 ", "", 1),
		"missing-quarantine-env":     strings.Replace(original, "HELOX_BUN_PROJECT_QUARANTINE=1 ", "", 1),
		"missing-materialize":        strings.Replace(original, "go run ./scripts/update-runtime materialize-bun", "echo skipped", 1),
		"wrong-native-filter":        strings.Replace(original, "TestBunPublicNativeResolverIntegration|TestBunActualNativeProjectSelectionBoundary", "Missing", 1),
		"ignored-quarantine-failure": strings.Replace(original, bunProjectQuarantineCommand, bunProjectQuarantineCommand+" || true", 1),
		"duplicate-native":           strings.Replace(original, bunNativeIntegrationCommand, bunNativeIntegrationCommand+"\n"+bunNativeIntegrationCommand, 1),
		"commented-quarantine":       strings.Replace(original, bunProjectQuarantineCommand, "# "+bunProjectQuarantineCommand, 1),
		"wrong-cwd":                  strings.Replace(original, "cd internal/sandbox", "cd internal/bootstrap", 1),
		"relaxed-absence":            strings.Replace(original, "test ! -e /usr/lib/heliopause", "echo existing-ok", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validateBunRuntimeScript(candidate)) == 0 {
				t.Fatal("invalid concrete Bun binding admitted")
			}
		})
	}
	if output, err := exec.Command("bash", "-n", path).CombinedOutput(); err != nil {
		t.Fatal("Bash syntax", string(output), err)
	}
	if err := exec.Command("bash", path, "unexpected").Run(); err == nil {
		t.Fatal("extra argument reached supply preparation")
	}
}

func TestBunActualFollowsProtectedSandboxRestoration(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validateBunInspectionClientOrder(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	call := "          bash scripts/ci-bun-runtime.sh\n"
	for name, candidate := range map[string]string{
		"omitted":            strings.Replace(original, call, "", 1),
		"commented":          strings.Replace(original, call, "          # bash scripts/ci-bun-runtime.sh\n", 1),
		"echoed":             strings.Replace(original, call, "          echo 'bash scripts/ci-bun-runtime.sh'\n", 1),
		"before-restoration": strings.Replace(strings.Replace(original, call, "", 1), "          bash scripts/ci-pnpm-runtime.sh project\n", call+"          bash scripts/ci-pnpm-runtime.sh project\n", 1),
		"helper-omitted":     strings.Replace(original, "          haa_start_policy_helper\n"+call, call, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validateBunInspectionClientOrder(candidate)) == 0 {
				t.Fatal("Bun execution without its owning client/helper sequence admitted")
			}
		})
	}
}

func TestBunProjectSupplyRequiresAllCurrentConsumersAndExactMode(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-bun-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for name, candidate := range map[string]string{
		"missing-cache":            strings.Replace(original, "HELOX_BUN_CACHE_INTEGRATION=1 ", "", 1),
		"missing-transaction":      strings.Replace(original, "HELOX_BUN_PROJECT_TRANSACTION_INTEGRATION=1 ", "", 1),
		"missing-cli":              strings.Replace(original, bunProjectCLICommand, "echo skipped", 1),
		"missing-original":         strings.Replace(original, bunOriginalProjectCommand, "echo skipped", 1),
		"ignored-original-failure": strings.Replace(original, bunOriginalProjectCommand, bunOriginalProjectCommand+" || true", 1),
		"duplicate-cli":            strings.Replace(original, bunProjectCLICommand, bunProjectCLICommand+"\n"+bunProjectCLICommand, 1),
		"wrong-project-cwd":        strings.Replace(original, "cd internal/bootstrap", "cd internal/sandbox", 1),
		"wrong-mode":               strings.Replace(original, `if [[ "$bun_profile" == project ]]; then`, `if [[ "$bun_profile" == missing ]]; then`, 1),
		"relaxed-arguments":        strings.Replace(original, `case "$#:${1-}" in`, `case "${1-}" in`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validateBunRuntimeScript(candidate)) == 0 {
				t.Fatal("incomplete project binding admitted")
			}
		})
	}
	path := filepath.Join("..", "ci-bun-runtime.sh")
	for _, args := range [][]string{{"unexpected"}, {"sandbox"}, {""}, {"project", "extra"}} {
		if err := exec.Command("bash", append([]string{path}, args...)...).Run(); err == nil {
			t.Fatal("unsupported supply arguments admitted", args)
		}
	}
}

func TestBunProjectActualRequiresBootstrapHelperAndSandboxRestoration(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validateBunProjectClientOrder(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	call := "          bash scripts/ci-bun-runtime.sh project\n"
	setup := "          haa_install_ci_client \"$RUNNER_TEMP/helox-pnpm-project-test\"\n          haa_start_policy_helper\n" + call
	for name, candidate := range map[string]string{
		"missing":             strings.Replace(original, call, "", 1),
		"commented":           strings.Replace(original, call, "          # bash scripts/ci-bun-runtime.sh project\n", 1),
		"echoed":              strings.Replace(original, call, "          echo 'bash scripts/ci-bun-runtime.sh project'\n", 1),
		"missing-helper":      strings.Replace(original, "          haa_start_policy_helper\n"+call, call, 1),
		"wrong-client":        strings.Replace(original, setup, strings.Replace(setup, "helox-pnpm-project-test", "helox-sandbox-test", 1), 1),
		"missing-restoration": strings.Replace(original, call+"          haa_install_ci_client \"$RUNNER_TEMP/helox-sandbox-test\"\n", call, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validateBunProjectClientOrder(candidate)) == 0 {
				t.Fatal("unsafe project lifecycle admitted")
			}
		})
	}
}
