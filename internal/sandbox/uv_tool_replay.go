package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"path"
	"sort"
	"strconv"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

const uvToolReplayCommand = "# HAA metadata-only command placeholder; never execute.\n"

type uvToolReplayMember struct {
	body []byte
	link string
	mode int64
}

// Only controller-created metadata/scaffold is introduced. No original RECORD,
// package module, startup file, interpreter or executable command is transported.
func introduceUVToolUpgradeInput(ctx context.Context, runner inputCommandRunner, container string, input artifactpypi.UVToolUpgradeInput, controls []artifactpypi.UVInstalledWheelControls) error {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) {
		return errors.New("tool original replay transport is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	body, err := uvToolUpgradeReplayArchive(input, controls)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return runner.RunInput(ctx, bytes.NewReader(body), "docker", "cp", "--archive", "-", container+":"+pythonSitePath)
}

func uvToolUpgradeReplayArchive(input artifactpypi.UVToolUpgradeInput, controls []artifactpypi.UVInstalledWheelControls) ([]byte, error) {
	runtime := PinnedPythonRuntime()
	if input.Original().Python() != runtime.PythonVersion {
		return nil, errors.New("tool replay requires the pinned Python version")
	}
	checked, err := input.ReplayWheelControls(controls, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		return nil, err
	}
	project := input.Receipt().Requirement().Project()
	env := "tools/" + project
	site := env + "/lib/python3.14/site-packages"
	receipt, err := input.Receipt().WithUVSelectionLayout(uvNativeToolHome+"/"+project, uvNativeToolBin)
	if err != nil {
		return nil, err
	}
	rootDirectory := ""
	for _, installed := range input.Original().Installed() {
		if installed.Project() == project {
			rootDirectory = installed.Directory()
		}
	}
	if rootDirectory == "" {
		return nil, errors.New("tool replay main distribution is absent")
	}
	files := map[string]uvToolReplayMember{}
	add := func(name string, body []byte, link string, mode int64) error {
		if !validClosureDestination(name) || files[name].mode != 0 {
			return errors.New("tool replay generated path collides or is invalid")
		}
		files[name] = uvToolReplayMember{body: body, link: link, mode: mode}
		return nil
	}
	for _, value := range []struct{ name, body string }{{"tools/.gitignore", "*\n"}, {"tools/.lock", ""}, {env + "/.gitignore", "*\n"}, {env + "/CACHEDIR.TAG", "Signature: 8a477f597d28d172789f06886806bc55\n"}, {env + "/pyvenv.cfg", uvToolVenvConfiguration()}, {env + "/uv-receipt.toml", string(receipt.Body())}} {
		mode := int64(0600)
		if value.name == "tools/.lock" {
			mode = 0666
		} // Exact pinned native auxiliary, inside private0700 tools.
		if err := add(value.name, []byte(value.body), "", mode); err != nil {
			return nil, err
		}
	}
	for name, target := range map[string]string{env + "/bin/python": "/usr/local/bin/python", env + "/bin/python3": "python", env + "/bin/python3.14": "python", env + "/lib64": "lib"} {
		if err := add(name, nil, target, 0777); err != nil {
			return nil, err
		}
	}
	for _, command := range receipt.Commands() {
		if err := add(env+"/bin/"+command.Name, []byte(uvToolReplayCommand), "", 0400); err != nil {
			return nil, err
		}
		if err := add("tool-bin/"+command.Name, nil, uvNativeToolHome+"/"+project+"/bin/"+command.Name, 0777); err != nil {
			return nil, err
		}
	}
	for _, control := range checked {
		directory := site + "/" + control.DistInfo
		if err := add(directory+"/METADATA", control.Metadata, "", 0400); err != nil {
			return nil, err
		}
		if err := add(directory+"/WHEEL", control.Wheel, "", 0400); err != nil {
			return nil, err
		}
		var record bytes.Buffer
		writer := csv.NewWriter(&record)
		for _, value := range []struct {
			name string
			body []byte
		}{{control.DistInfo + "/METADATA", control.Metadata}, {control.DistInfo + "/WHEEL", control.Wheel}} {
			if err := writeUVToolReplayRecord(writer, value.name, value.body); err != nil {
				return nil, err
			}
		}
		if control.DistInfo == rootDirectory {
			for _, command := range receipt.Commands() {
				if err := writeUVToolReplayRecord(writer, "../../../bin/"+command.Name, []byte(uvToolReplayCommand)); err != nil {
					return nil, err
				}
			}
		}
		if err := writer.Write([]string{control.DistInfo + "/RECORD", "", ""}); err != nil {
			return nil, err
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, err
		}
		if err := add(directory+"/RECORD", record.Bytes(), "", 0400); err != nil {
			return nil, err
		}
	}
	directories := map[string]bool{"tools": true, "tool-bin": true, env: true, env + "/bin": true, env + "/lib": true, env + "/lib/python3.14": true, site: true}
	return serializeToolReplayArchive(files, directories)
}

func serializeToolReplayArchive(files map[string]uvToolReplayMember, directories map[string]bool) ([]byte, error) {
	var names, dirs []string
	for name := range files {
		names = append(names, name)
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			directories[parent] = true
		}
	}
	for name := range directories {
		if _, exists := files[name]; exists {
			return nil, errors.New("tool replay file overlaps a generated directory")
		}
		dirs = append(dirs, name)
	}
	if len(names)+len(dirs) > artifactpypi.UpgradeInputDistributionLimit*4+1024 {
		return nil, errors.New("tool replay member count exceeds bounds")
	}
	sort.Strings(names)
	sort.Strings(dirs)
	var output bytes.Buffer
	archive := tar.NewWriter(&output)
	for _, name := range dirs {
		if err := archive.WriteHeader(&tar.Header{Name: name + "/", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
			return nil, err
		}
	}
	for _, name := range names {
		member := files[name]
		header := &tar.Header{Name: name, Mode: member.mode, Uid: 1000, Gid: 1000, Typeflag: tar.TypeReg, Size: int64(len(member.body))}
		if member.link != "" {
			header.Typeflag = tar.TypeSymlink
			header.Linkname = member.link
			header.Size = 0
		}
		if err := archive.WriteHeader(header); err != nil {
			return nil, err
		}
		if len(member.body) > 0 {
			if _, err := archive.Write(member.body); err != nil {
				return nil, err
			}
		}
		if output.Len() > 2*uvNativeLockLimit {
			return nil, errors.New("tool replay archive exceeds byte bounds")
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	if output.Len() > 2*uvNativeLockLimit {
		return nil, errors.New("tool replay archive exceeds byte bounds")
	}
	return output.Bytes(), nil
}

func writeUVToolReplayRecord(writer *csv.Writer, name string, body []byte) error {
	// Names here are generated from validated dist-info/command basenames, never
	// from an original RECORD or a caller-provided relative path.
	if strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("generated tool replay RECORD path is invalid")
	}
	sum := sha256.Sum256(body)
	return writer.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), strconv.Itoa(len(body))})
}
