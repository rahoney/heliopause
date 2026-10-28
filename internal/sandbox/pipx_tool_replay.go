package sandbox

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

// introducePipxToolUpgradeInput transports controller-generated inert metadata
// only. Its container must already belong to the caller's native session.
func introducePipxToolUpgradeInput(ctx context.Context, runner inputCommandRunner, container string, input artifactpypi.PipxToolUpgradeInput, controls []artifactpypi.UVInstalledWheelControls) error {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) {
		return errors.New("pipx original replay transport unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	body, err := pipxToolUpgradeReplayArchive(input, controls)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return runner.RunInput(ctx, bytes.NewReader(body), "docker", "cp", "--archive", "-", container+":"+pythonSitePath)
}

// No original RECORD, package module, startup hook, command implementation or
// interpreter bytes enter this view. It is not completed native output.
func pipxToolUpgradeReplayArchive(input artifactpypi.PipxToolUpgradeInput, controls []artifactpypi.UVInstalledWheelControls) ([]byte, error) {
	runtime := PinnedPythonRuntime()
	if input.Original().Python() != runtime.PythonVersion {
		return nil, errors.New("pipx original requires pinned Python")
	}
	checked, err := input.ReplayWheelControls(controls, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		return nil, err
	}
	project := input.Receipt().Requirement().Project()
	env := "pipx/venvs/" + project
	site := env + "/lib/python3.14/site-packages"
	receipt, err := input.Receipt().WithPipxLayout(pipxNativeHome+"/venvs/"+project, pipxNativeBin, "/usr/local/bin/python")
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
		return nil, errors.New("pipx original main distribution absent")
	}
	files := map[string]uvToolReplayMember{}
	add := func(name string, body []byte, link string, mode int64) error {
		if !validClosureDestination(name) || files[name].mode != 0 {
			return errors.New("pipx generated original path collides or is invalid")
		}
		files[name] = uvToolReplayMember{body: body, link: link, mode: mode}
		return nil
	}
	for _, file := range []struct {
		name, body string
		mode       int64
	}{
		{"pipx/.cache/CACHEDIR.TAG", "Signature: 8a477f597d28d172789f06886806bc55\n", 0644},
		{"pipx/py/CACHEDIR.TAG", "Signature: 8a477f597d28d172789f06886806bc55\n", 0644},
		{env + "/.gitignore", "*\n", 0644}, {env + "/CACHEDIR.TAG", "Signature: 8a477f597d28d172789f06886806bc55\n", 0644},
		{env + "/.lock", "", 0666}, {"pipx/venvs/." + project + ".lock", "", 0644},
		{env + "/pyvenv.cfg", uvToolVenvConfiguration(), 0644}, {env + "/pipx_metadata.json", string(receipt.Body()), 0600},
	} {
		if err := add(file.name, []byte(file.body), "", file.mode); err != nil {
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
		if err := add("tool-bin/"+command.Name, nil, pipxNativeHome+"/venvs/"+project+"/bin/"+command.Name, 0777); err != nil {
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
		for _, file := range []struct {
			name string
			body []byte
		}{{control.DistInfo + "/METADATA", control.Metadata}, {control.DistInfo + "/WHEEL", control.Wheel}} {
			if err := writeUVToolReplayRecord(writer, file.name, file.body); err != nil {
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
	directories := map[string]bool{"pipx": true, "pipx/.cache": true, "pipx/py": true, "pipx/logs": true, "pipx/venvs": true, env: true, env + "/bin": true, env + "/lib": true, env + "/lib/python3.14": true, site: true, "tool-bin": true, "tool-man": true}
	return serializeToolReplayArchive(files, directories)
}
