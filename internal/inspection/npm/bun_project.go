package npm

import (
	"context"
	"errors"
	"sync"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// BunInspectionPreparer and BunInstanceExecutor are consumer-owned, typed
// capabilities. Inspection does not depend on a container runtime package.
type BunInspectionPreparer interface {
	PrepareInspection(context.Context, artifactnpm.BunInspectionInput, []artifactnpm.ArchiveInventory) (artifactnpm.BunInspectionLayout, error)
}
type BunInstanceExecutor interface {
	ExecuteBunRuntimeInstance(context.Context, artifactnpm.BunInspectionLayout, domain.AcquiredArtifact, string) (domain.SandboxResult, error)
}

// BunProjectInspector retains frozen complete-project context beside the
// source-neutral Inspection interface. It cannot silently inspect one primary.
type BunProjectInspector struct {
	frozen    artifactnpm.BunInspectionContext
	static    *Inspector
	prepare   BunInspectionPreparer
	execute   BunInstanceExecutor
	handoffMu sync.Mutex
	active    bool
	prepared  artifactnpm.BunInspectionLayout
}

func NewBunProjectInspector(frozen artifactnpm.BunInspectionContext, static *Inspector, prepare BunInspectionPreparer, execute BunInstanceExecutor) (*BunProjectInspector, error) {
	if !frozen.Valid() || static == nil || prepare == nil || execute == nil {
		return nil, errors.New("bun project inspection requires frozen context and all consumers")
	}
	return &BunProjectInspector{frozen: frozen, static: static, prepare: prepare, execute: execute}, nil
}
func (i *BunProjectInspector) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return domain.InspectionReport{}, errors.New("bun inspection requires the complete frozen project")
}

func (i *BunProjectInspector) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	return i.inspectInput(ctx, func() (artifactnpm.BunInspectionInput, error) {
		return i.frozen.BindAcquired(snapshot, acquired)
	})
}

func (i *BunProjectInspector) inspectInput(ctx context.Context, bind func() (artifactnpm.BunInspectionInput, error)) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if i == nil || ctx == nil || i.static == nil || i.prepare == nil || i.execute == nil {
		return nil, errors.New("bun project inspection is unavailable")
	}
	i.handoffMu.Lock()
	if i.active {
		i.handoffMu.Unlock()
		return nil, errors.New("bun project inspector is already executing")
	}
	i.active = true
	i.prepared = artifactnpm.BunInspectionLayout{}
	i.handoffMu.Unlock()
	defer func() { i.handoffMu.Lock(); i.active = false; i.handoffMu.Unlock() }()
	input, err := bind()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	if len(input.Artifacts()) == 0 {
		return map[domain.DependencyNodeID]domain.InspectionReport{}, nil
	}
	instances := map[domain.AcquiredArtifact][]string{}
	for _, subject := range input.Artifacts() {
		selected, err := input.InstancesFor(subject)
		if err != nil {
			return nil, err
		}
		for _, instance := range selected {
			instances[subject] = append(instances[subject], instance.Key)
		}
	}
	var layout artifactnpm.BunInspectionLayout
	reports, err := inspectNPMProjectPlan(ctx, npmProjectInspectionPlan{
		client: projectInspectionBun, static: i.static, subjects: input.Artifacts(), instances: instances, layout: &layout,
		beginBudget: func() (npmProjectInventoryBudget, error) { return input.BeginWorkspace() },
		prepare: func(ctx context.Context, inventories []artifactnpm.ArchiveInventory) error {
			var err error
			layout, err = i.prepare.PrepareInspection(ctx, input, inventories)
			if err != nil {
				return err
			}
			if !layout.MatchesInput(input) {
				return errors.New("bun prepared layout runtime or binding is invalid")
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
			return i.execute.ExecuteBunRuntimeInstance(ctx, layout, subject, instance)
		},
	})
	if err != nil || ctx.Err() != nil || !layout.MatchesInput(input) {
		return reports, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	i.handoffMu.Lock()
	i.prepared = layout
	i.handoffMu.Unlock()
	return reports, nil
}

// PreparedLayoutFor exposes only this request's completely observed immutable
// layout. Consumers still require actual recorded Evidence, Policy and bytes.
func (i *BunProjectInspector) PreparedLayoutFor(snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.BunInspectionLayout, error) {
	if i == nil {
		return artifactnpm.BunInspectionLayout{}, errors.New("bun completed layout is unavailable")
	}
	input, err := i.frozen.BindAcquired(snapshot, acquired)
	if err != nil {
		return artifactnpm.BunInspectionLayout{}, err
	}
	return i.preparedLayoutFor(input)
}

func (i *BunProjectInspector) preparedLayoutFor(input artifactnpm.BunInspectionInput) (artifactnpm.BunInspectionLayout, error) {
	i.handoffMu.Lock()
	defer i.handoffMu.Unlock()
	if i.active || !i.prepared.MatchesInput(input) {
		return artifactnpm.BunInspectionLayout{}, errors.New("bun completed layout does not match the exact request")
	}
	return i.prepared, nil
}
