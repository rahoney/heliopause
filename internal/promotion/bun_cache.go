package promotion

import (
	"context"
	"errors"
	"path/filepath"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// A completed layout is structural/observation handoff, never approval.
type BunLayoutProvider interface {
	PreparedLayoutFor(domain.ProjectDependencySnapshot, map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.BunInspectionLayout, error)
}

// BunProjectRecordBuilder retains explicit control absence in operation records.
type BunProjectRecordBuilder interface {
	BuildProjectWithControlPresence(context.Context, domain.OperationID, domain.ProjectVerifiedSet, []domain.LockedDependency, []domain.DependencyEdge, []domain.ProjectControlFile, string) (domain.ContentDigest, []byte, []byte, error)
}

// BunVerifiedCache publishes sealed regular data only; it never extracts links.
type BunVerifiedCache struct {
	intakeRoot, evidenceRoot, cacheRoot string
	frozen                              artifactnpm.BunInspectionContext
	evidence                            ProjectEvidenceReader
	layout                              BunLayoutProvider
	recordBuilder                       BunProjectRecordBuilder
	operation                           domain.OperationID
	beforeTreeSync                      func(string) error
}

func NewBunVerifiedCache(intake, evidence, cache string, frozen artifactnpm.BunInspectionContext, reader ProjectEvidenceReader, layout BunLayoutProvider, records BunProjectRecordBuilder, operation domain.OperationID) (*BunVerifiedCache, error) {
	roots := []string{intake, evidence, cache}
	if !frozen.Valid() || reader == nil || layout == nil || records == nil || operation.String() == "" || !separateRoots(append(roots, frozen.Snapshot().Context().Target().String())) {
		return nil, errors.New("bun cache requires frozen context, actual operation, records and separate trusted roots")
	}
	for _, root := range roots {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
			return nil, errors.New("bun cache roots must be canonical absolute paths")
		}
	}
	return &BunVerifiedCache{intakeRoot: intake, evidenceRoot: evidence, cacheRoot: cache, frozen: frozen, evidence: reader, layout: layout, recordBuilder: records, operation: operation}, nil
}

func (c *BunVerifiedCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
	doc, input, err := c.approval(ctx, set)
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	var archive []byte
	if !doc.DependencyFree {
		acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
		for _, entry := range set.Inspected().Inspections() {
			acquired[entry.Node()] = entry.Artifact()
		}
		layout, err := c.layout.PreparedLayoutFor(set.Inspected().Snapshot(), acquired)
		if err != nil || !layout.MatchesInput(input) || layout.Digest().String() != doc.Layout {
			return domain.StagedProjectSet{}, errors.Join(errors.New("bun completed layout differs from recorded approval"), err)
		}
		archive, err = renderNPMProjectCacheLayout(ctx, layout)
		if err != nil {
			return domain.StagedProjectSet{}, err
		}
	}
	if err := verifyNPMProjectCacheSources(ctx, c.intakeRoot, input.Artifacts()); err != nil {
		return domain.StagedProjectSet{}, err
	}
	material, err := c.projectRecords(ctx, set, doc)
	if err != nil {
		return domain.StagedProjectSet{}, err
	}
	doc.Manifest = material.manifest.String()
	if err := ctx.Err(); err != nil {
		return domain.StagedProjectSet{}, errors.Join(err, context.Cause(ctx))
	}
	var present []domain.ProjectControlFile
	for _, control := range c.frozen.Controls() {
		if control.Present() {
			present = append(present, control)
		}
	}
	return publishNPMProjectCache(ctx, set, doc, c.cacheRoot, present, archive, material, c.beforeTreeSync, npmProjectCacheBun)
}

func (c *BunVerifiedCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (string, error) {
	if !staged.Valid() {
		return "", errors.New("bun staged project is required")
	}
	expected, _, err := c.approval(ctx, staged.Set())
	if err != nil {
		return "", err
	}
	material, err := c.projectRecords(ctx, staged.Set(), expected)
	if err != nil {
		return "", err
	}
	expected.Manifest = material.manifest.String()
	return openNPMApprovedProjectCache(ctx, c.cacheRoot, staged.ContentHandle(), staged.Digest(), expected, &material, npmProjectCacheBun)
}
