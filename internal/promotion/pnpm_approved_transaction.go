package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func (g *approvedPNPMProjectGuard) Commit(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) (resultErr error) {
	if g == nil || !update.Valid() || update.HasRequestedResolution() || !staged.Valid() || update.Snapshot().Context() != g.install || update.Snapshot().Source().String() != "npm" || !sameGoTransactionControls(g.controls, update.OriginalControls()) {
		return errors.New("pnpm commit differs from the guarded original project")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	g.ctx = ctx
	cache, err := g.owner.selectedCache(update.Snapshot())
	if err != nil {
		return err
	}
	if cache == nil || cache.intakeRoot != g.owner.intakeRoot || cache.evidenceRoot != g.owner.evidenceRoot || cache.cacheRoot != g.owner.cacheRoot || cache.operation != g.owner.operation || !cache.frozen.MatchesSnapshot(update.Snapshot()) || !cache.frozen.MatchesSnapshot(staged.Set().Inspected().Snapshot()) || !sameGoTransactionControls(cache.frozen.Controls(), update.SelectedControls()) {
		return errors.New("pnpm selected cache belongs to another operation or frozen context")
	}
	data, err := cache.OpenProjectCache(ctx, staged)
	if err != nil {
		return err
	}
	receiptRoot, err := os.OpenRoot(filepath.Dir(data))
	if err != nil {
		return err
	}
	body, _, readErr := readGoTransactionControl(receiptRoot, "receipt.json")
	closeErr := receiptRoot.Close()
	if readErr != nil || closeErr != nil || pnpmCacheHash(body) != staged.Digest().String() {
		return errors.Join(errors.New("pnpm staged receipt changed before commit"), readErr, closeErr)
	}
	var approval pnpmCacheDocument
	if err := json.Unmarshal(body, &approval); err != nil {
		return err
	}
	if err := verifyPNPMRecordedApproval(ctx, g.owner.evidence, approval); err != nil {
		return err
	}
	manifest, err := domain.NewSHA256Digest(approval.Manifest)
	if err != nil {
		return err
	}
	prepared, err := cache.PrepareProject(ctx, staged)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, prepared.Close()) }()
	profile := &pnpmProjectProfile{owner: g, prepared: prepared, retain: map[string]bool{}, approval: pnpmProjectApproval{Schema: 1, OriginalProof: g.originalProof, Cache: staged.ContentHandle(), CacheDigest: staged.Digest().String(), Approval: approval}}
	for _, control := range update.SelectedControls() {
		original := g.controlInfo[control.Name()]
		if original == nil {
			continue
		}
		if err := prepared.verifyEntry(ctx, control.Name()); err != nil {
			return err
		}
		f, err := prepared.root.OpenFile(control.Name(), os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		chmodErr := f.Chmod(original.Mode().Perm())
		syncErr := f.Sync()
		info, statErr := f.Stat()
		closeErr := f.Close()
		if chmodErr != nil || syncErr != nil || statErr != nil || closeErr != nil {
			return errors.Join(chmodErr, syncErr, statErr, closeErr)
		}
		entry := prepared.entries[control.Name()]
		entry.info = info
		prepared.entries[control.Name()] = entry
		for _, before := range g.controls {
			if before.Name() == control.Name() && before.Present() && before.Digest() == control.Digest() {
				profile.retain[control.Name()] = true
			}
		}
	}
	if g.previous != nil && g.previous.Approval.Graph == approval.Graph && g.previous.Approval.Runtime == approval.Runtime && g.previous.Approval.Layout == approval.Layout {
		profile.retain["node_modules"] = true
	}
	// The state envelope has four fixed-size census digests. Check its exact
	// serialized bound before any old project member can be moved.
	for _, name := range []string{"package.json", "pnpm-lock.yaml", "node_modules", "haa"} {
		profile.approval.Members = append(profile.approval.Members, pnpmMemberApproval{name, strings.Repeat("0", 64)})
	}
	stateBody, err := json.Marshal(profile.approval)
	if err != nil || len(stateBody) > pnpmCacheReceiptLimit {
		return errors.Join(errors.New("pnpm independent approval exceeds its fixed bound"), err)
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	plan := npmProjectPlan{root: g.install.Target().String(), rootInfo: g.guard.rootInfo, pnpm: profile}
	transaction, err := beginNPMProjectTransaction(plan, prepared.Directory())
	if err != nil {
		return err
	}
	if err := transaction.commit(); err != nil {
		return err
	}
	g.committed = true
	g.recordManifest = manifest
	g.recordHandle = staged.ContentHandle()
	g.owner.published = g
	return nil
}

// The existing generic port also supports snapshot-only download operations.
// PNPM mutation requires its frozen original/selected control pair instead.
func (g *approvedPNPMProjectGuard) CommitSnapshot(context.Context, domain.ProjectDependencySnapshot, domain.StagedProjectSet) error {
	return errors.New("pnpm mutation requires original and selected project controls")
}
