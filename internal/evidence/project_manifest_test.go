package evidence_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
)

func projectRecordFixture(t *testing.T, empty bool) (domain.ProjectVerifiedSet, []domain.LockedDependency, []domain.ProjectControlFile) {
	t.Helper()
	control, _ := domain.NewProjectControlFile("selected.lock", []byte("immutable selected controls"), true)
	digest, _ := domain.NewProjectControlDigest(control.Name(), control.Digest())
	target, _ := domain.NewInstallTarget("/tmp/complete-project-records")
	install, _ := domain.NewPythonVenvInstallContext(target)
	source, _ := domain.NewSourceID("pypi")
	graph, _ := domain.NewSHA256Digest(strings.Repeat("f", 64))
	nodes := []domain.LockedDependency{}
	artifacts := []domain.ResolvedArtifact{}
	inspections := []domain.DependencyInspection{}
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "synthetic-project", 1, []string{"SYNTHETIC_ALLOW"})
	if !empty {
		for _, name := range []string{"alpha", "beta"} {
			id, _ := domain.NewResolvedArtifactIdentity(source, name, "1.0", "wheel")
			resolved, _ := domain.NewResolvedArtifact(id, "https://files.pythonhosted.org/packages/"+name+"-1.0-py3-none-any.whl", "sha256:"+graph.String())
			artifacts = append(artifacts, resolved)
			node, _ := domain.ProjectDependencyNodeID(id)
			locked, _ := domain.NewLockedDependencyWithRecordPath(node, domain.DependencyTransitive, resolved, name+"-1.0-py3-none-any.whl", false)
			nodes = append(nodes, locked)
			run, _ := domain.NewRunID()
			a, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, graph, "intake:"+run.String()+":wheel", 1, resolved.DeclaredIntegrity())
			checks := []domain.CheckExecution{}
			refs := []domain.EvidenceReference{}
			for _, kind := range []domain.CheckKind{domain.CheckVerification, domain.CheckInspection} {
				cID, _ := domain.NewCheckID("synthetic-" + strings.ToLower(string(kind)))
				check, _ := domain.NewCheckExecution(cID, kind, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
				checks = append(checks, check)
				eID, _ := domain.NewEvidenceID(cID.String())
				ref, _ := domain.NewEvidenceReference(eID, "evidence:"+run.String()+":"+eID.String())
				refs = append(refs, ref)
			}
			i, err := domain.NewDependencyInspection(node, run, a, checks, refs, decision)
			if err != nil {
				t.Fatal(err)
			}
			inspections = append(inspections, i)
		}
	}
	var snapshot domain.ProjectDependencySnapshot
	var err error
	if empty {
		snapshot, err = domain.NewDependencyFreeProjectSnapshot(install, source, []domain.ProjectControlDigest{digest}, graph)
	} else {
		snapshot, err = domain.NewProjectDependencySnapshot(install, source, []domain.ProjectControlDigest{digest}, artifacts, graph)
	}
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, inspections)
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	return set, nodes, []domain.ProjectControlFile{control}
}

func TestProjectManifestIsPrimaryFreeDeterministicAndCoversActualFacts(t *testing.T) {
	for _, empty := range []bool{false, true} {
		set, nodes, controls := projectRecordFixture(t, empty)
		op, _ := domain.NewOperationID()
		edges := []domain.DependencyEdge{}
		if !empty {
			edge, _ := domain.NewDependencyEdge(nodes[0].Node(), nodes[1].Node())
			edges = append(edges, edge)
		}
		id, manifest, bom, err := (evidence.Generator{}).BuildProject(context.Background(), op, set, nodes, edges, controls, "synthetic-runtime")
		if err != nil {
			t.Fatal(err)
		}
		if !empty {
			nodes[0], nodes[1] = nodes[1], nodes[0]
		}
		second, m2, b2, err := (evidence.Generator{}).BuildProject(context.Background(), op, set, nodes, edges, controls, "synthetic-runtime")
		if err != nil || id != second || string(manifest) != string(m2) || string(bom) != string(b2) {
			t.Fatal("project records are not deterministic", err)
		}
		var doc map[string]any
		if json.Unmarshal(manifest, &doc) != nil || doc["dependency_free"] != empty || doc["operation_id"] != op.String() || doc["manifest_id"] != id.String() {
			t.Fatal("project manifest context is incomplete")
		}
		if _, ok := doc["primary"]; ok {
			t.Fatal("project manifest invented a primary")
		}
		var sbom struct {
			Components   []json.RawMessage `json:"components"`
			Dependencies []json.RawMessage `json:"dependencies"`
		}
		if json.Unmarshal(bom, &sbom) != nil || len(sbom.Components) != len(nodes) || len(sbom.Dependencies) != len(nodes) {
			t.Fatal("SBOM does not cover exact complete set")
		}
	}
}

func TestProjectManifestRejectsForeignControlsNodesEdgesAndCancellation(t *testing.T) {
	for _, mode := range []string{"control", "missing-node", "primary", "foreign-edge", "repeated-edge", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			set, nodes, controls := projectRecordFixture(t, false)
			op, _ := domain.NewOperationID()
			ctx := context.Background()
			edges := []domain.DependencyEdge{}
			switch mode {
			case "control":
				controls[0], _ = domain.NewProjectControlFile(controls[0].Name(), []byte("changed"), true)
			case "missing-node":
				nodes = nodes[:1]
			case "primary":
				n := nodes[0]
				nodes[0], _ = domain.NewLockedDependencyWithRecordPath(n.Node(), domain.DependencyPrimary, n.Artifact(), n.RecordPath(), false)
			case "foreign-edge":
				other, _ := domain.NewDependencyNodeID("foreign")
				edge, _ := domain.NewDependencyEdge(nodes[0].Node(), other)
				edges = append(edges, edge)
			case "repeated-edge":
				edge, _ := domain.NewDependencyEdge(nodes[0].Node(), nodes[1].Node())
				edges = append(edges, edge, edge)
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if id, _, _, err := (evidence.Generator{}).BuildProject(ctx, op, set, nodes, edges, controls, "synthetic-runtime"); err == nil || id.String() != "" {
				t.Fatal("invalid project records accepted")
			}
		})
	}
}

func TestProjectManifestPreservesExplicitControlAbsence(t *testing.T) {
	original, nodes, controls := projectRecordFixture(t, true)
	op, _ := domain.NewOperationID()
	ordinary, ordinaryBody, ordinarySBOM, err := (evidence.Generator{}).BuildProject(context.Background(), op, original, nodes, nil, controls, "synthetic-runtime")
	if err != nil {
		t.Fatal(err)
	}
	explicit, explicitBody, explicitSBOM, err := (evidence.Generator{}).BuildProjectWithControlPresence(context.Background(), op, original, nodes, nil, controls, "synthetic-runtime")
	if err != nil || ordinary != explicit || string(ordinaryBody) != string(explicitBody) || string(ordinarySBOM) != string(explicitSBOM) {
		t.Fatal("present-control wire changed", err)
	}
	var ids []domain.ContentDigest
	for _, present := range []bool{false, true} {
		control, err := domain.NewProjectControlFile("selected.lock", nil, present)
		if err != nil {
			t.Fatal(err)
		}
		digest, _ := domain.NewProjectControlDigest(control.Name(), control.Digest())
		prior := original.Inspected().Snapshot()
		snapshot, err := domain.NewDependencyFreeProjectSnapshot(prior.Context(), prior.Source(), []domain.ProjectControlDigest{digest}, prior.GraphDigest())
		if err != nil {
			t.Fatal(err)
		}
		inspected, err := domain.NewInspectedProjectSet(snapshot, nil)
		if err != nil {
			t.Fatal(err)
		}
		set, err := domain.NewProjectVerifiedSet(inspected, original.Decision())
		if err != nil {
			t.Fatal(err)
		}
		if !present {
			if id, _, _, err := (evidence.Generator{}).BuildProject(context.Background(), op, set, nil, nil, []domain.ProjectControlFile{control}, "synthetic-runtime"); err == nil || id.String() != "" {
				t.Fatal("ordinary builder accepted absent control")
			}
		}
		id, body, _, err := (evidence.Generator{}).BuildProjectWithControlPresence(context.Background(), op, set, nil, nil, []domain.ProjectControlFile{control}, "synthetic-runtime")
		if err != nil {
			t.Fatal(err)
		}
		var record struct {
			Controls []struct {
				Name, SHA256 string
				Content      []byte
				Absent       bool
			} `json:"project_controls"`
		}
		if json.Unmarshal(body, &record) != nil || len(record.Controls) != 1 || record.Controls[0].Absent == present || record.Controls[0].SHA256 != control.Digest().String() {
			t.Fatal("explicit presence lost")
		}
		ids = append(ids, id)
	}
	if ids[0] == ids[1] {
		t.Fatal("absent and present zero-byte controls share Manifest identity")
	}
}
