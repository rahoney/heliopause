package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Execute the actual workflow selection/input block with only its external
// commands replaced. This catches local-only inputs absent from CI delivery.
func qualificationInputs(contents, selected string) ([]string, error) {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return nil, err
	}
	jobs, ok := doc["jobs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("jobs missing")
	}
	job, ok := jobs["gvisor-integration"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("integration missing")
	}
	steps, _ := job["steps"].([]any)
	var body string
	for _, value := range steps {
		step, _ := value.(map[string]any)
		run, _ := step["run"].(string)
		start := strings.Index(run, "selected_cuda_profiles=0")
		if start < 0 {
			continue
		}
		end := strings.Index(run[start:], ordinaryCILifecycleGuard)
		if end < 0 {
			return nil, fmt.Errorf("end missing")
		}
		body = run[start : start+end]
	}
	if body == "" {
		return nil, fmt.Errorf("qualification commands missing")
	}
	for _, p := range []string{"cu126", "cu130", "cu132"} {
		v := "false"
		if p == selected {
			v = "true"
		}
		body = strings.ReplaceAll(body, "${{ github.event_name == 'workflow_dispatch' && inputs.pytorch_"+p+"_qualification }}", v)
	}
	body = strings.ReplaceAll(body, "/usr/libexec/heliopause/helox", "record_client")
	stub := `go() { test "$*" = 'run ./scripts/check qualification-freshness'; }
record_client() {
 test "$HELOX_PYTORCH_FULL_INTEGRATION" = 1
 printf '%s\t%s\t%s\t%s\n' "$HELOX_PYTORCH_PROFILE" "${HELOX_PYTORCH_INSPECTION_PREREQUISITE-}" "${HELOX_PYTORCH_QUALIFICATION_PLAN-}" "${HELOX_PYTORCH_NOT_ATTESTED_MODULE-}"
}
`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "--noprofile", "--norc", "-e", "-u", "-o", "pipefail", "-c", stub+body)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("actual block: %w: %s", err, out)
	}
	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n"), nil
}

func checkQualificationInputs(contents, selected string) error {
	lines, err := qualificationInputs(contents, selected)
	if err != nil {
		return err
	}
	want := 1
	if selected != "" {
		want = 2
	}
	if len(lines) != want {
		return fmt.Errorf("invocations=%d", len(lines))
	}
	if lines[0] != "cpu\t\t\t" {
		return fmt.Errorf("CPU inherited augmentation: %q", lines[0])
	}
	if selected == "" {
		return nil
	}
	fields := strings.Split(lines[1], "\t")
	if len(fields) != 4 || fields[0] != selected || fields[1] != "" || fields[2] != "cuda-bindings-numpy" || fields[3] != "triton.profiler.viewer" {
		return fmt.Errorf("CUDA scope missing: %q", lines[1])
	}
	return nil
}

func TestPyTorchQualificationUsesExplicitBoundedInputs(t *testing.T) {
	data, err := os.ReadFile("../../" + workflowRelativePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"", "cu126", "cu130", "cu132"} {
		t.Run("profile-"+profile, func(t *testing.T) {
			if err := checkQualificationInputs(string(data), profile); err != nil {
				t.Fatal(err)
			}
		})
	}
	multiple := strings.Replace(string(data), "${{ github.event_name == 'workflow_dispatch' && inputs.pytorch_cu130_qualification }}", "true", 1)
	if _, err := qualificationInputs(multiple, "cu126"); err == nil {
		t.Fatal("multiple CUDA profiles accepted")
	}
	for name, broken := range map[string]string{
		"missing plan":             strings.ReplaceAll(string(data), "HELOX_PYTORCH_QUALIFICATION_PLAN=", "UNUSED_INPUT="),
		"unknown plan":             strings.ReplaceAll(string(data), "cuda-bindings-numpy", "unknown-plan"),
		"missing command evidence": strings.ReplaceAll(string(data), "HELOX_PYTORCH_NOT_ATTESTED_MODULE=", "UNUSED_EXPECTATION="),
	} {
		t.Run(name, func(t *testing.T) {
			if checkQualificationInputs(broken, "cu126") == nil {
				t.Fatal("broken delivery accepted")
			}
		})
	}
}

func TestCUDAOnlyQualificationPreservesOrdinaryAndCommonScopes(t *testing.T) {
	data, err := os.ReadFile("../../" + workflowRelativePath)
	if err != nil {
		t.Fatal(err)
	}
	original := string(data)
	if findings := validateCUDAQualificationScope(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	for name, candidate := range map[string]string{
		"ordinary-routed-away":    strings.Replace(original, cudaOnlyCIExpression, "true", 1),
		"non-manual-routing":      strings.Replace(original, "github.event_name == 'workflow_dispatch' && (inputs.", "(inputs.", 1),
		"step-override":           strings.Replace(original, "      - name: Qualify production Node execution preparation\n", "      - name: Qualify production Node execution preparation\n        env:\n          HELOX_CI_CUDA_ONLY: 'true'\n", 1),
		"missing-ordinary-scope":  strings.Replace(original, ordinaryCILifecycleGuard, "if false; then", 1),
		"reversed-ordinary-scope": strings.Replace(original, ordinaryCILifecycleGuard, `if [ "$HELOX_CI_CUDA_ONLY" = true ]; then`, 1),
		"missing-execution-scope": strings.Replace(original, `if [ "$HELOX_CI_CUDA_ONLY" = true ]; then`, "if false; then", 1),
		"missing-full-client":     strings.Replace(original, `go test -c -o "$RUNNER_TEMP/helox-bootstrap-test" ./internal/bootstrap`, "echo omitted", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(validateCUDAQualificationScope(candidate)) == 0 {
				t.Fatal("invalid qualification dispatch accepted")
			}
		})
	}
	blocks, _ := activeJobRunLines(original, "gvisor-integration", currentExecutionPreparationStep)
	if len(blocks) != 1 {
		t.Fatal("execution step missing")
	}
	// Execute the actual early routing before any helper lifecycle can run.
	body := kernelQualificationLines(strings.Join(blocks[0], "\n"))
	body, _, ok := strings.Cut(body, "\nsource scripts/ci-integration-cleanup.sh")
	if !ok {
		t.Fatal("cleanup boundary missing")
	}
	for _, mode := range []string{"false", "true"} {
		command := exec.Command("bash", "--noprofile", "--norc", "-c", body+"\necho ordinary_continuation")
		command.Env = append(os.Environ(), "HELOX_CI_CUDA_ONLY="+mode)
		output, err := command.CombinedOutput()
		if err != nil || strings.Contains(string(output), "ordinary_continuation") != (mode == "false") {
			t.Fatal("incorrect ordinary/manual route", mode, string(output), err)
		}
	}
}
