package pypi

import (
	"bytes"
	"errors"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// UVNativeProjectLock is an unapproved native uv.lock, distinct from the
// existing complete pylock export. Neither a declared edge nor this type
// authenticates an archive or permits installation/build/execution.
type UVNativeProjectLock struct {
	body, manifest []byte
	python         string
	packages       []UVLockedPackage
	sources        map[string]uvNativeArchive
}

type uvNativeArchive struct {
	URL      string `toml:"url"`
	Hash     string `toml:"hash"`
	Size     int64  `toml:"size"`
	Uploaded string `toml:"upload-time"`
}

type uvNativePackage struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Source  struct {
		Registry *string `toml:"registry"`
		Virtual  *string `toml:"virtual"`
	} `toml:"source"`
	Dependencies []struct {
		Name string `toml:"name"`
	} `toml:"dependencies"`
	Wheels   []uvNativeArchive `toml:"wheels"`
	Sdist    *uvNativeArchive  `toml:"sdist"`
	Metadata *struct {
		RequiresDist []struct {
			Name      string `toml:"name"`
			Specifier string `toml:"specifier"`
		} `toml:"requires-dist"`
	} `toml:"metadata"`
}

type uvNativeProjectLockWire struct {
	Version  int               `toml:"version"`
	Revision int               `toml:"revision"`
	Python   string            `toml:"requires-python"`
	Packages []uvNativePackage `toml:"package"`
}

func ParseUVNativeProjectLock(input UVProjectInput, body []byte) (UVNativeProjectLock, error) {
	if !input.Valid() || !validUVProjectText(body, false) {
		return UVNativeProjectLock{}, errors.New("native UV project lock input is unavailable")
	}
	var wire uvNativeProjectLockWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return UVNativeProjectLock{}, err
	}
	var manifest uvProjectInputWire
	if toml.Unmarshal(input.body, &manifest) != nil || manifest.Project == nil || wire.Version != 1 || wire.Revision != 5 || !requiresPythonMetadataMatches(manifest.Project.Python, wire.Python) || len(wire.Packages) == 0 || len(wire.Packages) > PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts()+1 {
		return UVNativeProjectLock{}, errors.New("native UV project lock schema, runtime or package count is unsupported")
	}
	byName := map[string]uvNativePackage{}
	var root uvNativePackage
	sources := map[string]uvNativeArchive{}
	validationPackages := make([]map[string]any, 0, len(wire.Packages)-1)
	for _, item := range wire.Packages {
		name, err := NormalizeProjectName(item.Name)
		version, versionErr := NormalizeVersion(item.Version)
		if err != nil || name != item.Name || byName[name].Name != "" || versionErr != nil || version != item.Version || !IsFinalVersion(version) {
			return UVNativeProjectLock{}, errors.New("native UV project package identity is invalid or repeated")
		}
		byName[name] = item
		if name == input.project {
			if item.Version != input.version || item.Source.Virtual == nil || *item.Source.Virtual != "." || item.Source.Registry != nil || len(item.Wheels) != 0 || item.Sdist != nil {
				return UVNativeProjectLock{}, errors.New("native UV virtual root differs from frozen project")
			}
			root = item
			continue
		}
		if item.Source.Registry == nil || *item.Source.Registry != "https://pypi.org/simple" || item.Source.Virtual != nil || item.Metadata != nil || len(item.Wheels) == 0 || len(item.Wheels) > maxUVWheelAlternatives {
			return UVNativeProjectLock{}, errors.New("native UV dependency source or metadata is unsupported")
		}
		wheels := make([]map[string]any, 0, len(item.Wheels))
		for _, wheel := range item.Wheels {
			if !validUVNativeArchiveMetadata(wheel) {
				return UVNativeProjectLock{}, errors.New("native UV wheel metadata is unsupported")
			}
			wheels = append(wheels, map[string]any{"url": wheel.URL, "size": wheel.Size, "hashes": map[string]string{"sha256": strings.TrimPrefix(wheel.Hash, "sha256:")}})
		}
		if item.Sdist != nil {
			if err := validateUVNativeSdist(*item.Sdist, item.Name, item.Version); err != nil {
				return UVNativeProjectLock{}, err
			}
			sources[item.Name] = *item.Sdist
		}
		validationPackages = append(validationPackages, map[string]any{"name": item.Name, "version": item.Version, "wheels": wheels})
	}
	if root.Name == "" || validateUVNativeRoot(input, root, byName) != nil || validateUVNativeClosure(root.Name, byName) != nil {
		return UVNativeProjectLock{}, errors.New("native UV project root, constraints or complete closure differs from manifest")
	}
	// This is an internal validation view, never claimed to be native export
	// evidence. Reuse all existing wheel source/hash/size/identity budgets.
	view, err := toml.Marshal(map[string]any{"lock-version": "1.0", "created-by": "uv", "requires-python": wire.Python, "packages": validationPackages})
	if err != nil {
		return UVNativeProjectLock{}, err
	}
	parsed, err := ParseUVProjectLock(view, runtimeidentity.PythonVersion)
	if err != nil {
		return UVNativeProjectLock{}, err
	}
	return UVNativeProjectLock{body: append([]byte(nil), body...), manifest: input.Body(), python: wire.Python, packages: parsed.Packages(), sources: sources}, nil
}

func validUVNativeArchiveMetadata(archive uvNativeArchive) bool {
	if !strings.HasPrefix(archive.Hash, "sha256:") || !sha256HexPattern.MatchString(strings.TrimPrefix(archive.Hash, "sha256:")) || archive.Size <= 0 || archive.Size > PublicPyPIProfile().ResourcePolicy().MaxArtifactCompressed() {
		return false
	}
	if archive.Uploaded != "" {
		if _, err := time.Parse(time.RFC3339Nano, archive.Uploaded); err != nil {
			return false
		}
	}
	return true
}

func validateUVNativeSdist(archive uvNativeArchive, name, version string) error {
	location, err := url.Parse(archive.URL)
	if err != nil || !validUVNativeArchiveMetadata(archive) {
		return errors.New("native UV inert source alternative is invalid")
	}
	filename := path.Base(location.Path)
	project, selected, err := parseSdistFilename(filename)
	if err != nil || project != name || selected != version || validateDistributionURLForSource(archive.URL, filename, PublicPyPIProfile(), false) != nil {
		return errors.New("native UV inert source alternative identity differs")
	}
	return nil
}

func validateUVNativeRoot(input UVProjectInput, root uvNativePackage, packages map[string]uvNativePackage) error {
	if len(root.Dependencies) != len(input.requirements) {
		return errors.New("native UV root dependency count differs")
	}
	metadata := map[string]string{}
	if root.Metadata != nil {
		for _, item := range root.Metadata.RequiresDist {
			requirement, err := parseUVProjectRequirement(item.Name + item.Specifier)
			if err != nil || requirement.Project() != item.Name || metadata[item.Name] != "" {
				return errors.New("native UV root requirement metadata is invalid")
			}
			metadata[item.Name] = uvProjectRequirementKey(requirement)
		}
	}
	if len(metadata) != len(input.requirements) {
		return errors.New("native UV root requirement metadata is incomplete")
	}
	edges := map[string]bool{}
	for _, edge := range root.Dependencies {
		edges[edge.Name] = true
	}
	for _, requirement := range input.requirements {
		if !edges[requirement.Project()] || metadata[requirement.Project()] != uvProjectRequirementKey(requirement) || !requirement.AcceptsVersion(packages[requirement.Project()].Version) {
			return errors.New("native UV selected root version or constraint differs")
		}
	}
	return nil
}

func uvProjectRequirementKey(requirement UpgradeRequirement) string {
	parts := strings.Split(strings.TrimPrefix(requirement.Requirement(), requirement.Project()), ",")
	for i := range parts {
		parts[i] = strings.ReplaceAll(parts[i], " ", "")
	}
	sort.Strings(parts)
	return requirement.Project() + strings.Join(parts, ",")
}

func validateUVNativeClosure(root string, packages map[string]uvNativePackage) error {
	for name, item := range packages {
		if len(item.Dependencies) > len(packages) {
			return errors.New("native UV dependency edges exceed bounds")
		}
		seen := map[string]bool{}
		for _, edge := range item.Dependencies {
			if edge.Name == name || edge.Name == root || packages[edge.Name].Name == "" || seen[edge.Name] {
				return errors.New("native UV dependency edge is absent, invalid or repeated")
			}
			seen[edge.Name] = true
		}
	}
	visited := map[string]bool{}
	queue := []string{root}
	for len(queue) != 0 {
		name := queue[0]
		queue = queue[1:]
		if visited[name] {
			continue
		}
		visited[name] = true
		for _, edge := range packages[name].Dependencies {
			queue = append(queue, edge.Name)
		}
	}
	if len(visited) != len(packages) {
		return errors.New("native UV lock contains disconnected packages")
	}
	return nil
}

func (l UVNativeProjectLock) Control() (domain.ProjectControlFile, error) {
	if len(l.body) == 0 {
		return domain.ProjectControlFile{}, errors.New("native UV project lock is unavailable")
	}
	return domain.NewProjectControlFile(UVNativeLockControl, l.body, true)
}

// ValidateTargetExport only links two unapproved selections. Actual installed
// wheel choice, independent source/archives and current Policy remain required.
func (l UVNativeProjectLock) ValidateTargetExport(export UVProjectLock) error {
	if len(l.body) == 0 || len(export.body) == 0 || len(l.packages) != len(export.packages) || export.python != runtimeidentity.PythonVersion {
		return errors.New("native UV lock and complete target export differ")
	}
	byName := map[string]UVLockedPackage{}
	for _, item := range l.packages {
		byName[item.Name] = item
	}
	for _, item := range export.packages {
		native, ok := byName[item.Name]
		if !ok || native.Version != item.Version {
			return errors.New("UV export package version differs from native lock")
		}
		wheels := map[UVLockedWheel]bool{}
		for _, wheel := range native.Wheels {
			wheels[wheel] = true
		}
		for _, wheel := range item.Wheels {
			if !wheels[wheel] {
				return errors.New("UV export wheel differs from native lock")
			}
		}
	}
	return nil
}
