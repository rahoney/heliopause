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
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

func TestInstallInspectResolvesLockedGraphBeforeInspectingEveryEntry(t *testing.T) {
	t.Parallel()

	fake, err := fakeworkflow.New(fakeworkflow.Safe)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := fake.Reference()
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, err := application.NewInstallRequest(reference, installContext)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &lockedResolver{graph: fixtureLockedGraph(t)}
	service, err := application.NewInstallInspectService(
		resolver, fake, fake, fake, fake, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !resolver.called || !result.Set().Valid() || len(result.Set().Inspections()) != 1 || result.OperationID().String() == "" || result.Resolution().RuntimeIdentity() == "" {
		t.Fatalf("resolver called=%v result=%#v", resolver.called, result)
	}
	if result.Decision().Decision() != domain.DecisionAllow || !reflect.DeepEqual(result.Decision().Reasons(), []string{"M4_VERIFIED_SET_COMPLETED"}) {
		t.Fatalf("set decision = %#v", result.Decision())
	}
	if got, want := fake.Calls(), []string{"acquire", "verify", "inspect", "evidence"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
}

func TestInstallInspectFailsClosedWhenLockedResolutionFails(t *testing.T) {
	t.Parallel()

	fake, err := fakeworkflow.New(fakeworkflow.Safe)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := fake.Reference()
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	resolver := &lockedResolver{err: errors.New("locked resolution failed")}
	service, err := application.NewInstallInspectService(
		resolver, fake, fake, fake, fake, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Inspect(context.Background(), request); !errors.Is(err, resolver.err) {
		t.Fatalf("Inspect() error = %v, want resolver error", err)
	}
	if got := fake.Calls(); len(got) != 0 {
		t.Fatalf("entry inspection ran after resolution failure: %v", got)
	}
}

func TestInstallInspectCreatesIndependentCompletedRecordsForEveryLockedNode(t *testing.T) {
	t.Parallel()

	graph := twoNodeGraph(t)
	resolver := &lockedResolver{graph: graph}
	ports := newMultiInspectionPorts(t, graph)
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "first")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	service, err := application.NewInstallInspectService(
		resolver, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision().Decision() != domain.DecisionAllow || len(result.Set().Inspections()) != 2 {
		t.Fatalf("set decision=%#v inspections=%#v", result.Decision(), result.Set().Inspections())
	}
	for _, inspection := range result.Set().Inspections() {
		if inspection.RunID().String() == "" || len(inspection.Evidence()) != 2 || inspection.PolicyDecision().Decision() != domain.DecisionAllow {
			t.Fatalf("incomplete dependency record: %#v", inspection)
		}
	}
	if got, want := ports.calls, []string{"acquire:first", "verify:first", "inspect:first", "evidence:first", "acquire:second", "verify:second", "inspect:second", "evidence:second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls=%v want=%v", got, want)
	}
}

type lockedResolver struct {
	graph  domain.LockedDependencyGraph
	err    error
	called bool
}

func (r *lockedResolver) ResolveDependencies(_ context.Context, _ domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	r.called = true
	if r.err != nil {
		return domain.DependencyResolution{}, r.err
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	return domain.NewDependencyResolution(r.graph, "fixture-runtime", digest)
}

func fixtureLockedGraph(t *testing.T) domain.LockedDependencyGraph {
	t.Helper()
	source, _ := domain.NewSourceID("fixture")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "safe", "1.0.0", "default")
	resolved, _ := domain.NewResolvedArtifact(identity, "fixture-artifact:safe", "sha512-fixture")
	nodeID, _ := domain.NewDependencyNodeID("primary")
	node, _ := domain.NewLockedDependency(nodeID, domain.DependencyPrimary, resolved)
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func twoNodeGraph(t *testing.T) domain.LockedDependencyGraph {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	firstIdentity, _ := domain.NewResolvedArtifactIdentity(source, "first", "1.0.0", "tarball")
	secondIdentity, _ := domain.NewResolvedArtifactIdentity(source, "second", "1.0.0", "tarball")
	firstResolved, _ := domain.NewResolvedArtifact(firstIdentity, "https://registry.npmjs.org/first/-/first-1.0.0.tgz", "sha512-first")
	secondResolved, _ := domain.NewResolvedArtifact(secondIdentity, "https://registry.npmjs.org/second/-/second-1.0.0.tgz", "sha512-second")
	firstID, _ := domain.NewDependencyNodeID("first")
	secondID, _ := domain.NewDependencyNodeID("second")
	first, _ := domain.NewLockedDependency(firstID, domain.DependencyPrimary, firstResolved)
	second, _ := domain.NewLockedDependency(secondID, domain.DependencyTransitive, secondResolved)
	edge, _ := domain.NewDependencyEdge(firstID, secondID)
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{first, second}, []domain.DependencyEdge{edge})
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

type multiInspectionPorts struct {
	artifacts map[domain.ResolvedArtifactIdentity]domain.ResolvedArtifact
	calls     []string
}

func newMultiInspectionPorts(t *testing.T, graph domain.LockedDependencyGraph) *multiInspectionPorts {
	t.Helper()
	artifacts := make(map[domain.ResolvedArtifactIdentity]domain.ResolvedArtifact)
	for _, node := range graph.Nodes() {
		artifacts[node.Artifact().Identity()] = node.Artifact()
	}
	return &multiInspectionPorts{artifacts: artifacts}
}

func (p *multiInspectionPorts) Acquire(_ context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if p.artifacts[resolved.Identity()] != resolved || runID.String() == "" {
		return domain.AcquiredArtifact{}, errors.New("unexpected locked acquisition")
	}
	p.calls = append(p.calls, "acquire:"+resolved.Identity().Name())
	digestCharacter := "a"
	if resolved.Identity().Name() == "second" {
		digestCharacter = "b"
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat(digestCharacter, 64))
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+runID.String()+":tarball", 1, resolved.DeclaredIntegrity())
}

func (p *multiInspectionPorts) Resolve(_ context.Context, _ domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("recursive Install must not re-resolve individual dependencies")
}

func (p *multiInspectionPorts) Verify(_ context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	p.calls = append(p.calls, "verify:"+artifact.Identity().Name())
	checkID, _ := domain.NewCheckID("fixture-integrity")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidence, _ := domain.NewEvidence(mustEvidenceID("integrity-"+artifact.Identity().Name()), checkID, artifact.Identity(), artifact.Digest(), "fixture", "integrity completed")
	return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
}

func (p *multiInspectionPorts) Inspect(_ context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	p.calls = append(p.calls, "inspect:"+artifact.Identity().Name())
	checkID, _ := domain.NewCheckID("fixture-inspection")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidence, _ := domain.NewEvidence(mustEvidenceID("inspection-"+artifact.Identity().Name()), checkID, artifact.Identity(), artifact.Digest(), "fixture", "inspection completed")
	return domain.NewInspectionReport(check, nil, []domain.Evidence{evidence})
}

func (p *multiInspectionPorts) Record(_ context.Context, _ domain.RunID, evidence []domain.Evidence) ([]domain.EvidenceReference, error) {
	if len(evidence) != 2 {
		return nil, errors.New("unexpected evidence batch")
	}
	p.calls = append(p.calls, "evidence:"+evidence[0].Identity().Name())
	references := make([]domain.EvidenceReference, 0, len(evidence))
	for _, item := range evidence {
		reference, err := domain.NewEvidenceReference(item.ID(), "fixture:"+item.ID().String())
		if err != nil {
			return nil, err
		}
		references = append(references, reference)
	}
	return references, nil
}

func mustEvidenceID(value string) domain.EvidenceID {
	id, err := domain.NewEvidenceID(value)
	if err != nil {
		panic(err)
	}
	return id
}
