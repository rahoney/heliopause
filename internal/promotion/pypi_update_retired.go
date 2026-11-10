package promotion

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

const pypiUpgradeDirectoryLimit = 100000
const pypiRetiredMetadataDepthLimit = 64

func sameRetiredMetadataDirectory(before, after os.FileInfo) bool {
	return before != nil && after != nil && before.IsDir() && after.IsDir() && os.SameFile(before, after) && before.Mode() == after.Mode() && before.ModTime() == after.ModTime()
}

// Obsolete wheel files can leave nested empty license directories. These are
// frozen namespace state, not installed metadata or newly inferred ownership.
// Root-scoped reads never traverse a link outside the retired tree.
func snapshotPyPIRetiredMetadata(directory string, limit int) (tree map[string]os.FileInfo, empty bool, resultErr error) {
	if trustedExistingDirectory(directory) != nil {
		return nil, false, errors.New("untrusted installed metadata directory")
	}
	before, err := os.Lstat(directory)
	if err != nil {
		return nil, false, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, false, err
	}
	defer func() {
		if err := root.Close(); err != nil {
			tree = nil
			empty = false
			resultErr = errors.Join(resultErr, err)
		}
	}()
	opened, err := root.Stat(".")
	if err != nil || !sameRetiredMetadataDirectory(before, opened) {
		return nil, false, errors.New("installed metadata root changed")
	}
	if _, err := root.Lstat("METADATA"); err == nil {
		return nil, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	if limit < 1 || limit > pypiUpgradeDirectoryLimit {
		return nil, false, errors.New("retired metadata directory budget exceeded")
	}
	type pending struct {
		relative string
		depth    int
	}
	queue := []pending{{".", 0}}
	tree = map[string]os.FileInfo{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.depth > pypiRetiredMetadataDepthLimit || len(current.relative) > 4096 {
			return nil, false, errors.New("retired metadata tree depth exceeds bound")
		}
		info, err := root.Lstat(current.relative)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, false, errors.New("retired metadata directory identity changed")
		}
		file, err := root.Open(current.relative)
		if err != nil {
			return nil, false, err
		}
		actual, statErr := file.Stat()
		if statErr != nil || !sameRetiredMetadataDirectory(info, actual) {
			_ = file.Close()
			return nil, false, errors.New("retired metadata opened directory changed")
		}
		tree[current.relative] = info
		remaining := limit - len(tree) - len(queue)
		entries, readErr := file.ReadDir(remaining + 1)
		closeErr := file.Close()
		if readErr != nil && readErr != io.EOF || closeErr != nil {
			return nil, false, errors.Join(readErr, closeErr)
		}
		for _, entry := range entries {
			if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil, false, nil
			}
			if len(tree)+len(queue) >= limit {
				return nil, false, errors.New("retired metadata directory count exceeds bound")
			}
			queue = append(queue, pending{filepath.Join(current.relative, entry.Name()), current.depth + 1})
		}
	}
	for relative, original := range tree {
		now, err := root.Lstat(relative)
		if err != nil || !sameRetiredMetadataDirectory(original, now) {
			return nil, false, errors.New("retired metadata tree changed while reading")
		}
	}
	after, err := os.Lstat(directory)
	if err != nil || !sameRetiredMetadataDirectory(before, after) {
		return nil, false, errors.New("retired metadata root changed while reading")
	}
	return tree, true, nil
}
