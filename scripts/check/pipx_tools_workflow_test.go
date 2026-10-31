package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipxInstalledToolsConcreteNamespaceAndOrdinaryConsumers(t *testing.T) {
	path := filepath.Join("..", "ci-pipx-tools.sh")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validatePipxInstalledToolsScript(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	for name, candidate := range map[string]string{
		"host namespace":                    strings.Replace(original, `test "$(readlink /proc/self/ns/mnt)" != "$(readlink /proc/1/ns/mnt)"`, "true", 1),
		"root child":                        strings.Replace(original, `--reuid="$user_id"`, `--reuid=0`, 1),
		"writable supply":                   strings.Replace(original, "remount,bind,ro,noexec,nosuid,nodev", "remount,bind,rw", 1),
		"wrong data owner":                  strings.Replace(original, "install -m 0444 -o 0 -g 0", "install -m 0644 -o 1001 -g 1001", 1),
		"missing namespace reference check": strings.Replace(original, `if [[ "$current" == "$namespace" ]]; then cleanup_status=1; fi`, `true`, 1),
		"erased primary":                    strings.Replace(original, `if ((primary == 0)); then primary=$cleanup_status; fi`, `primary=$cleanup_status`, 1),
		"missing frozen input":              strings.Replace(original, `HELOX_PIPX_QUERY_SUPPLY_ROOT="$(dirname -- "$input")/supply"`, "", 1),
		"skip actual":                       strings.Replace(original, "HELOX_PIPX_PUBLIC_UPGRADE_INTEGRATION=1", "HELOX_PIPX_PUBLIC_UPGRADE_INTEGRATION=0", 1),
		"missing native parent":             strings.Replace(original, "|TestLinuxPipxPublicOriginalUpgradeIntegration)", ")", 1),
		"echo native":                       strings.Replace(original, pipxInstalledNativeCommand, "echo "+pipxInstalledNativeCommand, 1),
		"ignored native":                    strings.Replace(original, pipxInstalledNativeCommand, pipxInstalledNativeCommand+" || true", 1),
		"duplicate native":                  strings.Replace(original, pipxInstalledNativeCommand, pipxInstalledNativeCommand+"\n"+pipxInstalledNativeCommand, 1),
		"wrong package":                     strings.Replace(original, "cd internal/sandbox", "cd internal/bootstrap", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if candidate == original {
				t.Fatal("negative made no change")
			}
			if len(validatePipxInstalledToolsScript(candidate)) == 0 {
				t.Fatal("invalid installed tools fixture admitted")
			}
		})
	}
	if out, err := exec.Command("bash", "-n", path).CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
}
