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
	verificationproject "github.com/rahoney/heliopause/internal/verification/projectbuild"
)

type npmProjectBindings struct {
	intakeRoot, evidenceRoot, cacheRoot, stateRoot string
	source                                         *artifactnpm.Resolver
	store                                          *local.Store
	static                                         *inspectionnpm.Inspector
	native                                         *sandbox.NPMResolver
	backend                                        *sandbox.Backend
}

// One request owns the frozen complete npm selection. The existing Application
// workflow retains all guard/inspection/Policy/staging/commit ordering.
type npmProjectService struct {
	bindings                  npmProjectBindings
	intent                    artifactnpm.NPMProjectIntent
	operation                 domain.OperationID
	used                      bool
	rootRequired, rootChecked bool
	rootSource                domain.AcquiredArtifact
	rootResult                domain.OperationResult
	rootDependencies          *inspectionnpm.NPMProjectInspector
	frozen                    artifactnpm.NPMProjectInspectionContext
	inspection                *application.ProjectInspectService
	cache                     *promotion.NPMProjectVerifiedCache
	promoter                  *promotion.PNPMProjectPromotion
	workflow                  *application.NPMCompleteProjectService
}

func newNPMProjectService(bindings npmProjectBindings, intent artifactnpm.NPMProjectIntent) (*npmProjectService, error) {
	if bindings.source == nil || bindings.store == nil || bindings.static == nil || bindings.native == nil || bindings.backend == nil || (intent != artifactnpm.NPMProjectInstall && intent != artifactnpm.NPMProjectCI) {
		return nil, errors.New("npm project requires actual source/runtime/inspection and a supported intent")
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		return nil, err
	}
	s := &npmProjectService{bindings: bindings, intent: intent, operation: operation}
	s.promoter, err = promotion.NewNPMProjectPromotion(bindings.intakeRoot, bindings.evidenceRoot, bindings.cacheRoot, bindings.stateRoot, bindings.store, operation, func(snapshot domain.ProjectDependencySnapshot) (*promotion.NPMProjectVerifiedCache, error) {
		if s.cache == nil || !s.frozen.MatchesSnapshot(snapshot) {
			return nil, errors.New("npm operation has no cache for this exact selection")
		}
		return s.cache, nil
	})
	if err != nil {
		return nil, err
	}
	s.workflow, err = application.NewNPMCompleteProjectService("npm", s, s, s, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *npmProjectService) Apply(ctx context.Context, install domain.InstallContext) (domain.ProjectVerifiedSet, error) {
	return s.workflow.Apply(ctx, install)
}

func (s *npmProjectService) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	// This permits edited controls only with independently owned old payload and
	// approval. This constructor grants no unmanaged native-adoption verifier.
	if s == nil || s.used {
		return nil, errors.New("npm request permits one guarded source selection")
	}
	guard, err := s.promoter.BeginUpgrade(ctx, install, true)
	if err != nil {
		return nil, err
	}
	required, err := guard.NPMProjectRootLifecycleRequired(ctx)
	if err == nil {
		s.rootChecked = true
		s.rootRequired = required
	}
	if err == nil && required {
		s.rootSource, err = guard.FreezeNPMProjectSource(ctx)
	}
	if err != nil {
		return nil, errors.Join(err, guard.Close())
	}
	s.rootRequired = required
	return guard, nil
}

func (s *npmProjectService) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, controls []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if s == nil || s.used || selector != "" {
		return domain.ProjectDependencyUpdate{}, errors.New("npm request permits exactly one configured whole selection")
	}
	s.used = true
	s.frozen = artifactnpm.NPMProjectInspectionContext{}
	s.cache = nil
	s.inspection = nil
	input, err := artifactnpm.NewNPMProjectInput(s.intent, controls, runtimeidentity.NodeNPMVersion)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	selected, err := s.bindings.native.SelectProject(ctx, input)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	bound, err := s.bindings.source.BindNPMProjectLock(ctx, selected.Selection())
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	frozen, err := bound.InspectionContext(install, selected.Controls(), sandbox.NPMProjectRuntimeIdentity())
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	inspector, err := inspectionnpm.NewNPMProjectInspector(frozen, s.bindings.static, s.bindings.backend, s.bindings.backend)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	quarantine, err := application.NewProjectInspectService(s.bindings.source, verificationnpm.IntegrityVerifier{}, inspector, s.bindings.store, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return s.operation, nil }, domain.NewRunID)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	cache, err := promotion.NewNPMProjectVerifiedCache(s.bindings.intakeRoot, s.bindings.evidenceRoot, s.bindings.cacheRoot, frozen, s.bindings.store, inspector, evidence.Generator{}, s.operation)
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot(controls, selected.Controls(), frozen.Snapshot())
	if err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	s.frozen, s.inspection, s.cache = frozen, quarantine, cache
	s.rootDependencies = inspector
	return update, nil
}

func (s *npmProjectService) InspectProject(ctx context.Context, snapshot domain.ProjectDependencySnapshot) (domain.ProjectVerifiedSet, error) {
	if s == nil || s.inspection == nil || !s.frozen.MatchesSnapshot(snapshot) {
		return domain.ProjectVerifiedSet{}, errors.New("npm inspection lacks exact native context")
	}
	verified, err := s.inspection.InspectProject(ctx, snapshot)
	if err != nil || !s.rootRequired {
		return verified, err
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, entry := range verified.Inspected().Inspections() {
		acquired[entry.Node()] = entry.Artifact()
	}
	dependencies, err := s.frozen.BindAcquired(snapshot, acquired)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	var layout artifactnpm.NPMProjectInspectionLayout
	if !snapshot.DependencyFree() {
		layout, err = s.rootDependencies.PreparedLayoutFor(snapshot, acquired)
		if err != nil {
			return domain.ProjectVerifiedSet{}, err
		}
	}
	input, err := artifactnpm.NewNPMProjectRootInspectionInput(dependencies, layout, s.rootSource)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	source, err := artifactnpm.NewNPMProjectRootArtifact(s.rootSource)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	reader, err := sandbox.NewNPMProjectSourceReader(s.bindings.intakeRoot)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	verifier, err := verificationproject.NewSourceVerifier(reader, "npm")
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	var executionErr error
	inspector, err := inspectionnpm.NewNPMProjectRootInspector(input, reader, func(run context.Context, current artifactnpm.NPMProjectRootInspectionInput) (domain.SandboxResult, error) {
		var observed domain.SandboxResult
		observed, executionErr = s.bindings.backend.ExecuteNPMProjectRoot(run, current)
		return observed, executionErr
	})
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	inspect, err := application.NewInspectService(source, verifier, inspector, s.bindings.store, policy.NPMProjectRoot{}, func() (domain.OperationID, error) { return s.operation, nil }, domain.NewRunID)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	reference, err := domain.NewArtifactReference(s.rootSource.Identity().Source(), s.rootSource.Identity().Name())
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	request, err := application.NewInspectRequest(reference)
	if err != nil {
		return domain.ProjectVerifiedSet{}, err
	}
	s.rootResult, err = inspect.Inspect(ctx, request)
	decision, known := s.rootResult.PolicyDecision()
	if err != nil || executionErr != nil || s.rootResult.Status() != domain.OperationCompleted || !known || decision.Decision() != domain.DecisionAllow {
		return domain.ProjectVerifiedSet{}, &npmRootInspectionFailure{verified.Inspected(), decision, errors.Join(errors.New("local project lifecycle did not permit installation"), err, executionErr)}
	}
	return verified, nil
}

type npmRootInspectionFailure struct {
	inspected domain.InspectedProjectSet
	decision  domain.PolicyDecision
	cause     error
}

func (e *npmRootInspectionFailure) Error() string                         { return e.cause.Error() }
func (e *npmRootInspectionFailure) Unwrap() error                         { return e.cause }
func (e *npmRootInspectionFailure) Inspected() domain.InspectedProjectSet { return e.inspected }
func (e *npmRootInspectionFailure) Decision() domain.PolicyDecision       { return e.decision }

func (s *npmProjectService) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if s == nil || s.cache == nil || !set.Valid() || !s.frozen.MatchesSnapshot(set.Inspected().Snapshot()) {
		return domain.StagedProjectSet{}, errors.New("npm staging lacks exact current approval")
	}
	return s.cache.StageProject(ctx, set)
}
