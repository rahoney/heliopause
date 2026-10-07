package terraformprovider

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// ResolveProjectDependencyUpdate selects the complete explicit provider set
// from guarded, frozen controls. Package reads calculate lock data only; the
// existing inspection workflow subsequently verifies signer, observes each ELF,
// records independent Evidence and obtains entry/set Policy decisions.
func (c *Client) ResolveProjectDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, controls []domain.ProjectControlFile) (domain.ProjectDependencyUpdate, error) {
	if c == nil || ctx == nil || ctx.Err() != nil || !install.Valid() || len(controls) != 2 {
		return domain.ProjectDependencyUpdate{}, errors.New("terraform guarded update request is invalid")
	}
	parsed, e := ParseReference(reference.Locator())
	if e != nil || parsed != reference {
		return domain.ProjectDependencyUpdate{}, errors.New("terraform exact primary reference is invalid")
	}
	requirements, e := Requirements(controls)
	if e != nil {
		return domain.ProjectDependencyUpdate{}, e
	}
	var originalLock []byte
	var config domain.ProjectControlFile
	for _, control := range controls {
		switch control.Name() {
		case LockControl:
			originalLock = control.Body()
		case ConfigurationControl:
			config = control
		default:
			return domain.ProjectDependencyUpdate{}, errors.New("terraform frozen controls have unexpected members")
		}
	}
	if !config.Present() {
		return domain.ProjectDependencyUpdate{}, errors.New("terraform frozen configuration is absent")
	}
	locks, e := ParseLock(originalLock)
	if e != nil {
		return domain.ProjectDependencyUpdate{}, e
	}
	locked := map[string]LockedProvider{}
	for _, l := range locks {
		locked[l.Address] = l
	}
	primaryParts := strings.Split(reference.Locator(), "@")
	primaryAddress := "registry.terraform.io/" + primaryParts[0]
	found := false
	var selected []LockedProvider
	var artifacts []domain.ResolvedArtifact
	var nodes []domain.LockedDependency
	for _, r := range requirements {
		if ctx.Err() != nil {
			return domain.ProjectDependencyUpdate{}, ctx.Err()
		}
		old, exists := locked[r.Address]
		delete(locked, r.Address)
		version := old.Version
		primary := r.Address == primaryAddress
		if primary {
			version = primaryParts[1]
			found = true
		}
		if !primary && !exists {
			return domain.ProjectDependencyUpdate{}, errors.New("terraform non-primary provider requires an exact existing lock selection")
		}
		for _, constraint := range r.Constraints {
			if !MatchesConstraint(version, constraint) {
				return domain.ProjectDependencyUpdate{}, errors.New("terraform exact selection does not satisfy configuration constraints")
			}
		}
		ref, e := ParseReference(strings.TrimPrefix(r.Address, "registry.terraform.io/") + "@" + version)
		if e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		resolved, e := c.Resolve(ctx, ref)
		if e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		run, e := domain.NewRunID()
		if e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		acquired, e := c.Acquire(ctx, run, resolved)
		if e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		bundle, e := ReadIntake(c.intakeRoot, acquired)
		if e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		contents, e := InspectPackage(ctx, bundle, ref.Locator())
		if e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		if exists && old.Version == version {
			if e := VerifyPackageLock(old, contents); e != nil {
				return domain.ProjectDependencyUpdate{}, e
			}
		}
		if e := c.freezePackage(resolved, bundle); e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		p := LockedProvider{Address: r.Address, Version: version, Constraints: strings.Join(r.Constraints, ", "), Hashes: []string{contents.H1, contents.ZH}}
		if exists && old.Version == version {
			for _, hash := range old.Hashes {
				if hash != contents.H1 && hash != contents.ZH {
					p.Hashes = append(p.Hashes, hash)
				}
			}
		}
		selected = append(selected, p)
		artifacts = append(artifacts, resolved)
		key, _, _ := ParseIntegrity(resolved.DeclaredIntegrity())
		id, _ := domain.NewDependencyNodeID("t" + key[:24])
		if primary {
			node, e := domain.NewLockedDependency(id, domain.DependencyPrimary, resolved)
			if e != nil {
				return domain.ProjectDependencyUpdate{}, e
			}
			// Independent required providers are project roots, not dependencies
			// of the requested provider. The complete snapshot owns their coverage.
			nodes = append(nodes, node)
		}
	}
	if !found || len(locked) != 0 {
		return domain.ProjectDependencyUpdate{}, errors.New("terraform requested/locked provider set differs from complete configuration")
	}
	body, e := EncodeLock(selected)
	if e != nil {
		return domain.ProjectDependencyUpdate{}, e
	}
	lock, e := domain.NewProjectControlFile(LockControl, body, true)
	if e != nil {
		return domain.ProjectDependencyUpdate{}, e
	}
	after := []domain.ProjectControlFile{config, lock}
	var digests []domain.ProjectControlDigest
	for _, control := range after {
		d, e := domain.NewProjectControlDigest(control.Name(), control.Digest())
		if e != nil {
			return domain.ProjectDependencyUpdate{}, e
		}
		digests = append(digests, d)
	}
	graph, e := domain.NewLockedDependencyGraph(nodes, nil)
	if e != nil {
		return domain.ProjectDependencyUpdate{}, e
	}
	// Deterministic framed identity binds exact selected controls and sources.
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Identity().Name() < artifacts[j].Identity().Name() })
	binding := ConfigurationControl + "\x00" + config.Digest().String() + "\x00" + LockControl + "\x00" + lock.Digest().String()
	for _, a := range artifacts {
		binding += "\x00" + a.Identity().Name() + "\x00" + a.Identity().Version() + "\x00" + a.DeclaredIntegrity()
	}
	digest, _ := domain.NewSHA256Digest(sha256Hex([]byte(binding)))
	snapshot, e := domain.NewProjectDependencySnapshot(install, Source(), digests, artifacts, digest)
	if e != nil {
		return domain.ProjectDependencyUpdate{}, e
	}
	resolution, e := domain.NewDependencyResolution(graph, "terraform-registry:registry.terraform.io;platform:linux/amd64", digest)
	if e != nil {
		return domain.ProjectDependencyUpdate{}, e
	}
	return domain.NewProjectDependencyUpdate(controls, after, snapshot, resolution)
}

func (c *Client) freezePackage(resolved domain.ResolvedArtifact, bundle Bundle) error {
	key, archive, e := ParseIntegrity(resolved.DeclaredIntegrity())
	if e != nil || bundle.RegistryDigest() != key || bundle.ArchiveDigest() != archive {
		return errors.New("terraform package differs from exact selected source")
	}
	body, e := encodeParts(bundleHeader, bundle.parts, bundleLimits)
	if e != nil {
		return e
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.prepared == nil {
		c.prepared = map[string][]byte{}
	}
	if prior := c.prepared[key]; prior != nil {
		if sha256Hex(prior) != sha256Hex(body) {
			return errors.New("terraform frozen package changed")
		}
		return nil
	}
	if len(body) > providerExpandedLimit-c.preparedBytes {
		return errors.New("terraform complete frozen package set exceeds byte bounds")
	}
	c.prepared[key] = body
	c.preparedBytes += len(body)
	return nil
}
