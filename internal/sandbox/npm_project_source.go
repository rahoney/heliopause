package sandbox

import (
	"archive/tar"
	"context"
	"errors"
	"path/filepath"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// NPMProjectSourceReader consumes local data under the existing anchored source
// transport reader. It creates no runtime and confers no source safety/approval.
type NPMProjectSourceReader struct{ intake string }

func NewNPMProjectSourceReader(intake string) (*NPMProjectSourceReader, error) {
	if !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" {
		return nil, errors.New("npm project source intake must be canonical")
	}
	return &NPMProjectSourceReader{intake}, nil
}

func npmProjectSourceLimits() projectBuildInputLimits {
	return projectBuildInputLimits{variant: "npm-source", maxPath: 1024, files: artifactnpm.ArchiveEntryLimit, payload: artifactnpm.NPMProjectInspectionWorkspaceLimit, file: artifactnpm.ArchiveFileLimit, control: artifactnpm.ProjectControlLimit, controls: [2]string{"package.json", "package-lock.json"}, preserveSourceExecutable: true}
}

func (r *NPMProjectSourceReader) openSource(ctx context.Context, source domain.AcquiredArtifact) (*goBuildInputFile, *goBuildInputManifest, error) {
	if r == nil || ctx == nil {
		return nil, nil, errors.New("npm project source reader unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, errors.Join(err, context.Cause(ctx))
	}
	input, err := openProjectBuildInput(r.intake, source, "project", npmProjectSourceLimits())
	if err != nil {
		return nil, nil, err
	}
	manifest := &goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}}
	err = input.scan(ctx, manifest, nil)
	if err == nil && (len(manifest.controls["package.json"]) == 0 || manifest.storage > artifactnpm.NPMProjectInspectionWorkspaceLimit) {
		err = errors.New("npm local source manifest/storage is unavailable")
	}
	if err != nil {
		return nil, nil, errors.Join(err, input.close())
	}
	return input, manifest, nil
}

func (r *NPMProjectSourceReader) VerifyBuildSource(ctx context.Context, source domain.AcquiredArtifact) error {
	input, _, err := r.openSource(ctx, source)
	if err != nil {
		return err
	}
	return input.close()
}

// ReadSource checks the exact consumed archive twice around normalized copying;
// only reconstructed headers enter the private project tree. No Host extraction.
func (r *NPMProjectSourceReader) ReadSource(ctx context.Context, source domain.AcquiredArtifact) (layout artifactnpm.PNPMLayoutArchive, resultErr error) {
	input, manifest, err := r.openSource(ctx, source)
	if err != nil {
		return layout, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, input.close())
		if resultErr != nil {
			layout = artifactnpm.PNPMLayoutArchive{}
		}
	}()
	output := nativeToolArchiveOutput{limit: artifactnpm.NPMProjectInspectionWorkspaceLimit}
	writer := tar.NewWriter(&output)
	if err := writer.WriteHeader(&tar.Header{Name: "project", Typeflag: tar.TypeDir, Mode: 0o500, Uid: 1000, Gid: 1000}); err != nil {
		return layout, err
	}
	if err := input.scan(ctx, manifest, writer); err != nil {
		return layout, err
	}
	if err := writer.Close(); err != nil || output.failed {
		return layout, errors.Join(errors.New("npm local source copy is unavailable"), err)
	}
	return artifactnpm.ParsePNPMInspectionPayloadArchive(ctx, output.body)
}
