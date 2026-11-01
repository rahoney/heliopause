package promotion

import (
	"context"
	"errors"
	"reflect"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// PythonRequirementsPreparation keeps one source closure, original environment
// and additive selection under the existing Python transaction guards.
type PythonRequirementsPreparation struct {
	promoter     *PyPIPromotion
	resolver     ports.ProjectLockUpdateResolver
	request      artifactpypi.RequirementsRequest
	trustedRoots []string
	native       *pypiNativeEnrollment
	started      bool
	active       *pythonProjectSession
}

func NewPythonRequirementsPreparation(promoter *PyPIPromotion, resolver ports.ProjectLockUpdateResolver, request artifactpypi.RequirementsRequest, trustedRoots []string) (*PythonRequirementsPreparation, error) {
	if promoter == nil || resolver == nil || request.Client != "pip" && request.Client != "uv" || len(request.Sources)+len(request.Constraints)+len(request.Positional) == 0 || len(request.Sources)+len(request.Constraints) > artifactpypi.MaximumRequirementsFiles || len(request.Positional) > artifactpypi.RequirementsClauseLimit || len(trustedRoots) == 0 || len(trustedRoots) > 32 {
		return nil, errors.New("requirements installation needs one bounded request, guarded resolver and storage")
	}
	found := false
	for _, root := range trustedRoots {
		found = found || root == promoter.stagingRoot
	}
	if !found {
		return nil, errors.New("requirements source guard omits actual staging")
	}
	request.Sources = append([]string(nil), request.Sources...)
	request.Constraints = append([]string(nil), request.Constraints...)
	request.Positional = append([]string(nil), request.Positional...)
	return &PythonRequirementsPreparation{promoter: promoter, resolver: resolver, request: request, trustedRoots: append([]string(nil), trustedRoots...)}, nil
}

func (p *PythonRequirementsPreparation) WithNativeEnrollment(resolver ports.DependencyResolver, intake ports.Artifact, intakeRoot string) (*PythonRequirementsPreparation, error) {
	if p == nil || p.started || resolver == nil || intake == nil {
		return nil, errors.New("requirements original enrollment is unavailable")
	}
	found := false
	for _, root := range p.trustedRoots {
		found = found || root == intakeRoot
	}
	if !found {
		return nil, errors.New("requirements original intake is outside trusted storage")
	}
	client := nativePipClient
	if p.request.Client == "uv" {
		client = nativeUVClient
	}
	copy := *p
	copy.native = &pypiNativeEnrollment{resolver: resolver, intake: intake, intakeRoot: intakeRoot, runner: p.promoter.runner, client: client}
	return &copy, nil
}

func (p *PythonRequirementsPreparation) Begin(ctx context.Context, install domain.InstallContext) (result *pythonProjectSession, resultErr error) {
	if p == nil || p.started || ctx == nil || !install.Valid() || install.Mode() != domain.InstallPythonVenv || p.promoter.goos != "linux" || p.promoter.goarch != "amd64" || strings.Contains(install.Target().String(), ",") {
		return nil, errors.New("requirements installation needs one existing Linux amd64 venv transaction")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.started = true
	discover := discoverPythonVenv
	if p.request.Client == "uv" {
		discover = discoverUVPythonVenv
	}
	plan, err := discover(install.Target().String())
	if err != nil {
		return nil, err
	}
	tx, err := beginPyPIVenvTransactionWithDiscovery(plan, discover)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	s := &pythonProjectSession{install: install, transaction: tx}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, s.Close())
			result = nil
		}
	}()
	input, err := freezePythonRequirements(ctx, tx, p.request.Sources, p.request.Constraints, p.request.Positional, p.trustedRoots)
	if err != nil {
		return nil, err
	}
	s.original, err = capturePyPIInstalledSnapshot(tx)
	if err != nil {
		return nil, err
	}
	s.proof, err = preparePythonCompleteOwnership(ctx, tx, s.original, install, p.native, p.request.Adopt, s.VerifyUnchanged)
	if err != nil {
		return nil, err
	}
	input, err = input.WithInstallation(p.request.Client, p.request.Upgrade, s.original.metadataFiles())
	if err != nil {
		return nil, err
	}
	if s.proof.Present() {
		input, err = input.WithOwnershipProof(s.proof.Body())
		if err != nil {
			return nil, err
		}
	}
	if p.request.RequireHashes {
		input, err = input.WithRequiredHashes()
		if err != nil {
			return nil, err
		}
	}
	s.baseControls, err = input.Controls()
	if err != nil {
		return nil, err
	}
	s.controls = append([]domain.ProjectControlFile(nil), s.baseControls...)
	if err := s.VerifyUnchanged(ctx); err != nil {
		return nil, err
	}
	p.active = s
	return s, nil
}

func (p *PythonRequirementsPreparation) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.active == nil {
		return domain.ProjectDependencyUpdate{}, errors.New("requirements selection needs its original held guard")
	}
	s := p.active
	if s.attempted || selector != "" || install != s.install || !sameUVSyncControls(original, s.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("requirements selection changed or was repeated")
	}
	if err := s.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	s.attempted = true
	update, err := p.resolver.ResolveProjectLockUpdate(ctx, install, s.controls, "")
	if err = errors.Join(err, s.VerifyUnchanged(ctx)); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	runtime := sandbox.PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	if _, err := artifactpypi.ParseRequirementsUpdate(update, target); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	if update.Snapshot().Context() != s.install || !sameUVSyncControls(update.OriginalControls(), s.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("requirements selected another original environment")
	}
	s.selected = update
	return update, nil
}

func (p *PythonRequirementsPreparation) SelectedUpdate(ctx context.Context) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.active == nil || !p.active.selected.Valid() {
		return domain.ProjectDependencyUpdate{}, errors.New("requirements selection is unavailable")
	}
	if err := p.active.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	return p.active.selected, nil
}

type PythonRequirementsPromotion struct {
	preparation    *PythonRequirementsPreparation
	cache          *PythonProjectCache
	guard          *pythonRequirementsGuard
	stageAttempted bool
	staged         domain.StagedProjectSet
	material       pythonProjectMaterial
}

func NewPythonRequirementsPromotion(preparation *PythonRequirementsPreparation, cache *PythonProjectCache) (*PythonRequirementsPromotion, error) {
	if preparation == nil || preparation.started || cache == nil || cache.storage == nil || preparation.promoter.stagingRoot != cache.storage.stagingRoot {
		return nil, errors.New("requirements publication needs the same request and trusted cache")
	}
	for _, root := range []string{cache.storage.intakeRoot, cache.storage.evidenceRoot, cache.storage.stagingRoot} {
		found := false
		for _, trusted := range preparation.trustedRoots {
			found = found || trusted == root
		}
		if !found {
			return nil, errors.New("requirements guard omits actual trusted storage")
		}
	}
	return &PythonRequirementsPromotion{preparation: preparation, cache: cache}, nil
}

func (p *PythonRequirementsPromotion) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if p == nil || p.guard != nil {
		return nil, errors.New("requirements publication request was already begun")
	}
	session, err := p.preparation.Begin(ctx, install)
	if err != nil {
		return nil, err
	}
	p.guard = &pythonRequirementsGuard{pythonProjectSession: session, publication: p}
	return p.guard, nil
}

func (p *PythonRequirementsPromotion) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if p == nil || p.guard == nil || p.stageAttempted {
		return domain.StagedProjectSet{}, errors.New("requirements staging is unavailable or repeated")
	}
	update, err := p.preparation.SelectedUpdate(ctx)
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	if update.Snapshot().DependencyFree() {
		return domain.StagedProjectSet{}, errors.New("empty requirements are a no-op and cannot be staged")
	}
	p.stageAttempted = true
	staged, err := p.cache.stageProject(ctx, set, update)
	if err = errors.Join(err, p.guard.VerifyUnchanged(ctx)); err != nil {
		return domain.StagedProjectSet{}, err
	}
	p.staged = staged
	return staged, nil
}

type pythonRequirementsGuard struct {
	*pythonProjectSession
	publication     *PythonRequirementsPromotion
	commitAttempted bool
}

func (*pythonRequirementsGuard) CommitSnapshot(context.Context, domain.ProjectDependencySnapshot, domain.StagedProjectSet) error {
	return errors.New("requirements publication needs its single approved update")
}

func (g *pythonRequirementsGuard) Commit(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) error {
	if g == nil || g.commitAttempted || !sameUVSyncUpdate(update, g.selected) || update.Snapshot().DependencyFree() || !staged.Valid() || !reflect.DeepEqual(staged, g.publication.staged) {
		return errors.New("requirements publication differs from its approved additive selection")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	g.commitAttempted = true
	p := g.publication
	root, material, err := p.cache.openProject(ctx, staged, update)
	if err != nil {
		return err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	return withApprovedPythonProjectOutput(ctx, p.preparation.promoter, g.transaction, root, material, staged, func(desired []pypiDestination) error {
		if err := g.VerifyUnchanged(ctx); err != nil {
			return err
		}
		if err := g.transaction.commit(desired); err != nil {
			return err
		}
		g.committed = true
		p.material = material
		return nil
	})
}

func (p *PythonRequirementsPromotion) PublishedRecords() (domain.ContentDigest, string, string, bool) {
	if p == nil || p.guard == nil || !p.guard.committed || p.material.manifest.String() == "" {
		return domain.ContentDigest{}, "", "", false
	}
	return p.material.manifest, p.staged.ContentHandle() + ":" + manifestFilename, p.staged.ContentHandle() + ":" + sbomFilename, true
}
