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

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMArchiveTransport = errors.New("pnpm controlled archive introduction is unavailable or changed")

type pnpmInspectionArchive struct {
	subject domain.AcquiredArtifact
	name    string
	body    []byte
}

type pnpmArchiveContextReader struct {
	context context.Context
	reader  io.Reader
}

func (r pnpmArchiveContextReader) Read(body []byte) (int, error) {
	if err := r.context.Err(); err != nil {
		return 0, errors.Join(err, context.Cause(r.context))
	}
	return r.reader.Read(body)
}

// Freeze complete measured archives before any native tool/DNS/resource work.
// Only RunID-derived intake names are opened; Host names are never guest input.
func (i *DockerArtifactIntroducer) freezePNPMArchives(ctx context.Context, input artifactnpm.PNPMInspectionInput) (archives []pnpmInspectionArchive, resultErr error) {
	if i == nil || ctx == nil || !input.Valid() || len(input.Artifacts()) == 0 {
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
	for _, subject := range input.Artifacts() {
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

func readPNPMIntakeArchive(ctx context.Context, root *os.Root, name string, subject domain.AcquiredArtifact) (body []byte, resultErr error) {
	if ctx == nil || root == nil || subject.SizeBytes() == 0 || subject.SizeBytes() > artifactnpm.PNPMInspectionWorkspaceLimit {
		return nil, errPNPMArchiveTransport
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errPNPMArchiveTransport, err, context.Cause(ctx))
	}
	parent := filepath.Dir(name)
	beforeParent, err := root.Lstat(parent)
	if err != nil || !beforeParent.IsDir() {
		return nil, errPNPMArchiveTransport
	}
	before, err := root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Size() != int64(subject.SizeBytes()) {
		return nil, errPNPMArchiveTransport
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, errPNPMArchiveTransport
	}
	defer func() {
		if err := file.Close(); err != nil {
			body = nil
			resultErr = errors.Join(resultErr, errPNPMArchiveTransport, err)
		}
	}()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) || !opened.Mode().IsRegular() || opened.Size() != before.Size() {
		return nil, errPNPMArchiveTransport
	}
	body, err = io.ReadAll(io.LimitReader(pnpmArchiveContextReader{ctx, file}, int64(subject.SizeBytes())+1))
	if err != nil || ctx.Err() != nil || uint64(len(body)) != subject.SizeBytes() {
		return nil, errors.Join(errPNPMArchiveTransport, err, ctx.Err(), context.Cause(ctx))
	}
	hash := sha256.Sum256(body)
	if hex.EncodeToString(hash[:]) != subject.Digest().String() {
		return nil, errPNPMArchiveTransport
	}
	after, fileErr := root.Lstat(name)
	afterParent, parentErr := root.Lstat(parent)
	if fileErr != nil || parentErr != nil || !os.SameFile(before, after) || !after.Mode().IsRegular() || after.Size() != before.Size() || !os.SameFile(beforeParent, afterParent) || !afterParent.IsDir() {
		return nil, errPNPMArchiveTransport
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
	if ctx == nil || runner == nil || !exactObservationContainerID(container) || expected.subject.SizeBytes() == 0 || expected.subject.SizeBytes() > artifactnpm.PNPMInspectionWorkspaceLimit || expected.name != "artifacts/"+expected.subject.Digest().String()+".tgz" {
		return errPNPMArchiveTransport
	}
	output := nativeToolArchiveOutput{limit: int64(expected.subject.SizeBytes()) + 64<<10}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+pnpmProjectRoot+"/"+expected.name, "-"); err != nil || output.failed {
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
