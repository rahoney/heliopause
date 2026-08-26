package domain

import (
	"errors"
	"sort"
)

// ProjectControlDigest binds one project control file to a dependency snapshot.
type ProjectControlDigest struct {
	name   string
	digest ContentDigest
}

func NewProjectControlDigest(name string, digest ContentDigest) (ProjectControlDigest, error) {
	if name == "" || digest.String() == "" {
		return ProjectControlDigest{}, errors.New("valid project control digest is required")
	}
	return ProjectControlDigest{name: name, digest: digest}, nil
}

func (d ProjectControlDigest) Name() string          { return d.name }
func (d ProjectControlDigest) Digest() ContentDigest { return d.digest }

// ProjectDependencySnapshot freezes a complete project state without
// fabricating a primary artifact. This is deliberately distinct from
// LockedDependencyGraph, which requires exactly one primary artifact.
type ProjectDependencySnapshot struct {
	context      InstallContext
	source       SourceID
	controls     []ProjectControlDigest
	dependencies []ResolvedArtifact
	graphDigest  ContentDigest
}

func NewProjectDependencySnapshot(context InstallContext, source SourceID, controls []ProjectControlDigest, dependencies []ResolvedArtifact, graphDigest ContentDigest) (ProjectDependencySnapshot, error) {
	if !context.Valid() || source.String() == "" || len(controls) == 0 || len(dependencies) == 0 || graphDigest.String() == "" {
		return ProjectDependencySnapshot{}, errors.New("valid project dependency snapshot is required")
	}
	controlCopy := append([]ProjectControlDigest(nil), controls...)
	sort.Slice(controlCopy, func(i, j int) bool { return controlCopy[i].name < controlCopy[j].name })
	for index, control := range controlCopy {
		if control.name == "" || control.digest.String() == "" || (index != 0 && controlCopy[index-1].name == control.name) {
			return ProjectDependencySnapshot{}, errors.New("project controls are invalid")
		}
	}
	dependencyCopy := append([]ResolvedArtifact(nil), dependencies...)
	sort.Slice(dependencyCopy, func(i, j int) bool {
		left, right := dependencyCopy[i].Identity(), dependencyCopy[j].Identity()
		return left.Name()+"@"+left.Version()+"#"+left.Variant() < right.Name()+"@"+right.Version()+"#"+right.Variant()
	})
	for index, dependency := range dependencyCopy {
		identity := dependency.Identity()
		if identity.Source().String() == "" || identity.Source() != source || (index != 0 && dependencyCopy[index-1].Identity() == identity) {
			return ProjectDependencySnapshot{}, errors.New("project dependencies are invalid")
		}
	}
	return ProjectDependencySnapshot{context: context, source: source, controls: controlCopy, dependencies: dependencyCopy, graphDigest: graphDigest}, nil
}

func (s ProjectDependencySnapshot) Valid() bool {
	return s.context.Valid() && s.source.String() != "" && len(s.controls) != 0 && len(s.dependencies) != 0 && s.graphDigest.String() != ""
}
func (s ProjectDependencySnapshot) Context() InstallContext { return s.context }
func (s ProjectDependencySnapshot) Source() SourceID        { return s.source }
func (s ProjectDependencySnapshot) ControlDigests() []ProjectControlDigest {
	return append([]ProjectControlDigest(nil), s.controls...)
}
func (s ProjectDependencySnapshot) Dependencies() []ResolvedArtifact {
	return append([]ResolvedArtifact(nil), s.dependencies...)
}
func (s ProjectDependencySnapshot) GraphDigest() ContentDigest { return s.graphDigest }
