// Package gomodule owns the bounded, source-pinned normalization boundary for
// public Go Modules. Go command output is treated as untrusted adapter input;
// no VCS or ambient proxy configuration is accepted here.
package gomodule

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/module"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	proxyEndpoint     = "https://proxy.golang.org"
	sumDBEndpoint     = "sum.golang.org"
	maxDownloadOutput = 4 << 20
)

var goModuleSource = mustSource("go-proxy")

// Source is the one supported public Go module source identity.
func Source() domain.SourceID { return goModuleSource }

// Endpoints are fixed by the M12 source contract and are not user input.
func Endpoints() (string, string) { return proxyEndpoint, sumDBEndpoint }

// ResolverEnvironment is the complete Go resolver policy. Callers must pass
// this environment explicitly to an isolated `go` process; inheriting the
// caller's GOPROXY/GOPRIVATE/GOVCS is forbidden.
func ResolverEnvironment() []string {
	return []string{
		"GOPROXY=" + proxyEndpoint,
		"GOSUMDB=" + sumDBEndpoint,
		"GOPRIVATE=",
		"GONOPROXY=",
		"GONOSUMDB=",
		"GOVCS=*:off",
		"GOTOOLCHAIN=local",
		"GOENV=off",
		"GOWORK=off",
		// Writable directories are required to dispose of this operation-private
		// cache. This never enables a caller's tool-exec or acquisition options.
		"GOFLAGS=-modcacherw",
	}
}

// ValidateResolverEnvironment rejects an ambient environment before any Go
// command executes. Only the exact canonical values above are accepted.
func ValidateResolverEnvironment(environment []string) error {
	allowed := make(map[string]string, len(ResolverEnvironment()))
	for _, entry := range ResolverEnvironment() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			allowed[key] = value
		}
	}
	seen := map[string]bool{}
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		expected, known := allowed[key]
		if !ok || !known || expected != value || seen[key] {
			return errors.New("go module resolver environment is not canonical")
		}
		seen[key] = true
	}
	for key := range allowed {
		if !seen[key] {
			return errors.New("go module resolver environment is incomplete")
		}
	}
	return nil
}

// ResolverEnvironmentForCache extends the fixed source policy with an
// operation-private module cache. The cache path is infrastructure-selected,
// never inherited from the caller's GOPATH or GOMODCACHE.
func ResolverEnvironmentForCache(cache string) ([]string, error) {
	if !filepath.IsAbs(cache) || filepath.Clean(cache) != cache || cache == "/" {
		return nil, errors.New("go module cache path is invalid")
	}
	environment := append([]string(nil), ResolverEnvironment()...)
	return append(environment, "GOMODCACHE="+cache), nil
}

// ValidateResolverEnvironmentForCache rejects a substituted or missing cache
// while retaining the canonical resolver source policy.
func ValidateResolverEnvironmentForCache(environment []string, cache string) error {
	expected, err := ResolverEnvironmentForCache(cache)
	if err != nil || len(environment) != len(expected) {
		return errors.New("go module resolver environment is not canonical")
	}
	for index := range expected {
		if environment[index] != expected[index] {
			return errors.New("go module resolver environment is not canonical")
		}
	}
	return nil
}

// BuildEnvironmentForCache is intentionally separate from resolver policy:
// builds cannot acquire dependencies and must fail if the verified cache is
// incomplete. The cache path remains infrastructure-selected.
func BuildEnvironmentForCache(cache string) ([]string, error) {
	if !filepath.IsAbs(cache) || filepath.Clean(cache) != cache || cache == "/" {
		return nil, errors.New("go module cache path is invalid")
	}
	return []string{
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOPRIVATE=",
		"GONOPROXY=",
		"GONOSUMDB=*",
		"GOVCS=*:off",
		"GOTOOLCHAIN=local",
		"GOENV=off",
		"GOWORK=off",
		"GOFLAGS=-mod=readonly",
		"GOMODCACHE=" + cache,
	}, nil
}

// ValidateBuildEnvironmentForCache rejects resolver settings, ambient source
// policy, or a substituted cache from the network-disabled build boundary.
func ValidateBuildEnvironmentForCache(environment []string, cache string) error {
	expected, err := BuildEnvironmentForCache(cache)
	if err != nil || len(environment) != len(expected) {
		return errors.New("go module build environment is not canonical")
	}
	for index := range expected {
		if environment[index] != expected[index] {
			return errors.New("go module build environment is not canonical")
		}
	}
	return nil
}

// Reference is an exact module path and semantic version request.
func ParseReference(value string) (domain.ArtifactReference, error) {
	if strings.Count(value, "@") != 1 {
		return domain.ArtifactReference{}, errors.New("go module reference requires module@version")
	}
	parts := strings.SplitN(value, "@", 2)
	if !validModuleVersion(parts[0], parts[1]) {
		return domain.ArtifactReference{}, errors.New("go module reference is invalid")
	}
	return domain.NewArtifactReference(goModuleSource, parts[0]+"@"+parts[1])
}

func validModuleVersion(pathValue, version string) bool {
	return module.Check(pathValue, version) == nil && module.CanonicalVersion(version) == version
}

// DownloadRecord is the bounded subset of `go mod download -json` needed for
// exact identity and checksum declarations. Origin is repository metadata, not
// proof of the acquisition endpoint; the pending isolated source-provenance
// boundary has not yet attested records carrying it.
type DownloadRecord struct {
	Path     string          `json:"Path"`
	Version  string          `json:"Version"`
	Info     string          `json:"Info"`
	GoMod    string          `json:"GoMod"`
	Zip      string          `json:"Zip"`
	Sum      string          `json:"Sum"`
	GoModSum string          `json:"GoModSum"`
	Origin   json.RawMessage `json:"Origin"`
}

// ParseDownloadJSON parses the bounded stream of Go command JSON objects. A record
// with unattested origin metadata or missing checksums is not admitted here.
func ParseDownloadJSON(body []byte) ([]DownloadRecord, error) {
	if len(body) == 0 || len(body) > maxDownloadOutput {
		return nil, errors.New("go module download output exceeds bound")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var records []DownloadRecord
	seen := map[string]bool{}
	for {
		var record DownloadRecord
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil || record.Path == "" || record.Version == "" || record.Zip == "" || record.GoMod == "" || record.Sum == "" || record.GoModSum == "" {
			return nil, errors.New("go module download record is incomplete")
		}
		if !validModuleVersion(record.Path, record.Version) || strings.Contains(record.Zip, "\\") || strings.Contains(record.GoMod, "\\") {
			return nil, errors.New("go module download record identity is invalid")
		}
		if len(record.Origin) != 0 && string(record.Origin) != "null" && string(record.Origin) != "{}" {
			return nil, errors.New("go module output source provenance is not attested")
		}
		if _, err := h1Digest(record.Sum); err != nil {
			return nil, errors.New("go module SumDB checksum is invalid")
		}
		if _, err := h1Digest(record.GoModSum); err != nil {
			return nil, errors.New("go module go.mod SumDB checksum is invalid")
		}
		key := recordKey(record.Path, record.Version)
		if seen[key] {
			return nil, errors.New("go module download output contains duplicate module")
		}
		seen[key] = true
		records = append(records, record)
	}
	if len(records) == 0 {
		return nil, errors.New("go module download output is invalid")
	}
	sort.Slice(records, func(i, j int) bool {
		return recordKey(records[i].Path, records[i].Version) < recordKey(records[j].Path, records[j].Version)
	})
	return records, nil
}

// BuildLockedGraph converts exact download records and `go mod graph` edges
// into the generic Domain graph. Edges outside the primary closure are rejected
// rather than silently dropping resolver output.
func BuildLockedGraph(reference domain.ArtifactReference, records []DownloadRecord, graphOutput, goMod []byte) (domain.LockedDependencyGraph, error) {
	if parsed, err := ParseReference(reference.Locator()); err != nil || parsed != reference || len(records) == 0 {
		return domain.LockedDependencyGraph{}, errors.New("go module graph request is invalid")
	}
	byKey := make(map[string]DownloadRecord, len(records))
	for _, record := range records {
		byKey[recordKey(record.Path, record.Version)] = record
	}
	requestedPath, requestedVersion := strings.SplitN(reference.Locator(), "@", 2)[0], strings.SplitN(reference.Locator(), "@", 2)[1]
	primaryKey := recordKey(requestedPath, requestedVersion)
	if _, ok := byKey[primaryKey]; !ok {
		return domain.LockedDependencyGraph{}, errors.New("requested Go module is absent from exact graph")
	}
	edges, err := normalizeProjectGraph(graphOutput, records, goMod)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	selected := map[string]bool{primaryKey: true}
	for changed := true; changed; {
		changed = false
		for _, edge := range edges {
			if selected[edge.from] && !selected[edge.to] {
				selected[edge.to] = true
				changed = true
			}
		}
	}
	nodes := make([]domain.LockedDependency, 0, len(selected))
	nodeIDs := map[string]domain.DependencyNodeID{}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		record := byKey[key]
		node, err := newNodeID(key)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		nodeIDs[key] = node
		_, err = h1Digest(record.Sum)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		identity, err := domain.NewResolvedArtifactIdentity(goModuleSource, record.Path, record.Version, "module")
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		locator, err := ProxyURL(record.Path, record.Version, ".zip")
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		artifact, err := domain.NewResolvedArtifact(identity, locator, "h1="+record.Sum+";go.mod="+record.GoModSum)
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
		if !selected[edge.from] || !selected[edge.to] {
			continue
		}
		from, fromOK := nodeIDs[edge.from]
		to, toOK := nodeIDs[edge.to]
		if !fromOK || !toOK || from == to {
			continue
		}
		value, err := domain.NewDependencyEdge(from, to)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		domainEdges = append(domainEdges, value)
	}
	return domain.NewLockedDependencyGraph(nodes, domainEdges)
}

// BuildProjectSnapshot freezes every exact public module selected by a Go
// project. The local main module is not an acquired artifact, so this does
// not weaken LockedDependencyGraph's exactly-one-primary invariant.
func BuildProjectSnapshot(installContext domain.InstallContext, records []DownloadRecord, graphOutput, goMod, goSum []byte) (domain.ProjectDependencySnapshot, error) {
	if !installContext.Valid() || len(records) == 0 || len(goMod) == 0 || len(goSum) == 0 {
		return domain.ProjectDependencySnapshot{}, errors.New("go project snapshot request is invalid")
	}
	if err := ValidateProjectSums(goSum, records); err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	byKey := make(map[string]DownloadRecord, len(records))
	for _, record := range records {
		key := recordKey(record.Path, record.Version)
		if _, exists := byKey[key]; exists {
			return domain.ProjectDependencySnapshot{}, errors.New("go project snapshot contains duplicate module")
		}
		byKey[key] = record
	}
	if _, err := normalizeProjectGraph(graphOutput, records, goMod); err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	dependencies := make([]domain.ResolvedArtifact, 0, len(records))
	for _, record := range records {
		identity, err := domain.NewResolvedArtifactIdentity(goModuleSource, record.Path, record.Version, "module")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		locator, err := ProxyURL(record.Path, record.Version, ".zip")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		artifact, err := domain.NewResolvedArtifact(identity, locator, "h1="+record.Sum+";go.mod="+record.GoModSum)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		dependencies = append(dependencies, artifact)
	}
	modHash, sumHash := sha256.Sum256(goMod), sha256.Sum256(goSum)
	modDigest, err := domain.NewSHA256Digest(hex.EncodeToString(modHash[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	sumDigest, err := domain.NewSHA256Digest(hex.EncodeToString(sumHash[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	graphDigest, err := FreezeResolutionDigest(records, graphOutput, goMod, goSum)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	modControl, err := domain.NewProjectControlDigest("go.mod", modDigest)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	sumControl, err := domain.NewProjectControlDigest("go.sum", sumDigest)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	return domain.NewProjectDependencySnapshot(installContext, goModuleSource, []domain.ProjectControlDigest{modControl, sumControl}, dependencies, graphDigest)
}

func recordKey(pathValue, version string) string { return pathValue + "@" + version }

func h1Digest(value string) (domain.ContentDigest, error) {
	if !strings.HasPrefix(value, "h1:") {
		return domain.ContentDigest{}, errors.New("go module checksum must use h1")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "h1:"))
	if err != nil || len(decoded) != 32 {
		return domain.ContentDigest{}, errors.New("go module checksum is not a SHA-256 h1 value")
	}
	return domain.NewSHA256Digest(hex.EncodeToString(decoded))
}

func newNodeID(key string) (domain.DependencyNodeID, error) {
	// Module paths contain '/', so node identity is an opaque stable digest.
	digest := sha256.Sum256([]byte(key))
	return domain.NewDependencyNodeID("m" + hex.EncodeToString(digest[:])[:24])
}

func mustSource(value string) domain.SourceID {
	source, err := domain.NewSourceID(value)
	if err != nil {
		panic(err)
	}
	return source
}

// ProxyURL returns the only canonical acquisition URL accepted for a module.
func ProxyURL(modulePath, version, suffix string) (string, error) {
	if !validModuleVersion(modulePath, version) || (suffix != ".zip" && suffix != ".mod" && suffix != ".info") {
		return "", errors.New("go module proxy URL input is invalid")
	}
	escapedPath, pathErr := module.EscapePath(modulePath)
	escapedVersion, versionErr := module.EscapeVersion(version)
	if pathErr != nil || versionErr != nil {
		return "", errors.New("go module proxy URL escaping failed")
	}
	return proxyEndpoint + "/" + escapedPath + "/@v/" + escapedVersion + suffix, nil
}
