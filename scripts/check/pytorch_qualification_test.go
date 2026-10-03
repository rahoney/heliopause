package main

import (
	"context"
	"encoding/json"
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
		end := strings.Index(run[start:], `haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"`)
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
 printf '%s\t%s\t%s\n' "$HELOX_PYTORCH_PROFILE" "${HELOX_PYTORCH_INSPECTION_PREREQUISITE-}" "${HELOX_PYTORCH_NOT_ATTESTED_MODULE-}"
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
	if lines[0] != "cpu\t\t" {
		return fmt.Errorf("CPU inherited augmentation: %q", lines[0])
	}
	if selected == "" {
		return nil
	}
	fields := strings.Split(lines[1], "\t")
	if len(fields) != 3 || fields[0] != selected || fields[2] != "triton.profiler.viewer" {
		return fmt.Errorf("CUDA scope missing: %q", lines[1])
	}
	var pin map[string]string
	if err := json.Unmarshal([]byte(fields[1]), &pin); err != nil {
		return err
	}
	target := "bbacde6f75665b197016b986164cfdaa33b17515e5e635a63ddb75926aaa71c3"
	if selected == "cu126" {
		target = "94e4f9bd6b9b21aad545e0d441707094e4ff88ab420d62b85fcdc8b6da1e039f"
	}
	expected := map[string]string{"target_sha256": target, "source": "pypi", "project": "numpy", "version": "2.4.6", "filename": "numpy-2.4.6-cp314-cp314-manylinux_2_27_x86_64.manylinux_2_28_x86_64.whl", "sha256": "a2c306dea656c12c68f51f4cea133cbe78ca7435eb28c735eac1d3ebe73be6e8", "reason": "BROADER_MODULE_PROBES"}
	if len(pin) != len(expected) {
		return fmt.Errorf("pin fields differ")
	}
	for key, value := range expected {
		if pin[key] != value {
			return fmt.Errorf("incorrect %s", key)
		}
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
	for name, broken := range map[string]string{
		"missing prerequisite":     strings.ReplaceAll(string(data), "HELOX_PYTORCH_INSPECTION_PREREQUISITE=", "UNUSED_INPUT="),
		"wrong digest":             strings.ReplaceAll(string(data), "a2c306dea656c12c68f51f4cea133cbe78ca7435eb28c735eac1d3ebe73be6e8", strings.Repeat("0", 64)),
		"missing command evidence": strings.ReplaceAll(string(data), "HELOX_PYTORCH_NOT_ATTESTED_MODULE=", "UNUSED_EXPECTATION="),
	} {
		t.Run(name, func(t *testing.T) {
			if checkQualificationInputs(broken, "cu126") == nil {
				t.Fatal("broken delivery accepted")
			}
		})
	}
}
