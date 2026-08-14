//go:build !linux || !amd64

package promotion

import (
	"errors"
	"os"
)

// Non-Linux production calls are rejected before this test portability path.
func renameNoReplace(oldPath, newPath string) error {
	if _, err := os.Lstat(newPath); !errors.Is(err, os.ErrNotExist) {
		return errors.New("destination exists or cannot be verified")
	}
	return os.Rename(oldPath, newPath)
}
