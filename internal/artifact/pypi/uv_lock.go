package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const maxUVWheelAlternatives = 64

// UVLock retains an unapproved, bounded native uv pylock selection. A package
// may have several wheel alternatives. None is implicitly chosen or executable.
type UVLock struct {
	reference domain.ArtifactReference
	python    string
	packages  []UVLockedPackage
}

// UVLockedPackage is a registry version selection, not a verified artifact.
type UVLockedPackage struct {
	Name, Version string
	Wheels        []UVLockedWheel
}

// UVLockedWheel is one lock-declared wheel candidate. Its bytes still require
// independent Simple metadata and complete archive/digest/RECORD validation.
type UVLockedWheel struct {
	URL, Filename, SHA256 string
	Size                  int64
}

func (l UVLock) Packages() []UVLockedPackage {
	return cloneUVLockedPackages(l.packages)
}

func cloneUVLockedPackages(packages []UVLockedPackage) []UVLockedPackage {
	result := append([]UVLockedPackage(nil), packages...)
	for i := range result {
		result[i].Wheels = append([]UVLockedWheel(nil), result[i].Wheels...)
	}
	return result
}

type uvLockWire struct {
	Version  string `toml:"lock-version"`
	Creator  string `toml:"created-by"`
	Python   string `toml:"requires-python"`
	Packages *[]struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
		Wheels  []struct {
			URL      string            `toml:"url"`
			Size     int64             `toml:"size"`
			Hashes   map[string]string `toml:"hashes"`
			Uploaded time.Time         `toml:"upload-time"`
		} `toml:"wheels"`
	} `toml:"packages"`
}

// ParseUVLock accepts the reviewed wheel-only native pylock 1.0 format. The
// trusted runtime selects uv's version separately; created-by is not provenance.
// Unknown fields (including local/VCS sources, sdists and universal markers)
// fail closed. This does not fabricate dependency edges from a flat lock.
func ParseUVLock(reference domain.ArtifactReference, body []byte, expectedPython string) (UVLock, error) {
	if reference.Source() != PublicPyPIProfile().Source() || len(body) == 0 || len(body) > maxReportBytes {
		return UVLock{}, errors.New("uv lock input is unsupported")
	}
	root, err := RequestedProject(reference)
	if err != nil {
		return UVLock{}, err
	}
	requested, exact, err := RequestedVersion(reference)
	if err != nil {
		return UVLock{}, err
	}
	packages, err := parseUVLockedPackages(body, expectedPython, root, requested, exact, false)
	if err != nil {
		return UVLock{}, err
	}
	return UVLock{reference: reference, python: expectedPython, packages: packages}, nil
}

// The complete-project caller alone accepts a present, explicit empty array.
// All source/header/identity/archive-alternative budgets remain common.
func parseUVLockedPackages(body []byte, expectedPython, root, requested string, exact, allowEmpty bool) ([]UVLockedPackage, error) {
	if len(body) == 0 || len(body) > maxReportBytes {
		return nil, errors.New("uv lock input is unsupported")
	}
	policy := PublicPyPIProfile().ResourcePolicy()
	var wire uvLockWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return nil, errors.New("uv lock format is invalid or unsupported")
	}
	requirement, err := ParseBoundedRequirement("python" + wire.Python)
	if err != nil || !requirement.Satisfies(expectedPython) || wire.Python == "" || wire.Version != "1.0" || wire.Creator != "uv" || wire.Packages == nil {
		return nil, errors.New("uv lock header or target is unsupported")
	}
	if (!allowEmpty && len(*wire.Packages) == 0) || len(*wire.Packages) > policy.MaxGraphArtifacts() {
		return nil, errors.New("uv lock header or target is unsupported")
	}
	result := make([]UVLockedPackage, 0, len(*wire.Packages))
	seen := make(map[string]bool)
	totalWheels := 0
	var declaredMaximum int64
	for _, item := range *wire.Packages {
		name, err := NormalizeProjectName(item.Name)
		if err != nil || name != item.Name || seen[name] {
			return nil, errors.New("uv lock package identity is ambiguous")
		}
		seen[name] = true
		version, err := NormalizeVersion(item.Version)
		if err != nil || version != item.Version || !IsFinalVersion(version) || name == root && exact && version != requested {
			return nil, errors.New("uv lock package version does not bind its request")
		}
		if len(item.Wheels) == 0 || len(item.Wheels) > maxUVWheelAlternatives {
			return nil, errors.New("uv lock wheel alternatives exceed supported bounds")
		}
		totalWheels += len(item.Wheels)
		if totalWheels > maxPyPIReportEntries*4 {
			return nil, errors.New("uv lock total wheel alternatives exceed bounds")
		}
		entry := UVLockedPackage{Name: name, Version: version}
		var packageMaximum int64
		urls := make(map[string]bool)
		for _, wheel := range item.Wheels {
			location, err := url.Parse(wheel.URL)
			if err != nil {
				return nil, errors.New("uv lock wheel URL is invalid")
			}
			filename := path.Base(location.Path)
			if validateDistributionURLForSource(wheel.URL, filename, PublicPyPIProfile(), false) != nil || urls[wheel.URL] || wheel.Size <= 0 || wheel.Size > policy.MaxArtifactCompressed() || len(wheel.Hashes) != 1 || !sha256HexPattern.MatchString(wheel.Hashes["sha256"]) {
				return nil, errors.New("uv lock wheel source, hash or size is invalid")
			}
			project, version, _, _, _, err := ParseWheelFilename(filename)
			if err != nil || project != name || version != entry.Version {
				return nil, errors.New("uv lock wheel filename identity mismatch")
			}
			packageMaximum = max(packageMaximum, wheel.Size)
			urls[wheel.URL] = true
			entry.Wheels = append(entry.Wheels, UVLockedWheel{URL: wheel.URL, Filename: filename, SHA256: wheel.Hashes["sha256"], Size: wheel.Size})
		}
		if declaredMaximum > policy.MaxGraphCompressed()-packageMaximum {
			return nil, errors.New("uv lock exceeds public graph download budget")
		}
		declaredMaximum += packageMaximum
		result = append(result, entry)
	}
	if root != "" && !seen[root] {
		return nil, errors.New("uv lock requested package is absent")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// UVWheelInput supplies the exact materialized wheel selected from the lock.
// The caller's selection and contents are untrusted until BindWheels validates
// them. The Reader must remain stable while it is read, as for InspectWheel.
type UVWheelInput struct {
	URL    string
	Reader io.ReaderAt
	Size   int64
}

// BindWheels validates one actual wheel per locked package against the lock,
// the complete archive and independent Simple metadata, then reconstructs the
// existing graph from authenticated Requires-Dist. It never returns Policy.
func (l UVLock) BindWheels(inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) (domain.LockedDependencyGraph, error) {
	if len(l.packages) == 0 {
		return domain.LockedDependencyGraph{}, errors.New("uv selected wheel set is incomplete")
	}
	root, _ := RequestedProject(l.reference)
	checked, err := bindUVSelectedWheels(l.packages, l.python, root, inputs, pages, target, limits)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	return BuildLockedGraph(l.reference, checked)
}

func bindUVSelectedWheels(packages []UVLockedPackage, python, root string, inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) ([]Candidate, error) {
	if len(pages) != len(packages) {
		return nil, errors.New("uv selected wheel set is incomplete")
	}
	candidates, err := uvArchiveCandidates(packages, python, root, inputs, target, limits)
	if err != nil {
		return nil, err
	}
	if len(packages) == 0 {
		return candidates, nil
	}
	checked, err := CrossCheckReport(InstallationReport{candidates: candidates}, pages)
	if err != nil {
		return nil, err
	}
	if err := validateUVCandidateConstraints(checked, python); err != nil {
		return nil, err
	}
	return checked, nil
}

func uvArchiveCandidates(packages []UVLockedPackage, python, root string, inputs []UVWheelInput, target WheelTarget, limits WheelLimits) ([]Candidate, error) {
	return inspectUVArchiveCandidates(packages, python, root, inputs, target, limits, nil)
}

func inspectUVArchiveCandidates(packages []UVLockedPackage, python, root string, inputs []UVWheelInput, target WheelTarget, limits WheelLimits, collect func(WheelInspection) error) ([]Candidate, error) {
	if len(inputs) != len(packages) {
		return nil, errors.New("uv selected wheel set is incomplete")
	}
	parts := strings.Split(python, ".")
	if len(parts) != 3 || target.Python != "cp"+parts[0]+parts[1] || target.ABI != target.Python {
		return nil, errors.New("uv wheel target differs from locked Python")
	}
	policy := PublicPyPIProfile().ResourcePolicy()
	allowed := policy.WheelLimits()
	if limits.MaxCompressed <= 0 || limits.MaxUncompressed <= 0 || limits.MaxFiles <= 0 || limits.MaxMetadata <= 0 || limits.MaxCompressed > allowed.MaxCompressed || limits.MaxUncompressed > allowed.MaxUncompressed || limits.MaxFiles > allowed.MaxFiles || limits.MaxMetadata > allowed.MaxMetadata {
		return nil, errors.New("uv wheel limits exceed public source policy")
	}
	if len(packages) == 0 {
		return []Candidate{}, nil
	}
	var compressed, expanded, files int64
	selected := make(map[string]UVWheelInput)
	for _, input := range inputs {
		if input.Reader == nil || input.Size <= 0 || compressed > policy.MaxGraphCompressed()-input.Size {
			return nil, errors.New("uv selected wheel input is invalid")
		}
		if _, exists := selected[input.URL]; exists {
			return nil, errors.New("uv selected wheel input is duplicated")
		}
		compressed += input.Size
		selected[input.URL] = input
	}
	candidates := make([]Candidate, 0, len(inputs))
	for _, item := range packages {
		var candidate Candidate
		matches := 0
		for _, wheel := range item.Wheels {
			input, ok := selected[wheel.URL]
			if !ok {
				continue
			}
			matches++
			if input.Size != wheel.Size {
				return nil, errors.New("uv selected wheel size mismatch")
			}
			inspection, err := InspectWheel(input.Reader, input.Size, wheel.Filename, wheel.SHA256, target, limits)
			if err != nil {
				return nil, errors.New("uv selected wheel validation failed")
			}
			for _, file := range inspection.Files {
				if files >= policy.MaxGraphFiles() || expanded > policy.MaxGraphUncompressed()-file.Size {
					return nil, errors.New("uv wheel graph exceeds public expansion budget")
				}
				files++
				expanded += file.Size
			}
			// Reuse metadata/marker normalization, not pip resolution or a fake pip
			// report. These fields come only from the digest-validated wheel above.
			candidate, err = candidateFromWheelMetadata(inspection, wheel.URL, wheel.SHA256, python, item.Name == root)
			if err != nil {
				return nil, err
			}
			if collect != nil {
				if err := collect(inspection); err != nil {
					return nil, err
				}
			}
		}
		if matches != 1 {
			return nil, errors.New("uv selected wheel is absent or ambiguous")
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func validateUVCandidateConstraints(candidates []Candidate, python string) error {
	versions := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		versions[candidate.project] = candidate.version
		if candidate.requiresPython != "" {
			requirement, err := ParseBoundedRequirement("python" + candidate.requiresPython)
			if err != nil || !requirement.Satisfies(python) {
				return errors.New("uv wheel Python requirement is unsupported or unsatisfied")
			}
		}
	}
	for _, candidate := range candidates {
		for _, constraint := range candidate.requirements {
			requirement, err := ParseBoundedRequirement(constraint)
			if err != nil || !requirement.Satisfies(versions[requirement.Project()]) {
				return errors.New("uv selected dependency constraint is unsupported or unsatisfied")
			}
		}
	}
	return nil
}

// SelectInstalledWheels binds a bounded data-only snapshot read by the trusted
// controller from uv's private no-build materialization. It identifies exactly
// one lock candidate per distribution using standard WHEEL tags. Its output
// remains unapproved: fetch the exact archives and use BindWheels before a graph
// can be used. Installed metadata and INSTALLER/exit/log output are not authority.
func (l UVLock) SelectInstalledWheels(body []byte, target WheelTarget) ([]UVLockedWheel, error) {
	return selectUVInstalledWheels(l.packages, l.python, body, target, false)
}

func selectUVInstalledWheels(packages []UVLockedPackage, python string, body []byte, target WheelTarget, allowEmpty bool) ([]UVLockedWheel, error) {
	if (!allowEmpty && len(packages) == 0) || len(body) == 0 || len(body) > maxReportBytes {
		return nil, errors.New("uv installed wheel snapshot exceeds bounds")
	}
	parts := strings.Split(python, ".")
	if len(parts) != 3 || target.Python != "cp"+parts[0]+parts[1] || target.ABI != target.Python {
		return nil, errors.New("uv installed wheel target mismatch")
	}
	var installed []struct {
		Name    string   `json:"name"`
		Version string   `json:"version"`
		Tags    []string `json:"wheel_tags"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&installed); err != nil || installed == nil || len(installed) != len(packages) {
		return nil, errors.New("uv installed wheel snapshot is invalid or incomplete")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("uv installed wheel snapshot has trailing input")
	}
	locked := make(map[string]UVLockedPackage, len(packages))
	for _, entry := range packages {
		locked[entry.Name] = entry
	}
	seen := make(map[string]bool, len(installed))
	var result []UVLockedWheel
	for _, item := range installed {
		name, err := NormalizeProjectName(item.Name)
		entry, ok := locked[name]
		if err != nil || !ok || seen[name] || item.Version != entry.Version || len(item.Tags) == 0 || len(item.Tags) > maxUVWheelAlternatives {
			return nil, errors.New("uv installed wheel identity is ambiguous")
		}
		seen[name] = true
		tags := make(map[string]bool, len(item.Tags))
		for _, tag := range item.Tags {
			if len(tag) > 160 || tags[tag] {
				return nil, errors.New("uv installed wheel tag is invalid")
			}
			tags[tag] = true
		}
		matches := 0
		for _, wheel := range entry.Wheels {
			_, _, python, abi, platform, err := ParseWheelFilename(wheel.Filename)
			if err == nil && wheelTagsCompatible(python, abi, platform, target) && wheelMetadataTagsMatch(item.Tags, python, abi, platform) {
				result = append(result, wheel)
				matches++
			}
		}
		if matches != 1 {
			return nil, errors.New("uv installed wheel does not select one compatible locked candidate")
		}
	}
	return result, nil
}

// Both native selection and complete-project static preparation normalize real wheel metadata here.
func candidateFromWheelMetadata(inspection WheelInspection, distributionURL, declaredSHA, python string, requested bool) (Candidate, error) {
	metadata := pipInstall{Requested: requested}
	metadata.DownloadInfo.URL = distributionURL
	metadata.DownloadInfo.ArchiveInfo.Hashes = map[string]string{"sha256": declaredSHA}
	metadata.Metadata.Name, metadata.Metadata.Version = inspection.Project, inspection.Version
	metadata.Metadata.RequiresPython, metadata.Metadata.RequiresDist = inspection.RequiresPython, inspection.RequiresDist
	return parseReportCandidate(metadata, PublicPyPIProfile(), python, nil)
}
