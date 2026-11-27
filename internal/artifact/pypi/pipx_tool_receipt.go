package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pipxPathWire struct {
	Type string `json:"__type__"`
	Path string `json:"__Path__"`
}

type pipxToolReceiptWire struct {
	Backend     string                     `json:"backend"`
	Environment string                     `json:"environment"`
	Exposure    *bool                      `json:"exposure_enabled"`
	Injected    map[string]json.RawMessage `json:"injected_packages"`
	Main        *struct {
		AppPaths            []pipxPathWire             `json:"app_paths"`
		AppPathsDeps        map[string]json.RawMessage `json:"app_paths_of_dependencies"`
		Apps                []string                   `json:"apps"`
		AppsDeps            []string                   `json:"apps_of_dependencies"`
		CompletionPaths     []pipxPathWire             `json:"completion_paths"`
		CompletionPathsDeps map[string]json.RawMessage `json:"completion_paths_of_dependencies"`
		Completions         []string                   `json:"completions"`
		CompletionsDeps     []string                   `json:"completions_of_dependencies"`
		Cooldown            json.RawMessage            `json:"cooldown_days"`
		ExpectedApps        []string                   `json:"expected_apps"`
		IncludeApps         *bool                      `json:"include_apps"`
		IncludeDependencies bool                       `json:"include_dependencies"`
		IncludeResources    []string                   `json:"include_resources_from"`
		LockFile            json.RawMessage            `json:"lock_file"`
		ManPages            []string                   `json:"man_pages"`
		ManPagesDeps        []string                   `json:"man_pages_of_dependencies"`
		ManPaths            []pipxPathWire             `json:"man_paths"`
		ManPathsDeps        map[string]json.RawMessage `json:"man_paths_of_dependencies"`
		Package             string                     `json:"package"`
		Requirement         string                     `json:"package_or_url"`
		Version             string                     `json:"package_version"`
		Pinned              bool                       `json:"pinned"`
		PipArgs             []string                   `json:"pip_args"`
		Suffix              string                     `json:"suffix"`
	} `json:"main_package"`
	Schema      string       `json:"pipx_metadata_version"`
	Python      string       `json:"python_version"`
	Interpreter pipxPathWire `json:"source_interpreter"`
	VenvArgs    []string     `json:"venv_args"`
}

// ParsePipxToolReceipt reads the actual pipx 1.18.4 metadata 0.12 UV-backend
// single-package console-script subset. No Python code is executed. Stored pip
// arguments remain opaque and must never become an execution recipe.
func ParsePipxToolReceipt(body []byte) (ToolReceipt, error) {
	if !validToolText(body) {
		return ToolReceipt{}, errors.New("pipx metadata is absent, invalid or exceeds bounds")
	}
	if err := validateNativeJSONKeys(body, 6); err != nil {
		return ToolReceipt{}, err
	}
	// encoding/json otherwise matches field names case-insensitively, allowing
	// differently cased duplicates to override an earlier native field.
	var keys any
	if err := json.Unmarshal(body, &keys); err != nil {
		return ToolReceipt{}, err
	}
	if !exactPipxKeyCase(keys) {
		return ToolReceipt{}, errors.New("pipx metadata field casing is unsupported")
	}
	var wire pipxToolReceiptWire
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return ToolReceipt{}, err
	}
	m := wire.Main
	if wire.Schema != "0.12" || wire.Backend != "uv" || wire.Exposure == nil || !*wire.Exposure || len(wire.Injected) != 0 || m == nil || wire.Python != "Python "+runtimeidentity.PythonVersion || wire.Interpreter.Type != "Path" || !validToolAbsolutePath(wire.Interpreter.Path) || len(wire.VenvArgs) != 0 {
		return ToolReceipt{}, errors.New("pipx metadata schema, backend, Python or environment is unsupported")
	}
	project, err := NormalizeProjectName(m.Package)
	requirement, reqErr := parseUVProjectRequirement(m.Requirement)
	version, verErr := NormalizeVersion(m.Version)
	if err != nil || project != m.Package || wire.Environment != project || reqErr != nil || requirement.Project() != project || verErr != nil || version != m.Version || !requirement.AcceptsVersion(version) {
		return ToolReceipt{}, errors.New("pipx main package identity or version is inconsistent")
	}
	if m.IncludeApps == nil || !*m.IncludeApps || m.IncludeDependencies || m.Pinned || m.Suffix != "" || len(m.IncludeResources)+len(m.ExpectedApps)+len(m.AppPathsDeps)+len(m.AppsDeps)+len(m.CompletionPaths)+len(m.CompletionPathsDeps)+len(m.Completions)+len(m.CompletionsDeps)+len(m.ManPages)+len(m.ManPagesDeps)+len(m.ManPaths)+len(m.ManPathsDeps) != 0 || !emptyPipxNull(m.Cooldown) || !emptyPipxNull(m.LockFile) {
		return ToolReceipt{}, errors.New("pipx injected, pinned, dependency or resource exposure is unsupported")
	}
	var paths []string
	for _, p := range m.AppPaths {
		if p.Type != "Path" {
			return ToolReceipt{}, errors.New("pipx app path type is unsupported")
		}
		paths = append(paths, p.Path)
	}
	commands, err := toolCommands(m.Apps, paths)
	if err != nil {
		return ToolReceipt{}, err
	}
	return ToolReceipt{client: "pipx", environment: wire.Environment, interpreter: wire.Interpreter.Path, version: version, requirement: requirement, commands: commands, body: bytes.Clone(body)}, nil
}

func emptyPipxNull(value json.RawMessage) bool {
	return len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

func exactPipxKeyCase(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key != "__Path__" && key != strings.ToLower(key) || !exactPipxKeyCase(child) {
				return false
			}
		}
	case []any:
		for _, child := range v {
			if !exactPipxKeyCase(child) {
				return false
			}
		}
	}
	return true
}
