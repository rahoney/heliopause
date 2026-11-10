package promotion

import (
	"context"
	"encoding/json"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const pnpmProjectRecordLimit = 32 << 20
const pnpmProjectRecordsLimit = 2*pnpmProjectRecordLimit + pnpmCacheReceiptLimit

type pnpmProjectRecords struct {
	manifest                   domain.ContentDigest
	document, sbom, provenance []byte
}

// Generic records describe unique archives. This separate native projection
// retains every peer instance and root selector without inventing a primary.
type pnpmProjectProvenance struct {
	Schema, Operation, Manifest, Graph, Runtime, Layout string
	Roots                                               []artifactnpm.PNPMLockedRoot
	Packages                                            []artifactnpm.PNPMLockedPackage
	Instances                                           []artifactnpm.PNPMLockedSnapshot
}

func (c *PNPMVerifiedCache) projectRecords(ctx context.Context, set domain.ProjectVerifiedSet, approval pnpmCacheDocument) (pnpmProjectRecords, error) {
	if c.recordBuilder == nil || c.operation.String() == "" {
		return pnpmProjectRecords{}, errors.New("pnpm project records require their actual operation and builder")
	}
	lock := c.frozen.Lock()
	packages := map[string]domain.DependencyNodeID{}
	keys := map[string]string{}
	for _, pkg := range lock.Packages() {
		keys[pkg.Name+"@"+pkg.Version] = pkg.Key
	}
	var nodes []domain.LockedDependency
	for _, artifact := range set.Inspected().Snapshot().Dependencies() {
		key := keys[artifact.Identity().Name()+"@"+artifact.Identity().Version()]
		node, err := domain.ProjectDependencyNodeID(artifact.Identity())
		if err != nil || key == "" || packages[key].String() != "" {
			return pnpmProjectRecords{}, errors.Join(errors.New("pnpm record package binding is incomplete"), err)
		}
		dependency, err := domain.NewLockedDependencyWithRecordPath(node, domain.DependencyTransitive, artifact, key, false)
		if err != nil {
			return pnpmProjectRecords{}, err
		}
		packages[key] = node
		nodes = append(nodes, dependency)
	}
	instances := map[string]domain.DependencyNodeID{}
	for _, instance := range lock.Snapshots() {
		node := packages[instance.Package]
		if node.String() == "" {
			return pnpmProjectRecords{}, errors.New("pnpm record instance package is missing")
		}
		instances[instance.Key] = node
	}
	var edges []domain.DependencyEdge
	seen := map[domain.DependencyEdge]bool{}
	for _, instance := range lock.Snapshots() {
		for _, target := range instance.Dependencies {
			from, to := instances[instance.Key], instances[target]
			if to.String() == "" {
				return pnpmProjectRecords{}, errors.New("pnpm record instance edge is missing")
			}
			if from == to {
				continue // Full instance edges, including this one, remain in provenance.
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
	digest, document, sbom, err := c.recordBuilder.BuildProject(ctx, c.operation, set, nodes, edges, c.frozen.Controls(), c.frozen.RuntimeIdentity())
	if err != nil {
		return pnpmProjectRecords{}, err
	}
	if digest.String() == "" || len(document) == 0 || len(document) > pnpmProjectRecordLimit || len(sbom) == 0 || len(sbom) > pnpmProjectRecordLimit || !json.Valid(document) || !json.Valid(sbom) {
		return pnpmProjectRecords{}, errors.New("pnpm project records are invalid or exceed fixed bounds")
	}
	provenance, err := json.Marshal(pnpmProjectProvenance{"helox.pnpm-project-provenance/v1", c.operation.String(), digest.String(), approval.Graph, approval.Runtime, approval.Layout, lock.Roots(), lock.Packages(), lock.Snapshots()})
	if err != nil || len(provenance) > pnpmCacheReceiptLimit {
		return pnpmProjectRecords{}, errors.Join(errors.New("pnpm project provenance exceeds its fixed bound"), err)
	}
	return pnpmProjectRecords{digest, document, sbom, provenance}, nil
}
