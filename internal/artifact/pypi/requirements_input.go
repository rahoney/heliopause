package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const RequirementsInputControlName = "python-requirements-input.json"
const MaximumRequirementsFiles = 32
const maximumRequirementsIncludeDepth = MaximumRequirementsFiles
const RequirementsInputByteLimit = maxReportBytes
const RequirementsClauseLimit = maxPyPIReportEntries

// RequirementsRequest is unfrozen CLI intent. Promotion independently freezes
// every source and the actual destination before it may become resolver input.
type RequirementsRequest struct {
	Sources, Constraints, Positional []string
	Client                           string
	Upgrade, Adopt, RequireHashes    bool
}

type RequirementsSource struct {
	Name       string `json:"name"`
	Constraint bool   `json:"constraint"`
}

// RequirementsInput binds the complete, root-confined source closure. Native
// grammar validation and whole-set selection still precede inspection/Policy.
type RequirementsInput struct{ body []byte }

type requirementsInputWire struct {
	Version        int                     `json:"version"`
	Sources        []RequirementsSource    `json:"sources"`
	Files          map[string][]byte       `json:"files"`
	Positional     []string                `json:"positional"`
	Client         string                  `json:"client,omitempty"`
	Upgrade        bool                    `json:"upgrade,omitempty"`
	Runtime        string                  `json:"runtime_lock_sha256,omitempty"`
	Installed      []InstalledMetadataFile `json:"installed,omitempty"`
	OwnershipProof []byte                  `json:"ownership_proof,omitempty"`
	RequireHashes  bool                    `json:"require_hashes,omitempty"`
}

func NewRequirementsInput(sources []RequirementsSource, files map[string][]byte, positional []string) (RequirementsInput, error) {
	invalid := errors.New("requirements input must contain a bounded complete relative source closure")
	if len(sources)+len(positional) == 0 || len(sources)+len(positional) > maxPyPIReportEntries || len(files) > MaximumRequirementsFiles {
		return RequirementsInput{}, invalid
	}
	total := 0
	for _, value := range positional {
		if len(value) > maxReportBytes-total {
			return RequirementsInput{}, invalid
		}
		total += len(value)
		if strings.ContainsAny(value, "\r\n\x00") {
			return RequirementsInput{}, invalid
		}
		document, err := ParseRequirementsFile([]byte(value))
		if err != nil || len(document.lines) != 1 || document.lines[0].Include != "" || document.lines[0].Text != strings.TrimSpace(value) || strings.HasPrefix(document.lines[0].Text, "-") {
			return RequirementsInput{}, invalid
		}
	}
	documents := make(map[string]RequirementsFile, len(files))
	for name, body := range files {
		if !ValidRequirementsSourceName(name) || len(body) > maxReportBytes-total {
			return RequirementsInput{}, invalid
		}
		total += len(body)
		document, err := ParseRequirementsFile(body)
		if err != nil {
			return RequirementsInput{}, fmt.Errorf("requirements %q: %w", name, err)
		}
		documents[name] = document
	}
	active, visited := map[string]bool{}, map[string]bool{}
	heights := map[string]int{}
	var visit func(string, int) error
	visit = func(name string, depth int) error {
		if !ValidRequirementsSourceName(name) || active[name] || depth > maximumRequirementsIncludeDepth {
			return invalid
		}
		document, ok := documents[name]
		if !ok {
			return invalid
		}
		if visited[name] {
			if depth+heights[name]-1 > maximumRequirementsIncludeDepth {
				return invalid
			}
			return nil
		}
		active[name] = true
		heights[name] = 1
		for _, line := range document.lines {
			if line.Include != "" {
				child := path.Join(path.Dir(name), line.Include)
				if err := visit(child, depth+1); err != nil {
					return err
				}
				if heights[child]+1 > heights[name] {
					heights[name] = heights[child] + 1
				}
			}
		}
		delete(active, name)
		visited[name] = true
		return nil
	}
	for _, source := range sources {
		if err := visit(source.Name, 1); err != nil {
			return RequirementsInput{}, err
		}
	}
	if len(visited) != len(files) {
		return RequirementsInput{}, invalid
	}
	body, err := json.Marshal(requirementsInputWire{Version: 1, Sources: sources, Files: files, Positional: positional})
	if err != nil || len(body) > maxReportBytes {
		return RequirementsInput{}, invalid
	}
	return RequirementsInput{body: body}, nil
}

// WithInstallation binds one additive client intent and authenticated inert
// metadata. Source parsing alone does not authorize reading an installed venv.
func (r RequirementsInput) WithInstallation(client string, upgrade bool, files []InstalledMetadataFile) (RequirementsInput, error) {
	invalid := errors.New("requirements installation needs one pinned client and bounded original metadata")
	var wire requirementsInputWire
	if json.Unmarshal(r.body, &wire) != nil || wire.Client != "" || client != "pip" && client != "uv" || len(files) > UpgradeInputDistributionLimit {
		return RequirementsInput{}, invalid
	}
	seen := map[string]bool{}
	wire.Installed = make([]InstalledMetadataFile, 0, len(files))
	total := len(r.body)
	for _, file := range files {
		if len(file.Directory) > RequirementsInputByteLimit-total {
			return RequirementsInput{}, invalid
		}
		total += len(file.Directory)
		if len(file.Body) > RequirementsInputByteLimit-total {
			return RequirementsInput{}, invalid
		}
		total += len(file.Body)
		project, _, err := parseInstalledMetadataIdentity(file)
		if err != nil || seen[project.Project()] {
			return RequirementsInput{}, invalid
		}
		seen[project.Project()] = true
		wire.Installed = append(wire.Installed, InstalledMetadataFile{file.Directory, append([]byte(nil), file.Body...)})
	}
	sort.Slice(wire.Installed, func(i, j int) bool { return wire.Installed[i].Directory < wire.Installed[j].Directory })
	wire.Client, wire.Upgrade, wire.Runtime = client, upgrade, runtimeidentity.RuntimeLockSHA256
	body, err := json.Marshal(wire)
	if err != nil || len(body) > RequirementsInputByteLimit {
		return RequirementsInput{}, invalid
	}
	return RequirementsInput{body: body}, nil
}

func (r RequirementsInput) Installation() (client string, upgrade bool, files []InstalledMetadataFile, err error) {
	var wire requirementsInputWire
	if json.Unmarshal(r.body, &wire) != nil || wire.Runtime != runtimeidentity.RuntimeLockSHA256 || wire.Client != "pip" && wire.Client != "uv" {
		return "", false, nil, errors.New("requirements installation intent is unavailable")
	}
	return wire.Client, wire.Upgrade, wire.Installed, nil
}

// The held promotion guard supplies this proof only after independent original
// byte/ownership verification. Retaining it is provenance, not authority.
func (r RequirementsInput) WithOwnershipProof(body []byte) (RequirementsInput, error) {
	if _, _, _, err := r.Installation(); err != nil {
		return RequirementsInput{}, err
	}
	var wire requirementsInputWire
	if json.Unmarshal(r.body, &wire) != nil || len(wire.OwnershipProof) != 0 || len(body) == 0 || len(body) > RequirementsInputByteLimit || !json.Valid(body) {
		return RequirementsInput{}, errors.New("requirements original ownership proof is unavailable")
	}
	wire.OwnershipProof = body
	encoded, err := json.Marshal(wire)
	if err != nil || len(encoded) > RequirementsInputByteLimit {
		return RequirementsInput{}, errors.New("requirements ownership provenance exceeds bounds")
	}
	return RequirementsInput{body: encoded}, nil
}

func (r RequirementsInput) WithRequiredHashes() (RequirementsInput, error) {
	if _, _, _, err := r.Installation(); err != nil {
		return RequirementsInput{}, err
	}
	var wire requirementsInputWire
	if json.Unmarshal(r.body, &wire) != nil {
		return RequirementsInput{}, errors.New("requirements hash intent is unavailable")
	}
	wire.RequireHashes = true
	body, err := json.Marshal(wire)
	if err != nil || len(body) > RequirementsInputByteLimit {
		return RequirementsInput{}, errors.New("requirements hash intent exceeds bounds")
	}
	return RequirementsInput{body: body}, nil
}

// ValidRequirementsSourceName rejects hidden metadata and credential
// namespaces as well as escaping paths. Names never grant content trust.
func ValidRequirementsSourceName(name string) bool {
	if !utf8.ValidString(name) || name == "" || len(name) > 4096 || path.IsAbs(name) || path.Clean(name) != name || name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\:\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(strings.ToLower(part), ".env") || strings.HasPrefix(strings.ToLower(part), "credentials.") || strings.HasPrefix(strings.ToLower(part), "secrets.") {
			return false
		}
		switch strings.ToLower(part) {
		case "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "credentials", "secrets", "authorized_keys":
			return false
		}
	}
	return true
}

func (r RequirementsInput) Control() (domain.ProjectControlFile, error) {
	if len(r.body) == 0 {
		return domain.ProjectControlFile{}, errors.New("requirements input is unavailable")
	}
	return domain.NewProjectControlFile(RequirementsInputControlName, r.body, true)
}

func (r RequirementsInput) Controls() ([]domain.ProjectControlFile, error) {
	if _, _, _, err := r.Installation(); err != nil {
		return nil, err
	}
	source, err := r.Control()
	if err != nil {
		return nil, err
	}
	selection, err := domain.NewProjectControlFile(RequirementsSelectionControlName, nil, false)
	if err != nil {
		return nil, err
	}
	return []domain.ProjectControlFile{source, selection}, nil
}

func ParseRequirementsControl(control domain.ProjectControlFile) (RequirementsInput, error) {
	if control.Name() != RequirementsInputControlName || !control.Present() {
		return RequirementsInput{}, errors.New("requirements source control is unavailable")
	}
	var wire requirementsInputWire
	decoder := json.NewDecoder(bytes.NewReader(control.Body()))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&wire) != nil || ensureSingleJSONValue(decoder) != nil || wire.Version != 1 {
		return RequirementsInput{}, errors.New("requirements source control is invalid")
	}
	input, err := NewRequirementsInput(wire.Sources, wire.Files, wire.Positional)
	if err == nil && wire.Client != "" {
		input, err = input.WithInstallation(wire.Client, wire.Upgrade, wire.Installed)
	}
	if err == nil && len(wire.OwnershipProof) > 0 {
		input, err = input.WithOwnershipProof(wire.OwnershipProof)
	}
	if err == nil && wire.RequireHashes {
		input, err = input.WithRequiredHashes()
	}
	if err != nil || !bytes.Equal(input.body, control.Body()) {
		return RequirementsInput{}, errors.New("requirements source control is not canonical")
	}
	return input, nil
}

// ParserControl sends only logical dependency declarations to the metadata
// guest. Original comments and unrelated file content remain in the host guard.
func (r RequirementsInput) ParserControl() (domain.ProjectControlFile, error) {
	control, err := r.Control()
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	var original requirementsInputWire
	if json.Unmarshal(r.body, &original) != nil {
		return domain.ProjectControlFile{}, errors.New("requirements source is unavailable")
	}
	files := map[string][]RequirementsLine{}
	for name, body := range original.Files {
		document, err := ParseRequirementsFile(body)
		if err != nil {
			return domain.ProjectControlFile{}, err
		}
		files[name] = document.Lines()
	}
	wire := struct {
		Version       int                           `json:"version"`
		SourceSHA256  string                        `json:"source_sha256"`
		Sources       []RequirementsSource          `json:"sources"`
		Files         map[string][]RequirementsLine `json:"files"`
		Positional    []string                      `json:"positional"`
		RequireHashes bool                          `json:"require_hashes"`
	}{1, control.Digest().String(), original.Sources, files, original.Positional, original.RequireHashes}
	body, err := json.Marshal(wire)
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	return domain.NewProjectControlFile("python-requirements-parser.json", body, true)
}
