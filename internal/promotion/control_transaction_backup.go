package promotion

import (
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// These records cover only the two Go/Cargo control transaction directories.
// Never recursively remove a recovery directory: another writer's member must
// survive even when it appears after the inventory check.
type controlBackupRecord struct {
	info   os.FileInfo
	digest [32]byte
}

func checkControlBackup(root *os.Root, directory string, info os.FileInfo, selectedInfo os.FileInfo, records map[string]controlBackupRecord) error {
	for _, sub := range []string{"", "selected"} {
		expected := info
		name := directory
		if sub != "" {
			expected = selectedInfo
			name = filepath.Join(directory, sub)
		}
		current, err := root.Lstat(name)
		if expected == nil && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || expected == nil || !current.IsDir() || !os.SameFile(expected, current) || expected.Mode() != current.Mode() {
			return errors.New("control transaction backup directory changed")
		}
		f, err := root.Open(name)
		if err != nil {
			return errors.New("open control transaction backup directory")
		}
		opened, statErr := f.Stat()
		entries, readErr := f.ReadDir(8)
		closeErr := f.Close()
		if statErr != nil || !os.SameFile(expected, opened) || readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) >= 8 {
			return errors.New("control transaction backup inventory is unavailable or excessive")
		}
		count := 0
		for _, entry := range entries {
			if sub == "" && entry.Name() == "selected" && selectedInfo != nil {
				continue
			}
			key := entry.Name()
			if sub != "" {
				key = filepath.Join(sub, key)
			}
			record, known := records[key]
			if !known {
				return errors.New("control transaction backup contains a foreign member")
			}
			body, member, err := readGoTransactionControl(root, filepath.Join(directory, key))
			if err != nil || record.info == nil || !os.SameFile(record.info, member) || record.info.Mode() != member.Mode() || !pypiSingleLink(member) || sha256.Sum256(body) != record.digest {
				return errors.New("control transaction backup member changed")
			}
			count++
		}
		wanted := 0
		for key := range records {
			if filepath.Dir(key) == "." && sub == "" || filepath.Dir(key) == "selected" && sub == "selected" {
				wanted++
			}
		}
		if count != wanted {
			return errors.New("control transaction backup member is missing")
		}
	}
	return nil
}

func removeControlBackup(root *os.Root, directory string, info, selectedInfo os.FileInfo, records map[string]controlBackupRecord) error {
	if err := checkControlBackup(root, directory, info, selectedInfo, records); err != nil {
		return err
	}
	for name, record := range records {
		body, current, err := readGoTransactionControl(root, filepath.Join(directory, name))
		if err != nil || !os.SameFile(record.info, current) || record.info.Mode() != current.Mode() || !pypiSingleLink(current) || sha256.Sum256(body) != record.digest {
			return errors.New("control transaction backup changed before cleanup")
		}
		if err := root.Remove(filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	if selectedInfo != nil {
		current, err := root.Lstat(filepath.Join(directory, "selected"))
		if err != nil || !os.SameFile(selectedInfo, current) || selectedInfo.Mode() != current.Mode() {
			return errors.New("control transaction selected boundary changed before cleanup")
		}
		if err := root.Remove(filepath.Join(directory, "selected")); err != nil {
			return err
		}
	}
	current, err := root.Lstat(directory)
	if err != nil || !os.SameFile(info, current) || info.Mode() != current.Mode() {
		return errors.New("control transaction backup boundary changed before cleanup")
	}
	return root.Remove(directory)
}

func (t *goProjectTransaction) backupRecords(originals bool) map[string]controlBackupRecord {
	records := map[string]controlBackupRecord{}
	for _, control := range t.plan.controls {
		name := control.Name()
		if originals && t.moved[name] {
			records[name] = controlBackupRecord{t.plan.members[name], sha256.Sum256(control.Body())}
		}
		if !t.published[name] && t.selectedInfo[name] != nil {
			records[filepath.Join("selected", name)] = controlBackupRecord{t.selectedInfo[name], t.selectedHashes[name]}
		}
	}
	if originals && t.metadataMoved {
		records["go-transaction.json"] = controlBackupRecord{t.originalMetadataInfo, t.originalMetadataHash}
	}
	return records
}
func (t *goProjectTransaction) verifyBackupContents(originals bool) error {
	current, err := os.Lstat(t.plan.root)
	if err != nil || !os.SameFile(t.plan.rootInfo, current) || t.plan.rootInfo.Mode() != current.Mode() {
		return errors.New("go transaction root changed")
	}
	return checkControlBackup(t.boundary, filepath.Base(t.backup), t.backupInfo, t.selectedDirectoryInfo, t.backupRecords(originals))
}
func (t *goProjectTransaction) removeBackup(originals bool) error {
	if err := t.verifyBackupContents(originals); err != nil {
		return err
	}
	return removeControlBackup(t.boundary, filepath.Base(t.backup), t.backupInfo, t.selectedDirectoryInfo, t.backupRecords(originals))
}

func (t *cargoProjectTransaction) backupRecords(originals bool) map[string]controlBackupRecord {
	records := map[string]controlBackupRecord{}
	for _, control := range t.plan.controls {
		name := control.Name()
		if originals && t.moved[name] {
			records[name] = controlBackupRecord{t.plan.members[name], sha256.Sum256(control.Body())}
		}
		if !t.published[name] && t.selectedInfo[name] != nil {
			records[filepath.Join("selected", name)] = controlBackupRecord{t.selectedInfo[name], t.selectedHashes[name]}
		}
	}
	if originals && t.metadataMoved {
		records["cargo-transaction.json"] = controlBackupRecord{t.originalMetadataInfo, t.originalMetadataHash}
	}
	return records
}
func (t *cargoProjectTransaction) verifyBackupContents(originals bool) error {
	current, err := os.Lstat(t.plan.root)
	if err != nil || !os.SameFile(t.plan.rootInfo, current) || t.plan.rootInfo.Mode() != current.Mode() {
		return errors.New("cargo transaction root changed")
	}
	return checkControlBackup(t.boundary, filepath.Base(t.backup), t.backupInfo, t.selectedDirectoryInfo, t.backupRecords(originals))
}
func (t *cargoProjectTransaction) removeBackup(originals bool) error {
	if err := t.verifyBackupContents(originals); err != nil {
		return err
	}
	return removeControlBackup(t.boundary, filepath.Base(t.backup), t.backupInfo, t.selectedDirectoryInfo, t.backupRecords(originals))
}

func controlBackupUnchanged(root *os.Root, name string, record controlBackupRecord) error {
	body, member, err := readGoTransactionControl(root, name)
	if err != nil || record.info == nil || !os.SameFile(record.info, member) || record.info.Mode() != member.Mode() || !pypiSingleLink(member) || sha256.Sum256(body) != record.digest {
		return errors.New("control transaction backup member changed")
	}
	return nil
}

func (t *goProjectTransaction) verifyBackupBoundary() error {
	current, err := os.Lstat(t.plan.root)
	backup, backupErr := t.boundary.Lstat(filepath.Base(t.backup))
	if err != nil || backupErr != nil || t.backupInfo == nil || !os.SameFile(t.plan.rootInfo, current) || t.plan.rootInfo.Mode() != current.Mode() || !os.SameFile(t.backupInfo, backup) || t.backupInfo.Mode() != backup.Mode() {
		return errors.New("go transaction backup boundary changed")
	}
	return nil
}

func (t *cargoProjectTransaction) verifyBackupBoundary() error {
	current, err := os.Lstat(t.plan.root)
	backup, backupErr := t.boundary.Lstat(filepath.Base(t.backup))
	if err != nil || backupErr != nil || t.backupInfo == nil || !os.SameFile(t.plan.rootInfo, current) || t.plan.rootInfo.Mode() != current.Mode() || !os.SameFile(t.backupInfo, backup) || t.backupInfo.Mode() != backup.Mode() {
		return errors.New("cargo transaction backup boundary changed")
	}
	return nil
}
