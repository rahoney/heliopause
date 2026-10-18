package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// UpgradeProposal is selection data, never a complete inspected set or ALLOW.
// A dry-run delta can legitimately omit the root and unchanged dependencies.
type UpgradeProposal struct {
	input       UpgradeInput
	reference   domain.ArtifactReference
	versions    map[string]string
	changed     []Candidate
	constraints []byte
}

func (u UpgradeInput) ParseUpgradeDelta(body []byte, pip string) (UpgradeProposal, error) {
	if u.requirement.project == "" || u.python == "" || pip == "" || len(body) == 0 || len(body) > maxReportBytes {
		return UpgradeProposal{}, errors.New("bounded upgrade delta and frozen input are required")
	}
	var report pipReport
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(&report) != nil || ensureSingleJSONValue(decoder) != nil || report.Version != "1" || report.PipVersion != pip || report.Environment.ImplementationName != "cpython" || report.Environment.ImplementationVersion != u.python || report.Environment.PythonFullVersion != u.python || report.Environment.PlatformMachine != "x86_64" || report.Environment.SysPlatform != "linux" || report.Install == nil || len(report.Install) > maxPyPIReportEntries {
		return UpgradeProposal{}, errors.New("upgrade delta runtime or bounds are invalid")
	}
	proposal := UpgradeProposal{input: u, versions: map[string]string{}}
	for _, project := range u.installed {
		proposal.versions[project.project] = project.version
	}
	seen := map[string]bool{}
	for _, item := range report.Install {
		candidate, err := parseReportCandidate(item, PublicPyPIProfile(), u.python, nil)
		if err != nil || seen[candidate.project] || candidate.primary != (candidate.project == u.requirement.project) {
			return UpgradeProposal{}, errors.New("upgrade delta contains an invalid or ambiguous candidate")
		}
		seen[candidate.project] = true
		proposal.versions[candidate.project] = candidate.version
		proposal.changed = append(proposal.changed, candidate)
	}
	version, present := proposal.versions[u.requirement.project]
	if !present || !u.requirement.AcceptsVersion(version) || len(proposal.versions) > maxPyPIReportEntries {
		return UpgradeProposal{}, errors.New("upgrade delta does not select the requested supported root")
	}
	reference, err := ParseReference(u.requirement.project + "@" + version)
	if err != nil {
		return UpgradeProposal{}, err
	}
	proposal.reference = reference
	names := make([]string, 0, len(proposal.versions))
	for name := range proposal.versions {
		names = append(names, name)
	}
	sort.Strings(names)
	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, name+"=="+proposal.versions[name])
	}
	for _, project := range u.installed {
		if project.version == proposal.versions[project.project] {
			lines = append(lines, project.requirements...)
		}
	}
	proposal.constraints = []byte(strings.Join(lines, "\n") + "\n")
	if len(proposal.constraints) > upgradeInputLimit {
		return UpgradeProposal{}, errors.New("retained upgrade constraints exceed bounds")
	}
	return proposal, nil
}

func (p UpgradeProposal) Reference() domain.ArtifactReference { return p.reference }
func (p UpgradeProposal) Constraints() []byte {
	return append([]byte(nil), p.constraints...)
}

// ValidateFinal binds the existing complete single-root report to the
// only-if-needed proposal. Unrelated installed packages are constraints, not
// fabricated graph roots. Final source cross-checking remains mandatory.
func (p UpgradeProposal) ValidateFinal(report InstallationReport) error {
	if p.reference.Source().String() != "pypi" || len(report.candidates) == 0 {
		return errors.New("complete public upgrade selection is required")
	}
	selected := map[string]Candidate{}
	primary := 0
	for _, candidate := range report.candidates {
		if _, duplicate := selected[candidate.project]; duplicate || candidate.source != p.reference.Source() || p.versions[candidate.project] != candidate.version {
			return errors.New("final upgrade candidate differs from frozen version selection")
		}
		if candidate.primary {
			primary++
			if candidate.project != p.input.requirement.project || !p.input.requirement.AcceptsVersion(candidate.version) {
				return errors.New("final upgrade root differs from requested selection")
			}
		}
		selected[candidate.project] = candidate
	}
	if primary != 1 {
		return errors.New("final upgrade selection requires exactly the requested root")
	}
	for _, changed := range p.changed {
		candidate, present := selected[changed.project]
		if !present || changed.source != candidate.source || changed.project != candidate.project || changed.version != candidate.version || changed.filename != candidate.filename || changed.url != candidate.url || changed.sha256 != candidate.sha256 || changed.requiresPython != candidate.requiresPython || changed.primary != candidate.primary || !slices.Equal(changed.dependencies, candidate.dependencies) || !slices.Equal(changed.requirements, candidate.requirements) {
			return errors.New("upgrade changed candidate was omitted or substituted between selections")
		}
	}
	return nil
}
