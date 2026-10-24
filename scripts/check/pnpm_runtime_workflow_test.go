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
	for _, candidate := range []string{strings.Replace(original, `test "$#" -le 1`, `test "$#" -le 2`, 1), strings.Replace(original, "sandbox|project) ;;", "*) ;;", 1), strings.Replace(original, "if [[ \"$pnpm_profile\" == project ]]; then", "if true; then", 1), strings.Replace(original, "  exit 0\nfi", "fi", 1)} {
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
