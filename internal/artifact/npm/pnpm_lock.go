package npm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const pnpmLockLimit = ProjectControlLimit
const pnpmLockNodeLimit = 4096
const pnpmLockEdgeLimit = 16384

var errPNPMLock = errors.New("pnpm public lock selection is invalid or unsupported")

// PNPMLock is a bounded, unapproved native selection. Package bytes still
// require independent registry binding and the ordinary quarantine workflow.
// Peer-qualified snapshots remain distinct from per-version package metadata.
type PNPMLock struct {
	valid     bool
	roots     []PNPMLockedRoot
	packages  []PNPMLockedPackage
	snapshots []PNPMLockedSnapshot
}

func (l PNPMLock) DependencyFree() bool { return l.valid && len(l.roots) == 0 }

type PNPMLockedRoot struct{ Group, Name, Specifier, Snapshot string }

type PNPMLockedPackage struct {
	Key, Name, Version, Integrity, Tarball, Deprecated string
	HasBin, Prepare                                    bool
	Engines, PeerDependencies                          map[string]string
}

type PNPMLockedSnapshot struct {
	Key, Package    string
	Dependencies    map[string]string
	TransitivePeers []string
}

func (l PNPMLock) Roots() []PNPMLockedRoot { return append([]PNPMLockedRoot(nil), l.roots...) }

func (l PNPMLock) Packages() []PNPMLockedPackage {
	result := append([]PNPMLockedPackage(nil), l.packages...)
	for i := range result {
		result[i].Engines = clonePNPMStrings(result[i].Engines)
		result[i].PeerDependencies = clonePNPMStrings(result[i].PeerDependencies)
	}
	return result
}

func (l PNPMLock) Snapshots() []PNPMLockedSnapshot {
	result := append([]PNPMLockedSnapshot(nil), l.snapshots...)
	for i := range result {
		result[i].Dependencies = clonePNPMStrings(result[i].Dependencies)
		result[i].TransitivePeers = append([]string(nil), result[i].TransitivePeers...)
	}
	return result
}

func clonePNPMStrings(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

// ParsePNPMLock accepts the reviewed registry/resource-integrity v9 subset.
// It does not infer tarball URLs or manufacture a primary for a whole project.
func ParsePNPMLock(manifest, body []byte, expectedClientVersion string) (PNPMLock, error) {
	constraints, err := pnpmManifestConstraints(manifest, expectedClientVersion)
	if err != nil || len(body) == 0 || len(body) > pnpmLockLimit {
		return PNPMLock{}, errPNPMLock
	}
	if err := validatePNPMYAML(body); err != nil {
		return PNPMLock{}, err
	}
	var wire pnpmLockWire
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	decoder.KnownFields(true)
	if err := decoder.Decode(&wire); err != nil || wire.Version != "9.0" || wire.Settings == nil ||
		wire.Settings.AutoInstallPeers == nil || !*wire.Settings.AutoInstallPeers ||
		wire.Settings.ExcludeLinks == nil || *wire.Settings.ExcludeLinks ||
		wire.Settings.IntegrityTarget != "" && wire.Settings.IntegrityTarget != "resource" ||
		len(wire.Importers) != 1 || len(wire.Packages) > pnpmLockNodeLimit || len(wire.Snapshots) > pnpmLockNodeLimit {
		return PNPMLock{}, errPNPMLock
	}
	importer, ok := wire.Importers["."]
	if !ok || len(importer.Optional) != 0 {
		return PNPMLock{}, errPNPMLock
	}
	selection := PNPMLock{}
	for _, group := range []struct {
		name string
		deps map[string]pnpmRootWire
	}{{"dependencies", importer.Dependencies}, {"devDependencies", importer.Dev}} {
		if len(group.deps) != len(constraints[group.name]) {
			return PNPMLock{}, errPNPMLock
		}
		for name, root := range group.deps {
			if spec, exists := constraints[group.name][name]; !exists || root.Specifier != spec {
				return PNPMLock{}, errPNPMLock
			}
			key := name + "@" + root.Version
			if _, _, _, err := pnpmPackageKey(key); err != nil {
				return PNPMLock{}, err
			}
			if _, exists := wire.Snapshots[key]; !exists {
				return PNPMLock{}, errPNPMLock
			}
			selection.roots = append(selection.roots, PNPMLockedRoot{group.name, name, root.Specifier, key})
		}
	}
	for key, value := range wire.Packages {
		name, version, base, err := pnpmPackageKey(key)
		if err != nil || base != key || !validSHA512SRI(value.Resolution.Integrity) ||
			value.Resolution.Tarball != "" && !validRegistryTarball(value.Resolution.Tarball) ||
			len(value.Deprecated) > 4096 || !pnpmStringMetadata(value.Engines, false) || !pnpmStringMetadata(value.Peers, true) {
			return PNPMLock{}, errPNPMLock
		}
		selection.packages = append(selection.packages, PNPMLockedPackage{key, name, version, value.Resolution.Integrity, value.Resolution.Tarball, value.Deprecated, value.HasBin, value.Prepare, value.Engines, value.Peers})
	}
	edges := 0
	usedPackages := map[string]bool{}
	for key, value := range wire.Snapshots {
		_, _, base, err := pnpmPackageKey(key)
		metadata, exists := wire.Packages[base]
		if err != nil || !exists || len(value.Optional) != 0 || value.IsOptional || value.Patched || value.ID != "" || len(value.TransitivePeers) > 256 {
			return PNPMLock{}, errPNPMLock
		}
		usedPackages[base] = true
		dependencies := make(map[string]string, len(value.Dependencies))
		for name, version := range value.Dependencies {
			target := name + "@" + version
			if _, _, _, err := pnpmPackageKey(target); err != nil || target == key {
				return PNPMLock{}, errPNPMLock
			}
			if _, exists := wire.Snapshots[target]; !exists {
				return PNPMLock{}, errPNPMLock
			}
			dependencies[name] = target
			edges++
			if edges > pnpmLockEdgeLimit {
				return PNPMLock{}, errPNPMLock
			}
		}
		for name := range metadata.Peers {
			if _, exists := dependencies[name]; !exists {
				return PNPMLock{}, errPNPMLock
			}
		}
		seen := map[string]bool{}
		for _, peer := range value.TransitivePeers {
			if validatePackageName(peer) != nil || seen[peer] {
				return PNPMLock{}, errPNPMLock
			}
			seen[peer] = true
		}
		sort.Strings(value.TransitivePeers)
		selection.snapshots = append(selection.snapshots, PNPMLockedSnapshot{key, base, dependencies, value.TransitivePeers})
	}
	if edges > pnpmLockEdgeLimit || len(usedPackages) != len(wire.Packages) || !pnpmAllReachable(selection) {
		return PNPMLock{}, errPNPMLock
	}
	sort.Slice(selection.roots, func(i, j int) bool {
		a, b := selection.roots[i], selection.roots[j]
		return a.Group+"/"+a.Name < b.Group+"/"+b.Name
	})
	sort.Slice(selection.packages, func(i, j int) bool { return selection.packages[i].Key < selection.packages[j].Key })
	sort.Slice(selection.snapshots, func(i, j int) bool { return selection.snapshots[i].Key < selection.snapshots[j].Key })
	selection.valid = true
	return selection, nil
}

func pnpmManifestConstraints(body []byte, expectedClientVersion string) (map[string]map[string]string, error) {
	if !exactVersion.MatchString(expectedClientVersion) || strictControlJSON(body) != nil {
		return nil, errPNPMLock
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return nil, errPNPMLock
	}
	for _, key := range []string{"optionalDependencies", "peerDependencies", "workspaces", "pnpm", "overrides", "resolutions", "bundledDependencies", "bundleDependencies", "devEngines"} {
		if value, exists := fields[key]; exists && string(bytes.TrimSpace(value)) != "{}" && string(bytes.TrimSpace(value)) != "[]" {
			return nil, errPNPMLock
		}
	}
	if value, exists := fields["packageManager"]; exists {
		var manager string
		if json.Unmarshal(value, &manager) != nil || manager != "pnpm@"+expectedClientVersion {
			return nil, errPNPMLock
		}
	}
	groups := map[string]map[string]string{}
	seen := map[string]bool{}
	for _, name := range []string{"dependencies", "devDependencies"} {
		var group map[string]string
		if value, exists := fields[name]; exists {
			if json.Unmarshal(value, &group) != nil || group == nil {
				return nil, errPNPMLock
			}
		}
		for pkg, spec := range group {
			if validatePackageName(pkg) != nil || seen[pkg] || spec != strings.TrimSpace(spec) || !publicDependencySpec.MatchString(spec) {
				return nil, errPNPMLock
			}
			seen[pkg] = true
		}
		groups[name] = group
	}
	if len(seen) > 256 {
		return nil, errPNPMLock
	}
	return groups, nil
}

func pnpmPackageKey(key string) (name, version, base string, err error) {
	if len(key) == 0 || len(key) > 2048 {
		return "", "", "", errPNPMLock
	}
	base = key
	if index := strings.IndexByte(key, '('); index >= 0 {
		base = key[:index]
		depth := 0
		previous := rune(0)
		for _, r := range key[index:] {
			switch r {
			case '(':
				depth++
				if depth > 32 {
					return "", "", "", errPNPMLock
				}
			case ')':
				depth--
				if depth < 0 || previous == '(' {
					return "", "", "", errPNPMLock
				}
			default:
				if depth == 0 || !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("@/._+-=", r)) {
					return "", "", "", errPNPMLock
				}
			}
			previous = r
		}
		if depth != 0 {
			return "", "", "", errPNPMLock
		}
	}
	index := strings.LastIndexByte(base, '@')
	if index <= 0 || validatePackageName(base[:index]) != nil || !exactVersion.MatchString(base[index+1:]) {
		return "", "", "", errPNPMLock
	}
	return base[:index], base[index+1:], base, nil
}

func pnpmStringMetadata(values map[string]string, packageNames bool) bool {
	if len(values) > 256 {
		return false
	}
	for key, value := range values {
		if key == "" || len(key) > 214 || len(value) == 0 || len(value) > 512 || strings.ContainsAny(key+value, "\x00\r\n") || packageNames && validatePackageName(key) != nil {
			return false
		}
	}
	return true
}

func pnpmAllReachable(lock PNPMLock) bool {
	byKey := make(map[string]PNPMLockedSnapshot, len(lock.snapshots))
	for _, snapshot := range lock.snapshots {
		byKey[snapshot.Key] = snapshot
	}
	visited := map[string]bool{}
	queue := make([]string, 0, len(lock.roots))
	for _, root := range lock.roots {
		queue = append(queue, root.Snapshot)
	}
	for len(queue) != 0 {
		key := queue[0]
		queue = queue[1:]
		if visited[key] {
			continue
		}
		visited[key] = true
		for _, target := range byKey[key].Dependencies {
			queue = append(queue, target)
		}
	}
	return len(visited) == len(byKey)
}

// Validate the YAML node tree before decoding maps: aliases/merge expansion,
// duplicate keys and ambiguous tags never reach the semantic interpretation.
func validatePNPMYAML(body []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	var document yaml.Node
	if decoder.Decode(&document) != nil || document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return errPNPMLock
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errPNPMLock
	}
	count := 0
	var visit func(*yaml.Node, int) bool
	visit = func(node *yaml.Node, depth int) bool {
		count++
		if count > 131072 || depth > 64 || node.Anchor != "" || node.Alias != nil || node.Kind == yaml.AliasNode {
			return false
		}
		switch node.Kind {
		case yaml.MappingNode:
			if node.Tag != "!!map" || len(node.Content)%2 != 0 {
				return false
			}
			seen := map[string]bool{}
			for i := 0; i < len(node.Content); i += 2 {
				key := node.Content[i]
				if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" || seen[key.Value] {
					return false
				}
				seen[key.Value] = true
			}
		case yaml.SequenceNode:
			if node.Tag != "!!seq" {
				return false
			}
		case yaml.ScalarNode:
			if node.Tag != "!!str" && node.Tag != "!!bool" && node.Tag != "!!int" && node.Tag != "!!float" {
				return false
			}
		default:
			return false
		}
		for _, child := range node.Content {
			if !visit(child, depth+1) {
				return false
			}
		}
		return true
	}
	if !visit(document.Content[0], 0) {
		return errPNPMLock
	}
	return nil
}
