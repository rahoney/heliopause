package pypi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

// Uses real authenticated wheel parsing and the production composite owner;
// only the external transaction is replaced by a recording runner.
func TestInspectionPrerequisiteKeepsRequestedGraphAndBaseFailure(t *testing.T) {
	for _, profile := range []string{"cpu", "cu126", "cu130", "cu132"} {
		t.Run(profile, func(t *testing.T) {
			selectedProfile, ok := artifactpypi.PyTorchProfile(profile)
			if !ok {
				t.Fatal(profile)
			}
			ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), selectedProfile)
			if err != nil {
				t.Fatal(err)
			}
			for _, rootFails := range []bool{false, true} {
				root := t.TempDir()
				graph, acquired, input := prerequisiteGraphFixture(t, root)
				beforeNodes, beforeEdges := graph.Nodes(), graph.Edges()
				runner := &prerequisiteRunner{t: t, rootFails: rootFails}
				static, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
				dynamic, _ := NewDynamicInspector(runner)
				inspector, _ := NewCompositeInspector(static, dynamic)
				inspector.WithInspectionPrerequisites(func(context.Context, domain.LockedDependencyGraph) ([]InspectionPrerequisite, error) {
					return []InspectionPrerequisite{input}, nil
				})
				lifetime := &prerequisiteStorageCompletion{completed: make(map[domain.RunID]bool)}
				reports, err := inspector.InspectGraph(ports.WithStorage(ctx, lifetime), graph, acquired)
				if err != nil {
					t.Fatal(err)
				}
				directory, _, _, locationErr := static.artifactLocation(input.Artifact)
				inputRun, runErr := domain.ParseRunID(filepath.Base(directory))
				if locationErr != nil || runErr != nil || len(lifetime.completed) != 1 || !lifetime.completed[inputRun] {
					t.Fatal("supporting input completion was not bound to its own Run", lifetime.completed, locationErr, runErr)
				}
				if len(reports) != 2 || !reflect.DeepEqual(beforeNodes, graph.Nodes()) || !reflect.DeepEqual(beforeEdges, graph.Edges()) {
					t.Fatal("inspection input changed original membership")
				}
				if len(runner.calls) != 2 || runner.calls[0] != "dependency:supplemented" || runner.calls[1] != "root:base" {
					t.Fatal(runner.calls)
				}
				rootReport := reports[graph.Primary()]
				status := domain.ExecutionCompleted
				if rootFails {
					status = domain.ExecutionIncomplete
				}
				executions := rootReport.Executions()
				if len(executions) != 2 || !executions[1].Required() || executions[1].Status() != status {
					t.Fatalf("base failure changed: %+v", executions)
				}
				rootArtifact := acquired[graph.Primary()]
				verified, err := (verificationpypi.IntegrityVerifier{}).Verify(ctx, rootArtifact)
				if err != nil {
					t.Fatal(err)
				}
				var refs []domain.EvidenceReference
				for _, ev := range append(verified.Evidence(), rootReport.Evidence()...) {
					ref, err := domain.NewEvidenceReference(ev.ID(), "evidence:"+ev.ID().String())
					if err != nil {
						t.Fatal(err)
					}
					refs = append(refs, ref)
				}
				run, _ := domain.NewRunID()
				declaredVerification, err := (verificationpypi.IntegrityVerifier{}).ExpectedChecks(ctx, rootArtifact, domain.CheckVerification)
				if err != nil {
					t.Fatal(err)
				}
				declaredInspection, err := inspector.ExpectedChecks(ctx, rootArtifact, domain.CheckInspection)
				if err != nil {
					t.Fatal(err)
				}
				plan, err := domain.NewCheckPlan(run, rootArtifact, append(declaredVerification, declaredInspection...))
				if err != nil {
					t.Fatal(err)
				}
				policyInput, err := domain.NewPolicyInput(run, rootArtifact, verified, rootReport, refs, plan)
				if err != nil {
					t.Fatal(err)
				}
				decision, err := (policy.M3{}).Evaluate(policyInput)
				if err != nil {
					t.Fatal(err)
				}
				if (decision.Decision() == domain.DecisionAllow) == rootFails {
					t.Fatal("base failure changed Policy eligibility")
				}
				for _, report := range reports {
					for _, ev := range report.Evidence() {
						if ev.Identity().Name() == "support" {
							t.Fatal("input became original graph report")
						}
					}
				}
				for node, report := range reports {
					if node == graph.Primary() {
						continue
					}
					summary := ""
					matches := 0
					for _, fact := range report.Evidence() {
						if fact.Kind() == "pypi-dynamic-import-supplemented" {
							summary = fact.Summary()
							matches++
						}
					}
					if matches != 1 {
						t.Fatalf("supplemented scope evidence count=%d", matches)
					}
					if !strings.Contains(summary, `"original_environment":"NOT_ATTESTED"`) || !strings.Contains(summary, input.Artifact.Digest().String()) {
						t.Fatal(summary)
					}
				}
			}
		})
	}
}

func TestInspectionPrerequisiteRejectsUnapprovedOrUnvalidatedInputsBeforeExecution(t *testing.T) {
	for _, mode := range []string{"root", "unselected", "unapproved", "wrong-source", "wrong-digest", "wrong-ABI", "tampered", "replacement", "shadow", "transitive", "startup"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			graph, acquired, input := prerequisiteGraphFixture(t, root)
			switch mode {
			case "root":
				input.TargetDigest = acquired[graph.Primary()].Digest().String()
			case "unselected":
				input.TargetDigest = strings.Repeat("f", 64)
			case "unapproved":
				input.Reason = "ARTIFACT_REQUESTED_IT"
			case "wrong-source":
				source, _ := domain.NewSourceID("pytorch-cu126")
				id, _ := domain.NewResolvedArtifactIdentity(source, "support", "1.0", "wheel")
				input.Artifact, _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(id, input.Artifact.Digest(), input.Artifact.ContentHandle(), input.Artifact.SizeBytes(), mustDeclaredIntegrity(t, input.Artifact))
			case "wrong-digest":
				digest, _ := domain.NewSHA256Digest(strings.Repeat("f", 64))
				input.Artifact, _ = domain.NewAcquiredArtifactWithDeclaredIntegrity(input.Artifact.Identity(), digest, input.Artifact.ContentHandle(), input.Artifact.SizeBytes(), mustDeclaredIntegrity(t, input.Artifact))
			case "wrong-ABI":
				input.Artifact = prerequisiteFixtureWheel(t, root, "support", "cp313-cp313-manylinux_2_36_x86_64", nil, "")
			case "tampered":
				parts := strings.Split(input.Artifact.ContentHandle(), ":")
				if err := os.WriteFile(filepath.Join(root, parts[1], "wheel.whl"), []byte("tampered"), 0600); err != nil {
					t.Fatal(err)
				}
			case "replacement":
				input.Artifact = prerequisiteFixtureWheel(t, root, "dependency", "py3-none-any", nil, "")
			case "shadow":
				input.Artifact = prerequisiteFixtureWheel(t, root, "support", "py3-none-any", map[string]string{"dependency/other.py": "pass\n"}, "")
			case "transitive":
				input.Artifact = prerequisiteFixtureWheel(t, root, "support", "py3-none-any", nil, "Requires-Dist: root>=1\n")
			case "startup":
				input.Artifact = prerequisiteFixtureWheel(t, root, "support", "py3-none-any", map[string]string{"active.pth": "import support\n"}, "")
			}
			static, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
			dynamic, _ := NewDynamicInspector(panicWheelRunner{})
			inspector, _ := NewCompositeInspector(static, dynamic)
			inspector.WithInspectionPrerequisites(func(context.Context, domain.LockedDependencyGraph) ([]InspectionPrerequisite, error) {
				return []InspectionPrerequisite{input}, nil
			})
			if _, err := inspector.InspectGraph(context.Background(), graph, acquired); err == nil {
				t.Fatal("invalid prerequisite accepted")
			}
		})
	}
}

func TestInspectionPrerequisiteEvidenceBindsInputsAndPreservesFindings(t *testing.T) {
	root := t.TempDir()
	_, acquired, input := prerequisiteGraphFixture(t, root)
	var target domain.AcquiredArtifact
	for _, a := range acquired {
		if a.Identity().Name() == "dependency" {
			target = a
		}
	}
	static, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	wheel, _, err := static.InspectWheel(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	runner := &prerequisiteRunner{t: t, observations: []domain.SandboxObservation{observation(t, domain.ObservationProcess, "process-exec-unexpected")}}
	inspector, _ := NewDynamicInspector(runner)
	report, err := inspector.InspectWheelWithPrerequisites(context.Background(), target, wheel, []domain.AcquiredArtifact{target}, []domain.AcquiredArtifact{input.Artifact})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings()) != 1 || report.Findings()[0].Code() != "M3_UNEXPECTED_PROCESS" {
		t.Fatal(report.Findings())
	}
	summary := report.Evidence()[0].Summary()
	other := prerequisiteFixtureWheel(t, root, "support", "py3-none-any", map[string]string{"support/resource": "changed"}, "")
	changed, err := inspector.InspectWheelWithPrerequisites(context.Background(), target, wheel, []domain.AcquiredArtifact{target}, []domain.AcquiredArtifact{other})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Evidence()[0].Summary() == summary || len(summary) > 1024 {
		t.Fatal("evidence identity did not bind input")
	}
	for _, inputs := range [][]domain.AcquiredArtifact{nil, {input.Artifact, other}} {
		if _, err := inspector.InspectWheelWithPrerequisites(context.Background(), target, wheel, []domain.AcquiredArtifact{target}, inputs); err == nil {
			t.Fatal("missing/extra input accepted")
		}
	}
}

type prerequisiteRunner struct {
	t            *testing.T
	rootFails    bool
	calls        []string
	observations []domain.SandboxObservation
}

func (r *prerequisiteRunner) InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error) {
	r.t.Fatal("untyped execution used")
	return domain.SandboxResult{}, nil
}
func (r *prerequisiteRunner) InspectWheelWithClosure(context.Context, domain.AcquiredArtifact, []string, []domain.AcquiredArtifact) (domain.SandboxResult, error) {
	r.t.Fatal("untyped closure used")
	return domain.SandboxResult{}, nil
}
func (r *prerequisiteRunner) InspectWheelWithPlan(_ context.Context, target domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, base []domain.AcquiredArtifact) (sandbox.PythonObservationResult, error) {
	r.calls = append(r.calls, target.Identity().Name()+":base")
	for _, a := range base {
		if a.Identity().Name() == "support" {
			r.t.Fatal("supplement leaked into base")
		}
	}
	if len(plan.Units) != 1 {
		r.t.Fatal("original plan modified")
	}
	if r.rootFails && target.Identity().Name() == "root" {
		session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
		result, _ := domain.NewSandboxResult(session, domain.SandboxIncomplete, "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE", nil)
		return sandbox.PythonObservationResult{SandboxResult: result}, nil
	}
	return sandbox.PythonObservationResult{SandboxResult: completedResult(r.t)}, nil
}
func (r *prerequisiteRunner) InspectWheelWithPrerequisites(_ context.Context, target domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, base, inputs []domain.AcquiredArtifact) (sandbox.PythonObservationResult, error) {
	r.calls = append(r.calls, target.Identity().Name()+":supplemented")
	if target.Identity().Name() == "root" || len(inputs) != 1 || inputs[0].Identity().Name() != "support" || len(plan.Units) != 1 {
		r.t.Fatal("wrong target/environment")
	}
	for _, a := range base {
		if a.Identity().Name() == "support" {
			r.t.Fatal("original closure modified")
		}
	}
	return sandbox.PythonObservationResult{SandboxResult: completedResultWithObservations(r.t, r.observations)}, nil
}

func prerequisiteGraphFixture(t *testing.T, root string) (domain.LockedDependencyGraph, map[domain.DependencyNodeID]domain.AcquiredArtifact, InspectionPrerequisite) {
	t.Helper()
	var nodes []domain.LockedDependency
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, name := range []string{"root", "dependency"} {
		a := prerequisiteFixtureWheel(t, root, name, "py3-none-any", nil, "")
		resolved, _ := domain.NewResolvedArtifact(a.Identity(), "https://files.pythonhosted.org/"+name+".whl", mustDeclaredIntegrity(t, a))
		id, _ := domain.NewDependencyNodeID(name)
		role := domain.DependencyTransitive
		if name == "root" {
			role = domain.DependencyPrimary
		}
		node, _ := domain.NewLockedDependency(id, role, resolved)
		nodes = append(nodes, node)
		acquired[id] = a
	}
	edge, _ := domain.NewDependencyEdge(nodes[0].Node(), nodes[1].Node())
	graph, err := domain.NewLockedDependencyGraph(nodes, []domain.DependencyEdge{edge})
	if err != nil {
		t.Fatal(err)
	}
	return graph, acquired, InspectionPrerequisite{TargetDigest: acquired[nodes[1].Node()].Digest().String(), Artifact: prerequisiteFixtureWheel(t, root, "support", "py3-none-any", nil, ""), Reason: "BROADER_MODULE_PROBES"}
}

func prerequisiteFixtureWheel(t *testing.T, root, name, tag string, extra map[string]string, metadata string) domain.AcquiredArtifact {
	t.Helper()
	dist := name + "-1.0.dist-info"
	files := map[string]string{name + "/__init__.py": "pass\n", dist + "/METADATA": "Metadata-Version: 2.4\nName: " + name + "\nVersion: 1.0\n" + metadata, dist + "/WHEEL": "Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: " + tag + "\n"}
	for k, v := range extra {
		files[k] = v
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	record := ""
	for _, k := range keys {
		h := sha256.Sum256([]byte(files[k]))
		record += fmt.Sprintf("%s,sha256=%s,%d\n", k, base64.RawURLEncoding.EncodeToString(h[:]), len(files[k]))
	}
	files[dist+"/RECORD"] = record + dist + "/RECORD,,\n"
	keys = append(keys, dist+"/RECORD")
	sort.Strings(keys)
	var body bytes.Buffer
	z := zip.NewWriter(&body)
	for _, k := range keys {
		w, err := z.Create(k)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte(files[k])); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(body.Bytes())
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(h[:]))
	run, _ := domain.NewRunID()
	dir := filepath.Join(root, run.String())
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string][]byte{"wheel.whl": body.Bytes(), "filename": []byte(name + "-1.0-" + tag + ".whl")} {
		if err := os.WriteFile(filepath.Join(dir, k), v, 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, _ := domain.NewSourceID("pypi")
	id, _ := domain.NewResolvedArtifactIdentity(source, name, "1.0", "wheel")
	a, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, digest, "intake:"+run.String()+":wheel", uint64(body.Len()), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// Input completion is independent of the original requested graph's Policy.
// The common finalizer still waits for every shared consumer before deletion.
type prerequisiteStorageCompletion struct{ completed map[domain.RunID]bool }

func (*prerequisiteStorageCompletion) Policy() domain.StoragePolicy {
	return domain.DefaultStoragePolicy()
}
func (*prerequisiteStorageCompletion) Reserve(context.Context, ports.StorageRequest) (ports.StorageLease, error) {
	return nil, fmt.Errorf("unexpected inspection-time acquisition")
}
func (s *prerequisiteStorageCompletion) NoteInspection(_ context.Context, run domain.RunID, complete bool) error {
	s.completed[run] = complete
	return nil
}

func TestInspectionPrerequisiteKeepsNativeOperationSummaryWithinEvidenceBudget(t *testing.T) {
	root := t.TempDir()
	_, acquired, input := prerequisiteGraphFixture(t, root)
	var target domain.AcquiredArtifact
	for _, artifact := range acquired {
		if artifact.Identity().Name() == "dependency" {
			target = artifact
		}
	}
	static, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	wheel, _, err := static.InspectWheel(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	var observations []domain.SandboxObservation
	// Fixed normalized counts from the current tiny transient-file native case.
	for subject, count := range map[string]uint64{"filesystem-workspace-access": 55, "filesystem-workspace-create": 20, "filesystem-workspace-permission-operation": 14, "filesystem-workspace-rename": 5, "filesystem-workspace-size-operation": 16, "filesystem-workspace-unlink": 19, "filesystem-workspace-write": 17} {
		fact, err := domain.NewCountedSandboxObservation(domain.ObservationFilesystem, subject, count)
		if err != nil {
			t.Fatal(err)
		}
		observations = append(observations, fact)
	}
	for subject, count := range map[string]uint64{"environment-minimal-v1-completed": 1, "environment-mock-development-v1-completed": 1, "process-exec-expected": 73, "python-observation-experiment-completed": 1} {
		fact, err := domain.NewCountedSandboxObservation(domain.ObservationProcess, subject, count)
		if err != nil {
			t.Fatal(err)
		}
		observations = append(observations, fact)
	}
	inspector, _ := NewDynamicInspector(&prerequisiteRunner{t: t, observations: observations})
	report, err := inspector.InspectWheelWithPrerequisites(context.Background(), target, wheel, []domain.AcquiredArtifact{target}, []domain.AcquiredArtifact{input.Artifact})
	if err != nil || report.Execution().Status() != domain.ExecutionCompleted {
		t.Fatalf("normal observed prerequisite failed normalization: %v", err)
	}
	var metadata, observed domain.Evidence
	for _, fact := range report.Evidence() {
		if len(fact.Summary()) > 1024 {
			t.Fatal("evidence bound enlarged")
		}
		if fact.Kind() == "pypi-dynamic-import-supplemented" {
			metadata = fact
		}
		if fact.ID().String() == supplementedObservationEvidenceID {
			observed = fact
		}
	}
	digest := sha256.Sum256([]byte(observed.Summary()))
	if observed.Summary() == "" || observed.Identity() != target.Identity() || observed.Digest() != target.Digest() || observed.CheckID() != metadata.CheckID() || !strings.Contains(metadata.Summary(), `"schema":"m5-inspection-environment/v2"`) || !strings.Contains(metadata.Summary(), observed.ID().String()) || !strings.Contains(metadata.Summary(), hex.EncodeToString(digest[:])) {
		t.Fatal("supplemented observations lost same-subject reference or digest")
	}
	for _, subject := range []string{"filesystem-workspace-create", "filesystem-workspace-write", "filesystem-workspace-rename", "filesystem-workspace-unlink", "filesystem-workspace-permission-operation", "filesystem-workspace-size-operation"} {
		if !strings.Contains(observed.Summary(), subject) {
			t.Fatal("successful operation evidence omitted", subject)
		}
	}
}
