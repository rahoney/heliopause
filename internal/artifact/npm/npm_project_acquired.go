package npm

import (
	"github.com/rahoney/heliopause/internal/core/domain"
	"strings"
)

// bindNPMProjectAcquired preserves exact unique archive/run coverage. It grants
// neither authenticity beyond measured SRI equality nor inspection/Policy approval.
func bindNPMProjectAcquired(snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact, mismatch error) ([]domain.AcquiredArtifact, error) {
	artifacts := make([]domain.AcquiredArtifact, 0, len(acquired))
	runs := map[domain.RunID]bool{}
	for _, resolved := range snapshot.Dependencies() {
		node, err := domain.ProjectDependencyNodeID(resolved.Identity())
		if err != nil {
			return nil, mismatch
		}
		artifact, present := acquired[node]
		declared, hasDeclared := artifact.DeclaredIntegrity()
		observed, hasObserved := artifact.ObservedIntegrity()
		parts := strings.Split(artifact.ContentHandle(), ":")
		if !present || artifact.Identity() != resolved.Identity() || artifact.Digest().String() == "" || artifact.SizeBytes() == 0 ||
			!hasDeclared || !hasObserved || declared != resolved.DeclaredIntegrity() || observed != declared ||
			len(parts) != 3 || parts[0] != "intake" || parts[2] != "tarball" {
			return nil, mismatch
		}
		run, err := domain.ParseRunID(parts[1])
		if err != nil || runs[run] {
			return nil, mismatch
		}
		runs[run] = true
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}
