package promotion

import (
	"context"
	"encoding/json"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Unique archives feed generic records; every physical placement stays here.
type bunProjectProvenance struct {
	Schema, Operation, Manifest, Graph, Runtime, Layout string
	Roots                                               []artifactnpm.BunLockedRoot
	Instances                                           []artifactnpm.BunLockedPackage
}

func (c *BunVerifiedCache) projectRecords(ctx context.Context, set domain.ProjectVerifiedSet, approval pnpmCacheDocument) (pnpmProjectRecords, error) {
	if c.recordBuilder == nil || c.operation.String() == "" {
		return pnpmProjectRecords{}, errors.New("bun project records require actual operation and builder")
	}
	lock := c.frozen.Lock()
	unique := map[string]domain.DependencyNodeID{}
	instances := map[string]domain.DependencyNodeID{}
	var nodes []domain.LockedDependency
	for _, artifact := range set.Inspected().Snapshot().Dependencies() {
		key := artifact.Identity().Name() + "@" + artifact.Identity().Version()
		node, err := domain.ProjectDependencyNodeID(artifact.Identity())
		if err != nil || unique[key].String() != "" {
			return pnpmProjectRecords{}, errors.Join(errors.New("bun record package binding is incomplete"), err)
		}
		dependency, err := domain.NewLockedDependencyWithRecordPath(node, domain.DependencyTransitive, artifact, key, false)
		if err != nil {
			return pnpmProjectRecords{}, err
		}
		unique[key] = node
		nodes = append(nodes, dependency)
	}
	for _, instance := range lock.Packages() {
		node := unique[instance.Name+"@"+instance.Version]
		if node.String() == "" || instances[instance.Key].String() != "" {
			return pnpmProjectRecords{}, errors.New("bun record instance package is missing or duplicated")
		}
		instances[instance.Key] = node
	}
	var edges []domain.DependencyEdge
	seen := map[domain.DependencyEdge]bool{}
	for _, instance := range lock.Packages() {
		for _, target := range instance.Targets {
			from, to := instances[instance.Key], instances[target]
			if to.String() == "" {
				return pnpmProjectRecords{}, errors.New("bun record instance edge is missing")
			}
			if from == to {
				continue
			}
			edge, err := domain.NewDependencyEdge(from, to)
			if err != nil {
				return pnpmProjectRecords{}, err
			}
			if !seen[edge] {
				seen[edge] = true
				edges = append(edges, edge)
			}
		}
	}
	digest, document, sbom, err := c.recordBuilder.BuildProjectWithControlPresence(ctx, c.operation, set, nodes, edges, c.frozen.Controls(), c.frozen.RuntimeIdentity())
	if err != nil {
		return pnpmProjectRecords{}, err
	}
	if digest.String() == "" || len(document) == 0 || len(document) > pnpmProjectRecordLimit || len(sbom) == 0 || len(sbom) > pnpmProjectRecordLimit || !json.Valid(document) || !json.Valid(sbom) {
		return pnpmProjectRecords{}, errors.New("bun project records are invalid or exceed fixed bounds")
	}
	provenance, err := json.Marshal(bunProjectProvenance{"helox.bun-project-provenance/v1", c.operation.String(), digest.String(), approval.Graph, approval.Runtime, approval.Layout, lock.Roots(), lock.Packages()})
	if err != nil || len(provenance) > pnpmCacheReceiptLimit {
		return pnpmProjectRecords{}, errors.Join(errors.New("bun project provenance exceeds fixed bound"), err)
	}
	return pnpmProjectRecords{digest, document, sbom, provenance}, nil
}
