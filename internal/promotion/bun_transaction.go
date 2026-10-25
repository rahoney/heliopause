package promotion

import (
	"context"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
	"path/filepath"
)

// BunProjectPromotion binds the existing owning transaction to a typed Bun cache.
// Unmanaged native payload adoption remains unavailable in this constructor.
type BunProjectPromotion struct{ engine *PNPMProjectPromotion }

func NewBunProjectPromotion(intake, evidence, cache, state string, reader ProjectEvidenceReader, operation domain.OperationID, selected func(domain.ProjectDependencySnapshot) (*BunVerifiedCache, error)) (*BunProjectPromotion, error) {
	roots := []string{intake, evidence, cache, state}
	if reader == nil || operation.String() == "" || selected == nil || !separateRoots(roots) {
		return nil, errors.New("bun mutation requires independent roots, operation and selected cache")
	}
	for _, root := range roots {
		if root == "/" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return nil, errors.New("bun mutation roots must be canonical")
		}
	}
	return &BunProjectPromotion{engine: &PNPMProjectPromotion{client: npmProjectCacheBun, intakeRoot: intake, evidenceRoot: evidence, cacheRoot: cache, stateRoot: state, evidence: reader, operation: operation, selectedBunCache: selected}}, nil
}
func (p *BunProjectPromotion) Begin(ctx context.Context, install domain.InstallContext) (*approvedPNPMProjectGuard, error) {
	if p == nil || p.engine == nil {
		return nil, errors.New("bun promotion unavailable")
	}
	return p.engine.Begin(ctx, install)
}
func (p *BunProjectPromotion) PublishedRecords() (domain.ContentDigest, string, string, bool) {
	if p == nil || p.engine == nil {
		return domain.ContentDigest{}, "", "", false
	}
	return p.engine.PublishedRecords()
}
func (p *PNPMProjectPromotion) lockName() string {
	if p.client == npmProjectCacheBun {
		return "bun.lock"
	}
	return "pnpm-lock.yaml"
}
func (p *PNPMProjectPromotion) clientPrefix() string {
	if p.client == npmProjectCacheBun {
		return "bun"
	}
	return "pnpm"
}
func (p *PNPMProjectPromotion) markerName() string { return p.clientPrefix() + "-transaction.json" }

func (p *PNPMProjectPromotion) verifyRecordedApproval(ctx context.Context, doc pnpmCacheDocument) error {
	if p.client == npmProjectCacheBun {
		controls := map[string]cacheFileRecord{}
		for _, c := range doc.Controls {
			if c.Path != "package.json" && c.Path != "bun.lock" || controls[c.Path].Path != "" {
				return errors.New("bun retained controls differ")
			}
			controls[c.Path] = c
		}
		if len(controls) != 2 || controls["package.json"].Kind != "file" || controls["package.json"].Size <= 0 {
			return errors.New("bun retained manifest is absent")
		}
		lock := controls["bun.lock"]
		if lock.Kind == "absent" {
			if !doc.DependencyFree || lock.Size != 0 || lock.SHA256 != pnpmCacheHash(nil) {
				return errors.New("bun absent retained lock is invalid")
			}
		} else if lock.Kind != "file" || lock.Size <= 0 {
			return errors.New("bun retained lock is invalid")
		}
	}
	return verifyPNPMRecordedApproval(ctx, p.evidence, doc)
}
func (p *pnpmProjectProfile) absentSelected(name string) bool {
	if p.owner.owner.client != npmProjectCacheBun || name != "bun.lock" {
		return false
	}
	for _, c := range p.approval.Approval.Controls {
		if c.Path == name {
			return c.Kind == "absent"
		}
	}
	return false
}
func npmApprovedMemberDigest(client npmProjectCacheClient, name string, member *npmTransactionMember, doc pnpmCacheDocument) (string, error) {
	if client == npmProjectCacheBun && name == "bun.lock" && member == nil {
		for _, c := range doc.Controls {
			if c.Path == name && c.Kind == "absent" && doc.DependencyFree {
				return pnpmCacheHash([]byte("helox.npm-project-member/absent/v1")), nil
			}
		}
	}
	return npmPNPMMemberDigest(member)
}

func (p *PNPMProjectPromotion) sameControls(a, b []domain.ProjectControlFile) bool {
	if p.client == npmProjectCacheBun {
		return len(a) == 2 && len(b) == 2 && sameUVSyncControls(a, b)
	}
	return sameGoTransactionControls(a, b)
}

func (g *approvedPNPMProjectGuard) selectedCacheData(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) (string, func() (*PNPMPreparedProject, error), error) {
	if g.owner.client == npmProjectCacheBun {
		cache, err := g.owner.selectedBunCache(update.Snapshot())
		if err != nil {
			return "", nil, err
		}
		if cache == nil || cache.intakeRoot != g.owner.intakeRoot || cache.evidenceRoot != g.owner.evidenceRoot || cache.cacheRoot != g.owner.cacheRoot || cache.operation != g.owner.operation || !cache.frozen.MatchesSnapshot(update.Snapshot()) || !cache.frozen.MatchesSnapshot(staged.Set().Inspected().Snapshot()) || !sameUVSyncControls(cache.frozen.Controls(), update.SelectedControls()) {
			return "", nil, errors.New("bun selected cache belongs to another operation or frozen context")
		}
		data, err := cache.OpenProjectCache(ctx, staged)
		if err != nil {
			return "", nil, err
		}
		return data, func() (*PNPMPreparedProject, error) {
			prepared, err := cache.PrepareProject(ctx, staged)
			if err != nil {
				return nil, err
			}
			return prepared.owned, nil
		}, nil
	}
	cache, err := g.owner.selectedCache(update.Snapshot())
	if err != nil {
		return "", nil, err
	}
	if cache == nil || cache.intakeRoot != g.owner.intakeRoot || cache.evidenceRoot != g.owner.evidenceRoot || cache.cacheRoot != g.owner.cacheRoot || cache.operation != g.owner.operation || !cache.frozen.MatchesSnapshot(update.Snapshot()) || !cache.frozen.MatchesSnapshot(staged.Set().Inspected().Snapshot()) || !sameGoTransactionControls(cache.frozen.Controls(), update.SelectedControls()) {
		return "", nil, errors.New("pnpm selected cache belongs to another operation or frozen context")
	}
	data, err := cache.OpenProjectCache(ctx, staged)
	if err != nil {
		return "", nil, err
	}
	return data, func() (*PNPMPreparedProject, error) { return cache.PrepareProject(ctx, staged) }, nil
}
