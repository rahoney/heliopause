package pypi

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// PipxSupplyClosure records static, independently metadata-bound wheel facts.
// Publisher signatures, runtime admission and installation authority are separate.
type PipxSupplyClosure struct {
	wheels []PipxSupplyWheel
	files  []PipxSupplyFile
}

type PipxSupplyWheel struct {
	Project  string `json:"project"`
	Version  string `json:"version"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

type PipxSupplyFile struct {
	Project     string             `json:"project"`
	Path        string             `json:"path"`
	Scheme      InstallationScheme `json:"scheme"`
	Destination string             `json:"destination"`
	Role        RuntimeRole        `json:"role"`
	Size        int64              `json:"size"`
	SHA256      string             `json:"sha256"`
}

func (c PipxSupplyClosure) Valid() bool { return len(c.wheels) > 0 && len(c.files) > 0 }
func (c PipxSupplyClosure) Python() string {
	if !c.Valid() {
		return ""
	}
	return runtimeidentity.PythonVersion
}
func (c PipxSupplyClosure) Wheels() []PipxSupplyWheel {
	return append([]PipxSupplyWheel(nil), c.wheels...)
}
func (c PipxSupplyClosure) Files() []PipxSupplyFile { return append([]PipxSupplyFile(nil), c.files...) }

// BindPipxSupplyClosure consumes explicit wheel selections, not a fabricated
// native uv lock or installed receipt. It reuses the existing complete archive,
// metadata, marker and constraint checks for the current pinned Python tuple.
func BindPipxSupplyClosure(packages []UVLockedPackage, inputs []UVWheelInput, pages []SimpleProject) (PipxSupplyClosure, error) {
	return bindPipxSupplyClosure(packages, inputs, pages, []string{"pipx"})
}

// BindPipxRuntimeSupply validates the explicitly selected pipx and pytest runtime
// roots. It does not invent an upstream pipx dependency or augment a user graph.
func BindPipxRuntimeSupply(packages []UVLockedPackage, inputs []UVWheelInput, pages []SimpleProject) (PipxSupplyClosure, error) {
	return bindPipxSupplyClosure(packages, inputs, pages, []string{"pipx", "pytest"})
}

func bindPipxSupplyClosure(packages []UVLockedPackage, inputs []UVWheelInput, pages []SimpleProject, roots []string) (PipxSupplyClosure, error) {
	policy := PublicPyPIProfile().ResourcePolicy()
	if len(packages) == 0 || len(packages) > policy.MaxGraphArtifacts() {
		return PipxSupplyClosure{}, errors.New("pipx supply closure count is invalid")
	}
	seen := map[string]bool{}
	rootVersion := ""
	for _, p := range packages {
		name, err := NormalizeProjectName(p.Name)
		version, verErr := NormalizeVersion(p.Version)
		if err != nil || verErr != nil || name != p.Name || version != p.Version || seen[name] || !pipxSupplyProject(name) || len(p.Wheels) != 1 {
			return PipxSupplyClosure{}, errors.New("pipx supply selection is unsupported or ambiguous")
		}
		seen[name] = true
		w := p.Wheels[0]
		wheelName, wheelVersion, python, abi, platform, err := ParseWheelFilename(w.Filename)
		if err != nil || wheelName != name || wheelVersion != version || len(python) != 1 || python[0] != "py3" || len(abi) != 1 || abi[0] != "none" || len(platform) != 1 || platform[0] != "any" {
			return PipxSupplyClosure{}, errors.New("pipx supply requires exact pure-Python wheel selections")
		}
		if name == "pipx" {
			rootVersion = version
		}
	}
	if rootVersion == "" {
		return PipxSupplyClosure{}, errors.New("pipx supply main wheel is missing")
	}
	var files []PipxSupplyFile
	paths := map[string]bool{}
	byURL := map[string]UVWheelInput{}
	for _, input := range inputs {
		byURL[input.URL] = input
	}
	var expanded int64
	target := WheelTarget{runtimeidentity.PythonInterpreterTag, runtimeidentity.PythonABITag, runtimeidentity.PythonPlatformTag}
	candidates, err := inspectUVArchiveCandidates(packages, runtimeidentity.PythonVersion, "pipx", inputs, target, policy.WheelLimits(), func(w WheelInspection) error {
		if err := validatePipxMetadataSurface(w); err != nil {
			return err
		}
		if len(w.NativeExtensions)+len(w.Surface.PythonExtensions)+len(w.Surface.NativeLibraries)+len(w.Surface.Scripts)+len(w.Surface.UnresolvedFiles) != 0 {
			return errors.New("pipx supply contains an unsupported executable surface")
		}
		if w.Project == "pipx" {
			entries := w.Surface.EntryPointDetails
			if len(entries) != 1 || entries[0].Group != "console_scripts" || entries[0].Name != "pipx" || entries[0].Module != "pipx.__main__" || entries[0].Attr != "cli" || len(entries[0].Extras) != 0 {
				return errors.New("pipx supply entrypoint differs from the reviewed loader")
			}
		}
		installed := map[string]InstalledFile{}
		for _, f := range w.Surface.InstalledFiles {
			installed[f.ArchivePath] = f
		}
		for _, f := range w.Files {
			mapped, ok := installed[f.Path]
			head := strings.SplitN(mapped.Destination, "/", 2)[0]
			allowed := mapped.Scheme == SchemeSite && pipxSupplyOwnsSite(w.Project, mapped.Destination) || mapped.Scheme == SchemeDistInfo && head == w.Project+"-"+w.Version+".dist-info" || mapped.Scheme == SchemeData && mapped.Role == RuntimeRoleResource
			key := "site/" + mapped.Destination
			if mapped.Scheme == SchemeData {
				key = "data/" + mapped.Destination
			}
			if !ok || !allowed || paths[key] || strings.HasSuffix(f.Path, ".pyc") || strings.HasSuffix(f.Path, ".pyo") || strings.Contains("/"+f.Path+"/", "/__pycache__/") {
				return errors.New("pipx supply file is outside its isolated module/metadata root or collides")
			}
			if f.Path == w.Project+"-"+w.Version+".dist-info/RECORD" {
				var input UVWheelInput
				for _, p := range packages {
					if p.Name == w.Project {
						input = byURL[p.Wheels[0].URL]
					}
				}
				var err error
				f.Size, f.SHA256, err = pipxSupplyRecord(input, f.Path, policy.WheelLimits().MaxMetadata)
				if err != nil {
					return err
				}
			}
			if !sha256HexPattern.MatchString(f.SHA256) || f.Size < 0 || expanded > policy.MaxGraphUncompressed()-f.Size {
				return errors.New("pipx supply file evidence or expansion bound is invalid")
			}
			expanded += f.Size
			paths[key] = true
			files = append(files, PipxSupplyFile{w.Project, f.Path, mapped.Scheme, mapped.Destination, mapped.Role, f.Size, f.SHA256})
		}
		return nil
	})
	if err != nil {
		return PipxSupplyClosure{}, err
	}
	checked, err := CrossCheckReport(InstallationReport{candidates: candidates}, pages)
	if err != nil {
		return PipxSupplyClosure{}, err
	}
	if err := validatePipxSupplyConstraints(checked, runtimeidentity.PythonVersion); err != nil {
		return PipxSupplyClosure{}, err
	}
	if err := validatePipxSupplyGraphs(checked, roots); err != nil {
		return PipxSupplyClosure{}, err
	}
	// A regular file must never also be a parent directory in the merged tree.
	for filename := range paths {
		parts := strings.Split(filename, "/")
		for i := 1; i < len(parts); i++ {
			if paths[strings.Join(parts[:i], "/")] {
				return PipxSupplyClosure{}, errors.New("pipx supply file/directory collision")
			}
		}
	}
	var wheels []PipxSupplyWheel
	for _, p := range packages {
		w := p.Wheels[0]
		wheels = append(wheels, PipxSupplyWheel{p.Name, p.Version, w.Filename, w.URL, w.SHA256, w.Size})
	}
	sort.Slice(wheels, func(i, j int) bool { return wheels[i].Project < wheels[j].Project })
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return PipxSupplyClosure{wheels, files}, nil
}

// pipx's real metadata includes userpath!=1.9,>=1.9.2. Keep the qualified
// general/UV requirement language unchanged: this supply-only consumer handles
// final numeric exclusions with the existing equality semantics and delegates
// all remaining Python/dependency constraints to the existing validator.
func validatePipxSupplyConstraints(candidates []Candidate, python string) error {
	versions := map[string]string{}
	for _, candidate := range candidates {
		versions[candidate.project] = candidate.version
	}
	positive := append([]Candidate(nil), candidates...)
	for i, candidate := range candidates {
		positive[i].requirements = nil
		for _, requirement := range candidate.requirements {
			if !strings.Contains(requirement, "!=") {
				positive[i].requirements = append(positive[i].requirements, requirement)
				continue
			}
			head := boundedRequirementHead.FindStringSubmatch(requirement)
			if head == nil || head[2] != "" {
				return errors.New("pipx supply exclusion requirement is unsupported")
			}
			project, err := NormalizeProjectName(head[1])
			if err != nil {
				return err
			}
			var parts []string
			for _, part := range strings.Split(head[3], ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					return errors.New("pipx supply exclusion constraint is empty")
				}
				if !strings.HasPrefix(part, "!=") {
					parts = append(parts, part)
					continue
				}
				version := strings.TrimSpace(strings.TrimPrefix(part, "!="))
				if _, ok := parseNumericRelease(version); !ok {
					return errors.New("pipx supply supports only final numeric exclusions")
				}
				excluded, err := ParseBoundedRequirement(project + "==" + version)
				if err != nil || excluded.Satisfies(versions[project]) {
					return errors.New("pipx supply selected an excluded dependency version")
				}
			}
			positive[i].requirements = append(positive[i].requirements, project+strings.Join(parts, ","))
		}
	}
	return validateUVCandidateConstraints(positive, python)
}

// InspectWheel deliberately preserves RECORD's self-hash exemption. A supply
// materialization manifest additionally needs its actual immutable bytes; read
// only that bounded member of the already validated archive, without rewriting it.
func pipxSupplyRecord(input UVWheelInput, name string, limit int64) (int64, string, error) {
	if input.Reader == nil || limit <= 0 || limit > defaultWheelMaxMetadata {
		return 0, "", errors.New("pipx supply RECORD input or metadata bound is invalid")
	}
	archive, err := zip.NewReader(input.Reader, input.Size)
	if err != nil {
		return 0, "", err
	}
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		if file.UncompressedSize64 > uint64(limit) {
			return 0, "", errors.New("pipx supply RECORD exceeds metadata bound")
		}
		reader, err := file.Open()
		if err != nil {
			return 0, "", err
		}
		body, readErr := io.ReadAll(io.LimitReader(reader, limit+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || int64(len(body)) > limit || uint64(len(body)) != file.UncompressedSize64 {
			return 0, "", errors.New("pipx supply RECORD read failed or changed")
		}
		sum := sha256.Sum256(body)
		return int64(len(body)), hex.EncodeToString(sum[:]), nil
	}
	return 0, "", errors.New("pipx supply RECORD is missing")
}

func pipxSupplyProject(name string) bool {
	switch name {
	case "pipx", "argcomplete", "click", "filelock", "packaging", "platformdirs", "userpath", "pytest", "iniconfig", "pluggy", "pygments":
		return true
	default:
		return false
	}
}

// Each explicitly requested runtime root retains the existing single-primary
// graph contract. Shared dependencies have identical original candidate bytes.
func validatePipxSupplyGraphs(candidates []Candidate, roots []string) error {
	byName := make(map[string]Candidate, len(candidates))
	for _, candidate := range candidates {
		byName[candidate.project] = candidate
	}
	complete := map[string]bool{}
	for _, root := range roots {
		main, ok := byName[root]
		if !ok {
			return errors.New("pipx runtime supply root is missing")
		}
		reached := map[string]bool{}
		queue := []string{root}
		var subset []Candidate
		for len(queue) > 0 {
			name := queue[0]
			queue = queue[1:]
			if reached[name] {
				continue
			}
			candidate, ok := byName[name]
			if !ok {
				return errors.New("pipx runtime dependency is missing")
			}
			reached[name] = true
			complete[name] = true
			candidate.primary = name == root
			subset = append(subset, candidate)
			queue = append(queue, candidate.dependencies...)
		}
		reference, err := ParseReference(root + "@" + main.version)
		if err != nil {
			return err
		}
		if _, err := BuildLockedGraph(reference, subset); err != nil {
			return err
		}
	}
	if len(complete) != len(candidates) {
		return errors.New("pipx supply includes unreachable wheels")
	}
	return nil
}

// These are reviewed original import namespaces, not trust or execution grants.
// The pytest wheel also owns _pytest and the single py.py compatibility module.
func pipxSupplyOwnsSite(project, destination string) bool {
	head := strings.SplitN(destination, "/", 2)[0]
	return head == project || project == "pytest" && (head == "_pytest" || destination == "py.py")
}

func pipxSupplyOwnsImport(project, module string) bool {
	root := strings.SplitN(module, ".", 2)[0]
	return root == project || project == "pytest" && (root == "_pytest" || module == "py")
}
