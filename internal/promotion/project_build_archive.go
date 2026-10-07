package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// freezeProjectBuildArchive persists controller-selected transport data. The
// caller supplies independently checked bytes; this helper grants no approval.
func freezeProjectBuildArchive(ctx context.Context, intake string, identity domain.ResolvedArtifactIdentity, limit int64, write func(io.Writer) error) (result domain.AcquiredArtifact, resultErr error) {
	if ctx == nil || ctx.Err() != nil || limit <= 0 || write == nil || identity.Source().String() != "project-local" {
		return result, errors.New("project build archive request is invalid")
	}
	if err := ensureTrustedRoot(intake); err != nil {
		return result, err
	}
	before, err := os.Lstat(intake)
	if err != nil {
		return result, err
	}
	parent, err := os.OpenRoot(intake)
	if err != nil {
		return result, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, parent.Close())
		if resultErr != nil {
			result = domain.AcquiredArtifact{}
		}
	}()
	opened, err := parent.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		return result, errors.New("project build intake changed")
	}
	run, err := domain.NewRunID()
	if err != nil {
		return result, err
	}
	if err := parent.Mkdir(run.String(), 0o700); err != nil {
		return result, err
	}
	owned, err := parent.Lstat(run.String())
	if err != nil {
		return result, err
	}
	defer func() {
		if resultErr != nil {
			current, err := parent.Lstat(run.String())
			if err != nil || !current.IsDir() || !os.SameFile(owned, current) {
				resultErr = errors.Join(resultErr, errors.New("project build archive cleanup ownership uncertain"))
			} else {
				resultErr = errors.Join(resultErr, parent.RemoveAll(run.String()))
			}
			result = domain.AcquiredArtifact{}
		}
	}()
	root, err := parent.OpenRoot(run.String())
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	opened, err = root.Stat(".")
	if err != nil || !os.SameFile(owned, opened) || opened.Mode() != os.ModeDir|0o700 {
		return result, errors.New("project build archive ownership changed")
	}
	file, err := root.OpenFile(identity.Variant()+".tar", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return result, err
	}
	hash := sha256.New()
	bounded := &goBuildCacheArchiveWriter{writer: io.MultiWriter(file, hash), remaining: limit}
	writeErr := write(bounded)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr, ctx.Err()); err != nil {
		return result, err
	}
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash.Sum(nil)))
	if err != nil {
		return result, err
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":"+identity.Variant(), uint64(limit-bounded.remaining), "sha256:"+digest.String())
}
