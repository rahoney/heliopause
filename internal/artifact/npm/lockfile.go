package npm

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const packageLockLimit = 8 << 20

// ParsePackageLockV3 converts the supported, isolated npm package-lock v3
// subset into a bounded Domain graph. It rejects unsupported lock semantics
// rather than silently approximating npm's resolver behavior.
func ParsePackageLockV3(reference domain.ArtifactReference, body []byte) (domain.LockedDependencyGraph, error) {
	if reference.Source().String() != "npm" {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock requires an npm artifact reference")
	}
	requested, err := parseLocator(reference.Locator())
	if err != nil {
		return domain.LockedDependencyGraph{}, fmt.Errorf("parse requested npm package: %w", err)
	}
	if len(body) == 0 || len(body) > packageLockLimit {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock exceeds supported bounds")
	}
	var lock packageLockV3
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&lock); err != nil {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock is invalid JSON")
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock contains trailing data")
	}
	if lock.LockfileVersion != 3 || len(lock.Packages) < 2 {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock must be version 3 with package entries")
	}
	root, ok := lock.Packages[""]
	if !ok || len(root.Dependencies) != 1 || hasUnsupportedPackageFields(root) {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock root semantics are unsupported")
	}
	if _, ok := root.Dependencies[requested.name]; !ok {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock does not contain the requested root dependency")
	}

	entries := make(map[string]parsedLockPackage, len(lock.Packages)-1)
	for packagePath, item := range lock.Packages {
		if packagePath == "" {
			continue
		}
		name, valid := packageNameFromLockPath(packagePath)
		if !valid || hasUnsupportedPackageFields(item) {
			return domain.LockedDependencyGraph{}, errors.New("npm package lock contains unsupported package semantics")
		}
		if item.Name != "" && item.Name != name {
			return domain.LockedDependencyGraph{}, errors.New("npm package lock package name does not match its path")
		}
		if !exactVersion.MatchString(item.Version) || !validRegistryTarball(item.Resolved) || !validSHA512SRI(item.Integrity) {
			return domain.LockedDependencyGraph{}, errors.New("npm package lock requires exact registry tarball and SHA-512 integrity")
		}
		entries[packagePath] = parsedLockPackage{name: name, value: item}
	}
	primaryPath := path.Join("node_modules", requested.name)
	primary, ok := entries[primaryPath]
	if !ok || primary.name != requested.name {
		return domain.LockedDependencyGraph{}, errors.New("npm package lock primary dependency is unavailable")
	}

	nodes := make(map[string]domain.LockedDependency, len(entries))
	for packagePath, entry := range entries {
		nodeID, err := lockNodeID(packagePath)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		identity, err := domain.NewResolvedArtifactIdentity(reference.Source(), entry.name, entry.value.Version, "tarball")
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		resolved, err := domain.NewResolvedArtifact(identity, entry.value.Resolved, entry.value.Integrity)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		role := domain.DependencyTransitive
		if packagePath == primaryPath {
			role = domain.DependencyPrimary
		}
		node, err := domain.NewLockedDependencyWithRecordPath(nodeID, role, resolved, packagePath, entry.value.HasInstallScript || entry.value.Gypfile)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		nodes[packagePath] = node
	}

	edges := make([]domain.DependencyEdge, 0)
	for packagePath, entry := range entries {
		for name := range entry.value.Dependencies {
			if err := validatePackageName(name); err != nil {
				return domain.LockedDependencyGraph{}, errors.New("npm package lock dependency name is invalid")
			}
			dependencyPath, ok := resolveLockedDependencyPath(packagePath, name, entries)
			if !ok {
				return domain.LockedDependencyGraph{}, errors.New("npm package lock dependency cannot be resolved exactly")
			}
			edge, err := domain.NewDependencyEdge(nodes[packagePath].Node(), nodes[dependencyPath].Node())
			if err != nil {
				return domain.LockedDependencyGraph{}, err
			}
			edges = append(edges, edge)
		}
	}
	nodeList := make([]domain.LockedDependency, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}
	return domain.NewLockedDependencyGraph(nodeList, edges)
}

type packageLockV3 struct {
	LockfileVersion int
	Packages        map[string]packageLockPackage
}

type packageLockPackage struct {
	Name                 string
	Version              string
	Resolved             string
	Integrity            string
	Dependencies         map[string]string
	OptionalDependencies map[string]string
	PeerDependencies     map[string]string
	OS                   []string
	CPU                  []string
	Link                 bool
	InBundle             bool
	HasInstallScript     bool
	Gypfile              bool
	Optional             bool
	Dev                  bool
	DevOptional          bool
}

type parsedLockPackage struct {
	name  string
	value packageLockPackage
}

func hasUnsupportedPackageFields(item packageLockPackage) bool {
	return len(item.OptionalDependencies) != 0 || len(item.PeerDependencies) != 0 || len(item.OS) != 0 || len(item.CPU) != 0 ||
		item.Link || item.InBundle || item.Optional || item.Dev || item.DevOptional
}

func packageNameFromLockPath(value string) (string, bool) {
	if value == "" || path.Clean(value) != value || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "../") {
		return "", false
	}
	segments := strings.Split(value, "/")
	for index := 0; index < len(segments); {
		if segments[index] != "node_modules" || index+1 >= len(segments) {
			return "", false
		}
		index++
		if strings.HasPrefix(segments[index], "@") {
			if index+1 >= len(segments) {
				return "", false
			}
			index += 2
		} else {
			index++
		}
		if index == len(segments) {
			break
		}
	}
	last := strings.LastIndex(value, "/node_modules/")
	name := value
	if last >= 0 {
		name = value[last+len("/node_modules/"):]
	} else if strings.HasPrefix(value, "node_modules/") {
		name = strings.TrimPrefix(value, "node_modules/")
	}
	return name, validatePackageName(name) == nil
}

func lockNodeID(packagePath string) (domain.DependencyNodeID, error) {
	digest := sha256.Sum256([]byte(packagePath))
	return domain.NewDependencyNodeID("lock-" + hex.EncodeToString(digest[:16]))
}

func validRegistryTarball(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && parsed.Scheme == "https" && strings.EqualFold(parsed.Host, "registry.npmjs.org") &&
		parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" && strings.HasPrefix(parsed.Path, "/") && strings.HasSuffix(parsed.Path, ".tgz")
}

func validSHA512SRI(value string) bool {
	if !strings.HasPrefix(value, "sha512-") {
		return false
	}
	encoded := strings.TrimPrefix(value, "sha512-")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	return err == nil && len(decoded) == sha256.Size*2 && base64.StdEncoding.EncodeToString(decoded) == encoded
}

func resolveLockedDependencyPath(fromPath, name string, entries map[string]parsedLockPackage) (string, bool) {
	base := fromPath
	for {
		candidate := path.Join(base, "node_modules", name)
		if _, ok := entries[candidate]; ok {
			return candidate, true
		}
		next := parentPackagePath(base)
		if next == base {
			return "", false
		}
		base = next
	}
}

func parentPackagePath(value string) string {
	index := strings.LastIndex(value, "/node_modules/")
	if index >= 0 {
		return value[:index]
	}
	return ""
}
