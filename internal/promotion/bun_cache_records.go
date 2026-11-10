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
	var placements []npmHoistedRecordPlacement
	for _, pkg := range lock.Packages() {
		placements = append(placements, npmHoistedRecordPlacement{pkg.Key, pkg.Name, pkg.Version, pkg.Targets})
	}
	nodes, edges, err := npmHoistedProjectRecordGraph(set.Inspected().Snapshot(), placements)
	if err != nil {
		return pnpmProjectRecords{}, err
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

type npmHoistedRecordPlacement struct {
	key, name, version string
	targets            map[string]string
}

func npmHoistedProjectRecordGraph(snapshot domain.ProjectDependencySnapshot, placements []npmHoistedRecordPlacement) ([]domain.LockedDependency, []domain.DependencyEdge, error) {
	unique := map[string]domain.DependencyNodeID{}
	instances := map[string]domain.DependencyNodeID{}
	var nodes []domain.LockedDependency
	for _, artifact := range snapshot.Dependencies() {
		key := artifact.Identity().Name() + "@" + artifact.Identity().Version()
		node, err := domain.ProjectDependencyNodeID(artifact.Identity())
		if err != nil || unique[key].String() != "" {
			return nil, nil, errors.Join(errors.New("hoisted project record package binding is incomplete"), err)
		}
		dependency, err := domain.NewLockedDependencyWithRecordPath(node, domain.DependencyTransitive, artifact, key, false)
		if err != nil {
			return nil, nil, err
		}
		unique[key] = node
		nodes = append(nodes, dependency)
	}
	for _, instance := range placements {
		node := unique[instance.name+"@"+instance.version]
		if node.String() == "" || instances[instance.key].String() != "" {
			return nil, nil, errors.New("hoisted project record instance package is missing or duplicated")
		}
		instances[instance.key] = node
	}
	var edges []domain.DependencyEdge
	seen := map[domain.DependencyEdge]bool{}
	for _, instance := range placements {
		for _, target := range instance.targets {
			from, to := instances[instance.key], instances[target]
			if to.String() == "" {
				return nil, nil, errors.New("hoisted project record instance edge is missing")
			}
			if from == to {
				continue
			}
			edge, err := domain.NewDependencyEdge(from, to)
			if err != nil {
				return nil, nil, err
			}
			if !seen[edge] {
				seen[edge] = true
				edges = append(edges, edge)
			}
		}
	}
	return nodes, edges, nil
}
