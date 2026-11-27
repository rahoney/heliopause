//go:build realcorpus

package githubrelease

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPipxRuntimeSelectedIntakeAndRequiredReport(t *testing.T) {
	filename := os.Getenv("HELOX_PIPX_RUNTIME_ZIP")
	if filename == "" {
		t.Skip("explicit selected immutable ZIP required")
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"complete", "incomplete", "runner-error", "cancel-after-runner"} {
		t.Run(mode, func(t *testing.T) {
			reader, artifact, _ := pipxInputFixture(t, body)
			_, _, observed := pipxReportFixture(t)
			sentinel := errors.New("test runner failure")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			spy := &pipxInspectionSpy{observed: observed}
			switch mode {
			case "incomplete":
				spy.observed.SandboxResult = result(t, domain.SandboxIncomplete, "P4_PIPX_ACCOUNTING_UNAVAILABLE", nil)
			case "runner-error":
				spy.err = sentinel
			case "cancel-after-runner":
				spy.after = cancel
			}
			inspector := &PipxRuntimeInspector{static: reader, runner: spy}
			report, err := inspector.Inspect(ctx, artifact)
			if spy.calls != 1 {
				t.Fatal("selected input did not reach runner once")
			}
			if mode == "runner-error" {
				if !errors.Is(err, sentinel) || len(report.Evidence()) != 0 {
					t.Fatal("runner error hidden", err)
				}
				return
			}
			if mode == "cancel-after-runner" {
				if !errors.Is(err, context.Canceled) || len(report.Evidence()) != 0 {
					t.Fatal("late cancellation ignored", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := domain.ExecutionCompleted
			if mode == "incomplete" {
				want = domain.ExecutionIncomplete
			}
			if report.Execution().Status() != want || !report.Execution().Required() {
				t.Fatal("incorrect required state")
			}
			for _, evidence := range report.Evidence() {
				if evidence.Identity() != artifact.Identity() || evidence.Digest() != artifact.Digest() {
					t.Fatal("original HAA subject lost")
				}
			}
		})
	}
}
