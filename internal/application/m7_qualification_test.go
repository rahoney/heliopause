package application_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
)

type m7QualificationFixture struct {
	name     string
	source   string
	variant  string
	locator  string
	content  []byte
	declared string
}

func TestM7QualificationBindsExactBytesAcrossEcosystems(t *testing.T) {
	t.Parallel()
	for _, fixture := range m7QualificationFixtures() {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			ports := newM7QualificationPorts(t, fixture)
			source, err := domain.NewSourceID(fixture.source)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := domain.NewArtifactReference(source, fixture.locator)
			if err != nil {
				t.Fatal(err)
			}
			target, err := domain.NewInstallTarget("/tmp/heliopause-m7-" + fixture.name)
			if err != nil {
				t.Fatal(err)
			}
			installContext, err := domain.NewInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			request, err := application.NewInstallRequest(reference, installContext)
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewInstallInspectService(
				ports, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID,
			)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Inspect(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Decision().Decision() != domain.DecisionAllow || len(result.Set().Inspections()) != 1 {
				t.Fatalf("inspection decision=%#v set=%#v", result.Decision(), result.Set())
			}
			inspection := result.Set().Inspections()[0]
			wantDigest := sha256.Sum256(fixture.content)
			if inspection.Artifact().Identity().Source().String() != fixture.source ||
				inspection.Artifact().Digest().String() != fmt.Sprintf("%x", wantDigest) ||
				inspection.Artifact().ContentHandle() != "fixture-content:"+fixture.name ||
				inspection.Artifact().SizeBytes() != uint64(len(fixture.content)) {
				t.Fatalf("artifact binding=%#v", inspection.Artifact())
			}
			if got := ports.calls; !reflect.DeepEqual(got, []string{"acquire", "verify", "inspect", "evidence"}) {
				t.Fatalf("calls=%v", got)
			}
		})
	}
}

func TestM7QualificationNonAllowNeverInvokesPromotion(t *testing.T) {
	for _, fixture := range m7QualificationFixtures() {
		for _, decision := range []domain.Decision{domain.DecisionManualReview, domain.DecisionBlock} {
			fixture, decision := fixture, decision
			t.Run(fixture.name+"/"+string(decision), func(t *testing.T) {
				ports := newM7QualificationPorts(t, fixture)
				source, _ := domain.NewSourceID(fixture.source)
				reference, _ := domain.NewArtifactReference(source, fixture.locator)
				target, _ := domain.NewInstallTarget("/tmp/heliopause-m7-nonallow-" + fixture.name)
				installContext, _ := domain.NewInstallContext(target)
				request, _ := application.NewInstallRequest(reference, installContext)
				inspection, err := application.NewInstallInspectService(
					ports, ports, ports, ports, ports, policy.M1{}, fixedSetPolicy{decision: decision}, domain.NewOperationID, domain.NewRunID,
				)
				if err != nil {
					t.Fatal(err)
				}
				calls := []string{}
				service, err := application.NewInstallService(
					inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls},
				)
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := service.Install(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				if outcome.Inspection().Decision().Decision() != decision || len(calls) != 0 || outcome.Promoted().Target().String() != "" {
					t.Fatalf("decision=%s calls=%v promoted=%#v", outcome.Inspection().Decision().Decision(), calls, outcome.Promoted())
				}
			})
		}
	}
}

func TestM7QualificationAcquireFailureHasNoPolicyOrPromotion(t *testing.T) {
	fixture := m7QualificationFixtures()[0]
	ports := newM7QualificationPorts(t, fixture)
	ports.acquireErr = errors.New("synthetic acquisition failure")
	source, _ := domain.NewSourceID(fixture.source)
	reference, _ := domain.NewArtifactReference(source, fixture.locator)
	target, _ := domain.NewInstallTarget("/tmp/heliopause-m7-acquire-failure")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	service, err := application.NewInstallInspectService(
		ports, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(context.Background(), request)
	if !errors.Is(err, ports.acquireErr) || result.Decision().Decision() != "" || result.Set().Valid() {
		t.Fatalf("result=%#v error=%v", result, err)
	}
}

func m7QualificationFixtures() []m7QualificationFixture {
	return []m7QualificationFixture{
		{name: "npm", source: "npm", variant: "tarball", locator: "fixture@1.0.0", content: []byte("npm synthetic tarball bytes"), declared: "sha256:npm"},
		{name: "pypi", source: "pypi", variant: "wheel", locator: "fixture@1.0.0", content: []byte("pypi synthetic wheel bytes"), declared: "sha256:pypi"},
		{name: "github", source: "github-release", variant: "zip", locator: "owner/repo@v1.0.0#fixture.zip", content: []byte("GitHub synthetic archive bytes"), declared: "sha256:github"},
	}
}

type m7QualificationPorts struct {
	fixture    m7QualificationFixture
	resolved   domain.ResolvedArtifact
	graph      domain.LockedDependencyGraph
	acquireErr error
	calls      []string
}

var (
	_ ports.Artifact           = (*m7QualificationPorts)(nil)
	_ ports.DependencyResolver = (*m7QualificationPorts)(nil)
	_ ports.Verification       = (*m7QualificationPorts)(nil)
	_ ports.Inspection         = (*m7QualificationPorts)(nil)
	_ ports.Evidence           = (*m7QualificationPorts)(nil)
)

func newM7QualificationPorts(t *testing.T, fixture m7QualificationFixture) *m7QualificationPorts {
	t.Helper()
	source, err := domain.NewSourceID(fixture.source)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(source, fixture.name, "1.0.0", fixture.variant)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := domain.NewResolvedArtifact(identity, "fixture:"+fixture.name, fixture.declared)
	if err != nil {
		t.Fatal(err)
	}
	node, err := domain.NewDependencyNodeID(fixture.name)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := domain.NewLockedDependency(node, domain.DependencyPrimary, resolved)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{locked}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &m7QualificationPorts{fixture: fixture, resolved: resolved, graph: graph}
}

func (p *m7QualificationPorts) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return p.resolved, nil
}

func (p *m7QualificationPorts) Acquire(_ context.Context, _ domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	p.calls = append(p.calls, "acquire")
	if p.acquireErr != nil {
		return domain.AcquiredArtifact{}, p.acquireErr
	}
	if resolved != p.resolved {
		return domain.AcquiredArtifact{}, errors.New("qualification resolved identity mismatch")
	}
	sum := sha256.Sum256(p.fixture.content)
	digest, err := domain.NewSHA256Digest(fmt.Sprintf("%x", sum))
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "fixture-content:"+p.fixture.name, uint64(len(p.fixture.content)), resolved.DeclaredIntegrity())
}

func (p *m7QualificationPorts) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	digest, err := domain.NewSHA256Digest("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(p.graph, "m7-synthetic-runtime", digest)
}

func (p *m7QualificationPorts) Verify(_ context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	p.calls = append(p.calls, "verify")
	if artifact.Identity() != p.resolved.Identity() {
		return domain.VerificationReport{}, errors.New("qualification verification identity mismatch")
	}
	checkID, _ := domain.NewCheckID("m7-integrity")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidence, err := domain.NewEvidence(domainID("m7-integrity-"+p.fixture.name), checkID, artifact.Identity(), artifact.Digest(), "m7-integrity", "synthetic digest verified")
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
}

func (p *m7QualificationPorts) Inspect(_ context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	p.calls = append(p.calls, "inspect")
	checkID, _ := domain.NewCheckID("m7-inspection")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidence, err := domain.NewEvidence(domainID("m7-inspection-"+p.fixture.name), checkID, artifact.Identity(), artifact.Digest(), "m7-inspection", "synthetic inspection completed")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewInspectionReport(check, nil, []domain.Evidence{evidence})
}

func (p *m7QualificationPorts) Record(_ context.Context, _ domain.RunID, evidence []domain.Evidence) ([]domain.EvidenceReference, error) {
	p.calls = append(p.calls, "evidence")
	references := make([]domain.EvidenceReference, len(evidence))
	for index, item := range evidence {
		reference, err := domain.NewEvidenceReference(item.ID(), "m7-evidence:"+item.ID().String())
		if err != nil {
			return nil, err
		}
		references[index] = reference
	}
	return references, nil
}

func domainID(value string) domain.EvidenceID {
	id, err := domain.NewEvidenceID(value)
	if err != nil {
		panic(err)
	}
	return id
}
