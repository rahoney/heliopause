package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// Freeze unapproved positive source under the same file/parent witnesses as
// the held venv transaction. Only the later complete guard grants publication.
func freezeUVSyncSource(ctx context.Context, transaction *pypiVenvTransaction, source string, format artifactpypi.UVSyncSourceFormat, allowEmpty bool, trustedRoots []string) (artifactpypi.UVSyncInput, error) {
	body, snapshot, err := freezePythonSourceFile(ctx, transaction, source, trustedRoots, artifactpypi.UVSyncSourceByteLimit)
	if err != nil {
		return artifactpypi.UVSyncInput{}, err
	}

	input, err := artifactpypi.ParseUVSyncInput(format, body, allowEmpty, sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	if input.ShouldSkip() {
		return artifactpypi.UVSyncInput{}, errors.New("blank UV sync source requires the separate nonmutating skip path")
	}
	transaction.before[source] = &snapshot
	if err := ctx.Err(); err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	if err := transaction.checkBefore(); err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	return input, nil
}

func freezePythonSourceFile(ctx context.Context, transaction *pypiVenvTransaction, source string, trustedRoots []string, limit int64) ([]byte, pypiFileSnapshot, error) {
	if ctx == nil || transaction == nil || transaction.before == nil || len(trustedRoots) == 0 {
		return nil, pypiFileSnapshot{}, errors.New("python source requires a held transaction and trusted boundaries")
	}
	if err := ctx.Err(); err != nil {
		return nil, pypiFileSnapshot{}, err
	}
	roots := append([]string{transaction.plan.root, source}, trustedRoots...)
	for _, path := range roots {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || rejectSymlinkPathAllowMissing(path) != nil {
			return nil, pypiFileSnapshot{}, errors.New("python source boundaries are not canonical")
		}
	}
	if !separateRoots(roots) {
		return nil, pypiFileSnapshot{}, errors.New("python source overlaps its target or trusted storage")
	}
	if _, frozen := transaction.before[source]; frozen {
		return nil, pypiFileSnapshot{}, errors.New("python source was already frozen")
	}
	if err := transaction.checkBefore(); err != nil {
		return nil, pypiFileSnapshot{}, err
	}
	if err := transaction.freezeParents(filepath.Dir(source)); err != nil {
		return nil, pypiFileSnapshot{}, err
	}
	snapshot, err := snapshotBoundedPromotionFile(source, limit)
	if err != nil {
		return nil, pypiFileSnapshot{}, err
	}
	body, err := readBoundedPromotionFile(source, limit)
	if err != nil {
		return nil, pypiFileSnapshot{}, err
	}
	sum := sha256.Sum256(body)
	if snapshot.Digest != hex.EncodeToString(sum[:]) || !snapshot.matches(source) {
		return nil, pypiFileSnapshot{}, errors.New("python source changed during capture")
	}
	return body, snapshot, nil
}
