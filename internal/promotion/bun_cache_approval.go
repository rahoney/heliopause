package promotion

import (
	"context"
	"errors"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func (c *BunVerifiedCache) approval(ctx context.Context, set domain.ProjectVerifiedSet) (pnpmCacheDocument, artifactnpm.BunInspectionInput, error) {
	invalid := errors.New("bun staging approval or recorded instance coverage is invalid")
	if c == nil || ctx == nil || !set.Valid() || !c.frozen.MatchesSnapshot(set.Inspected().Snapshot()) {
		return pnpmCacheDocument{}, artifactnpm.BunInspectionInput{}, invalid
	}
	if err := ctx.Err(); err != nil {
		return pnpmCacheDocument{}, artifactnpm.BunInspectionInput{}, errors.Join(err, context.Cause(ctx))
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, inspection := range set.Inspected().Inspections() {
		subject := inspection.Artifact()
		if subject.ContentHandle() != "intake:"+inspection.RunID().String()+":tarball" {
			return pnpmCacheDocument{}, artifactnpm.BunInspectionInput{}, invalid
		}
		acquired[inspection.Node()] = subject
	}
	input, err := c.frozen.BindAcquired(set.Inspected().Snapshot(), acquired)
	if err != nil {
		return pnpmCacheDocument{}, input, err
	}
	doc, err := npmProjectCacheApprovalEvidence(ctx, set, c.operation, c.frozen.RuntimeIdentity(), c.frozen.Controls(), c.evidence, func(subject domain.AcquiredArtifact) ([]string, error) {
		instances, err := input.InstancesFor(subject)
		if err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(instances))
		for _, instance := range instances {
			keys = append(keys, instance.Key)
		}
		return keys, nil
	}, npmProjectCacheBun)
	return doc, input, err
}
