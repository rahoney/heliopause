//go:build linux

package bootstrap

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

func TestProtectedCompatibilityHealthUsesPublicCLI(t *testing.T) {
	for _, scenario := range []fakeworkflow.Scenario{fakeworkflow.Safe, fakeworkflow.Review, fakeworkflow.Blocked, fakeworkflow.AcquireError} {
		t.Run(string(scenario), func(t *testing.T) {
			fake, err := fakeworkflow.New(scenario)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := fake.Reference()
			if err != nil {
				t.Fatal(err)
			}
			fixture, err := application.NewInspectRequest(reference)
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewInspectService(fake, fake, fake, fake, policy.M1{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			inspector := &compatibilityInspector{t: t, service: service, fixture: fixture}
			run := func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
				if stdout != io.Discard || stderr != io.Discard {
					t.Fatal("internal health must not expose nominal output")
				}
				root, err := cli.New(stdout, stderr)
				if err != nil {
					return err
				}
				root.SetArgs(args)
				if _, err := cli.PreparePresentation(root, args); err != nil {
					return err
				}
				if err := cli.AddNPMInspect(root, inspector); err != nil {
					return err
				}
				return root.ExecuteContext(ctx)
			}
			err = checkProtectedCompatibilityHealth(context.Background(), true, func(context.Context) cli.DoctorReport {
				return cli.DoctorReport{Healthy: true}
			}, run)
			if inspector.calls != 1 {
				t.Fatalf("public npm inspect calls=%d, want1; error=%v", inspector.calls, err)
			}
			if (err == nil) != (scenario == fakeworkflow.Safe) {
				t.Fatalf("health error=%v for scenario=%s", err, scenario)
			}
			if scenario == fakeworkflow.AcquireError && !errors.Is(err, fakeworkflow.ErrAcquire) {
				t.Fatalf("lost nominal failure: %v", err)
			}
		})
	}
}

type compatibilityInspector struct {
	t       *testing.T
	service cli.Inspector
	fixture application.InspectRequest
	calls   int
}

func (i *compatibilityInspector) Inspect(ctx context.Context, request application.InspectRequest) (domain.OperationResult, error) {
	i.calls++
	reference, err := artifactnpm.ParseReference("is-number@7.0.0")
	if err != nil {
		i.t.Fatal(err)
	}
	want, err := application.NewInspectRequest(reference)
	if err != nil || request != want {
		i.t.Fatalf("health dispatched another npm subject: %v", err)
	}
	return i.service.Inspect(ctx, i.fixture)
}

func TestProtectedCompatibilityHealthDiagnosisAndRollback(t *testing.T) {
	for _, test := range []struct {
		name    string
		healthy bool
		full    bool
	}{
		{name: "unhealthy candidate", full: true},
		{name: "unhealthy rollback"},
		{name: "healthy rollback", healthy: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkProtectedCompatibilityHealth(context.Background(), test.full, func(context.Context) cli.DoctorReport {
				return cli.DoctorReport{Healthy: test.healthy}
			}, func(context.Context, []string, io.Writer, io.Writer) error {
				t.Fatal("nominal probe must not run")
				return nil
			})
			if test.healthy && err != nil || !test.healthy && !errors.Is(err, releaseactivation.ErrHealth) {
				t.Fatalf("health error=%v, healthy=%v", err, test.healthy)
			}
		})
	}
}
