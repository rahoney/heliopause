package promotion

import (
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func (p *PyPIPromotion) promoteToolSelection(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext, guard *toolFreshTarget, receipt domain.ProjectControlFile) (promoted domain.PromotedInstall, resultErr error) {
	if guard == nil || guard.tx == nil || guard.tx.plan.root != install.Target().String() || install.Mode() != domain.InstallPythonVenv || guard.work != nil {
		return promoted, errors.New("UV tool materialization differs from held fresh target")
	}
	stagedRoot, err := p.validatedPyPIStagedRoot(ctx, staged, bundle)
	if err != nil {
		return promoted, err
	}
	if err := guard.verify(ctx); err != nil {
		return promoted, err
	}
	work, err := newToolPythonWorkspace(guard.tx)
	if err != nil {
		return promoted, err
	}
	guard.work = work
	defer func() {
		resultErr = errors.Join(resultErr, work.Close())
		if resultErr != nil {
			promoted = domain.PromotedInstall{}
		}
	}()
	requirements, expected, err := preparePyPIProject(work.path, stagedRoot, bundle)
	if err != nil {
		return promoted, err
	}
	if err := work.takeInputs(bundle, requirements, expected, artifactpypi.ResourcePolicyFromContext(ctx).WheelLimits()); err != nil {
		return promoted, err
	}
	desired, err := p.materializeToolPythonProject(ctx, work.path, requirements, expected, install)
	if err != nil {
		return promoted, err
	}
	if err := work.takeOutputs(desired); err != nil {
		return promoted, err
	}
	if err := guard.verify(ctx); err != nil {
		return promoted, err
	}
	if err := guard.commitTool(ctx, desired, receipt, work.path); err != nil {
		return promoted, err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), install.Target())
}
