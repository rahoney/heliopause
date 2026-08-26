// Package gomodule owns the bounded, source-pinned normalization boundary for
// public Go Modules. Go command output is treated as untrusted adapter input;
// no VCS or ambient proxy configuration is accepted here.
package gomodule

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	proxyEndpoint     = "https://proxy.golang.org"
	sumDBEndpoint     = "sum.golang.org"
	maxDownloadOutput = 4 << 20
)

var (
	modulePathPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~/-]*$`)
	moduleVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	goModuleSource       = mustSource("go-proxy")
)

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
		if !ok || allowed[key] != value || seen[key] {
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

// Reference is an exact module path and semantic version request.
func ParseReference(value string) (domain.ArtifactReference, error) {
	if strings.Count(value, "@") != 1 {
		return domain.ArtifactReference{}, errors.New("go module reference requires module@version")
	}
	parts := strings.SplitN(value, "@", 2)
	if !validModulePath(parts[0]) || !moduleVersionPattern.MatchString(parts[1]) {
		return domain.ArtifactReference{}, errors.New("go module reference is invalid")
	}
	return domain.NewArtifactReference(goModuleSource, parts[0]+"@"+parts[1])
}

func validModulePath(value string) bool {
	return value != "" && !strings.Contains(value, "..") && !strings.ContainsAny(value, ":?#\\\\") && modulePathPattern.MatchString(value)
}

// DownloadRecord is the bounded subset of `go mod download -json` needed for
// exact identity and SumDB-backed integrity. Origin is deliberately rejected.
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

// ParseDownloadJSON parses newline-delimited Go command records. A record
// with VCS Origin or missing SumDB checksums is never promoted.
func ParseDownloadJSON(body []byte) ([]DownloadRecord, error) {
	if len(body) == 0 || len(body) > maxDownloadOutput {
		return nil, errors.New("go module download output exceeds bound")
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 1024), maxDownloadOutput)
	var records []DownloadRecord
	seen := map[string]bool{}
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var record DownloadRecord
		decoder := json.NewDecoder(bytes.NewReader(line))
		if err := decoder.Decode(&record); err != nil || record.Path == "" || record.Version == "" || record.Zip == "" || record.GoMod == "" || record.Sum == "" || record.GoModSum == "" {
			return nil, errors.New("go module download record is incomplete")
		}
		if !validModulePath(record.Path) || !moduleVersionPattern.MatchString(record.Version) || strings.Contains(record.Zip, "\\") || strings.Contains(record.GoMod, "\\") {
			return nil, errors.New("go module download record identity is invalid")
		}
		if len(record.Origin) != 0 && string(record.Origin) != "null" && string(record.Origin) != "{}" {
			return nil, errors.New("direct VCS module fallback is forbidden")
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
	if err := scanner.Err(); err != nil || len(records) == 0 {
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
func BuildLockedGraph(reference domain.ArtifactReference, records []DownloadRecord, graphOutput []byte) (domain.LockedDependencyGraph, error) {
	if reference.Source() != goModuleSource || len(records) == 0 {
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
	edges, err := parseGraphEdges(graphOutput, byKey)
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
	byKey := make(map[string]DownloadRecord, len(records))
	for _, record := range records {
		key := recordKey(record.Path, record.Version)
		if _, exists := byKey[key]; exists {
			return domain.ProjectDependencySnapshot{}, errors.New("go project snapshot contains duplicate module")
		}
		byKey[key] = record
	}
	if err := validateCompleteProjectGraph(graphOutput, byKey); err != nil {
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
	modHash, sumHash, graphHash := sha256.Sum256(goMod), sha256.Sum256(goSum), sha256.Sum256(graphOutput)
	modDigest, err := domain.NewSHA256Digest(hex.EncodeToString(modHash[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	sumDigest, err := domain.NewSHA256Digest(hex.EncodeToString(sumHash[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	graphDigest, err := domain.NewSHA256Digest(hex.EncodeToString(graphHash[:]))
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

func validateCompleteProjectGraph(body []byte, records map[string]DownloadRecord) error {
	if len(body) == 0 || len(body) > maxDownloadOutput {
		return errors.New("go project graph is invalid")
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	seen := map[string]bool{}
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			return errors.New("go module graph edge is invalid")
		}
		from, err := parseModuleKey(parts[0])
		if err != nil {
			return err
		}
		to, err := parseModuleKey(parts[1])
		if err != nil {
			return err
		}
		if _, known := records[from]; known {
			seen[from] = true
		}
		if _, known := records[to]; known {
			seen[to] = true
		}
		if _, fromKnown := records[from]; fromKnown {
			if _, toKnown := records[to]; !toKnown {
				return errors.New("go module graph references unknown target")
			}
		}
	}
	if err := scanner.Err(); err != nil || len(seen) != len(records) {
		return errors.New("go project graph is incomplete")
	}
	return nil
}

type graphEdge struct{ from, to string }

func parseGraphEdges(body []byte, records map[string]DownloadRecord) ([]graphEdge, error) {
	if len(body) > maxDownloadOutput {
		return nil, errors.New("go module graph exceeds bound")
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	edges := []graphEdge{}
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			return nil, errors.New("go module graph edge is invalid")
		}
		from, err := parseModuleKey(parts[0])
		if err != nil {
			return nil, err
		}
		to, err := parseModuleKey(parts[1])
		if err != nil {
			return nil, err
		}
		if _, ok := records[from]; !ok {
			// `go mod graph` includes the local main module, which is not an
			// acquired public module record. Its outgoing edge is outside this
			// source graph; dependency records remain subject to exact checks.
			continue
		}
		if _, ok := records[to]; !ok {
			return nil, errors.New("go module graph references unknown target")
		}
		edges = append(edges, graphEdge{from: from, to: to})
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.New("go module graph cannot be read")
	}
	return edges, nil
}

func parseModuleKey(value string) (string, error) {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 || !validModulePath(parts[0]) || !moduleVersionPattern.MatchString(parts[1]) {
		return "", errors.New("go module graph module identity is invalid")
	}
	return recordKey(parts[0], parts[1]), nil
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
	if !validModulePath(modulePath) || !moduleVersionPattern.MatchString(version) || (suffix != ".zip" && suffix != ".mod" && suffix != ".info") {
		return "", errors.New("go module proxy URL input is invalid")
	}
	encoded := []string{}
	for _, segment := range strings.Split(modulePath, "/") {
		if segment == "" {
			return "", errors.New("go module path contains an empty segment")
		}
		encoded = append(encoded, url.PathEscape(strings.ToLower(segment)))
	}
	return proxyEndpoint + "/" + strings.Join(encoded, "/") + "/@v/" + version + suffix, nil
}
