package npm

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"sort"
	"strings"
)

var errNPMProjectLock = errors.New("npm complete project lock is invalid, incomplete or unsupported")

// NPMProjectLock preserves every native placement and peer edge. Unique archive
// selection and target applicability are separate from lock identity.
type NPMProjectLock struct {
	valid                bool
	version              string
	manifestSHA, lockSHA [sha256.Size]byte
	roots                []NPMProjectRoot
	packages             []NPMProjectPackage
}

type NPMProjectRoot struct {
	Group, Name, Specifier, Location string
}

type NPMProjectPackage struct {
	Location, Name, Version, Registry, Integrity         string
	Dependencies, OptionalDependencies, PeerDependencies map[string]string
	OptionalPeers                                        map[string]bool
	OS, CPU, LibC                                        []string
	Targets, Bins                                        map[string]string
	HasInstallScript                                     bool
}

func (l NPMProjectLock) DependencyFree() bool { return l.valid && len(l.packages) == 0 }
func (l NPMProjectLock) Roots() []NPMProjectRoot {
	return append([]NPMProjectRoot(nil), l.roots...)
}
func (l NPMProjectLock) Packages() []NPMProjectPackage {
	return cloneNPMProjectPackages(l.packages)
}

func cloneNPMProjectPackages(packages []NPMProjectPackage) []NPMProjectPackage {
	result := append([]NPMProjectPackage(nil), packages...)
	for i := range result {
		result[i].Dependencies = maps.Clone(result[i].Dependencies)
		result[i].OptionalDependencies = maps.Clone(result[i].OptionalDependencies)
		result[i].OptionalPeers = maps.Clone(result[i].OptionalPeers)
		result[i].OS = slices.Clone(result[i].OS)
		result[i].CPU = slices.Clone(result[i].CPU)
		result[i].LibC = slices.Clone(result[i].LibC)
		result[i].PeerDependencies = maps.Clone(result[i].PeerDependencies)
		result[i].Targets = maps.Clone(result[i].Targets)
		result[i].Bins = maps.Clone(result[i].Bins)
	}
	return result
}

type npmProjectPackageWire struct {
	packageLockPackage
	PeerDependenciesMeta map[string]struct{ Optional bool }
	LibC                 []string
	Bin                  json.RawMessage
}

type npmProjectEdge struct {
	target   string
	optional bool
}

// ParseNPMProjectLock uses the supported pinned Linux amd64/glibc runtime.
// Platform strings describe that capability, not independently pinned versions.
// The native client must first reconcile install, or validate frozen ci, controls.
func ParseNPMProjectLock(manifest, body []byte, clientVersion string) (NPMProjectLock, error) {
	groups, err := npmProjectConstraints(manifest, clientVersion)
	if err != nil || !exactVersion.MatchString(clientVersion) {
		return NPMProjectLock{}, errNPMProjectLock
	}
	lockedGroups, err := npmProjectLockConstraints(body, clientVersion)
	if err != nil || !sameNPMProjectConstraints(groups, lockedGroups) {
		return NPMProjectLock{}, errNPMProjectLock
	}
	var lock struct{ Packages map[string]json.RawMessage }
	if json.Unmarshal(body, &lock) != nil {
		return NPMProjectLock{}, errNPMProjectLock
	}
	entries := make(map[string]parsedLockPackage, len(lock.Packages)-1)
	wire := make(map[string]npmProjectPackageWire, len(lock.Packages)-1)
	applicable := make(map[string]bool, len(lock.Packages)-1)
	for location, value := range lock.Packages {
		if location == "" {
			continue
		}
		name, valid := packageNameFromLockPath(location)
		var item npmProjectPackageWire
		if !valid || json.Unmarshal(value, &item) != nil || len(location) > 4096 || strings.Count(location, "node_modules/") > 32 {
			return NPMProjectLock{}, errNPMProjectLock
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(value, &fields) != nil || !validNPMProjectPeerMeta(fields) {
			return NPMProjectLock{}, errNPMProjectLock
		}
		applicable[location] = true
		for _, platform := range []struct {
			values []string
			target string
		}{{item.OS, "linux"}, {item.CPU, "x64"}, {item.LibC, "glibc"}} {
			matches, err := npmProjectPlatformMatches(platform.values, platform.target)
			if err != nil {
				return NPMProjectLock{}, err
			}
			applicable[location] = applicable[location] && matches
		}
		entries[location] = parsedLockPackage{name, item.packageLockPackage}
		wire[location] = item
	}
	var rootFields map[string]json.RawMessage
	if json.Unmarshal(manifest, &rootFields) != nil || !validNPMProjectPeerMeta(rootFields) {
		return NPMProjectLock{}, errNPMProjectLock
	}
	var rootMeta map[string]struct{ Optional bool }
	if raw, exists := rootFields["peerDependenciesMeta"]; exists {
		if json.Unmarshal(raw, &rootMeta) != nil {
			return NPMProjectLock{}, errNPMProjectLock
		}
	}
	selection := NPMProjectLock{version: clientVersion, manifestSHA: sha256.Sum256(manifest), lockSHA: sha256.Sum256(body)}
	fullEdges := map[string]map[string]npmProjectEdge{}
	edgeCount := 0
	bind := func(location string, dependencies, optional, peers map[string]string, peerMeta map[string]struct{ Optional bool }) (map[string]npmProjectEdge, error) {
		edges := map[string]npmProjectEdge{}
		for _, group := range []struct {
			values   map[string]string
			optional bool
			peer     bool
		}{{dependencies, false, false}, {optional, true, false}, {peers, false, true}} {
			for name := range group.values {
				// npm optionalDependencies override an equally named dependency.
				if !group.optional && !group.peer {
					if _, overridden := optional[name]; overridden {
						continue
					}
				}
				edgeCount++
				if edgeCount > pnpmLockEdgeLimit {
					return nil, errNPMProjectLock
				}
				from := location
				if group.peer && location != "" {
					from = parentPackagePath(location)
				}
				target, exists := resolveLockedDependencyPath(from, name, entries)
				mayOmit := group.optional || group.peer && peerMeta[name].Optional
				if !exists && !mayOmit {
					return nil, errNPMProjectLock
				}
				if previous, duplicate := edges[name]; duplicate {
					if previous.target != target {
						return nil, errNPMProjectLock
					}
					mayOmit = previous.optional && mayOmit
				}
				edges[name] = npmProjectEdge{target, mayOmit}
			}
		}
		return edges, nil
	}
	for location, item := range wire {
		parent := parentPackagePath(location)
		if parent != "" {
			if _, exists := entries[parent]; !exists {
				return NPMProjectLock{}, errNPMProjectLock
			}
		}
		edges, err := bind(location, item.Dependencies, item.OptionalDependencies, item.PeerDependencies, item.PeerDependenciesMeta)
		if err != nil {
			return NPMProjectLock{}, err
		}
		fullEdges[location] = edges
	}
	// All default dependency groups are selected. Native npm owns their range
	// solving; a name in more than one group is still one installed placement.
	rootDependencies := maps.Clone(groups["dependencies"])
	if rootDependencies == nil {
		rootDependencies = map[string]string{}
	}
	for name, spec := range groups["devDependencies"] {
		rootDependencies[name] = spec
	}
	rootEdges, err := bind("", rootDependencies, groups["optionalDependencies"], groups["peerDependencies"], rootMeta)
	if err != nil {
		return NPMProjectLock{}, err
	}
	for _, group := range npmProjectDependencyGroups {
		for name, spec := range groups[group] {
			selection.roots = append(selection.roots, NPMProjectRoot{group, name, spec, rootEdges[name].target})
		}
	}
	// The complete lock must contain no unreachable/foreign placement. Keep
	// platform-excluded optional branches in this structural check as well.
	reachable := map[string]bool{}
	queue := []string{}
	for _, edge := range rootEdges {
		if edge.target != "" && !reachable[edge.target] {
			reachable[edge.target] = true
			queue = append(queue, edge.target)
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, edge := range fullEdges[queue[i]] {
			if edge.target != "" && !reachable[edge.target] {
				reachable[edge.target] = true
				queue = append(queue, edge.target)
			}
		}
	}
	if len(reachable) != len(entries) {
		return NPMProjectLock{}, errNPMProjectLock
	}
	selected := map[string]bool{}
	queue = nil
	add := func(edge npmProjectEdge) error {
		if edge.target == "" {
			if !edge.optional {
				return errNPMProjectLock
			}
			return nil
		}
		if !applicable[edge.target] {
			if !edge.optional {
				return errNPMProjectLock
			}
			return nil
		}
		if !selected[edge.target] {
			selected[edge.target] = true
			queue = append(queue, edge.target)
		}
		return nil
	}
	for _, edge := range rootEdges {
		if err := add(edge); err != nil {
			return NPMProjectLock{}, err
		}
	}
	for i := 0; i < len(queue); i++ {
		location := queue[i]
		item := wire[location]
		pkg := NPMProjectPackage{Location: location, Name: entries[location].name, Version: item.Version, Registry: item.Resolved, Integrity: item.Integrity, Dependencies: maps.Clone(item.Dependencies), OptionalDependencies: maps.Clone(item.OptionalDependencies), PeerDependencies: maps.Clone(item.PeerDependencies), OptionalPeers: map[string]bool{}, OS: slices.Clone(item.OS), CPU: slices.Clone(item.CPU), LibC: slices.Clone(item.LibC), Targets: map[string]string{}, Bins: map[string]string{}, HasInstallScript: item.HasInstallScript || item.Gypfile}
		for name, meta := range item.PeerDependenciesMeta {
			pkg.OptionalPeers[name] = meta.Optional
		}
		for name, edge := range fullEdges[location] {
			if err := add(edge); err != nil {
				return NPMProjectLock{}, err
			}
			if selected[edge.target] {
				pkg.Targets[name] = edge.target
			}
		}
		if len(item.Bin) != 0 {
			pkg.Bins, err = bunPackageBins(item.Bin, pkg.Name)
			if err != nil {
				return NPMProjectLock{}, errNPMProjectLock
			}
		}
		selection.packages = append(selection.packages, pkg)
	}
	sort.Slice(selection.roots, func(i, j int) bool {
		return selection.roots[i].Group+"/"+selection.roots[i].Name < selection.roots[j].Group+"/"+selection.roots[j].Name
	})
	sort.Slice(selection.packages, func(i, j int) bool { return selection.packages[i].Location < selection.packages[j].Location })
	selection.valid = true
	return selection, nil
}

func validNPMProjectPeerMeta(fields map[string]json.RawMessage) bool {
	raw, exists := fields["peerDependenciesMeta"]
	if !exists {
		return true
	}
	var meta map[string]struct{ Optional bool }
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&meta) != nil || meta == nil {
		return false
	}
	for name := range meta {
		if validatePackageName(name) != nil {
			return false
		}
	}
	return true
}

func npmProjectPlatformMatches(values []string, target string) (bool, error) {
	allowed, hasPositive, excluded := len(values) == 0, false, false
	for _, value := range values {
		if !packageSegment.MatchString(strings.TrimPrefix(value, "!")) {
			return false, errNPMProjectLock
		}
		if strings.HasPrefix(value, "!") {
			if value[1:] == target || value == "!any" {
				excluded = true
			}
			continue
		}
		hasPositive = true
		if value == target || value == "any" {
			allowed = true
		}
	}
	return !excluded && (allowed || !hasPositive), nil
}
