package releaseinstall

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/semver"
)

const maximumManifestBytes = 1 << 20

var (
	commitPattern         = regexp.MustCompile(`^[0-9a-f]{40}$`)
	releaseRunPattern     = regexp.MustCompile(`^https://github\.com/rahoney/heliopause/actions/runs/([1-9][0-9]{0,19})$`)
	runtimeReleasePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

// ReleaseFile is an untrusted manifest declaration, never installation approval.
type ReleaseFile struct {
	Name, Platform, Role, SHA256 string
	Size                         int64
}

// ReleaseManifest is a bounded, structurally checked view. Authenticity and
// content inspection still belong to their existing providers and Policy.
type ReleaseManifest struct{ value manifest }

func (m ReleaseManifest) Version() string      { return m.value.Version }
func (m ReleaseManifest) SourceCommit() string { return m.value.SourceCommit }
func (m ReleaseManifest) WorkflowRun() string {
	match := releaseRunPattern.FindStringSubmatch(m.value.WorkflowRun)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}
func (m ReleaseManifest) RuntimeLockSHA256() string { return m.value.RuntimeLock.SHA256 }
func (m ReleaseManifest) GVisorRelease() string     { return m.value.RuntimeLock.GVisorRelease }
func (m ReleaseManifest) GVisorCommit() string      { return m.value.RuntimeLock.GVisorCommit }

func (m ReleaseManifest) Files() []ReleaseFile {
	if m.value.Schema != manifestSchema {
		return nil
	}
	files := make([]ReleaseFile, 0, len(m.value.Assets)+2)
	for _, item := range m.value.Assets {
		files = append(files, ReleaseFile(item))
	}
	for _, item := range []record{m.value.SBOM, m.value.RuntimeImages} {
		files = append(files, ReleaseFile{Name: item.Name, Role: "release-metadata", SHA256: item.SHA256, Size: item.Size})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files
}

// NativeUpdateFiles selects the complete approved Linux/amd64 native role set.
// Historical releases lacking the separate installer fail closed.
func (m ReleaseManifest) NativeUpdateFiles() ([]ReleaseFile, error) {
	roles := map[string]string{
		"helox-linux-amd64":                     "helox",
		"haa_gvisor_observer-linux-amd64":       "gvisor-observer",
		"haa-network-policy-helper-linux-amd64": "network-policy-helper",
		"haa-release-installer-linux-amd64":     "release-installer",
	}
	result := make([]ReleaseFile, 0, len(roles))
	for _, item := range m.Files() {
		role, found := roles[item.Name]
		if found && item.Platform == "linux/amd64" && item.Role == role {
			result = append(result, item)
		}
	}
	if len(result) != len(roles) {
		return nil, errors.New("release does not contain the complete native update role set")
	}
	return result, nil
}

func parseManifest(body []byte) (manifest, error) {
	var value manifest
	if json.Unmarshal(body, &value) != nil || value.Schema != manifestSchema || !versionPattern.MatchString(value.Version) || !commitPattern.MatchString(value.SourceCommit) || value.WorkflowRun == "" || len(value.Assets) == 0 {
		return manifest{}, errors.New("release manifest is invalid")
	}
	if !validRuntimeLock(value.RuntimeLock) || !validRecord(value.SBOM, sbomName) || !validRecord(value.RuntimeImages, runtimeImagesName) {
		return manifest{}, errors.New("release manifest binding is invalid")
	}
	return value, nil
}

// ParseReleaseManifest is the strict product reader of the existing release
// wire format. It never reads a path, executes a file or changes installation.
func ParseReleaseManifest(body []byte) (ReleaseManifest, error) {
	if len(body) == 0 || len(body) > maximumManifestBytes || !utf8.Valid(body) || !uniqueManifestJSON(body) {
		return ReleaseManifest{}, errors.New("release manifest JSON is invalid or excessive")
	}
	value, err := parseManifest(body)
	if err != nil {
		return ReleaseManifest{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&value) != nil || decoder.Decode(&struct{}{}) != io.EOF || !semver.IsValid(value.Version) || !releaseRunPattern.MatchString(value.WorkflowRun) {
		return ReleaseManifest{}, errors.New("release manifest context or fields are invalid")
	}
	if value.SBOM.Size > 8<<20 || value.RuntimeImages.Size > 1<<20 || !runtimeReleasePattern.MatchString(value.RuntimeLock.GVisorRelease) {
		return ReleaseManifest{}, errors.New("release metadata binding exceeds supported bounds")
	}
	want := map[string]struct{ platform, role string }{
		"helox-linux-amd64":                     {"linux/amd64", "helox"},
		"helox-linux-arm64":                     {"linux/arm64", "helox"},
		"helox-darwin-amd64":                    {"darwin/amd64", "helox"},
		"helox-darwin-arm64":                    {"darwin/arm64", "helox"},
		"haa_gvisor_observer-linux-amd64":       {"linux/amd64", "gvisor-observer"},
		"haa-network-policy-helper-linux-amd64": {"linux/amd64", "network-policy-helper"},
		"helox-linux-amd64.deb":                 {"linux/amd64", "os-package-deb"},
		"helox-linux-amd64.rpm":                 {"linux/amd64", "os-package-rpm"},
		"haa-release-installer-linux-amd64":     {"linux/amd64", "release-installer"},
	}
	if len(value.Assets) != 8 && len(value.Assets) != 9 {
		return ReleaseManifest{}, errors.New("release asset set is incomplete")
	}
	seen := make(map[string]bool, len(value.Assets))
	for _, item := range value.Assets {
		spec, found := want[item.Name]
		if !found || seen[item.Name] || spec.platform != item.Platform || spec.role != item.Role || !validRecord(record{item.Name, item.SHA256, item.Size}, item.Name) {
			return ReleaseManifest{}, errors.New("release asset binding is invalid or ambiguous")
		}
		seen[item.Name] = true
	}
	for name := range want {
		if name != "haa-release-installer-linux-amd64" && !seen[name] {
			return ReleaseManifest{}, errors.New("release asset set is incomplete")
		}
	}
	return ReleaseManifest{value: value}, nil
}

// The decoder's struct matching is case-insensitive; reject noncanonical and
// duplicate keys before decoding so no consumer can interpret another value.
func uniqueManifestJSON(body []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var value func(int) bool
	value = func(depth int) bool {
		if depth > 8 {
			return false
		}
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		delimiter, compound := token.(json.Delim)
		if !compound {
			return true
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return false
				}
				name, ok := key.(string)
				if !ok || seen[name] || strings.ToLower(name) != name {
					return false
				}
				seen[name] = true
				if !value(depth + 1) {
					return false
				}
			}
			end, err := decoder.Token()
			return err == nil && end == json.Delim('}')
		case '[':
			for decoder.More() {
				if !value(depth + 1) {
					return false
				}
			}
			end, err := decoder.Token()
			return err == nil && end == json.Delim(']')
		default:
			return false
		}
	}
	if !value(0) {
		return false
	}
	_, err := decoder.Token()
	return err == io.EOF
}
