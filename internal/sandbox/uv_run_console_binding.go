package sandbox

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errUVRunConsoleData = errors.New("native UV console DATA differs from the complete frozen original or supported layout")

// uvRunConsoleOriginal measures current owned original bytes. The acquired
// subject and returned console entry are DATA, not inspection/Policy authority.
func uvRunConsoleOriginal(frozen *pipxNativeInput, command string) (artifactpypi.ConsoleExecutionEntry, []artifactpypi.WheelInspection, error) {
	return pythonRunConsoleOriginal(frozen, command, false)
}
func pipxRunConsoleOriginal(frozen *pipxNativeInput, command string) (artifactpypi.ConsoleExecutionEntry, []artifactpypi.WheelInspection, error) {
	return pythonRunConsoleOriginal(frozen, command, true)
}
func pythonRunConsoleOriginal(frozen *pipxNativeInput, command string, pipx bool) (artifactpypi.ConsoleExecutionEntry, []artifactpypi.WheelInspection, error) {
	if !frozen.valid() {
		return artifactpypi.ConsoleExecutionEntry{}, nil, errPipxNativeInput
	}
	var primary domain.ResolvedArtifactIdentity
	for _, node := range frozen.graph.Nodes() {
		if node.Role() == domain.DependencyPrimary {
			primary = node.Artifact().Identity()
		}
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	limits := artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits()
	var entry artifactpypi.ConsoleExecutionEntry
	var inspections []artifactpypi.WheelInspection
	for _, file := range frozen.files {
		info, err := artifactpypi.InspectWheel(bytes.NewReader(file.body), int64(len(file.body)), file.name, file.digest, target, limits)
		if err != nil {
			return artifactpypi.ConsoleExecutionEntry{}, nil, err
		}
		inspections = append(inspections, info)
		if info.Project != primary.Name() || info.Version != primary.Version() {
			continue
		}
		if entry.Valid() || primary.Source() != artifactpypi.PublicPyPIProfile().Source() || primary.Variant() != "wheel" {
			return artifactpypi.ConsoleExecutionEntry{}, nil, errUVRunConsoleData
		}
		digest, err := domain.NewSHA256Digest(file.digest)
		if err != nil {
			return artifactpypi.ConsoleExecutionEntry{}, nil, err
		}
		measured, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(primary, digest, "uv-native-data:"+frozen.id, uint64(len(file.body)), "sha256:"+file.digest)
		if err != nil {
			return artifactpypi.ConsoleExecutionEntry{}, nil, err
		}
		inventory, err := artifactpypi.InspectConsoleExecutionWheel(measured, bytes.NewReader(file.body), file.name, target, limits)
		if err != nil {
			return artifactpypi.ConsoleExecutionEntry{}, nil, err
		}
		if pipx {
			entry, err = artifactpypi.SelectPipxConsoleExecutionEntry(inventory, command)
		} else {
			entry, err = artifactpypi.SelectConsoleExecutionEntry(inventory, command)
		}
		if err != nil {
			return artifactpypi.ConsoleExecutionEntry{}, nil, err
		}
	}
	if !entry.Valid() || len(inspections) != len(frozen.files) {
		return artifactpypi.ConsoleExecutionEntry{}, nil, errUVRunConsoleData
	}
	return entry, inspections, nil
}

// bindUVRunConsoleData checks selection/source/console DATA only. Native
// installed RECORD, cache metadata and activation scripts confer no ownership
// and are never copied into the later current approved execution closure.
func bindUVRunConsoleData(frozen *pipxNativeInput, command string, facts []uvRunCensusFact) (artifactpypi.ConsoleExecutionEntry, error) {
	return bindPythonRunConsoleData(frozen, command, facts, false)
}

func bindPipxRunConsoleData(frozen *pipxNativeInput, command string, facts []uvRunCensusFact) (artifactpypi.ConsoleExecutionEntry, error) {
	return bindPythonRunConsoleData(frozen, command, facts, true)
}

// Both pinned native frontends use the same UV installer. Share the complete
// original/control/payload checks; only their measured environment recipes differ.
func bindPythonRunConsoleData(frozen *pipxNativeInput, command string, facts []uvRunCensusFact, pipx bool) (artifactpypi.ConsoleExecutionEntry, error) {
	entry, inspections, err := pythonRunConsoleOriginal(frozen, command, pipx)
	if err != nil {
		return artifactpypi.ConsoleExecutionEntry{}, err
	}
	if len(facts) == 0 || len(facts) > 4096 {
		return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
	}
	byName := map[string]uvRunCensusFact{}
	var expanded int64
	for _, f := range facts {
		root := f.Name == "" || f.Name == "."
		if !root && !validClosureDestination(f.Name) || f.Size < 0 || f.Size > (4<<20)-expanded {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		if _, ok := byName[f.Name]; ok {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		expanded += f.Size
		if len(f.Body) > 0 {
			sum := sha256.Sum256(f.Body)
			if int64(len(f.Body)) != f.Size || hex.EncodeToString(sum[:]) != f.SHA256 {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
		}
		byName[f.Name] = f
	}
	regular := func(f uvRunCensusFact) bool {
		return (f.Kind == tar.TypeReg || f.Kind == 0) && f.UID == 1000 && f.GID == 1000 && f.Mode&^int64(0777) == 0 && f.Mode&0022 == 0 && f.Link == ""
	}
	var env string
	for _, f := range facts {
		parts := strings.Split(f.Name, "/")
		prefix, digits := "uv-run-cache/archive-v0", 16
		configuration := uvToolVenvConfiguration() + "relocatable = true\n"
		if pipx {
			prefix, digits = "pipx/.cache", 15
			configuration = uvToolVenvConfiguration()
		}
		if len(parts) != 4 || strings.Join(parts[:2], "/") != prefix || parts[3] != "pyvenv.cfg" {
			continue
		}
		root := strings.TrimSuffix(f.Name, "/pyvenv.cfg")
		alias, ok := byName[root+"/bin/python"]
		if !ok || alias.Kind != tar.TypeSymlink {
			continue
		}
		if env != "" || len(parts[2]) != digits || !regular(f) || string(f.Body) != configuration {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		for _, c := range parts[2] {
			if pipx && !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
		}
		env = root
	}
	if env == "" {
		return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
	}
	if pipx {
		metadata, ok := byName[env+"/pipx_metadata.json"]
		if !ok || !regular(metadata) || metadata.Mode != 0600 || metadata.Size < 1 || metadata.Size > 64<<10 {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		for _, f := range facts {
			if f.Name == env || strings.HasPrefix(f.Name, env+"/") {
				continue
			}
			directory := f.Name == "" || f.Name == "." || f.Name == "pipx" || f.Name == "pipx/.cache" || f.Name == "pipx/py" || f.Name == "pipx/venvs" || f.Name == "pipx/logs" || f.Name == "tool-bin" || f.Name == "tool-man"
			if directory && f.Kind == tar.TypeDir && f.UID == 1000 && f.GID == 1000 && f.Size == 0 && f.Link == "" && f.Mode&^int64(0777) == 0 && f.Mode&0022 == 0 {
				continue
			}
			auxiliary := f.Name == "pipx/.cache/CACHEDIR.TAG" || f.Name == "pipx/py/CACHEDIR.TAG" || path.Dir(f.Name) == "pipx/logs" && pipxNativeLogName.MatchString(path.Base(f.Name))
			if !regular(f) || !auxiliary && !(f.Name == "pipx/.cache/."+path.Base(env)+".lock" && f.Size == 0) {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
		}
	}
	aliases := map[string]string{"bin/python": "/usr/local/bin/python", "bin/python3": "python", "bin/python3.14": "python", "lib64": "lib"}
	for name, target := range aliases {
		f, ok := byName[env+"/"+name]
		if !ok || f.Kind != tar.TypeSymlink || f.Link != target || f.Size != 0 || f.Mode != 0777 || f.UID != 1000 || f.GID != 1000 {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
	}
	for _, name := range []string{env, env + "/bin", env + "/lib", env + "/lib/python3.14", env + "/lib/python3.14/site-packages"} {
		f, ok := byName[name]
		if !ok || f.Kind != tar.TypeDir || f.Size != 0 || f.Link != "" || f.UID != 1000 || f.GID != 1000 || f.Mode&^int64(0777) != 0 || f.Mode&0022 != 0 {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
	}
	site := env + "/lib/python3.14/site-packages/"
	controls := map[string]*artifactpypi.UVInstalledWheelControls{}
	for _, f := range facts {
		if !strings.HasPrefix(f.Name, env+"/") {
			continue
		}
		if f.Kind == tar.TypeDir {
			if f.UID != 1000 || f.GID != 1000 || f.Size != 0 || f.Link != "" || f.Mode&^int64(0777) != 0 || f.Mode&0022 != 0 {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
			continue
		}
		if f.Kind == tar.TypeSymlink {
			if _, ok := aliases[strings.TrimPrefix(f.Name, env+"/")]; !ok {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
			continue
		}
		// Same empty UV lock DATA exception as the existing pipx install reader.
		// It is never an executable payload or installed ownership evidence.
		if pipx && f.Name == env+"/.lock" && f.Kind == tar.TypeReg && f.UID == 1000 && f.GID == 1000 &&
			f.Mode == 0666 && f.Size == 0 && f.Link == "" && f.SHA256 == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" && len(f.Body) == 0 {
			continue
		}
		if !regular(f) {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		if !strings.HasPrefix(f.Name, site) {
			continue
		}
		relative := strings.TrimPrefix(f.Name, site)
		parts := strings.Split(relative, "/")
		if len(parts) != 2 || !strings.HasSuffix(parts[0], ".dist-info") || parts[1] != "METADATA" && parts[1] != "WHEEL" {
			continue
		}
		if len(f.Body) == 0 {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		control := controls[parts[0]]
		if control == nil {
			control = &artifactpypi.UVInstalledWheelControls{DistInfo: parts[0]}
			controls[parts[0]] = control
		}
		if parts[1] == "METADATA" {
			control.Metadata = bytes.Clone(f.Body)
		} else {
			control.Wheel = bytes.Clone(f.Body)
		}
	}
	complete := make([]artifactpypi.UVInstalledWheelControls, 0, len(controls))
	for _, control := range controls {
		complete = append(complete, *control)
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	if _, err := frozen.lock.SelectInstalledWheelControls(complete, target); err != nil {
		return artifactpypi.ConsoleExecutionEntry{}, err
	}
	expected := map[string]bool{}
	metadataDirs := map[string]bool{}
	consoleNames := map[string]bool{}
	for _, info := range inspections {
		var directory string
		for _, file := range info.Files {
			if strings.HasSuffix(file.Path, ".dist-info/METADATA") {
				directory = path.Dir(file.Path)
			}
		}
		if directory == "" {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		metadataDirs[directory] = true
		for _, ep := range info.Surface.EntryPointDetails {
			if ep.Group == "console_scripts" {
				consoleNames[ep.Name] = true
			}
		}
		for _, file := range info.Surface.InstalledFiles {
			if file.Scheme != artifactpypi.SchemeSite && file.Scheme != artifactpypi.SchemeDistInfo {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
			name := site + file.Destination
			f, ok := byName[name]
			if !ok || expected[name] || !regular(f) {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
			expected[name] = true
			if file.Destination == directory+"/RECORD" {
				continue
			} // Native generated DATA, never ownership.
			if f.Size != file.Size || f.SHA256 != file.SHA256 {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
		}
	}
	for _, f := range facts {
		if !strings.HasPrefix(f.Name, site) || f.Kind == tar.TypeDir || expected[f.Name] {
			continue
		}
		relative := strings.TrimPrefix(f.Name, site)
		directory := path.Dir(relative)
		base := path.Base(relative)
		if !metadataDirs[directory] {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
		switch base {
		case "INSTALLER":
			sum := sha256.Sum256([]byte("uv"))
			if f.Size != 2 || f.SHA256 != hex.EncodeToString(sum[:]) {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
		case "REQUESTED":
			sum := sha256.Sum256(nil)
			if f.Size != 0 || f.SHA256 != hex.EncodeToString(sum[:]) {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			}
		case "uv_cache.json":
			if f.Size < 1 || f.Size > 4096 {
				return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
			} // Retained unapproved native cache DATA, never source authority.
		default:
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
	}
	scaffold := map[string]bool{"activate": true, "activate.bat": true, "activate.fish": true, "activate.nu": true, "activate.ps1": true, "activate.xsh": true, "activate_this.py": true, "deactivate.bat": true, "pydoc.bat": true}
	if pipx {
		scaffold["activate.csh"] = true
	}
	for _, f := range facts {
		if !strings.HasPrefix(f.Name, env+"/") || f.Kind == tar.TypeDir || f.Kind == tar.TypeSymlink || strings.HasPrefix(f.Name, site) {
			continue
		}
		relative := strings.TrimPrefix(f.Name, env+"/")
		if pipx && (relative == "pipx_metadata.json" || relative == ".lock" && f.Size == 0) {
			continue
		}
		if relative == "pyvenv.cfg" || relative == ".gitignore" || relative == "CACHEDIR.TAG" {
			continue
		}
		if path.Dir(relative) != "bin" || !scaffold[path.Base(relative)] && !consoleNames[path.Base(relative)] {
			return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
		}
	}
	script, ok := byName[env+"/bin/"+entry.Command()]
	if !ok || !regular(script) || script.Mode != 0755 || len(script.Body) == 0 || int64(len(script.Body)) != script.Size {
		return artifactpypi.ConsoleExecutionEntry{}, errUVRunConsoleData
	}
	if pipx {
		err = artifactpypi.ValidateUVNativeConsoleScript(entry.EntryPoint(), pythonSitePath+"/"+env+"/bin/python", script.Body)
	} else {
		err = artifactpypi.ValidateUVRelocatableConsoleScript(entry.EntryPoint(), script.Body)
	}
	if err != nil {
		return artifactpypi.ConsoleExecutionEntry{}, err
	}
	return entry, nil
}
