package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type ordinaryPreparationFixture struct {
	session ports.InstallSession
	calls   *[]string
	err     error
}

func (p ordinaryPreparationFixture) BeginInstall(context.Context, domain.ArtifactReference, domain.InstallContext) (ports.InstallSession, error) {
	*p.calls = append(*p.calls, "begin")
	return p.session, p.err
}

// No Reference method: ordinary preparation must not acquire update intent.
type ordinaryInstallSession struct {
	resolver ports.DependencyResolver
	promoter ports.Promotion
	calls    *[]string
	closeErr error
}

func (s *ordinaryInstallSession) ResolveDependencies(ctx context.Context, ref domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	*s.calls = append(*s.calls, "resolve")
	return s.resolver.ResolveDependencies(ctx, ref, install)
}
func (s *ordinaryInstallSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	return s.promoter.Promote(ctx, staged, bundle, install)
}
func (s *ordinaryInstallSession) Close() error {
	*s.calls = append(*s.calls, "close")
	return s.closeErr
}

func TestPreparedInstallKeepsOrdinaryIntentAndOriginalPipeline(t *testing.T) {
	request, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	session := &ordinaryInstallSession{resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls}
	if _, update := any(session).(ports.InstallUpdateSession); update {
		t.Fatal("ordinary guard became an update session")
	}
	service, err := application.NewPreparedInstallService(base, ordinaryPreparationFixture{session: session, calls: &calls})
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.Install(context.Background(), request)
	if err != nil || outcome.Promoted().Target() != request.Context().Target() || outcome.Inspection().Request().Operation() != domain.OperationInstall {
		t.Fatal(outcome, err)
	}
	if !reflect.DeepEqual(calls, []string{"begin", "resolve", "manifest", "staging", "promotion", "close"}) {
		t.Fatal(calls)
	}
	calls = nil
	if _, err := base.Install(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"manifest", "staging", "promotion"}) {
		t.Fatal("base was rebound", calls)
	}
}

func TestPreparedInstallPreservesPolicyAndPrimaryCleanupFailures(t *testing.T) {
	for _, decision := range []domain.Decision{domain.DecisionBlock, domain.DecisionManualReview} {
		request, inspection := installWorkflowFixtureWithPolicy(t, fixedSetPolicy{decision: decision})
		calls := []string{}
		base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
		session := &ordinaryInstallSession{resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls}
		service, _ := application.NewPreparedInstallService(base, ordinaryPreparationFixture{session: session, calls: &calls})
		outcome, err := service.Install(context.Background(), request)
		if err != nil || outcome.Inspection().Decision().Decision() != decision || !reflect.DeepEqual(calls, []string{"begin", "resolve", "close"}) {
			t.Fatal(outcome, err, calls)
		}
	}
	for _, stagedFailure := range []bool{false, true} {
		request, inspection := installWorkflowFixture(t)
		calls := []string{}
		primary, cleanup := errors.New("controlled staging failure"), errors.New("controlled close failure")
		staging := &stagingPort{calls: &calls}
		if stagedFailure {
			staging.err = primary
		}
		base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, staging, &promotionPort{calls: &calls})
		session := &ordinaryInstallSession{resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls, closeErr: cleanup}
		service, _ := application.NewPreparedInstallService(base, ordinaryPreparationFixture{session: session, calls: &calls})
		outcome, err := service.Install(context.Background(), request)
		if !errors.Is(err, cleanup) || outcome.Inspection().Decision().Decision() != domain.DecisionAllow {
			t.Fatal(outcome, err)
		}
		if stagedFailure {
			if !errors.Is(err, primary) || outcome.FailureCode() != "STAGING_FAILED" || outcome.Promoted().Target().String() != "" {
				t.Fatal(outcome, err)
			}
		} else if outcome.FailureCode() != "INSTALL_GUARD_CLOSE_FAILED" || outcome.Promoted().Target() != request.Context().Target() {
			t.Fatal(outcome, err)
		}
	}
}

func TestPreparedInstallRejectsBeforeSelectionAndPreservesStartedFailure(t *testing.T) {
	request, inspection := installWorkflowFixture(t)
	calls := []string{}
	base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
	if _, err := application.NewPreparedInstallService(nil, ordinaryPreparationFixture{}); err == nil {
		t.Fatal("nil pipeline accepted")
	}
	if _, err := application.NewPreparedInstallService(base, nil); err == nil {
		t.Fatal("nil preparation accepted")
	}
	for _, preparationErr := range []error{nil, errors.New("controlled preparation failure")} {
		calls = nil
		service, _ := application.NewPreparedInstallService(base, ordinaryPreparationFixture{calls: &calls, err: preparationErr})
		outcome, err := service.Install(context.Background(), request)
		if err == nil || outcome.FailureCode() != "INSTALL_PREPARATION_FAILED" || outcome.Inspection().OperationID().String() == "" || outcome.Inspection().Set().Valid() || outcome.Inspection().Request().Operation() != domain.OperationInstall || !reflect.DeepEqual(calls, []string{"begin"}) {
			t.Fatal(outcome, err, calls)
		}
		if preparationErr != nil && !errors.Is(err, preparationErr) {
			t.Fatal("cause lost", err)
		}
	}
	calls = nil
	service, _ := application.NewPreparedInstallService(base, ordinaryPreparationFixture{calls: &calls})
	install, _ := domain.NewNPMProjectInstallContext(request.Context().Target())
	update, _ := application.NewInstallUpdateRequest(request.Source(), "first", install, false)
	if _, err := service.Install(context.Background(), update); err == nil || len(calls) != 0 {
		t.Fatal("update accepted", err, calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Install(ctx, request); !errors.Is(err, context.Canceled) || len(calls) != 0 {
		t.Fatal("canceled preparation started", err, calls)
	}
}

func TestPreparedInstallTracksStorageBeforeBeginAndAfterActualClose(t *testing.T) {
	for _, scenario := range []string{"normal", "policy denied", "begin failure", "close failure"} {
		t.Run(scenario, func(t *testing.T) {
			decision := domain.DecisionAllow
			if scenario == "policy denied" {
				decision = domain.DecisionBlock
			}
			request, inspection := installWorkflowFixtureWithPolicy(t, fixedSetPolicy{decision: decision})
			calls := []string{}
			base, _ := application.NewInstallService(inspection, &manifestPort{calls: &calls}, &stagingPort{calls: &calls}, &promotionPort{calls: &calls})
			cause := errors.New("controlled lifecycle fault")
			session := &ordinaryInstallSession{resolver: &lockedResolver{graph: twoNodeGraph(t)}, promoter: &promotionPort{calls: &calls}, calls: &calls}
			preparation := ordinaryPreparationFixture{session: session, calls: &calls}
			if scenario == "begin failure" {
				preparation.err = cause
			}
			if scenario == "close failure" {
				session.closeErr = cause
			}
			service, _ := application.NewPreparedInstallService(base, preparation)
			tracker := &workflowStorageUse{calls: &calls}
			_, err := service.Install(ports.WithStorage(context.Background(), tracker), request)
			if len(calls) < 3 || calls[0] != "storage.begin" || calls[len(calls)-1] != "storage.end" {
				t.Fatal("storage lifetime did not enclose the guard", calls)
			}
			last := tracker.causes[len(tracker.causes)-1]
			if scenario == "begin failure" || scenario == "close failure" {
				if !errors.Is(err, cause) || !errors.Is(last, cause) {
					t.Fatal("primary/close error lost", err, last)
				}
			} else if err != nil || last != nil {
				t.Fatal("normal close or policy denial became unknown", err, last)
			}
			if scenario != "begin failure" && calls[len(calls)-2] != "close" {
				t.Fatal("storage released before actual guard close", calls)
			}
		})
	}
}

type workflowStorageUse struct {
	calls  *[]string
	causes []error
}

func (*workflowStorageUse) Policy() domain.StoragePolicy { return domain.DefaultStoragePolicy() }
func (*workflowStorageUse) Reserve(context.Context, ports.StorageRequest) (ports.StorageLease, error) {
	return nil, errors.New("unexpected fake-workflow storage write")
}
func (s *workflowStorageUse) TrackUse(context.Context) (func(error), error) {
	*s.calls = append(*s.calls, "storage.begin")
	return func(cause error) { *s.calls = append(*s.calls, "storage.end"); s.causes = append(s.causes, cause) }, nil
}
