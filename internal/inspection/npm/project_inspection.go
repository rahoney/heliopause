package npm

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type npmProjectInventoryBudget interface {
	Add(artifactnpm.ArchiveInventory) error
	Complete() error
}

// Concrete client adapters keep their typed native context/layout. This shared
// workflow owns static-all ordering, distinct observed sessions and coverage.
type npmProjectInspectionPlan struct {
	client      projectInspectionClient
	static      *Inspector
	subjects    []domain.AcquiredArtifact
	instances   map[domain.AcquiredArtifact][]string
	beginBudget func() (npmProjectInventoryBudget, error)
	layout      projectInstanceFactLayout
	prepare     func(context.Context, []artifactnpm.ArchiveInventory) error
	execute     func(context.Context, domain.AcquiredArtifact, string) (domain.SandboxResult, error)
}

type npmProjectInspectionState struct {
	subject              domain.AcquiredArtifact
	instances            []string
	leaves               []domain.InspectionReport
	attempted, completed int
	failed               bool
}

func inspectNPMProjectPlan(ctx context.Context, plan npmProjectInspectionPlan) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if ctx == nil || plan.static == nil || plan.prepare == nil || plan.execute == nil || plan.layout == nil || plan.beginBudget == nil {
		return nil, errors.New("complete project inspection lacks its actual consumers")
	}
	prefix, err := projectInspectionPrefix(plan.client)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	if len(plan.subjects) == 0 {
		return map[domain.DependencyNodeID]domain.InspectionReport{}, nil
	}
	if len(plan.instances) != len(plan.subjects) {
		return nil, errors.New(prefix + " inspection has incomplete instance/budget coverage")
	}
	states := make([]npmProjectInspectionState, 0, len(plan.subjects))
	for _, subject := range plan.subjects {
		instances := plan.instances[subject]
		if len(instances) == 0 {
			return nil, errors.New(prefix + " inspection has no instance for an archive")
		}
		states = append(states, npmProjectInspectionState{subject: subject, instances: instances})
	}
	finish := func(cause error) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
		reports := make(map[domain.DependencyNodeID]domain.InspectionReport, len(states))
		for _, state := range states {
			leaves := append([]domain.InspectionReport{}, state.leaves...)
			coverage, err := npmProjectCoverageReport(state.subject, len(state.instances), state.attempted, state.completed, state.failed, plan.client)
			if err != nil {
				return nil, errors.Join(cause, err)
			}
			leaves = append(leaves, coverage)
			report, err := domain.NewCompositeInspectionReport(leaves)
			if err != nil {
				return nil, errors.Join(cause, err)
			}
			node, err := domain.ProjectDependencyNodeID(state.subject.Identity())
			if err != nil {
				return nil, errors.Join(cause, err)
			}
			reports[node] = report
		}
		return reports, cause
	}
	budget, err := plan.beginBudget()
	if err != nil {
		return finish(err)
	}
	inventories := make([]artifactnpm.ArchiveInventory, 0, len(states))
	ready := true
	for index := range states {
		state := &states[index]
		inventory, report, err := plan.static.InspectArchive(ctx, state.subject)
		if len(report.Executions()) != 0 {
			state.leaves = append(state.leaves, report)
		}
		if err != nil {
			return finish(err)
		}
		if report.Execution().Status() != domain.ExecutionCompleted || len(report.Findings()) != 0 || !inventory.Matches(state.subject) {
			ready = false
			continue
		}
		if err := budget.Add(inventory); err != nil {
			return finish(err)
		}
		inventories = append(inventories, inventory)
	}
	if !ready {
		return finish(nil)
	}
	if err := budget.Complete(); err != nil {
		return finish(err)
	}
	if err := plan.prepare(ctx, inventories); err != nil {
		return finish(err)
	}
	if err := observeNPMProjectInstances(ctx, states, plan.client, plan.layout, plan.execute); err != nil {
		return finish(err)
	}
	return finish(errors.Join(ctx.Err(), context.Cause(ctx)))
}

func observeNPMProjectInstances(ctx context.Context, states []npmProjectInspectionState, client projectInspectionClient, layout projectInstanceFactLayout, execute func(context.Context, domain.AcquiredArtifact, string) (domain.SandboxResult, error)) error {
	prefix, err := projectInspectionPrefix(client)
	if err != nil {
		return err
	}
	sessions := map[domain.SandboxSessionID]bool{}
	for index := range states {
		state := &states[index]
		for _, instance := range state.instances {
			if err := ctx.Err(); err != nil {
				return errors.Join(err, context.Cause(ctx))
			}
			state.attempted++
			result, runErr := execute(ctx, state.subject, instance)
			runErr = errors.Join(runErr, ctx.Err(), context.Cause(ctx))
			facts, factErr := npmProjectInstanceFacts(state.subject, instance, layout, result, client)
			if len(facts.Executions()) != 0 {
				state.leaves = append(state.leaves, facts)
			}
			session := result.SessionID()
			invalid := session.String() == "" || sessions[session] || result.Status() != domain.SandboxCompleted || !completeDynamicEnvironments(result)
			sessions[session] = true
			for _, fact := range result.Observations() {
				if fact.Category() == domain.ObservationResource {
					invalid = true
				}
			}
			if runErr != nil || factErr != nil || invalid {
				state.failed = true
				return errors.Join(runErr, factErr, errors.New(prefix+" actual instance coverage is incomplete"))
			}
			state.completed++
		}
	}
	return nil
}
