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
}

func NewUVProjectCache(cache *PythonProjectCache, update domain.ProjectDependencyUpdate) (*UVProjectCache, error) {
	if cache == nil {
		return nil, errors.New("UV project requires the approved Python cache")
	}
	if err := sandbox.ValidateUVProjectUpdate(update); err != nil {
		return nil, err
	}
	return &UVProjectCache{cache, update}, nil
}

func (c *UVProjectCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	if c == nil || c.cache == nil {
		return domain.StagedProjectSet{}, errors.New("UV project cache is unavailable")
	}
	if err := sandbox.ValidateUVProjectUpdate(c.update); err != nil {
		return domain.StagedProjectSet{}, err
	}
	return c.cache.stageProject(ctx, set, c.update)
}

// OpenProjectCache performs the existing receipt/current Evidence/whole-file
// readback before returning data. It does not publish a Host environment.
func (c *UVProjectCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (string, error) {
	if c == nil || c.cache == nil {
		return "", errors.New("UV project cache is unavailable")
	}
	if err := sandbox.ValidateUVProjectUpdate(c.update); err != nil {
		return "", err
	}
	root, _, err := c.cache.openProject(ctx, staged, c.update)
	return root, err
}
