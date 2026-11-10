//go:build linux && realcorpus

package bootstrap

import (
	"os"
	"testing"
)

func TestLinuxPythonPipxRunExecutionWorkflowIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYTHON_PIPX_RUN_EXECUTION_WORKFLOW_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user current pipx workflow qualification")
	}
	runPythonNativeExecutionWorkflowCases(t, true)
}

func TestLinuxPipxRunExecutionCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_RUN_EXECUTION_CLI_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user pipx run CLI qualification")
	}
	runPythonNativeExecutionCLICases(t, true)
}
