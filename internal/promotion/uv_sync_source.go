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
	if ctx == nil || transaction == nil || transaction.before == nil || len(trustedRoots) == 0 {
		return artifactpypi.UVSyncInput{}, errors.New("UV sync source requires a held transaction and trusted boundaries")
	}
	if err := ctx.Err(); err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	roots := append([]string{transaction.plan.root, source}, trustedRoots...)
	for _, path := range roots {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || rejectSymlinkPathAllowMissing(path) != nil {
			return artifactpypi.UVSyncInput{}, errors.New("UV sync source boundaries are not canonical")
		}
	}
	if !separateRoots(roots) {
		return artifactpypi.UVSyncInput{}, errors.New("UV sync source overlaps its target or trusted storage")
	}
	if _, frozen := transaction.before[source]; frozen {
		return artifactpypi.UVSyncInput{}, errors.New("UV sync source was already frozen")
	}
	if err := transaction.checkBefore(); err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	if err := transaction.freezeParents(filepath.Dir(source)); err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	snapshot, err := snapshotBoundedPromotionFile(source, artifactpypi.UVSyncSourceByteLimit)
	if err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	body, err := readBoundedPromotionFile(source, artifactpypi.UVSyncSourceByteLimit)
	if err != nil {
		return artifactpypi.UVSyncInput{}, err
	}
	sum := sha256.Sum256(body)
	if snapshot.Digest != hex.EncodeToString(sum[:]) || !snapshot.matches(source) {
		return artifactpypi.UVSyncInput{}, errors.New("UV sync source changed during capture")
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
