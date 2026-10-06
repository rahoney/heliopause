package application

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// ProjectBuildService uses the existing Run/Evidence/Policy lifecycle for one
// guarded build. Source integrity and execution observations remain separate.
type ProjectBuildService struct {
	mutation     ports.ProjectBuildMutation
	verification ports.Verification
	inspection   ports.ProjectBuildInspection
	evidence     ports.Evidence
	policy       Policy
	newOperation func() (domain.OperationID, error)
	newRun       func() (domain.RunID, error)
}

func NewProjectBuildService(mutation ports.ProjectBuildMutation, verification ports.Verification, inspection ports.ProjectBuildInspection, evidence ports.Evidence, policy Policy, newOperation func() (domain.OperationID, error), newRun func() (domain.RunID, error)) (*ProjectBuildService, error) {
	if mutation == nil || verification == nil || inspection == nil || evidence == nil || policy == nil || newOperation == nil || newRun == nil {
		return nil, errors.New("project build requires guarded inputs, verification, inspection, Evidence and Policy")
	}
	return &ProjectBuildService{mutation, verification, inspection, evidence, policy, newOperation, newRun}, nil
}

func (s *ProjectBuildService) Build(ctx context.Context, install domain.InstallContext, selector string) (result domain.OperationResult, published domain.PublishedProjectBuild, resultErr error) {
	if s == nil || ctx == nil || !install.Valid() {
		return result, published, errors.New("project build request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return result, published, err
	}
	guard, err := s.mutation.BeginBuild(ctx, install)
	if err != nil {
		return result, published, err
	}
	if guard == nil {
		return result, published, errors.New("project build guard is unavailable")
	}
	defer func() {
		resultErr = errors.Join(resultErr, guard.Close())
		if resultErr != nil {
			published = domain.PublishedProjectBuild{}
		}
	}()
	operationID, err := s.newOperation()
	if err != nil {
		return result, published, err
	}
	runID, err := s.newRun()
	if err != nil {
		return result, published, err
	}
	inputs, err := guard.FreezeBuildInputs(ctx, runID, selector)
	if err != nil {
		return result, published, err
	}
	if !inputs.Valid() || inputs.RunID() != runID || inputs.Selector() != selector || inputs.Snapshot().Context() != install {
		return result, published, errors.New("project build guard returned substituted inputs")
	}
	reference, err := domain.NewArtifactReference(inputs.Source().Identity().Source(), inputs.Source().Identity().Name())
	if err != nil {
		return result, published, err
	}
	run, err := domain.NewInspectionRun(runID, operationID, reference, inputs.Source().Identity())
	if err != nil {
		return result, published, err
	}
	fail := func(code string, cause error) (domain.OperationResult, domain.PublishedProjectBuild, error) {
		code = strings.Replace(code, "M12_GO_", "M12_"+strings.ToUpper(inputs.Kind())+"_", 1)
		r, e := failRun(run, reference, inputs.Source(), nil, nil, code, "Project build did not complete.", cause)
		return r, domain.PublishedProjectBuild{}, e
	}
	if err := run.Activate(); err != nil {
		return fail("M12_GO_BUILD_RUN_FAILED", err)
	}
	if err := run.BindAcquiredArtifact(inputs.Source()); err != nil {
		return fail("M12_GO_BUILD_INPUT_FAILED", err)
	}
	verification, err := s.verification.Verify(ctx, inputs.Source())
	if err != nil {
		return fail("M12_GO_BUILD_VERIFICATION_FAILED", err)
	}
	report, err := s.inspection.InspectBuild(ctx, inputs)
	if err != nil {
		return fail("M12_GO_BUILD_OBSERVATION_FAILED", err)
	}
	if !reflect.DeepEqual(report.Inputs(), inputs) {
		return fail("M12_GO_BUILD_BINDING_FAILED", errors.New("project build inspection returned substituted inputs"))
	}
	checks := append([]domain.CheckExecution{verification.Execution()}, report.Inspection().Executions()...)
	facts := append(verification.Evidence(), report.Inspection().Evidence()...)
	refs, err := s.evidence.Record(ctx, runID, facts)
	if err != nil {
		r, e := failRun(run, reference, inputs.Source(), checks, nil, "EVIDENCE_RECORD_FAILED", "Build Evidence recording failed.", err)
		return r, published, e
	}
	input, err := domain.NewPolicyInput(runID, inputs.Source(), verification, report.Inspection(), refs)
	if err != nil {
		r, e := failRun(run, reference, inputs.Source(), checks, refs, "POLICY_INPUT_INVALID", "Build Policy input is invalid.", err)
		return r, published, e
	}
	decision, err := s.policy.Evaluate(input)
	if err != nil {
		r, e := failRun(run, reference, inputs.Source(), checks, refs, "POLICY_EVALUATION_FAILED", "Build Policy evaluation failed.", err)
		return r, published, e
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return result, published, err
	}
	result, err = domain.NewInspectOperationResult(domain.OperationResultData{OperationID: operationID, Status: domain.OperationCompleted, Reference: reference, ResolvedIdentity: inputs.Source().Identity(), Digest: inputs.Source().Digest(), RunID: runID, RunOutcome: domain.RunCompleted, Checks: checks, Evidence: refs, PolicyDecision: decision})
	if err != nil {
		return result, published, err
	}
	if decision.Decision() != domain.DecisionAllow {
		return result, published, nil
	}
	approved, err := domain.NewApprovedProjectBuild(report, verification, result)
	if err != nil {
		return result, published, err
	}
	if err := guard.VerifyBuildSource(ctx); err != nil {
		return result, published, err
	}
	published, err = guard.PublishBuild(ctx, approved)
	if err != nil || !published.Valid() {
		return result, domain.PublishedProjectBuild{}, errors.Join(errors.New("project build output publication failed"), err)
	}
	return result, published, nil
}
