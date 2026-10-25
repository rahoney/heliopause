package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

type bunProjectBindings struct {
	intakeRoot, evidenceRoot, cacheRoot, stateRoot string
	source                                         *artifactnpm.Resolver
	store                                          *local.Store
	static                                         *inspectionnpm.Inspector
	native                                         *sandbox.BunResolver
	prepare                                        *sandbox.BunInspectionPreparer
	backend                                        *sandbox.Backend
}

// One command owns one native selection and its exact frozen peer context.
// The source-neutral workflow never has to reconstruct that native state.
type bunProjectSession struct {
	bindings   bunProjectBindings
	intent     artifactnpm.BunSelectionIntent
	selector   string
	operation  domain.OperationID
	used       bool
	frozen     artifactnpm.BunInspectionContext
	inspection *application.ProjectInspectService
	cache      *promotion.BunVerifiedCache
}

type bunProjectMutationAdapter struct {
	adopt bool
	owner *promotion.BunProjectPromotion
}

type bunProjectService struct {
	workflow *application.BunProjectService
	promoter *promotion.BunProjectPromotion
}

func (s *bunProjectService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	return s.workflow.Apply(ctx, install)
}

func (a bunProjectMutationAdapter) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	return a.owner.BeginUpgrade(ctx, install, a.adopt)
}

func newBunProjectService(bindings bunProjectBindings, intent artifactnpm.BunSelectionIntent, selector string, adopt bool) (*bunProjectService, error) {
	if bindings.source == nil || bindings.store == nil || bindings.static == nil || bindings.native == nil || bindings.prepare == nil || bindings.backend == nil {
		return nil, errors.New("bun project requires the actual source, native runtime and inspection adapters")
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		return nil, err
	}
	s := &bunProjectSession{bindings: bindings, intent: intent, selector: selector, operation: operation}
	owner, err := promotion.NewBunProjectPromotionWithOriginalVerifier(bindings.intakeRoot, bindings.evidenceRoot, bindings.cacheRoot, bindings.stateRoot, bindings.store, operation, func(snapshot domain.ProjectDependencySnapshot) (*promotion.BunVerifiedCache, error) {
		if s.cache == nil || !s.frozen.MatchesSnapshot(snapshot) {
			return nil, errors.New("bun operation has no cache for this exact native selection")
		}
		return s.cache, nil
	}, bindings.proveBunOriginal)
	if err != nil {
		return nil, err
	}
	workflow, err := application.NewBunProjectService(s, bunProjectMutationAdapter{owner: owner, adopt: adopt}, s, s)
	if err != nil {
		return nil, err
	}
	return &bunProjectService{workflow, owner}, nil
}

func (s *bunProjectSession) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if s == nil || s.used || selector != "" {
		return domain.ProjectDependencyUpdate{}, errors.New("bun request permits exactly one configured native selection")
	}
	s.used = true
	s.frozen = artifactnpm.BunInspectionContext{}
	s.cache = nil
	s.inspection = nil
	input, err := artifactnpm.NewBunSelectionInput(s.intent, s.selector, controls, runtimeidentity.BunVersion)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	selected, err := s.bindings.native.SelectProject(ctx, input)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	frozen, err := selected.BindInspectionContext(ctx, s.bindings.source, install)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	inspector, err := inspectionnpm.NewBunProjectInspector(frozen, s.bindings.static, s.bindings.prepare, s.bindings.backend)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	quarantine, err := application.NewProjectInspectService(s.bindings.source, verificationnpm.IntegrityVerifier{}, inspector, s.bindings.store, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return s.operation, nil }, domain.NewRunID)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	cache, err := promotion.NewBunVerifiedCache(s.bindings.intakeRoot, s.bindings.evidenceRoot, s.bindings.cacheRoot, frozen, s.bindings.store, inspector, evidence.Generator{}, s.operation)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshotWithControlPresence(controls, selected.Controls(), frozen.Snapshot())
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	s.frozen = frozen
	s.inspection = quarantine
	s.cache = cache
	return update, nil
}

func (s *bunProjectSession) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot) (domain.ProjectVerifiedSet, error) {
	if s == nil || s.inspection == nil || !s.frozen.MatchesSnapshot(snapshot) {
		return domain.ProjectVerifiedSet{}, errors.New("bun inspection lacks the exact native context")
	}
	return s.inspection.InspectProject(ctx, snapshot)
}

func (s *bunProjectSession) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if s == nil || s.cache == nil || !set.Valid() || !s.frozen.MatchesSnapshot(set.Inspected().Snapshot()) {
		return domain.StagedProjectSet{}, errors.New("bun staging lacks the exact native approval")
	}
	return s.cache.StageProject(ctx, set)
}
