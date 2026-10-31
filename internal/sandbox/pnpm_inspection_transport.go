package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMArchiveTransport = errors.New("pnpm controlled archive introduction is unavailable or changed")

type pnpmInspectionArchive struct {
	subject domain.AcquiredArtifact
	name    string
	body    []byte
}

// Freeze complete measured archives before any native tool/DNS/resource work.
// Only RunID-derived intake names are opened; Host names are never guest input.
func (i *DockerArtifactIntroducer) freezePNPMArchives(ctx context.Context, input artifactnpm.PNPMInspectionInput) ([]pnpmInspectionArchive, error) {
	if i == nil || ctx == nil || !input.Valid() || len(input.Artifacts()) == 0 {
		return nil, errPNPMArchiveTransport
	}
	return i.freezeNPMProjectArchives(ctx, input.Artifacts())
}

func (i *DockerArtifactIntroducer) freezeBunArchives(ctx context.Context, input artifactnpm.BunInspectionInput) ([]pnpmInspectionArchive, error) {
	if i == nil || ctx == nil || !input.Valid() || len(input.Artifacts()) == 0 {
		return nil, errPNPMArchiveTransport
	}
	return i.freezeNPMProjectArchives(ctx, input.Artifacts())
}

func (i *DockerArtifactIntroducer) freezeNPMProjectArchives(ctx context.Context, subjects []domain.AcquiredArtifact) (archives []pnpmInspectionArchive, resultErr error) {
	if i == nil || ctx == nil || len(subjects) == 0 {
		return nil, errPNPMArchiveTransport
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errPNPMArchiveTransport, err, context.Cause(ctx))
	}
	root, err := os.OpenRoot(i.intakeRoot)
	if err != nil {
		return nil, errPNPMArchiveTransport
	}
	defer func() {
		if err := root.Close(); err != nil {
			archives = nil
			resultErr = errors.Join(resultErr, errPNPMArchiveTransport, err)
		}
	}()
	var total uint64
	for _, subject := range subjects {
		if subject.Identity().Source().String() != "npm" || subject.Identity().Variant() != "tarball" || subject.SizeBytes() == 0 || subject.SizeBytes() > artifactnpm.PNPMInspectionWorkspaceLimit-total {
			return nil, errPNPMArchiveTransport
		}
		absolute, err := i.tarballPath(subject.ContentHandle())
		if err != nil {
			return nil, errPNPMArchiveTransport
		}
		name, err := filepath.Rel(i.intakeRoot, absolute)
		if err != nil {
			return nil, errPNPMArchiveTransport
		}
		body, err := readPNPMIntakeArchive(ctx, root, name, subject)
		if err != nil {
			return nil, err
		}
		total += subject.SizeBytes()
		archives = append(archives, pnpmInspectionArchive{subject, "artifacts/" + subject.Digest().String() + ".tgz", body})
	}
	return archives, nil
}

func readPNPMIntakeArchive(ctx context.Context, root *os.Root, name string, subject domain.AcquiredArtifact) ([]byte, error) {
	parts := strings.Split(subject.ContentHandle(), ":")
	if len(parts) != 3 || name != filepath.Join(parts[1], "tarball.tgz") {
		return nil, errPNPMArchiveTransport
	}
	body, err := artifactnpm.ReadPNPMIntakeArchive(ctx, root, subject)
	if err != nil {
		return nil, errors.Join(errPNPMArchiveTransport, err)
	}
	return body, nil
}

func copyPNPMInspectionArchives(ctx context.Context, runner CommandRunner, container string, archives []pnpmInspectionArchive) error {
	input, ok := runner.(inputCommandRunner)
	streams, streamOK := runner.(streamingDockerOutput)
	if ctx == nil || !ok || !streamOK || !exactObservationContainerID(container) || len(archives) == 0 {
		return errPNPMArchiveTransport
	}
	var body bytes.Buffer
	writer := tar.NewWriter(&body)
	for _, name := range []string{"artifacts", "tmp", "cache"} {
		if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o700, Uid: 1000, Gid: 1000}); err != nil {
			return errors.Join(errPNPMArchiveTransport, err)
		}
	}
	for _, archive := range archives {
		if err := ctx.Err(); err != nil {
			return errors.Join(errPNPMArchiveTransport, err, context.Cause(ctx))
		}
		if archive.name != "artifacts/"+archive.subject.Digest().String()+".tgz" || uint64(len(archive.body)) != archive.subject.SizeBytes() || len(archive.body) > artifactnpm.PNPMInspectionWorkspaceLimit || body.Len() > artifactnpm.PNPMInspectionWorkspaceLimit-len(archive.body) {
			return errPNPMArchiveTransport
		}
		if err := writer.WriteHeader(&tar.Header{Name: archive.name, Typeflag: tar.TypeReg, Mode: 0o600, Uid: 1000, Gid: 1000, Size: int64(len(archive.body))}); err != nil {
			return errors.Join(errPNPMArchiveTransport, err)
		}
		if _, err := writer.Write(archive.body); err != nil {
			return errors.Join(errPNPMArchiveTransport, err)
		}
	}
	if err := writer.Close(); err != nil {
		return errors.Join(errPNPMArchiveTransport, err)
	}
	if body.Len() > artifactnpm.PNPMInspectionWorkspaceLimit {
		return errPNPMArchiveTransport
	}
	if err := input.RunInput(ctx, &body, "docker", "cp", "--archive", "-", container+":"+pnpmProjectRoot); err != nil {
		return errors.Join(errPNPMArchiveTransport, err)
	}
	for _, archive := range archives {
		if err := verifyPNPMGuestArchive(ctx, streams, container, archive); err != nil {
			return err
		}
	}
	return nil
}

func verifyPNPMGuestArchive(ctx context.Context, runner streamingDockerOutput, container string, expected pnpmInspectionArchive) error {
	return verifyNativeNPMGuestArchive(ctx, runner, container, nativeRuntimePNPM, expected)
}

func verifyNativeNPMGuestArchive(ctx context.Context, runner streamingDockerOutput, container string, kind nativeRuntimeToolKind, expected pnpmInspectionArchive) error {
	spec, known := nativeProjectIdentity(kind)
	if !known {
		return errPNPMArchiveTransport
	}
	if ctx == nil || runner == nil || !exactObservationContainerID(container) || expected.subject.SizeBytes() == 0 || expected.subject.SizeBytes() > artifactnpm.PNPMInspectionWorkspaceLimit || expected.name != "artifacts/"+expected.subject.Digest().String()+".tgz" {
		return errPNPMArchiveTransport
	}
	output := nativeToolArchiveOutput{limit: int64(expected.subject.SizeBytes()) + 64<<10}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+spec.root+"/"+expected.name, "-"); err != nil || output.failed {
		return errors.Join(errPNPMArchiveTransport, err)
	}
	input := bytes.NewReader(output.body)
	reader := tar.NewReader(input)
	header, err := reader.Next()
	if err != nil || header.Typeflag != tar.TypeReg || header.Name != filepath.Base(expected.name) || header.Linkname != "" || header.Uid != 1000 || header.Gid != 1000 || header.Mode != 0o600 || header.Size != int64(expected.subject.SizeBytes()) {
		return errPNPMArchiveTransport
	}
	body, err := io.ReadAll(io.LimitReader(reader, header.Size+1))
	if err != nil || int64(len(body)) != header.Size {
		return errors.Join(errPNPMArchiveTransport, err)
	}
	hash := sha256.Sum256(body)
	if hex.EncodeToString(hash[:]) != expected.subject.Digest().String() {
		return errPNPMArchiveTransport
	}
	if _, err := reader.Next(); err != io.EOF {
		return errPNPMArchiveTransport
	}
	for _, value := range output.body[len(output.body)-input.Len():] {
		if value != 0 {
			return errPNPMArchiveTransport
		}
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(errPNPMArchiveTransport, err, context.Cause(ctx))
	}
	return nil
}
