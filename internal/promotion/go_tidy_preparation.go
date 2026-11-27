package promotion

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// GoTidyPreparation retains the existing approval guard and source witness
// through the shared complete-project update workflow. It is request-scoped.
type GoTidyPreparation struct {
	promoter *GoProjectPromotion
	resolver *sandbox.GoModuleResolver
	adopt    bool
	guard    *approvedGoProjectGuard
}

func NewGoTidyPreparation(promoter *GoProjectPromotion, resolver *sandbox.GoModuleResolver, adopt bool) (*GoTidyPreparation, error) {
	if promoter == nil || resolver == nil {
		return nil, errors.New("go tidy requires existing promotion and isolated resolver")
	}
	return &GoTidyPreparation{promoter: promoter, resolver: resolver, adopt: adopt}, nil
}

func (p *GoTidyPreparation) Begin(ctx context.Context, install domain.InstallContext) (ports.ProjectMutationGuard, error) {
	if p == nil || p.guard != nil {
		return nil, errors.New("go tidy preparation already used or unavailable")
	}
	var g *approvedGoProjectGuard
	var err error
	if p.adopt {
		g, err = p.promoter.BeginUpgrade(ctx, install, true)
	} else {
		g, err = p.promoter.Begin(ctx, install)
	}
	if err != nil {
		return nil, err
	}
	g.plan.upgrade = true
	if err = g.plan.validateUpgradeControls(); err == nil {
		g.tidySource, err = sandbox.CaptureGoProjectSource(ctx, install.Target().String())
	}
	if err == nil {
		err = g.VerifyUnchanged(ctx)
	}
	if err != nil {
		return nil, errors.Join(err, g.Close())
	}
	p.guard = g
	return g, nil
}

func (p *GoTidyPreparation) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.guard == nil || selector != "" || p.guard.context != install || !sameGoTransactionControls(p.guard.Controls(), original) {
		return domain.ProjectDependencyUpdate{}, errors.New("go tidy request differs from its held original")
	}
	if err := p.guard.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	return p.resolver.ResolveProjectTidy(ctx, install, original, p.guard.tidySource)
}
