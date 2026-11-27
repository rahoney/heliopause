package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const pnpmControlLimit = 4 << 20

var errPNPMControls = errors.New("pnpm fixed project controls are unavailable or changed")

// Input is data-only and confined to the fixed private project mount. The
// caller must semantically validate the manifest before creating resources.
func copyPNPMProjectControls(ctx context.Context, runner CommandRunner, container string, controls []domain.ProjectControlFile) error {
	return copyNativeProjectControls(ctx, runner, container, nativeRuntimePNPM, controls)
}

func readPNPMProjectControl(ctx context.Context, runner streamingDockerOutput, container, name string) (domain.ProjectControlFile, error) {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) || name != "package.json" && name != "pnpm-lock.yaml" {
		return domain.ProjectControlFile{}, errPNPMControls
	}
	if err := ctx.Err(); err != nil {
		return domain.ProjectControlFile{}, errors.Join(errPNPMControls, err)
	}
	output := nativeToolArchiveOutput{limit: pnpmControlLimit + 64<<10}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+pnpmProjectDirectory+"/"+name, "-"); err != nil || output.failed || ctx.Err() != nil {
		return domain.ProjectControlFile{}, errors.Join(errPNPMControls, err, ctx.Err())
	}
	input := bytes.NewReader(output.body)
	archive := tar.NewReader(input)
	header, err := archive.Next()
	if err != nil {
		return domain.ProjectControlFile{}, errors.Join(errPNPMControls, err)
	}
	if header.Typeflag != tar.TypeReg || header.Name != name || header.Linkname != "" || header.Uid != 1000 || header.Gid != 1000 ||
		header.Mode != 0600 && header.Mode != 0644 || header.Size <= 0 || header.Size > pnpmControlLimit {
		return domain.ProjectControlFile{}, fmt.Errorf("%w: metadata name_matches=%t type=%d uid=%d gid=%d mode=%#o bytes=%d", errPNPMControls, header.Name == name, header.Typeflag, header.Uid, header.Gid, header.Mode, header.Size)
	}
	body, err := io.ReadAll(archive)
	if err != nil || int64(len(body)) != header.Size {
		return domain.ProjectControlFile{}, errors.Join(errPNPMControls, err)
	}
	if _, err := archive.Next(); err != io.EOF {
		return domain.ProjectControlFile{}, errPNPMControls
	}
	for _, value := range output.body[len(output.body)-input.Len():] {
		if value != 0 {
			return domain.ProjectControlFile{}, errPNPMControls
		}
	}
	if err := ctx.Err(); err != nil {
		return domain.ProjectControlFile{}, errors.Join(errPNPMControls, err)
	}
	return domain.NewProjectControlFile(name, body, true)
}
