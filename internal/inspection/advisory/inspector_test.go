package advisory

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func advisoryArtifact(t *testing.T, source, name, version string) domain.AcquiredArtifact {
	t.Helper()
	s, err := domain.NewSourceID(source)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(s, name, version, "test")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithIntegrity(identity, digest, "intake:advisory:test", 1, "sha256:"+digest.String(), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func calculatorBuildInfo() *debug.BuildInfo {
	return &debug.BuildInfo{Deps: []*debug.Module{{Path: "github.com/pandatix/go-cvss", Version: "v0.6.4", Sum: "h1:9w2RCO/Q4UTiJyEgpCHRiVc6CfrsFEnkoX+OtATqKio="}}}
}

type retainedInspector struct {
	calls   []string
	failure error
}

func (p *retainedInspector) Inspect(_ context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	p.calls = append(p.calls, "single")
	return retainedReport(artifact), p.failure
}

func retainedReport(artifact domain.AcquiredArtifact) domain.InspectionReport {
	id, _ := domain.NewCheckID("original-inspection")
	profile, _ := domain.CheckProfileDigest("test-original", "original")
	execution, _ := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	factID, _ := domain.NewEvidenceID("original-fact")
	fact, _ := domain.NewEvidence(factID, id, artifact.Identity(), artifact.Digest(), "original-fact", "original inspection retained", profile)
	report, _ := domain.NewInspectionReport(execution, nil, []domain.Evidence{fact})
	return report
}

func (p *retainedInspector) ExpectedChecks(context.Context, domain.AcquiredArtifact, domain.CheckKind) ([]domain.CheckRequirement, error) {
	id, _ := domain.NewCheckID("original-inspection")
	profile, _ := domain.CheckProfileDigest("test-original", "original")
	requirement, err := domain.NewCheckRequirement(id, domain.CheckInspection, true, profile)
	return []domain.CheckRequirement{requirement}, err
}

func TestArtifactMappingUsesExactExistingSourceAndVersionGrammar(t *testing.T) {
	for _, tc := range []struct{ source, name, version, ecosystem string }{
		{"npm", "@fixture/tool", "1.2.3", "npm"}, {"pypi", "jinja2", "2.10", "PyPI"}, {"go-proxy", "example.com/module", "v1.2.3", "Go"}, {"crates-io", "serde_json", "1.0.0", "crates.io"},
		{"pytorch-cu126", "torch", "2.9.0+cu126", ""}, {"pypi", "jinja2", "2.10+local", ""}, {"pypi", "Jinja2", "2.10", ""}, {"npm", "fixture", "latest", ""}, {"go-proxy", "example.com/module", "1.2.3", ""}, {"cargo", "serde", "1.0.0", ""}, {"github-release", "jinja2", "2.10", ""}, {"terraform-provider", "jinja2", "2.10", ""},
	} {
		t.Run(tc.source+tc.name+tc.version, func(t *testing.T) {
			query, ok := packageForArtifact(advisoryArtifact(t, tc.source, tc.name, tc.version))
			if ok != (tc.ecosystem != "") || query.Ecosystem != tc.ecosystem {
				t.Fatal("incorrect advisory namespace projection", query, ok)
			}
		})
	}
}

func TestInspectorBindsCalculatorProfileToActualLinkedModule(t *testing.T) {
	inner := &retainedInspector{}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("not used") })}
	first, err := newInspector(inner, client, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	updated := calculatorBuildInfo()
	updated.Deps[0].Version = "v0.6.5"
	second, err := newInspector(inner, client, updated)
	if err != nil || first.profile == second.profile {
		t.Fatal("calculator update reused old profile", err)
	}
	for _, info := range []*debug.BuildInfo{nil, {}, {Deps: []*debug.Module{{Path: "github.com/pandatix/go-cvss", Version: "v0.6.4"}}}, {Deps: []*debug.Module{{Path: "github.com/pandatix/go-cvss", Version: "v0.6.4", Sum: calculatorBuildInfo().Deps[0].Sum, Replace: &debug.Module{Path: "local"}}}}} {
		if _, err := newInspector(inner, client, info); err == nil {
			t.Fatal("unverified calculator identity accepted")
		}
	}
}

func TestInspectorPreservesOriginalPlanAndSeparatesCompleteEmptyFromFailure(t *testing.T) {
	artifact := advisoryArtifact(t, "npm", "fixture", "1.0.0")
	inner := &retainedInspector{}
	cause := errors.New("provider unavailable")
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return jsonResponse(`{"results":[{}]}`), nil
		}
		return nil, cause
	})}
	inspector, err := newInspector(inner, client, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	checks, err := inspector.ExpectedChecks(context.Background(), artifact, domain.CheckInspection)
	if err != nil || len(checks) != 2 || checks[0].ID().String() != "original-inspection" || !checks[1].Required() || checks[1].ProfileDigest() != inspector.profile {
		t.Fatal("original plan lost", err)
	}
	report, err := inspector.Inspect(context.Background(), artifact)
	if err != nil || len(report.Executions()) != 2 || report.Executions()[1].Status() != domain.ExecutionCompleted || len(report.Findings()) != 0 || len(report.Evidence()) != 2 {
		t.Fatal("complete empty response was not recorded", err)
	}
	report, err = inspector.Inspect(context.Background(), artifact)
	if !errors.Is(err, cause) || len(report.Executions()) != 2 || report.Executions()[1].Status() != domain.ExecutionIncomplete || len(report.Findings()) != 0 || len(report.Evidence()) != 1 {
		t.Fatal("failed query became empty completion or original error/facts lost", err)
	}
	unsupported := advisoryArtifact(t, "github-release", "fixture", "1.0.0")
	report, err = inspector.Inspect(context.Background(), unsupported)
	if err != nil || calls != 2 || report.Executions()[1].Capability() != domain.CapabilityUnsupported {
		t.Fatal("unsupported subject was queried or silently omitted", err)
	}
	inner.failure = cause
	report, err = inspector.Inspect(context.Background(), artifact)
	if !errors.Is(err, cause) || calls != 2 || len(report.Executions()) != 1 {
		t.Fatal("original provider failure triggered more queries or lost its report", err)
	}
}

func TestInspectorNormalizesAffectedScoresWithoutTrustingHumanSeverity(t *testing.T) {
	artifact := advisoryArtifact(t, "npm", "fixture", "1.0.0")
	inspector, err := newInspector(&retainedInspector{}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("unused") })}, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	var record osvRecord
	if err := json.Unmarshal([]byte(recordFixture), &record); err != nil {
		t.Fatal(err)
	}
	facts := []responseFact{{requestSHA256: strings.Repeat("a", 64), responseSHA256: strings.Repeat("b", 64), fetchedAt: record.Modified, responseBytes: int64(len(recordFixture))}}
	query := osvPackage{"npm", "fixture", "1.0.0"}
	for _, tc := range []struct {
		severity []osvSeverity
		want     string
	}{
		{nil, "M13_ADVISORY_REVIEW"},
		{[]osvSeverity{{"CVSS_V3", "invalid"}}, "M13_ADVISORY_REVIEW"},
		{[]osvSeverity{{"CVSS_V3", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}}, "M13_ADVISORY_HIGH"},
	} {
		record.Severity = tc.severity
		report, err := inspector.report(artifact, query, true, []osvRecord{record}, facts, nil)
		if err != nil || report.Execution().Status() != domain.ExecutionCompleted || len(report.Findings()) != 1 || report.Findings()[0].Code() != tc.want {
			t.Fatal("score/unknown distinction lost", report.Findings(), err)
		}
	}
	query.Version = "1.2.3"
	report, err := inspector.report(artifact, query, true, []osvRecord{record}, facts, nil)
	if err != nil || len(report.Findings()) != 0 {
		t.Fatal("fixed boundary was marked affected", err)
	}
}

type graphRetainedInspector struct{ retainedInspector }

func (p *graphRetainedInspector) InspectGraph(_ context.Context, graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	p.calls = append(p.calls, "graph-static-all-then-closure")
	reports := map[domain.DependencyNodeID]domain.InspectionReport{}
	for _, node := range graph.Nodes() {
		reports[node.Node()] = retainedReport(acquired[node.Node()])
	}
	return reports, p.failure
}

func advisoryGraph(t *testing.T) (domain.LockedDependencyGraph, map[domain.DependencyNodeID]domain.AcquiredArtifact) {
	t.Helper()
	var nodes []domain.LockedDependency
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for index, name := range []string{"root", "fixture"} {
		artifact := advisoryArtifact(t, "npm", name, "1.0.0")
		integrity, _ := artifact.DeclaredIntegrity()
		resolved, err := domain.NewResolvedArtifact(artifact.Identity(), "https://registry.npmjs.org/"+name+"/-/"+name+"-1.0.0.tgz", integrity)
		if err != nil {
			t.Fatal(err)
		}
		id, err := domain.NewDependencyNodeID(name)
		if err != nil {
			t.Fatal(err)
		}
		role := domain.DependencyTransitive
		if index == 0 {
			role = domain.DependencyPrimary
		}
		node, err := domain.NewLockedDependency(id, role, resolved)
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, node)
		acquired[id] = artifact
	}
	edge, err := domain.NewDependencyEdge(nodes[0].Node(), nodes[1].Node())
	if err != nil {
		t.Fatal(err)
	}
	graph, err := domain.NewLockedDependencyGraph(nodes, []domain.DependencyEdge{edge})
	if err != nil {
		t.Fatal(err)
	}
	return graph, acquired
}

func TestInspectorRetainsGraphEntryAndQueriesTransitiveNodesTogether(t *testing.T) {
	graph, acquired := advisoryGraph(t)
	inner := &graphRetainedInspector{}
	posts, gets := 0, 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if len(inner.calls) != 1 || inner.calls[0] != "graph-static-all-then-closure" {
			t.Fatal("advisory replaced original graph phase")
		}
		if request.Method == http.MethodPost {
			posts++
			var message struct {
				Queries []batchQuery `json:"queries"`
			}
			if json.NewDecoder(request.Body).Decode(&message) != nil || len(message.Queries) != 2 {
				t.Fatal("whole frozen graph was not queried once")
			}
			rows := make([]json.RawMessage, len(message.Queries))
			for n, query := range message.Queries {
				rows[n] = json.RawMessage(`{}`)
				if query.Package.Name == "fixture" {
					rows[n] = json.RawMessage(`{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]}`)
				}
			}
			body, err := json.Marshal(struct {
				Results []json.RawMessage `json:"results"`
			}{rows})
			if err != nil {
				t.Fatal(err)
			}
			return jsonResponse(string(body)), nil
		}
		gets++
		return jsonResponse(recordFixture), nil
	})}
	inspector, err := newInspector(inner, client, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	reports, err := inspector.InspectGraph(context.Background(), graph, acquired)
	if err != nil || len(reports) != 2 || posts != 1 || gets != 1 {
		t.Fatal("whole graph inspection failed", err, posts, gets)
	}
	for _, node := range graph.Nodes() {
		report := reports[node.Node()]
		if len(report.Executions()) != 2 || report.Executions()[0].ID().String() != "original-inspection" {
			t.Fatal("original graph report lost")
		}
		want := 0
		if node.Artifact().Identity().Name() == "fixture" {
			want = 1
		}
		if len(report.Findings()) != want {
			t.Fatal("transitive finding attached to wrong node", node.Node())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspector.InspectGraph(ctx, graph, acquired); !errors.Is(err, context.Canceled) || len(inner.calls) != 1 || posts != 1 {
		t.Fatal("cancelled graph reached inspection or lost its cause", err)
	}
	// Acquired membership is checked before any existing dynamic provider runs.
	delete(acquired, graph.Primary())
	if _, err := inspector.InspectGraph(context.Background(), graph, acquired); err == nil || len(inner.calls) != 1 || posts != 1 {
		t.Fatal("incomplete acquired set reached provider or lookup")
	}
}

type selectionRetainedInspector struct{ retainedInspector }

func (p *selectionRetainedInspector) InspectProject(_ context.Context, _ domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	p.calls = append(p.calls, "project")
	return selectionReports(acquired), p.failure
}
func (p *selectionRetainedInspector) InspectProjectSelection(_ context.Context, update domain.ProjectDependencyUpdate, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if len(update.SelectedControls()) != 1 || string(update.SelectedControls()[0].Body()) != "frozen control bytes" {
		return nil, errors.New("frozen controls lost")
	}
	p.calls = append(p.calls, "frozen-selection")
	return selectionReports(acquired), p.failure
}
func (p *selectionRetainedInspector) InspectExecutionSelection(_ context.Context, selection domain.ExecutionDependencySelection, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
	if len(selection.RootGraphs()) != 2 {
		return nil, errors.New("supplemental root lost")
	}
	p.calls = append(p.calls, "execution-selection")
	return selectionReports(acquired), p.failure
}
func selectionReports(acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) map[domain.DependencyNodeID]domain.InspectionReport {
	reports := make(map[domain.DependencyNodeID]domain.InspectionReport, len(acquired))
	for node, artifact := range acquired {
		reports[node] = retainedReport(artifact)
	}
	return reports
}

func TestInspectorForwardsFrozenProjectAndSupplementalExecutionInputs(t *testing.T) {
	graph, graphAcquired := advisoryGraph(t)
	target, err := domain.NewInstallTarget("/advisory-fixture")
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	control, err := domain.NewProjectControlFile("package-lock.json", []byte("frozen control bytes"), true)
	if err != nil {
		t.Fatal(err)
	}
	controlDigest, err := domain.NewProjectControlDigest(control.Name(), control.Digest())
	if err != nil {
		t.Fatal(err)
	}
	var resolved []domain.ResolvedArtifact
	projectAcquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, node := range graph.Nodes() {
		resolved = append(resolved, node.Artifact())
		id, err := domain.ProjectDependencyNodeID(node.Artifact().Identity())
		if err != nil {
			t.Fatal(err)
		}
		projectAcquired[id] = graphAcquired[node.Node()]
	}
	snapshot, err := domain.NewProjectDependencySnapshot(install, resolved[0].Identity().Source(), []domain.ProjectControlDigest{controlDigest}, resolved, control.Digest())
	if err != nil {
		t.Fatal(err)
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot([]domain.ProjectControlFile{control}, []domain.ProjectControlFile{control}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var supplemental domain.LockedDependencyGraph
	for _, node := range graph.Nodes() {
		if node.Node() == graph.Primary() {
			continue
		}
		primary, err := domain.NewLockedDependency(node.Node(), domain.DependencyPrimary, node.Artifact())
		if err != nil {
			t.Fatal(err)
		}
		supplemental, err = domain.NewLockedDependencyGraph([]domain.LockedDependency{primary}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	selection, err := domain.NewExecutionDependencySelection(graph, []domain.LockedDependencyGraph{supplemental})
	if err != nil {
		t.Fatal(err)
	}
	inner := &selectionRetainedInspector{}
	posts := 0
	inspector, err := newInspector(inner, &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		posts++
		var message struct {
			Queries []batchQuery `json:"queries"`
		}
		if json.NewDecoder(request.Body).Decode(&message) != nil || len(message.Queries) != 2 {
			t.Fatal("selection membership lost")
		}
		return jsonResponse(`{"results":[{},{}]}`), nil
	})}, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	for _, invoke := range []func() (map[domain.DependencyNodeID]domain.InspectionReport, error){
		func() (map[domain.DependencyNodeID]domain.InspectionReport, error) {
			return inspector.InspectProject(context.Background(), snapshot, projectAcquired)
		},
		func() (map[domain.DependencyNodeID]domain.InspectionReport, error) {
			return inspector.InspectProjectSelection(context.Background(), update, projectAcquired)
		},
		func() (map[domain.DependencyNodeID]domain.InspectionReport, error) {
			return inspector.InspectExecutionSelection(context.Background(), selection, graphAcquired)
		},
	} {
		reports, err := invoke()
		if err != nil || len(reports) != 2 {
			t.Fatal("existing selection entry failed", err)
		}
		for _, report := range reports {
			if len(report.Executions()) != 2 {
				t.Fatal("selection report missing original/advisory")
			}
		}
	}
	if posts != 3 || strings.Join(inner.calls, ",") != "project,frozen-selection,execution-selection" {
		t.Fatal("typed selection entry was replaced by per-artifact inspection", inner.calls, posts)
	}
	unsupported := &retainedInspector{}
	wrapped, err := newInspector(unsupported, inspector.client, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrapped.InspectExecutionSelection(context.Background(), selection, graphAcquired); err == nil || len(unsupported.calls) != 0 || posts != 3 {
		t.Fatal("wrapper invented execution-selection support for a per-artifact inspector", err)
	}
}
