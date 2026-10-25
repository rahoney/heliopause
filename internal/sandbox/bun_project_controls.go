package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunControls = errors.New("bun fixed project controls are unavailable or changed")

func copyBunProjectControls(ctx context.Context, runner CommandRunner, container string, controls []domain.ProjectControlFile) error {
	if err := copyNativeProjectControls(ctx, runner, container, nativeRuntimeBun, controls); err != nil {
		return errors.Join(errBunControls, err)
	}
	return nil
}

// Read one complete bounded directory stream. A failed Docker read never
// proves lock absence; only a complete project census can do that.
func readBunProjectControls(ctx context.Context, runner streamingDockerOutput, container string) ([]domain.ProjectControlFile, error) {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) {
		return nil, errBunControls
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errBunControls, err)
	}
	output := nativeToolArchiveOutput{limit: 2*pnpmControlLimit + 64<<10}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+bunProjectDirectory, "-"); err != nil || output.failed || ctx.Err() != nil {
		return nil, errors.Join(errBunControls, err, ctx.Err())
	}
	return parseBunProjectControls(ctx, output.body)
}

func parseBunProjectControls(ctx context.Context, body []byte) ([]domain.ProjectControlFile, error) {
	if ctx == nil || int64(len(body)) > 2*pnpmControlLimit+64<<10 {
		return nil, errBunControls
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errBunControls, err)
	}
	input := bytes.NewReader(body)
	archive := tar.NewReader(input)
	header, err := archive.Next()
	if err != nil || header.Typeflag != tar.TypeDir || (header.Name != "project/" && header.Name != "project") || header.Linkname != "" || header.Uid != 1000 || header.Gid != 1000 || header.Mode != 0700 || header.Size != 0 {
		return nil, errBunControls
	}
	values := map[string][]byte{}
	for count := 0; ; count++ {
		header, err = archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil || count >= 2 {
			return nil, errors.Join(errBunControls, err)
		}
		name := ""
		switch header.Name {
		case "project/package.json":
			name = "package.json"
		case "project/bun.lock":
			name = "bun.lock"
		default:
			return nil, errBunControls
		}
		if _, seen := values[name]; seen || header.Typeflag != tar.TypeReg || header.Linkname != "" || header.Uid != 1000 || header.Gid != 1000 || (header.Mode != 0600 && header.Mode != 0644) || header.Size <= 0 || header.Size > pnpmControlLimit {
			return nil, errBunControls
		}
		value, err := io.ReadAll(archive)
		if err != nil || int64(len(value)) != header.Size || ctx.Err() != nil {
			return nil, errors.Join(errBunControls, err, ctx.Err())
		}
		values[name] = value
	}
	for _, value := range body[len(body)-input.Len():] {
		if value != 0 {
			return nil, errBunControls
		}
	}
	if len(values["package.json"]) == 0 {
		return nil, errBunControls
	}
	result := make([]domain.ProjectControlFile, 0, 2)
	for _, name := range []string{"package.json", "bun.lock"} {
		value, present := values[name]
		control, err := domain.NewProjectControlFile(name, value, present)
		if err != nil {
			return nil, err
		}
		result = append(result, control)
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errBunControls, err)
	}
	return result, nil
}
