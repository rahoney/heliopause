package npm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var errPNPMIntakeArchive = errors.New("pnpm controlled archive is unavailable or changed")

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

// ReadPNPMIntakeArchive rechecks one canonical Run-bound source without
// extraction or execution. Native introduction and approved staging share it.
func ReadPNPMIntakeArchive(ctx context.Context, root *os.Root, subject domain.AcquiredArtifact) (body []byte, resultErr error) {
	if ctx == nil || root == nil || subject.SizeBytes() == 0 || subject.SizeBytes() > PNPMInspectionWorkspaceLimit {
		return nil, errPNPMIntakeArchive
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errPNPMIntakeArchive, err, context.Cause(ctx))
	}
	parts := strings.Split(subject.ContentHandle(), ":")
	if subject.Identity().Source().String() != "npm" || subject.Identity().Variant() != "tarball" || len(parts) != 3 || parts[0] != "intake" || parts[2] != "tarball" {
		return nil, errPNPMIntakeArchive
	}
	if _, err := domain.ParseRunID(parts[1]); err != nil {
		return nil, errPNPMIntakeArchive
	}
	name := filepath.Join(parts[1], "tarball.tgz")
	parent := filepath.Dir(name)
	beforeParent, err := root.Lstat(parent)
	if err != nil || !beforeParent.IsDir() {
		return nil, errPNPMIntakeArchive
	}
	before, err := root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Size() != int64(subject.SizeBytes()) {
		return nil, errPNPMIntakeArchive
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, errPNPMIntakeArchive
	}
	defer func() {
		if err := file.Close(); err != nil {
			body = nil
			resultErr = errors.Join(resultErr, errPNPMIntakeArchive, err)
		}
	}()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) || !opened.Mode().IsRegular() || opened.Size() != before.Size() {
		return nil, errPNPMIntakeArchive
	}
	body, err = io.ReadAll(io.LimitReader(pnpmArchiveContextReader{ctx, file}, int64(subject.SizeBytes())+1))
	if err != nil || ctx.Err() != nil || uint64(len(body)) != subject.SizeBytes() {
		return nil, errors.Join(errPNPMIntakeArchive, err, ctx.Err(), context.Cause(ctx))
	}
	hash := sha256.Sum256(body)
	if hex.EncodeToString(hash[:]) != subject.Digest().String() {
		return nil, errPNPMIntakeArchive
	}
	after, fileErr := root.Lstat(name)
	afterParent, parentErr := root.Lstat(parent)
	if fileErr != nil || parentErr != nil || !os.SameFile(before, after) || !after.Mode().IsRegular() || after.Size() != before.Size() || !os.SameFile(beforeParent, afterParent) || !afterParent.IsDir() {
		return nil, errPNPMIntakeArchive
	}
	return body, nil
}
