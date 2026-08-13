// Package fakeworkflow provides deterministic, I/O-free M1 Port implementations for tests.
package fakeworkflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// Scenario selects one synthetic M1 outcome.
type Scenario string

const (
	Safe         Scenario = "safe"
	Review       Scenario = "review"
	Blocked      Scenario = "blocked"
	AcquireError Scenario = "acquire-error"
	ResolveError Scenario = "resolve-error"
)

var (
	ErrResolve = errors.New("synthetic resolve failure")
	ErrAcquire = errors.New("synthetic acquire failure")
)

// Ports implements all four M1 outbound Port contracts without external I/O.
type Ports struct {
	scenario Scenario
	mu       sync.Mutex
	calls    []string
}

var (
	_ ports.Artifact     = (*Ports)(nil)
	_ ports.Verification = (*Ports)(nil)
	_ ports.Inspection   = (*Ports)(nil)
	_ ports.Evidence     = (*Ports)(nil)
)

// New creates deterministic synthetic Ports for a supported fixture scenario.
func New(scenario Scenario) (*Ports, error) {
	switch scenario {
	case Safe, Review, Blocked, AcquireError, ResolveError:
		return &Ports{scenario: scenario}, nil
	default:
		return nil, fmt.Errorf("unknown fake workflow scenario %q", scenario)
	}
}

// Reference returns the scenario's deterministic Artifact Reference.
func (p *Ports) Reference() (domain.ArtifactReference, error) {
	source, err := domain.NewSourceID("fixture")
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	return domain.NewArtifactReference(source, string(p.scenario))
}

// Calls returns an owned copy of the observed Port call order.
func (p *Ports) Calls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.calls...)
}

func (p *Ports) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	p.record("resolve")
	if err := validContext(ctx); err != nil {
		return domain.ResolvedArtifact{}, err
	}
	want, err := p.Reference()
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	if reference != want {
		return domain.ResolvedArtifact{}, errors.New("fake resolve reference mismatch")
	}
	if p.scenario == ResolveError {
		return domain.ResolvedArtifact{}, ErrResolve
	}
	identity, err := domain.NewResolvedArtifactIdentity(reference.Source(), string(p.scenario), "1.0.0", "default")
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	return domain.NewResolvedArtifact(identity, "fixture-artifact:"+string(p.scenario), "sha512-fixture")
}

func (p *Ports) Acquire(ctx context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	p.record("acquire")
	if err := validContext(ctx); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if runID.String() == "" {
		return domain.AcquiredArtifact{}, errors.New("fake acquire requires Run ID")
	}
	want, err := p.expectedIdentity()
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if resolved.Identity() != want {
		return domain.AcquiredArtifact{}, errors.New("fake acquire identity mismatch")
	}
	if p.scenario == AcquireError {
		return domain.AcquiredArtifact{}, ErrAcquire
	}
	digest, err := domain.NewSHA256Digest(strings.Repeat(digestCharacter(p.scenario), 64))
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	return domain.NewAcquiredArtifact(resolved.Identity(), digest, "fixture-content:"+string(p.scenario), 128)
}

func (p *Ports) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	p.record("verify")
	if err := validContext(ctx); err != nil {
		return domain.VerificationReport{}, err
	}
	if err := p.validateArtifact(artifact); err != nil {
		return domain.VerificationReport{}, err
	}
	checkID, err := domain.NewCheckID("fixture-integrity")
	if err != nil {
		return domain.VerificationReport{}, err
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.VerificationReport{}, err
	}
	evidence, err := newEvidence("fixture-integrity-evidence", checkID, artifact, "integrity", "Observed digest matches the deterministic fixture subject.")
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReport(execution, domain.VerificationVerified, []domain.Evidence{evidence})
}

func (p *Ports) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	p.record("inspect")
	if err := validContext(ctx); err != nil {
		return domain.InspectionReport{}, err
	}
	if err := p.validateArtifact(artifact); err != nil {
		return domain.InspectionReport{}, err
	}
	checkID, err := domain.NewCheckID("fixture-static")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if p.scenario == Review {
		execution, executionErr := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilityUnsupported, domain.ExecutionNotExecuted, "M1_CAPABILITY_UNSUPPORTED")
		if executionErr != nil {
			return domain.InspectionReport{}, executionErr
		}
		return domain.NewInspectionReport(execution, nil, nil)
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidence, err := newEvidence("fixture-static-evidence", checkID, artifact, "static-inspection", "Synthetic static inspection completed for the fixture subject.")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	var findings []domain.Finding
	if p.scenario == Blocked {
		finding, findingErr := domain.NewFinding("M1_BLOCK_FINDING", []domain.EvidenceID{evidence.ID()})
		if findingErr != nil {
			return domain.InspectionReport{}, findingErr
		}
		findings = []domain.Finding{finding}
	}
	return domain.NewInspectionReport(execution, findings, []domain.Evidence{evidence})
}

func (p *Ports) Record(ctx context.Context, runID domain.RunID, evidence []domain.Evidence) ([]domain.EvidenceReference, error) {
	p.record("evidence")
	if err := validContext(ctx); err != nil {
		return nil, err
	}
	if runID.String() == "" {
		return nil, errors.New("fake evidence record requires a Run ID")
	}
	if len(evidence) == 0 {
		return nil, errors.New("fake evidence record requires a non-empty batch")
	}
	references := make([]domain.EvidenceReference, len(evidence))
	seen := make(map[domain.EvidenceID]bool, len(evidence))
	for index, item := range evidence {
		if seen[item.ID()] {
			return nil, errors.New("fake evidence record rejects duplicate Evidence IDs")
		}
		seen[item.ID()] = true
		if err := p.validateEvidenceSubject(item); err != nil {
			return nil, err
		}
		reference, err := domain.NewEvidenceReference(item.ID(), "fixture-evidence:"+item.ID().String())
		if err != nil {
			return nil, err
		}
		references[index] = reference
	}
	return references, nil
}

func (p *Ports) validateEvidenceSubject(evidence domain.Evidence) error {
	want, err := p.expectedIdentity()
	if err != nil {
		return err
	}
	if evidence.Identity() != want || evidence.Digest().String() != strings.Repeat(digestCharacter(p.scenario), 64) {
		return errors.New("fake evidence subject mismatch")
	}
	return nil
}

func (p *Ports) expectedIdentity() (domain.ResolvedArtifactIdentity, error) {
	reference, err := p.Reference()
	if err != nil {
		return domain.ResolvedArtifactIdentity{}, err
	}
	return domain.NewResolvedArtifactIdentity(reference.Source(), string(p.scenario), "1.0.0", "default")
}

func (p *Ports) validateArtifact(artifact domain.AcquiredArtifact) error {
	want, err := p.expectedIdentity()
	if err != nil {
		return err
	}
	if artifact.Identity() != want || artifact.Digest().String() != strings.Repeat(digestCharacter(p.scenario), 64) {
		return errors.New("fake workflow artifact subject mismatch")
	}
	return nil
}

func (p *Ports) record(call string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, call)
}

func validContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

func digestCharacter(scenario Scenario) string {
	switch scenario {
	case Safe:
		return "a"
	case Review:
		return "b"
	case Blocked:
		return "c"
	default:
		return "d"
	}
}

func newEvidence(idValue string, checkID domain.CheckID, artifact domain.AcquiredArtifact, kind, summary string) (domain.Evidence, error) {
	id, err := domain.NewEvidenceID(idValue)
	if err != nil {
		return domain.Evidence{}, err
	}
	return domain.NewEvidence(id, checkID, artifact.Identity(), artifact.Digest(), kind, summary)
}
