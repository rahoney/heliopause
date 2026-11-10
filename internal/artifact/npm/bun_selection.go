package npm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunBinding = errors.New("bun selection differs from independent public registry metadata")

// BunSelection binds each unique archive and declared dependency/peer/bin
// metadata to the independent npm source. It grants no archive, lifecycle,
// layout, Policy or promotion approval.
type BunSelection struct {
	bound     bool
	lock      BunLock
	artifacts []domain.ResolvedArtifact
}

func (s BunSelection) SourceBound() bool { return s.bound }
func (s BunSelection) Lock() BunLock     { return s.lock }
func (s BunSelection) Artifacts() []domain.ResolvedArtifact {
	return append([]domain.ResolvedArtifact(nil), s.artifacts...)
}

func (r *Resolver) BindBunLock(ctx context.Context, lock BunLock) (BunSelection, error) {
	if ctx == nil || r == nil || r.registry == nil || r.client == nil || !lock.valid || r.registry.Scheme != "https" || !strings.EqualFold(r.registry.Host, "registry.npmjs.org") || r.registry.Path != "/" {
		return BunSelection{}, errBunBinding
	}
	if err := ctx.Err(); err != nil {
		return BunSelection{}, errors.Join(errBunBinding, err)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	selection := BunSelection{lock: lock}
	seen := map[string]bool{}
	for _, pkg := range lock.packages {
		if err := ctx.Err(); err != nil {
			return BunSelection{}, errors.Join(errBunBinding, err)
		}
		key := pkg.Name + "@" + pkg.Version
		if seen[key] {
			continue
		}
		artifact, err := r.bindBunPackage(ctx, pkg)
		if err != nil || ctx.Err() != nil {
			return BunSelection{}, errors.Join(errBunBinding, err, ctx.Err())
		}
		seen[key] = true
		selection.artifacts = append(selection.artifacts, artifact)
	}
	if err := ctx.Err(); err != nil {
		return BunSelection{}, errors.Join(errBunBinding, err)
	}
	sort.Slice(selection.artifacts, func(i, j int) bool {
		a, b := selection.artifacts[i].Identity(), selection.artifacts[j].Identity()
		return a.Name()+"@"+a.Version() < b.Name()+"@"+b.Version()
	})
	selection.bound = true
	return selection, nil
}

func (r *Resolver) bindBunPackage(ctx context.Context, pkg BunLockedPackage) (domain.ResolvedArtifact, error) {
	// The already implemented exact-version HTTP owner retains its deadline,
	// origin, escaped path, response type and body bound. PNPM is unchanged.
	body, err := r.fetchMetadataBody(ctx, pkg.Name, pkg.Version)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	if !utf8.Valid(body) || strictControlJSON(body) != nil {
		return domain.ResolvedArtifact{}, errBunBinding
	}
	fields, err := bunJSONObject(body)
	if err != nil || !supportedBunSourceMetadata(fields) {
		return domain.ResolvedArtifact{}, errBunBinding
	}
	dependencies, err := bunOptionalConstraints(fields, "dependencies")
	if err != nil || !maps.Equal(dependencies, pkg.Dependencies) {
		return domain.ResolvedArtifact{}, errBunBinding
	}
	peers, err := bunOptionalConstraints(fields, "peerDependencies")
	if err != nil || !maps.Equal(peers, pkg.PeerDependencies) {
		return domain.ResolvedArtifact{}, errBunBinding
	}
	bins := map[string]string{}
	if raw, exists := fields["bin"]; exists {
		bins, err = bunPackageBins(raw, pkg.Name)
		if err != nil {
			return domain.ResolvedArtifact{}, errBunBinding
		}
	}
	if !maps.Equal(bins, pkg.Bins) {
		return domain.ResolvedArtifact{}, errBunBinding
	}
	var metadata npmVersionMetadata
	if json.Unmarshal(body, &metadata) != nil {
		return domain.ResolvedArtifact{}, errBunBinding
	}
	source, err := domain.NewSourceID("npm")
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	artifact, err := r.resolvedVersion(source, pkg.Name, pkg.Version, metadata)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	if !bunArtifactMatches(pkg, artifact) {
		return domain.ResolvedArtifact{}, errBunBinding
	}
	return artifact, nil
}

func bunArtifactMatches(pkg BunLockedPackage, artifact domain.ResolvedArtifact) bool {
	identity := artifact.Identity()
	return identity.Source().String() == "npm" && identity.Name() == pkg.Name && identity.Version() == pkg.Version && identity.Variant() == "tarball" && artifact.DeclaredIntegrity() == pkg.Integrity && validRegistryTarball(artifact.AcquisitionLocator()) && (pkg.Registry == "" || pkg.Registry == artifact.AcquisitionLocator())
}

func supportedBunSourceMetadata(fields map[string]json.RawMessage) bool {
	for _, key := range []string{"optionalDependencies", "peerDependenciesMeta", "bundledDependencies", "bundleDependencies", "os", "cpu", "libc"} {
		if body, present := fields[key]; present {
			body = bytes.TrimSpace(body)
			empty := emptyBunAggregate(body, false)
			switch key {
			case "optionalDependencies", "peerDependenciesMeta":
				empty = emptyBunAggregate(body, true)
			case "bundledDependencies", "bundleDependencies":
				empty = empty || bytes.Equal(body, []byte("false"))
			}
			if !empty {
				return false
			}
		}
	}
	return true
}
