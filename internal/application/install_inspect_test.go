package application_test

import (
	"context"
	"errors"
	"reflect"
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
	set, decision, err := service.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !resolver.called || !set.Valid() || len(set.Inspections()) != 1 {
		t.Fatalf("resolver called=%v set=%#v", resolver.called, set)
	}
	if decision.Decision() != domain.DecisionAllow || !reflect.DeepEqual(decision.Reasons(), []string{"M4_VERIFIED_SET_COMPLETED"}) {
		t.Fatalf("set decision = %#v", decision)
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
	if _, _, err := service.Inspect(context.Background(), request); !errors.Is(err, resolver.err) {
		t.Fatalf("Inspect() error = %v, want resolver error", err)
	}
	if got := fake.Calls(); len(got) != 0 {
		t.Fatalf("entry inspection ran after resolution failure: %v", got)
	}
}

type lockedResolver struct {
	graph  domain.LockedDependencyGraph
	err    error
	called bool
}

func (r *lockedResolver) ResolveDependencies(_ context.Context, _ domain.ArtifactReference, _ domain.InstallContext) (domain.LockedDependencyGraph, error) {
	r.called = true
	if r.err != nil {
		return domain.LockedDependencyGraph{}, r.err
	}
	return r.graph, nil
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
