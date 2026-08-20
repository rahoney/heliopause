package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	maxSimpleResponseBytes = 4 << 20
	maxReportBytes         = 4 << 20
	maxPyPIReportEntries   = 1024
)

var (
	sha256HexPattern      = regexp.MustCompile(`^[a-f0-9]{64}$`)
	simpleAPIVersion      = regexp.MustCompile(`^([0-9]+)\.([0-9]+)$`)
	requirementNamePrefix = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)(?:\s*(?:\([^)]*\)|[<>=!~].*)?)?$`)
)

// SimpleProject is a bounded, parser-normalized PyPI Simple API project page.
// It never contains a raw HTTP response or unvalidated endpoint.
type SimpleProject struct {
	project string
	files   []SimpleFile
}

func (p SimpleProject) Project() string { return p.project }
func (p SimpleProject) Files() []SimpleFile {
	return append([]SimpleFile(nil), p.files...)
}

// SimpleFile retains only the file metadata required to cross-check a pip
// candidate before Controlled Intake observes its content.
type SimpleFile struct {
	filename       string
	url            string
	sha256         string
	requiresPython string
	yanked         bool
	size           uint64
}

func (f SimpleFile) Filename() string       { return f.filename }
func (f SimpleFile) URL() string            { return f.url }
func (f SimpleFile) SHA256() string         { return f.sha256 }
func (f SimpleFile) RequiresPython() string { return f.requiresPython }
func (f SimpleFile) Yanked() bool           { return f.yanked }
func (f SimpleFile) Size() uint64           { return f.size }

// Candidate is one exact pip report selection after its project, version and
// declared dependency names have been normalized. It is adapter-only data.
type Candidate struct {
	project        string
	version        string
	filename       string
	url            string
	sha256         string
	requiresPython string
	primary        bool
	dependencies   []string
}

func (c Candidate) Project() string        { return c.project }
func (c Candidate) Version() string        { return c.version }
func (c Candidate) Filename() string       { return c.filename }
func (c Candidate) URL() string            { return c.url }
func (c Candidate) SHA256() string         { return c.sha256 }
func (c Candidate) RequiresPython() string { return c.requiresPython }
func (c Candidate) Primary() bool          { return c.primary }
func (c Candidate) Dependencies() []string { return append([]string(nil), c.dependencies...) }

// InstallationReport is the bounded, normalized result of a stable pip
// installation report v1. Its raw bytes remain in the Sandbox adapter.
type InstallationReport struct{ candidates []Candidate }

func (r InstallationReport) Candidates() []Candidate {
	return append([]Candidate(nil), r.candidates...)
}

type simpleResponse struct {
	Meta struct {
		APIVersion string `json:"api-version"`
	} `json:"meta"`
	Name  string       `json:"name"`
	Files []simpleFile `json:"files"`
}

type simpleFile struct {
	Filename       string            `json:"filename"`
	URL            string            `json:"url"`
	Hashes         map[string]string `json:"hashes"`
	RequiresPython string            `json:"requires-python"`
	Yanked         json.RawMessage   `json:"yanked"`
	Size           uint64            `json:"size"`
}

// ParseSimpleProject accepts only a PyPI Simple API JSON v1 project page and
// enforces the public PyPI distribution endpoint for every listed file.
func ParseSimpleProject(project string, body []byte) (SimpleProject, error) {
	project, err := NormalizeProjectName(project)
	if err != nil {
		return SimpleProject{}, errors.New("PyPI Simple project is invalid")
	}
	if len(body) == 0 || len(body) > maxSimpleResponseBytes {
		return SimpleProject{}, errors.New("PyPI Simple response exceeds supported bounds")
	}
	var response simpleResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil || ensureSingleJSONValue(decoder) != nil {
		return SimpleProject{}, errors.New("PyPI Simple response is invalid JSON")
	}
	version := simpleAPIVersion.FindStringSubmatch(response.Meta.APIVersion)
	if version == nil || version[1] != "1" {
		return SimpleProject{}, errors.New("PyPI Simple API version is unsupported")
	}
	responseProject, err := NormalizeProjectName(response.Name)
	if err != nil || responseProject != project || len(response.Files) == 0 || len(response.Files) > maxPyPIReportEntries {
		return SimpleProject{}, errors.New("PyPI Simple project metadata is invalid")
	}
	files := make([]SimpleFile, 0, len(response.Files))
	seen := make(map[string]bool, len(response.Files))
	for _, raw := range response.Files {
		file, err := parseSimpleFile(raw)
		if err != nil || seen[file.url] {
			return SimpleProject{}, errors.New("PyPI Simple file metadata is invalid")
		}
		seen[file.url] = true
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].url < files[j].url })
	return SimpleProject{project: project, files: files}, nil
}

func parseSimpleFile(raw simpleFile) (SimpleFile, error) {
	if raw.Filename == "" || raw.Size == 0 || !sha256HexPattern.MatchString(raw.Hashes["sha256"]) || !validRequiresPython(raw.RequiresPython) {
		return SimpleFile{}, errors.New("invalid file metadata")
	}
	parsed, err := parseDistributionURL(raw.URL, raw.Filename, true)
	if err != nil {
		return SimpleFile{}, err
	}
	if parsed.Fragment != "sha256="+raw.Hashes["sha256"] {
		return SimpleFile{}, errors.New("invalid distribution hash fragment")
	}
	yanked, err := parseYanked(raw.Yanked)
	if err != nil {
		return SimpleFile{}, err
	}
	return SimpleFile{filename: raw.Filename, url: raw.URL, sha256: raw.Hashes["sha256"], requiresPython: raw.RequiresPython, yanked: yanked, size: raw.Size}, nil
}

func parseYanked(raw json.RawMessage) (bool, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("false")) {
		return false, nil
	}
	if bytes.Equal(raw, []byte("true")) {
		return true, nil
	}
	var reason string
	if err := json.Unmarshal(raw, &reason); err != nil || reason == "" {
		return false, errors.New("invalid yanked value")
	}
	return true, nil
}

type pipReport struct {
	Version     string         `json:"version"`
	PipVersion  string         `json:"pip_version"`
	Install     []pipInstall   `json:"install"`
	Environment pipEnvironment `json:"environment"`
}

type pipEnvironment struct {
	ImplementationName    string `json:"implementation_name"`
	ImplementationVersion string `json:"implementation_version"`
	PythonFullVersion     string `json:"python_full_version"`
	PlatformMachine       string `json:"platform_machine"`
	SysPlatform           string `json:"sys_platform"`
}

type pipInstall struct {
	DownloadInfo struct {
		URL         string `json:"url"`
		ArchiveInfo struct {
			Hash   string            `json:"hash"`
			Hashes map[string]string `json:"hashes"`
		} `json:"archive_info"`
	} `json:"download_info"`
	IsDirect  bool `json:"is_direct"`
	Requested bool `json:"requested"`
	Metadata  struct {
		Name           string   `json:"name"`
		Version        string   `json:"version"`
		RequiresPython string   `json:"requires_python"`
		RequiresDist   []string `json:"requires_dist"`
	} `json:"metadata"`
}

// ParseInstallationReport parses the stable pip report v1 emitted from the
// locked runtime. It intentionally rejects direct sources, unsupported
// dependency markers/extras and any target-environment mismatch.
func ParseInstallationReport(reference domain.ArtifactReference, body []byte, expectedPip, expectedPython string) (InstallationReport, error) {
	requestedProject, err := RequestedProject(reference)
	if err != nil {
		return InstallationReport{}, errors.New("PyPI report reference is invalid")
	}
	requestedVersion, versionRequested, err := RequestedVersion(reference)
	if err != nil {
		return InstallationReport{}, errors.New("PyPI report reference is invalid")
	}
	if len(body) == 0 || len(body) > maxReportBytes {
		return InstallationReport{}, errors.New("pip installation report exceeds supported bounds")
	}
	var report pipReport
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&report); err != nil || ensureSingleJSONValue(decoder) != nil {
		return InstallationReport{}, errors.New("pip installation report is invalid JSON")
	}
	if report.Version != "1" || report.PipVersion != expectedPip || report.Environment.ImplementationName != "cpython" || report.Environment.ImplementationVersion != expectedPython || report.Environment.PythonFullVersion != expectedPython || report.Environment.PlatformMachine != "x86_64" || report.Environment.SysPlatform != "linux" || len(report.Install) == 0 || len(report.Install) > maxPyPIReportEntries {
		return InstallationReport{}, errors.New("pip installation report runtime is invalid")
	}
	candidates := make([]Candidate, 0, len(report.Install))
	seenProjects := make(map[string]bool, len(report.Install))
	primaryCount := 0
	for _, item := range report.Install {
		candidate, err := parseReportCandidate(item)
		if err != nil || seenProjects[candidate.project] {
			return InstallationReport{}, errors.New("pip installation report candidate is invalid")
		}
		seenProjects[candidate.project] = true
		if candidate.primary {
			primaryCount++
			if candidate.project != requestedProject || versionRequested && candidate.version != requestedVersion {
				return InstallationReport{}, errors.New("pip installation report primary candidate is invalid")
			}
			if !versionRequested && !IsFinalVersion(candidate.version) {
				return InstallationReport{}, errors.New("pip installation report selected a non-final release")
			}
		}
		candidates = append(candidates, candidate)
	}
	if primaryCount != 1 {
		return InstallationReport{}, errors.New("pip installation report requires exactly one primary candidate")
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].project < candidates[j].project })
	return InstallationReport{candidates: candidates}, nil
}

func parseReportCandidate(item pipInstall) (Candidate, error) {
	if item.IsDirect || !item.Requested && item.Metadata.Name == "" || !sha256HexPattern.MatchString(item.DownloadInfo.ArchiveInfo.Hashes["sha256"]) || item.DownloadInfo.ArchiveInfo.Hash != "sha256="+item.DownloadInfo.ArchiveInfo.Hashes["sha256"] {
		return Candidate{}, errors.New("invalid pip candidate")
	}
	project, err := NormalizeProjectName(item.Metadata.Name)
	if err != nil {
		return Candidate{}, err
	}
	version, err := NormalizeVersion(item.Metadata.Version)
	if err != nil || !validRequiresPython(item.Metadata.RequiresPython) {
		return Candidate{}, errors.New("invalid pip candidate metadata")
	}
	filename := path.Base(item.DownloadInfo.URL)
	if _, err := parseDistributionURL(item.DownloadInfo.URL, filename, false); err != nil {
		return Candidate{}, err
	}
	dependencies := make([]string, 0, len(item.Metadata.RequiresDist))
	seenDependencies := make(map[string]bool, len(item.Metadata.RequiresDist))
	for _, requirement := range item.Metadata.RequiresDist {
		dependency, err := parseDeclaredDependency(requirement)
		if err != nil || dependency == project || seenDependencies[dependency] {
			return Candidate{}, errors.New("unsupported pip dependency metadata")
		}
		seenDependencies[dependency] = true
		dependencies = append(dependencies, dependency)
	}
	sort.Strings(dependencies)
	return Candidate{project: project, version: version, filename: filename, url: item.DownloadInfo.URL, sha256: item.DownloadInfo.ArchiveInfo.Hashes["sha256"], requiresPython: item.Metadata.RequiresPython, primary: item.Requested, dependencies: dependencies}, nil
}

func parseDeclaredDependency(value string) (string, error) {
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, ";@[]") {
		return "", errors.New("unsupported dependency requirement")
	}
	matches := requirementNamePrefix.FindStringSubmatch(value)
	if matches == nil {
		return "", errors.New("unsupported dependency requirement")
	}
	return NormalizeProjectName(matches[1])
}

// CrossCheckReport matches every selected pip candidate to exactly one non-
// yanked Simple API file with the same endpoint, filename, SHA-256 and
// Requires-Python metadata.
func CrossCheckReport(report InstallationReport, pages []SimpleProject) ([]Candidate, error) {
	if len(report.candidates) == 0 || len(pages) != len(report.candidates) {
		return nil, errors.New("PyPI Simple cross-check is incomplete")
	}
	byProject := make(map[string]SimpleProject, len(pages))
	for _, page := range pages {
		if page.project == "" || byProject[page.project].project != "" {
			return nil, errors.New("PyPI Simple cross-check is ambiguous")
		}
		byProject[page.project] = page
	}
	for _, candidate := range report.candidates {
		page, ok := byProject[candidate.project]
		if !ok || !matchesSimpleFile(candidate, page.files) {
			return nil, errors.New("pip candidate does not match PyPI Simple metadata")
		}
	}
	return report.Candidates(), nil
}

func matchesSimpleFile(candidate Candidate, files []SimpleFile) bool {
	matches := 0
	for _, file := range files {
		if file.filename != candidate.filename || file.sha256 != candidate.sha256 || file.requiresPython != candidate.requiresPython || file.yanked || !sameDistributionURL(file.url, candidate.url) {
			continue
		}
		matches++
	}
	return matches == 1
}

// BuildLockedGraph converts cross-checked candidates into existing generic
// Domain values. Dependency edges come only from normalized selected report
// metadata; unknown or unreachable requirements remain fail-closed.
func BuildLockedGraph(reference domain.ArtifactReference, candidates []Candidate) (domain.LockedDependencyGraph, error) {
	if reference.Source().String() != "pypi" || len(candidates) == 0 || len(candidates) > maxPyPIReportEntries {
		return domain.LockedDependencyGraph{}, errors.New("PyPI candidate graph is invalid")
	}
	nodes := make(map[string]domain.LockedDependency, len(candidates))
	edgesByProject := make(map[string][]string, len(candidates))
	for _, candidate := range candidates {
		if candidate.project == "" || nodes[candidate.project].Node().String() != "" {
			return domain.LockedDependencyGraph{}, errors.New("PyPI candidate graph is ambiguous")
		}
		variant, ok := distributionVariant(candidate.filename)
		if !ok {
			return domain.LockedDependencyGraph{}, errors.New("PyPI distribution type is unsupported")
		}
		nodeID := candidateNodeID(candidate)
		identity, err := domain.NewResolvedArtifactIdentity(reference.Source(), candidate.project, candidate.version, variant)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		artifact, err := domain.NewResolvedArtifact(identity, candidate.url, "sha256:"+candidate.sha256)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		role := domain.DependencyTransitive
		if candidate.primary {
			role = domain.DependencyPrimary
		}
		node, err := domain.NewLockedDependencyWithRecordPath(nodeID, role, artifact, candidate.filename, false)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		nodes[candidate.project] = node
		edgesByProject[candidate.project] = candidate.dependencies
	}
	edges := make([]domain.DependencyEdge, 0)
	for project, dependencies := range edgesByProject {
		for _, dependency := range dependencies {
			to, ok := nodes[dependency]
			if !ok {
				return domain.LockedDependencyGraph{}, errors.New("pip dependency graph is incomplete")
			}
			edge, err := domain.NewDependencyEdge(nodes[project].Node(), to.Node())
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

func candidateNodeID(candidate Candidate) domain.DependencyNodeID {
	digest := sha256.Sum256([]byte(candidate.project + "\x00" + candidate.version + "\x00" + candidate.filename))
	id, _ := domain.NewDependencyNodeID("pypi-" + hex.EncodeToString(digest[:16]))
	return id
}

func distributionVariant(filename string) (string, bool) {
	switch {
	case strings.HasSuffix(filename, ".whl"):
		return "wheel", true
	case strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".zip"):
		return "sdist", true
	default:
		return "", false
	}
}

func parseDistributionURL(rawURL, filename string, allowHashFragment bool) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "files.pythonhosted.org") || parsed.Port() != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Path == "" || path.Base(parsed.Path) != filename || filename == "." || filename == "/" {
		return nil, errors.New("PyPI distribution URL is invalid")
	}
	if !allowHashFragment && parsed.Fragment != "" {
		return nil, errors.New("pip distribution URL must not have a fragment")
	}
	return parsed, nil
}

func sameDistributionURL(simpleURL, reportURL string) bool {
	simpleRaw, err := url.Parse(simpleURL)
	if err != nil {
		return false
	}
	simpleParsed, err := parseDistributionURL(simpleURL, path.Base(simpleRaw.Path), true)
	if err != nil {
		return false
	}
	reportRaw, err := url.Parse(reportURL)
	if err != nil {
		return false
	}
	reportParsed, err := parseDistributionURL(reportURL, path.Base(reportRaw.Path), false)
	if err != nil {
		return false
	}
	return simpleParsed.Scheme == reportParsed.Scheme && strings.EqualFold(simpleParsed.Host, reportParsed.Host) && simpleParsed.EscapedPath() == reportParsed.EscapedPath()
}

func validRequiresPython(value string) bool {
	return len(value) <= 1024 && value == strings.TrimSpace(value) && strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) < 0
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}
