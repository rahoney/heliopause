package sandbox

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// ResolveToolRun binds the native selection to independent public originals
// without manufacturing an installation target for ephemeral execution.
func (r *PNPMResolver) ResolveToolRun(ctx context.Context, reference domain.ArtifactReference, source *artifactnpm.Resolver) (artifactnpm.PNPMInspectionContext, error) {
	if ctx == nil || source == nil {
		return artifactnpm.PNPMInspectionContext{}, errors.New("pnpm one-off resolution requires context and public source")
	}
	cache, err := r.SelectToolRun(ctx, reference)
	if err != nil {
		return artifactnpm.PNPMInspectionContext{}, err
	}
	return source.BindPNPMRunCache(ctx, cache, reference, PNPMResolverRuntimeIdentity())
}
