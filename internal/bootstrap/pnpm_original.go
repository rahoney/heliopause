package bootstrap

import (
	"context"
	"errors"
	"fmt"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// Original source proof precedes selection. It includes retired packages and
// does no native execution, observation, Policy or selected-cache publication.
func (b pnpmProjectBindings) provePNPMOriginal(ctx context.Context, install domain.InstallContext, controls []domain.ProjectControlFile, original artifactnpm.PNPMOriginalLayout) (domain.ContentDigest, error) {
	if ctx == nil || b.source == nil || b.static == nil || len(controls) != 2 || !install.Valid() || install.Mode() != domain.InstallNPMProject {
		return domain.ContentDigest{}, errors.New("pnpm original source proof requires actual adapters and original controls")
	}
	if err := ctx.Err(); err != nil {
		return domain.ContentDigest{}, errors.Join(err, context.Cause(ctx))
	}
	if controls[0].Name() != "package.json" || !controls[0].Present() || controls[1].Name() != "pnpm-lock.yaml" || !controls[1].Present() {
		return domain.ContentDigest{}, errors.New("pnpm native adoption requires its original manifest and lock")
	}
	lock, err := artifactnpm.ParsePNPMLock(controls[0].Body(), controls[1].Body(), runtimeidentity.PNPMVersion)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	selected, err := b.source.BindPNPMLock(ctx, lock)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	frozen, err := selected.InspectionContext(install, controls, sandbox.PNPMResolverRuntimeIdentity())
	if err != nil {
		return domain.ContentDigest{}, err
	}
	if len(frozen.Snapshot().Dependencies()) == 0 {
		return domain.ContentDigest{}, errors.New("pnpm native adoption requires a nonempty supported original layout")
	}
	acquired := make(map[domain.DependencyNodeID]domain.AcquiredArtifact)
	var compressed uint64
	for _, resolved := range frozen.Snapshot().Dependencies() {
		if err := ctx.Err(); err != nil {
			return domain.ContentDigest{}, errors.Join(err, context.Cause(ctx))
		}
		run, err := domain.NewRunID()
		if err != nil {
			return domain.ContentDigest{}, err
		}
		subject, err := b.source.Acquire(ctx, run, resolved)
		if err != nil {
			return domain.ContentDigest{}, err
		}
		if subject.Identity() != resolved.Identity() || subject.SizeBytes() > artifactnpm.PNPMInspectionWorkspaceLimit-compressed {
			return domain.ContentDigest{}, errors.New("pnpm original source acquisition exceeds its fixed workspace bound")
		}
		compressed += subject.SizeBytes()
		report, err := (verificationnpm.IntegrityVerifier{}).Verify(ctx, subject)
		if err != nil || report.Execution().Status() != domain.ExecutionCompleted || report.Outcome() != domain.VerificationVerified {
			return domain.ContentDigest{}, errors.Join(errors.New("pnpm original archive integrity was not verified"), err)
		}
		node, err := domain.ProjectDependencyNodeID(resolved.Identity())
		if err != nil {
			return domain.ContentDigest{}, err
		}
		acquired[node] = subject
	}
	input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	budget, err := input.BeginWorkspace()
	if err != nil {
		return domain.ContentDigest{}, err
	}
	inventories := make([]artifactnpm.ArchiveInventory, 0, len(acquired))
	for _, subject := range input.Artifacts() {
		inventory, report, err := b.static.InspectArchive(ctx, subject)
		if err != nil {
			return domain.ContentDigest{}, err
		}
		if report.Execution().Status() != domain.ExecutionCompleted || len(report.Findings()) != 0 || !inventory.Matches(subject) {
			return domain.ContentDigest{}, fmt.Errorf("pnpm original static inventory unavailable for %s@%s", subject.Identity().Name(), subject.Identity().Version())
		}
		if err := budget.Add(inventory); err != nil {
			return domain.ContentDigest{}, err
		}
		inventories = append(inventories, inventory)
	}
	if err := budget.Complete(); err != nil {
		return domain.ContentDigest{}, err
	}
	return input.ProveOriginalLayout(ctx, inventories, original)
}
