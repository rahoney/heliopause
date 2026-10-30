package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPNPMProjectProfileBindsActualConsumerAndRejectsInvalidInput(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-pnpm-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for name, replacement := range map[string]string{
		"comment":                 "# " + pnpmBootstrapIntegrationCommand,
		"missing-transaction-env": strings.Replace(pnpmBootstrapIntegrationCommand, "HELOX_PNPM_PROJECT_TRANSACTION_INTEGRATION=1 ", "", 1),
		"missing-cache-env":       strings.Replace(pnpmBootstrapIntegrationCommand, "HELOX_PNPM_CACHE_INTEGRATION=1 ", "", 1),
		"wrong-filter":            strings.Replace(pnpmBootstrapIntegrationCommand, "ApprovedCache|ProjectTransaction", "Missing", 1),
		"ignore-failure":          pnpmBootstrapIntegrationCommand + " || true",
		"duplicate":               pnpmBootstrapIntegrationCommand + "\n" + pnpmBootstrapIntegrationCommand,
	} {
		t.Run(name, func(t *testing.T) {
			if len(validatePNPMRuntimeScript(strings.Replace(original, pnpmBootstrapIntegrationCommand, replacement, 1))) == 0 {
				t.Fatal("missing actual bootstrap coverage accepted")
			}
		})
	}
	for _, candidate := range []string{strings.Replace(original, `test "$#" -le 1`, `test "$#" -le 2`, 1), strings.Replace(original, "sandbox|project|run) ;;", "*) ;;", 1), strings.Replace(original, "if [[ \"$pnpm_profile\" == project ]]; then", "if true; then", 1), strings.Replace(original, pnpmOriginalIntegrationCommand+"\n  exit 0\nfi", pnpmOriginalIntegrationCommand+"\nfi", 1)} {
		if len(validatePNPMRuntimeScript(candidate)) == 0 {
			t.Fatal("unbounded or nonexclusive project profile accepted")
		}
	}
	// Exercise the real shell before any environment/tool/runtime preparation.
	command := exec.Command("bash", filepath.Join("..", "ci-pnpm-runtime.sh"), "unknown")
	output, err := command.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 || !strings.Contains(string(output), "unsupported PNPM CI profile") {
		t.Fatal("unknown profile reached preparation", string(output), err)
	}
	command = exec.Command("bash", filepath.Join("..", "ci-pnpm-runtime.sh"), "project", "extra")
	if err := command.Run(); err == nil {
		t.Fatal("extra argument reached preparation")
	}
}

func TestPNPMCLIActualCannotLoseAuthenticatedCommandOrCoverage(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-pnpm-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for name, replacement := range map[string]string{
		"comment":         "# " + pnpmCLIIntegrationCommand,
		"wrong-env":       strings.Replace(pnpmCLIIntegrationCommand, "CLI_INTEGRATION=1", "CLI_INTEGRATION=0", 1),
		"wrong-test":      strings.Replace(pnpmCLIIntegrationCommand, "TestPNPMProjectCLIIntegration", "TestMissingCLIIntegration", 1),
		"wrong-client":    strings.Replace(pnpmCLIIntegrationCommand, "/usr/libexec/heliopause/helox", "go test ./internal/bootstrap", 1),
		"ignored-failure": pnpmCLIIntegrationCommand + " || true",
		"duplicate":       pnpmCLIIntegrationCommand + "\n" + pnpmCLIIntegrationCommand,
	} {
		t.Run(name, func(t *testing.T) {
			if len(validatePNPMRuntimeScript(strings.Replace(original, pnpmCLIIntegrationCommand, replacement, 1))) == 0 {
				t.Fatal("CLI actual missing or untrusted execution accepted")
			}
		})
	}
}

func TestPNPMProjectActualRequiresMatchingClientAndParentRestoration(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validatePNPMProjectClientOrder(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	call := "          bash scripts/ci-pnpm-runtime.sh project\n"
	restore := call + "          haa_install_ci_client \"$RUNNER_TEMP/helox-sandbox-test\"\n          haa_start_policy_helper\n"
	if strings.Count(original, restore) != 1 {
		t.Fatal("no unique project return to sandbox client")
	}
	for name, candidate := range map[string]string{
		"missing-tag":             strings.Replace(original, pnpmBootstrapTestBuildCommand, strings.Replace(pnpmBootstrapTestBuildCommand, "-tags=realcorpus ", "", 1), 1),
		"wrong-package":           strings.Replace(original, pnpmBootstrapTestBuildCommand, strings.Replace(pnpmBootstrapTestBuildCommand, "./internal/bootstrap", "./internal/sandbox", 1), 1),
		"wrong-client":            strings.Replace(original, `haa_install_ci_client "$RUNNER_TEMP/helox-pnpm-project-test"`, `haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"`, 1),
		"missing-call":            strings.Replace(original, call, "", 1),
		"commented-call":          strings.Replace(original, call, "          # bash scripts/ci-pnpm-runtime.sh project\n", 1),
		"no-restore":              strings.Replace(original, restore, call, 1),
		"wrong-restore":           strings.Replace(original, restore, strings.Replace(restore, "helox-sandbox-test", "helox-bootstrap-test", 1), 1),
		"no-helper-after-restore": strings.Replace(original, restore, strings.TrimSuffix(restore, "          haa_start_policy_helper\n"), 1),
		"wrong-order":             strings.Replace(original, restore, strings.Replace(restore, call, "", 1)+call, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validatePNPMProjectClientOrder(candidate)) == 0 {
				t.Fatal("unprepared project client or lost restoration accepted")
			}
		})
	}
}

func TestPNPMRuntimeHelperKeepsSupplyAndActualCoverage(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-pnpm-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validatePNPMRuntimeScript(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	for name, replacement := range map[string]string{
		"comment":          "# " + pnpmRuntimeIntegrationCommand,
		"skip":             strings.Replace(pnpmRuntimeIntegrationCommand, "RUNTIME=1", "RUNTIME=0", 1),
		"missing-node":     strings.Replace(pnpmRuntimeIntegrationCommand, "|NodeQuery", "", 1),
		"missing-artifact": strings.Replace(pnpmRuntimeIntegrationCommand, "|ReleaseArtifactProbe", "", 1),
		"duplicate":        pnpmRuntimeIntegrationCommand + "\n" + pnpmRuntimeIntegrationCommand,
	} {
		t.Run(name, func(t *testing.T) {
			if findings := validatePNPMRuntimeScript(strings.Replace(original, pnpmRuntimeIntegrationCommand, replacement, 1)); len(findings) == 0 {
				t.Fatal("PNPM actual coverage was omitted")
			}
		})
	}
	if findings := validatePNPMRuntimeScript(strings.Replace(original, "go run ./scripts/update-runtime materialize-pnpm", "echo skipped-pnpm-materialization", 1)); len(findings) == 0 {
		t.Fatal("unverified PNPM supply accepted")
	}
}

func TestPNPMInspectionCommandsCannotLoseProjectOrControlledCoverage(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-pnpm-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{pnpmProjectIntegrationCommand, pnpmControlledIntegrationCommand, pnpmPublicQuarantineCommand} {
		for _, replacement := range []string{"# " + command, strings.Replace(command, "=1", "=0", 1), command + "\n" + command} {
			if len(validatePNPMRuntimeScript(strings.Replace(string(body), command, replacement, 1))) == 0 {
				t.Fatal("project/controlled execution omitted")
			}
		}
	}
	workflow, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	original := string(workflow)
	if findings := validatePNPMInspectionClientOrder(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	call := "          bash scripts/ci-pnpm-runtime.sh\n"
	moved := strings.Replace(original, call, "", 1)
	moved = strings.Replace(moved, "          "+pnpmSandboxTestBuildCommand, "          bash scripts/ci-pnpm-runtime.sh\n          "+pnpmSandboxTestBuildCommand, 1)
	for _, candidate := range []string{moved, strings.Replace(original, pnpmSandboxTestBuildCommand, strings.Replace(pnpmSandboxTestBuildCommand, "-tags=realcorpus ", "", 1), 1), strings.Replace(original, call, "          # bash scripts/ci-pnpm-runtime.sh\n", 1)} {
		if len(validatePNPMInspectionClientOrder(candidate)) == 0 {
			t.Fatal("unprepared client identity accepted")
		}
	}
}

func TestPNPMControlledFixtureRequiresPackageSubshell(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-pnpm-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	scope := "(\n  cd internal/sandbox\n  " + pnpmControlledIntegrationCommand + "\n)"
	if strings.Count(original, scope) != 1 {
		t.Fatal("controlled fixture has no unique package scope")
	}
	for name, replacement := range map[string]string{
		"missing cd":       strings.Replace(scope, "  cd internal/sandbox\n", "", 1),
		"wrong cd":         strings.Replace(scope, "cd internal/sandbox", "cd internal/cli", 1),
		"commented cd":     strings.Replace(scope, "cd internal/sandbox", "# cd internal/sandbox", 1),
		"unscoped":         pnpmControlledIntegrationCommand,
		"outside subshell": "cd internal/sandbox\n(\n  " + pnpmControlledIntegrationCommand + "\n)",
		"duplicate scope":  scope + "\n" + scope,
	} {
		t.Run(name, func(t *testing.T) {
			if len(validatePNPMRuntimeScript(strings.Replace(original, scope, replacement, 1))) == 0 {
				t.Fatal("incorrect controlled fixture cwd accepted")
			}
		})
	}
}

func TestPNPMOriginalActualCannotLoseAuthenticatedCommandOrCoverage(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-pnpm-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for name, replacement := range map[string]string{
		"comment":         "# " + pnpmOriginalIntegrationCommand,
		"wrong-env":       strings.Replace(pnpmOriginalIntegrationCommand, "ORIGINAL_INTEGRATION=1", "ORIGINAL_INTEGRATION=0", 1),
		"wrong-test":      strings.Replace(pnpmOriginalIntegrationCommand, "TestPNPMNativeOriginalIntegration", "TestMissingCLIIntegration", 1),
		"wrong-client":    strings.Replace(pnpmOriginalIntegrationCommand, "/usr/libexec/heliopause/helox", "go test ./internal/bootstrap", 1),
		"ignored-failure": pnpmOriginalIntegrationCommand + " || true",
		"duplicate":       pnpmOriginalIntegrationCommand + "\n" + pnpmOriginalIntegrationCommand,
	} {
		t.Run(name, func(t *testing.T) {
			if len(validatePNPMRuntimeScript(strings.Replace(original, pnpmOriginalIntegrationCommand, replacement, 1))) == 0 {
				t.Fatal("Native original actual missing or untrusted execution accepted")
			}
		})
	}
}

func TestPNPMRunProfileKeepsExactDispatchAndPrimaryFailure(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "ci-pnpm-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	for _, change := range []string{
		strings.Replace(pnpmRunIntegrationBlock, pnpmRunIntegrationCommand, "# "+pnpmRunIntegrationCommand, 1),
		strings.Replace(pnpmRunIntegrationBlock, "WORKFLOW_INTEGRATION=1", "WORKFLOW_INTEGRATION=0", 1),
		strings.Replace(pnpmRunIntegrationBlock, "Workflow|CLI(Refusal)?", "Missing", 1),
		strings.Replace(pnpmRunIntegrationBlock, `if ((pnpm_run_status)); then exit "$pnpm_run_status"; fi`, "", 1),
		strings.Replace(pnpmRunIntegrationBlock, "exit 0\nfi", "fi", 1),
	} {
		if len(validatePNPMRuntimeScript(strings.Replace(original, pnpmRunIntegrationBlock, change, 1))) == 0 {
			t.Fatal("incomplete PNPM run profile accepted")
		}
	}
	start := strings.Index(pnpmRunIntegrationBlock, "  pnpm_run_log_status=0")
	end := strings.Index(pnpmRunIntegrationBlock, "  awk '")
	if start < 0 || end <= start {
		t.Fatal("missing producer/log status boundary")
	}
	for _, status := range []string{"0", "42"} {
		command := exec.Command("bash", "-c", `pnpm_run_status=$1; pnpm_run_log=$2`+"\n"+pnpmRunIntegrationBlock[start:end], "pnpm-primary", status, filepath.Join(t.TempDir(), "missing.log"))
		err := command.Run()
		exit, ok := err.(*exec.ExitError)
		want := 1
		if status == "42" {
			want = 42
		}
		if !ok || exit.ExitCode() != want {
			t.Fatal("log readback replaced the original failure", status, err)
		}
	}
}

func TestPNPMRunGuardRejectsMissingDuplicateAndForeignCases(t *testing.T) {
	testNativeNodeRunGuardRejectsMissingDuplicateAndForeignCases(t, false)
}

func TestBunRunGuardRejectsMissingDuplicateAndForeignCases(t *testing.T) {
	testNativeNodeRunGuardRejectsMissingDuplicateAndForeignCases(t, true)
}

func testNativeNodeRunGuardRejectsMissingDuplicateAndForeignCases(t *testing.T, bun bool) {
	t.Helper()
	block := pnpmRunIntegrationBlock
	if bun {
		block = bunRunIntegrationBlock
	}
	start := strings.Index(block, "awk '\n")
	end := strings.Index(block[start:], "\n' ") + start
	if start < 0 || end <= start {
		t.Fatal("missing native run guard")
	}
	program := block[start+len("awk '\n") : end]
	fixture := `--- PASS: TestLinuxPNPMRunExecutionWorkflowIntegration (1s)
--- PASS: TestLinuxPNPMRunExecutionCLIIntegration (1s)
--- PASS: TestLinuxPNPMRunExecutionCLIRefusalIntegration (1s)
    node_execution_run_integration_test.go:1: production_pnpm_run_application attempt=1 artifact_role=true target_loaded=true exit_code=0 stdout_bytes=1 stderr_bytes=0 user_program_run=true
    node_execution_run_integration_test.go:1: production_pnpm_run_application attempt=2 artifact_role=true target_loaded=true exit_code=1 stdout_bytes=1 stderr_bytes=0 user_program_run=true
    npm_execution_integration_test.go:1: production_pnpm_run_cli case=1 alias=dlx format=json language=en show_output=false exit_code=0 current_policy=ALLOW guest_cleanup=CONFIRMED
    npm_execution_integration_test.go:1: production_pnpm_run_cli case=2 alias=pnx format=json language=ko show_output=true exit_code=1 current_policy=ALLOW guest_cleanup=CONFIRMED
    npm_execution_integration_test.go:1: production_pnpm_run_cli case=3 alias=pnpx format=human language=ko show_output=true exit_code=0 current_policy=ALLOW guest_cleanup=CONFIRMED
    npm_execution_integration_test.go:1: production_pnpm_run_cli_graph nodes=12 edges=13 primary=rimraf current_policy=ALLOW
    npm_execution_refusal_integration_test.go:1: production_pnpm_run_refusal case=1 reason=ambiguous current_policy=ALLOW preparation=INCOMPLETE runtime_attempted=false exit_absent=true output_bytes=0 stored_manifest=0
    npm_execution_refusal_integration_test.go:1: production_pnpm_run_refusal case=2 reason=missing current_policy=ALLOW preparation=INCOMPLETE runtime_attempted=false exit_absent=true output_bytes=0 stored_manifest=0
`
	for name, input := range map[string]string{
		"complete": fixture, "empty": "", "child-only": "    --- PASS: TestLinuxPNPMRunExecutionCLIIntegration/case-1 (1s)\n",
		"missing-parent":   strings.Replace(fixture, "--- PASS: TestLinuxPNPMRunExecutionCLIIntegration (1s)\n", "", 1),
		"duplicate":        fixture + "    npm_execution_integration_test.go:1: production_pnpm_run_cli_graph nodes=12 edges=13 primary=rimraf current_policy=ALLOW\n",
		"skip":             fixture + "--- SKIP: TestForeign (0s)\n",
		"unknown-case":     strings.Replace(fixture, "production_pnpm_run_cli case=3", "production_pnpm_run_cli case=4", 1),
		"wrong-guest-code": strings.Replace(fixture, "attempt=2 artifact_role=true target_loaded=true exit_code=1", "attempt=2 artifact_role=true target_loaded=true exit_code=0", 1),
		"foreign-source":   strings.Replace(fixture, "npm_execution_integration_test.go:1: production_pnpm_run_cli case=1", "foreign.go:1: production_pnpm_run_cli case=1", 1),
	} {
		if bun {
			input = strings.NewReplacer("PNPM", "Bun", "pnpm", "bun", "alias=dlx", "alias=bunx", "alias=pnx", "alias=x", "alias=pnpx", "alias=bunx").Replace(input)
		}
		t.Run(name, func(t *testing.T) {
			command := exec.Command("awk", program)
			command.Stdin = strings.NewReader(input)
			output, err := command.CombinedOutput()
			if (err == nil) != (name == "complete") {
				t.Fatal("incorrect PNPM cohort acceptance", string(output), err)
			}
		})
	}
}
