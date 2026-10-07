package domain

import (
	"errors"
	"strings"
)

// StagedProjectSet binds an infrastructure cache receipt to the full approved
// project snapshot. A path or artifact-created cache marker is never enough.
type StagedProjectSet struct {
	set    ProjectVerifiedSet
	handle string
	digest ContentDigest
}

func NewStagedProjectSet(set ProjectVerifiedSet, handle string, digest ContentDigest) (StagedProjectSet, error) {
	if !set.Valid() || digest.String() == "" {
		return StagedProjectSet{}, errors.New("staged project cache requires complete approval and receipt digest")
	}
	id, ok := strings.CutPrefix(handle, "project-cache:")
	if !ok {
		return StagedProjectSet{}, errors.New("project cache handle is invalid")
	}
	if _, err := ParseRunID(id); err != nil {
		return StagedProjectSet{}, errors.New("project cache handle is invalid")
	}
	return StagedProjectSet{set, handle, digest}, nil
}
func (s StagedProjectSet) Valid() bool {
	return s.set.Valid() && s.handle != "" && s.digest.String() != ""
}
func (s StagedProjectSet) Set() ProjectVerifiedSet { return s.set }
func (s StagedProjectSet) ContentHandle() string   { return s.handle }
func (s StagedProjectSet) Digest() ContentDigest   { return s.digest }
