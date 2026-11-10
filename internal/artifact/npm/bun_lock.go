package npm

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"maps"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const bunLockNodeLimit = 4096
const bunLockEdgeLimit = 16384

// BunLock is an unapproved native tree selection, not a verified archive set.
// Tree locations remain separate even when they share one registry archive.
type BunLock struct {
	valid          bool
	version        string
	roots          []BunLockedRoot
	packages       []BunLockedPackage
	manifestDigest [sha256.Size]byte
	lockDigest     domain.ContentDigest
	lockPresent    bool
}

type BunLockedRoot struct{ Group, Name, Specifier, Instance string }

type BunLockedPackage struct {
	Key, Name, Version, Registry, Integrity string
	Dependencies, PeerDependencies          map[string]string
	Targets, Bins                           map[string]string
	HasInstallScript                        bool
}

func (l BunLock) DependencyFree() bool   { return l.valid && len(l.roots) == 0 }
func (l BunLock) Roots() []BunLockedRoot { return append([]BunLockedRoot(nil), l.roots...) }
func (l BunLock) Packages() []BunLockedPackage {
	result := append([]BunLockedPackage(nil), l.packages...)
	for i := range result {
		result[i].Dependencies = maps.Clone(result[i].Dependencies)
		result[i].PeerDependencies = maps.Clone(result[i].PeerDependencies)
		result[i].Targets = maps.Clone(result[i].Targets)
		result[i].Bins = maps.Clone(result[i].Bins)
	}
	return result
}

// ParseBunLock accepts the observed v2/config1 public-registry subset. The
// control's explicit presence distinguishes an absent empty lock from a failed
// transport or a present zero-byte file. No lock is manufactured for empty input.
func ParseBunLock(manifest []byte, control domain.ProjectControlFile, expectedClientVersion string) (BunLock, error) {
	name, groups, err := bunManifestConstraints(manifest, expectedClientVersion)
	if err != nil || control.Name() != "bun.lock" {
		return BunLock{}, errBunLock
	}
	selection := BunLock{version: expectedClientVersion, manifestDigest: sha256.Sum256(manifest), lockDigest: control.Digest(), lockPresent: control.Present()}
	if !control.Present() {
		if len(control.Body()) != 0 || len(groups["dependencies"])+len(groups["devDependencies"]) != 0 {
			return BunLock{}, errBunLock
		}
		selection.valid = true
		return selection, nil
	}
	body, err := normalizeBunLockJSON(control.Body())
	if err != nil {
		return BunLock{}, errBunLock
	}
	wire, err := bunJSONObject(body, "lockfileVersion", "configVersion", "workspaces", "packages")
	if err != nil || string(bytes.TrimSpace(wire["lockfileVersion"])) != "2" || string(bytes.TrimSpace(wire["configVersion"])) != "1" {
		return BunLock{}, errBunLock
	}
	workspaces, err := bunJSONObject(wire["workspaces"])
	if err != nil || len(workspaces) != 1 {
		return BunLock{}, errBunLock
	}
	workspace, err := bunJSONObject(workspaces[""], "name", "dependencies", "devDependencies")
	if err != nil {
		return BunLock{}, errBunLock
	}
	rootName := ""
	if value, exists := workspace["name"]; exists {
		var ok bool
		rootName, ok = bunJSONString(value)
		if !ok {
			return BunLock{}, errBunLock
		}
	}
	if rootName != name {
		return BunLock{}, errBunLock
	}
	for _, group := range []string{"dependencies", "devDependencies"} {
		dependencies, err := bunOptionalConstraints(workspace, group)
		if err != nil || !maps.Equal(dependencies, groups[group]) {
			return BunLock{}, errBunLock
		}
		for pkg, spec := range dependencies {
			selection.roots = append(selection.roots, BunLockedRoot{group, pkg, spec, pkg})
		}
	}
	packages, err := bunJSONObject(wire["packages"])
	if err != nil || len(packages) > bunLockNodeLimit {
		return BunLock{}, errBunLock
	}
	byKey := make(map[string]BunLockedPackage, len(packages))
	unique := map[string]BunLockedPackage{}
	for key, value := range packages {
		pkg, err := parseBunPackage(key, value)
		if err != nil {
			return BunLock{}, errBunLock
		}
		identity := pkg.Name + "@" + pkg.Version
		if previous, exists := unique[identity]; exists && !sameBunPackageMetadata(previous, pkg) {
			return BunLock{}, errBunLock
		}
		unique[identity], byKey[key] = pkg, pkg
	}
	if err := bindBunTree(&selection, byKey); err != nil {
		return BunLock{}, errBunLock
	}
	sort.Slice(selection.roots, func(i, j int) bool {
		a, b := selection.roots[i], selection.roots[j]
		return a.Group+"/"+a.Name < b.Group+"/"+b.Name
	})
	sort.Slice(selection.packages, func(i, j int) bool { return selection.packages[i].Key < selection.packages[j].Key })
	selection.valid = true
	return selection, nil
}

func bunManifestConstraints(body []byte, version string) (string, map[string]map[string]string, error) {
	if len(version) > 128 || !exactVersion.MatchString(version) || !utf8.Valid(body) || strictControlJSON(body) != nil {
		return "", nil, errBunLock
	}
	fields, err := bunJSONObject(body)
	if err != nil {
		return "", nil, err
	}
	for _, key := range []string{"optionalDependencies", "peerDependencies", "workspaces", "bun", "overrides", "resolutions", "bundledDependencies", "bundleDependencies", "devEngines", "trustedDependencies", "catalog", "catalogs", "patchedDependencies"} {
		if value, exists := fields[key]; exists {
			object := true
			switch key {
			case "workspaces", "bundledDependencies", "bundleDependencies", "trustedDependencies":
				object = false
			}
			if !emptyBunAggregate(value, object) {
				return "", nil, errBunLock
			}
		}
	}
	if value, exists := fields["packageManager"]; exists {
		manager, ok := bunJSONString(value)
		if !ok || manager != "bun@"+version {
			return "", nil, errBunLock
		}
	}
	if value, exists := fields["engines"]; exists {
		engines, err := bunJSONObject(value)
		if err != nil {
			return "", nil, errBunLock
		}
		for key, value := range engines {
			if _, ok := bunJSONString(value); !ok || key == "runtime" {
				return "", nil, errBunLock
			}
		}
	}
	name := ""
	if value, exists := fields["name"]; exists {
		var ok bool
		name, ok = bunJSONString(value)
		if !ok || validatePackageName(name) != nil {
			return "", nil, errBunLock
		}
	}
	groups := map[string]map[string]string{}
	seen := map[string]bool{}
	for _, group := range []string{"dependencies", "devDependencies"} {
		dependencies, err := bunOptionalConstraints(fields, group)
		if err != nil {
			return "", nil, err
		}
		for pkg := range dependencies {
			if seen[pkg] {
				return "", nil, errBunLock
			}
			seen[pkg] = true
		}
		groups[group] = dependencies
	}
	if len(seen) > 256 {
		return "", nil, errBunLock
	}
	return name, groups, nil
}

// Callers already validated JSON. An empty object and empty array are distinct;
// whitespace inside either remains valid and does not introduce a setting.
func emptyBunAggregate(body []byte, object bool) bool {
	body = bytes.TrimSpace(body)
	open, close := byte('['), byte(']')
	if object {
		open, close = '{', '}'
	}
	return len(body) >= 2 && body[0] == open && body[len(body)-1] == close && len(bytes.TrimSpace(body[1:len(body)-1])) == 0
}

func bunJSONObject(body []byte, allowed ...string) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(body)
	var result map[string]json.RawMessage
	if len(trimmed) == 0 || trimmed[0] != '{' || json.Unmarshal(trimmed, &result) != nil || result == nil {
		return nil, errBunLock
	}
	for key := range result {
		if len(allowed) == 0 {
			continue
		}
		known := false
		for _, name := range allowed {
			known = known || key == name
		}
		if !known {
			return nil, errBunLock
		}
	}
	return result, nil
}

func bunJSONString(body []byte) (string, bool) {
	body = bytes.TrimSpace(body)
	var result string
	returnValue := len(body) > 0 && body[0] == '"' && json.Unmarshal(body, &result) == nil
	return result, returnValue
}

func bunOptionalConstraints(object map[string]json.RawMessage, key string) (map[string]string, error) {
	result := map[string]string{}
	body, exists := object[key]
	if !exists {
		return result, nil
	}
	fields, err := bunJSONObject(body)
	if err != nil {
		return nil, err
	}
	for name, value := range fields {
		spec, ok := bunJSONString(value)
		if !ok || validatePackageName(name) != nil || spec != strings.TrimSpace(spec) || !publicDependencySpec.MatchString(spec) || !supportedBunConstraint(spec) {
			return nil, errBunLock
		}
		result[name] = spec
	}
	return result, nil
}

func sameBunPackageMetadata(a, b BunLockedPackage) bool {
	return a.Name == b.Name && a.Version == b.Version && a.Registry == b.Registry && a.Integrity == b.Integrity && a.HasInstallScript == b.HasInstallScript && maps.Equal(a.Dependencies, b.Dependencies) && maps.Equal(a.PeerDependencies, b.PeerDependencies) && maps.Equal(a.Bins, b.Bins)
}
