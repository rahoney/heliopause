package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

type archiveActionPorts struct {
	*multiInspectionPorts
	kind       string
	badSubject bool
}

func newArchiveActionPorts(t *testing.T, graph domain.LockedDependencyGraph) *archiveActionPorts {
	t.Helper()
	return &archiveActionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), kind: domain.EvidenceHostInstallActionNotRequired}
}

func (p *archiveActionPorts) Inspect(_ context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	p.calls = append(p.calls, "inspect:"+artifact.Identity().Name())
	checkID, err := domain.NewCheckID("npm-static-archive")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	check, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	id, err := domain.NewEvidenceID("npm-host-install-action")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	identity := artifact.Identity()
	if p.badSubject {
		identity, err = domain.NewResolvedArtifactIdentity(identity.Source(), "other-subject", identity.Version(), identity.Variant())
		if err != nil {
			return domain.InspectionReport{}, err
		}
	}
	fact, err := domain.NewEvidence(id, checkID, identity, artifact.Digest(), p.kind, "archive installation fact")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewInspectionReport(check, nil, []domain.Evidence{fact})
}

// The two-item batch is the verifier's existing fact plus the explicit archive
// action fact. No generic fixture's Evidence-count expectation is relaxed.
type archiveActionBatchPorts struct {
	*archiveActionPorts
	batches int
}

func (p *archiveActionBatchPorts) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	p.batches++
	if len(acquired) != len(snapshot.Dependencies()) {
		return nil, errors.New("complete project subjects are required")
	}
	reports := map[domain.DependencyNodeID]domain.InspectionReport{}
	for node, artifact := range acquired {
		report, err := p.archiveActionPorts.Inspect(ctx, artifact)
		if err != nil {
			return nil, err
		}
		reports[node] = report
	}
	return reports, nil
}

func TestProjectInspectionRetainsArchiveActionThroughSingleAndBatch(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, kind := range []string{domain.EvidenceHostInstallActionNotRequired, domain.EvidenceHostInstallActionRequired} {
			for _, badSubject := range []bool{false, true} {
				graph := twoNodeGraph(t)
				p := newArchiveActionPorts(t, graph)
				p.kind, p.badSubject = kind, badSubject
				var inspector interface {
					Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error)
				} = p
				bp := &archiveActionBatchPorts{archiveActionPorts: p}
				if batch {
					inspector = bp
				}
				s, err := application.NewProjectInspectService(p, p, inspector, p, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
				if err != nil {
					t.Fatal(err)
				}
				approved, err := s.InspectProject(context.Background(), inspectionProjectSnapshot(t, graph))
				if batch && bp.batches != 1 {
					t.Fatal("batch path was not consumed")
				}
				if badSubject {
					if err == nil || approved.Valid() {
						t.Fatal("wrong subject returned project approval")
					}
					continue
				}
				var inspected domain.InspectedProjectSet
				if kind == domain.EvidenceHostInstallActionRequired {
					var denied *application.ProjectPolicyFailure
					if !errors.As(err, &denied) || denied.Decision().Decision() != domain.DecisionManualReview || denied.Decision().Reasons()[0] != "M4_HOST_LIFECYCLE_UNSUPPORTED" || approved.Valid() {
						t.Fatalf("Host install requirement lost: %v", err)
					}
					inspected = denied.Inspected()
				} else {
					if err != nil || !approved.Valid() {
						t.Fatalf("known script-free set failed: %v", err)
					}
					inspected = approved.Inspected()
				}
				for _, entry := range inspected.Inspections() {
					fact, known := entry.HostInstallActionEvidence()
					if !known || fact.Kind() != kind || fact.Identity() != entry.Artifact().Identity() || fact.Digest() != entry.Artifact().Digest() {
						t.Fatal("exact archive fact was lost")
					}
				}
			}
		}
	}
}
