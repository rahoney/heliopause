package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type pipxNativeOutput struct {
	receipt  artifactpypi.ToolReceipt
	controls []artifactpypi.UVInstalledWheelControls
	links    map[string]string
	graph    domain.LockedDependencyGraph
}

var errPipxNativeOutput = errors.New("pipx complete native output is absent, changed or outside the reviewed layout")
var pipxNativeLogName = regexp.MustCompile(`^cmd_[0-9]{4}-[0-9]{2}-[0-9]{2}_[0-9]{2}\.[0-9]{2}\.[0-9]{2}(?:_[0-9]{1,12})?(?:_[a-z]+_errors)?\.log$`)

// Both transport and native contents are unapproved data. This does not grant
// execution admission, Policy approval, target ownership or payload promotion.
func readPipxNativeOutput(ctx context.Context, runner CommandRunner, container, project string, frozen *pipxNativeInput) (pipxNativeOutput, error) {
	if !frozen.valid() {
		return pipxNativeOutput{}, errPipxNativeInput
	}
	var result pipxNativeOutput
	_, err := readUVNativeTar(ctx, runner, container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		var err error
		result, err = scanPipxNativeOutput(input, project)
		return uvNativeOutput{}, err
	})
	if err != nil {
		return pipxNativeOutput{}, err
	}
	graph, err := frozen.bindNativeOutput(result)
	if err != nil {
		return pipxNativeOutput{}, err
	}
	result.graph = graph
	return result, nil
}

func scanPipxNativeOutput(input io.Reader, project string) (pipxNativeOutput, error) {
	canonical, err := artifactpypi.NormalizeProjectName(project)
	if input == nil || err != nil || canonical != project {
		return pipxNativeOutput{}, errPipxNativeOutput
	}
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	maxEntries := policy.MaxGraphFiles()*8 + int64(policy.MaxGraphArtifacts())*4 + 128
	bounded := &io.LimitedReader{R: input, N: policy.MaxGraphUncompressed() + 3*uvNativeLockLimit + maxEntries*1024 + 10240}
	archive := tar.NewReader(bounded)
	env := "pipx/venvs/" + project
	site := env + "/lib/python3.14/site-packages"
	directories := map[string]bool{}
	fixedDirectories := map[string]bool{"pipx": true, "pipx/.cache": true, "pipx/py": true, "pipx/logs": true, "pipx/venvs": true, env: true, env + "/bin": true, env + "/lib": true, env + "/lib/python3.14": true, site: true, "tool-bin": true, "tool-man": true}
	auxiliary := map[string]bool{"pipx/.cache/CACHEDIR.TAG": true, "pipx/py/CACHEDIR.TAG": true, env + "/.gitignore": true, env + "/CACHEDIR.TAG": true, env + "/.lock": true, "pipx/venvs/." + project + ".lock": true}
	for _, name := range []string{"activate", "activate.bat", "activate.csh", "activate.fish", "activate.nu", "activate.ps1", "activate.xsh", "activate_this.py", "deactivate.bat", "pydoc.bat"} {
		auxiliary[env+"/bin/"+name] = true
	}
	aliases := map[string]string{env + "/bin/python": "/usr/local/bin/python", env + "/bin/python3": "python", env + "/bin/python3.14": "python", env + "/lib64": "lib"}
	seen := map[string]bool{}
	scripts := map[string]bool{}
	links := map[string]string{}
	controls := map[string]*artifactpypi.UVInstalledWheelControls{}
	var receipt []byte
	var entries, expanded, controlBytes, metadataBytes, logBytes int64
	var logs int
	rootSeen := false
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
		entries++
		name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
		root := name == "" || name == "."
		link := h.Typeflag == tar.TypeSymlink
		uvLock := name == env+"/.lock" && h.Size == 0 && h.Mode == 0666 && h.Typeflag == tar.TypeReg
		if entries > maxEntries || seen[name] || h.Uid != 1000 || h.Gid != 1000 || h.Mode&^int64(0777) != 0 || h.Size < 0 || !root && !validClosureDestination(name) || link && h.Mode != 0777 || !link && h.Mode&0022 != 0 && !uvLock || !link && h.Linkname != "" {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
		seen[name] = true
		if h.Typeflag == tar.TypeDir {
			if h.Size != 0 || root && rootSeen || !root && !fixedDirectories[name] && name != "tool-completions" && !strings.HasPrefix(name, site+"/") {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
			if root {
				rootSeen = true
			}
			directories[name] = true
			continue
		}
		if root || strings.HasSuffix(h.Name, "/") {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
		if link {
			if h.Size != 0 {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
			if target, ok := aliases[name]; ok {
				if h.Linkname != target {
					return pipxNativeOutput{}, errPipxNativeOutput
				}
			} else if path.Dir(name) == "tool-bin" {
				links[pythonSitePath+"/"+name] = h.Linkname
			} else {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
			continue
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != 0 || h.Size > policy.MaxGraphUncompressed()-expanded {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
		expanded += h.Size
		if _, ok := aliases[name]; ok {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
		rel := strings.TrimPrefix(name, site+"/")
		parts := strings.Split(rel, "/")
		metadata := strings.HasPrefix(name, site+"/") && len(parts) == 2 && strings.HasSuffix(parts[0], ".dist-info") && (parts[1] == "METADATA" || parts[1] == "WHEEL")
		switch {
		case metadata, name == env+"/pipx_metadata.json", name == env+"/pyvenv.cfg":
			limit := int64(uvNativeLockLimit)
			if metadata {
				limit = policy.WheelLimits().MaxMetadata
				if h.Size > uvNativeLockLimit-metadataBytes {
					return pipxNativeOutput{}, errPipxNativeOutput
				}
				metadataBytes += h.Size
			}
			if h.Size == 0 || h.Size > limit || h.Size > 3*uvNativeLockLimit-controlBytes {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
			controlBytes += h.Size
			body, err := io.ReadAll(io.LimitReader(archive, h.Size))
			if err != nil || int64(len(body)) != h.Size {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
			switch {
			case metadata:
				c := controls[parts[0]]
				if c == nil {
					if len(controls) >= policy.MaxGraphArtifacts() {
						return pipxNativeOutput{}, errPipxNativeOutput
					}
					c = &artifactpypi.UVInstalledWheelControls{DistInfo: parts[0]}
					controls[parts[0]] = c
				}
				if parts[1] == "METADATA" {
					c.Metadata = body
				} else {
					c.Wheel = body
				}
			case name == env+"/pipx_metadata.json":
				receipt = body
			default:
				if string(body) != uvToolVenvConfiguration() {
					return pipxNativeOutput{}, errPipxNativeOutput
				}
			}
		default:
			log := path.Dir(name) == "pipx/logs" && len(path.Base(name)) <= 100 && pipxNativeLogName.MatchString(path.Base(name))
			allowed := strings.HasPrefix(name, site+"/") || path.Dir(name) == env+"/bin" || auxiliary[name] || log
			if !allowed {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
			if log {
				logs++
				if logs > 100 || h.Size > uvNativeLockLimit-logBytes {
					return pipxNativeOutput{}, errPipxNativeOutput
				}
				logBytes += h.Size
			}
			if auxiliary[name] && h.Size > uvNativeLockLimit {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
			if path.Dir(name) == env+"/bin" && !auxiliary[name] {
				scripts[name] = true
			}
			if _, err := io.CopyN(io.Discard, archive, h.Size); err != nil {
				return pipxNativeOutput{}, errPipxNativeOutput
			}
		}
	}
	if err := drainUVTarPadding(bounded); err != nil {
		return pipxNativeOutput{}, err
	}
	if !rootSeen || bounded.N == 0 || len(receipt) == 0 || len(controls) == 0 || logs == 0 {
		return pipxNativeOutput{}, errPipxNativeOutput
	}
	for name := range fixedDirectories {
		if !directories[name] {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
	}
	for name := range aliases {
		if !seen[name] {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
	}
	for name := range auxiliary {
		if !seen[name] {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
	}
	if !seen[env+"/pyvenv.cfg"] {
		return pipxNativeOutput{}, errPipxNativeOutput
	}
	for name := range seen {
		if name == "" || name == "." {
			continue
		}
		parent := path.Dir(name)
		if parent != "." && !directories[parent] {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
	}
	r, err := artifactpypi.ParsePipxToolReceipt(receipt)
	if err != nil || r.Requirement().Project() != project {
		return pipxNativeOutput{}, errPipxNativeOutput
	}
	if err := r.ValidateExposedCommands(pipxNativeHome+"/venvs/"+project, pipxNativeBin, "/usr/local/bin/python", links); err != nil {
		return pipxNativeOutput{}, err
	}
	if len(scripts) != len(r.Commands()) {
		return pipxNativeOutput{}, errPipxNativeOutput
	}
	for _, command := range r.Commands() {
		if !scripts[env+"/bin/"+command.Name] {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
	}
	result := pipxNativeOutput{receipt: r, links: links}
	for _, c := range controls {
		if len(c.Metadata) == 0 || len(c.Wheel) == 0 {
			return pipxNativeOutput{}, errPipxNativeOutput
		}
		result.controls = append(result.controls, *c)
	}
	sort.Slice(result.controls, func(i, j int) bool { return result.controls[i].DistInfo < result.controls[j].DistInfo })
	return result, nil
}

func (p *pipxNativeInput) bindNativeOutput(output pipxNativeOutput) (domain.LockedDependencyGraph, error) {
	if !p.valid() || output.receipt.Client() != "pipx" || len(output.controls) != len(p.files) {
		return domain.LockedDependencyGraph{}, errPipxNativeOutput
	}
	runtime := PinnedPythonRuntime()
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	limits := artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits()
	var inputs []artifactpypi.UVWheelInput
	expected := map[string]artifactpypi.WheelFile{}
	for _, file := range p.files {
		inspection, err := artifactpypi.InspectWheel(bytes.NewReader(file.body), int64(len(file.body)), file.name, file.digest, target, limits)
		if err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		for _, entry := range inspection.Files {
			if strings.HasSuffix(entry.Path, ".dist-info/METADATA") || strings.HasSuffix(entry.Path, ".dist-info/WHEEL") {
				expected[entry.Path] = entry
			}
		}
		inputs = append(inputs, artifactpypi.UVWheelInput{URL: file.url, Reader: bytes.NewReader(file.body), Size: int64(len(file.body))})
	}
	seen := map[string]bool{}
	var metadata []artifactpypi.InstalledMetadataFile
	for _, control := range output.controls {
		if seen[control.DistInfo] {
			return domain.LockedDependencyGraph{}, errPipxNativeOutput
		}
		seen[control.DistInfo] = true
		for name, body := range map[string][]byte{"METADATA": control.Metadata, "WHEEL": control.Wheel} {
			file, ok := expected[control.DistInfo+"/"+name]
			sum := sha256.Sum256(body)
			if !ok || int64(len(body)) != file.Size || hex.EncodeToString(sum[:]) != file.SHA256 {
				return domain.LockedDependencyGraph{}, errPipxNativeOutput
			}
		}
		metadata = append(metadata, artifactpypi.InstalledMetadataFile{Directory: control.DistInfo, Body: control.Metadata})
	}
	graph, err := output.receipt.BindWheels(p.lock, inputs, p.pages, target, limits)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	original, err := artifactpypi.NewPipxToolUpgradeInput(output.receipt, metadata, runtime.PythonVersion)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	if _, err := original.ReplayWheelControls(output.controls, target); err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	return graph, nil
}
