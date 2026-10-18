package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

const npmSnapshotFileLimit int64 = 64 << 20
const npmSnapshotTreeLimit int64 = 1 << 30
const npmSnapshotEntryLimit = 100000

// Reuse the existing transaction fingerprint, with a bounded read for npm's
// pre-resolution originals. Other ecosystems retain their existing semantics.
func snapshotNPMFile(path string) (pypiFileSnapshot, error) {
	if rejectSymlinkPath(path) != nil {
		return pypiFileSnapshot{}, errors.New("unsafe npm snapshot path")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || !pypiSingleLink(before) || before.Size() < 0 || before.Size() > npmSnapshotFileLimit {
		return pypiFileSnapshot{}, errors.New("npm snapshot file is unsupported or exceeds bounds")
	}
	file, err := os.Open(path)
	if err != nil {
		return pypiFileSnapshot{}, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !os.SameFile(before, opened) || !pypiSingleLink(opened) {
		_ = file.Close()
		return pypiFileSnapshot{}, errors.New("npm snapshot file identity changed")
	}
	hash := sha256.New()
	n, copyErr := io.Copy(hash, io.LimitReader(file, npmSnapshotFileLimit+1))
	closeErr := file.Close()
	after, statErr := os.Lstat(path)
	if copyErr != nil || closeErr != nil || statErr != nil || n > npmSnapshotFileLimit || n != before.Size() || !os.SameFile(before, after) || !pypiSingleLink(after) || after.Size() != n || before.Mode() != after.Mode() || before.ModTime() != after.ModTime() {
		return pypiFileSnapshot{}, errors.New("npm file changed during bounded snapshot")
	}
	device, inode, ok := pypiObjectIdentity(before)
	if !ok {
		return pypiFileSnapshot{}, errors.New("npm file identity is unavailable")
	}
	return pypiFileSnapshot{hex.EncodeToString(hash.Sum(nil)), n, before.Mode(), device, inode, before}, nil
}

func npmSnapshotMatches(before pypiFileSnapshot, path string) bool {
	now, err := snapshotNPMFile(path)
	return err == nil && os.SameFile(before.info, now.info) && before.Digest == now.Digest && before.Size == now.Size && before.Mode == now.Mode
}
