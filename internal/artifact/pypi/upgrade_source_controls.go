package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const SourcePinnedUpgradeSelectionControlName = "pypi-update-source-selection.json"
const SourcePinnedUpgradeReportControlName = "pypi-update-source-report.json"
const SourcePinnedUpgradeConstraintsControlName = "pypi-update-constraints.txt"
const sourcePinnedUpgradeReportsLimit = 64
const sourcePinnedUpgradeBytesLimit = 16 << 20

// SourcePinnedUpgradeRecord preserves the original bounded pip reports from one
// source-pinned call. SelectionReport is required only in the first traversal.
type SourcePinnedUpgradeRecord struct {
	SourceID        string   `json:"source_id"`
	Requirement     string   `json:"requirement"`
	RequestedExtras []string `json:"requested_extras"`
	SelectionReport []byte   `json:"selection_report,omitempty"`
	CandidateReport []byte   `json:"candidate_report"`
}

type sourcePinnedUpgradeReports struct {
	Version int                         `json:"version"`
	Reports []SourcePinnedUpgradeRecord `json:"reports"`
}

type SourcePinnedUpgrade struct {
	graph    domain.LockedDependencyGraph
	controls []domain.ProjectControlFile
	digest   domain.ContentDigest
}

func (s SourcePinnedUpgrade) Graph() domain.LockedDependencyGraph { return s.graph }
func (s SourcePinnedUpgrade) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}
func (s SourcePinnedUpgrade) Digest() domain.ContentDigest { return s.digest }

// ExactUpgradeRequirement preserves the already validated candidate's explicit
// extra context when the second traversal requests precisely the selected bytes.
func (c Candidate) ExactUpgradeRequirement() (string, error) {
	profile, ok := ProfileForSource(c.source)
	if !ok || !sourceOwnsProject(profile, c.source, c.project) || boundedExtraSet(c.requestedExtras) != nil || len(c.requestedExtras) > 0 && c.project != "cuda-toolkit" {
		return "", errors.New("source-bound candidate and extra context are required")
	}
	reference, err := ParseReferenceForSource(c.project+"@"+c.version, c.source)
	if err != nil {
		return "", err
	}
	project, _ := RequestedProject(reference)
	version, _, _ := RequestedVersion(reference)
	if project != c.project || version != c.version {
		return "", errors.New("candidate identity is not canonical")
	}
	name := project
	if len(c.requestedExtras) > 0 {
		extras := append([]string(nil), c.requestedExtras...)
		sort.Strings(extras)
		if !slices.Equal(extras, c.requestedExtras) {
			return "", errors.New("candidate extras are not canonical")
		}
		name += "[" + strings.Join(extras, ",") + "]"
	}
	request := name + "==" + version
	if len(c.requestedExtras) > 0 {
		request += `; platform_system == "Linux"`
	}
	return request, nil
}

func (u UpgradeInput) BuildSourcePinnedUpgrade(first, final []SourcePinnedUpgradeRecord, pip string) (SourcePinnedUpgrade, error) {
	if !IsPyTorchSource(u.requirement.source) {
		return SourcePinnedUpgrade{}, errors.New("named upgrade input is required")
	}
	firstControl, err := sourcePinnedUpgradeControl(SourcePinnedUpgradeSelectionControlName, first)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	finalControl, err := sourcePinnedUpgradeControl(SourcePinnedUpgradeReportControlName, final)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	selected, err := u.parseSourcePinnedFirst(first, pip)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	if err := validateSourcePinnedIncoming(selected); err != nil {
		return SourcePinnedUpgrade{}, err
	}
	if err := u.validateSourcePinnedRetention(selected); err != nil {
		return SourcePinnedUpgrade{}, err
	}
	proposal, err := u.ProposeSourcePinnedUpgrade(selected)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	candidates, err := u.parseSourcePinnedFinal(final, selected, pip)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	if err := proposal.ValidateSourcePinnedFinal(candidates); err != nil {
		return SourcePinnedUpgrade{}, err
	}
	if err := validateSourcePinnedIncoming(candidates); err != nil {
		return SourcePinnedUpgrade{}, err
	}
	graph, err := BuildLockedGraph(u.requirement.Reference(), candidates)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	input, err := u.Control()
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	constraints, err := domain.NewProjectControlFile(SourcePinnedUpgradeConstraintsControlName, proposal.Constraints(), true)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	sum := sha256.Sum256(finalControl.Body())
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	return SourcePinnedUpgrade{graph, []domain.ProjectControlFile{input, firstControl, finalControl, constraints}, digest}, nil
}

func (u UpgradeInput) ParseSourcePinnedUpgrade(controls []domain.ProjectControlFile, pip string) (SourcePinnedUpgrade, error) {
	if len(controls) != 4 {
		return SourcePinnedUpgrade{}, errors.New("source-pinned upgrade requires its four complete controls")
	}
	items := map[string]domain.ProjectControlFile{}
	for _, item := range controls {
		if !item.Present() || len(item.Body()) == 0 || len(item.Body()) > upgradeInputLimit {
			return SourcePinnedUpgrade{}, errors.New("source-pinned control is absent or exceeds bounds")
		}
		if _, exists := items[item.Name()]; exists {
			return SourcePinnedUpgrade{}, errors.New("source-pinned controls are ambiguous")
		}
		items[item.Name()] = item
	}
	input, err := u.Control()
	if err != nil || !bytes.Equal(input.Body(), items[UpgradeInputControlName].Body()) {
		return SourcePinnedUpgrade{}, errors.New("source-pinned original input changed")
	}
	first, err := parseSourcePinnedUpgradeControl(items[SourcePinnedUpgradeSelectionControlName])
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	final, err := parseSourcePinnedUpgradeControl(items[SourcePinnedUpgradeReportControlName])
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	selection, err := u.BuildSourcePinnedUpgrade(first, final, pip)
	if err != nil {
		return SourcePinnedUpgrade{}, err
	}
	for _, expected := range selection.controls {
		if !bytes.Equal(expected.Body(), items[expected.Name()].Body()) {
			return SourcePinnedUpgrade{}, errors.New("source-pinned selection or retained constraints changed")
		}
	}
	return selection, nil
}

func sourcePinnedUpgradeControl(name string, records []SourcePinnedUpgradeRecord) (domain.ProjectControlFile, error) {
	if len(records) == 0 || len(records) > sourcePinnedUpgradeReportsLimit {
		return domain.ProjectControlFile{}, errors.New("source-pinned report count exceeds bounds")
	}
	total := 0
	for _, record := range records {
		if record.SourceID == "" || len(record.SourceID) > 64 || boundedExtraSet(record.RequestedExtras) != nil {
			return domain.ProjectControlFile{}, errors.New("source-pinned report metadata exceeds bounds")
		}
		if len(record.Requirement) == 0 || len(record.Requirement) > 1024 || len(record.CandidateReport) == 0 || len(record.CandidateReport) > maxReportBytes || len(record.SelectionReport) > maxReportBytes {
			return domain.ProjectControlFile{}, errors.New("source-pinned report exceeds bounds")
		}
		total += len(record.CandidateReport) + len(record.SelectionReport)
		if total > sourcePinnedUpgradeBytesLimit {
			return domain.ProjectControlFile{}, errors.New("source-pinned cumulative reports exceed bounds")
		}
	}
	body, err := json.Marshal(sourcePinnedUpgradeReports{1, records})
	if err != nil || len(body) > upgradeInputLimit {
		return domain.ProjectControlFile{}, errors.New("encoded source-pinned reports exceed the control bound")
	}
	return domain.NewProjectControlFile(name, body, true)
}

func parseSourcePinnedUpgradeControl(control domain.ProjectControlFile) ([]SourcePinnedUpgradeRecord, error) {
	if !control.Present() || len(control.Body()) > upgradeInputLimit {
		return nil, errors.New("bounded source-pinned control is required")
	}
	var wire sourcePinnedUpgradeReports
	decoder := json.NewDecoder(bytes.NewReader(control.Body()))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&wire) != nil || ensureSingleJSONValue(decoder) != nil || wire.Version != 1 {
		return nil, errors.New("source-pinned report control is invalid")
	}
	canonical, err := sourcePinnedUpgradeControl(control.Name(), wire.Reports)
	if err != nil || !bytes.Equal(canonical.Body(), control.Body()) {
		return nil, errors.New("source-pinned report control is not canonical")
	}
	return wire.Reports, nil
}

func (u UpgradeInput) parseSourcePinnedFirst(records []SourcePinnedUpgradeRecord, pip string) ([]Candidate, error) {
	selected := map[string]Candidate{}
	for _, record := range records {
		source, err := domain.NewSourceID(record.SourceID)
		if err != nil {
			return nil, err
		}
		node, err := u.ParseSourcePinnedNodeSelection(record.Requirement, source, record.RequestedExtras, record.SelectionReport, pip)
		if err != nil {
			return nil, err
		}
		profile, _ := ProfileForSource(source)
		report, err := ParseInstallationReportForProfileWithExtras(node.Reference(), record.CandidateReport, pip, u.python, profile, record.RequestedExtras)
		if err != nil || len(report.Candidates()) != 1 {
			return nil, errors.New("fresh source-pinned node report is incomplete")
		}
		candidate := report.Candidates()[0]
		if err := node.ValidateFreshCandidate(candidate); err != nil {
			return nil, err
		}
		selected[candidate.project] = candidate.WithPrimary(candidate.project == u.requirement.project)
	}
	return sortedSourcePinnedCandidates(selected), nil
}

func (u UpgradeInput) parseSourcePinnedFinal(records []SourcePinnedUpgradeRecord, expected []Candidate, pip string) ([]Candidate, error) {
	wanted := map[string]Candidate{}
	for _, item := range expected {
		wanted[item.project] = item
	}
	selected := map[string]Candidate{}
	for _, record := range records {
		if len(record.SelectionReport) != 0 {
			return nil, errors.New("final source-pinned traversal cannot select new versions")
		}
		project, err := DependencyProject(record.Requirement)
		if err != nil {
			return nil, err
		}
		candidate, present := wanted[project]
		request, requestErr := candidate.ExactUpgradeRequirement()
		if !present || requestErr != nil || record.Requirement != request || record.SourceID != candidate.source.String() || !slices.Equal(record.RequestedExtras, candidate.requestedExtras) {
			return nil, errors.New("final source-pinned request differs from exact proposed intent")
		}
		reference, err := ParseReferenceForSource(project+"@"+candidate.version, candidate.source)
		if err != nil {
			return nil, err
		}
		profile, _ := ProfileForSource(candidate.source)
		report, err := ParseInstallationReportForProfileWithExtras(reference, record.CandidateReport, pip, u.python, profile, record.RequestedExtras)
		if err != nil || len(report.Candidates()) != 1 {
			return nil, errors.New("final source-pinned report is incomplete")
		}
		actual := report.Candidates()[0].WithPrimary(project == u.requirement.project)
		if !sameUpgradeCandidate(candidate, actual) {
			return nil, errors.New("source-pinned candidate changed before final inspection")
		}
		selected[project] = actual
	}
	return sortedSourcePinnedCandidates(selected), nil
}

func sortedSourcePinnedCandidates(items map[string]Candidate) []Candidate {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]Candidate, 0, len(names))
	for _, name := range names {
		result = append(result, items[name])
	}
	return result
}

func validateSourcePinnedIncoming(candidates []Candidate) error {
	requirements, err := sourcePinnedIncoming(candidates)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if !candidate.primary && !candidate.SatisfiesBoundedRequirements(requirements[candidate.project]) {
			return errors.New("source-pinned final candidate violates incoming requirements")
		}
	}
	return nil
}

func sourcePinnedIncoming(candidates []Candidate) (map[string][]BoundedRequirement, error) {
	requirements := map[string][]BoundedRequirement{}
	for _, candidate := range candidates {
		for _, value := range candidate.requirements {
			request, err := ParseBoundedRequirement(value)
			if err != nil {
				return nil, err
			}
			requirements[request.Project()] = append(requirements[request.Project()], request)
		}
	}
	return requirements, nil
}

func (u UpgradeInput) validateSourcePinnedRetention(candidates []Candidate) error {
	requirements, err := sourcePinnedIncoming(candidates)
	if err != nil {
		return err
	}
	installed := map[string]string{}
	for _, item := range u.installed {
		installed[item.project] = item.version
	}
	for _, candidate := range candidates {
		if candidate.primary {
			continue
		}
		version, present := installed[candidate.project]
		if !present {
			continue
		}
		original := Candidate{project: candidate.project, source: candidate.source, version: version}
		if original.SatisfiesBoundedRequirements(requirements[candidate.project]) && candidate.version != version {
			return errors.New("source-pinned selection changed a compatible installed dependency")
		}
	}
	return nil
}
