package npm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMRunBinding = errors.New("pnpm one-off resolution differs from its request, native instances or independent source")

// BindPNPMRunCache authenticates each unique tarball independently and retains
// every native peer instance in the execution graph. Cache payloads and links
// grant no approval, installation ownership or permission to execute.
func (r *Resolver) BindPNPMRunCache(ctx context.Context, cache PNPMRunCache, reference domain.ArtifactReference, runtimeIdentity string) (frozen PNPMInspectionContext, resultErr error) {
	if ctx == nil {
		return frozen, errPNPMRunBinding
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			frozen = PNPMInspectionContext{}
		}
	}()
	if ctx.Err() != nil || runtimeIdentity == "" || cache.root == "" || len(cache.controls) != 2 {
		return frozen, errPNPMRunBinding
	}
	primary, err := cache.lock.runPrimary(reference)
	if err != nil {
		return frozen, err
	}
	selected, err := r.BindPNPMLock(ctx, cache.lock)
	if err != nil {
		return frozen, err
	}
	graph, err := selected.runGraph(primary)
	if err != nil {
		return frozen, err
	}
	type controlDigest struct{ Name, SHA256 string }
	document := struct {
		Schema, Source, Reference, Runtime string
		Controls                           []controlDigest
	}{Schema: "pnpm-dlx-selection/v1", Source: reference.Source().String(), Reference: reference.Locator(), Runtime: runtimeIdentity}
	for _, control := range cache.controls {
		document.Controls = append(document.Controls, controlDigest{control.Name(), control.Digest().String()})
	}
	body, err := json.Marshal(document)
	if err != nil {
		return frozen, err
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return frozen, err
	}
	resolution, err := domain.NewDependencyResolution(graph, "pnpm-dlx-isolated:"+runtimeIdentity, digest)
	if err != nil {
		return frozen, err
	}
	return PNPMInspectionContext{selection: selected, controls: append([]domain.ProjectControlFile(nil), cache.controls...), runtime: runtimeIdentity, execution: resolution}, nil
}

func (s PNPMSelection) runGraph(primary string) (domain.LockedDependencyGraph, error) {
	if !s.bound || !s.lock.valid || len(s.artifacts) != len(s.lock.packages) {
		return domain.LockedDependencyGraph{}, errPNPMRunBinding
	}
	artifacts := map[string]domain.ResolvedArtifact{}
	packages := map[string]PNPMLockedPackage{}
	for i, pkg := range s.lock.packages {
		packages[pkg.Key], artifacts[pkg.Key] = pkg, s.artifacts[i]
	}
	nodes := map[string]domain.LockedDependency{}
	paths := map[string]bool{}
	var all []domain.LockedDependency
	for _, instance := range s.lock.snapshots {
		pkg, present := packages[instance.Package]
		if !present {
			return domain.LockedDependencyGraph{}, errPNPMRunBinding
		}
		directory, err := PNPMInstanceDirectory(instance.Key)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		record := path.Join("node_modules/.pnpm", directory, "node_modules", pkg.Name)
		if paths[record] {
			return domain.LockedDependencyGraph{}, errPNPMRunBinding
		}
		paths[record] = true
		id, err := lockNodeID("pnpm:" + instance.Key)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		role := domain.DependencyTransitive
		if instance.Key == primary {
			role = domain.DependencyPrimary
		}
		node, err := domain.NewLockedDependencyWithRecordPath(id, role, artifacts[pkg.Key], record, pkg.Prepare)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		nodes[instance.Key] = node
		all = append(all, node)
	}
	var edges []domain.DependencyEdge
	for _, instance := range s.lock.snapshots {
		for _, dependency := range instance.Dependencies {
			edge, err := domain.NewDependencyEdge(nodes[instance.Key].Node(), nodes[dependency].Node())
			if err != nil {
				return domain.LockedDependencyGraph{}, err
			}
			edges = append(edges, edge)
		}
	}
	return domain.NewLockedDependencyGraph(all, edges)
}
