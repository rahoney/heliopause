package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestKernelTerminalQualificationRejectsInactiveOrIncompleteWorkflow(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	original := string(b)
	selected := selectedGoForTest(t)
	if got := validateCIWorkflow(original, selected); len(got) != 0 {
		t.Fatal(got)
	}
	marker := "      - name: " + kernelTerminalQualificationStep + "\n"
	for name, changed := range map[string]string{
		"omitted":           strings.Replace(original, marker, "      - name: omitted kernel qualification\n", 1),
		"conditional":       strings.Replace(original, marker, marker+"        if: false\n", 1),
		"ignored failure":   strings.Replace(original, marker, marker+"        continue-on-error: true\n", 1),
		"commented command": strings.Replace(original, kernelTerminalQualificationCommand, "# "+kernelTerminalQualificationCommand, 1),
		"duplicate command": strings.Replace(original, kernelTerminalQualificationCommand, kernelTerminalQualificationCommand+"\n            "+kernelTerminalQualificationCommand, 1),
		"wrong client":      strings.Replace(original, "          kernel_terminal_log=", "          haa_install_ci_client \"$RUNNER_TEMP/helox\"\n          kernel_terminal_log=", 1),
		"skip accepted":     strings.Replace(original, "END { exit (bad || seen != 1 || cases != 8) }", "END { exit (seen != 1 || cases != 8) }", 1),
		"partial cases":     strings.Replace(original, "cases != 8", "cases != 7", 1),
		"discard primary":   strings.Replace(original, "if ((kernel_terminal_status)); then exit \"$kernel_terminal_status\"; fi", "true", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if got := validateCIWorkflow(changed, selected); len(got) == 0 {
				t.Fatal("missing executed kernel coverage accepted")
			}
		})
	}
}

// Run the actual fixed status/log guard with inert command/lifecycle stubs.
// This does not launch a helper/Docker or claim runtime qualification.
func TestKernelTerminalQualificationActualBashGuard(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("Bash unavailable")
	}
	parent := "--- PASS: TestLinuxDirectExecKernelTerminalIntegration (0.01s)\n"
	cases := strings.Repeat("kernel_terminal case=fixed\n", 8)
	for _, tc := range []struct {
		name, log    string
		status, want int
	}{
		{"complete", cases + parent, 0, 0},
		{"primary failure", cases + parent, 37, 37},
		{"skipped", cases + parent + "--- SKIP: another\n", 0, 1},
		{"missing parent", cases, 0, 1},
		{"duplicate parent", cases + parent + parent, 0, 1},
		{"missing case", strings.Repeat("kernel_terminal case=fixed\n", 7) + parent, 0, 1},
		{"extra case", cases + "kernel_terminal case=extra\n" + parent, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			work := t.TempDir()
			for _, dir := range []string{"scripts", "internal/sandbox", "runner"} {
				if err := os.MkdirAll(filepath.Join(work, dir), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			stub := "haa_ci_initialize() { :; }\nhaa_install_ci_client() { :; }\nhaa_start_policy_helper() { :; }\n"
			if err := os.WriteFile(filepath.Join(work, "scripts", "ci-integration-cleanup.sh"), []byte(stub), 0o600); err != nil {
				t.Fatal(err)
			}
			fixture := filepath.Join(work, "fixture.sh")
			if err := os.WriteFile(fixture, []byte("printf '%s' \"$HAA_KERNEL_LOG\"\nexit \"$HAA_KERNEL_EXIT\"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			body := strings.Replace(kernelTerminalQualificationRun, kernelTerminalQualificationCommand, `bash "$HAA_KERNEL_STUB"`, 1)
			cmd := exec.Command("bash", "-c", body)
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "HELOX_CI_CUDA_ONLY=false", "RUNNER_TEMP="+filepath.Join(work, "runner"), "HAA_KERNEL_STUB="+fixture, "HAA_KERNEL_LOG="+tc.log, "HAA_KERNEL_EXIT="+strconv.Itoa(tc.status))
			output, err := cmd.CombinedOutput()
			status := 0
			if err != nil {
				if exit, ok := err.(*exec.ExitError); ok {
					status = exit.ExitCode()
				} else {
					t.Fatal(err)
				}
			}
			if status != tc.want {
				t.Fatalf("guard status %d want %d: %s", status, tc.want, output)
			}
		})
	}
}
