package pypi

import (
	"bytes"
	"errors"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const UVProjectExportControl = "uv-project-export.pylock.toml"

// UVProjectExport retains native export bytes separately from its derived
// wheel-only validation view. Both remain unapproved; BindProject reuses
// independent full archive/metadata validation and grants no Policy authority.
type UVProjectExport struct {
	body       []byte
	native     UVNativeProjectLock
	selection  UVProjectLock
	portable   []uvProjectExportPackage
	target     *uvProjectTargetWire
	projection RequirementsNativeInput
}

type uvProjectExportArchive struct {
	Name     string            `toml:"name,omitempty"`
	URL      string            `toml:"url"`
	Size     int64             `toml:"size"`
	Hashes   map[string]string `toml:"hashes"`
	Uploaded time.Time         `toml:"upload-time"`
}

type uvProjectExportPackage struct {
	Name    string                   `toml:"name"`
	Version string                   `toml:"version"`
	Marker  string                   `toml:"marker,omitempty"`
	Index   string                   `toml:"index"`
	Sdist   *uvProjectExportArchive  `toml:"sdist"`
	Wheels  []uvProjectExportArchive `toml:"wheels"`
}

type uvProjectExportWire struct {
	Version  string                    `toml:"lock-version"`
	Creator  string                    `toml:"created-by"`
	Python   string                    `toml:"requires-python"`
	Packages *[]uvProjectExportPackage `toml:"packages"`
}

// UV emits a normalized wheel name for legacy mixed-case distribution names.
// Keep the actual URL/filename for independent source and archive verification;
// every version, build and compatibility tag must remain byte-identical.
func validUVExportWheelName(declared, filename, project string) bool {
	_, suffix, present := strings.Cut(filename, "-")
	return declared == "" || declared == filename || present && declared == strings.ReplaceAll(project, "-", "_")+"-"+suffix
}

func (l UVNativeProjectLock) ParseTargetExport(body []byte) (UVProjectExport, error) {
	if len(l.body) == 0 || !validUVProjectText(body, false) {
		return UVProjectExport{}, errors.New("UV native project or target export is unavailable")
	}
	if l.rich {
		return l.parseScopedExport(body)
	}
	var wire uvProjectExportWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return UVProjectExport{}, err
	}
	if wire.Version != "1.0" || wire.Creator != "uv" || !uvProjectPythonMatches(l.python, wire.Python) || wire.Packages == nil || len(*wire.Packages) != len(l.packages) {
		return UVProjectExport{}, errors.New("UV project export header or complete set differs from native lock")
	}
	packages := make([]map[string]any, 0, len(*wire.Packages))
	for _, item := range *wire.Packages {
		if item.Marker != "" || !uvPublicProjectIndex(item.Index) || len(item.Wheels) == 0 || len(item.Wheels) > maxUVWheelAlternatives {
			return UVProjectExport{}, errors.New("UV project export source or wheel count is unsupported")
		}
		original, hasSource := l.sources[item.Name]
		if hasSource != (item.Sdist != nil) {
			return UVProjectExport{}, errors.New("UV project export source alternative presence changed")
		}
		if item.Sdist != nil && (item.Sdist.Name != "" || item.Sdist.URL != original.URL || item.Sdist.Size != original.Size || len(item.Sdist.Hashes) != 1 || item.Sdist.Hashes["sha256"] != strings.TrimPrefix(original.Hash, "sha256:")) {
			return UVProjectExport{}, errors.New("UV project export source alternative differs from native declaration")
		}
		wheels := make([]map[string]any, 0, len(item.Wheels))
		for _, archive := range item.Wheels {
			filename := archive.URL[strings.LastIndex(archive.URL, "/")+1:]
			if !validUVExportWheelName(archive.Name, filename, item.Name) {
				return UVProjectExport{}, errors.New("UV export wheel name differs from native archive")
			}
			wheel := map[string]any{"url": archive.URL, "size": archive.Size, "hashes": archive.Hashes}
			if !archive.Uploaded.IsZero() {
				wheel["upload-time"] = archive.Uploaded
			}
			wheels = append(wheels, wheel)
		}
		packages = append(packages, map[string]any{"name": item.Name, "version": item.Version, "wheels": wheels})
	}
	view, err := toml.Marshal(map[string]any{"lock-version": wire.Version, "created-by": wire.Creator, "requires-python": wire.Python, "packages": packages})
	if err != nil {
		return UVProjectExport{}, err
	}
	selection, err := ParseUVProjectLock(view, runtimeidentity.PythonVersion)
	if err != nil {
		return UVProjectExport{}, err
	}
	if err := l.ValidateTargetExport(selection); err != nil {
		return UVProjectExport{}, err
	}
	return UVProjectExport{body: append([]byte(nil), body...), native: l, selection: selection}, nil
}

func (e UVProjectExport) Packages() []UVLockedPackage { return e.selection.Packages() }

func (e UVProjectExport) SelectInstalledWheelControls(controls []UVInstalledWheelControls, target WheelTarget) ([]UVLockedWheel, error) {
	if len(e.body) == 0 {
		return nil, errors.New("UV project export is unavailable")
	}
	return e.selection.SelectInstalledWheelControls(controls, target)
}

// Controls binds actual native inputs/output and the separately named derived
// validation view. A caller must also include its frozen intent/runtime guard.
func (e UVProjectExport) Controls() ([]domain.ProjectControlFile, error) {
	if len(e.body) == 0 {
		return nil, errors.New("UV project export is unavailable")
	}
	manifest, err := domain.NewProjectControlFile(UVProjectInputControl, e.native.manifest, true)
	if err != nil {
		return nil, err
	}
	native, err := e.native.Control()
	if err != nil {
		return nil, err
	}
	export, err := domain.NewProjectControlFile(UVProjectExportControl, e.body, true)
	if err != nil {
		return nil, err
	}
	view, err := e.selection.Control()
	if err != nil {
		return nil, err
	}
	return []domain.ProjectControlFile{manifest, native, export, view}, nil
}

func (e UVProjectExport) BindProject(install domain.InstallContext, controls []domain.ProjectControlFile, inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) (domain.ProjectDependencySnapshot, error) {
	expected, err := e.Controls()
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	for _, want := range expected {
		matches := 0
		for _, got := range controls {
			if got.Name() == want.Name() {
				if !got.Present() || !bytes.Equal(got.Body(), want.Body()) {
					return domain.ProjectDependencySnapshot{}, errors.New("UV project provenance control differs from native selection")
				}
				matches++
			}
		}
		if matches != 1 {
			return domain.ProjectDependencySnapshot{}, errors.New("UV project provenance control is missing or repeated")
		}
	}
	if e.native.rich {
		if len(pages) != len(e.selection.packages) {
			return domain.ProjectDependencySnapshot{}, errors.New("UV target independent source coverage is incomplete")
		}
		candidates, err := e.bindTargetMetadata(inputs, pages, target, limits, nil)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		intent, err := e.IntentControl()
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		found := false
		for _, got := range controls {
			if got.Name() == intent.Name() {
				found = got.Digest() == intent.Digest()
			}
		}
		if !found {
			return domain.ProjectDependencySnapshot{}, errors.New("UV target metadata intent differs")
		}
		return e.selection.snapshotFromCandidates(install, controls, candidates, target)
	}
	return e.selection.BindProject(install, controls, inputs, pages, target, limits)
}
