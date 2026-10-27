package pypi

import (
	"archive/zip"
	"bytes"
	"errors"
	"sort"
	"strings"
)

// PipxRuntimePlan is a static plan for the selected original ZIP namespace.
// It is neither a wheel installation plan nor external completion evidence.
type PipxRuntimePlan struct {
	zip           PipxRuntimeZIP
	distributions []PipxRuntimeDistribution
	units         []PlannedObservationUnit
}

type PipxRuntimeDistribution struct{ Project, Version string }

func (p PipxRuntimePlan) Valid() bool {
	return p.zip.Valid() && len(p.distributions) > 0 && len(p.units) > 0
}
func (p PipxRuntimePlan) ZIP() PipxRuntimeZIP { return p.zip }
func (p PipxRuntimePlan) Units() []PlannedObservationUnit {
	return append([]PlannedObservationUnit(nil), p.units...)
}
func (p PipxRuntimePlan) Distributions() []PipxRuntimeDistribution {
	return append([]PipxRuntimeDistribution(nil), p.distributions...)
}

// ObservationPlan rechecks original embedded metadata and RECORD membership.
// The fixed loader module is a required runtime import, not a promise that a
// post-install console command works. No member is extracted or executed.
func (z PipxRuntimeZIP) ObservationPlan() (PipxRuntimePlan, error) {
	if !z.Valid() {
		return PipxRuntimePlan{}, errors.New("pipx runtime plan requires selected ZIP bytes")
	}
	distributions, units, err := pipxRuntimeZIPPlan(z.body)
	if err != nil {
		return PipxRuntimePlan{}, err
	}
	return PipxRuntimePlan{zip: z, distributions: distributions, units: units}, nil
}

func pipxRuntimeZIPPlan(body []byte) ([]PipxRuntimeDistribution, []PlannedObservationUnit, error) {
	facts, err := pipxRuntimeZIPFiles(body)
	if err != nil {
		return nil, nil, err
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, nil, err
	}
	entries := make(map[string]*zip.File, len(archive.File))
	for _, entry := range archive.File {
		entries[entry.Name] = entry
	}
	limits := PublicPyPIProfile().ResourcePolicy().WheelLimits()
	var distributions []PipxRuntimeDistribution
	owners := map[string]string{}
	for _, fact := range facts {
		parts := strings.Split(fact.Path, "/")
		if len(parts) != 2 || parts[1] != "METADATA" || !strings.HasSuffix(parts[0], ".dist-info") {
			continue
		}
		metadata, err := readZipEntry(entries[fact.Path], limits.MaxMetadata)
		if err != nil {
			return nil, nil, err
		}
		headers, err := parseRFC822Metadata(metadata, limits.MaxMetadata)
		if err != nil {
			return nil, nil, err
		}
		rawName, nameOK := headers.first("name")
		rawVersion, versionOK := headers.first("version")
		project, nameErr := NormalizeProjectName(rawName)
		version, versionErr := NormalizeVersion(rawVersion)
		if !nameOK || !versionOK || nameErr != nil || versionErr != nil || !pipxSupplyProject(project) || version != rawVersion || owners[project] != "" || parts[0] != wheelDistInfo(project, version) {
			return nil, nil, errors.New("pipx runtime distribution identity is invalid")
		}
		owners[project] = parts[0]
		distributions = append(distributions, PipxRuntimeDistribution{project, version})
	}
	if owners["pipx"] == "" || len(distributions) > 7 {
		return nil, nil, errors.New("pipx runtime distribution set is invalid")
	}
	sort.Slice(distributions, func(i, j int) bool { return distributions[i].Project < distributions[j].Project })
	assigned := map[string]bool{}
	candidates := map[string]ObservationCoverage{}
	for _, distribution := range distributions {
		distInfo := owners[distribution.Project]
		dataDir := strings.TrimSuffix(distInfo, ".dist-info") + ".data"
		var regular []string
		original := map[string]PipxRuntimeZIPFile{}
		for _, fact := range facts {
			head := strings.SplitN(fact.Path, "/", 2)[0]
			if head != distribution.Project && head != distInfo && head != dataDir {
				continue
			}
			if assigned[fact.Path] {
				return nil, nil, errors.New("pipx runtime member has multiple owners")
			}
			assigned[fact.Path] = true
			regular = append(regular, fact.Path)
			original[fact.Path] = fact
		}
		imports, err := pipxRuntimeDistributionImports(distribution, distInfo, dataDir, regular, original, entries, limits)
		if err != nil {
			return nil, nil, err
		}
		for _, unit := range imports {
			previous, exists := candidates[unit.Candidate]
			if !exists || previous != RequiredObservation {
				candidates[unit.Candidate] = unit.Coverage
			}
		}
	}
	if len(assigned) != len(facts) {
		return nil, nil, errors.New("pipx runtime ZIP contains unowned members")
	}
	names := make([]string, 0, len(candidates))
	for name := range candidates {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 || len(names) > PublicPyPIProfile().ResourcePolicy().MaxObservationImportsPerArtifact() {
		return nil, nil, errors.New("pipx runtime import plan exceeds bound")
	}
	units := make([]PlannedObservationUnit, 0, len(names))
	for _, name := range names {
		units = append(units, PlannedObservationUnit{Kind: DirectImportUnit, Candidate: name, Coverage: candidates[name]})
	}
	return distributions, units, nil
}

func pipxRuntimeDistributionImports(distribution PipxRuntimeDistribution, distInfo, dataDir string, regular []string, original map[string]PipxRuntimeZIPFile, entries map[string]*zip.File, limits WheelLimits) ([]PlannedObservationUnit, error) {
	metadata, err := readZipEntry(entries[distInfo+"/METADATA"], limits.MaxMetadata)
	if err != nil {
		return nil, err
	}
	headers, err := parseRFC822Metadata(metadata, limits.MaxMetadata)
	if err != nil {
		return nil, err
	}
	wheel, err := readZipEntry(entries[distInfo+"/WHEEL"], limits.MaxMetadata)
	if err != nil {
		return nil, err
	}
	wheelHeaders, err := parseRFC822Metadata(wheel, limits.MaxMetadata)
	if err != nil {
		return nil, err
	}
	wheelVersion, _ := wheelHeaders.first("wheel-version")
	pure, _ := wheelHeaders.first("root-is-purelib")
	tags := wheelHeaders.all("tag")
	if !supportedWheelVersion(wheelVersion) || pure != "true" || len(tags) != 1 || tags[0] != "py3-none-any" {
		return nil, errors.New("pipx runtime embedded wheel is not pure Python")
	}
	record, err := readZipEntry(entries[distInfo+"/RECORD"], limits.MaxMetadata)
	if err != nil {
		return nil, err
	}
	files, err := validateRecord(record, regular, entries, distInfo)
	if err != nil {
		return nil, err
	}
	for index, file := range files {
		fact, ok := original[file.Path]
		if !ok {
			return nil, errors.New("pipx runtime RECORD contains foreign member")
		}
		files[index].Size = fact.Size
		files[index].SHA256 = fact.SHA256
	}
	declared, err := declaredWheelImports(headers.all("import-name"), headers.all("import-namespace"))
	if err != nil {
		return nil, err
	}
	epList := headers.all("entry-points")
	var details []EntryPoint
	if entry := entries[distInfo+"/entry_points.txt"]; entry != nil {
		body, err := readZipEntry(entry, limits.MaxMetadata)
		if err != nil {
			return nil, err
		}
		parsed, epDetails, err := parseEntryPointsTxt(body)
		if err != nil {
			return nil, err
		}
		epList = append(epList, parsed...)
		details = epDetails
	}
	surface, imports, empty, err := classifyWheelSurface(files, distInfo, dataDir, declared, epList, details, entries, limits)
	if err != nil {
		return nil, err
	}
	if empty || len(surface.SiteStartupHooks)+len(surface.Scripts)+len(surface.NativeLibraries)+len(surface.PythonExtensions)+len(surface.UnresolvedFiles)+len(surface.NamespaceContainers) != 0 {
		return nil, errors.New("pipx runtime executable surface is unsupported")
	}
	for _, file := range surface.InstalledFiles {
		if file.Scheme == SchemeSite && (file.ArchivePath != file.Destination || !strings.HasPrefix(file.Destination, distribution.Project+"/")) {
			return nil, errors.New("pipx runtime module namespace requires relocation")
		}
	}
	for _, name := range imports {
		if name != distribution.Project && !strings.HasPrefix(name, distribution.Project+".") {
			return nil, errors.New("pipx runtime import escapes original owner")
		}
	}
	if distribution.Project == "pipx" {
		if len(details) != 1 || details[0].Group != "console_scripts" || details[0].Name != "pipx" || details[0].Module != "pipx.__main__" || details[0].Attr != "cli" || len(details[0].Extras) != 0 || entries["pipx/__main__.py"] == nil {
			return nil, errors.New("pipx runtime loader identity is invalid")
		}
		imports = append(imports, "pipx.__main__")
	}
	required := deduplicateSorted(imports)
	candidates := append([]string(nil), required...)
	for _, entry := range details {
		if !installedModuleExists(surface.InstalledFiles, entry.Module) || (entry.Module != distribution.Project && !strings.HasPrefix(entry.Module, distribution.Project+".")) {
			return nil, errors.New("pipx runtime entrypoint has no owned module")
		}
		candidates = append(candidates, entry.Module)
		if entry.Group != "console_scripts" && entry.Group != "gui_scripts" {
			required = append(required, entry.Module)
		}
	}
	classification := ObservationPlan{RequiredImportCandidates: deduplicateSorted(required), EntryPointDetails: details}
	units := make([]PlannedObservationUnit, 0, len(candidates))
	for _, name := range deduplicateSorted(candidates) {
		units = append(units, PlannedObservationUnit{Kind: DirectImportUnit, Candidate: name, Coverage: classification.importCoverage(name)})
	}
	if len(epList) != len(details) {
		return nil, errors.New("pipx runtime entrypoint metadata is unresolved")
	}
	return units, nil
}
