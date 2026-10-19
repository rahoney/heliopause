package promotion

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// A managed target has exactly one asset and one controller marker. Keep the
// existing M6 asset bound instead of importing npm's smaller file limit.
type githubDirectorySnapshot struct {
	info  os.FileInfo
	files map[string]pypiFileSnapshot
}

func snapshotGitHubDirectory(path string, expected githubAssetApproval, body []byte) (*githubDirectorySnapshot, error) {
	if trustedExistingDirectory(path) != nil {
		return nil, errors.New("GitHub installed directory is untrusted")
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != 0o700 {
		return nil, errors.New("GitHub installed directory permissions changed")
	}
	result := &githubDirectorySnapshot{info: info, files: map[string]pypiFileSnapshot{}}
	for name, limit := range map[string]int64{expected.Asset: githubAssetLimit, githubOwnershipMarker: githubApprovalLimit} {
		record, err := snapshotBoundedPromotionFile(filepath.Join(path, name), limit)
		if err != nil {
			return nil, err
		}
		result.files[name] = record
	}
	asset := result.files[expected.Asset]
	marker := result.files[githubOwnershipMarker]
	if asset.Size != int64(expected.Size) || asset.Digest != expected.SHA256 || asset.Mode.Perm() != os.FileMode(expected.Mode) || marker.Mode.Perm() != 0o600 {
		return nil, errors.New("GitHub installed asset differs from independent approval")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	actual, _, err := readGoTransactionControl(root, githubOwnershipMarker)
	if err != nil || !bytes.Equal(actual, body) {
		return nil, errors.New("GitHub installed marker differs from independent approval")
	}
	if err := result.verify(path); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *githubDirectorySnapshot) verify(path string) error {
	if s == nil || s.info == nil || trustedExistingDirectory(path) != nil {
		return errors.New("GitHub directory snapshot is unavailable")
	}
	info, err := os.Lstat(path)
	if err != nil || !os.SameFile(s.info, info) || info.Mode() != s.info.Mode() || !info.ModTime().Equal(s.info.ModTime()) {
		return errors.New("GitHub directory identity, permissions or timestamp changed")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("GitHub directory changed during open")
	}
	file, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(len(s.files) + 1)
	closeErr := file.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) != len(s.files) {
		return errors.New("GitHub target contains missing or foreign members")
	}
	for _, entry := range entries {
		record, known := s.files[entry.Name()]
		if !known || entry.IsDir() || !record.matches(filepath.Join(path, entry.Name())) {
			return errors.New("GitHub installed member changed or is foreign")
		}
		current, err := root.Lstat(entry.Name())
		if err != nil || !current.ModTime().Equal(record.info.ModTime()) {
			return errors.New("GitHub installed member timestamp changed")
		}
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, current) || !current.ModTime().Equal(info.ModTime()) {
		return errors.New("GitHub directory changed during verification")
	}
	return nil
}

// Remove only the exact proven tree. A new/changed member or root is preserved.
func (s *githubDirectorySnapshot) remove(path string) error {
	if err := s.verify(path); err != nil {
		return err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	for name, record := range s.files {
		current, err := root.Lstat(name)
		if err != nil || !record.matches(filepath.Join(path, name)) || !current.ModTime().Equal(record.info.ModTime()) {
			return errors.New("GitHub member changed before removal")
		}
		if err := root.Remove(name); err != nil {
			return err
		}
	}
	parent, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer parent.Close()
	current, err := parent.Lstat(filepath.Base(path))
	if err != nil || !os.SameFile(s.info, current) {
		return errors.New("GitHub removal root changed")
	}
	return parent.Remove(filepath.Base(path))
}
