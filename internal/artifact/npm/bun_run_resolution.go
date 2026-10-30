package npm

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunRunBinding = errors.New("bun one-off resolution differs from its request, native instances or independent source")

// BindBunRunCache authenticates unique originals and retains every native tree
// placement and edge in an execution graph. It creates no installation target,
// inspection result or permission to execute cache payloads.
func (r *Resolver) BindBunRunCache(ctx context.Context, cache BunRunCache, reference domain.ArtifactReference, runtimeIdentity string) (frozen BunInspectionContext, resultErr error) {
	if ctx == nil {
		return frozen, errBunRunBinding
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			frozen = BunInspectionContext{}
		}
	}()
	if ctx.Err() != nil || runtimeIdentity == "" || cache.root == "" || len(cache.controls) != 2 {
		return frozen, errBunRunBinding
	}
	primary, err := cache.lock.runPrimary(reference)
	if err != nil {
		return frozen, err
	}
	selected, err := r.BindBunLock(ctx, cache.lock)
	if err != nil {
		return frozen, err
	}
	graph, err := selected.runGraph(primary)
	if err != nil {
		return frozen, err
	}
	resolution, err := nativeNodeRunResolution(graph, reference, runtimeIdentity, cache.controls, "bunx")
	if err != nil {
		return frozen, err
	}
	return BunInspectionContext{selection: selected, controls: append([]domain.ProjectControlFile(nil), cache.controls...), runtime: runtimeIdentity, execution: resolution}, nil
}

func (s BunSelection) runGraph(primary string) (domain.LockedDependencyGraph, error) {
	if !s.bound || !s.lock.valid || !s.completeBunArtifacts() {
		return domain.LockedDependencyGraph{}, errBunRunBinding
	}
	originals := map[string]domain.ResolvedArtifact{}
	for _, artifact := range s.artifacts {
		originals[artifact.Identity().Name()+"@"+artifact.Identity().Version()] = artifact
	}
	nodes := map[string]domain.LockedDependency{}
	paths := map[string]bool{}
	var all []domain.LockedDependency
	for _, instance := range s.lock.packages {
		record, err := bunInstanceRoot(instance.Key)
		if err != nil || paths[record] {
			return domain.LockedDependencyGraph{}, errors.Join(errBunRunBinding, err)
		}
		paths[record] = true
		id, err := lockNodeID("bun:" + instance.Key)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		role := domain.DependencyTransitive
		if instance.Key == primary {
			role = domain.DependencyPrimary
		}
		node, err := domain.NewLockedDependencyWithRecordPath(id, role, originals[instance.Name+"@"+instance.Version], record, instance.HasInstallScript)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		nodes[instance.Key] = node
		all = append(all, node)
	}
	var edges []domain.DependencyEdge
	for _, instance := range s.lock.packages {
		for _, target := range instance.Targets {
			edge, err := domain.NewDependencyEdge(nodes[instance.Key].Node(), nodes[target].Node())
			if err != nil {
				return domain.LockedDependencyGraph{}, err
			}
			edges = append(edges, edge)
		}
	}
	return domain.NewLockedDependencyGraph(all, edges)
}
