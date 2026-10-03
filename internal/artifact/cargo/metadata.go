// Package cargo owns the normalized public crates.io metadata boundary.
// Registry identity, checksums and graph edges are retained; Cargo's raw JSON
// and configuration are never promoted to Core or Evidence.
package cargo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	registrySourceURL = "registry+https://github.com/rust-lang/crates.io-index"
	crateDownloadHost = "https://static.crates.io/crates"
	maxMetadataBytes  = 8 << 20
)

var (
	crateNamePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
	crateVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	crateSource         = mustSource("crates-io")
)

func Source() domain.SourceID { return crateSource }

// ParseReference accepts only an exact public crates.io crate/version.
func ParseReference(value string) (domain.ArtifactReference, error) {
	if strings.Count(value, "@") != 1 {
		return domain.ArtifactReference{}, errors.New("crate reference requires crate@version")
	}
	parts := strings.SplitN(value, "@", 2)
	if !crateNamePattern.MatchString(parts[0]) || !crateVersionPattern.MatchString(parts[1]) {
		return domain.ArtifactReference{}, errors.New("crate reference is invalid")
	}
	return domain.NewArtifactReference(crateSource, parts[0]+"@"+parts[1])
}

type metadataDocument struct {
	Packages []metadataPackage `json:"packages"`
	Resolve  metadataResolve   `json:"resolve"`
}
type metadataPackage struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Version  string  `json:"version"`
	Source   *string `json:"source"`
	Checksum *string `json:"checksum"`
}
type metadataResolve struct {
	Nodes []metadataNode `json:"nodes"`
}
type metadataNode struct {
	ID   string        `json:"id"`
	Deps []metadataDep `json:"deps"`
}
type metadataDep struct {
	Pkg string `json:"pkg"`
}

// PackageRecord is a source-pinned exact crate package.
type PackageRecord struct {
	ID, Name, Version, Checksum string
}
type metadataEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ParseMetadata validates Cargo metadata and rejects path/git/alternate
// registry packages. The root package may be absent from the returned records.
func ParseMetadata(body []byte) ([]PackageRecord, []byte, error) {
	if len(body) == 0 || len(body) > maxMetadataBytes {
		return nil, nil, errors.New("cargo metadata exceeds bound")
	}
	var document metadataDocument
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&document); err != nil || len(document.Packages) == 0 {
		return nil, nil, errors.New("cargo metadata is invalid")
	}
	seen := map[string]bool{}
	records := make([]PackageRecord, 0, len(document.Packages))
	for _, packageValue := range document.Packages {
		if packageValue.ID == "" || !crateNamePattern.MatchString(packageValue.Name) || !crateVersionPattern.MatchString(packageValue.Version) {
			return nil, nil, errors.New("cargo metadata contains an unsupported or unverified package")
		}
		if packageValue.Source == nil && packageValue.Checksum == nil {
			continue // local root package; it is not an acquired registry artifact
		}
		if packageValue.Source == nil || *packageValue.Source != registrySourceURL || packageValue.Checksum == nil || !isSHA256(*packageValue.Checksum) {
			return nil, nil, errors.New("cargo metadata contains an unsupported or unverified package")
		}
		if seen[packageValue.ID] {
			return nil, nil, errors.New("cargo metadata contains duplicate package IDs")
		}
		seen[packageValue.ID] = true
		records = append(records, PackageRecord{ID: packageValue.ID, Name: packageValue.Name, Version: packageValue.Version, Checksum: *packageValue.Checksum})
	}
	edges := make([]metadataEdge, 0)
	for _, node := range document.Resolve.Nodes {
		if !seen[node.ID] {
			continue
		}
		for _, dependency := range node.Deps {
			if !seen[dependency.Pkg] {
				return nil, nil, errors.New("cargo graph references unknown package")
			}
			edges = append(edges, metadataEdge{From: node.ID, To: dependency.Pkg})
		}
	}
	edgeBytes, err := json.Marshal(edges)
	if err != nil {
		return nil, nil, errors.New("cargo graph normalization failed")
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	return records, edgeBytes, nil
}

// BuildLockedGraph creates the generic exact graph for one requested crate.
func BuildLockedGraph(reference domain.ArtifactReference, records []PackageRecord, edgeBytes []byte) (domain.LockedDependencyGraph, error) {
	if reference.Source() != crateSource {
		return domain.LockedDependencyGraph{}, errors.New("cargo graph source is invalid")
	}
	parts := strings.SplitN(reference.Locator(), "@", 2)
	if len(parts) != 2 {
		return domain.LockedDependencyGraph{}, errors.New("cargo graph reference is invalid")
	}
	byKey := map[string]PackageRecord{}
	for _, record := range records {
		byKey[record.Name+"@"+record.Version] = record
	}
	primaryKey := parts[0] + "@" + parts[1]
	primary, ok := byKey[primaryKey]
	if !ok {
		return domain.LockedDependencyGraph{}, errors.New("requested crate is absent from Cargo metadata")
	}
	_ = primary
	var edges []metadataEdge
	if err := json.Unmarshal(edgeBytes, &edges); err != nil {
		return domain.LockedDependencyGraph{}, errors.New("cargo graph edges are invalid")
	}
	byID := map[string]PackageRecord{}
	for _, record := range records {
		byID[record.ID] = record
	}
	selected := map[string]bool{primaryKey: true}
	for changed := true; changed; {
		changed = false
		for _, edge := range edges {
			from, fromOK := byID[edge.From]
			to, toOK := byID[edge.To]
			if !fromOK || !toOK {
				return domain.LockedDependencyGraph{}, errors.New("cargo graph edge references unknown package")
			}
			fromKey, toKey := from.Name+"@"+from.Version, to.Name+"@"+to.Version
			if selected[fromKey] && !selected[toKey] {
				selected[toKey] = true
				changed = true
			}
		}
	}
	nodeIDs := map[string]domain.DependencyNodeID{}
	nodes := make([]domain.LockedDependency, 0, len(selected))
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.SplitN(key, "@", 2)
		record, found := byKey[key]
		if !found || len(parts) != 2 {
			return domain.LockedDependencyGraph{}, errors.New("cargo selected package is invalid")
		}
		node, err := newNodeID(record.ID)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		nodeIDs[key] = node
		identity, err := domain.NewResolvedArtifactIdentity(crateSource, record.Name, record.Version, "crate")
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		locator, err := DownloadURL(record.Name, record.Version)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		artifact, err := domain.NewResolvedArtifact(identity, locator, "sha256="+record.Checksum)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		role := domain.DependencyTransitive
		if key == primaryKey {
			role = domain.DependencyPrimary
		}
		locked, err := domain.NewLockedDependency(node, role, artifact)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		nodes = append(nodes, locked)
	}
	domainEdges := make([]domain.DependencyEdge, 0, len(edges))
	for _, edge := range edges {
		from, fromOK := byID[edge.From]
		to, toOK := byID[edge.To]
		if !fromOK || !toOK {
			return domain.LockedDependencyGraph{}, errors.New("cargo graph edge is invalid")
		}
		fromKey, toKey := from.Name+"@"+from.Version, to.Name+"@"+to.Version
		if !selected[fromKey] || !selected[toKey] {
			continue
		}
		value, err := domain.NewDependencyEdge(nodeIDs[fromKey], nodeIDs[toKey])
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		domainEdges = append(domainEdges, value)
	}
	return domain.NewLockedDependencyGraph(nodes, domainEdges)
}

func DownloadURL(name, version string) (string, error) {
	if !crateNamePattern.MatchString(name) || !crateVersionPattern.MatchString(version) {
		return "", errors.New("crate download URL input is invalid")
	}
	return crateDownloadHost + "/" + name + "/" + name + "-" + version + ".crate", nil
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func newNodeID(value string) (domain.DependencyNodeID, error) {
	digest := sha256.Sum256([]byte(value))
	return domain.NewDependencyNodeID("c" + hex.EncodeToString(digest[:])[:24])
}
func mustSource(value string) domain.SourceID {
	source, err := domain.NewSourceID(value)
	if err != nil {
		panic(err)
	}
	return source
}
