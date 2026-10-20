package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
)

type updatePreparationFixture struct {
	session ports.InstallUpdateSession
	calls   *[]string
	err     error
}

func (p updatePreparationFixture) BeginInstallUpdate(_ context.Context, _ domain.SourceID, _ string, _ domain.InstallContext, _ bool) (ports.InstallUpdateSession, error) {
	*p.calls = append(*p.calls, "begin")
	return p.session, p.err
}

type updateSessionFixture struct {
	reference domain.ArtifactReference
	resolver  ports.DependencyResolver
	promoter  ports.Promotion
	calls     *[]string
	closeErr  error
}

func (s *updateSessionFixture) Reference() domain.ArtifactReference { return s.reference }
func (s *updateSessionFixture) ResolveDependencies(ctx context.Context, ref domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	*s.calls = append(*s.calls, "resolve")
	return s.resolver.ResolveDependencies(ctx, ref, install)
}
func (s *updateSessionFixture) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	return s.promoter.Promote(ctx, staged, bundle, install)
}
func (s *updateSessionFixture) Close() error { *s.calls = append(*s.calls, "close"); return s.closeErr }

func TestInstallUpdateGuardsBeforeSelectionAndReusesPipelineWithoutRebindingBase(t *testing.T) {
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, err := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	if err != nil {
		t.Fatal(err)
	}
	session := &updateSessionFixture{reference: old.Reference(), resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls}
	updater, err := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
	if err != nil {
		t.Fatal(err)
	}
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, err := application.NewInstallUpdateRequest(old.Source(), "", install, false)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := updater.Install(context.Background(), request)
	if err != nil || outcome.Promoted().Target() != install.Target() || outcome.Inspection().Request().Operation() != domain.OperationUpdate {
		t.Fatal(outcome, err)
	}
	if !reflect.DeepEqual(calls, []string{"begin", "resolve", "manifest", "staging", "promotion", "close"}) {
		t.Fatal(calls)
	}
	calls = nil
	if _, err := base.Install(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"manifest", "staging", "promotion"}) || old.Operation() != domain.OperationInstall {
		t.Fatal("base installer mutated by update", calls)
	}
}

func TestInstallUpdateRejectsBeforePublicationAndPreservesPrimaryAndCloseFailure(t *testing.T) {
	for _, decision := range []domain.Decision{domain.DecisionBlock, domain.DecisionManualReview} {
		old, inspection := installWorkflowFixtureWithPolicy(t, fixedSetPolicy{decision: decision})
		calls := []string{}
		base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
		session := &updateSessionFixture{reference: old.Reference(), resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls}
		updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
		install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
		request, _ := application.NewInstallUpdateRequest(old.Source(), "first", install, false)
		outcome, err := updater.Install(context.Background(), request)
		if err != nil || outcome.Inspection().Decision().Decision() != decision || !reflect.DeepEqual(calls, []string{"begin", "resolve", "close"}) {
			t.Fatal(calls, outcome, err)
		}
	}
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	primary, cleanup := errors.New("staging failed"), errors.New("guard close failed")
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls, err: primary}, &promotionPort{calls: &calls})
	session := &updateSessionFixture{reference: old.Reference(), resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls, closeErr: cleanup}
	updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, _ := application.NewInstallUpdateRequest(old.Source(), "first", install, false)
	outcome, err := updater.Install(context.Background(), request)
	if !errors.Is(err, primary) || !errors.Is(err, cleanup) || outcome.FailureCode() != "STAGING_FAILED" || outcome.Inspection().Decision().Decision() != domain.DecisionAllow || !reflect.DeepEqual(calls, []string{"begin", "resolve", "manifest", "staging", "close"}) {
		t.Fatal(outcome, calls, err)
	}
}

func TestInstallUpdatePreparationFailureIsStartedUpdateWithoutFabricatedGraph(t *testing.T) {
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	want := errors.New("original state unavailable")
	updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{calls: &calls, err: want})
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, _ := application.NewInstallUpdateRequest(old.Source(), "", install, false)
	outcome, err := updater.Install(context.Background(), request)
	if !errors.Is(err, want) || outcome.Inspection().OperationID().String() == "" || outcome.Inspection().Request().Operation() != domain.OperationUpdate || outcome.Inspection().Request().Reference().Locator() != "*" || outcome.Inspection().Set().Valid() || !reflect.DeepEqual(calls, []string{"begin"}) {
		t.Fatal(outcome, calls, err)
	}
}

func TestBaseInstallerCannotAcceptAnUpdateWithoutOriginalStateSession(t *testing.T) {
	old, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, _ := application.NewInstallUpdateRequest(old.Source(), "first", install, false)
	if _, err := base.Install(context.Background(), request); err == nil || len(calls) != 0 {
		t.Fatal("update bypassed early guard", calls, err)
	}
}

type updateFindingPorts struct {
	*multiInspectionPorts
	badName     string
	mode        string
	findingCode string
}

func (p *updateFindingPorts) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	report, err := p.multiInspectionPorts.Inspect(ctx, artifact)
	if err != nil || artifact.Identity().Name() != p.badName {
		return report, err
	}
	check := report.Execution()
	if p.mode == "inspection-incomplete" {
		check, _ = domain.NewCheckExecution(check.ID(), check.Kind(), true, domain.CapabilitySupported, domain.ExecutionIncomplete, "CONTROLLED_UPDATE_FIXTURE")
		return domain.NewInspectionReport(check, nil, report.Evidence())
	}
	code := p.findingCode
	if code == "" {
		code = "M3_HONEYTOKEN_ACCESS"
	}
	finding, _ := domain.NewFinding(code, []domain.EvidenceID{report.Evidence()[0].ID()})
	return domain.NewInspectionReport(check, []domain.Finding{finding}, report.Evidence())
}

func (p *updateFindingPorts) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	report, err := p.multiInspectionPorts.Verify(ctx, artifact)
	if err != nil || artifact.Identity().Name() != p.badName || p.mode != "integrity-mismatch" {
		return report, err
	}
	return domain.NewVerificationReport(report.Execution(), domain.VerificationMismatch, report.Evidence())
}

func TestInstallUpdateInspectsNewPrimaryAndTransitiveBytesBeforePolicy(t *testing.T) {
	for _, ecosystem := range []string{"npm", "pypi"} {
		t.Run(ecosystem, func(t *testing.T) {
			for _, mode := range []string{"primary-v2", "transitive-only-v2", "inspection-incomplete", "integrity-mismatch"} {
				t.Run(mode, func(t *testing.T) {
					old := twoNodeGraph(t)
					badName := "first"
					if mode == "transitive-only-v2" {
						badName = "second"
					}
					nodes := old.Nodes()
					for i, node := range nodes {
						identity := node.Artifact().Identity()
						version := identity.Version()
						if identity.Name() == badName {
							version = "2.0.0"
						}
						source, _ := domain.NewSourceID(ecosystem)
						variant := identity.Variant()
						if ecosystem == "pypi" {
							variant = "wheel"
						}
						identity, _ = domain.NewResolvedArtifactIdentity(source, identity.Name(), version, variant)
						artifact, _ := domain.NewResolvedArtifact(identity, strings.ReplaceAll(node.Artifact().AcquisitionLocator(), "1.0.0", version), "sha512-new-version")
						nodes[i], _ = domain.NewLockedDependency(node.Node(), node.Role(), artifact)
					}
					graph, err := domain.NewLockedDependencyGraph(nodes, old.Edges())
					if err != nil {
						t.Fatal(err)
					}
					fixtures := &updateFindingPorts{multiInspectionPorts: newMultiInspectionPorts(t, graph), badName: badName, mode: mode}
					var setPolicy application.DependencySetPolicy = policy.M4{}
					if ecosystem == "pypi" {
						setPolicy = policy.M5{}
					}
					inspection, err := application.NewInstallInspectService(&lockedResolver{graph: graph}, fixtures, fixtures, fixtures, fixtures, policy.M3{}, setPolicy, domain.NewOperationID, domain.NewRunID)
					if err != nil {
						t.Fatal(err)
					}
					calls := []string{}
					base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
					source, _ := domain.NewSourceID(ecosystem)
					reference, _ := domain.NewArtifactReference(source, "first")
					target, _ := domain.NewInstallTarget("/tmp/guarded-" + ecosystem + "-update")
					install, _ := domain.NewNPMProjectInstallContext(target)
					if ecosystem == "pypi" {
						install, _ = domain.NewPythonVenvInstallContext(target)
					}
					session := &updateSessionFixture{reference: reference, resolver: &lockedResolver{graph: graph}, promoter: &promotionPort{calls: &calls}, calls: &calls}
					updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
					request, _ := application.NewInstallUpdateRequest(source, "", install, false)
					outcome, err := updater.Install(context.Background(), request)
					want := domain.DecisionBlock
					if mode == "inspection-incomplete" {
						want = domain.DecisionManualReview
					}
					if err != nil || outcome.Inspection().Decision().Decision() != want || !reflect.DeepEqual(calls, []string{"begin", "resolve", "close"}) {
						t.Fatal("new unsafe/incomplete selection reached publication", outcome, calls, err)
					}
					for _, name := range []string{"first", "second"} {
						for _, phase := range []string{"acquire:", "verify:", "inspect:", "evidence:"} {
							if !containsCall(fixtures.calls, phase+name) {
								t.Fatal("selected dependency skipped", phase, name, fixtures.calls)
							}
						}
					}
					for _, item := range outcome.Inspection().Set().Inspections() {
						if item.Artifact().Identity().Name() == badName && item.Artifact().Identity().Version() != "2.0.0" {
							t.Fatal("stale version evidence reused")
						}
					}
				})
			}
		})
	}
}

// These inert fixtures exercise M6's actual policies in the existing update
// orchestrator. Filesystem ownership/rollback and real public bytes are tested
// independently by the promotion and installed-consumer suites.
type gitHubUpdatePolicyPorts struct {
	*updateFindingPorts
	recorded map[domain.RunID][]domain.Evidence
}

func (p *gitHubUpdatePolicyPorts) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	artifact, err := p.multiInspectionPorts.Acquire(ctx, run, resolved)
	if err != nil {
		return artifact, err
	}
	digest, err := domain.NewSHA256Digest(strings.TrimPrefix(resolved.DeclaredIntegrity(), "sha256:"))
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(artifact.Identity(), digest, artifact.ContentHandle(), artifact.SizeBytes(), resolved.DeclaredIntegrity())
}

func (p *gitHubUpdatePolicyPorts) Record(ctx context.Context, run domain.RunID, evidence []domain.Evidence) ([]domain.EvidenceReference, error) {
	if _, err := p.multiInspectionPorts.Record(ctx, run, evidence); err != nil {
		return nil, err
	}
	p.recorded[run] = append([]domain.Evidence(nil), evidence...)
	references := make([]domain.EvidenceReference, 0, len(evidence))
	for _, item := range evidence {
		ref, err := domain.NewEvidenceReference(item.ID(), "fixture:"+run.String()+":"+item.ID().String())
		if err != nil {
			return nil, err
		}
		references = append(references, ref)
	}
	return references, nil
}

func (p *gitHubUpdatePolicyPorts) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if p.mode == "integrity-mismatch" {
		return p.multiInspectionPorts.Inspect(ctx, artifact)
	}
	return p.updateFindingPorts.Inspect(ctx, artifact)
}

func TestGitHubUpdateRechecksUnsafeIncompleteAndMismatchedV2WithM6(t *testing.T) {
	for _, mode := range []string{"unsafe-v2", "inspection-incomplete", "integrity-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			source, _ := domain.NewSourceID("github-release")
			graph := func(version, digestByte string) domain.LockedDependencyGraph {
				identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", version, "tool.zip")
				resolved, _ := domain.NewResolvedArtifact(identity, "fixture:owner/repo@"+version+"#tool.zip", "sha256:"+strings.Repeat(digestByte, 64))
				id, _ := domain.NewDependencyNodeID("github-release-primary")
				node, _ := domain.NewLockedDependency(id, domain.DependencyPrimary, resolved)
				set, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
				if err != nil {
					t.Fatal(err)
				}
				return set
			}
			old, next := graph("v1", "a"), graph("v2", "b")
			fixtures := &gitHubUpdatePolicyPorts{updateFindingPorts: &updateFindingPorts{multiInspectionPorts: newMultiInspectionPorts(t, old)}, recorded: make(map[domain.RunID][]domain.Evidence)}
			fixtures.artifacts[next.Nodes()[0].Artifact().Identity()] = next.Nodes()[0].Artifact()
			inspection, err := application.NewInstallInspectService(&lockedResolver{graph: old}, fixtures, fixtures, fixtures, fixtures, policy.M6{}, policy.M6{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			calls := []string{}
			base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
			target, _ := domain.NewInstallTarget("/tmp/guarded-github-update")
			initial, _ := domain.NewInstallContext(target)
			reference, _ := domain.NewArtifactReference(source, "owner/repo@v1#tool.zip")
			request, _ := application.NewInstallRequest(reference, initial)
			seed, err := base.Install(context.Background(), request)
			if err != nil || seed.Inspection().Decision().Decision() != domain.DecisionAllow || !reflect.DeepEqual(calls, []string{"manifest", "staging", "promotion"}) {
				t.Fatal("allowed v1 baseline missing", seed, calls, err)
			}
			previous := seed.Inspection().Set().Inspections()[0]
			calls, fixtures.calls = nil, nil
			fixtures.badName, fixtures.mode, fixtures.findingCode = "owner-repo", mode, "M6_ARCHIVE_INVALID"
			reference, _ = domain.NewArtifactReference(source, "owner/repo@v2#tool.zip")
			session := &updateSessionFixture{reference: reference, resolver: &lockedResolver{graph: next}, promoter: &promotionPort{calls: &calls}, calls: &calls}
			updater, _ := application.NewInstallUpdateService(base, updatePreparationFixture{session: session, calls: &calls})
			managed, _ := domain.NewGitHubAssetInstallContext(target)
			request, _ = application.NewInstallUpdateRequest(source, reference.Locator(), managed, false)
			outcome, err := updater.Install(context.Background(), request)
			want, reason := domain.DecisionBlock, "M6_ARCHIVE_INVALID"
			if mode == "inspection-incomplete" {
				want, reason = domain.DecisionManualReview, "M6_REQUIRED_CHECK_INCOMPLETE"
			} else if mode == "integrity-mismatch" {
				reason = "M6_INTEGRITY_MISMATCH"
			}
			if err != nil || outcome.Inspection().Request().Operation() != domain.OperationUpdate || outcome.Inspection().Decision().Decision() != want || !reflect.DeepEqual(calls, []string{"begin", "resolve", "close"}) || outcome.Promoted().Target().String() != "" {
				t.Fatal("unapproved v2 reached publication", outcome, calls, err)
			}
			current := outcome.Inspection().Set().Inspections()[0]
			if current.PolicyDecision().PolicyID() != "m6-github-release" || !reflect.DeepEqual(current.PolicyDecision().Reasons(), []string{reason}) || current.RunID() == previous.RunID() || current.Artifact().Identity() != next.Nodes()[0].Artifact().Identity() || current.Artifact().Digest() == previous.Artifact().Digest() || reflect.DeepEqual(current.Evidence(), previous.Evidence()) {
				t.Fatal("new M6 subject/policy/evidence missing", previous, current)
			}
			if !reflect.DeepEqual(fixtures.calls, []string{"acquire:owner-repo", "verify:owner-repo", "inspect:owner-repo", "evidence:owner-repo"}) || len(fixtures.recorded) != 2 {
				t.Fatal("fresh inspection skipped", fixtures.calls, fixtures.recorded)
			}
			for _, evidence := range fixtures.recorded[current.RunID()] {
				if evidence.Identity() != current.Artifact().Identity() || evidence.Digest() != current.Artifact().Digest() {
					t.Fatal("stale Evidence subject", evidence)
				}
			}
		})
	}
}
