package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func npmExecutionControlFixture(t *testing.T, sourceName, integrity string) domain.ExecutionContextData {
	t.Helper()
	source, err := domain.NewSourceID(sourceName)
	if err != nil {
		t.Fatal(err)
	}
	var nodes []domain.LockedDependency
	var inspections []domain.DependencyInspection
	var primary domain.AcquiredArtifact
	digest := func(value string) domain.ContentDigest {
		sum := sha256.Sum256([]byte(value))
		result, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	allow, err := domain.NewPolicyDecision(domain.DecisionAllow, "fixture-policy", 1, []string{"ALLOW"})
	if err != nil {
		t.Fatal(err)
	}
	for index, name := range []string{"primary", "child"} {
		node, err := domain.NewDependencyNodeID(name)
		if err != nil {
			t.Fatal(err)
		}
		identity, err := domain.NewResolvedArtifactIdentity(source, name, "1.0.0", "tarball")
		if err != nil {
			t.Fatal(err)
		}
		resolved, err := domain.NewResolvedArtifact(identity, "registry:"+name, integrity)
		if err != nil {
			t.Fatal(err)
		}
		role := domain.DependencyTransitive
		if index == 0 {
			role = domain.DependencyPrimary
		}
		locked, err := domain.NewLockedDependencyWithRecordPath(node, role, resolved, "node_modules/"+name, false)
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, locked)
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := domain.NewAcquiredArtifact(identity, digest(name), "intake:"+run.String()+":tarball", uint64(len(name)))
		if integrity != "" {
			artifact, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest(name), "intake:"+run.String()+":tarball", uint64(len(name)), integrity)
		}
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			primary = artifact
		}
		checkID, _ := domain.NewCheckID("fixture-check")
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		evidenceID, _ := domain.NewEvidenceID("fixture-" + name)
		reference, _ := domain.NewEvidenceReference(evidenceID, "fixture:"+name)
		inspection, err := domain.NewDependencyInspection(node, run, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, allow)
		if err != nil {
			t.Fatal(err)
		}
		inspections = append(inspections, inspection)
	}
	edge, err := domain.NewDependencyEdge(nodes[0].Node(), nodes[1].Node())
	if err != nil {
		t.Fatal(err)
	}
	graph, err := domain.NewLockedDependencyGraph(nodes, []domain.DependencyEdge{edge})
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewVerifiedSet(inspected, allow)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewDependencyResolution(graph, "fixture-resolver", digest("original npm lock"))
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	return domain.ExecutionContextData{OperationID: operation, Resolution: resolution, Set: set, EntryArtifact: primary, Command: "primary", EntryPath: "cli.js", EntryDigest: digest("entry bytes"), EntrySize: 11, RuntimeIdentity: "fixture-node-runtime", Arguments: []string{"private fixture argument"}}
}

func TestNPMExecutionControlsRenderExactWholeSetWithoutInstallTarget(t *testing.T) {
	t.Parallel()
	data := npmExecutionControlFixture(t, "npm", "sha512-declared")
	execution, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	controls, err := BuildNPMExecutionControls(context.Background(), execution)
	if err != nil || len(controls) != 2 || controls[0].Name() != "package.json" || controls[1].Name() != "package-lock.json" {
		t.Fatal("controller files unavailable", err)
	}
	var manifest struct {
		Name, Version string
		Private       bool
		Dependencies  map[string]string
	}
	if json.Unmarshal(controls[0].Body(), &manifest) != nil || !manifest.Private || manifest.Version != "1.0.0" || len(manifest.Dependencies) != 1 || manifest.Dependencies["primary"] != "file:.heliopause/artifacts/"+data.EntryArtifact.Digest().String()+".tgz" {
		t.Fatal("root is not exact generated local source")
	}
	var lock struct {
		LockfileVersion int
		Packages        map[string]struct {
			Name, Version, Resolved, Integrity string
			Dependencies                       map[string]string
		}
	}
	if json.Unmarshal(controls[1].Body(), &lock) != nil || lock.LockfileVersion != 3 || len(lock.Packages) != 3 {
		t.Fatal("complete v3 graph unavailable")
	}
	for _, inspection := range data.Set.Inspected().Inspections() {
		name := inspection.Artifact().Identity().Name()
		pkg, present := lock.Packages["node_modules/"+name]
		if !present || pkg.Name != name || pkg.Version != "1.0.0" || pkg.Resolved != "file:.heliopause/artifacts/"+inspection.Artifact().Digest().String()+".tgz" || pkg.Integrity != "sha512-declared" {
			t.Fatal("package source or declared integrity changed")
		}
	}
	if lock.Packages["node_modules/primary"].Dependencies["child"] != "1.0.0" {
		t.Fatal("exact dependency edge lost")
	}
	for _, file := range controls {
		if !file.Present() || strings.Contains(string(file.Body()), "private fixture argument") || strings.Contains(string(file.Body()), "registry:") {
			t.Fatal("private request or nonlocal source leaked")
		}
	}
	data.Arguments = []string{"other private argument"}
	otherExecution, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	other, err := BuildNPMExecutionControls(context.Background(), otherExecution)
	if err != nil {
		t.Fatal(err)
	}
	for i := range controls {
		if !bytes.Equal(controls[i].Body(), other[i].Body()) || controls[i].Digest() != other[i].Digest() {
			t.Fatal("argv changed controller source identity")
		}
	}
	body := controls[0].Body()
	body[0] = 'X'
	if controls[0].Body()[0] != '{' {
		t.Fatal("control getter exposed mutable storage")
	}
}

func TestNPMExecutionControlsRejectUnsupportedCurrentSetAndCancellation(t *testing.T) {
	t.Parallel()
	for _, input := range []struct{ source, integrity string }{{"pypi", "sha512-declared"}, {"github-release", "sha512-declared"}} {
		data := npmExecutionControlFixture(t, input.source, input.integrity)
		execution, err := domain.NewExecutionContext(data)
		if err != nil {
			t.Fatal(err)
		}
		if controls, err := BuildNPMExecutionControls(context.Background(), execution); err == nil || len(controls) != 0 {
			t.Fatal("unsupported source rendered")
		}
	}
	data := npmExecutionControlFixture(t, "npm", "sha512-declared")
	execution, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("fixture cancellation")
	cancel(cause)
	if controls, err := BuildNPMExecutionControls(ctx, execution); len(controls) != 0 || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation cause lost", err)
	}
	for _, missing := range []context.Context{nil} {
		if controls, err := BuildNPMExecutionControls(missing, execution); err == nil || len(controls) != 0 {
			t.Fatal("nil context accepted")
		}
	}
	if controls, err := BuildNPMExecutionControls(context.Background(), domain.ExecutionContext{}); err == nil || len(controls) != 0 {
		t.Fatal("zero execution context accepted")
	}
}
