package cargo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
)

const (
	MaxProjectControlBytes = 4 << 20
	maxProjectPackages     = 4096
	maxProjectEdges        = 16384
)

type lockDocument struct {
	Version  int           `toml:"version"`
	Packages []lockPackage `toml:"package"`
}

type lockPackage struct {
	Name         string   `toml:"name"`
	Version      string   `toml:"version"`
	Source       string   `toml:"source"`
	Checksum     string   `toml:"checksum"`
	Dependencies []string `toml:"dependencies"`
	resolvedDeps map[string]bool
}

type projectMetadata struct {
	Version          int              `json:"version"`
	WorkspaceRoot    string           `json:"workspace_root"`
	WorkspaceMembers []string         `json:"workspace_members"`
	Packages         []projectPackage `json:"packages"`
	Resolve          struct {
		Root  *string       `json:"root"`
		Nodes []projectNode `json:"nodes"`
	} `json:"resolve"`
}

type projectPackage struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Version      string  `json:"version"`
	Source       *string `json:"source"`
	Checksum     *string `json:"checksum"`
	ManifestPath string  `json:"manifest_path"`
}

type projectNode struct {
	ID       string       `json:"id"`
	Features []string     `json:"features"`
	Deps     []projectDep `json:"deps"`
}

type projectDep struct {
	Name  string `json:"name"`
	Pkg   string `json:"pkg"`
	Kinds []struct {
		Kind   *string `json:"kind"`
		Target *string `json:"target"`
	} `json:"dep_kinds"`
}

type frozenNode struct {
	Identity string   `json:"identity"`
	Features []string `json:"features"`
}

type frozenEdge struct {
	From, To, Name, Kind, Target string
}

// ParseLockedMetadata joins real Cargo metadata to frozen Cargo.lock checksums.
// These are declarations, not registry authentication: the acquisition verifier
// must independently compare official sparse-index checksums and acquired bytes.
// state is canonical graph/feature/workspace data without private absolute paths.
func ParseLockedMetadata(body, lockBody []byte, projectRoot string) (records []PackageRecord, edges, state []byte, resultErr error) {
	if !filepath.IsAbs(projectRoot) || filepath.Clean(projectRoot) != projectRoot || projectRoot == "/" {
		return nil, nil, nil, errors.New("cargo metadata project root is invalid")
	}
	locked, err := parseLock(lockBody)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := validateMetadataJSON(body); err != nil {
		return nil, nil, nil, err
	}
	var document projectMetadata
	if err := json.Unmarshal(body, &document); err != nil || document.Version != 1 || document.WorkspaceRoot != projectRoot || len(document.Packages) == 0 || len(document.Packages) > maxProjectPackages || len(document.Resolve.Nodes) != len(document.Packages) || len(document.WorkspaceMembers) == 0 || len(document.WorkspaceMembers) > maxProjectPackages {
		return nil, nil, nil, errors.New("cargo project metadata is invalid or incomplete")
	}
	byID, identities, local := map[string]projectPackage{}, map[string]string{}, map[string]bool{}
	lockKeys := map[string]string{}
	seenIdentity := map[string]bool{}
	for _, value := range document.Packages {
		if !boundedText(value.ID, 4096) || !crateNamePattern.MatchString(value.Name) || !crateVersionPattern.MatchString(value.Version) {
			return nil, nil, nil, errors.New("cargo project package identity is invalid")
		}
		if _, exists := byID[value.ID]; exists {
			return nil, nil, nil, errors.New("cargo project metadata contains duplicate package IDs")
		}
		key := value.Name + "@" + value.Version
		if value.Source == nil {
			relative, err := filepath.Rel(projectRoot, value.ManifestPath)
			if err != nil || !filepath.IsAbs(value.ManifestPath) || filepath.Clean(value.ManifestPath) != value.ManifestPath || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.Base(relative) != "Cargo.toml" || value.Checksum != nil {
				return nil, nil, nil, errors.New("cargo local package escapes the project boundary")
			}
			if _, exists := locked["local:"+key]; !exists {
				return nil, nil, nil, errors.New("cargo local package is absent from frozen lock")
			}
			identities[value.ID] = "project:" + key + ":" + filepath.ToSlash(relative)
			lockKeys[value.ID] = "local:" + key
			local[value.ID] = true
		} else {
			if *value.Source != registrySourceURL {
				return nil, nil, nil, errors.New("cargo package source is not public crates.io")
			}
			entry, exists := locked["registry:"+key]
			if !exists || (value.Checksum != nil && *value.Checksum != entry.Checksum) {
				return nil, nil, nil, errors.New("cargo metadata differs from frozen lock checksum")
			}
			records = append(records, PackageRecord{ID: value.ID, Name: value.Name, Version: value.Version, Checksum: entry.Checksum})
			identities[value.ID] = "crates-io:" + key
			lockKeys[value.ID] = "registry:" + key
		}
		if seenIdentity[identities[value.ID]] {
			return nil, nil, nil, errors.New("cargo project metadata contains duplicate identities")
		}
		seenIdentity[identities[value.ID]] = true
		byID[value.ID] = value
	}
	members := make([]string, 0, len(document.WorkspaceMembers))
	seenMembers := map[string]bool{}
	for _, id := range document.WorkspaceMembers {
		if !local[id] || seenMembers[id] {
			return nil, nil, nil, errors.New("cargo workspace membership is invalid")
		}
		seenMembers[id] = true
		members = append(members, identities[id])
	}
	if document.Resolve.Root != nil && (!local[*document.Resolve.Root] || !seenMembers[*document.Resolve.Root]) {
		return nil, nil, nil, errors.New("cargo resolved root is not an owned workspace member")
	}
	var registryEdges []metadataEdge
	seenRegistryEdges := map[metadataEdge]bool{}
	var nodes []frozenNode
	var graphEdges []frozenEdge
	adjacent := map[string][]string{}
	seenNodes, seenEdges := map[string]bool{}, map[frozenEdge]bool{}
	for _, node := range document.Resolve.Nodes {
		if _, exists := byID[node.ID]; !exists || seenNodes[node.ID] || len(node.Features) > 1024 {
			return nil, nil, nil, errors.New("cargo project graph node is invalid")
		}
		seenNodes[node.ID] = true
		features := append([]string(nil), node.Features...)
		sort.Strings(features)
		for index, feature := range features {
			if !boundedText(feature, 512) || (index > 0 && features[index-1] == feature) {
				return nil, nil, nil, errors.New("cargo graph features are invalid")
			}
		}
		nodes = append(nodes, frozenNode{identities[node.ID], features})
		if len(node.Deps) > maxProjectEdges {
			return nil, nil, nil, errors.New("cargo project dependency count exceeds bound")
		}
		for _, dependency := range node.Deps {
			if _, exists := byID[dependency.Pkg]; !exists || !boundedText(dependency.Name, 128) || len(dependency.Kinds) == 0 || len(dependency.Kinds) > 64 {
				return nil, nil, nil, errors.New("cargo graph dependency is invalid")
			}
			if !locked[lockKeys[node.ID]].resolvedDeps[lockKeys[dependency.Pkg]] {
				return nil, nil, nil, errors.New("cargo graph edge is absent from frozen lock")
			}
			adjacent[node.ID] = append(adjacent[node.ID], dependency.Pkg)
			for _, kind := range dependency.Kinds {
				value := frozenEdge{From: identities[node.ID], To: identities[dependency.Pkg], Name: dependency.Name}
				if kind.Kind != nil {
					if *kind.Kind != "build" && *kind.Kind != "dev" {
						return nil, nil, nil, errors.New("cargo dependency kind is unsupported")
					}
					value.Kind = *kind.Kind
				}
				if kind.Target != nil {
					if !boundedText(*kind.Target, 4096) {
						return nil, nil, nil, errors.New("cargo dependency target exceeds bound")
					}
					value.Target = *kind.Target
				}
				if seenEdges[value] || len(graphEdges) >= maxProjectEdges {
					return nil, nil, nil, errors.New("cargo graph edges are duplicate or exceed bound")
				}
				seenEdges[value] = true
				graphEdges = append(graphEdges, value)
			}
			if !local[node.ID] && !local[dependency.Pkg] {
				edge := metadataEdge{From: node.ID, To: dependency.Pkg}
				if !seenRegistryEdges[edge] {
					seenRegistryEdges[edge] = true
					registryEdges = append(registryEdges, edge)
				}
			}
		}
	}
	reachable := map[string]bool{}
	queue := append([]string(nil), document.WorkspaceMembers...)
	for _, id := range queue {
		reachable[id] = true
	}
	for index := 0; index < len(queue); index++ {
		for _, id := range adjacent[queue[index]] {
			if !reachable[id] {
				reachable[id] = true
				queue = append(queue, id)
			}
		}
	}
	if len(reachable) != len(byID) {
		return nil, nil, nil, errors.New("cargo graph contains packages disconnected from workspace members")
	}
	sort.Strings(members)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Identity < nodes[j].Identity })
	sort.Slice(graphEdges, func(i, j int) bool {
		left, _ := json.Marshal(graphEdges[i])
		right, _ := json.Marshal(graphEdges[j])
		return bytes.Compare(left, right) < 0
	})
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	sort.Slice(registryEdges, func(i, j int) bool {
		if registryEdges[i].From != registryEdges[j].From {
			return registryEdges[i].From < registryEdges[j].From
		}
		return registryEdges[i].To < registryEdges[j].To
	})
	edges, err = json.Marshal(registryEdges)
	if err != nil {
		return nil, nil, nil, errors.New("cargo registry graph normalization failed")
	}
	root := ""
	if document.Resolve.Root != nil {
		root = identities[*document.Resolve.Root]
	}
	state, err = json.Marshal(struct {
		Schema  int          `json:"schema"`
		Root    string       `json:"root"`
		Members []string     `json:"members"`
		Nodes   []frozenNode `json:"nodes"`
		Edges   []frozenEdge `json:"edges"`
	}{1, root, members, nodes, graphEdges})
	if err != nil {
		return nil, nil, nil, errors.New("cargo project graph normalization failed")
	}
	return records, edges, state, nil
}

func parseLock(body []byte) (map[string]lockPackage, error) {
	if len(body) == 0 || len(body) > MaxProjectControlBytes || !utf8.Valid(body) {
		return nil, errors.New("cargo lock exceeds bounds")
	}
	var document lockDocument
	decoder := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil || (document.Version != 3 && document.Version != 4) || len(document.Packages) == 0 || len(document.Packages) > maxProjectPackages {
		return nil, errors.New("cargo lock grammar or version is unsupported")
	}
	values := map[string]lockPackage{}
	byName := map[string][]string{}
	for _, entry := range document.Packages {
		if !crateNamePattern.MatchString(entry.Name) || !crateVersionPattern.MatchString(entry.Version) || len(entry.Dependencies) > maxProjectEdges {
			return nil, errors.New("cargo lock package identity is invalid")
		}
		key := "local:" + entry.Name + "@" + entry.Version
		if entry.Source != "" {
			if entry.Source != registrySourceURL || !isSHA256(entry.Checksum) || entry.Checksum != strings.ToLower(entry.Checksum) {
				return nil, errors.New("cargo lock source or checksum is invalid")
			}
			key = "registry:" + entry.Name + "@" + entry.Version
		} else if entry.Checksum != "" {
			return nil, errors.New("cargo local lock package has a registry checksum")
		}
		if _, exists := values[key]; exists {
			return nil, errors.New("cargo lock contains duplicate package identity")
		}
		for _, dependency := range entry.Dependencies {
			if !boundedText(dependency, 4096) {
				return nil, errors.New("cargo lock dependency is invalid")
			}
		}
		values[key] = entry
		byName[entry.Name] = append(byName[entry.Name], key)
	}
	edges := 0
	for key, entry := range values {
		entry.resolvedDeps = map[string]bool{}
		for _, dependency := range entry.Dependencies {
			fields := strings.Fields(dependency)
			if len(fields) == 0 || len(fields) > 3 || !crateNamePattern.MatchString(fields[0]) || (len(fields) >= 2 && !crateVersionPattern.MatchString(fields[1])) || (len(fields) == 3 && fields[2] != "("+registrySourceURL+")") {
				return nil, errors.New("cargo lock dependency reference is unsupported")
			}
			matches := []string{}
			for _, candidateKey := range byName[fields[0]] {
				candidate := values[candidateKey]
				if (len(fields) < 2 || candidate.Version == fields[1]) && (len(fields) < 3 || candidate.Source == registrySourceURL) {
					matches = append(matches, candidateKey)
				}
			}
			if len(matches) != 1 || entry.resolvedDeps[matches[0]] || edges >= maxProjectEdges {
				return nil, errors.New("cargo lock dependency is unknown, ambiguous, duplicate or exceeds bound")
			}
			entry.resolvedDeps[matches[0]] = true
			edges++
		}
		values[key] = entry
	}
	return values, nil
}

func boundedText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}

func validateMetadataJSON(body []byte) error {
	if len(body) == 0 || len(body) > maxMetadataBytes || !utf8.Valid(body) {
		return errors.New("cargo metadata exceeds bounds")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := metadataJSONValue(decoder, 0); err != nil {
		return errors.New("cargo metadata JSON is invalid or ambiguous")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("cargo metadata contains trailing JSON")
	}
	return nil
}

func metadataJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("metadata JSON depth exceeds bound")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, aggregate := token.(json.Delim)
	if !aggregate {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return errors.New("metadata JSON delimiter is invalid")
	}
	keys := map[string]bool{}
	for decoder.More() {
		if delimiter == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || keys[name] {
				return errors.New("metadata JSON key is duplicate")
			}
			keys[name] = true
		}
		if err := metadataJSONValue(decoder, depth+1); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil || (delimiter == '{' && closing != json.Delim('}')) || (delimiter == '[' && closing != json.Delim(']')) {
		return errors.New("metadata JSON closing delimiter is invalid")
	}
	return nil
}
