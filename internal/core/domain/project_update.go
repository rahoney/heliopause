package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

// ProjectControlFile is bounded opaque transaction content. Its digest binds
// bytes, not approval; ecosystem parsing remains inside adapters. Presence is
// explicit so an absent file cannot be confused with an existing empty file.
type ProjectControlFile struct {
	name    string
	body    []byte
	present bool
	digest  ContentDigest
}

func NewProjectControlFile(name string, body []byte, present bool) (ProjectControlFile, error) {
	if name == "" || len(name) > 128 || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00\r\n") || len(body) > 4<<20 || (!present && len(body) != 0) {
		return ProjectControlFile{}, errors.New("project control content is invalid")
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return ProjectControlFile{}, errors.New("project control name is invalid")
		}
	}
	hash := sha256.Sum256(body)
	digest, err := NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		return ProjectControlFile{}, err
	}
	return ProjectControlFile{name, append([]byte(nil), body...), present, digest}, nil
}

func (f ProjectControlFile) Name() string          { return f.name }
func (f ProjectControlFile) Body() []byte          { return append([]byte(nil), f.body...) }
func (f ProjectControlFile) Present() bool         { return f.present }
func (f ProjectControlFile) Digest() ContentDigest { return f.digest }

// ProjectDependencyUpdate freezes original controls and a complete selected
// project from one private selection. A requested-primary resolution is explicit;
// neither form grants Policy approval or publication authority.
type ProjectDependencyUpdate struct {
	original     []ProjectControlFile
	selected     []ProjectControlFile
	snapshot     ProjectDependencySnapshot
	resolution   DependencyResolution
	snapshotOnly bool
}

func NewProjectDependencyUpdate(original, selected []ProjectControlFile, snapshot ProjectDependencySnapshot, resolution DependencyResolution) (ProjectDependencyUpdate, error) {
	return newProjectDependencyUpdate(original, selected, snapshot, resolution, false)
}

// NewProjectDependencyUpdateFromSnapshot binds a complete-project selection
// without inventing a primary or relaxing the requested-primary constructor.
func NewProjectDependencyUpdateFromSnapshot(original, selected []ProjectControlFile, snapshot ProjectDependencySnapshot) (ProjectDependencyUpdate, error) {
	return newProjectDependencyUpdate(original, selected, snapshot, DependencyResolution{}, true)
}

func newProjectDependencyUpdate(original, selected []ProjectControlFile, snapshot ProjectDependencySnapshot, resolution DependencyResolution, snapshotOnly bool) (ProjectDependencyUpdate, error) {
	if !snapshot.Valid() || (!snapshotOnly && (resolution.LockfileDigest() != snapshot.GraphDigest() || len(resolution.Graph().Nodes()) == 0)) || (snapshotOnly && (resolution.LockfileDigest().String() != "" || len(resolution.Graph().Nodes()) != 0)) || len(original) == 0 || len(original) > 32 || len(original) != len(selected) || len(selected) != len(snapshot.ControlDigests()) {
		return ProjectDependencyUpdate{}, errors.New("project update requires matching complete selection")
	}
	before := append([]ProjectControlFile(nil), original...)
	after := append([]ProjectControlFile(nil), selected...)
	sort.Slice(before, func(i, j int) bool { return before[i].name < before[j].name })
	sort.Slice(after, func(i, j int) bool { return after[i].name < after[j].name })
	controls := snapshot.ControlDigests()
	for i := range before {
		if before[i].name == "" || before[i].digest.String() == "" || before[i].name != after[i].name || !after[i].present || after[i].name != controls[i].Name() || after[i].digest != controls[i].Digest() || (i != 0 && before[i-1].name == before[i].name) {
			return ProjectDependencyUpdate{}, errors.New("project update controls differ from selected snapshot")
		}
	}
	complete := map[ResolvedArtifactIdentity]ResolvedArtifact{}
	for _, artifact := range snapshot.Dependencies() {
		complete[artifact.Identity()] = artifact
	}
	for _, node := range resolution.Graph().Nodes() {
		artifact := node.Artifact()
		if selected, ok := complete[artifact.Identity()]; !ok || selected != artifact {
			return ProjectDependencyUpdate{}, errors.New("requested resolution differs from complete project")
		}
	}
	return ProjectDependencyUpdate{before, after, snapshot, resolution, snapshotOnly}, nil
}

func (u ProjectDependencyUpdate) Valid() bool {
	return u.snapshot.Valid() && len(u.original) != 0 && (u.snapshotOnly || u.resolution.LockfileDigest() == u.snapshot.GraphDigest())
}

func (u ProjectDependencyUpdate) HasRequestedResolution() bool { return u.Valid() && !u.snapshotOnly }
func (u ProjectDependencyUpdate) OriginalControls() []ProjectControlFile {
	return append([]ProjectControlFile(nil), u.original...)
}
func (u ProjectDependencyUpdate) SelectedControls() []ProjectControlFile {
	return append([]ProjectControlFile(nil), u.selected...)
}
func (u ProjectDependencyUpdate) Snapshot() ProjectDependencySnapshot { return u.snapshot }
func (u ProjectDependencyUpdate) Resolution() DependencyResolution    { return u.resolution }
