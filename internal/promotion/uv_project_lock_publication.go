package promotion

import (
	"context"
	"errors"
	"reflect"
	"runtime"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// UVProjectLockPromotion connects one native selection/current approval to
// only the held project's lock file. Approved wheels remain in the cache.
type UVProjectLockPromotion struct {
	root                    string
	resolver                ports.ProjectLockUpdateResolver
	cache                   *PythonProjectCache
	guard                   *uvProjectLockGuard
	started, stageAttempted bool
	projectCache            *UVProjectCache
	staged                  domain.StagedProjectSet
	material                pythonProjectMaterial
}

func NewUVProjectLockPromotion(root string, resolver ports.ProjectLockUpdateResolver, cache *PythonProjectCache) (*UVProjectLockPromotion, error) {
	if root == "" || resolver == nil || cache == nil || cache.storage == nil {
		return nil, errors.New("UV lock requires native selection and actual approved storage")
	}
	return &UVProjectLockPromotion{root: root, resolver: resolver, cache: cache}, nil
}

type uvProjectLockGuard struct {
	owner                                         *UVProjectLockPromotion
	files                                         *uvProjectLockFiles
	install                                       domain.InstallContext
	controls                                      []domain.ProjectControlFile
	selected                                      domain.ProjectDependencyUpdate
	attempted, commitAttempted, committed, closed bool
	closeErr                                      error
}

func (p *UVProjectLockPromotion) Begin(ctx context.Context, install domain.InstallContext) (result ports.ProjectMutationGuard, resultErr error) {
	if p == nil || p.started || ctx == nil || !install.Valid() || install.Mode() != domain.InstallPythonProjectControls || install.Target().String() != p.root || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil, errors.New("UV lock requires one Linux amd64 project-controls transaction")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.started = true
	files, err := freezeUVProjectLockFiles(ctx, p.root, []string{p.cache.storage.intakeRoot, p.cache.storage.evidenceRoot, p.cache.storage.stagingRoot})
	if err != nil {
		return nil, err
	}
	g := &uvProjectLockGuard{owner: p, files: files, install: install}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, g.Close())
			result = nil
		}
	}()
	g.controls, err = sandbox.NewUVProjectOriginalControls(files.input)
	if err != nil {
		return nil, err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return nil, err
	}
	p.guard = g
	return g, nil
}
func (g *uvProjectLockGuard) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), g.controls...)
}
func (g *uvProjectLockGuard) VerifyUnchanged(ctx context.Context) error {
	if g == nil || g.closed || g.committed {
		return errors.New("UV lock original guard is unavailable")
	}
	return g.files.verify(ctx)
}
func (g *uvProjectLockGuard) Close() error {
	if g == nil {
		return nil
	}
	if !g.closed {
		g.closed = true
		g.closeErr = g.files.close()
	}
	return g.closeErr
}
func (p *UVProjectLockPromotion) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.guard == nil {
		return domain.ProjectDependencyUpdate{}, errors.New("UV lock selection requires its original guard")
	}
	g := p.guard
	if g.attempted || install != g.install || selector != "" || !sameUVSyncControls(original, g.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("UV lock original selection changed or repeated")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	g.attempted = true
	update, err := p.resolver.ResolveProjectLockUpdate(ctx, install, g.Controls(), "")
	if err = errors.Join(err, g.VerifyUnchanged(ctx)); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	if update.Snapshot().Context() != install || !sameUVSyncControls(update.OriginalControls(), g.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("UV lock native selection changed original context or controls")
	}
	if err := sandbox.ValidateUVProjectUpdate(update); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	g.selected = update
	return update, nil
}
func (p *UVProjectLockPromotion) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if p == nil || p.guard == nil || p.stageAttempted || !p.guard.selected.Valid() {
		return domain.StagedProjectSet{}, errors.New("UV lock staging is unavailable or repeated")
	}
	if err := p.guard.VerifyUnchanged(ctx); err != nil {
		return domain.StagedProjectSet{}, err
	}
	p.stageAttempted = true
	cache, err := NewUVProjectCache(p.cache, p.guard.selected)
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
func (*uvProjectLockGuard) CommitSnapshot(context.Context, domain.ProjectDependencySnapshot, domain.StagedProjectSet) error {
	return errors.New("UV lock requires its exact approved native update")
}
func (g *uvProjectLockGuard) Commit(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) error {
	if g == nil || g.commitAttempted || !sameUVSyncUpdate(update, g.selected) || !staged.Valid() || !reflect.DeepEqual(staged, g.owner.staged) || g.owner.projectCache == nil {
		return errors.New("UV lock publication differs from its single approved selection")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	g.commitAttempted = true
	_, material, err := g.owner.projectCache.openProject(ctx, staged)
	if err != nil {
		return err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	var lock domain.ProjectControlFile
	for _, control := range update.SelectedControls() {
		if control.Name() == artifactpypi.UVNativeLockControl {
			lock = control
		}
	}
	if err := g.files.commit(ctx, lock); err != nil {
		return err
	}
	g.committed = true
	g.owner.material = material
	return nil
}
func (p *UVProjectLockPromotion) PublishedRecords() (domain.ContentDigest, string, string, bool) {
	if p == nil || p.guard == nil || !p.guard.committed || p.material.manifest.String() == "" {
		return domain.ContentDigest{}, "", "", false
	}
	return p.material.manifest, p.staged.ContentHandle() + ":" + manifestFilename, p.staged.ContentHandle() + ":" + sbomFilename, true
}
