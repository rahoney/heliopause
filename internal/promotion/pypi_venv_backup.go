package promotion

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// The ledger and moved originals are the only disposable backup members.
// Unrecognized, replaced or modified files retain the recovery boundary.
func (t *pypiVenvTransaction) removeBackup(originals bool) error {
	if err := t.verifyPathIdentity(t.backup); err != nil {
		return err
	}
	records := map[string]*pypiFileSnapshot{}
	if originals {
		for i, d := range t.backed {
			records["file-"+strconv.Itoa(i)] = t.before[d.Final]
		}
		if t.metadataBacked {
			records["metadata"] = t.metadata
		}
	}
	if !t.metadataPublished && t.publishedMetadata != nil {
		records["next-metadata"] = t.publishedMetadata
	}
	f, err := os.Open(t.backup)
	if err != nil {
		return err
	}
	info, statErr := f.Stat()
	entries, readErr := f.ReadDir(len(records) + 2)
	closeErr := f.Close()
	wanted := len(records)
	if t.journalIdentity != nil {
		wanted++
	}
	if statErr != nil || !os.SameFile(t.directories[t.backup], info) || readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) != wanted {
		return errors.New("recovery backup inventory changed")
	}
	for _, entry := range entries {
		if entry.Name() == "ledger.json" && t.journalIdentity != nil {
			if err := t.verifyBackupJournal(); err != nil {
				return err
			}
			continue
		}
		record, known := records[entry.Name()]
		if !known || record == nil || !record.matches(filepath.Join(t.backup, entry.Name())) {
			return errors.New("recovery backup member changed or is foreign")
		}
	}
	for name, record := range records {
		path := filepath.Join(t.backup, name)
		if !record.matches(path) {
			return errors.New("recovery backup changed before cleanup")
		}
		if err := t.remove(path); err != nil {
			return err
		}
	}
	if t.journalIdentity != nil {
		if err := t.verifyBackupJournal(); err != nil {
			return err
		}
		if err := t.remove(filepath.Join(t.backup, "ledger.json")); err != nil {
			return err
		}
	}
	if err := t.verifyPathIdentity(t.backup); err != nil {
		return err
	}
	return t.remove(t.backup)
}

func (t *pypiVenvTransaction) verifyBackupJournal() error {
	path := filepath.Join(t.backup, "ledger.json")
	info, err := os.Lstat(path)
	if err != nil || t.journalHash == nil || !info.Mode().IsRegular() || !os.SameFile(t.journalIdentity, info) || info.Mode() != t.journalIdentity.Mode() || !pypiSingleLink(info) || info.Size() != t.journalSize {
		return errors.New("recovery journal changed before cleanup")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	opened, statErr := f.Stat()
	h := sha256.New()
	n, readErr := io.Copy(h, io.LimitReader(f, t.journalSize+1))
	closeErr := f.Close()
	after, afterErr := os.Lstat(path)
	if statErr != nil || !os.SameFile(info, opened) || readErr != nil || closeErr != nil || afterErr != nil || !os.SameFile(info, after) || info.Mode() != after.Mode() || !pypiSingleLink(after) || after.Size() != t.journalSize || n != t.journalSize || !bytes.Equal(h.Sum(nil), t.journalHash.Sum(nil)) {
		return errors.New("recovery journal content changed before cleanup")
	}
	return nil
}
