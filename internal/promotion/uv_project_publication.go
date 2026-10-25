package promotion

import (
	"context"
	"errors"
	"reflect"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type UVProjectPromotion struct {
	preparation    *UVProjectPreparation
	cache          *PythonProjectCache
	projectCache   *UVProjectCache
	guard          *uvProjectPublicationGuard
	stageAttempted bool
	staged         domain.StagedProjectSet
	material       pythonProjectMaterial
}

func NewUVProjectPromotion(preparation *UVProjectPreparation, cache *PythonProjectCache) (*UVProjectPromotion, error) {
	if preparation == nil || preparation.started || cache == nil || cache.storage == nil || preparation.promoter.stagingRoot != cache.storage.stagingRoot {
		return nil, errors.New("UV project publication requires one request and its actual trusted cache")
	}
	for _, root := range []string{cache.storage.intakeRoot, cache.storage.evidenceRoot, cache.storage.stagingRoot} {
		found := false
		for _, trusted := range preparation.trustedRoots {
			found = found || root == trusted
		}
		if !found {
			return nil, errors.New("UV project guard omits actual trusted storage")
		}
	}
	return &UVProjectPromotion{preparation: preparation, cache: cache}, nil
}

func (p *UVProjectPromotion) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if p == nil || p.guard != nil {
		return nil, errors.New("UV project publication request was already begun")
	}
	session, err := p.preparation.Begin(ctx, install)
	if err != nil {
		return nil, err
	}
	p.guard = &uvProjectPublicationGuard{uvProjectSession: session, publication: p}
	return p.guard, nil
}

func (p *UVProjectPromotion) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if p == nil || p.guard == nil || p.stageAttempted {
		return domain.StagedProjectSet{}, errors.New("UV project staging is unavailable or repeated")
	}
	update, err := p.preparation.SelectedUpdate(ctx)
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	p.stageAttempted = true
	var cache *UVProjectCache
	if p.guard.proof.Present() {
		cache, err = newUVProjectCacheWithOriginalProof(p.cache, p.guard.baseSelected, update, p.guard.proof)
	} else {
		cache, err = NewUVProjectCache(p.cache, update)
	}
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	staged, err := cache.StageProject(ctx, set)
	if err = errors.Join(err, p.guard.VerifyUnchanged(ctx)); err != nil {
		return domain.StagedProjectSet{}, err
	}
	p.projectCache, p.staged = cache, staged
	return staged, nil
}

type uvProjectPublicationGuard struct {
	*uvProjectSession
	publication     *UVProjectPromotion
	commitAttempted bool
}

func (*uvProjectPublicationGuard) CommitSnapshot(context.Context, domain.ProjectDependencySnapshot, domain.StagedProjectSet) error {
	return errors.New("UV project requires its exact approved native update")
}

func (g *uvProjectPublicationGuard) Commit(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) error {
	if g == nil || g.commitAttempted || !sameUVSyncUpdate(update, g.selected) || !staged.Valid() || !reflect.DeepEqual(staged, g.publication.staged) || g.publication.projectCache == nil {
		return errors.New("UV project publication differs from its single approved selection")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	g.commitAttempted = true
	p := g.publication
	root, material, err := p.projectCache.openProject(ctx, staged)
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
		controls := []domain.ProjectControlFile{}
		for _, c := range update.SelectedControls() {
			if c.Name() == artifactpypi.UVProjectInputControl || c.Name() == artifactpypi.UVNativeLockControl {
				controls = append(controls, c)
			}
		}
		if err := g.transaction.commitCompleteProject(desired, update.Snapshot().DependencyFree(), controls); err != nil {
			return err
		}
		g.committed = true
		p.material = material
		return nil
	})
}

func (p *UVProjectPromotion) PublishedRecords() (domain.ContentDigest, string, string, bool) {
	if p == nil || p.guard == nil || !p.guard.committed || p.material.manifest.String() == "" {
		return domain.ContentDigest{}, "", "", false
	}
	return p.material.manifest, p.staged.ContentHandle() + ":" + manifestFilename, p.staged.ContentHandle() + ":" + sbomFilename, true
}
