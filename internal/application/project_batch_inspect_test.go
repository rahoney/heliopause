package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

type projectBatchInspectionPorts struct {
	*multiInspectionPorts
	t                                    *testing.T
	acquired, verified, batches, singles int
}

func (p *projectBatchInspectionPorts) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	p.acquired++
	return p.multiInspectionPorts.Acquire(ctx, run, resolved)
}
func (p *projectBatchInspectionPorts) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	p.verified++
	return p.multiInspectionPorts.Verify(ctx, artifact)
}
func (p *projectBatchInspectionPorts) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	p.singles++
	return domain.InspectionReport{}, errors.New("single-entry inspection loses complete-project dependency context")
}
func (p *projectBatchInspectionPorts) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	p.t.Helper()
	p.batches++
	if p.acquired != len(snapshot.Dependencies()) || p.verified != len(snapshot.Dependencies()) || len(acquired) != len(snapshot.Dependencies()) {
		p.t.Fatal("project inspection ran before acquiring/verifying complete set")
	}
	reports := map[domain.DependencyNodeID]domain.InspectionReport{}
	for _, resolved := range snapshot.Dependencies() {
		node, _ := domain.ProjectDependencyNodeID(resolved.Identity())
		a, ok := acquired[node]
		if !ok || a.Identity() != resolved.Identity() {
			p.t.Fatal("project lost exact acquired subject")
		}
		report, err := p.multiInspectionPorts.Inspect(ctx, a)
		if err != nil {
			return nil, err
		}
		reports[node] = report
	}
	return reports, nil
}

func TestProjectInspectionUsesCompleteProjectCapabilityBeforeSingleEntries(t *testing.T) {
	graph := twoNodeGraphForSource(t, "pypi")
	snapshot := inspectionProjectSnapshot(t, graph)
	p := &projectBatchInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), t: t}
	service, err := application.NewProjectInspectService(p, p, p, p, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	set, err := service.InspectProject(context.Background(), snapshot)
	if err != nil || !set.Valid() || p.batches != 1 || p.singles != 0 || len(set.Inspected().Inspections()) != 2 {
		t.Fatal("project capability was bypassed", err, p.batches, p.singles)
	}
}
