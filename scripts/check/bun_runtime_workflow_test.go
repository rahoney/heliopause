package main

import (
	"fmt"
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
		"missing-program-boundary-env": strings.Replace(original, "HELOX_BUN_PROGRAM_EXECUTION_BOUNDARY_INTEGRATION=1 ", "", 1),
		"wrong-program-filter":         strings.Replace(original, "Workflow|CLI|Boundary", "Workflow|CLI", 1),
		"missing-current-run-env":      strings.Replace(original, "HELOX_BUN_RUN_EXECUTION_CLI_INTEGRATION=1 ", "", 1),
		"case-sensitive-current-env":   strings.Replace(original, "HELOX_BUN_RUN_EXECUTION_WORKFLOW_INTEGRATION=1", "HELOX_Bun_RUN_EXECUTION_WORKFLOW_INTEGRATION=1", 1),
		"missing-Bun-JS-env":           strings.Replace(original, "HELOX_BUN_ARTIFACT_EXECUTION=1 ", "", 1),
		"commented-Bun-JS":             strings.Replace(original, bunArtifactExecutionCommand, "# "+bunArtifactExecutionCommand, 1),
		"echoed-Bun-JS":                strings.Replace(original, bunArtifactExecutionCommand, "echo skipped", 1),
		"ignored-Bun-JS-failure":       strings.Replace(original, bunArtifactExecutionCommand, bunArtifactExecutionCommand+" || true", 1),
		"duplicate-Bun-JS":             strings.Replace(original, bunArtifactExecutionCommand, bunArtifactExecutionCommand+"\n"+bunArtifactExecutionCommand, 1),
		"missing-native-env":           strings.Replace(original, "HELOX_BUN_NATIVE_RESOLVER_INTEGRATION=1 ", "", 1),
		"missing-quarantine-env":       strings.Replace(original, "HELOX_BUN_PROJECT_QUARANTINE=1 ", "", 1),
		"missing-materialize":          strings.Replace(original, "go run ./scripts/update-runtime materialize-bun", "echo skipped", 1),
		"wrong-native-filter":          strings.Replace(original, "TestBunPublicNativeResolverIntegration|TestBunActualNativeProjectSelectionBoundary", "Missing", 1),
		"ignored-quarantine-failure":   strings.Replace(original, bunProjectQuarantineCommand, bunProjectQuarantineCommand+" || true", 1),
		"duplicate-native":             strings.Replace(original, bunNativeIntegrationCommand, bunNativeIntegrationCommand+"\n"+bunNativeIntegrationCommand, 1),
		"commented-quarantine":         strings.Replace(original, bunProjectQuarantineCommand, "# "+bunProjectQuarantineCommand, 1),
		"wrong-cwd":                    strings.Replace(original, "cd internal/sandbox", "cd internal/bootstrap", 1),
		"relaxed-absence":              strings.Replace(original, "test ! -e /usr/lib/heliopause", "echo existing-ok", 1),
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

func TestBunProgramGuardRequiresCompleteDistinctActualCohort(t *testing.T) {
	start := strings.Index(bunProgramIntegrationBlock, "awk '\n")
	end := strings.Index(bunProgramIntegrationBlock[start:], "\n' ") + start
	program := bunProgramIntegrationBlock[start+len("awk '\n") : end]
	fixture := `--- PASS: TestLinuxBunProgramExecutionWorkflowIntegration (1s)
--- PASS: TestLinuxBunProgramExecutionCLIIntegration (1s)
--- PASS: TestLinuxBunProgramExecutionBoundaryIntegration (1s)
    node_execution_run_integration_test.go:1: production_bun_program_application attempt=1 artifact_role=true target_loaded=true exit_code=0 stdout_bytes=1 stderr_bytes=0 user_program_run=true
    node_execution_run_integration_test.go:1: production_bun_program_application attempt=2 artifact_role=true target_loaded=true exit_code=1 stdout_bytes=1 stderr_bytes=0 user_program_run=true
    npm_execution_integration_test.go:1: production_bun_program_cli case=1 alias=bunx format=json language=en show_output=false exit_code=0 current_policy=ALLOW guest_cleanup=CONFIRMED
    npm_execution_integration_test.go:1: production_bun_program_cli case=2 alias=x format=json language=ko show_output=true exit_code=1 current_policy=ALLOW guest_cleanup=CONFIRMED
    npm_execution_integration_test.go:1: production_bun_program_cli case=3 alias=bunx format=human language=ko show_output=true exit_code=0 current_policy=ALLOW guest_cleanup=CONFIRMED
    npm_execution_integration_test.go:1: production_bun_program_cli_graph nodes=12 edges=13 primary=rimraf current_policy=ALLOW
`
	boundary := func(mode string, complete bool) string {
		return fmt.Sprintf("    node_execution_boundary_integration_test.go:1: bun_program_boundary case=%s current_policy=ALLOW fixture_registry=CONTROLLED_TEST native_solver=NOT_RUN complete=%t stdout_bytes=0 stderr_bytes=0\n", mode, complete)
	}
	for _, mode := range []string{"argv", "term", "kill", "readonly", "cancel", "stdout-overflow", "stderr-overflow", "outside-read", "network", "child", "comm-spoof", "deadline"} {
		fixture += boundary(mode, mode == "argv" || mode == "term" || mode == "kill" || mode == "readonly")
	}
	for name, input := range map[string]string{
		"complete":           fixture,
		"empty":              "",
		"missing-parent":     strings.Replace(fixture, "--- PASS: TestLinuxBunProgramExecutionBoundaryIntegration (1s)\n", "", 1),
		"missing-boundary":   strings.Replace(fixture, boundary("network", false), "", 1),
		"duplicate-boundary": fixture + boundary("network", false),
		"unknown-boundary":   strings.Replace(fixture, "case=network", "case=foreign", 1),
		"hostile-complete":   strings.Replace(fixture, boundary("network", false), boundary("network", true), 1),
		"wrong-program-exit": strings.Replace(fixture, "attempt=2 artifact_role=true target_loaded=true exit_code=1", "attempt=2 artifact_role=true target_loaded=true exit_code=0", 1),
		"foreign-source":     strings.Replace(fixture, "node_execution_boundary_integration_test.go:1:", "foreign.go:1:", 1),
		"skip":               fixture + "--- SKIP: TestForeign (0s)\n",
	} {
		t.Run(name, func(t *testing.T) {
			command := exec.Command("awk", program)
			command.Stdin = strings.NewReader(input)
			output, err := command.CombinedOutput()
			if (err == nil) != (name == "complete") {
				t.Fatal("incorrect Bun program cohort acceptance", string(output), err)
			}
		})
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
