package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
)

// UVReleaseMetadataURL derives the only release-metadata endpoint used by the
// uv resolver. It cannot accept caller-selected origins, queries or fragments.
func UVReleaseMetadataURL(project, version string) (string, error) {
	name, err := NormalizeProjectName(project)
	if err != nil {
		return "", err
	}
	canonical, err := NormalizeVersion(version)
	if err != nil {
		return "", err
	}
	return "https://pypi.org/pypi/" + url.PathEscape(name) + "/" + url.PathEscape(canonical) + "/json", nil
}

// ParseUVReleaseMetadata normalizes the official exact-release JSON response
// into the existing distribution-list type. This is not a synthetic Simple API
// response: it validates both info identity and every release file independently.
// Keeping one exact release avoids relaxing the existing whole-project bounds.
func ParseUVReleaseMetadata(project, version string, body []byte) (SimpleProject, error) {
	name, err := NormalizeProjectName(project)
	if err != nil || name != project || len(body) == 0 || len(body) > maxSimpleResponseBytes {
		return SimpleProject{}, errors.New("uv release metadata request exceeds supported bounds")
	}
	canonical, err := NormalizeVersion(version)
	if err != nil || canonical != version {
		return SimpleProject{}, errors.New("uv release metadata version is invalid")
	}
	var response struct {
		Info struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"info"`
		Files []struct {
			Filename string            `json:"filename"`
			URL      string            `json:"url"`
			Size     uint64            `json:"size"`
			Digests  map[string]string `json:"digests"`
			Python   string            `json:"requires_python"`
			Yanked   json.RawMessage   `json:"yanked"`
			Type     string            `json:"packagetype"`
		} `json:"urls"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil || ensureSingleJSONValue(decoder) != nil {
		return SimpleProject{}, errors.New("uv release metadata is invalid JSON")
	}
	actualName, nameErr := NormalizeProjectName(response.Info.Name)
	actualVersion, versionErr := NormalizeVersion(response.Info.Version)
	if nameErr != nil || versionErr != nil || actualName != name || actualVersion != version || len(response.Files) == 0 || len(response.Files) > maxPyPIReportEntries {
		return SimpleProject{}, errors.New("uv release metadata identity or file count mismatch")
	}
	result := SimpleProject{project: name, source: PublicPyPIProfile().Source()}
	seen := make(map[string]bool, len(response.Files))
	for _, item := range response.Files {
		var fileName, fileVersion string
		switch item.Type {
		case "bdist_wheel":
			fileName, fileVersion, _, _, _, err = ParseWheelFilename(item.Filename)
		case "sdist":
			fileName, fileVersion, err = parseSdistFilename(item.Filename)
		default:
			return SimpleProject{}, errors.New("uv release metadata distribution type is unsupported")
		}
		if err != nil || fileName != name || fileVersion != version {
			return SimpleProject{}, errors.New("uv release metadata file identity mismatch")
		}
		file, err := parseSimpleFileForProfile(simpleFile{Filename: item.Filename, URL: item.URL, Size: item.Size, Hashes: item.Digests, RequiresPython: item.Python, Yanked: item.Yanked}, PublicPyPIProfile())
		if err != nil || seen[file.url] {
			return SimpleProject{}, errors.New("uv release metadata file is invalid or duplicated")
		}
		seen[file.url] = true
		result.files = append(result.files, file)
	}
	sort.Slice(result.files, func(i, j int) bool { return result.files[i].url < result.files[j].url })
	return result, nil
}
