package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

const uvNativeProjectRoot = pythonSitePath + "/project"

func uvProjectControlName(name string) bool {
	return name == artifactpypi.UVProjectInputControl || name == artifactpypi.UVNativeLockControl || name == artifactpypi.UVProjectExportControl
}

func uvProjectFileName(control string) string {
	if control == artifactpypi.UVProjectExportControl {
		return "pylock.toml"
	}
	if uvProjectControlName(control) {
		return control
	}
	return ""
}

func uvProjectControlPath(value string) bool {
	name := strings.TrimPrefix(value, uvNativeProjectRoot+"/")
	return strings.HasPrefix(value, uvNativeProjectRoot+"/") && (name == artifactpypi.UVProjectInputControl || name == artifactpypi.UVNativeLockControl || name == "pylock.toml")
}

func readUVProjectControl(ctx context.Context, runner CommandRunner, container, name string) ([]byte, error) {
	if !uvProjectControlName(name) {
		return nil, errors.New("UV project control name is unsupported")
	}
	file := uvProjectFileName(name)
	result, err := readUVNativeTar(ctx, runner, container, uvNativeProjectRoot+"/"+file, func(input io.Reader) (uvNativeOutput, error) {
		return scanUVNamedControl(input, file)
	})
	return result.lock, err
}

// Only controller-selected inert files are written into the fresh private
// output volume. Frozen project files and the separately derived pylock are
// rechecked by the final existing native census scanner.
func introduceUVProjectFiles(ctx context.Context, runner inputCommandRunner, container string, files map[string][]byte, view []byte) error {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) || len(files) < 1 || len(files) > 3 || len(files[artifactpypi.UVProjectInputControl]) == 0 || len(view) > uvNativeLockLimit {
		return errors.New("UV project input transport is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	frozen := len(view) != 0
	if frozen && (len(files) != 3 || len(files[artifactpypi.UVNativeLockControl]) == 0 || len(files[artifactpypi.UVProjectExportControl]) == 0) || !frozen && files[artifactpypi.UVProjectExportControl] != nil {
		return errors.New("UV project frozen transport controls are incomplete")
	}
	names := make([]string, 0, len(files))
	for name, body := range files {
		if !uvProjectControlName(name) || len(body) == 0 || len(body) > uvNativeLockLimit {
			return errors.New("UV project transport file is unsupported")
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var buffer bytes.Buffer
	archive := tar.NewWriter(&buffer)
	for _, directory := range []string{"project/", "site/"} {
		if err := archive.WriteHeader(&tar.Header{Name: directory, Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
			return err
		}
	}
	write := func(name string, body []byte, mode int64) error {
		if err := archive.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: mode, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
			return err
		}
		_, err := archive.Write(body)
		return err
	}
	mode := int64(0600)
	if frozen {
		mode = 0400
	}
	for _, name := range names {
		if err := write("project/"+uvProjectFileName(name), files[name], mode); err != nil {
			return err
		}
	}
	if frozen {
		if err := write("pylock.toml", view, 0400); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	return runner.RunInput(ctx, bytes.NewReader(buffer.Bytes()), "docker", "cp", "--archive", "-", container+":"+pythonSitePath)
}
