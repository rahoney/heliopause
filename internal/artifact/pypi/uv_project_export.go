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
	body      []byte
	native    UVNativeProjectLock
	selection UVProjectLock
}

type uvProjectExportArchive struct {
	URL      string            `toml:"url"`
	Size     int64             `toml:"size"`
	Hashes   map[string]string `toml:"hashes"`
	Uploaded time.Time         `toml:"upload-time"`
}

type uvProjectExportWire struct {
	Version  string `toml:"lock-version"`
	Creator  string `toml:"created-by"`
	Python   string `toml:"requires-python"`
	Packages *[]struct {
		Name    string                   `toml:"name"`
		Version string                   `toml:"version"`
		Index   string                   `toml:"index"`
		Sdist   *uvProjectExportArchive  `toml:"sdist"`
		Wheels  []uvProjectExportArchive `toml:"wheels"`
	} `toml:"packages"`
}

func (l UVNativeProjectLock) ParseTargetExport(body []byte) (UVProjectExport, error) {
	if len(l.body) == 0 || !validUVProjectText(body, false) {
		return UVProjectExport{}, errors.New("UV native project or target export is unavailable")
	}
	var wire uvProjectExportWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return UVProjectExport{}, err
	}
	if wire.Version != "1.0" || wire.Creator != "uv" || !requiresPythonMetadataMatches(l.python, wire.Python) || wire.Packages == nil || len(*wire.Packages) != len(l.packages) {
		return UVProjectExport{}, errors.New("UV project export header or complete set differs from native lock")
	}
	packages := make([]map[string]any, 0, len(*wire.Packages))
	for _, item := range *wire.Packages {
		if !uvPublicProjectIndex(item.Index) || len(item.Wheels) == 0 || len(item.Wheels) > maxUVWheelAlternatives {
			return UVProjectExport{}, errors.New("UV project export source or wheel count is unsupported")
		}
		original, hasSource := l.sources[item.Name]
		if hasSource != (item.Sdist != nil) {
			return UVProjectExport{}, errors.New("UV project export source alternative presence changed")
		}
		if item.Sdist != nil && (item.Sdist.URL != original.URL || item.Sdist.Size != original.Size || len(item.Sdist.Hashes) != 1 || item.Sdist.Hashes["sha256"] != strings.TrimPrefix(original.Hash, "sha256:")) {
			return UVProjectExport{}, errors.New("UV project export source alternative differs from native declaration")
		}
		wheels := make([]map[string]any, 0, len(item.Wheels))
		for _, item := range item.Wheels {
			wheel := map[string]any{"url": item.URL, "size": item.Size, "hashes": item.Hashes}
			if !item.Uploaded.IsZero() {
				wheel["upload-time"] = item.Uploaded
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
	return UVProjectExport{append([]byte(nil), body...), l, selection}, nil
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
	return e.selection.BindProject(install, controls, inputs, pages, target, limits)
}
