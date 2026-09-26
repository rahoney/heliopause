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

func TestInstallInspectKeepsSingleNodeGraphOnLegacyInspectionPath(t *testing.T) {
	graph := fixtureLockedGraph(t)
	ports := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph)}
	source, _ := domain.NewSourceID("fixture")
	reference, _ := domain.NewArtifactReference(source, "safe")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	service, err := application.NewInstallInspectService(&lockedResolver{graph: graph}, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Inspect(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if ports.graphCalls != 0 || !containsCall(ports.calls, "inspect:safe") {
		t.Fatalf("single-node graph calls=%v graphCalls=%d", ports.calls, ports.graphCalls)
	}
}

func TestInstallInspectUsesNeutralGraphCapabilityForEveryDependencyGraph(t *testing.T) {
	for _, sourceName := range []string{"pypi", "pytorch-cpu"} {
		t.Run(sourceName, func(t *testing.T) {
			graph := twoNodeGraphForSource(t, sourceName)
			ports := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph)}
			source, _ := domain.NewSourceID(sourceName)
			reference, _ := domain.NewArtifactReference(source, "first")
			target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
			installContext, _ := domain.NewInstallContext(target)
			request, _ := application.NewInstallRequest(reference, installContext)
			service, err := application.NewInstallInspectService(&lockedResolver{graph: graph}, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Inspect(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if ports.graphCalls != 1 || containsCall(ports.calls, "inspect:first") || containsCall(ports.calls, "inspect:second") || len(result.Set().Inspections()) != 2 {
				t.Fatalf("graph calls=%d calls=%v result=%#v", ports.graphCalls, ports.calls, result)
			}
			if got, want := ports.calls, []string{"acquire:first", "verify:first", "acquire:second", "verify:second", "graph", "evidence:first", "evidence:second"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("calls=%v want=%v", got, want)
			}
		})
	}
}

func TestInstallInspectRecordsBoundedGraphDynamicInstallDiagnostic(t *testing.T) {
	graph := twoNodeGraphForSource(t, "pypi")
	ports := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), dynamicFailure: "M5_PYPI_DYNAMIC_IMPORT_FAILED_MISSING_SHARED_LIBRARY"}
	source, _ := domain.NewSourceID("pypi")
	reference, _ := domain.NewArtifactReference(source, "first")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	service, err := application.NewInstallInspectService(&lockedResolver{graph: graph}, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := result.GraphDynamicInstallDiagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Node().String() != "first" || diagnostics[0].Package() != "first" || diagnostics[0].Reason() != "PYPI_DYNAMIC_IMPORT_FAILED" || diagnostics[0].Phase() != "" || diagnostics[0].FailureClass() != "MISSING_SHARED_LIBRARY" {
		t.Fatalf("dynamic diagnostics = %#v", diagnostics)
	}
}

func TestInstallInspectRecordsBoundedPerNodePolicyAttribution(t *testing.T) {
	graph := twoNodeGraphForSource(t, "pypi")
	ports := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), dynamicFinding: "M3_NETWORK_ATTEMPT"}
	source, _ := domain.NewSourceID("pypi")
	reference, _ := domain.NewArtifactReference(source, "first")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	service, err := application.NewInstallInspectService(&lockedResolver{graph: graph}, ports, ports, ports, ports, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := result.DependencyPolicyDiagnostics()
	if len(diagnostics) != 2 || diagnostics[0].Node().String() != "first" || diagnostics[0].Package() != "first" || diagnostics[0].Decision() != domain.DecisionManualReview || !reflect.DeepEqual(diagnostics[0].Reasons(), []string{"M3_NETWORK_ATTEMPT"}) || !reflect.DeepEqual(diagnostics[0].DynamicFindingCodes(), []string{"M3_NETWORK_ATTEMPT"}) {
		t.Fatalf("first policy diagnostic = %#v", diagnostics)
	}
	if diagnostics[1].Node().String() != "second" || diagnostics[1].Decision() != domain.DecisionAllow || len(diagnostics[1].DynamicFindingCodes()) != 0 {
		t.Fatalf("second policy diagnostic = %#v", diagnostics[1])
	}
}

func TestInstallInspectRejectsGraphReportKeySetMismatch(t *testing.T) {
	graph := twoNodeGraphForSource(t, "pypi")
	ports := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), omitReport: true}
	source, _ := domain.NewSourceID("pypi")
	reference, _ := domain.NewArtifactReference(source, "first")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	service, _ := application.NewInstallInspectService(&lockedResolver{graph: graph}, ports, ports, ports, ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if _, err := service.Inspect(context.Background(), request); err == nil {
		t.Fatal("graph report key set mismatch accepted")
	}
}

func TestInstallInspectRejectsAcquireHandleOutsidePendingRun(t *testing.T) {
	graph := twoNodeGraphForSource(t, "pypi")
	base := &graphInspectionPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph)}
	ports := badHandleGraphPorts{graphInspectionPorts: base}
	source, _ := domain.NewSourceID("pypi")
	reference, _ := domain.NewArtifactReference(source, "first")
	target, _ := domain.NewInstallTarget("/tmp/heliopause-install-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(reference, installContext)
	service, _ := application.NewInstallInspectService(&lockedResolver{graph: graph}, &ports, &ports, &ports, &ports, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if _, err := service.Inspect(context.Background(), request); err == nil {
		t.Fatal("acquired artifact with mismatched Run handle accepted")
	}
	if base.graphCalls != 0 {
		t.Fatalf("graph inspection ran after invalid acquisition handle: %d", base.graphCalls)
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
	return twoNodeGraphForSource(t, "npm")
}

func twoNodeGraphForSource(t *testing.T, sourceName string) domain.LockedDependencyGraph {
	t.Helper()
	source, _ := domain.NewSourceID(sourceName)
	variant, locator, integrity := "tarball", "https://registry.npmjs.org/first/-/first-1.0.0.tgz", "sha512-first"
	if sourceName != "npm" {
		variant, locator, integrity = "wheel", "https://example.test/first-1.0.0-py3-none-any.whl", "sha256:"+strings.Repeat("a", 64)
	}
	firstIdentity, _ := domain.NewResolvedArtifactIdentity(source, "first", "1.0.0", variant)
	secondIdentity, _ := domain.NewResolvedArtifactIdentity(source, "second", "1.0.0", variant)
	firstResolved, _ := domain.NewResolvedArtifact(firstIdentity, locator, integrity)
	secondResolved, _ := domain.NewResolvedArtifact(secondIdentity, strings.Replace(locator, "first", "second", 1), integrity)
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

type graphInspectionPorts struct {
	*multiInspectionPorts
	graphCalls     int
	omitReport     bool
	dynamicFailure string
	dynamicFinding string
}

func (p *graphInspectionPorts) InspectGraph(_ context.Context, graph domain.LockedDependencyGraph, artifacts map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	p.graphCalls++
	p.calls = append(p.calls, "graph")
	reports := make(map[domain.DependencyNodeID]domain.InspectionReport, len(artifacts))
	for _, dependency := range graph.Nodes() {
		artifact, ok := artifacts[dependency.Node()]
		if !ok {
			return nil, errors.New("missing graph artifact")
		}
		checkID, _ := domain.NewCheckID("graph-inspection-" + artifact.Identity().Name())
		status, limitation := domain.ExecutionCompleted, ""
		if p.dynamicFailure != "" && artifact.Identity().Name() == "first" {
			checkID, _ = domain.NewCheckID("pypi-dynamic-import")
			status, limitation = domain.ExecutionIncomplete, p.dynamicFailure
		}
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, status, limitation)
		evidence, _ := domain.NewEvidence(mustEvidenceID("graph-inspection-"+artifact.Identity().Name()), checkID, artifact.Identity(), artifact.Digest(), "fixture", "graph inspection completed")
		var findings []domain.Finding
		if p.dynamicFinding != "" && artifact.Identity().Name() == "first" {
			finding, _ := domain.NewFinding(p.dynamicFinding, []domain.EvidenceID{evidence.ID()})
			findings = []domain.Finding{finding}
		}
		report, _ := domain.NewInspectionReport(check, findings, []domain.Evidence{evidence})
		reports[dependency.Node()] = report
	}
	if p.omitReport {
		delete(reports, graph.Nodes()[0].Node())
	}
	return reports, nil
}

type badHandleGraphPorts struct{ *graphInspectionPorts }

func (p *badHandleGraphPorts) Acquire(ctx context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	artifact, err := p.multiInspectionPorts.Acquire(ctx, runID, resolved)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(artifact.Identity(), artifact.Digest(), "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:"+artifact.Identity().Variant(), artifact.SizeBytes(), resolved.DeclaredIntegrity())
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
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
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+runID.String()+":"+resolved.Identity().Variant(), 1, resolved.DeclaredIntegrity())
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
