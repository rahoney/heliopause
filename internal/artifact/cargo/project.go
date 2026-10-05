package cargo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// ValidateProjectLock rejects unsupported source declarations before any SDK
// selection command. A valid lock declaration is not registry authentication.
func ValidateProjectLock(body []byte) error {
	_, err := parseLock(body)
	return err
}

// ValidateProjectManifest checks source and path declarations before Cargo
// reads a copied manifest. Cargo retains language/resolution semantics. This
// lexical check is additional to the runtime's anchored complete source copy.
func ValidateProjectManifest(body []byte, relativeManifest string) error {
	if len(body) == 0 || len(body) > MaxProjectControlBytes || !utf8.Valid(body) || relativeManifest == "" || relativeManifest != path.Clean(relativeManifest) || path.Base(relativeManifest) != "Cargo.toml" || validateManifestPath("Cargo.toml", relativeManifest) != nil {
		return errors.New("cargo project manifest is invalid or excessive")
	}
	var document map[string]any
	if toml.Unmarshal(body, &document) != nil || len(document) == 0 {
		return errors.New("cargo project manifest grammar is invalid")
	}
	for _, key := range []string{"patch", "replace"} {
		if _, exists := document[key]; exists {
			return errors.New("cargo project source replacement is unsupported")
		}
	}
	for _, key := range []string{"dependencies", "dev-dependencies", "build-dependencies"} {
		if err := validateManifestDependencies(document[key], relativeManifest); err != nil {
			return err
		}
	}
	if value, exists := document["target"]; exists {
		targets, ok := value.(map[string]any)
		if !ok || len(targets) > 1024 {
			return errors.New("cargo target declarations are invalid")
		}
		for _, value := range targets {
			target, ok := value.(map[string]any)
			if !ok {
				return errors.New("cargo target table is invalid")
			}
			for _, key := range []string{"dependencies", "dev-dependencies", "build-dependencies"} {
				if err := validateManifestDependencies(target[key], relativeManifest); err != nil {
					return err
				}
			}
		}
	}
	if value, exists := document["workspace"]; exists {
		workspace, ok := value.(map[string]any)
		if !ok {
			return errors.New("cargo workspace table is invalid")
		}
		if err := validateManifestDependencies(workspace["dependencies"], relativeManifest); err != nil {
			return err
		}
		for _, key := range []string{"members", "exclude", "default-members"} {
			if values, exists := workspace[key]; exists {
				members, ok := values.([]any)
				if !ok || len(members) > maxProjectPackages {
					return errors.New("cargo workspace member declarations are invalid")
				}
				for _, member := range members {
					value, ok := member.(string)
					if !ok || validateManifestPath(relativeManifest, value) != nil {
						return errors.New("cargo workspace member escapes project boundary")
					}
				}
			}
		}
	}
	if value, exists := document["package"]; exists {
		packageTable, ok := value.(map[string]any)
		if !ok {
			return errors.New("cargo package table is invalid")
		}
		for _, key := range []string{"workspace", "build"} {
			if value, exists := packageTable[key]; exists {
				if _, ok := value.(bool); ok && key == "build" {
					continue
				}
				value, ok := value.(string)
				if !ok || validateManifestPath(relativeManifest, value) != nil {
					return errors.New("cargo package path escapes project boundary")
				}
			}
		}
	}
	for _, key := range []string{"lib", "bin", "test", "example", "bench"} {
		value, exists := document[key]
		if !exists {
			continue
		}
		var targets []any
		if key == "lib" {
			targets = []any{value}
		} else {
			var ok bool
			targets, ok = value.([]any)
			if !ok || len(targets) > maxProjectPackages {
				return errors.New("cargo package targets are invalid")
			}
		}
		for _, value := range targets {
			target, ok := value.(map[string]any)
			if !ok {
				return errors.New("cargo package target table is invalid")
			}
			if value, exists := target["path"]; exists {
				value, ok := value.(string)
				if !ok || validateManifestPath(relativeManifest, value) != nil {
					return errors.New("cargo package target escapes project boundary")
				}
			}
		}
	}
	return nil
}

func validateManifestDependencies(value any, manifest string) error {
	if value == nil {
		return nil
	}
	dependencies, ok := value.(map[string]any)
	if !ok || len(dependencies) > maxProjectPackages {
		return errors.New("cargo dependency declarations are invalid")
	}
	for name, value := range dependencies {
		if !crateNamePattern.MatchString(name) {
			return errors.New("cargo dependency name is invalid")
		}
		if version, ok := value.(string); ok && boundedText(version, 4096) {
			continue
		}
		declaration, ok := value.(map[string]any)
		if !ok || len(declaration) == 0 {
			return errors.New("cargo dependency declaration is invalid")
		}
		for _, key := range []string{"git", "branch", "tag", "rev", "registry-index"} {
			if _, exists := declaration[key]; exists {
				return errors.New("cargo dependency source is unsupported")
			}
		}
		if registry, exists := declaration["registry"]; exists && registry != "crates-io" {
			return errors.New("cargo dependency registry is not public crates.io")
		}
		if value, exists := declaration["path"]; exists {
			value, ok := value.(string)
			if !ok || validateManifestPath(manifest, value) != nil {
				return errors.New("cargo dependency path escapes project boundary")
			}
		}
	}
	return nil
}

func validateManifestPath(manifest, value string) error {
	if !boundedText(value, 4096) || path.IsAbs(value) || strings.ContainsAny(value, "\\:\x00\r\n") {
		return errors.New("cargo manifest path is not relative bounded content")
	}
	joined := path.Join(path.Dir(manifest), value)
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return errors.New("cargo manifest path escapes project boundary")
	}
	return nil
}

// SelectedLocalManifests follows only declared workspace members and local
// dependencies. Other Cargo.toml files remain opaque, authenticated source data;
// a malformed test asset does not become a selected package by its filename.
func SelectedLocalManifests(files map[string][]byte) ([]string, error) {
	if len(files) > MaxCrateFiles {
		return nil, errors.New("cargo project file count exceeds bound")
	}
	queue := []string{"Cargo.toml"}
	seen := map[string]bool{}
	references := 0
	matchWork := 0
	match := func(pattern, directory string) (bool, error) {
		matchWork += (strings.Count(pattern, "/") + 1) * (strings.Count(directory, "/") + 1)
		if matchWork > 1<<20 {
			return false, errors.New("cargo member glob expansion exceeds work bound")
		}
		return cargoMemberMatch(pattern, directory)
	}
	add := func(manifest, directory string) error {
		if validateManifestPath(manifest, directory) != nil {
			return errors.New("cargo local manifest escapes source boundary")
		}
		references++
		if references > maxProjectEdges {
			return errors.New("cargo local manifest references exceed bound")
		}
		queue = append(queue, path.Join(path.Dir(manifest), directory, "Cargo.toml"))
		return nil
	}
	for len(queue) > 0 {
		manifest := queue[0]
		queue = queue[1:]
		if seen[manifest] {
			continue
		}
		body, present := files[manifest]
		if !present || ValidateProjectManifest(body, manifest) != nil {
			return nil, errors.New("cargo selected local manifest is missing or unsupported")
		}
		seen[manifest] = true
		if len(seen) > maxProjectPackages {
			return nil, errors.New("cargo local package count exceeds bound")
		}
		var document map[string]any
		if toml.Unmarshal(body, &document) != nil {
			return nil, errors.New("cargo selected manifest grammar is invalid")
		}
		dependencies := []map[string]any{document}
		if targets, ok := document["target"].(map[string]any); ok {
			for _, value := range targets {
				dependencies = append(dependencies, value.(map[string]any))
			}
		}
		if workspace, ok := document["workspace"].(map[string]any); ok {
			dependencies = append(dependencies, workspace)
			members, _ := workspace["members"].([]any)
			excludes, _ := workspace["exclude"].([]any)
			for _, value := range members {
				pattern := value.(string)
				for name := range files {
					if path.Base(name) != "Cargo.toml" {
						continue
					}
					base := path.Dir(manifest)
					candidate := path.Dir(name)
					if base != "." {
						var inside bool
						candidate, inside = strings.CutPrefix(candidate, base+"/")
						if !inside {
							continue
						}
					}
					matched, err := match(pattern, candidate)
					if err != nil {
						return nil, err
					}
					for _, value := range excludes {
						excluded, err := match(value.(string), candidate)
						if err != nil {
							return nil, err
						}
						if excluded {
							matched = false
						}
					}
					if matched {
						if err := add(manifest, candidate); err != nil {
							return nil, err
						}
					}
				}
			}
		}
		if table, ok := document["package"].(map[string]any); ok {
			if directory, ok := table["workspace"].(string); ok {
				if err := add(manifest, directory); err != nil {
					return nil, err
				}
			}
		}
		for _, table := range dependencies {
			for _, key := range []string{"dependencies", "dev-dependencies", "build-dependencies"} {
				entries, _ := table[key].(map[string]any)
				for _, value := range entries {
					if declaration, ok := value.(map[string]any); ok {
						if directory, ok := declaration["path"].(string); ok {
							if err := add(manifest, directory); err != nil {
								return nil, err
							}
						}
					}
				}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for manifest := range seen {
		result = append(result, manifest)
	}
	sort.Strings(result)
	return result, nil
}

func cargoMemberMatch(pattern, directory string) (bool, error) {
	parts, names := strings.Split(path.Clean(pattern), "/"), strings.Split(directory, "/")
	for _, part := range parts {
		if part != "**" {
			if strings.Contains(part, "**") {
				return false, errors.New("cargo recursive glob must occupy one component")
			}
			if _, err := path.Match(part, ""); err != nil {
				return false, errors.New("cargo member glob is invalid")
			}
		}
	}
	// Iterative dynamic programming bounds recursive glob matching; no backtracking.
	previous := make([]bool, len(names)+1)
	previous[0] = true
	for _, part := range parts {
		next := make([]bool, len(names)+1)
		if part == "**" {
			next[0] = previous[0]
		}
		for index, name := range names {
			if part == "**" {
				next[index+1] = previous[index+1] || next[index]
			} else {
				matched, _ := path.Match(part, name)
				next[index+1] = previous[index] && matched
			}
		}
		previous = next
	}
	return previous[len(names)], nil
}

// BuildProjectSnapshot preserves every selected public package, rather than
// projecting only the requested primary's dependency closure. Empty is explicit.
func BuildProjectSnapshot(install domain.InstallContext, metadata, manifest, lock []byte, root, runtimeIdentity string, projectFiles map[string][]byte) (domain.ProjectDependencySnapshot, error) {
	if !install.Valid() || !boundedText(runtimeIdentity, 4096) || ValidateProjectManifest(manifest, "Cargo.toml") != nil {
		return domain.ProjectDependencySnapshot{}, errors.New("cargo project snapshot request is invalid")
	}
	records, _, state, err := ParseLockedMetadata(metadata, lock, root)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	// Local manifests are already inside the complete anchored source inventory.
	// Metadata can select a copied relative member, never an arbitrary host read.
	if !bytes.Equal(projectFiles["Cargo.toml"], manifest) || !bytes.Equal(projectFiles["Cargo.lock"], lock) {
		return domain.ProjectDependencySnapshot{}, errors.New("cargo snapshot controls differ from source inventory")
	}
	var document projectMetadata
	if err := json.Unmarshal(metadata, &document); err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	localManifests := map[string]string{}
	for _, p := range document.Packages {
		if p.Source != nil {
			continue
		}
		relative, ok := strings.CutPrefix(p.ManifestPath, root+"/")
		body, exists := projectFiles[relative]
		if !ok || !exists || ValidateProjectManifest(body, relative) != nil {
			return domain.ProjectDependencySnapshot{}, errors.New("cargo selected local manifest is outside frozen source inventory")
		}
		hash := sha256.Sum256(body)
		localManifests[relative] = hex.EncodeToString(hash[:])
	}
	controls := make([]domain.ProjectControlDigest, 0, 2)
	for _, control := range []struct {
		name string
		body []byte
	}{{"Cargo.toml", manifest}, {"Cargo.lock", lock}} {
		hash := sha256.Sum256(control.body)
		digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		value, err := domain.NewProjectControlDigest(control.name, digest)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		controls = append(controls, value)
	}
	dependencies := make([]domain.ResolvedArtifact, 0, len(records))
	for _, record := range records {
		identity, err := domain.NewResolvedArtifactIdentity(Source(), record.Name, record.Version, "crate")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		endpoint, err := DownloadURL(record.Name, record.Version)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		artifact, err := domain.NewResolvedArtifact(identity, endpoint, "sha256="+record.Checksum)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		dependencies = append(dependencies, artifact)
	}
	canonical, err := json.Marshal(struct {
		Runtime        string            `json:"runtime"`
		Controls       []string          `json:"controls"`
		LocalManifests map[string]string `json:"local_manifests"`
		Graph          json.RawMessage   `json:"graph"`
	}{runtimeIdentity, []string{controls[0].Digest().String(), controls[1].Digest().String()}, localManifests, state})
	if err != nil {
		return domain.ProjectDependencySnapshot{}, errors.New("cargo snapshot normalization failed")
	}
	hash := sha256.Sum256(canonical)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if len(dependencies) == 0 {
		return domain.NewDependencyFreeProjectSnapshot(install, Source(), controls, digest)
	}
	return domain.NewProjectDependencySnapshot(install, Source(), controls, dependencies, digest)
}

// EqualControls compares exact bytes; it never supplies approval authority.
func EqualControls(a, b []domain.ProjectControlFile) bool {
	if len(a) != 2 || len(b) != 2 {
		return false
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, value := range a {
		if _, exists := byName[value.Name()]; exists {
			return false
		}
		byName[value.Name()] = value
	}
	for _, value := range b {
		expected, ok := byName[value.Name()]
		if !ok || expected.Present() != value.Present() || expected.Digest() != value.Digest() || !bytes.Equal(expected.Body(), value.Body()) {
			return false
		}
		delete(byName, value.Name())
	}
	return len(byName) == 0
}
