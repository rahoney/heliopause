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
