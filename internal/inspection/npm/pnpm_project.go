package npm

import (
	"context"
	"errors"
	"sync"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMInspectionPreparer and PNPMInstanceExecutor are consumer-owned, typed
// capabilities. Inspection does not depend on a container runtime package.
type PNPMInspectionPreparer interface {
	PrepareInspection(context.Context, artifactnpm.PNPMInspectionInput, []artifactnpm.ArchiveInventory) (artifactnpm.PNPMInspectionLayout, error)
}
type PNPMInstanceExecutor interface {
	ExecutePNPMInstance(context.Context, artifactnpm.PNPMInspectionLayout, domain.AcquiredArtifact, string) (domain.SandboxResult, error)
}

// PNPMProjectInspector retains frozen complete-project context beside the
// source-neutral Inspection interface. It cannot silently inspect one primary.
type PNPMProjectInspector struct {
	frozen    artifactnpm.PNPMInspectionContext
	static    *Inspector
	prepare   PNPMInspectionPreparer
	execute   PNPMInstanceExecutor
	handoffMu sync.Mutex
	active    bool
	prepared  artifactnpm.PNPMInspectionLayout
}

func NewPNPMProjectInspector(frozen artifactnpm.PNPMInspectionContext, static *Inspector, prepare PNPMInspectionPreparer, execute PNPMInstanceExecutor) (*PNPMProjectInspector, error) {
	if !frozen.Valid() || static == nil || prepare == nil || execute == nil {
		return nil, errors.New("pnpm project inspection requires frozen context and all consumers")
	}
	return &PNPMProjectInspector{frozen: frozen, static: static, prepare: prepare, execute: execute}, nil
}
func (i *PNPMProjectInspector) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return domain.InspectionReport{}, errors.New("pnpm inspection requires the complete frozen project")
}

type pnpmInspectionState struct {
	subject              domain.AcquiredArtifact
	instances            []artifactnpm.PNPMLockedSnapshot
	leaves               []domain.InspectionReport
	attempted, completed int
	failed               bool
}

func (i *PNPMProjectInspector) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if i == nil || ctx == nil || i.static == nil || i.prepare == nil || i.execute == nil {
		return nil, errors.New("pnpm project inspection is unavailable")
	}
	i.handoffMu.Lock()
	if i.active {
		i.handoffMu.Unlock()
		return nil, errors.New("pnpm project inspector is already executing")
	}
	i.active = true
	i.prepared = artifactnpm.PNPMInspectionLayout{}
	i.handoffMu.Unlock()
	defer func() { i.handoffMu.Lock(); i.active = false; i.handoffMu.Unlock() }()
	input, err := i.frozen.BindAcquired(snapshot, acquired)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	if len(input.Artifacts()) == 0 {
		return map[domain.DependencyNodeID]domain.InspectionReport{}, nil
	}
	states := make([]pnpmInspectionState, 0, len(acquired))
	for _, subject := range input.Artifacts() {
		instances, err := input.InstancesFor(subject)
		if err != nil {
			return nil, err
		}
		states = append(states, pnpmInspectionState{subject: subject, instances: instances})
	}
	finish := func(cause error) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
		reports := make(map[domain.DependencyNodeID]domain.InspectionReport, len(states))
		for _, state := range states {
			leaves := append([]domain.InspectionReport{}, state.leaves...)
			coverage, err := pnpmCoverageReport(state.subject, len(state.instances), state.attempted, state.completed, state.failed)
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
	budget, err := input.BeginWorkspace()
	if err != nil {
		return finish(err)
	}
	inventories := make([]artifactnpm.ArchiveInventory, 0, len(states))
	ready := true
	for index := range states {
		state := &states[index]
		inventory, report, err := i.static.InspectArchive(ctx, state.subject)
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
	layout, err := i.prepare.PrepareInspection(ctx, input, inventories)
	if err != nil {
		return finish(err)
	}
	if !layout.MatchesInput(input) {
		return finish(errors.New("pnpm prepared layout runtime or binding is invalid"))
	}
	for _, state := range states {
		for _, instance := range state.instances {
			if _, err := layout.Target(state.subject, instance.Key); err != nil {
				return finish(err)
			}
		}
	}
	if err := i.observeInstances(ctx, layout, states); err != nil {
		return finish(err)
	}
	reports, err := finish(nil)
	if err != nil || ctx.Err() != nil {
		return reports, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	i.handoffMu.Lock()
	i.prepared = layout
	i.handoffMu.Unlock()
	return reports, nil
}

// PreparedLayoutFor exposes only this request's completely observed immutable
// layout. Consumers still require actual recorded Evidence, Policy and bytes.
func (i *PNPMProjectInspector) PreparedLayoutFor(snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.PNPMInspectionLayout, error) {
	if i == nil {
		return artifactnpm.PNPMInspectionLayout{}, errors.New("pnpm completed layout is unavailable")
	}
	input, err := i.frozen.BindAcquired(snapshot, acquired)
	if err != nil {
		return artifactnpm.PNPMInspectionLayout{}, err
	}
	i.handoffMu.Lock()
	defer i.handoffMu.Unlock()
	if i.active || !i.prepared.MatchesInput(input) {
		return artifactnpm.PNPMInspectionLayout{}, errors.New("pnpm completed layout does not match the exact request")
	}
	return i.prepared, nil
}

func (i *PNPMProjectInspector) observeInstances(ctx context.Context, layout artifactnpm.PNPMInspectionLayout, states []pnpmInspectionState) error {
	sessions := map[domain.SandboxSessionID]bool{}
	for index := range states {
		state := &states[index]
		for _, instance := range state.instances {
			if err := ctx.Err(); err != nil {
				return errors.Join(err, context.Cause(ctx))
			}
			state.attempted++
			result, runErr := i.execute.ExecutePNPMInstance(ctx, layout, state.subject, instance.Key)
			runErr = errors.Join(runErr, ctx.Err(), context.Cause(ctx))
			// Accepted partial facts remain evidence even when the same call failed.
			facts, factErr := pnpmInstanceFacts(state.subject, instance.Key, layout, result)
			if len(facts.Executions()) != 0 {
				state.leaves = append(state.leaves, facts)
			}
			session := result.SessionID()
			invalid := session.String() == "" || sessions[session] || result.Status() != domain.SandboxCompleted
			sessions[session] = true
			for _, fact := range result.Observations() {
				if fact.Category() == domain.ObservationResource {
					invalid = true
				}
			}
			if runErr != nil || factErr != nil || invalid {
				state.failed = true
				return errors.Join(runErr, factErr, errors.New("pnpm actual instance coverage is incomplete"))
			}
			state.completed++
		}
	}
	return nil
}
