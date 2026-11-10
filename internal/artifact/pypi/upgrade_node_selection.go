package pypi

import (
	"errors"
	"slices"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// UpgradeNodeSelection is one source-pinned pip --no-deps version choice.
// An empty delta retains a version, never its old approval or artifact bytes.
type UpgradeNodeSelection struct {
	reference domain.ArtifactReference
	changed   *Candidate
	extras    []string
}

func (s UpgradeNodeSelection) Reference() domain.ArtifactReference { return s.reference }

// ValidateFreshCandidate must precede demoting the single-report primary into
// its complete-graph role. Authentication and full new-set inspection remain
// separate existing responsibilities.
func (s UpgradeNodeSelection) ValidateFreshCandidate(candidate Candidate) error {
	project, err := RequestedProject(s.reference)
	version, present, versionErr := RequestedVersion(s.reference)
	if err != nil || versionErr != nil || !present || candidate.source != s.reference.Source() || candidate.project != project || candidate.version != version || !candidate.primary || !slices.Equal(candidate.requestedExtras, s.extras) {
		return errors.New("fresh source-pinned candidate differs from the node selection")
	}
	if s.changed != nil && !sameUpgradeCandidate(*s.changed, candidate) {
		return errors.New("source-pinned candidate bytes or metadata changed after version selection")
	}
	return nil
}

// ParseSourcePinnedNodeSelection validates a single-source no-deps delta for
// the named root or one canonically routed dependency. Whole-environment
// constraints/extras are checked on the complete proposal, not on this node.
func (u UpgradeInput) ParseSourcePinnedNodeSelection(requirement string, source domain.SourceID, extras []string, body []byte, pip string) (UpgradeNodeSelection, error) {
	if len(requirement) == 0 || len(requirement) > 1024 {
		return UpgradeNodeSelection{}, errors.New("bounded upgrade node requirement is required")
	}
	rootProfile, ok := ProfileForSource(u.requirement.source)
	if !ok || !IsPyTorchSource(u.requirement.source) {
		return UpgradeNodeSelection{}, errors.New("named upgrade input is required for source-pinned node selection")
	}
	project, err := DependencyProject(requirement)
	profile, supported := ProfileForSource(source)
	if err != nil || !supported || !sourceOwnsProject(rootProfile, source, project) || boundedExtraSet(extras) != nil {
		return UpgradeNodeSelection{}, errors.New("upgrade node source or requirement is unsupported")
	}
	canonicalExtras := append([]string(nil), extras...)
	sort.Strings(canonicalExtras)
	if !slices.Equal(extras, canonicalExtras) {
		return UpgradeNodeSelection{}, errors.New("upgrade node extras are not canonical")
	}
	var wanted UpgradeRequirement
	if project == u.requirement.project {
		if source != u.requirement.source || requirement != u.requirement.requirement || len(extras) != 0 {
			return UpgradeNodeSelection{}, errors.New("upgrade root intent changed during node selection")
		}
		wanted = u.requirement
	} else {
		bounded, parseErr := ParseBoundedRequirement(requirement)
		if parseErr != nil || bounded.Project() != project {
			return UpgradeNodeSelection{}, errors.New("upgrade dependency requirement is unsupported")
		}
		requested := bounded.Extras()
		sort.Strings(requested)
		if !slices.Equal(requested, extras) {
			return UpgradeNodeSelection{}, errors.New("upgrade dependency extra context differs from its requirement")
		}
		wanted, err = ParseUpgradeRequirementForSource(bounded.VersionConstraint(), source)
		if err != nil {
			return UpgradeNodeSelection{}, err
		}
	}
	report, err := parseUpgradeDeltaReport(body, pip, u.python)
	if err != nil {
		return UpgradeNodeSelection{}, err
	}
	if len(report.Install) > 1 {
		return UpgradeNodeSelection{}, errors.New("source-pinned no-deps selection contains additional candidates")
	}
	result := UpgradeNodeSelection{extras: canonicalExtras}
	version := ""
	if len(report.Install) == 1 {
		candidate, err := parseReportCandidate(report.Install[0], profile, u.python, extras)
		if err != nil || candidate.project != project || candidate.source != source || !candidate.primary {
			return UpgradeNodeSelection{}, errors.New("source-pinned no-deps candidate has a different identity")
		}
		version = candidate.version
		result.changed = &candidate
	} else {
		for _, installed := range u.installed {
			if installed.project == project {
				version = installed.version
				break
			}
		}
	}
	if version == "" || !wanted.AcceptsVersion(version) {
		return UpgradeNodeSelection{}, errors.New("source-pinned node did not select a supported requested version")
	}
	result.reference, err = ParseReferenceForSource(project+"@"+version, source)
	if err != nil {
		return UpgradeNodeSelection{}, err
	}
	return result, nil
}
