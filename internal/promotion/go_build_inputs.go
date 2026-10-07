package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// OpenBuildInputs reads only the independent retained approval held by the
// original project guard. It does not resolve, adopt a project, acquire data,
// or recover authority from project metadata or a cache pathname.
// The returned snapshot still requires offline graph revalidation before build.
func (g *approvedGoProjectGuard) OpenBuildInputs(ctx context.Context) (domain.ProjectDependencySnapshot, string, error) {
	if g == nil || len(g.originalState) == 0 {
		return domain.ProjectDependencySnapshot{}, "", errors.New("go build requires retained approved module cache; run go mod download first")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	if err := g.verifyRetainedApproval(ctx); err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	var doc goProjectApproval
	decoder := json.NewDecoder(bytes.NewReader(g.originalState))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return domain.ProjectDependencySnapshot{}, "", errors.New("go build retained approval is invalid")
	}
	graph, err := domain.NewSHA256Digest(doc.Approval.Graph)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	controls := make([]domain.ProjectControlDigest, 0, len(g.plan.controls))
	for _, file := range g.plan.controls {
		control, err := domain.NewProjectControlDigest(file.Name(), file.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		controls = append(controls, control)
	}
	dependencies := make([]domain.ResolvedArtifact, 0, len(doc.Approval.Entries))
	for _, entry := range doc.Approval.Entries {
		identity, err := domain.NewResolvedArtifactIdentity(artifactgo.Source(), entry.Module, entry.Version, "module")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		locator, err := artifactgo.ProxyURL(entry.Module, entry.Version, ".zip")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		dependency, err := domain.NewResolvedArtifact(identity, locator, entry.Integrity)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		dependencies = append(dependencies, dependency)
	}
	var snapshot domain.ProjectDependencySnapshot
	if doc.Approval.DependencyFree {
		snapshot, err = domain.NewDependencyFreeProjectSnapshot(g.context, artifactgo.Source(), controls, graph)
	} else {
		snapshot, err = domain.NewProjectDependencySnapshot(g.context, artifactgo.Source(), controls, dependencies, graph)
	}
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	digest, err := domain.NewSHA256Digest(doc.CacheDigest)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	cache, err := g.owner.cache.openProjectCacheDocument(ctx, doc.Cache, digest, doc.Approval)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	return snapshot, cache, nil
}
