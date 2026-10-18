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
	input          UpgradeInput
	reference      domain.ArtifactReference
	versions       map[string]string
	changed        []Candidate
	constraints    []byte
	requiredExtras map[string][]string
}

func (u UpgradeInput) ParseUpgradeDelta(body []byte, pip string) (UpgradeProposal, error) {
	profile, supported := ProfileForSource(u.requirement.Reference().Source())
	if !supported || u.python == "" || pip == "" || len(body) == 0 || len(body) > maxReportBytes {
		return UpgradeProposal{}, errors.New("bounded upgrade delta and frozen input are required")
	}
	report, err := parseUpgradeDeltaReport(body, pip, u.python)
	if err != nil {
		return UpgradeProposal{}, err
	}
	candidates := make([]Candidate, 0, len(report.Install))
	for _, item := range report.Install {
		candidate, err := parseReportCandidate(item, profile, u.python, nil)
		if err != nil {
			return UpgradeProposal{}, errors.New("upgrade delta contains an invalid or ambiguous candidate")
		}
		candidates = append(candidates, candidate)
	}
	return u.proposeUpgrade(candidates)
}

// ProposeSourcePinnedUpgrade uses the complete first-pass named traversal,
// including freshly authenticated unchanged versions. It is selection data,
// never a prior inspection result or permission to publish.
func (u UpgradeInput) ProposeSourcePinnedUpgrade(candidates []Candidate) (UpgradeProposal, error) {
	if !IsPyTorchSource(u.requirement.source) {
		return UpgradeProposal{}, errors.New("source-pinned upgrade requires a named profile")
	}
	if _, err := BuildLockedGraph(u.requirement.Reference(), candidates); err != nil {
		return UpgradeProposal{}, err
	}
	proposal, err := u.proposeUpgrade(candidates)
	if err != nil {
		return UpgradeProposal{}, err
	}
	for _, candidate := range candidates {
		if !slices.Equal(candidate.requestedExtras, proposal.requiredExtras[candidate.project]) {
			return UpgradeProposal{}, errors.New("source-pinned first selection omitted or added upgrade extras")
		}
	}
	return proposal, nil
}

func (u UpgradeInput) proposeUpgrade(candidates []Candidate) (UpgradeProposal, error) {
	profile, supported := ProfileForSource(u.requirement.source)
	if !supported || len(candidates) > maxPyPIReportEntries {
		return UpgradeProposal{}, errors.New("bounded source-bound upgrade candidates are required")
	}
	proposal := UpgradeProposal{input: u, versions: map[string]string{}}
	for _, project := range u.installed {
		proposal.versions[project.project] = project.version
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if seen[candidate.project] || !sourceOwnsProject(profile, candidate.source, candidate.project) || candidate.primary != (candidate.project == u.requirement.project) {
			return UpgradeProposal{}, errors.New("upgrade selection contains an invalid or ambiguous candidate")
		}
		seen[candidate.project] = true
		proposal.versions[candidate.project] = candidate.version
		proposal.changed = append(proposal.changed, candidate)
	}
	version, present := proposal.versions[u.requirement.project]
	if !present || !u.requirement.AcceptsVersion(version) || len(proposal.versions) > maxPyPIReportEntries {
		return UpgradeProposal{}, errors.New("upgrade delta does not select the requested supported root")
	}
	reference, err := ParseReferenceForSource(u.requirement.project+"@"+version, profile.Source())
	if err != nil {
		return UpgradeProposal{}, err
	}
	proposal.reference = reference
	proposal.requiredExtras, err = u.upgradeExtraObligations(proposal.versions, candidates)
	if err != nil {
		return UpgradeProposal{}, err
	}
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
			retained := project
			if extras := proposal.requiredExtras[project.project]; len(extras) > 0 {
				retained, err = parseUpgradeInstalledMetadataWithExtras(InstalledMetadataFile{project.directory, project.metadata}, u.python, extras)
				if err != nil {
					return UpgradeProposal{}, err
				}
			}
			lines = append(lines, retained.requirements...)
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

// ValidateSourcePinnedFinal binds the fresh second traversal without turning
// unrelated installed distributions into graph roots.
func (p UpgradeProposal) ValidateSourcePinnedFinal(candidates []Candidate) error {
	if !IsPyTorchSource(p.reference.Source()) {
		return errors.New("source-pinned upgrade requires a named profile")
	}
	if _, err := BuildLockedGraph(p.reference, candidates); err != nil {
		return err
	}
	return p.ValidateFinal(InstallationReport{candidates: candidates})
}

// ValidateFinal binds the existing complete single-root report to the
// only-if-needed proposal. Unrelated installed packages are constraints, not
// fabricated graph roots. Final source cross-checking remains mandatory.
func (p UpgradeProposal) ValidateFinal(report InstallationReport) error {
	profile, supported := ProfileForSource(p.reference.Source())
	if !supported || len(report.candidates) == 0 {
		return errors.New("complete source-bound upgrade selection is required")
	}
	selected := map[string]Candidate{}
	primary := 0
	for _, candidate := range report.candidates {
		if _, duplicate := selected[candidate.project]; duplicate || !sourceOwnsProject(profile, candidate.source, candidate.project) || p.versions[candidate.project] != candidate.version {
			return errors.New("final upgrade candidate differs from frozen version selection")
		}
		if !slices.Equal(candidate.requestedExtras, p.requiredExtras[candidate.project]) {
			return errors.New("final upgrade candidate omitted or added required extras")
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
		if !present || !sameUpgradeCandidate(changed, candidate) {
			return errors.New("upgrade changed candidate was omitted or substituted between selections")
		}
	}
	return nil
}

func parseUpgradeDeltaReport(body []byte, pip, python string) (pipReport, error) {
	if len(body) == 0 || len(body) > maxReportBytes || pip == "" || python == "" {
		return pipReport{}, errors.New("bounded upgrade delta and runtime are required")
	}
	var report pipReport
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(&report) != nil || ensureSingleJSONValue(decoder) != nil || report.Version != "1" || report.PipVersion != pip || report.Environment.ImplementationName != "cpython" || report.Environment.ImplementationVersion != python || report.Environment.PythonFullVersion != python || report.Environment.PlatformMachine != "x86_64" || report.Environment.SysPlatform != "linux" || report.Install == nil || len(report.Install) > maxPyPIReportEntries {
		return pipReport{}, errors.New("upgrade delta runtime or bounds are invalid")
	}
	return report, nil
}

func sameUpgradeCandidate(left, right Candidate) bool {
	return left.source == right.source && left.project == right.project && left.version == right.version && left.filename == right.filename && left.url == right.url && left.sha256 == right.sha256 && left.requiresPython == right.requiresPython && left.primary == right.primary && slices.Equal(left.dependencies, right.dependencies) && slices.Equal(left.requirements, right.requirements) && slices.Equal(left.requestedExtras, right.requestedExtras)
}
