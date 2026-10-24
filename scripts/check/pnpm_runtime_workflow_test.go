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
