package npm

import (
	"context"
	"errors"
	"sync"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type NPMProjectInspectionPreparer interface {
	PrepareNPMProjectInspection(context.Context, artifactnpm.NPMProjectInspectionInput, []artifactnpm.ArchiveInventory) (artifactnpm.NPMProjectInspectionLayout, error)
}
type NPMProjectInstanceExecutor interface {
	ExecuteNPMProjectInstance(context.Context, artifactnpm.NPMProjectInspectionLayout, domain.AcquiredArtifact, string) (domain.SandboxResult, error)
}

// NPMProjectInspector retains the exact whole native selection throughout
// static-all, materialization and independently observed logical instances.
type NPMProjectInspector struct {
	expectedInstances map[string][]string
	frozen            artifactnpm.NPMProjectInspectionContext
	static            *Inspector
	prepare           NPMProjectInspectionPreparer
	execute           NPMProjectInstanceExecutor
	mu                sync.Mutex
	active            bool
	prepared          artifactnpm.NPMProjectInspectionLayout
}

func NewNPMProjectInspector(frozen artifactnpm.NPMProjectInspectionContext, static *Inspector, prepare NPMProjectInspectionPreparer, execute NPMProjectInstanceExecutor) (*NPMProjectInspector, error) {
	if !frozen.Valid() || static == nil || prepare == nil || execute == nil {
		return nil, errors.New("npm project inspection requires exact context and actual consumers")
	}
	expectedInstances := make(map[string][]string)
	for _, pkg := range frozen.Lock().Packages() {
		key := pkg.Name + "\x00" + pkg.Version
		expectedInstances[key] = append(expectedInstances[key], pkg.Location)
	}
	return &NPMProjectInspector{expectedInstances: expectedInstances, frozen: frozen, static: static, prepare: prepare, execute: execute}, nil
}

func (i *NPMProjectInspector) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return domain.InspectionReport{}, errors.New("npm project inspection requires complete native context")
}

func (i *NPMProjectInspector) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if i == nil || ctx == nil {
		return nil, errors.New("npm project inspection is unavailable")
	}
	i.mu.Lock()
	if i.active {
		i.mu.Unlock()
		return nil, errors.New("npm project inspection is already executing")
	}
	i.active = true
	i.prepared = artifactnpm.NPMProjectInspectionLayout{}
	i.mu.Unlock()
	defer func() {
		i.mu.Lock()
		i.active = false
		i.mu.Unlock()
	}()
	input, err := i.frozen.BindAcquired(snapshot, acquired)
	if err != nil {
		return nil, err
	}
	instances := map[domain.AcquiredArtifact][]string{}
	for _, subject := range input.Artifacts() {
		selected, err := input.InstancesFor(subject)
		if err != nil {
			return nil, err
		}
		for _, instance := range selected {
			instances[subject] = append(instances[subject], instance.Location)
		}
	}
	var layout artifactnpm.NPMProjectInspectionLayout
	reports, err := inspectNPMProjectPlan(ctx, npmProjectInspectionPlan{
		client: projectInspectionNPM, static: i.static, subjects: input.Artifacts(), instances: instances, layout: &layout,
		beginBudget: func() (npmProjectInventoryBudget, error) { return input.BeginWorkspace() },
		prepare: func(ctx context.Context, inventories []artifactnpm.ArchiveInventory) error {
			var err error
			layout, err = i.prepare.PrepareNPMProjectInspection(ctx, input, inventories)
			if err != nil {
				return err
			}
			if !layout.MatchesInput(input) {
				return errors.New("npm project prepared layout differs from current context")
			}
			for subject, selected := range instances {
				for _, instance := range selected {
					if _, err := layout.Target(subject, instance); err != nil {
						return err
					}
				}
			}
			return nil
		},
		execute: func(ctx context.Context, subject domain.AcquiredArtifact, instance string) (domain.SandboxResult, error) {
			return i.execute.ExecuteNPMProjectInstance(ctx, layout, subject, instance)
		},
	})
	if err != nil || ctx.Err() != nil || !layout.MatchesInput(input) {
		return reports, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	i.mu.Lock()
	i.prepared = layout
	i.mu.Unlock()
	return reports, nil
}

func (i *NPMProjectInspector) PreparedLayoutFor(snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.NPMProjectInspectionLayout, error) {
	if i == nil {
		return artifactnpm.NPMProjectInspectionLayout{}, errors.New("npm completed layout is unavailable")
	}
	input, err := i.frozen.BindAcquired(snapshot, acquired)
	if err != nil {
		return artifactnpm.NPMProjectInspectionLayout{}, err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.active || !i.prepared.MatchesInput(input) {
		return artifactnpm.NPMProjectInspectionLayout{}, errors.New("npm completed layout differs from current exact input")
	}
	return i.prepared, nil
}

func (i *NPMProjectInspector) ExpectedChecks(ctx context.Context, a domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if i == nil || i.static == nil {
		return nil, errors.New("project declaration is unavailable")
	}
	return npmProjectExpectedChecks(ctx, a, kind, i.static, i.expectedInstances[a.Identity().Name()+"\x00"+a.Identity().Version()], projectInspectionNPM)
}
