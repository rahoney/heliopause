package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type NPMProjectLayoutProvider interface {
	PreparedLayoutFor(domain.ProjectDependencySnapshot, map[domain.DependencyNodeID]domain.AcquiredArtifact) (artifactnpm.NPMProjectInspectionLayout, error)
}

// The typed owner binds npm's complete selection to the existing authenticated
// cache and private tree engines. Native controls/layout alone never approve it.
type NPMProjectVerifiedCache struct {
	intakeRoot, evidenceRoot, cacheRoot string
	frozen                              artifactnpm.NPMProjectInspectionContext
	evidence                            ProjectEvidenceReader
	layout                              NPMProjectLayoutProvider
	recordBuilder                       NPMProjectRecordBuilder
	operation                           domain.OperationID
	beforeTreeSync                      func(string) error
	beforePrepareSync                   func(*PNPMPreparedProject) error
}

func NewNPMProjectVerifiedCache(intake, evidence, cache string, frozen artifactnpm.NPMProjectInspectionContext, reader ProjectEvidenceReader, layout NPMProjectLayoutProvider, records NPMProjectRecordBuilder, operation domain.OperationID) (*NPMProjectVerifiedCache, error) {
	roots := []string{intake, evidence, cache}
	if !frozen.Valid() || reader == nil || layout == nil || records == nil || operation.String() == "" || !separateRoots(append(roots, frozen.Snapshot().Context().Target().String())) {
		return nil, errors.New("npm project cache requires exact context, actual operation, records and separate roots")
	}
	for _, root := range roots {
		if root == "/" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return nil, errors.New("npm project cache roots must be canonical")
		}
	}
	return &NPMProjectVerifiedCache{intakeRoot: intake, evidenceRoot: evidence, cacheRoot: cache, frozen: frozen, evidence: reader, layout: layout, recordBuilder: records, operation: operation}, nil
}

func (c *NPMProjectVerifiedCache) approval(ctx context.Context, set domain.ProjectVerifiedSet) (pnpmCacheDocument, artifactnpm.NPMProjectInspectionInput, error) {
	invalid := errors.New("npm project staging lacks exact selection or recorded complete inspection")
	if c == nil || ctx == nil || !set.Valid() || !c.frozen.MatchesSnapshot(set.Inspected().Snapshot()) {
		return pnpmCacheDocument{}, artifactnpm.NPMProjectInspectionInput{}, invalid
	}
	if err := ctx.Err(); err != nil {
		return pnpmCacheDocument{}, artifactnpm.NPMProjectInspectionInput{}, errors.Join(err, context.Cause(ctx))
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, inspection := range set.Inspected().Inspections() {
		subject := inspection.Artifact()
		if subject.ContentHandle() != "intake:"+inspection.RunID().String()+":tarball" {
			return pnpmCacheDocument{}, artifactnpm.NPMProjectInspectionInput{}, invalid
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
			keys = append(keys, instance.Location)
		}
		return keys, nil
	}, npmProjectCacheNPM)
	return doc, input, err
}

func (c *NPMProjectVerifiedCache) StageProject(ctx context.Context, set domain.ProjectVerifiedSet) (domain.StagedProjectSet, error) {
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
			return domain.StagedProjectSet{}, errors.Join(errors.New("npm project layout differs from recorded approval"), err)
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
	return publishNPMProjectCache(ctx, set, doc, c.cacheRoot, c.frozen.Controls(), archive, material, c.beforeTreeSync, npmProjectCacheNPM)
}

func (c *NPMProjectVerifiedCache) OpenProjectCache(ctx context.Context, staged domain.StagedProjectSet) (string, error) {
	if !staged.Valid() {
		return "", errors.New("npm project staged approval is required")
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
	return openNPMApprovedProjectCache(ctx, c.cacheRoot, staged.ContentHandle(), staged.Digest(), expected, &material, npmProjectCacheNPM)
}

func (c *NPMProjectVerifiedCache) PrepareProject(ctx context.Context, staged domain.StagedProjectSet) (*PNPMPreparedProject, error) {
	data, err := c.OpenProjectCache(ctx, staged)
	if err != nil {
		return nil, err
	}
	entries, err := npmProjectHostEntries(ctx, data, staged.Digest(), c.frozen.Controls(), func(doc pnpmCacheDocument) (pnpmProjectRecords, error) {
		return c.projectRecords(ctx, staged.Set(), doc)
	}, npmProjectCacheNPM)
	if err != nil {
		return nil, err
	}
	return prepareNPMProject(ctx, c.frozen.Snapshot().Context().Target().String(), entries, c.beforePrepareSync, npmProjectCacheNPM)
}

func (c *NPMProjectVerifiedCache) projectRecords(ctx context.Context, set domain.ProjectVerifiedSet, approval pnpmCacheDocument) (pnpmProjectRecords, error) {
	if c == nil || c.recordBuilder == nil || c.operation.String() == "" {
		return pnpmProjectRecords{}, errors.New("npm project records require their operation and builder")
	}
	lock := c.frozen.Lock()
	var placements []npmHoistedRecordPlacement
	for _, pkg := range lock.Packages() {
		placements = append(placements, npmHoistedRecordPlacement{pkg.Location, pkg.Name, pkg.Version, pkg.Targets})
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
		return pnpmProjectRecords{}, errors.New("npm project records are invalid or exceed fixed bounds")
	}
	provenance, err := json.Marshal(struct {
		Schema, Operation, Manifest, Graph, Runtime, Layout string
		Roots                                               []artifactnpm.NPMProjectRoot
		Instances                                           []artifactnpm.NPMProjectPackage
	}{"helox.npm-project-provenance/v1", c.operation.String(), digest.String(), approval.Graph, approval.Runtime, approval.Layout, lock.Roots(), lock.Packages()})
	if err != nil || len(provenance) > pnpmCacheReceiptLimit {
		return pnpmProjectRecords{}, errors.Join(errors.New("npm project provenance exceeds fixed bound"), err)
	}
	return pnpmProjectRecords{digest, document, sbom, provenance}, nil
}

// The npm adapter uses the existing four-member guarded transaction; no native
// adoption verifier is supplied. Edited controls require independently owned old
// payload and approval, as enforced by BeginUpgrade's existing guard.
func NewNPMProjectPromotion(intake, evidence, cache, state string, reader ProjectEvidenceReader, operation domain.OperationID, selected func(domain.ProjectDependencySnapshot) (*NPMProjectVerifiedCache, error)) (*PNPMProjectPromotion, error) {
	roots := []string{intake, evidence, cache, state}
	if reader == nil || operation.String() == "" || selected == nil || !separateRoots(roots) {
		return nil, errors.New("npm project mutation requires separate roots, operation and selected cache")
	}
	for _, root := range roots {
		if root == "/" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return nil, errors.New("npm project mutation roots must be canonical")
		}
	}
	return &PNPMProjectPromotion{client: npmProjectCacheNPM, intakeRoot: intake, evidenceRoot: evidence, cacheRoot: cache, stateRoot: state, evidence: reader, operation: operation, selectedNPMCache: selected}, nil
}
