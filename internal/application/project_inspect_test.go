package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

func TestProjectInspectCoversFrozenSnapshotWithoutResolvingPrimary(t *testing.T) {
	graph := twoNodeGraph(t)
	p := newMultiInspectionPorts(t, graph)
	snapshot := inspectionProjectSnapshot(t, graph)
	s, err := application.NewProjectInspectService(p, p, p, p, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := s.InspectProject(context.Background(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !approved.Valid() || approved.Inspected().Snapshot().GraphDigest() != snapshot.GraphDigest() || len(approved.Inspected().Inspections()) != 2 {
		t.Fatal("project coverage or snapshot binding lost")
	}
	inspections := approved.Inspected().Inspections()
	if inspections[0].RunID() == inspections[1].RunID() {
		t.Fatal("project entries shared a run")
	}
	want := []string{"acquire:first", "verify:first", "inspect:first", "evidence:first", "acquire:second", "verify:second", "inspect:second", "evidence:second"}
	if !reflect.DeepEqual(p.calls, want) {
		t.Fatalf("calls=%v", p.calls)
	}
}

type projectFailingPorts struct {
	*multiInspectionPorts
	fail string
}

func (p *projectFailingPorts) Verify(ctx context.Context, a domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if a.Identity().Name() == "second" && p.fail == "verify" {
		return domain.VerificationReport{}, errors.New("fixture independent verification failed")
	}
	if a.Identity().Name() == "second" && p.fail == "mismatch" {
		r, err := p.multiInspectionPorts.Verify(ctx, a)
		if err != nil {
			return r, err
		}
		return domain.NewVerificationReport(r.Execution(), domain.VerificationMismatch, r.Evidence())
	}
	return p.multiInspectionPorts.Verify(ctx, a)
}
func (p *projectFailingPorts) Record(ctx context.Context, run domain.RunID, e []domain.Evidence) ([]domain.EvidenceReference, error) {
	if e[0].Identity().Name() == "second" && p.fail == "evidence" {
		return nil, errors.New("fixture evidence failed")
	}
	return p.multiInspectionPorts.Record(ctx, run, e)
}

func TestProjectInspectNeverReturnsPartialApproval(t *testing.T) {
	for _, fail := range []string{"verify", "mismatch", "evidence"} {
		t.Run(fail, func(t *testing.T) {
			graph := twoNodeGraph(t)
			p := &projectFailingPorts{newMultiInspectionPorts(t, graph), fail}
			s, err := application.NewProjectInspectService(p, p, p, p, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			approved, err := s.InspectProject(context.Background(), inspectionProjectSnapshot(t, graph))
			if err == nil || approved.Valid() {
				t.Fatal("failed project returned approval")
			}
			if fail == "mismatch" {
				var denied *application.ProjectPolicyFailure
				if !errors.As(err, &denied) || denied.Decision().Decision() != domain.DecisionBlock || len(denied.Inspected().Inspections()) != 2 {
					t.Fatal("trusted project failure facts lost")
				}
			}
		})
	}
}

func inspectionProjectSnapshot(t *testing.T, graph domain.LockedDependencyGraph) domain.ProjectDependencySnapshot {
	t.Helper()
	target, err := domain.NewInstallTarget(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := domain.NewSHA256Digest(strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	control, err := domain.NewProjectControlDigest("fixture.control", hash)
	if err != nil {
		t.Fatal(err)
	}
	var entries []domain.ResolvedArtifact
	for _, node := range graph.Nodes() {
		entries = append(entries, node.Artifact())
	}
	snapshot, err := domain.NewProjectDependencySnapshot(ctx, entries[0].Identity().Source(), []domain.ProjectControlDigest{control}, entries, hash)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
