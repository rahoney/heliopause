package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
