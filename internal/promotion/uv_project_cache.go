package promotion

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// UVProjectCache adds native project provenance to the existing approved
// Python cache. Selection never substitutes for current Evidence and Policy.
type UVProjectCache struct {
	cache  *PythonProjectCache
	update domain.ProjectDependencyUpdate
	base   domain.ProjectDependencyUpdate
	proof  domain.ProjectControlFile
}

func NewUVProjectCache(cache *PythonProjectCache, update domain.ProjectDependencyUpdate) (*UVProjectCache, error) {
	if cache == nil {
		return nil, errors.New("UV project requires the approved Python cache")
	}
	if err := sandbox.ValidateUVProjectUpdate(update); err != nil {
		return nil, err
	}
	return &UVProjectCache{cache: cache, update: update}, nil
}

func newUVProjectCacheWithOriginalProof(cache *PythonProjectCache, base, update domain.ProjectDependencyUpdate, proof domain.ProjectControlFile) (*UVProjectCache, error) {
	c := &UVProjectCache{cache: cache, base: base, update: update, proof: proof}
	if cache == nil {
		return nil, errors.New("UV project requires the approved Python cache")
	}
	if err := c.validateUpdate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *UVProjectCache) validateUpdate() error {
	if !c.proof.Present() {
		if c.base.Valid() {
			return errors.New("UV project original proof is missing")
		}
		return sandbox.ValidateUVProjectUpdate(c.update)
	}
	if c.proof.Name() != "pypi-update-native-ownership.json" {
		return errors.New("UV project original proof has an unexpected identity")
	}
	if err := sandbox.ValidateUVProjectUpdate(c.base); err != nil {
		return err
	}
	controls := append(c.base.SelectedControls(), c.proof)
	snapshot, err := bindPythonOriginalProof(c.base.Snapshot(), controls, "uv-project-original-proof/v1")
	if err != nil {
		return err
	}
	expected, err := domain.NewProjectDependencyUpdateFromSnapshot(append(c.base.OriginalControls(), c.proof), controls, snapshot)
	if err != nil {
		return err
	}
	if !sameUVSyncUpdate(expected, c.update) {
		return errors.New("UV project update differs from its authenticated original proof")
	}
	return nil
}

func (c *UVProjectCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if c == nil || c.cache == nil {
		return domain.StagedProjectSet{}, errors.New("UV project cache is unavailable")
	}
	if err := c.validateUpdate(); err != nil {
		return domain.StagedProjectSet{}, err
	}
	return c.cache.stageProject(ctx, set, c.update)
}

// OpenProjectCache performs the existing receipt/current Evidence/whole-file
// readback before returning data. It does not publish a Host environment.
func (c *UVProjectCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (string, error) {
	root, _, err := c.openProject(ctx, staged)
	return root, err
}

func (c *UVProjectCache) openProject(ctx context.Context, staged domain.StagedProjectSet) (string, pythonProjectMaterial, error) {
	if c == nil || c.cache == nil {
		return "", pythonProjectMaterial{}, errors.New("UV project cache is unavailable")
	}
	if err := c.validateUpdate(); err != nil {
		return "", pythonProjectMaterial{}, err
	}
	return c.cache.openProject(ctx, staged, c.update)
}
