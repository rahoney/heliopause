package npm

import (
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
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
		run, err := npmAcquiredRun(artifact, resolved, mismatch)
		if !present || err != nil || runs[run] {
			return nil, mismatch
		}
		runs[run] = true
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func npmAcquiredRun(artifact domain.AcquiredArtifact, resolved domain.ResolvedArtifact, mismatch error) (domain.RunID, error) {
	declared, hasDeclared := artifact.DeclaredIntegrity()
	observed, hasObserved := artifact.ObservedIntegrity()
	parts := strings.Split(artifact.ContentHandle(), ":")
	if artifact.Identity() != resolved.Identity() || artifact.Digest().String() == "" || artifact.SizeBytes() == 0 ||
		!hasDeclared || !hasObserved || declared != resolved.DeclaredIntegrity() || observed != declared ||
		len(parts) != 3 || parts[0] != "intake" || parts[2] != "tarball" {
		return domain.RunID{}, mismatch
	}
	run, err := domain.ParseRunID(parts[1])
	if err != nil {
		return domain.RunID{}, mismatch
	}
	return run, nil
}

// bindNativeExecutionAcquired preserves all nodes, one original per identity and
// distinct intake runs for different identities. Primary selection is retained.
func bindNativeExecutionAcquired(graph domain.LockedDependencyGraph, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact, originals []domain.ResolvedArtifact, mismatch error) ([]domain.AcquiredArtifact, error) {
	byIdentity := map[domain.ResolvedArtifactIdentity]domain.AcquiredArtifact{}
	runs := map[domain.RunID]domain.ResolvedArtifactIdentity{}
	for _, node := range graph.Nodes() {
		subject, present := acquired[node.Node()]
		run, err := npmAcquiredRun(subject, node.Artifact(), mismatch)
		if !present || err != nil {
			return nil, mismatch
		}
		if identity, duplicate := runs[run]; duplicate && identity != subject.Identity() {
			return nil, mismatch
		}
		runs[run] = subject.Identity()
		previous, duplicate := byIdentity[subject.Identity()]
		if duplicate && (previous.Digest() != subject.Digest() || previous.SizeBytes() != subject.SizeBytes()) {
			return nil, mismatch
		}
		if !duplicate || node.Node() == graph.Primary() {
			byIdentity[subject.Identity()] = subject
		}
	}
	var subjects []domain.AcquiredArtifact
	for _, original := range originals {
		subject, present := byIdentity[original.Identity()]
		if !present {
			return nil, mismatch
		}
		subjects = append(subjects, subject)
	}
	return subjects, nil
}
