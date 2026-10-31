package promotion

import (
	"errors"
	"os"
)

// The live creation descriptor pins the inode until ownership is checked and
// the owned name is removed. Path-only inode equality cannot prevent ABA reuse.
func verifyPinnedMutationLock(file *os.File, original, current os.FileInfo) error {
	if file == nil || original == nil || current == nil {
		return errors.New("mutation lock descriptor or identity unavailable")
	}
	held, err := file.Stat()
	if err != nil || !held.Mode().IsRegular() || !current.Mode().IsRegular() || !os.SameFile(original, held) || !os.SameFile(held, current) || !pypiSingleLink(held) || !pypiSingleLink(current) || held.Size() != 0 || current.Size() != 0 || held.Mode() != original.Mode() || current.Mode() != original.Mode() || held.Mode().Perm() != 0o600 || !held.ModTime().Equal(original.ModTime()) || !current.ModTime().Equal(original.ModTime()) {
		return errors.New("mutation lock changed, aliased or no longer owned")
	}
	return nil
}
