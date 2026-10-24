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

type projectPartialFailurePorts struct {
	*projectBatchInspectionPorts
	providerErr, recordErr error
	mode                   string
	records, policies      int
	retained               []domain.Evidence
}

func (p *projectPartialFailurePorts) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	reports, err := p.projectBatchInspectionPorts.InspectProject(ctx, snapshot, acquired)
	if err != nil {
		return nil, err
	}
	if p.mode == "missing" {
		for node := range reports {
			delete(reports, node)
			break
		}
	}
	if p.mode == "foreign" {
		var first domain.InspectionReport
		for _, report := range reports {
			first = report
			break
		}
		for node := range reports {
			reports[node] = first
		}
	}
	return reports, p.providerErr
}
func (p *projectPartialFailurePorts) Record(ctx context.Context, run domain.RunID, facts []domain.Evidence) ([]domain.EvidenceReference, error) {
	p.records++
	p.retained = append(p.retained, facts...)
	if p.recordErr != nil {
		return nil, p.recordErr
	}
	return p.multiInspectionPorts.Record(ctx, run, facts)
}
func (p *projectPartialFailurePorts) Evaluate(input domain.PolicyInput) (domain.PolicyDecision, error) {
	p.policies++
	return (policy.M3{}).Evaluate(input)
}
func TestProjectProviderFailureRecordsExactFactsWithoutPolicyOrPartialApproval(t *testing.T) {
	provider := errors.New("original project failure")
	record := errors.New("controlled evidence store failure")
	for _, mode := range []string{"exact", "missing", "foreign", "store-failure"} {
		t.Run(mode, func(t *testing.T) {
			graph := twoNodeGraph(t)
			base := &projectBatchInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), t: t}
			p := &projectPartialFailurePorts{projectBatchInspectionPorts: base, providerErr: provider, mode: mode}
			if mode == "store-failure" {
				p.recordErr = record
			}
			service, err := application.NewProjectInspectService(p, p, p, p, p, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			approved, err := service.InspectProject(context.Background(), inspectionProjectSnapshot(t, graph))
			if !errors.Is(err, provider) || approved.Valid() || p.policies != 0 {
				t.Fatal("failure cause or no-approval contract lost", err)
			}
			if mode == "missing" || mode == "foreign" {
				if p.records != 0 {
					t.Fatal("invalid facts were recorded")
				}
			} else if p.records != 2 || len(p.retained) != 4 {
				t.Fatal("exact partial evidence lost", p.records)
			}
			if mode == "store-failure" && !errors.Is(err, record) {
				t.Fatal("evidence failure lost original cause", err)
			}
		})
	}
}
