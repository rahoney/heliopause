package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

type terraformTerminalRunner struct {
	err       error
	arguments []string
}

func (r *terraformTerminalRunner) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output command")
}
func (r *terraformTerminalRunner) RunDiscard(_ context.Context, _ string, args ...string) error {
	r.arguments = append([]string(nil), args...)
	return r.err
}

func TestTerraformHelpRequiresExactConsumedTerminalAndFixedArguments(t *testing.T) {
	exit := func(code int) error {
		t.Helper()
		e := exec.Command("/bin/sh", "-c", fmt.Sprintf("exit %d", code)).Run()
		if e == nil {
			t.Fatal("fixture did not exit")
		}
		return e
	}
	one := exit(1)
	for _, tc := range []struct {
		name      string
		err       error
		cancelled bool
		want      string
	}{
		{"zero", nil, false, "terraform-help-exit-0"},
		{"exact-consumed-one", &observedDirectExecExit{one}, false, "terraform-help-exit-1"},
		{"unobserved-one", one, false, ""},
		{"string-one", errors.New("exit status 1"), false, ""},
		{"other-target-exit", &observedDirectExecExit{exit(2)}, false, ""},
		{"docker-launch-failure", &observedDirectExecExit{exit(125)}, false, ""},
		{"ambiguous-admission", errors.Join(&observedDirectExecExit{one}, errObserverAuthorityAmbiguous), false, ""},
		{"cancelled-zero", nil, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &terraformTerminalRunner{err: tc.err}
			backend := &TerraformProviderBackend{elf: &GitHubELFBackend{runner: runner}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelled {
				cancel()
			}
			subject, e := backend.probeTerminal(ctx, "0123456789abcdef")
			if subject != tc.want || (e == nil) != (tc.want != "") {
				t.Fatalf("terminal=%s error=%v", subject, e)
			}
			joined := strings.Join(runner.arguments, " ")
			if !strings.HasSuffix(joined, "/work/artifact -help") || strings.Contains(joined, "/bin/sh") {
				t.Fatal("fixed probe changed")
			}
		})
	}
}

func TestTerraformFaultRetainsOnlyAcceptedSuspiciousCounts(t *testing.T) {
	facts := retainedELFProbeFacts(map[string]uint64{"process-exec-expected": 35, "process-exec-unexpected": 1, "network-attempt": 2, "unknown": 1})
	if len(facts) != 2 || facts[0].Subject() != "network-attempt" || facts[0].Count() != 2 || facts[1].Subject() != "process-exec-unexpected" {
		t.Fatalf("partial facts=%v", facts)
	}
	if len(retainedELFProbeFacts(map[string]uint64{"network-attempt": 10001})) != 0 {
		t.Fatal("out-of-bounds diagnostic became an observation")
	}
}
