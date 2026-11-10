package promotion

import (
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// BunPreparedProject owns a private unpublished sibling and grants no target mutation.
type BunPreparedProject struct{ owned *PNPMPreparedProject }

func (p *BunPreparedProject) Directory() string {
	if p == nil {
		return ""
	}
	return p.owned.Directory()
}
func (p *BunPreparedProject) Verify(ctx context.Context) error {
	if p == nil {
		return errors.New("bun prepared project is not active")
	}
	return p.owned.Verify(ctx)
}
func (p *BunPreparedProject) Close() error {
	if p == nil {
		return errors.New("bun prepared project is not active")
	}
	return p.owned.Close()
}

func (c *BunVerifiedCache) hostEntries(ctx context.Context, staged domain.StagedProjectSet) ([]pnpmHostEntry, error) {
	data, err := c.OpenProjectCache(ctx, staged)
	if err != nil {
		return nil, err
	}
	var controls []domain.ProjectControlFile
	for _, control := range c.frozen.Controls() {
		if control.Present() {
			controls = append(controls, control)
		}
	}
	return npmProjectHostEntries(ctx, data, staged.Digest(), controls, func(doc pnpmCacheDocument) (pnpmProjectRecords, error) {
		return c.projectRecords(ctx, staged.Set(), doc)
	}, npmProjectCacheBun)
}

// PrepareProject copies authenticated approved data into an owned private sibling.
func (c *BunVerifiedCache) PrepareProject(ctx context.Context, staged domain.StagedProjectSet) (*BunPreparedProject, error) {
	entries, err := c.hostEntries(ctx, staged)
	if err != nil {
		return nil, err
	}
	var before func(*PNPMPreparedProject) error
	if c.beforePrepareSync != nil {
		before = func(p *PNPMPreparedProject) error { return c.beforePrepareSync(&BunPreparedProject{owned: p}) }
	}
	owned, err := prepareNPMProject(ctx, c.frozen.Snapshot().Context().Target().String(), entries, before, npmProjectCacheBun)
	if err != nil {
		return nil, err
	}
	return &BunPreparedProject{owned: owned}, nil
}
