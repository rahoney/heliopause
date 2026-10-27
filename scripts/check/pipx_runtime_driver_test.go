package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const pipxCIHarnessGo = "#!/bin/bash\nset -euo pipefail\nprintf 'go:%s\\n' \"$*\" >> \"$CALLS\"\nstage=$3\nif [[ \"$stage\" == -check ]]; then exit 0; fi\nif [[ \"$stage\" == fetch-pipx-supply && \"${FAIL_STAGE-}\" == fetch ]]; then exit 23; fi\noutput=''; notices=''\nwhile (($#)); do\n case \"$1\" in\n  --output) output=$2; shift 2 ;;\n  --notices-output) notices=$2; shift 2 ;;\n  *) shift ;;\n esac\ndone\nif [[ \"$stage\" == fetch-pipx-supply ]]; then\n mkdir -m 0700 -- \"$output\"\nelse\n printf 'synthetic DATA for shell dispatch only\\n' > \"$output\"\n printf 'synthetic notices for shell dispatch only\\n' > \"$notices\"\nfi\n"
const pipxCIHarnessClient = "#!/bin/bash\nset -euo pipefail\nprintf 'probe:%s\\n' \"$PWD\" >> \"$CALLS\"\nif [[ \"${HELOX_PIPX_GRAPH_INTEGRATION-}\" == 1 ]]; then\n names=TestPipxActualCompleteGraphCurrentEvidencePolicyAndStaging\nelse\n names=TestPipxAcquiredRuntimeRequiredInspectionIntegration\nfi\nif [[ \"${ROOT_PROBE-}\" == bootstrap ]]; then names='TestPipxBootstrapCopyRecoveryAndCleanupInDisposableNamespace TestInitialNativeFilesInDisposableNamespace TestProtectedStageInDisposableNamespace'; fi\nif [[ \"${ROOT_PROBE-}\" == pending ]]; then names=TestInterruptedSystemCopyInDisposableNamespace; fi\nfor name in $names; do\n case \"${REPORT_KIND-pass}\" in\n  empty) printf 'testing: warning: no tests to run\\n' ;;\n  skip) printf '%s\\n' \"--- SKIP: $name (0.00s)\" ;;\n  child) printf '    %s\\n' \"--- PASS: $name/child (0.00s)\" ;;\n  duplicate) printf '%s\\n' \"--- PASS: $name (0.00s)\" \"--- PASS: $name (0.00s)\" ;;\n  *) printf '%s\\n' \"--- PASS: $name (0.00s)\" ;;\n esac\ndone\nif [[ \"${REPORT_KIND-pass}\" == failure ]]; then exit 37; fi\n"
const pipxCIHarnessSudo = "#!/bin/bash\nset -euo pipefail\nprintf 'sudo:%s\\n' \"$*\" >> \"$CALLS\"\nif [[ \"$*\" == *helox-releaseinstall-test* ]]; then export ROOT_PROBE=pending; else export ROOT_PROBE=bootstrap; fi\nexec \"$CLIENT_STUB\"\n"

// Only the shell dispatch/cleanup boundary is under test. Frozen wheel/proof
// validation and actual Go clients use the separate explicit real-corpus runs.
func runPipxCIDriverHarness(t *testing.T, mode, report, failStage string, cleanupFailure bool) (int, string, string, int) {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	bin := filepath.Join(base, "bin")
	runner := filepath.Join(base, "runner")
	calls := filepath.Join(base, "calls")
	for _, name := range []string{bin, runner, filepath.Join(repo, "internal/bootstrap"), filepath.Join(repo, "internal/artifact/githubrelease"), filepath.Join(repo, "internal/releaseactivation"), filepath.Join(repo, "internal/releaseinstall"), filepath.Join(repo, "docs/planning/evidence/m13-003-release-build/v0.1.0")} {
		if err := os.MkdirAll(name, 0700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(name, []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
	}
	client := filepath.Join(bin, "client")
	write(client, pipxCIHarnessClient)
	write(filepath.Join(bin, "go"), pipxCIHarnessGo)
	write(filepath.Join(bin, "id"), "#!/bin/bash\nprintf '1001\\n'\n")
	write(filepath.Join(bin, "sudo"), pipxCIHarnessSudo)
	if cleanupFailure {
		write(filepath.Join(bin, "rm"), "#!/bin/bash\nexit 19\n")
	}
	write(filepath.Join(repo, "docs/planning/evidence/m13-003-release-build/v0.1.0/helox-release-sbom.cdx.json"), "{}")
	body, err := os.ReadFile(filepath.Join("..", "ci-pipx-runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// Replace only the concrete external test client in this temporary harness.
	// The production driver and the protected Host path remain untouched.
	driver := filepath.Join(base, "driver.sh")
	write(driver, strings.ReplaceAll(string(body), "/usr/libexec/heliopause/helox", client))
	command := exec.Command("bash", driver, mode)
	command.Dir = repo
	command.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RUNNER_TEMP="+runner, "CALLS="+calls, "CLIENT_STUB="+client, "REPORT_KIND="+report, "FAIL_STAGE="+failStage)
	out, err := command.CombinedOutput()
	status := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		status = exit.ExitCode()
	}
	recorded, _ := os.ReadFile(calls)
	entries, err := os.ReadDir(runner)
	if err != nil {
		t.Fatal(err)
	}
	return status, string(out), string(recorded), len(entries)
}

func TestPipxCIDriverDispatchRequiresExactFrameworkCompletion(t *testing.T) {
	for _, mode := range []string{"inspect", "graph", "namespace"} {
		for _, report := range []string{"pass", "empty", "skip", "child", "duplicate", "failure"} {
			t.Run(mode+"/"+report, func(t *testing.T) {
				status, out, calls, remaining := runPipxCIDriverHarness(t, mode, report, "", false)
				expected := 1
				if report == "pass" {
					expected = 0
				}
				if report == "failure" {
					expected = 37
				}
				if status != expected || remaining != 0 {
					t.Fatal("wrong completion/error/cleanup", status, expected, remaining, out)
				}
				want := 1
				if mode == "namespace" {
					want = 2
				}
				if report != "pass" && mode == "namespace" {
					want = 1
				}
				if strings.Count(calls, "probe:") != want {
					t.Fatal("wrong number of test dispatches", calls)
				}
				packageName := "internal/bootstrap"
				if mode == "graph" {
					packageName = "internal/artifact/githubrelease"
				}
				if mode == "namespace" {
					packageName = "internal/releaseactivation"
				}
				if !strings.Contains(calls, "/"+packageName+"\n") {
					t.Fatal("wrong package cwd", calls)
				}
			})
		}
	}
}
func TestPipxCIDriverPreservesFirstPreparationAndCleanupFailures(t *testing.T) {
	for _, cleanupFailure := range []bool{false, true} {
		status, out, calls, remaining := runPipxCIDriverHarness(t, "inspect", "pass", "fetch", cleanupFailure)
		if status != 23 || strings.Contains(calls, "probe:") || strings.Contains(calls, "materialize-pipx") {
			t.Fatal("preparation error replaced or consumer reached", status, out, calls)
		}
		if cleanupFailure && remaining != 1 || !cleanupFailure && remaining != 0 {
			t.Fatal("cleanup status differs", remaining, out)
		}
	}
	status, out, _, remaining := runPipxCIDriverHarness(t, "inspect", "pass", "", true)
	if status != 1 || remaining != 1 || !strings.Contains(out, "cleanup unproven") {
		t.Fatal("successful primary hid cleanup failure", status, remaining, out)
	}
}
