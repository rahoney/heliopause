package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// UVSyncProjectPromotion binds the request's one selection, approved storage
// and existing complete transaction. Native selection is never repeated here.
type UVSyncProjectPromotion struct {
	preparation    *UVSyncPreparation
	cache          *PythonProjectCache
	guard          *uvSyncPublicationGuard
	stageAttempted bool
	staged         domain.StagedProjectSet
	material       pythonProjectMaterial
}

func NewUVSyncProjectPromotion(preparation *UVSyncPreparation, cache *PythonProjectCache) (*UVSyncProjectPromotion, error) {
	if preparation == nil || preparation.started || cache == nil || preparation.promoter.stagingRoot != cache.storage.stagingRoot {
		return nil, errors.New("UV sync publication requires the same request and trusted staging")
	}
	for _, root := range []string{cache.storage.intakeRoot, cache.storage.evidenceRoot, cache.storage.stagingRoot} {
		found := false
		for _, trusted := range preparation.trustedRoots {
			found = found || root == trusted
		}
		if !found {
			return nil, errors.New("UV sync source guard omits an actual trusted storage root")
		}
	}
	return &UVSyncProjectPromotion{preparation: preparation, cache: cache}, nil
}

func (p *UVSyncProjectPromotion) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if p == nil || p.guard != nil {
		return nil, errors.New("UV sync publication request was already begun")
	}
	session, err := p.preparation.Begin(ctx, install)
	if err != nil {
		return nil, err
	}
	p.guard = &uvSyncPublicationGuard{uvSyncSession: session, publication: p}
	return p.guard, nil
}

func (p *UVSyncProjectPromotion) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if p == nil || p.guard == nil || p.stageAttempted {
		return domain.StagedProjectSet{}, errors.New("UV sync staging is unavailable or was repeated")
	}
	update, err := p.preparation.SelectedUpdate(ctx)
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	p.stageAttempted = true
	staged, err := p.cache.stageProject(ctx, set, update)
	if err = errors.Join(err, p.guard.VerifyUnchanged(ctx)); err != nil {
		return domain.StagedProjectSet{}, err
	}
	p.staged = staged
	return staged, nil
}

type uvSyncPublicationGuard struct {
	*uvSyncSession
	publication     *UVSyncProjectPromotion
	commitAttempted bool
}

func (*uvSyncPublicationGuard) CommitSnapshot(context.Context, domain.ProjectDependencySnapshot, domain.StagedProjectSet) error {
	return errors.New("UV sync publication requires the actual guarded complete update")
}

func (g *uvSyncPublicationGuard) Commit(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) (resultErr error) {
	if g == nil || g.commitAttempted || !sameUVSyncUpdate(update, g.selected) || !staged.Valid() || !reflect.DeepEqual(staged, g.publication.staged) {
		return errors.New("UV sync publication differs from its single approved selection")
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
	var desired []pypiDestination
	if !update.Snapshot().DependencyFree() {
		temporary, err := os.MkdirTemp(g.transaction.plan.root, ".haa-pypi-work-")
		if err != nil {
			return err
		}
		defer func() {
			if !g.transaction.recovery {
				resultErr = errors.Join(resultErr, os.RemoveAll(temporary))
			}
		}()
		if err := os.Chmod(temporary, 0o700); err != nil {
			return err
		}
		if err := copyExactRecord(filepath.Join(root, manifestFilename), filepath.Join(temporary, manifestFilename), material.document); err != nil {
			return err
		}
		if err := copyExactRecord(filepath.Join(root, sbomFilename), filepath.Join(temporary, sbomFilename), material.sbom); err != nil {
			return err
		}
		requirements, expected, err := preparePythonWheels(temporary, filepath.Join(root, "artifacts"), material.nodes, staged.Set().Inspected().Inspections())
		if err != nil {
			return err
		}
		desired, err = g.owner.promoter.materializePythonProject(ctx, temporary, requirements, expected)
		if err != nil {
			return err
		}
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	if err := g.transaction.commitComplete(desired, update.Snapshot().DependencyFree()); err != nil {
		return err
	}
	g.committed = true
	p.material = material
	return nil
}

// Records are exposed only after successful publication. Cache paths remain
// opaque handles; callers cannot substitute an arbitrary host file as a report.
func (p *UVSyncProjectPromotion) PublishedRecords() (domain.ContentDigest, string, string, bool) {
	if p == nil || p.guard == nil || !p.guard.committed || p.material.manifest.String() == "" {
		return domain.ContentDigest{}, "", "", false
	}
	return p.material.manifest, p.staged.ContentHandle() + ":" + manifestFilename, p.staged.ContentHandle() + ":" + sbomFilename, true
}
