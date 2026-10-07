package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const goTransactionMetadata = ".heliopause/go-transaction.json"

// goProjectPlan freezes the complete Go control-file transaction surface.
type goProjectPlan struct {
	root       string
	goModHash  [32]byte
	goSumHash  [32]byte
	rootInfo   os.FileInfo
	members    map[string]os.FileInfo
	controls   []domain.ProjectControlFile
	authorized bool
}

type goProjectGuard struct {
	path     string
	root     *os.Root
	rootInfo os.FileInfo
	fileInfo os.FileInfo
}

func acquireGoProjectGuard(root string) (goProjectGuard, error) {
	return acquireProjectTransactionGuard(root, ".heliopause-go-transaction.lock")
}

func acquireProjectTransactionGuard(root, lockName string) (goProjectGuard, error) {
	if lockName == "" || filepath.Base(lockName) != lockName || lockName == "." || lockName == ".." {
		return goProjectGuard{}, errors.New("project transaction lock name is invalid")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" || trustedExistingDirectory(root) != nil {
		return goProjectGuard{}, errors.New("go project root is untrusted")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return goProjectGuard{}, errors.New("go project root is unavailable")
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return goProjectGuard{}, errors.New("open Go project guard root")
	}
	opened, err := directory.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = directory.Close()
		return goProjectGuard{}, errors.New("go project root identity changed")
	}
	path := filepath.Join(root, lockName)
	file, err := directory.OpenFile(filepath.Base(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = directory.Close()
		return goProjectGuard{}, errors.New("go project is already being mutated or lock is unavailable")
	}
	fileInfo, statErr := file.Stat()
	if err := file.Close(); err != nil || statErr != nil {
		_ = directory.Remove(filepath.Base(path))
		_ = directory.Close()
		return goProjectGuard{}, errors.New("close Go project transaction lock")
	}
	return goProjectGuard{path, directory, info, fileInfo}, nil
}

func (g goProjectGuard) release() (resultErr error) {
	if g.path == "" || g.root == nil {
		return errors.New("go project transaction lock is unavailable")
	}
	defer func() { resultErr = errors.Join(resultErr, g.root.Close()) }()
	if err := g.verify(); err != nil {
		return err
	}
	if err := g.root.Remove(filepath.Base(g.path)); err != nil {
		return errors.New("remove exact Go project transaction lock")
	}
	return nil
}

func (g goProjectGuard) verify() error {
	rootInfo, rootErr := os.Lstat(filepath.Dir(g.path))
	info, err := g.root.Lstat(filepath.Base(g.path))
	if rootErr != nil || err != nil || !os.SameFile(g.rootInfo, rootInfo) || !os.SameFile(g.fileInfo, info) || !info.Mode().IsRegular() {
		return errors.New("go project guard identity changed")
	}
	return nil
}

func freezeGoProject(root string) (goProjectPlan, error) {
	if !filepath.IsAbs(root) || trustedExistingDirectory(root) != nil {
		return goProjectPlan{}, errors.New("go project root is untrusted")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return goProjectPlan{}, errors.New("go project root is unavailable")
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return goProjectPlan{}, errors.New("open Go project control root")
	}
	defer directory.Close()
	opened, err := directory.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return goProjectPlan{}, errors.New("go project root identity changed")
	}
	plan := goProjectPlan{root: root, rootInfo: info, members: map[string]os.FileInfo{}}
	for _, name := range []string{"go.mod", "go.sum"} {
		body, fileInfo, err := readGoTransactionControl(directory, name)
		present := true
		if name == "go.sum" && errors.Is(err, os.ErrNotExist) {
			body, fileInfo, err, present = nil, nil, nil, false
		}
		if err != nil {
			return goProjectPlan{}, errors.New("go project control file is unavailable")
		}
		if name == "go.mod" {
			if err := artifactgo.ValidateProjectMod(body); err != nil {
				return goProjectPlan{}, err
			}
			plan.goModHash = sha256.Sum256(body)
		} else {
			plan.goSumHash = sha256.Sum256(body)
		}
		control, err := domain.NewProjectControlFile(name, body, present)
		if err != nil {
			return goProjectPlan{}, err
		}
		plan.controls = append(plan.controls, control)
		plan.members[name] = fileInfo
	}
	return plan, nil
}

func readGoTransactionControl(root *os.Root, name string) ([]byte, os.FileInfo, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > artifactgo.MaxProjectControlBytes {
		return nil, nil, errors.New("go transaction member is not bounded regular content")
	}
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, errors.New("open Go transaction member")
	}
	opened, statErr := file.Stat()
	if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		_ = file.Close()
		return nil, nil, errors.New("go transaction member identity changed")
	}
	body, readErr := io.ReadAll(io.LimitReader(file, artifactgo.MaxProjectControlBytes+1))
	after, afterErr := file.Stat()
	closeErr := file.Close()
	current, currentErr := root.Lstat(name)
	if readErr != nil || afterErr != nil || closeErr != nil || currentErr != nil || !os.SameFile(info, current) || int64(len(body)) != info.Size() || after.Size() != info.Size() || after.ModTime() != info.ModTime() {
		return nil, nil, errors.New("go transaction member changed during read")
	}
	return body, info, nil
}

func (p goProjectPlan) verifyUnchanged() error {
	current, err := freezeGoProject(p.root)
	if err != nil || !os.SameFile(p.rootInfo, current.rootInfo) || current.goModHash != p.goModHash || current.goSumHash != p.goSumHash {
		return errors.New("go project changed during transaction")
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		before, after := p.members[name], current.members[name]
		if (before == nil) != (after == nil) || before != nil && !os.SameFile(before, after) {
			return errors.New("go project control identity changed during transaction")
		}
	}
	return nil
}

func (p goProjectPlan) verifyManaged() error {
	if !p.authorized {
		return errors.New("go project requires trusted approval or dependency-free adoption")
	}
	return nil
}

func (p goProjectPlan) privateWorkspace() (string, error) {
	workspace, err := os.MkdirTemp(filepath.Dir(p.root), "."+filepath.Base(p.root)+".haa-go-work-")
	if err != nil {
		return "", errors.New("create Go private transaction workspace")
	}
	for _, control := range p.controls {
		if os.WriteFile(filepath.Join(workspace, control.Name()), control.Body(), 0o600) != nil {
			_ = os.RemoveAll(workspace)
			return "", errors.New("copy Go transaction control files")
		}
	}
	return workspace, nil
}

type goProjectTransaction struct {
	plan                  goProjectPlan
	workspace             string
	backup                string
	moved                 map[string]bool
	published             map[string]bool
	metadataMoved         bool
	metadataPublished     bool
	metadataInfo          os.FileInfo
	originalMetadataInfo  os.FileInfo
	originalMetadataHash  [32]byte
	selectedDirectoryInfo os.FileInfo
	selectedHashes        map[string][32]byte
	metadataDirectoryInfo os.FileInfo
	publishApproval       func() error
	rollbackApproval      func() error
	boundary              *os.Root
	backupInfo            os.FileInfo
	selectedInfo          map[string]os.FileInfo
	expectedControls      []domain.ProjectControlFile
}

func beginGoProjectTransaction(plan goProjectPlan, workspace string, expected ...domain.ProjectControlFile) (*goProjectTransaction, error) {
	if len(expected) != 0 && len(expected) != 2 {
		return nil, errors.New("go transaction selected controls are incomplete")
	}
	if err := plan.verifyUnchanged(); err != nil {
		return nil, err
	}
	if err := plan.verifyManaged(); err != nil {
		return nil, err
	}
	if err := trustedExistingDirectory(workspace); err != nil {
		return nil, errors.New("go private transaction workspace is untrusted")
	}
	backup, err := os.MkdirTemp(plan.root, ".heliopause-go-commit-")
	if err != nil {
		return nil, errors.New("create Go rollback transaction")
	}
	boundary, err := os.OpenRoot(plan.root)
	if err != nil {
		_ = os.RemoveAll(backup)
		return nil, errors.New("open Go transaction boundary")
	}
	opened, err := boundary.Stat(".")
	if err != nil || !os.SameFile(plan.rootInfo, opened) {
		_ = boundary.Close()
		_ = os.RemoveAll(backup)
		return nil, errors.New("go transaction root identity changed")
	}
	t := &goProjectTransaction{plan: plan, workspace: workspace, backup: backup, boundary: boundary, moved: map[string]bool{}, published: map[string]bool{}, selectedInfo: map[string]os.FileInfo{}, selectedHashes: map[string][32]byte{}}
	t.backupInfo, err = boundary.Lstat(filepath.Base(backup))
	if err != nil {
		return nil, errors.Join(err, boundary.Close())
	}
	t.expectedControls = append([]domain.ProjectControlFile(nil), expected...)
	if err := t.prepareSelectedControls(); err != nil {
		cleanupErr := t.removeBackup(false)
		return nil, errors.Join(err, cleanupErr, boundary.Close())
	}
	return t, nil
}

func (t *goProjectTransaction) prepareSelectedControls() error {
	source, err := os.OpenRoot(t.workspace)
	if err != nil {
		return errors.New("open selected Go controls")
	}
	defer source.Close()
	directory := filepath.Join(filepath.Base(t.backup), "selected")
	if err := t.boundary.Mkdir(directory, 0o700); err != nil {
		return errors.New("create bounded Go publication directory")
	}
	t.selectedDirectoryInfo, err = t.boundary.Lstat(directory)
	if err != nil {
		return err
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		body, _, err := readGoTransactionControl(source, name)
		if err != nil {
			return errors.New("selected Go control is unavailable")
		}
		if len(t.expectedControls) != 0 {
			actual, err := domain.NewProjectControlFile(name, body, true)
			matched := false
			for _, wanted := range t.expectedControls {
				if wanted.Name() == name && wanted.Present() && wanted.Digest() == actual.Digest() {
					matched = true
				}
			}
			if err != nil || !matched {
				return errors.New("go transaction selected controls differ from approved bytes")
			}
		}
		if name == "go.mod" {
			if err := artifactgo.ValidateProjectMod(body); err != nil {
				return err
			}
		}
		f, err := t.boundary.OpenFile(filepath.Join(directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return errors.New("create selected Go control")
		}
		_, writeErr := f.Write(body)
		info, statErr := f.Stat()
		syncErr := f.Sync()
		closeErr := f.Close()
		if writeErr != nil || statErr != nil || syncErr != nil || closeErr != nil {
			return errors.New("persist selected Go control")
		}
		t.selectedInfo[name] = info
		t.selectedHashes[name] = sha256.Sum256(body)
	}
	return nil
}

func syncGoTransactionRoot(root *os.Root) error {
	f, err := root.Open(".")
	if err != nil {
		return errors.New("open Go transaction directory for sync")
	}
	return errors.Join(f.Sync(), f.Close())
}

func (t *goProjectTransaction) commit() (resultErr error) {
	defer func() { resultErr = errors.Join(resultErr, t.boundary.Close()) }()
	if err := t.plan.verifyUnchanged(); err != nil {
		return t.fail(err)
	}
	if err := t.backupCurrent(); err != nil {
		return t.fail(err)
	}
	if err := t.publishWorkspace(); err != nil {
		return t.fail(err)
	}
	if err := t.publishMetadata(); err != nil {
		return t.fail(err)
	}
	if t.publishApproval != nil {
		if err := t.publishApproval(); err != nil {
			return t.fail(err)
		}
	}
	if err := syncGoTransactionRoot(t.boundary); err != nil {
		return t.fail(errors.New("sync committed Go project"))
	}
	if err := t.removeBackup(true); err != nil {
		return errors.New("remove committed Go rollback backup; project is fail-closed")
	}
	return syncGoTransactionRoot(t.boundary)
}

func (t *goProjectTransaction) fail(cause error) error {
	rollbackErr := t.rollback()
	if rollbackErr != nil {
		return errors.Join(cause, rollbackErr, errors.New("go transaction rollback is incomplete; project is fail-closed"))
	}
	if err := t.removeBackup(false); err != nil {
		return errors.Join(cause, errors.New("remove rolled back Go transaction; project is fail-closed"))
	}
	return cause
}

func (t *goProjectTransaction) backupCurrent() error {
	if info, err := t.boundary.Lstat(".heliopause"); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return errors.New("go metadata directory is untrusted")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("go metadata directory is unavailable")
	}
	if body, info, err := readGoTransactionControl(t.boundary, goTransactionMetadata); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("inspect current Go transaction metadata")
		}
		t.originalMetadataInfo = info
		t.originalMetadataHash = sha256.Sum256(body)
		if err := renameRootNoReplace(t.boundary, goTransactionMetadata, filepath.Join(filepath.Base(t.backup), "go-transaction.json")); err != nil {
			return errors.New("backup current Go transaction metadata")
		}
		t.metadataMoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect current Go transaction metadata")
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		info, err := t.boundary.Lstat(name)
		if name == "go.sum" && errors.Is(err, os.ErrNotExist) && t.plan.members[name] == nil {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("inspect current Go transaction member")
		}
		if !os.SameFile(t.plan.members[name], info) {
			return errors.New("go transaction original member identity changed")
		}
		if err := renameRootNoReplace(t.boundary, name, filepath.Join(filepath.Base(t.backup), name)); err != nil {
			return errors.New("backup current Go transaction member")
		}
		t.moved[name] = true
	}
	return nil
}

func (t *goProjectTransaction) publishWorkspace() error {
	for _, name := range []string{"go.mod", "go.sum"} {
		selected := filepath.Join(filepath.Base(t.backup), "selected", name)
		info, err := t.boundary.Lstat(selected)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("verified Go transaction output is unavailable")
		}
		if _, err := t.boundary.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			return errors.New("go transaction publication destination changed")
		}
		if !os.SameFile(t.selectedInfo[name], info) {
			return errors.New("selected Go transaction member identity changed")
		}
		if err := t.boundary.Link(selected, name); err != nil {
			return errors.New("publish verified Go transaction member")
		}
		t.published[name] = true
		if err := t.boundary.Remove(selected); err != nil {
			return errors.New("remove published Go transaction staging member")
		}
	}
	return nil
}

func (t *goProjectTransaction) publishMetadata() error {
	if err := t.boundary.Mkdir(".heliopause", 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("create Go transaction metadata directory")
	}
	info, err := t.boundary.Lstat(".heliopause")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("go transaction metadata directory is untrusted")
	}
	t.metadataDirectoryInfo = info
	current, err := freezeGoProject(t.plan.root)
	if err != nil {
		return err
	}
	value := "{\"go_mod_sha256\":\"" + hex.EncodeToString(current.goModHash[:]) + "\",\"go_sum_sha256\":\"" + hex.EncodeToString(current.goSumHash[:]) + "\"}\n"
	metadata, err := t.boundary.OpenRoot(".heliopause")
	if err != nil {
		return errors.New("open Go metadata boundary")
	}
	defer metadata.Close()
	opened, err := metadata.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("go metadata boundary identity changed")
	}
	f, err := metadata.OpenFile(".go-transaction.tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create Go metadata record")
	}
	defer metadata.Remove(".go-transaction.tmp")
	_, writeErr := f.Write([]byte(value))
	metadataInfo, statErr := f.Stat()
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || statErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("persist Go metadata record")
	}
	if err := metadata.Rename(".go-transaction.tmp", "go-transaction.json"); err != nil {
		return errors.New("publish Go transaction metadata")
	}
	t.metadataPublished = true
	t.metadataInfo = metadataInfo
	return syncGoTransactionRoot(metadata)
}

func (t *goProjectTransaction) rollback() error {
	if err := t.verifyBackupBoundary(); err != nil {
		return err
	}
	var result error
	if t.rollbackApproval != nil {
		result = errors.Join(result, t.rollbackApproval())
	}
	if t.metadataPublished {
		info, err := t.boundary.Lstat(goTransactionMetadata)
		directory, directoryErr := t.boundary.Lstat(".heliopause")
		if err != nil || directoryErr != nil || !os.SameFile(t.metadataInfo, info) || !os.SameFile(t.metadataDirectoryInfo, directory) {
			result = errors.Join(result, errors.New("go rollback metadata identity changed"))
		} else if err := t.boundary.Remove(goTransactionMetadata); err != nil {
			result = errors.Join(result, errors.New("remove uncommitted Go transaction metadata"))
		}
	}
	if t.metadataMoved {
		backup := filepath.Join(filepath.Base(t.backup), "go-transaction.json")
		info, err := t.boundary.Lstat(backup)
		if err != nil || !os.SameFile(t.originalMetadataInfo, info) || controlBackupUnchanged(t.boundary, backup, controlBackupRecord{t.originalMetadataInfo, t.originalMetadataHash}) != nil {
			result = errors.Join(result, errors.New("go rollback original metadata identity changed"))
		} else if err := t.boundary.Link(backup, goTransactionMetadata); err != nil {
			result = errors.Join(result, errors.New("restore Go transaction metadata"))
		} else if err := t.boundary.Remove(backup); err != nil {
			result = errors.Join(result, errors.New("remove restored Go metadata backup"))
		}
	}
	for _, name := range []string{"go.sum", "go.mod"} {
		if t.published[name] {
			info, err := t.boundary.Lstat(name)
			if err != nil || !os.SameFile(t.selectedInfo[name], info) || controlBackupUnchanged(t.boundary, name, controlBackupRecord{t.selectedInfo[name], t.selectedHashes[name]}) != nil {
				result = errors.Join(result, errors.New("go rollback published member identity changed"))
				continue
			}
			if err := t.boundary.Remove(name); err != nil {
				result = errors.Join(result, errors.New("remove uncommitted Go transaction member"))
			}
		}
		if t.moved[name] {
			backup := filepath.Join(filepath.Base(t.backup), name)
			info, err := t.boundary.Lstat(backup)
			if err != nil || !os.SameFile(t.plan.members[name], info) || controlBackupUnchanged(t.boundary, backup, t.backupRecords(true)[name]) != nil {
				result = errors.Join(result, errors.New("go rollback original member identity changed"))
				continue
			}
			if err := t.boundary.Link(backup, name); err != nil {
				result = errors.Join(result, errors.New("restore Go transaction member"))
			} else if err := t.boundary.Remove(backup); err != nil {
				result = errors.Join(result, errors.New("remove restored Go transaction backup member"))
			}
		}
	}
	if err := syncGoTransactionRoot(t.boundary); err != nil {
		result = errors.Join(result, errors.New("sync rolled back Go transaction"))
	}
	return result
}
