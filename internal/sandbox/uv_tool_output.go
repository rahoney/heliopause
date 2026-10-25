package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const uvNativeToolHome = pythonSitePath + "/tools"
const uvNativeToolBin = pythonSitePath + "/tool-bin"

type uvToolNativeOutput struct {
	receipt  artifactpypi.ToolReceipt
	controls []artifactpypi.UVInstalledWheelControls
	links    map[string]string
	lock     []byte
}

func readUVToolNativeOutput(ctx context.Context, runner CommandRunner, container, project string, lock []byte) (uvToolNativeOutput, error) {
	var result uvToolNativeOutput
	_, err := readUVNativeTar(ctx, runner, container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
		var err error
		result, err = scanUVToolNativeOutput(input, project, lock)
		return uvNativeOutput{}, err
	})
	if err != nil {
		return uvToolNativeOutput{}, err
	}
	return result, nil
}

// Native files are unapproved selection data. Only fixed runtime aliases and
// receipt-bound exposed links are supported; no script or interpreter is run.
func scanUVToolNativeOutput(input io.Reader, project string, expectedLock []byte) (uvToolNativeOutput, error) {
	canonical, err := artifactpypi.NormalizeProjectName(project)
	if input == nil || err != nil || canonical != project {
		return uvToolNativeOutput{}, errors.New("UV tool output requires one canonical root")
	}
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	maxEntries := policy.MaxGraphFiles()*8 + int64(policy.MaxGraphArtifacts())*4 + 128
	bounded := &io.LimitedReader{R: input, N: policy.MaxGraphUncompressed() + 3*uvNativeLockLimit + maxEntries*1024 + 10240}
	archive := tar.NewReader(bounded)
	env := "tools/" + project
	site := env + "/lib/python3.14/site-packages"
	seen := map[string]bool{}
	controls := map[string]*artifactpypi.UVInstalledWheelControls{}
	links := map[string]string{}
	aliases := map[string]bool{}
	scripts := map[string]bool{}
	var receipt, lock []byte
	var count, expanded, controlBytes, metadataBytes int64
	fixedDirectories := map[string]bool{"tools": true, "tool-bin": true, env: true, env + "/bin": true, env + "/lib": true, env + "/lib/python3.14": true, site: true}
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return uvToolNativeOutput{}, errors.New("UV tool tar is invalid or truncated")
		}
		count++
		name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
		root := name == "" || name == "."
		auxiliary := (name == "tools/.lock" || name == env+"/.lock") && h.Size == 0 && h.Mode == 0666 && (h.Typeflag == tar.TypeReg || h.Typeflag == 0)
		link := h.Typeflag == tar.TypeSymlink
		if count > maxEntries || h.Uid != 1000 || h.Gid != 1000 || seen[name] || h.Size < 0 || h.Mode&^int64(0777) != 0 || (!root && !validClosureDestination(name)) || h.Mode&0022 != 0 && !auxiliary && !link || link && h.Mode != 0777 {
			return uvToolNativeOutput{}, errors.New("UV tool path, owner, mode or count is invalid")
		}
		seen[name] = true
		if h.Typeflag == tar.TypeDir {
			if h.Size != 0 || !root && !fixedDirectories[name] && !strings.HasPrefix(name, site+"/") {
				return uvToolNativeOutput{}, errors.New("UV tool directory is unsupported")
			}
			continue
		}
		if root {
			return uvToolNativeOutput{}, errors.New("UV tool root is not a directory")
		}
		if link {
			if h.Size != 0 {
				return uvToolNativeOutput{}, errors.New("UV tool link has payload")
			}
			expected := map[string]string{env + "/bin/python": "/usr/local/bin/python", env + "/bin/python3": "python", env + "/bin/python3.14": "python", env + "/lib64": "lib"}
			if target, ok := expected[name]; ok {
				if h.Linkname != target {
					return uvToolNativeOutput{}, errors.New("UV tool interpreter alias differs")
				}
				aliases[name] = true
			} else if path.Dir(name) == "tool-bin" {
				links[pythonSitePath+"/"+name] = h.Linkname
			} else {
				return uvToolNativeOutput{}, errors.New("UV tool link is outside the reviewed layout")
			}
			continue
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != 0 || h.Size > policy.MaxGraphUncompressed()-expanded {
			return uvToolNativeOutput{}, errors.New("UV tool member type or expansion is unsupported")
		}
		expanded += h.Size
		rel := strings.TrimPrefix(name, site+"/")
		parts := strings.Split(rel, "/")
		metadata := strings.HasPrefix(name, site+"/") && len(parts) == 2 && strings.HasSuffix(parts[0], ".dist-info") && (parts[1] == "METADATA" || parts[1] == "WHEEL")
		switch {
		case name == env+"/uv-receipt.toml", name == env+"/pyvenv.cfg", name == "pylock.toml", metadata:
			limit := int64(uvNativeLockLimit)
			if metadata {
				limit = policy.WheelLimits().MaxMetadata
				if metadataBytes > uvNativeLockLimit-h.Size {
					return uvToolNativeOutput{}, errors.New("UV tool metadata aggregate exceeds bounds")
				}
				metadataBytes += h.Size
			}
			if h.Size == 0 || h.Size > limit || controlBytes > 3*uvNativeLockLimit-h.Size {
				return uvToolNativeOutput{}, errors.New("UV tool control size exceeds bounds")
			}
			controlBytes += h.Size
			body, err := io.ReadAll(io.LimitReader(archive, h.Size))
			if err != nil || int64(len(body)) != h.Size {
				return uvToolNativeOutput{}, errors.New("UV tool control is truncated")
			}
			switch {
			case metadata:
				c := controls[parts[0]]
				if c == nil {
					if len(controls) >= policy.MaxGraphArtifacts() {
						return uvToolNativeOutput{}, errors.New("UV tool distribution count exceeds bounds")
					}
					c = &artifactpypi.UVInstalledWheelControls{DistInfo: parts[0]}
					controls[parts[0]] = c
				}
				if parts[1] == "METADATA" {
					c.Metadata = body
				} else {
					c.Wheel = body
				}
			case name == env+"/uv-receipt.toml":
				receipt = body
			case name == "pylock.toml":
				if len(expectedLock) == 0 || !bytes.Equal(body, expectedLock) {
					return uvToolNativeOutput{}, errors.New("UV tool frozen export changed")
				}
				lock = body
			default:
				expected := fmt.Sprintf("home = /usr/local/bin\nimplementation = CPython\nuv = %s\nversion_info = %s\ninclude-system-site-packages = false\n", runtimeidentity.UVVersion, runtimeidentity.PythonVersion)
				if string(body) != expected {
					return uvToolNativeOutput{}, errors.New("UV tool environment differs from pinned Python")
				}
			}
		default:
			allowed := strings.HasPrefix(name, site+"/") || path.Dir(name) == env+"/bin" || name == "tools/.gitignore" || name == env+"/.gitignore" || name == env+"/CACHEDIR.TAG" || auxiliary
			if !allowed || path.Dir(name) == "tool-bin" {
				return uvToolNativeOutput{}, errors.New("UV tool file is outside the reviewed layout")
			}
			if path.Dir(name) == env+"/bin" {
				scripts[name] = true
			}
			if _, err := io.CopyN(io.Discard, archive, h.Size); err != nil {
				return uvToolNativeOutput{}, errors.New("UV tool payload is truncated")
			}
		}
	}
	if err := drainUVTarPadding(bounded); err != nil {
		return uvToolNativeOutput{}, err
	}
	if bounded.N == 0 || len(controls) == 0 || len(receipt) == 0 || len(expectedLock) > 0 && len(lock) == 0 {
		return uvToolNativeOutput{}, errors.New("UV tool output is absent or exceeds bounds")
	}
	for _, name := range []string{"tools/.gitignore", "tools/.lock", env + "/.gitignore", env + "/CACHEDIR.TAG", env + "/pyvenv.cfg", env + "/bin/python", env + "/bin/python3", env + "/bin/python3.14", site} {
		if !seen[name] {
			return uvToolNativeOutput{}, errors.New("UV tool runtime scaffold is incomplete")
		}
	}
	for name := range fixedDirectories {
		if !seen[name] {
			return uvToolNativeOutput{}, errors.New("UV tool directory scaffold is incomplete")
		}
	}
	for _, name := range []string{env + "/bin/python", env + "/bin/python3", env + "/bin/python3.14"} {
		if !aliases[name] {
			return uvToolNativeOutput{}, errors.New("UV tool runtime alias is absent or replaced")
		}
	}
	r, err := artifactpypi.ParseUVToolReceipt(receipt)
	if err != nil || r.Requirement().Project() != project {
		return uvToolNativeOutput{}, errors.New("UV tool receipt is unsupported or has a foreign root")
	}
	if err := r.ValidateExposedCommands(uvNativeToolHome+"/"+project, uvNativeToolBin, "/usr/local/bin/python", links); err != nil {
		return uvToolNativeOutput{}, err
	}
	for _, command := range r.Commands() {
		if !scripts[env+"/bin/"+command.Name] {
			return uvToolNativeOutput{}, errors.New("UV tool exposed command has no native script")
		}
	}
	result := uvToolNativeOutput{receipt: r, links: links, lock: lock}
	for _, c := range controls {
		if len(c.Metadata) == 0 || len(c.Wheel) == 0 {
			return uvToolNativeOutput{}, errors.New("UV tool installed metadata is incomplete")
		}
		result.controls = append(result.controls, *c)
	}
	sort.Slice(result.controls, func(i, j int) bool { return result.controls[i].DistInfo < result.controls[j].DistInfo })
	return result, nil
}
