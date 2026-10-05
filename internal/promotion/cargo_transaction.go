package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const cargoTransactionMetadata = ".heliopause/cargo-transaction.json"

type cargoProjectPlan struct {
	root                 string
	cargoToml, cargoLock [32]byte
	rootInfo             os.FileInfo
	members              map[string]os.FileInfo
	controls             []domain.ProjectControlFile
	authorized           bool
}

func freezeCargoProject(root string) (result cargoProjectPlan, resultErr error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" || trustedExistingDirectory(root) != nil {
		return cargoProjectPlan{}, errors.New("cargo project root is untrusted")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return result, errors.New("cargo project root is unavailable")
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return result, errors.New("open anchored Cargo controls")
	}
	defer func() {
		resultErr = errors.Join(resultErr, directory.Close())
		if resultErr != nil {
			result = cargoProjectPlan{}
		}
	}()
	opened, err := directory.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return result, errors.New("cargo control root identity changed")
	}
	plan := cargoProjectPlan{root: root, rootInfo: info, members: map[string]os.FileInfo{}}
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		body, member, err := readGoTransactionControl(directory, name)
		present := true
		if name == "Cargo.lock" && errors.Is(err, os.ErrNotExist) {
			body, member, err, present = nil, nil, nil, false
		}
		if err != nil || (present && (member == nil || !pypiSingleLink(member) || len(body) == 0)) {
			return result, errors.New("cargo control is not bounded single-link content")
		}
		if name == "Cargo.toml" {
			if err := artifactcargo.ValidateProjectManifest(body, name); err != nil {
				return result, err
			}
			plan.cargoToml = sha256.Sum256(body)
		} else {
			plan.cargoLock = sha256.Sum256(body)
		}
		control, err := domain.NewProjectControlFile(name, body, present)
		if err != nil {
			return result, err
		}
		plan.controls = append(plan.controls, control)
		plan.members[name] = member
	}
	return plan, nil
}
func (p cargoProjectPlan) verifyUnchanged() error {
	current, err := freezeCargoProject(p.root)
	if err != nil || p.rootInfo == nil || !os.SameFile(p.rootInfo, current.rootInfo) || p.rootInfo.Mode() != current.rootInfo.Mode() || current.cargoToml != p.cargoToml || current.cargoLock != p.cargoLock {
		return errors.New("cargo project changed during transaction")
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		before, after := p.members[name], current.members[name]
		if (before == nil) != (after == nil) || (before != nil && (!os.SameFile(before, after) || before.Mode() != after.Mode())) {
			return errors.New("cargo control identity changed during transaction")
		}
	}
	return nil
}
func (p cargoProjectPlan) verifyManaged() error {
	if !p.authorized {
		return errors.New("cargo project requires independent complete approval")
	}
	return nil
}
func (p cargoProjectPlan) privateWorkspace() (string, error) {
	workspace, err := os.MkdirTemp(filepath.Dir(p.root), "."+filepath.Base(p.root)+".haa-cargo-work-")
	if err != nil {
		return "", errors.New("create Cargo private transaction workspace")
	}
	for _, control := range p.controls {
		if !control.Present() {
			continue
		}
		if os.WriteFile(filepath.Join(workspace, control.Name()), control.Body(), 0o600) != nil {
			_ = os.RemoveAll(workspace)
			return "", errors.New("copy Cargo transaction control files")
		}
	}
	return workspace, nil
}

type cargoProjectTransaction struct {
	checkpoint            func(string) error
	backupInfo            os.FileInfo
	plan                  cargoProjectPlan
	workspace             string
	backup                string
	moved                 map[string]bool
	published             map[string]bool
	metadataMoved         bool
	metadataPublished     bool
	metadataInfo          os.FileInfo
	originalMetadataInfo  os.FileInfo
	metadataDirectoryInfo os.FileInfo
	publishApproval       func() error
	rollbackApproval      func() error
	boundary              *os.Root
	selectedInfo          map[string]os.FileInfo
	expectedControls      []domain.ProjectControlFile
}

func beginCargoProjectTransaction(plan cargoProjectPlan, workspace string, expected ...domain.ProjectControlFile) (*cargoProjectTransaction, error) {
	if len(expected) != 0 && len(expected) != 2 {
		return nil, errors.New("cargo transaction selected controls are incomplete")
	}
	if err := plan.verifyUnchanged(); err != nil {
		return nil, err
	}
	if err := plan.verifyManaged(); err != nil {
		return nil, err
	}
	if err := trustedExistingDirectory(workspace); err != nil {
		return nil, errors.New("cargo private transaction workspace is untrusted")
	}
	backup, err := os.MkdirTemp(plan.root, ".heliopause-cargo-commit-")
	if err != nil {
		return nil, errors.New("create Cargo rollback transaction")
	}
	boundary, err := os.OpenRoot(plan.root)
	if err != nil {
		_ = os.RemoveAll(backup)
		return nil, errors.New("open Cargo transaction boundary")
	}
	opened, err := boundary.Stat(".")
	if err != nil || !os.SameFile(plan.rootInfo, opened) {
		_ = boundary.Close()
		_ = os.RemoveAll(backup)
		return nil, errors.New("cargo transaction root identity changed")
	}
	t := &cargoProjectTransaction{plan: plan, workspace: workspace, backup: backup, boundary: boundary, moved: map[string]bool{}, published: map[string]bool{}, selectedInfo: map[string]os.FileInfo{}}
	t.backupInfo, err = boundary.Lstat(filepath.Base(backup))
	if err != nil {
		return nil, errors.Join(err, boundary.Close())
	}
	t.expectedControls = append([]domain.ProjectControlFile(nil), expected...)
	if err := t.prepareSelectedControls(); err != nil {
		cleanupErr := boundary.RemoveAll(filepath.Base(backup))
		return nil, errors.Join(err, cleanupErr, boundary.Close())
	}
	return t, nil
}

func (t *cargoProjectTransaction) prepareSelectedControls() error {
	source, err := os.OpenRoot(t.workspace)
	if err != nil {
		return errors.New("open selected Cargo controls")
	}
	defer source.Close()
	directory := filepath.Join(filepath.Base(t.backup), "selected")
	if err := t.boundary.Mkdir(directory, 0o700); err != nil {
		return errors.New("create bounded Cargo publication directory")
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		body, member, err := readGoTransactionControl(source, name)
		if err != nil || !pypiSingleLink(member) {
			return errors.New("selected Cargo control is unavailable")
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
				return errors.New("cargo transaction selected controls differ from approved bytes")
			}
		}
		if name == "Cargo.toml" {
			if err := artifactcargo.ValidateProjectManifest(body, name); err != nil {
				return err
			}
		}
		f, err := t.boundary.OpenFile(filepath.Join(directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return errors.New("create selected Cargo control")
		}
		_, writeErr := f.Write(body)
		info, statErr := f.Stat()
		syncErr := f.Sync()
		closeErr := f.Close()
		if writeErr != nil || statErr != nil || syncErr != nil || closeErr != nil {
			return errors.New("persist selected Cargo control")
		}
		t.selectedInfo[name] = info
	}
	return nil
}

func syncCargoTransactionRoot(root *os.Root) error {
	f, err := root.Open(".")
	if err != nil {
		return errors.New("open Cargo transaction directory for sync")
	}
	return errors.Join(f.Sync(), f.Close())
}

func (t *cargoProjectTransaction) commit() (resultErr error) {
	defer func() { resultErr = errors.Join(resultErr, t.boundary.Close()) }()
	if err := t.check("before-backup"); err != nil {
		return t.fail(err)
	}
	if err := t.verifyBoundary(); err != nil {
		return t.fail(err)
	}
	if err := t.plan.verifyUnchanged(); err != nil {
		return t.fail(err)
	}
	if err := t.backupCurrent(); err != nil {
		return t.fail(err)
	}
	if err := t.check("before-publication"); err != nil {
		return t.fail(err)
	}
	if err := t.publishWorkspace(); err != nil {
		return t.fail(err)
	}
	if err := t.publishMetadata(); err != nil {
		return t.fail(err)
	}
	if err := t.check("before-approval"); err != nil {
		return t.fail(err)
	}
	if t.publishApproval != nil {
		if err := t.publishApproval(); err != nil {
			return t.fail(err)
		}
	}
	if err := syncCargoTransactionRoot(t.boundary); err != nil {
		return t.fail(errors.New("sync committed Cargo project"))
	}
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	if err := t.boundary.RemoveAll(filepath.Base(t.backup)); err != nil {
		return errors.New("remove committed Cargo rollback backup; project is fail-closed")
	}
	return syncCargoTransactionRoot(t.boundary)
}

func (t *cargoProjectTransaction) fail(cause error) error {
	rollbackErr := t.rollback()
	if rollbackErr != nil {
		return errors.Join(cause, rollbackErr, errors.New("cargo transaction rollback is incomplete; project is fail-closed"))
	}
	if err := t.boundary.RemoveAll(filepath.Base(t.backup)); err != nil {
		return errors.Join(cause, errors.New("remove rolled back Cargo transaction; project is fail-closed"))
	}
	return cause
}

func (t *cargoProjectTransaction) backupCurrent() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	if info, err := t.boundary.Lstat(".heliopause"); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return errors.New("cargo metadata directory is untrusted")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("cargo metadata directory is unavailable")
	}
	if info, err := t.boundary.Lstat(cargoTransactionMetadata); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("inspect current Cargo transaction metadata")
		}
		t.originalMetadataInfo = info
		if err := renameRootNoReplace(t.boundary, cargoTransactionMetadata, filepath.Join(filepath.Base(t.backup), "cargo-transaction.json")); err != nil {
			return errors.New("backup current Cargo transaction metadata")
		}
		t.metadataMoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect current Cargo transaction metadata")
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		body, info, err := readGoTransactionControl(t.boundary, name)
		if name == "Cargo.lock" && errors.Is(err, os.ErrNotExist) && t.plan.members[name] == nil {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("inspect current Cargo transaction member")
		}
		if !os.SameFile(t.plan.members[name], info) {
			return errors.New("cargo transaction original member identity changed")
		}
		matched := false
		actual, controlErr := domain.NewProjectControlFile(name, body, true)
		for _, expected := range t.plan.controls {
			if expected.Name() == name && expected.Digest() == actual.Digest() {
				matched = true
			}
		}
		if controlErr != nil || !matched || !pypiSingleLink(info) {
			return errors.New("cargo original control changed before backup")
		}
		if err := renameRootNoReplace(t.boundary, name, filepath.Join(filepath.Base(t.backup), name)); err != nil {
			return errors.New("backup current Cargo transaction member")
		}
		t.moved[name] = true
	}
	return nil
}

func (t *cargoProjectTransaction) publishWorkspace() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		selected := filepath.Join(filepath.Base(t.backup), "selected", name)
		body, info, err := readGoTransactionControl(t.boundary, selected)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("verified Cargo transaction output is unavailable")
		}
		if _, err := t.boundary.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			return errors.New("cargo transaction publication destination changed")
		}
		if !os.SameFile(t.selectedInfo[name], info) {
			return errors.New("selected Cargo transaction member identity changed")
		}
		if len(t.expectedControls) != 0 {
			actual, controlErr := domain.NewProjectControlFile(name, body, true)
			matched := false
			for _, expected := range t.expectedControls {
				if expected.Name() == name && expected.Digest() == actual.Digest() {
					matched = true
				}
			}
			if controlErr != nil || !matched || !pypiSingleLink(info) {
				return errors.New("cargo selected control changed before publication")
			}
		}
		if err := t.boundary.Link(selected, name); err != nil {
			return errors.New("publish verified Cargo transaction member")
		}
		t.published[name] = true
		if err := t.boundary.Remove(selected); err != nil {
			return errors.New("remove published Cargo transaction staging member")
		}
	}
	return nil
}

func (t *cargoProjectTransaction) publishMetadata() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	if err := t.boundary.Mkdir(".heliopause", 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("create Cargo transaction metadata directory")
	}
	info, err := t.boundary.Lstat(".heliopause")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("cargo transaction metadata directory is untrusted")
	}
	t.metadataDirectoryInfo = info
	current, err := freezeCargoProject(t.plan.root)
	if err != nil {
		return err
	}
	value := "{\"cargo_toml_sha256\":\"" + hex.EncodeToString(current.cargoToml[:]) + "\",\"cargo_lock_sha256\":\"" + hex.EncodeToString(current.cargoLock[:]) + "\"}\n"
	metadata, err := t.boundary.OpenRoot(".heliopause")
	if err != nil {
		return errors.New("open Cargo metadata boundary")
	}
	defer metadata.Close()
	opened, err := metadata.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("cargo metadata boundary identity changed")
	}
	f, err := metadata.OpenFile(".cargo-transaction.tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create Cargo metadata record")
	}
	defer metadata.Remove(".cargo-transaction.tmp")
	_, writeErr := f.Write([]byte(value))
	metadataInfo, statErr := f.Stat()
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || statErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("persist Cargo metadata record")
	}
	if err := metadata.Link(".cargo-transaction.tmp", "cargo-transaction.json"); err != nil {
		return errors.New("publish Cargo transaction metadata")
	}
	t.metadataPublished = true
	t.metadataInfo = metadataInfo
	if err := metadata.Remove(".cargo-transaction.tmp"); err != nil {
		return errors.New("remove Cargo metadata staging link")
	}
	return syncCargoTransactionRoot(metadata)
}

func (t *cargoProjectTransaction) rollback() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	var result error
	if t.rollbackApproval != nil {
		result = errors.Join(result, t.rollbackApproval())
	}
	if t.metadataPublished {
		info, err := t.boundary.Lstat(cargoTransactionMetadata)
		directory, directoryErr := t.boundary.Lstat(".heliopause")
		if err != nil || directoryErr != nil || !os.SameFile(t.metadataInfo, info) || !os.SameFile(t.metadataDirectoryInfo, directory) {
			result = errors.Join(result, errors.New("cargo rollback metadata identity changed"))
		} else if err := t.boundary.Remove(cargoTransactionMetadata); err != nil {
			result = errors.Join(result, errors.New("remove uncommitted Cargo transaction metadata"))
		}
	}
	if t.metadataMoved {
		backup := filepath.Join(filepath.Base(t.backup), "cargo-transaction.json")
		info, err := t.boundary.Lstat(backup)
		if err != nil || !os.SameFile(t.originalMetadataInfo, info) {
			result = errors.Join(result, errors.New("cargo rollback original metadata identity changed"))
		} else if err := t.boundary.Link(backup, cargoTransactionMetadata); err != nil {
			result = errors.Join(result, errors.New("restore Cargo transaction metadata"))
		} else if err := t.boundary.Remove(backup); err != nil {
			result = errors.Join(result, errors.New("remove restored Cargo metadata backup"))
		}
	}
	for _, name := range []string{"Cargo.lock", "Cargo.toml"} {
		if t.published[name] {
			info, err := t.boundary.Lstat(name)
			if err != nil || !os.SameFile(t.selectedInfo[name], info) {
				result = errors.Join(result, errors.New("cargo rollback published member identity changed"))
				continue
			}
			if err := t.boundary.Remove(name); err != nil {
				result = errors.Join(result, errors.New("remove uncommitted Cargo transaction member"))
			}
		}
		if t.moved[name] {
			backup := filepath.Join(filepath.Base(t.backup), name)
			info, err := t.boundary.Lstat(backup)
			if err != nil || !os.SameFile(t.plan.members[name], info) {
				result = errors.Join(result, errors.New("cargo rollback original member identity changed"))
				continue
			}
			if err := t.boundary.Link(backup, name); err != nil {
				result = errors.Join(result, errors.New("restore Cargo transaction member"))
			} else if err := t.boundary.Remove(backup); err != nil {
				result = errors.Join(result, errors.New("remove restored Cargo transaction backup member"))
			}
		}
	}
	if err := syncCargoTransactionRoot(t.boundary); err != nil {
		result = errors.Join(result, errors.New("sync rolled back Cargo transaction"))
	}
	return result
}

func (t *cargoProjectTransaction) check(phase string) error {
	if t.checkpoint != nil {
		return t.checkpoint(phase)
	}
	return nil
}
func (t *cargoProjectTransaction) verifyBoundary() error {
	root, err := os.Lstat(t.plan.root)
	backup, backupErr := t.boundary.Lstat(filepath.Base(t.backup))
	if err != nil || backupErr != nil || !os.SameFile(t.plan.rootInfo, root) || !os.SameFile(t.backupInfo, backup) || !backup.IsDir() || backup.Mode() != t.backupInfo.Mode() {
		return errors.New("cargo transaction boundary changed")
	}
	return nil
}
