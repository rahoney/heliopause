package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Control-only publication shares the installed project's lock, bounded file
// snapshots and known-backup cleanup. No venv plan or ownership is fabricated.
type uvProjectLockFiles struct {
	root                              string
	guard                             goProjectGuard
	input                             artifactpypi.UVProjectInput
	original                          []domain.ProjectControlFile
	snapshots                         map[string]*pypiFileSnapshot
	backup                            string
	backupInfo, selectedDirectoryInfo os.FileInfo
	records                           map[string]controlBackupRecord
	moved, published, recovery        bool
	checkpoint                        func(string) error
}

func freezeUVProjectLockFiles(ctx context.Context, root string, trustedRoots []string) (result *uvProjectLockFiles, resultErr error) {
	if ctx == nil || len(trustedRoots) == 0 {
		return nil, errors.New("UV lock requires frozen project and storage boundaries")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	roots := append([]string{root}, trustedRoots...)
	for _, path := range roots {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || rejectSymlinkPathAllowMissing(path) != nil {
			return nil, errors.New("UV lock boundaries are not canonical")
		}
	}
	if !separateRoots(roots) {
		return nil, errors.New("UV lock project overlaps trusted storage")
	}
	guard, err := acquireProjectTransactionGuard(root, uvProjectTransactionLock)
	if err != nil {
		return nil, err
	}
	p := &uvProjectLockFiles{root: root, guard: guard, snapshots: map[string]*pypiFileSnapshot{}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, p.close())
			result = nil
		}
	}()
	for _, name := range []string{artifactpypi.UVProjectInputControl, artifactpypi.UVNativeLockControl} {
		body, info, err := readGoTransactionControl(guard.root, name)
		present := true
		if name == artifactpypi.UVNativeLockControl && errors.Is(err, os.ErrNotExist) {
			body, info, err, present = nil, nil, nil, false
		}
		if err != nil || present && !pypiSingleLink(info) {
			return nil, errors.New("UV lock original is not bounded single-link content")
		}
		control, err := domain.NewProjectControlFile(name, body, present)
		if err != nil {
			return nil, err
		}
		p.original = append(p.original, control)
		if present {
			snapshot, err := snapshotBoundedPromotionFile(filepath.Join(root, name), 4<<20)
			if err != nil || !os.SameFile(info, snapshot.info) || snapshot.Digest != control.Digest().String() {
				return nil, errors.New("UV lock original changed during capture")
			}
			p.snapshots[name] = &snapshot
		}
	}
	p.input, err = artifactpypi.ParseUVProjectInput(p.original[0].Body(), p.original[1].Body(), p.original[1].Present(), artifactpypi.UVProjectLockUpgrade, "")
	if err != nil {
		return nil, err
	}
	if p.original[1].Present() {
		if err := artifactpypi.ValidateUVProjectOriginalLock(p.input, p.original[1].Body()); err != nil {
			return nil, err
		}
	}
	if err := p.verify(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *uvProjectLockFiles) verifyManifest() error {
	if err := p.guard.verify(); err != nil {
		return err
	}
	root, err := os.Lstat(p.root)
	if err != nil || !os.SameFile(p.guard.rootInfo, root) || p.guard.rootInfo.Mode() != root.Mode() {
		return errors.New("UV lock project boundary changed")
	}
	if snapshot := p.snapshots[artifactpypi.UVProjectInputControl]; snapshot == nil || !snapshot.matches(filepath.Join(p.root, artifactpypi.UVProjectInputControl)) {
		return errors.New("UV lock manifest changed during transaction")
	}
	return nil
}
func (p *uvProjectLockFiles) verify(ctx context.Context) error {
	if ctx == nil {
		return errors.New("UV lock context is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.verifyManifest(); err != nil {
		return err
	}
	path := filepath.Join(p.root, artifactpypi.UVNativeLockControl)
	if snapshot := p.snapshots[artifactpypi.UVNativeLockControl]; snapshot != nil {
		if !snapshot.matches(path) {
			return errors.New("UV lock original changed during transaction")
		}
	} else if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.New("UV lock original appeared during transaction")
	}
	return nil
}
func (p *uvProjectLockFiles) close() error {
	if p.recovery {
		return errors.Join(p.guard.verify(), p.guard.file.Close(), p.guard.root.Close())
	}
	return p.guard.release()
}
func (p *uvProjectLockFiles) backupRecords() map[string]controlBackupRecord {
	records := map[string]controlBackupRecord{}
	for name, record := range p.records {
		if name != filepath.Join("selected", artifactpypi.UVNativeLockControl) || !p.published {
			records[name] = record
		}
	}
	if p.moved {
		records[artifactpypi.UVNativeLockControl] = controlBackupRecord{p.snapshots[artifactpypi.UVNativeLockControl].info, sha256.Sum256(p.original[1].Body())}
	}
	return records
}
func (p *uvProjectLockFiles) verifyBackup() error {
	if err := p.guard.verify(); err != nil {
		return err
	}
	return checkControlBackup(p.guard.root, p.backup, p.backupInfo, p.selectedDirectoryInfo, p.backupRecords())
}
func (p *uvProjectLockFiles) removeBackup() error {
	if err := p.guard.verify(); err != nil {
		return err
	}
	return removeControlBackup(p.guard.root, p.backup, p.backupInfo, p.selectedDirectoryInfo, p.backupRecords())
}
func (p *uvProjectLockFiles) writeOwned(name string, body []byte, mode os.FileMode) error {
	file, err := p.guard.root.OpenFile(filepath.Join(p.backup, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	n, writeErr := file.Write(body)
	modeErr := file.Chmod(mode)
	info, statErr := file.Stat()
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || n != len(body) || modeErr != nil || statErr != nil || syncErr != nil || closeErr != nil {
		p.recovery = true // Preserve uncertain partial writes and the original lock.
		return errors.Join(errors.New("UV lock private output is incomplete; recovery retained"), writeErr, modeErr, statErr, syncErr, closeErr)
	}
	p.records[name] = controlBackupRecord{info, sha256.Sum256(body)}
	return nil
}
func (p *uvProjectLockFiles) check(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.guard.verify(); err != nil {
		return err
	}
	if p.checkpoint != nil {
		return p.checkpoint(name)
	}
	return nil
}
func (p *uvProjectLockFiles) rollback() error {
	if err := p.guard.verify(); err != nil {
		return err
	}
	if p.published {
		record := p.records[filepath.Join("selected", artifactpypi.UVNativeLockControl)]
		body, info, err := readGoTransactionControl(p.guard.root, artifactpypi.UVNativeLockControl)
		if err != nil || !pypiSingleLink(info) || !os.SameFile(record.info, info) || record.info.Mode() != info.Mode() || sha256.Sum256(body) != record.digest {
			return errors.New("UV lock published file changed; original retained for recovery")
		}
		if err := renameRootNoReplace(p.guard.root, artifactpypi.UVNativeLockControl, filepath.Join(p.backup, "selected", artifactpypi.UVNativeLockControl)); err != nil {
			return err
		}
		p.published = false
	}
	if p.moved {
		backup := filepath.Join(p.root, p.backup, artifactpypi.UVNativeLockControl)
		if !p.snapshots[artifactpypi.UVNativeLockControl].matches(backup) {
			return errors.New("UV lock rollback original changed")
		}
		if err := renameRootNoReplace(p.guard.root, filepath.Join(p.backup, artifactpypi.UVNativeLockControl), artifactpypi.UVNativeLockControl); err != nil {
			return err
		}
		p.moved = false
	}
	return errors.Join(p.verify(context.Background()), syncGoTransactionRoot(p.guard.root))
}

// The caller must already hold exact current approval/readback. This primitive
// writes only the selected native lock, with the original manifest immutable.
func (p *uvProjectLockFiles) commit(ctx context.Context, selected domain.ProjectControlFile) (resultErr error) {
	if p.backup != "" || selected.Name() != artifactpypi.UVNativeLockControl || !selected.Present() {
		return errors.New("UV lock publication requires one selected native lock")
	}
	if err := p.verify(ctx); err != nil {
		return err
	}
	if _, err := artifactpypi.ParseUVNativeProjectLock(p.input, selected.Body()); err != nil {
		return err
	}
	identity, err := domain.NewRunID()
	if err != nil {
		return err
	}
	backup := ".heliopause-uv-lock-commit-" + identity.String()
	if err := p.guard.root.Mkdir(backup, 0o700); err != nil {
		return err
	}
	p.backup = backup
	p.backupInfo, err = p.guard.root.Lstat(p.backup)
	if err != nil {
		p.recovery = true
		return err
	}
	p.records = map[string]controlBackupRecord{}
	defer func() {
		if resultErr == nil || p.recovery {
			return
		}
		rollbackErr := p.rollback()
		if rollbackErr != nil {
			p.recovery = true
			resultErr = errors.Join(resultErr, rollbackErr)
			return
		}
		if err := p.removeBackup(); err != nil {
			p.recovery = true
			resultErr = errors.Join(resultErr, err)
		}
	}()
	if err := p.guard.root.Mkdir(filepath.Join(p.backup, "selected"), 0o700); err != nil {
		return err
	}
	p.selectedDirectoryInfo, err = p.guard.root.Lstat(filepath.Join(p.backup, "selected"))
	if err != nil {
		p.recovery = true
		return err
	}
	journal, err := json.Marshal(struct {
		Schema                       int
		Manifest, Original, Selected string
		OriginalPresent              bool
	}{1, p.original[0].Digest().String(), p.original[1].Digest().String(), selected.Digest().String(), p.original[1].Present()})
	if err != nil {
		return err
	}
	if err := p.writeOwned("journal.json", journal, 0o600); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if old := p.snapshots[artifactpypi.UVNativeLockControl]; old != nil {
		mode = old.Mode.Perm()
	}
	if err := p.writeOwned(filepath.Join("selected", artifactpypi.UVNativeLockControl), selected.Body(), mode); err != nil {
		return err
	}
	if err := p.check(ctx, "before-backup"); err != nil {
		return err
	}
	if err := p.verify(ctx); err != nil {
		return err
	}
	if p.original[1].Present() {
		if err := renameRootNoReplace(p.guard.root, artifactpypi.UVNativeLockControl, filepath.Join(p.backup, artifactpypi.UVNativeLockControl)); err != nil {
			return err
		}
		p.moved = true
	}
	if err := p.verifyBackup(); err != nil {
		return err
	}
	if err := p.check(ctx, "before-publish"); err != nil {
		return err
	}
	if err := p.verifyManifest(); err != nil {
		return err
	}
	if err := renameRootNoReplace(p.guard.root, filepath.Join(p.backup, "selected", artifactpypi.UVNativeLockControl), artifactpypi.UVNativeLockControl); err != nil {
		return err
	}
	p.published = true
	if err := p.check(ctx, "after-publish"); err != nil {
		return err
	}
	if err := p.verifyManifest(); err != nil {
		return err
	}
	body, info, err := readGoTransactionControl(p.guard.root, artifactpypi.UVNativeLockControl)
	record := p.records[filepath.Join("selected", artifactpypi.UVNativeLockControl)]
	if err != nil || !pypiSingleLink(info) || !os.SameFile(record.info, info) || record.info.Mode() != info.Mode() || sha256.Sum256(body) != record.digest {
		return errors.New("UV lock published bytes differ from approved selection")
	}
	if err := syncGoTransactionRoot(p.guard.root); err != nil {
		return err
	}
	if err := p.check(ctx, "before-cleanup"); err != nil {
		return err
	}
	if err := p.verifyBackup(); err != nil {
		return err
	}
	if err := p.removeBackup(); err != nil {
		p.recovery = true
		return err
	}
	// Backup disposal is complete. A final sync failure retains the guard and
	// reports failure rather than claiming an available rollback or installation.
	if err := syncGoTransactionRoot(p.guard.root); err != nil {
		p.recovery = true
		return err
	}
	return nil
}
