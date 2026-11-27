package npm

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const pnpmBindingTimeout = 2 * time.Minute

var errPNPMBinding = errors.New("pnpm selection differs from independent public registry metadata")

// PNPMSelection binds native instance wiring to exact registry candidates.
// Source binding is not archive verification, Evidence, Policy or promotion.
type PNPMSelection struct {
	bound     bool
	lock      PNPMLock
	artifacts []domain.ResolvedArtifact
}

func (s PNPMSelection) SourceBound() bool { return s.bound }
func (s PNPMSelection) Lock() PNPMLock    { return s.lock }
func (s PNPMSelection) Artifacts() []domain.ResolvedArtifact {
	return append([]domain.ResolvedArtifact(nil), s.artifacts...)
}

// BindPNPMLock reuses the existing exact-version registry resolver. Native
// output supplies neither the registry URL nor a substitute for its SRI.
// Every peer instance remains in Lock; its shared tarball is bound once.
func (r *Resolver) BindPNPMLock(ctx context.Context, lock PNPMLock) (PNPMSelection, error) {
	if ctx == nil || r == nil || r.registry == nil || r.client == nil || !lock.valid ||
		r.registry.Scheme != "https" || !strings.EqualFold(r.registry.Host, "registry.npmjs.org") || r.registry.Path != "/" {
		return PNPMSelection{}, errPNPMBinding
	}
	if err := ctx.Err(); err != nil {
		return PNPMSelection{}, errors.Join(errPNPMBinding, err)
	}
	ctx, cancel := context.WithTimeout(ctx, pnpmBindingTimeout)
	defer cancel()
	selection := PNPMSelection{lock: lock}
	for _, pkg := range lock.packages {
		if err := ctx.Err(); err != nil {
			return PNPMSelection{}, errors.Join(errPNPMBinding, err)
		}
		reference, err := ParseReference(pkg.Name + "@" + pkg.Version)
		if err != nil {
			return PNPMSelection{}, errPNPMBinding
		}
		artifact, err := r.resolvePNPMVersion(ctx, reference)
		if err != nil || ctx.Err() != nil {
			return PNPMSelection{}, errors.Join(errPNPMBinding, err, ctx.Err())
		}
		identity := artifact.Identity()
		if identity.Source().String() != "npm" || identity.Name() != pkg.Name || identity.Version() != pkg.Version || identity.Variant() != "tarball" ||
			artifact.DeclaredIntegrity() != pkg.Integrity || !validRegistryTarball(artifact.AcquisitionLocator()) ||
			pkg.Tarball != "" && pkg.Tarball != artifact.AcquisitionLocator() {
			return PNPMSelection{}, errPNPMBinding
		}
		selection.artifacts = append(selection.artifacts, artifact)
	}
	if err := ctx.Err(); err != nil {
		return PNPMSelection{}, errors.Join(errPNPMBinding, err)
	}
	selection.bound = true
	return selection, nil
}

// The native selection is already exact. Fetch one version document rather
// than the package's growing history, retaining the ordinary transport bounds.
func (r *Resolver) resolvePNPMVersion(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	parsed, err := parseLocator(reference.Locator())
	if err != nil || !parsed.exact || reference.Source().String() != "npm" {
		return domain.ResolvedArtifact{}, errPNPMBinding
	}
	metadata, err := r.fetchVersionMetadata(ctx, parsed.name, parsed.selector)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	return r.resolvedVersion(reference.Source(), parsed.name, parsed.selector, metadata)
}
