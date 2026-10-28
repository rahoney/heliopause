package domain_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func executionContextFixture(t *testing.T) domain.ExecutionContextData {
	t.Helper()
	graph, first, second := inspectedGraph(t)
	a, b := dependencyInspection(t, first, "a"), dependencyInspection(t, second, "b")
	inspected, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{a, b})
	if err != nil {
		t.Fatal(err)
	}
	allow, _ := domain.NewPolicyDecision(domain.DecisionAllow, "set-policy", 3, []string{"ALLOW"})
	set, err := domain.NewVerifiedSet(inspected, allow)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	lock, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	member, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	resolution, err := domain.NewDependencyResolution(graph, "resolver-runtime-identity", lock)
	if err != nil {
		t.Fatal(err)
	}
	return domain.ExecutionContextData{OperationID: operation, Resolution: resolution, Set: set,
		EntryArtifact: a.Artifact(), Command: "first", EntryPath: "bin/cli.js", EntryDigest: member,
		EntrySize: 3, RuntimeIdentity: "required-execution-runtime", Arguments: []string{"a b", "", "--lang", "line\nbreak"}}
}

func TestExecutionContextBindsApprovedSetAndPrivateArguments(t *testing.T) {
	data := executionContextFixture(t)
	ctx, err := domain.NewExecutionContext(data)
	if err != nil || !ctx.Valid() {
		t.Fatal("matching non-install context rejected", err)
	}
	if ctx.OperationID() != data.OperationID || ctx.EntryArtifact() != data.EntryArtifact || ctx.Command() != data.Command ||
		ctx.EntryPath() != data.EntryPath || ctx.EntryDigest() != data.EntryDigest || ctx.EntrySize() != data.EntrySize ||
		ctx.RuntimeIdentity() != data.RuntimeIdentity || ctx.SourceFingerprint() != data.Resolution.LockfileDigest() || ctx.PolicyFingerprint().String() == "" {
		t.Fatal("frozen entry or source/runtime identity changed")
	}
	if !ctx.Set().Valid() || !slices.Equal(ctx.Resolution().Graph().Nodes(), data.Resolution.Graph().Nodes()) || !slices.Equal(ctx.Arguments(), data.Arguments) {
		t.Fatal("whole set or argument vector changed")
	}
	data.Arguments[0] = "changed by caller"
	returned := ctx.Arguments()
	returned[1] = "changed returned copy"
	if !slices.Equal(ctx.Arguments(), []string{"a b", "", "--lang", "line\nbreak"}) {
		t.Fatal("mutable argv alias retained")
	}
	other, err := domain.NewExecutionContext(data)
	if err != nil || other.PolicyFingerprint() != ctx.PolicyFingerprint() || other.SourceFingerprint() != ctx.SourceFingerprint() {
		t.Fatal("program data became policy or artifact identity", err)
	}
	if slices.Equal(other.Arguments(), ctx.Arguments()) {
		t.Fatal("independent argument vectors were conflated")
	}
}

func TestExecutionContextRejectsUnapprovedOrSubstitutedMembers(t *testing.T) {
	for _, change := range []string{"operation", "set", "resolution", "graph", "dependency instead of primary", "different acquisition", "empty runtime", "runtime control", "empty digest", "absolute member", "escape", "unclean member", "windows member", "zero member", "command path", "command control"} {
		t.Run(change, func(t *testing.T) {
			data := executionContextFixture(t)
			switch change {
			case "operation":
				data.OperationID = domain.OperationID{}
			case "set":
				data.Set = domain.VerifiedSet{}
			case "resolution":
				data.Resolution = domain.DependencyResolution{}
			case "graph":
				nodes := data.Resolution.Graph().Nodes()
				nodes[1], _ = domain.NewLockedDependency(nodes[1].Node(), domain.DependencyTransitive, nodes[0].Artifact())
				graph, err := domain.NewLockedDependencyGraph(nodes, data.Resolution.Graph().Edges())
				if err != nil {
					t.Fatal(err)
				}
				data.Resolution, _ = domain.NewDependencyResolution(graph, data.Resolution.RuntimeIdentity(), data.Resolution.LockfileDigest())
			case "dependency instead of primary":
				data.EntryArtifact = data.Set.Inspected().Inspections()[1].Artifact()
			case "different acquisition":
				run, err := domain.NewRunID()
				if err != nil {
					t.Fatal(err)
				}
				declared, _ := data.EntryArtifact.DeclaredIntegrity()
				data.EntryArtifact, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(data.EntryArtifact.Identity(), data.EntryArtifact.Digest(), "intake:"+run.String()+":tarball", data.EntryArtifact.SizeBytes(), declared)
				if err != nil {
					t.Fatal(err)
				}
			case "empty runtime":
				data.RuntimeIdentity = ""
			case "runtime control":
				data.RuntimeIdentity = "runtime\nother"
			case "empty digest":
				data.EntryDigest = domain.ContentDigest{}
			case "absolute member":
				data.EntryPath = "/cli.js"
			case "escape":
				data.EntryPath = "../cli.js"
			case "unclean member":
				data.EntryPath = "bin/../cli.js"
			case "windows member":
				data.EntryPath = `C:\cli.js`
			case "zero member":
				data.EntryPath = ""
			case "command path":
				data.Command = "/bin/sh"
			case "command control":
				data.Command = "first\nother"
			}
			if ctx, err := domain.NewExecutionContext(data); err == nil || ctx.Valid() {
				t.Fatal("invalid or substituted context accepted")
			}
		})
	}
	if (domain.ExecutionContext{}).Valid() {
		t.Fatal("zero context accepted")
	}
}

func TestExecutionContextPolicyFingerprintTracksActualPolicyVersions(t *testing.T) {
	data := executionContextFixture(t)
	ctx, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "set-policy", 4, []string{"ALLOW"})
	data.Set, err = domain.NewVerifiedSet(data.Set.Inspected(), decision)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := domain.NewExecutionContext(data)
	if err != nil || changed.PolicyFingerprint() == ctx.PolicyFingerprint() || changed.SourceFingerprint() != ctx.SourceFingerprint() {
		t.Fatal("actual policy version was not bound independently", err)
	}
	inspections := data.Set.Inspected().Inspections()
	old := inspections[0]
	entryDecision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "fixture-policy", 9, []string{"ALLOW"})
	inspections[0], err = domain.NewDependencyInspection(old.Node(), old.RunID(), old.Artifact(), old.Checks(), old.Evidence(), entryDecision)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedDependencySet(data.Resolution.Graph(), inspections)
	if err != nil {
		t.Fatal(err)
	}
	data.Set, _ = domain.NewVerifiedSet(inspected, decision)
	entryChanged, err := domain.NewExecutionContext(data)
	if err != nil || entryChanged.PolicyFingerprint() == changed.PolicyFingerprint() {
		t.Fatal("entry Policy version was omitted", err)
	}
}

func TestExecutionArgumentPreflightAndContextShareExactBounds(t *testing.T) {
	for _, test := range []struct {
		name     string
		values   []string
		accepted bool
	}{
		{"empty vector", nil, true},
		{"empty values", []string{"", " "}, true},
		{"data delimiters", []string{"--", "$(data)", "line\nbreak"}, true},
		{"count boundary", make([]string, domain.MaximumExecutionArguments), true},
		{"too many", make([]string, domain.MaximumExecutionArguments+1), false},
		{"byte boundary", []string{strings.Repeat("x", domain.MaximumExecutionArgumentBytes)}, true},
		{"too many bytes", []string{strings.Repeat("x", domain.MaximumExecutionArgumentBytes), "x"}, false},
		{"NUL", []string{"a\x00b"}, false},
		{"invalid UTF8", []string{string([]byte{0xff})}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := domain.ValidateExecutionArguments(test.values); (err == nil) != test.accepted {
				t.Fatal("preflight boundary differs", err)
			}
			data := executionContextFixture(t)
			data.Arguments = test.values
			ctx, err := domain.NewExecutionContext(data)
			if (err == nil) != test.accepted || ctx.Valid() != test.accepted {
				t.Fatal("context boundary differs", err)
			}
		})
	}
}
