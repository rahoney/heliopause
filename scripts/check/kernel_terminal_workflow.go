package main

import "strings"

const kernelTerminalQualificationStep = "Qualify kernel direct-exec image and terminal facts"
const kernelTerminalQualificationCommand = `HELOX_KERNEL_TERMINAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxDirectExecKernelTerminalIntegration$'`

// Fixed repository convention only; actual execution and cleanup are separate.
const kernelTerminalQualificationRun = "set -euo pipefail\n" + cudaOnlyExecutionStepGuard + "\n" + `source scripts/ci-integration-cleanup.sh
haa_ci_initialize
haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"
haa_start_policy_helper
umask 077
kernel_terminal_log="$RUNNER_TEMP/haa-kernel-terminal.log"
kernel_terminal_status=0
(
  cd internal/sandbox
  HELOX_KERNEL_TERMINAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxDirectExecKernelTerminalIntegration$'
) >"$kernel_terminal_log" 2>&1 || kernel_terminal_status=$?
kernel_terminal_log_status=0
cat -- "$kernel_terminal_log" || kernel_terminal_log_status=$?
if ((kernel_terminal_status)); then exit "$kernel_terminal_status"; fi
if ((kernel_terminal_log_status)); then exit "$kernel_terminal_log_status"; fi
awk '
  /--- SKIP:/ { bad=1 }
  /^--- PASS: TestLinuxDirectExecKernelTerminalIntegration \(/ { seen++ }
  /kernel_terminal case=/ { cases++ }
  END { exit (bad || seen != 1 || cases != 8) }
' "$kernel_terminal_log"
rm -- "$kernel_terminal_log"
`

func validateKernelTerminalQualification(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", kernelTerminalQualificationStep)
	if err != nil {
		return nil
	} // The structural validator owns parse failures.
	if len(blocks) == 1 && kernelQualificationLines(strings.Join(blocks[0], "\n")) == kernelQualificationLines(kernelTerminalQualificationRun) {
		return nil
	}
	return []string{"kernel image/terminal qualification requires one unconditional protected-client lifecycle, exact parent/eight cases and no-SKIP/primary-error guard"}
}

func kernelQualificationLines(run string) string {
	var active []string
	for _, line := range strings.Split(run, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			active = append(active, line)
		}
	}
	return strings.Join(active, "\n")
}
