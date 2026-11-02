package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

type declaredInspectFixture struct {
	*fakeworkflow.Ports
	mode     string
	profile  domain.ContentDigest
	declared int
}

func (f *declaredInspectFixture) ExpectedChecks(_ context.Context, _ domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	calls := f.Calls()
	if len(calls) != 2 || calls[0] != "resolve" || calls[1] != "acquire" {
		return nil, errors.New("declaration was not frozen before execution")
	}
	f.declared++
	names := []string{"fixture-static", "fixture-dynamic"}
	if kind == domain.CheckVerification {
		names = []string{"fixture-integrity"}
	}
	var result []domain.CheckRequirement
	for _, name := range names {
		id, err := domain.NewCheckID(name)
		if err != nil {
			return nil, err
		}
		requirement, err := domain.NewCheckRequirement(id, kind, true, f.profile)
		if err != nil {
			return nil, err
		}
		result = append(result, requirement)
	}
	return result, nil
}

func (f *declaredInspectFixture) Verify(ctx context.Context, a domain.AcquiredArtifact) (domain.VerificationReport, error) {
	report, err := f.Ports.Verify(ctx, a)
	if err != nil {
		return report, err
	}
	actual := report.Execution()
	execution, err := domain.NewCheckExecution(actual.ID(), actual.Kind(), actual.Required(), actual.Capability(), actual.Status(), "", f.profile)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReport(execution, report.Outcome(), report.Evidence())
}

func (f *declaredInspectFixture) Inspect(ctx context.Context, a domain.AcquiredArtifact) (domain.InspectionReport, error) {
	original, err := f.Ports.Inspect(ctx, a)
	if err != nil {
		return original, err
	}
	actual := original.Execution()
	execution, err := domain.NewCheckExecution(actual.ID(), actual.Kind(), true, actual.Capability(), actual.Status(), "", f.profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	first, err := domain.NewInspectionReport(execution, original.Findings(), original.Evidence())
	if err != nil {
		return first, err
	}
	reports := []domain.InspectionReport{first}
	if f.mode == "missing" {
		return first, nil
	}
	profile := f.profile
	if f.mode == "wrong-profile" {
		profile, _ = domain.CheckProfileDigest("foreign-profile-v1", "")
	}
	name := "fixture-dynamic"
	if f.mode == "duplicate" {
		name = "fixture-static"
	}
	id, _ := domain.NewCheckID(name)
	dynamic, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidenceID, _ := domain.NewEvidenceID("dynamic-evidence")
	evidence, err := domain.NewEvidence(evidenceID, id, a.Identity(), a.Digest(), "dynamic-fixture", "Completed independent declared experiment.")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	report, err := domain.NewInspectionReport(dynamic, nil, []domain.Evidence{evidence})
	if err != nil {
		return report, err
	}
	reports = append(reports, report)
	return domain.NewCompositeInspectionReport(reports)
}

func TestInspectFreezesIndependentChecksBeforeExecution(t *testing.T) {
	for _, mode := range []string{"complete", "missing", "duplicate", "wrong-profile"} {
		t.Run(mode, func(t *testing.T) {
			base, err := fakeworkflow.New(fakeworkflow.Safe)
			if err != nil {
				t.Fatal(err)
			}
			profile, _ := domain.CheckProfileDigest("declared-fixture-v1", "")
			fixture := &declaredInspectFixture{Ports: base, mode: mode, profile: profile}
			service, err := application.NewInspectService(fixture, fixture, fixture, fixture, policy.M3{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := fixture.Reference()
			if err != nil {
				t.Fatal(err)
			}
			request, err := application.NewInspectRequest(reference)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Inspect(context.Background(), request)
			if !result.CheckPlan().Valid() || len(result.CheckPlan().Requirements()) != 3 {
				t.Fatal("result lost its pre-execution declaration")
			}
			if fixture.declared != 2 {
				t.Fatal("declarations not obtained independently before both ports")
			}
			if mode == "complete" {
				decision, ok := result.PolicyDecision()
				if err != nil || !ok || decision.Decision() != domain.DecisionAllow {
					t.Fatalf("declared complete set refused: %v", err)
				}
			} else if err == nil || result.Status() != domain.OperationFailed {
				t.Fatal("missing/duplicate/foreign-profile check passed coverage")
			}
		})
	}
}
