package promotion

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// A transaction member is an exact file or tree, never an open-ended path.
type npmTransactionMember struct {
	directories map[string]os.FileInfo
	files       map[string]pypiFileSnapshot
}

func snapshotNPMTransactionMember(root string) (*npmTransactionMember, error) {
	m := &npmTransactionMember{directories: map[string]os.FileInfo{}, files: map[string]pypiFileSnapshot{}}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			info, err := os.Lstat(path)
			if err != nil || trustedExistingDirectory(path) != nil {
				return errors.New("npm transaction directory is untrusted")
			}
			m.directories[relative] = info
			return nil
		}
		m.files[relative], err = snapshotPyPIFile(path)
		return err
	})
	return m, err
}

func (m *npmTransactionMember) verify(root string) error {
	if m == nil {
		return errors.New("npm transaction member is unavailable")
	}
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if before, known := m.directories[relative]; known {
			info, err := os.Lstat(path)
			if err != nil || !entry.IsDir() || !os.SameFile(before, info) || before.Mode() != info.Mode() || trustedExistingDirectory(path) != nil {
				return errors.New("npm transaction directory changed")
			}
		} else if before, known := m.files[relative]; !known || !before.matches(path) {
			return errors.New("npm transaction member changed or is foreign")
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if count != len(m.directories)+len(m.files) {
		return errors.New("npm transaction member is missing")
	}
	return nil
}

func (m *npmTransactionMember) remove(path string) error {
	if err := m.verify(path); err != nil {
		return err
	}
	if record, file := m.files["."]; file {
		if !record.matches(path) {
			return errors.New("npm transaction file changed before removal")
		}
		return os.Remove(path)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	current, err := root.Stat(".")
	if err != nil || !os.SameFile(m.directories["."], current) {
		return errors.New("npm transaction removal boundary changed")
	}
	for name, record := range m.files {
		if !record.matches(filepath.Join(path, name)) {
			return errors.New("npm transaction file changed before removal")
		}
		if err := root.Remove(name); err != nil {
			return err
		}
	}
	directories := make([]string, 0, len(m.directories))
	for name := range m.directories {
		directories = append(directories, name)
	}
	sort.Slice(directories, func(i, j int) bool { return len(directories[i]) > len(directories[j]) })
	for _, name := range directories {
		current, err := root.Lstat(name)
		before := m.directories[name]
		if err != nil || !os.SameFile(before, current) || before.Mode() != current.Mode() {
			return errors.New("npm transaction directory changed before removal")
		}
		if name == "." {
			continue
		}
		if err := root.Remove(name); err != nil {
			return err
		}
	}
	return os.Remove(path)
}

func (t *npmProjectTransaction) verifyBoundaries() error {
	for path, before := range map[string]os.FileInfo{t.plan.root: t.plan.rootInfo, t.workspace: t.workspaceInfo, t.backup: t.backupInfo} {
		info, err := os.Lstat(path)
		if err != nil || before == nil || !info.IsDir() || !os.SameFile(before, info) || before.Mode() != info.Mode() || trustedExistingDirectory(path) != nil {
			return errors.New("npm transaction boundary changed")
		}
	}
	return nil
}

func (t *npmProjectTransaction) removeBackup(originals bool) error {
	if err := t.verifyBoundaries(); err != nil {
		return err
	}
	f, err := os.Open(t.backup)
	if err != nil {
		return err
	}
	opened, statErr := f.Stat()
	entries, readErr := f.ReadDir(5)
	closeErr := f.Close()
	wanted := 0
	if originals {
		wanted = len(t.before)
	}
	if statErr != nil || !os.SameFile(t.backupInfo, opened) || readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) != wanted {
		return errors.New("npm rollback backup inventory changed")
	}
	for _, entry := range entries {
		member, known := t.before[entry.Name()]
		if !originals || !known {
			return errors.New("npm rollback backup contains a foreign member")
		}
		if err := member.verify(filepath.Join(t.backup, entry.Name())); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		if err := t.before[entry.Name()].remove(filepath.Join(t.backup, entry.Name())); err != nil {
			return err
		}
	}
	if err := t.verifyBoundaries(); err != nil {
		return err
	}
	return os.Remove(t.backup)
}
