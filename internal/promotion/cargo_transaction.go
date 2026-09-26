package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

const cargoTransactionMetadata = ".heliopause/cargo-transaction.json"

type cargoProjectPlan struct {
	root                 string
	cargoToml, cargoLock [32]byte
}

func freezeCargoProject(root string) (cargoProjectPlan, error) {
	if !filepath.IsAbs(root) || trustedExistingDirectory(root) != nil {
		return cargoProjectPlan{}, errors.New("cargo project root is untrusted")
	}
	toml, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil || len(toml) == 0 {
		return cargoProjectPlan{}, errors.New("cargo.toml is unavailable")
	}
	lock, err := os.ReadFile(filepath.Join(root, "Cargo.lock"))
	if err != nil || len(lock) == 0 {
		return cargoProjectPlan{}, errors.New("cargo.lock is unavailable")
	}
	return cargoProjectPlan{root: root, cargoToml: sha256.Sum256(toml), cargoLock: sha256.Sum256(lock)}, nil
}
func (p cargoProjectPlan) verifyUnchanged() error {
	current, err := freezeCargoProject(p.root)
	if err != nil || current.cargoToml != p.cargoToml || current.cargoLock != p.cargoLock {
		return errors.New("cargo project changed during transaction")
	}
	return nil
}
func (p cargoProjectPlan) verifyManaged() error {
	body, err := os.ReadFile(filepath.Join(p.root, cargoTransactionMetadata))
	if err != nil {
		return errors.New("cargo project is not HAA-managed")
	}
	want := "{\"cargo_toml_sha256\":\"" + hex.EncodeToString(p.cargoToml[:]) + "\",\"cargo_lock_sha256\":\"" + hex.EncodeToString(p.cargoLock[:]) + "\"}\n"
	if string(body) != want {
		return errors.New("cargo managed project metadata does not match current state")
	}
	return nil
}
func (p cargoProjectPlan) privateWorkspace() (string, error) {
	workspace, err := os.MkdirTemp(filepath.Dir(p.root), "."+filepath.Base(p.root)+".haa-cargo-work-")
	if err != nil {
		return "", errors.New("create Cargo private transaction workspace")
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		body, readErr := os.ReadFile(filepath.Join(p.root, name))
		if readErr != nil || os.WriteFile(filepath.Join(workspace, name), body, 0o600) != nil {
			_ = os.RemoveAll(workspace)
			return "", errors.New("copy Cargo transaction control files")
		}
	}
	return workspace, nil
}

type cargoProjectTransaction struct {
	plan                             cargoProjectPlan
	workspace, backup                string
	moved, published                 map[string]bool
	metadataMoved, metadataPublished bool
}

func beginCargoProjectTransaction(plan cargoProjectPlan, workspace string) (*cargoProjectTransaction, error) {
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
	return &cargoProjectTransaction{plan: plan, workspace: workspace, backup: backup, moved: map[string]bool{}, published: map[string]bool{}}, nil
}
func (t *cargoProjectTransaction) commit() error {
	if err := t.plan.verifyUnchanged(); err != nil {
		return err
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
	if err := syncDirectory(t.plan.root); err != nil {
		return t.fail(errors.New("sync committed Cargo project"))
	}
	if err := os.RemoveAll(t.backup); err != nil {
		return errors.New("remove committed Cargo rollback backup; project is fail-closed")
	}
	return nil
}
func (t *cargoProjectTransaction) fail(cause error) error {
	if rollbackErr := t.rollback(); rollbackErr != nil {
		return errors.Join(cause, rollbackErr, errors.New("cargo transaction rollback is incomplete; project is fail-closed"))
	}
	if err := os.RemoveAll(t.backup); err != nil {
		return errors.Join(cause, errors.New("remove rolled back Cargo transaction; project is fail-closed"))
	}
	return cause
}
func (t *cargoProjectTransaction) backupCurrent() error {
	metadata := filepath.Join(t.plan.root, cargoTransactionMetadata)
	if info, err := os.Lstat(metadata); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || os.Rename(metadata, filepath.Join(t.backup, "cargo-transaction.json")) != nil {
			return errors.New("backup current Cargo transaction metadata")
		}
		t.metadataMoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect current Cargo transaction metadata")
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		info, err := os.Lstat(filepath.Join(t.plan.root, name))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || os.Rename(filepath.Join(t.plan.root, name), filepath.Join(t.backup, name)) != nil {
			return errors.New("backup current Cargo transaction member")
		}
		t.moved[name] = true
	}
	return nil
}
func (t *cargoProjectTransaction) publishWorkspace() error {
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		info, err := os.Lstat(filepath.Join(t.workspace, name))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || os.Rename(filepath.Join(t.workspace, name), filepath.Join(t.plan.root, name)) != nil {
			return errors.New("publish verified Cargo transaction member")
		}
		t.published[name] = true
	}
	return nil
}
func (t *cargoProjectTransaction) publishMetadata() error {
	directory := filepath.Join(t.plan.root, ".heliopause")
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("create Cargo transaction metadata directory")
	}
	if err := trustedExistingDirectory(directory); err != nil {
		return errors.New("cargo transaction metadata directory is untrusted")
	}
	current, err := freezeCargoProject(t.plan.root)
	if err != nil {
		return err
	}
	value := "{\"cargo_toml_sha256\":\"" + hex.EncodeToString(current.cargoToml[:]) + "\",\"cargo_lock_sha256\":\"" + hex.EncodeToString(current.cargoLock[:]) + "\"}\n"
	if err := writeNPMTransactionMetadata(directory, filepath.Join(directory, "cargo-transaction.json"), []byte(value)); err != nil {
		return errors.New("publish Cargo transaction metadata")
	}
	t.metadataPublished = true
	return nil
}
func (t *cargoProjectTransaction) rollback() error {
	var result error
	if t.metadataPublished {
		if err := os.Remove(filepath.Join(t.plan.root, cargoTransactionMetadata)); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, errors.New("remove uncommitted Cargo transaction metadata"))
		}
	}
	if t.metadataMoved {
		if err := os.Rename(filepath.Join(t.backup, "cargo-transaction.json"), filepath.Join(t.plan.root, cargoTransactionMetadata)); err != nil {
			result = errors.Join(result, errors.New("restore Cargo transaction metadata"))
		}
	}
	for _, name := range []string{"Cargo.lock", "Cargo.toml"} {
		if t.published[name] {
			if err := os.Remove(filepath.Join(t.plan.root, name)); err != nil {
				result = errors.Join(result, errors.New("remove uncommitted Cargo transaction member"))
			}
		}
		if t.moved[name] {
			if err := os.Rename(filepath.Join(t.backup, name), filepath.Join(t.plan.root, name)); err != nil {
				result = errors.Join(result, errors.New("restore Cargo transaction member"))
			}
		}
	}
	if err := syncDirectory(t.plan.root); err != nil {
		result = errors.Join(result, errors.New("sync rolled back Cargo transaction"))
	}
	return result
}
