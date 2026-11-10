package promotion

import (
	"context"
	"errors"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// BunOriginalVerifier is trusted composition's full independent old-source proof.
type BunOriginalVerifier func(context.Context, domain.InstallContext, []domain.ProjectControlFile, artifactnpm.BunOriginalLayout) (domain.ContentDigest, error)

func NewBunProjectPromotionWithOriginalVerifier(intake, evidence, cache, state string, reader ProjectEvidenceReader, operation domain.OperationID, selected func(domain.ProjectDependencySnapshot) (*BunVerifiedCache, error), verify BunOriginalVerifier) (*BunProjectPromotion, error) {
	if verify == nil {
		return nil, errors.New("bun original source verifier is required")
	}
	p, err := NewBunProjectPromotion(intake, evidence, cache, state, reader, operation, selected)
	if err != nil {
		return nil, err
	}
	p.engine.verifyBunOriginal = verify
	return p, nil
}
func (p *BunProjectPromotion) BeginUpgrade(ctx context.Context, install domain.InstallContext, adopt bool) (*approvedPNPMProjectGuard, error) {
	if p == nil || p.engine == nil {
		return nil, errors.New("bun promotion unavailable")
	}
	return p.engine.BeginUpgrade(ctx, install, adopt)
}
func (p *PNPMProjectPromotion) originalVerifierAvailable() bool {
	if p.client == npmProjectCacheBun {
		return p.verifyBunOriginal != nil
	}
	return p.verifyOriginal != nil
}
func (g *approvedPNPMProjectGuard) proveBunOriginal(ctx context.Context) error {
	if g.owner.verifyBunOriginal == nil || g.members["node_modules"] == nil || g.members["haa"] != nil || g.previous != nil || len(g.originalState) != 0 {
		return errors.New("bun original native adoption requires its independent source verifier")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	body, uid, gid, err := g.originalTransport(ctx)
	if err != nil {
		return err
	}
	original, err := artifactnpm.ParseBunOriginalLayout(ctx, body, uid, gid)
	if err != nil {
		return err
	}
	proof, err := g.owner.verifyBunOriginal(ctx, g.install, append([]domain.ProjectControlFile(nil), g.controls...), original)
	if err != nil {
		return errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	if proof.String() == "" {
		return errors.New("bun original source proof is missing")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	g.originalProof = proof.String()
	return nil
}
