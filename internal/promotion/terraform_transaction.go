package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const terraformTransactionMetadata = ".heliopause/terraform-transaction.json"

type terraformProjectPlan struct {
	root       string
	lockHash   [32]byte
	rootInfo   os.FileInfo
	lockInfo   os.FileInfo
	authorized bool
}

func freezeTerraformProject(root string) (terraformProjectPlan, error) {
	if !filepath.IsAbs(root) || trustedExistingDirectory(root) != nil {
		return terraformProjectPlan{}, errors.New("terraform project root is untrusted")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return terraformProjectPlan{}, errors.New("terraform root is unavailable")
	}
	boundary, err := os.OpenRoot(root)
	if err != nil {
		return terraformProjectPlan{}, err
	}
	defer boundary.Close()
	body, lockInfo, err := readGoTransactionControl(boundary, ".terraform.lock.hcl")
	if errors.Is(err, os.ErrNotExist) {
		body, lockInfo, err = nil, nil, nil
	}
	if err != nil || lockInfo != nil && !pypiSingleLink(lockInfo) {
		return terraformProjectPlan{}, errors.New("terraform lock file is unavailable")
	}
	opened, err := boundary.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return terraformProjectPlan{}, errors.New("terraform root changed")
	}
	return terraformProjectPlan{root: root, lockHash: sha256.Sum256(body), rootInfo: info, lockInfo: lockInfo}, nil
}
func (p terraformProjectPlan) verifyUnchanged() error {
	current, err := freezeTerraformProject(p.root)
	if err != nil || current.lockHash != p.lockHash || !os.SameFile(p.rootInfo, current.rootInfo) || (p.lockInfo == nil) != (current.lockInfo == nil) || p.lockInfo != nil && !os.SameFile(p.lockInfo, current.lockInfo) {
		return errors.New("terraform lock file changed during transaction")
	}
	return nil
}
func (p terraformProjectPlan) verifyManaged() error {
	if !p.authorized {
		return errors.New("terraform publication requires independent complete approval")
	}
	return nil
}
func (p terraformProjectPlan) privateWorkspace() (string, error) {
	workspace, err := os.MkdirTemp(filepath.Dir(p.root), "."+filepath.Base(p.root)+".haa-terraform-work-")
	if err != nil {
		return "", errors.New("create Terraform private transaction workspace")
	}
	boundary, readErr := os.OpenRoot(p.root)
	if readErr != nil {
		_ = os.RemoveAll(workspace)
		return "", readErr
	}
	defer boundary.Close()
	body, _, readErr := readGoTransactionControl(boundary, ".terraform.lock.hcl")
	if p.lockInfo == nil && errors.Is(readErr, os.ErrNotExist) {
		body, readErr = nil, nil
	}
	if readErr != nil || os.WriteFile(filepath.Join(workspace, ".terraform.lock.hcl"), body, 0o600) != nil {
		_ = os.RemoveAll(workspace)
		return "", errors.New("copy Terraform lock file")
	}
	return workspace, nil
}

type terraformProjectTransaction struct {
	plan                                                      terraformProjectPlan
	workspace, backup                                         string
	moved, published, metadataMoved, metadataPublished        bool
	boundary                                                  *os.Root
	backupInfo                                                os.FileInfo
	selectedInfo                                              os.FileInfo
	selectedDirectoryInfo                                     os.FileInfo
	expectedLock                                              []byte
	providers                                                 bool
	providersMoved, providersPublished                        bool
	originalTerraformInfo, selectedTerraformInfo              os.FileInfo
	providerRecords                                           []terraformFileRecord
	providerMembers                                           map[string]os.FileInfo
	metadataInfo, originalMetadataInfo, metadataDirectoryInfo os.FileInfo
	metadataValue                                             []byte
	metadataDirectoryCreated                                  bool
	originalMetadataValue                                     []byte
	originalMetadataFrozen                                    bool
	originalProviderRecords                                   []terraformFileRecord
	originalProviderMembers                                   map[string]os.FileInfo
	checkpoint                                                func(string) error
	publishApproval, rollbackApproval                         func() error
}

func beginTerraformProjectTransaction(plan terraformProjectPlan, workspace string) (*terraformProjectTransaction, error) {
	if err := plan.verifyUnchanged(); err != nil {
		return nil, err
	}
	if err := plan.verifyManaged(); err != nil {
		return nil, err
	}
	if err := trustedExistingDirectory(workspace); err != nil {
		return nil, errors.New("terraform private transaction workspace is untrusted")
	}
	backup, err := os.MkdirTemp(plan.root, ".heliopause-terraform-commit-")
	if err != nil {
		return nil, errors.New("create Terraform rollback transaction")
	}
	boundary, err := os.OpenRoot(plan.root)
	if err != nil {
		_ = os.RemoveAll(backup)
		return nil, err
	}
	backupInfo, err := boundary.Lstat(filepath.Base(backup))
	if err != nil {
		_ = boundary.Close()
		return nil, err
	}

	selected := filepath.Join(backup, "selected")
	if err := os.Mkdir(selected, 0o755); err != nil {
		_ = boundary.Close()
		return nil, err
	}
	source, err := os.OpenRoot(workspace)
	if err != nil {
		_ = boundary.Close()
		return nil, err
	}
	body, si, err := readGoTransactionControl(source, ".terraform.lock.hcl")
	_ = source.Close()
	if err != nil || !pypiSingleLink(si) {
		_ = boundary.Close()
		return nil, errors.New("terraform selected lock is invalid")
	}
	if err := writeRecord(selected, ".terraform.lock.hcl", body); err != nil {
		_ = boundary.Close()
		return nil, err
	}
	_, selectedInfo, err := readGoTransactionControl(boundary, filepath.Join(filepath.Base(backup), "selected", ".terraform.lock.hcl"))
	if err != nil {
		_ = boundary.Close()
		return nil, err
	}
	selectedDirectoryInfo, err := boundary.Lstat(filepath.Join(filepath.Base(backup), "selected"))
	if err != nil {
		_ = boundary.Close()
		return nil, err
	}
	return &terraformProjectTransaction{plan: plan, workspace: workspace, backup: backup, boundary: boundary, backupInfo: backupInfo, selectedInfo: selectedInfo, selectedDirectoryInfo: selectedDirectoryInfo, expectedLock: body}, nil
}
func (t *terraformProjectTransaction) commit() error {
	defer t.boundary.Close()
	if err := t.plan.verifyUnchanged(); err != nil {
		return err
	}
	if err := t.check("before-backup"); err != nil {
		return t.fail(err)
	}
	if err := t.backupCurrent(); err != nil {
		return t.fail(err)
	}
	if err := t.verifyBackupContents(true); err != nil {
		return t.fail(err)
	}
	if err := syncTree(t.backup); err != nil {
		return t.fail(err)
	}
	if err := syncDirectory(t.plan.root); err != nil {
		return t.fail(err)
	}
	if err := t.publishWorkspace(); err != nil {
		return t.fail(err)
	}
	if err := t.check("after-provider-publication"); err != nil {
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
	if err := t.check("after-approval"); err != nil {
		return t.fail(err)
	}
	if err := syncDirectory(t.plan.root); err != nil {
		return t.fail(errors.New("sync committed Terraform project"))
	}
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	if err := t.verifyBackupContents(true); err != nil {
		return t.fail(err)
	}
	if err := t.boundary.RemoveAll(filepath.Base(t.backup)); err != nil {
		return errors.New("remove committed Terraform rollback backup; project is fail-closed")
	}
	return syncDirectory(t.plan.root)
}
func (t *terraformProjectTransaction) fail(cause error) error {
	if rollbackErr := t.rollback(); rollbackErr != nil {
		return errors.Join(cause, rollbackErr, errors.New("terraform transaction rollback is incomplete; project is fail-closed"))
	}
	if err := t.verifyBackupContents(false); err != nil {
		return errors.Join(cause, err, errors.New("terraform rollback backup is uncertain; project is fail-closed"))
	}
	if err := t.boundary.RemoveAll(filepath.Base(t.backup)); err != nil {
		return errors.Join(cause, errors.New("remove rolled back Terraform transaction; project is fail-closed"))
	}
	return errors.Join(cause, syncDirectory(t.plan.root))
}
func (t *terraformProjectTransaction) backupCurrent() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	if body, info, err := readGoTransactionControl(t.boundary, terraformTransactionMetadata); err == nil {
		if t.originalMetadataFrozen && (t.originalMetadataInfo == nil || !os.SameFile(info, t.originalMetadataInfo) || !bytes.Equal(body, t.originalMetadataValue) || info.Mode() != t.originalMetadataInfo.Mode()) {
			return errors.New("terraform original metadata drifted before backup")
		}
		t.originalMetadataInfo = info
		t.originalMetadataValue = body
		if !info.Mode().IsRegular() || !pypiSingleLink(info) || renameRootNoReplace(t.boundary, terraformTransactionMetadata, filepath.Join(filepath.Base(t.backup), "terraform-transaction.json")) != nil {
			return errors.New("backup Terraform transaction metadata")
		}
		t.metadataMoved = true
	} else if !errors.Is(err, os.ErrNotExist) || t.originalMetadataFrozen && t.originalMetadataInfo != nil {
		return errors.New("inspect Terraform transaction metadata")
	}
	if t.plan.lockInfo != nil {
		body, info, err := readGoTransactionControl(t.boundary, ".terraform.lock.hcl")
		if err != nil || !os.SameFile(info, t.plan.lockInfo) || sha256.Sum256(body) != t.plan.lockHash || info.Mode() != t.plan.lockInfo.Mode() {
			return errors.New("terraform original lock drifted before backup")
		}
		if err := renameRootNoReplace(t.boundary, ".terraform.lock.hcl", filepath.Join(filepath.Base(t.backup), ".terraform.lock.hcl")); err != nil {
			return errors.New("backup Terraform lock file")
		}
		t.moved = true
	}
	if t.providers && t.originalTerraformInfo != nil {
		info, files, members, err := inspectTerraformInstallation(context.Background(), t.boundary, t.plan.root)
		if err != nil || !os.SameFile(info, t.originalTerraformInfo) || !sameTerraformFiles(files, t.originalProviderRecords) || !sameTerraformMembers(members, t.originalProviderMembers) {
			return errors.New("terraform provider destination identity changed")
		}
		if err := renameRootNoReplace(t.boundary, ".terraform", filepath.Join(filepath.Base(t.backup), ".terraform")); err != nil {
			return errors.New("backup current Terraform provider tree")
		}
		t.providersMoved = true
	}
	return nil
}
func (t *terraformProjectTransaction) publishWorkspace() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	body, info, err := readGoTransactionControl(t.boundary, filepath.Join(filepath.Base(t.backup), "selected", ".terraform.lock.hcl"))

	if err != nil || !pypiSingleLink(info) || t.selectedInfo != nil && (!os.SameFile(t.selectedInfo, info) || !bytes.Equal(body, t.expectedLock)) || renameRootNoReplace(t.boundary, filepath.Join(filepath.Base(t.backup), "selected", ".terraform.lock.hcl"), ".terraform.lock.hcl") != nil {
		return errors.New("publish Terraform lock file")
	}
	t.published = true
	if err := t.check("after-lock-publication"); err != nil {
		return err
	}
	if t.providers {
		selected := filepath.Join(t.backup, "selected", ".terraform")
		records, members, err := terraformTreeInventory(context.Background(), filepath.Join(selected, "providers"))
		info, e := t.boundary.Lstat(filepath.Join(filepath.Base(t.backup), "selected", ".terraform"))
		if err != nil || e != nil || !os.SameFile(info, t.selectedTerraformInfo) || !sameTerraformFiles(records, t.providerRecords) || !sameTerraformMembers(members, t.providerMembers) {
			return errors.New("terraform private provider output changed")
		}
		if err := renameRootNoReplace(t.boundary, filepath.Join(filepath.Base(t.backup), "selected", ".terraform"), ".terraform"); err != nil {
			return errors.New("publish selected Terraform providers")
		}
		t.providersPublished = true
	}
	return nil
}
func (t *terraformProjectTransaction) publishMetadata() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	if err := t.boundary.Mkdir(".heliopause", 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return errors.New("create Terraform metadata directory")
		}
	} else {
		t.metadataDirectoryCreated = true
	}
	info, err := t.boundary.Lstat(".heliopause")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("terraform metadata directory is untrusted")
	}
	t.metadataDirectoryInfo = info
	metadata, err := t.boundary.OpenRoot(".heliopause")
	if err != nil {
		return err
	}
	defer metadata.Close()
	opened, err := metadata.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("terraform metadata boundary changed")
	}
	value := t.metadataValue
	if len(value) == 0 {
		current, err := freezeTerraformProject(t.plan.root)
		if err != nil {
			return err
		}
		value = []byte("{\"lock_sha256\":\"" + hex.EncodeToString(current.lockHash[:]) + "\"}\n")
	}
	t.metadataValue = append([]byte(nil), value...)
	file, err := metadata.OpenFile(".terraform-transaction.tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer metadata.Remove(".terraform-transaction.tmp")
	_, we := file.Write(value)
	t.metadataInfo, err = file.Stat()
	se, ce := file.Sync(), file.Close()
	if err := errors.Join(we, err, se, ce); err != nil {
		return err
	}
	if err := metadata.Link(".terraform-transaction.tmp", "terraform-transaction.json"); err != nil {
		return err
	}
	t.metadataPublished = true
	if err := metadata.Remove(".terraform-transaction.tmp"); err != nil {
		return err
	}
	return syncCargoTransactionRoot(metadata)
}
func (t *terraformProjectTransaction) rollback() error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	var result error
	if t.rollbackApproval != nil {
		result = errors.Join(result, t.rollbackApproval())
	}
	if t.providersPublished {
		info, records, members, err := inspectTerraformInstallation(context.Background(), t.boundary, t.plan.root)
		if err != nil || !os.SameFile(info, t.selectedTerraformInfo) || !sameTerraformFiles(records, t.providerRecords) || !sameTerraformMembers(members, t.providerMembers) {
			result = errors.Join(result, errors.New("terraform provider rollback identity is uncertain"))
		} else {
			result = errors.Join(result, t.boundary.RemoveAll(".terraform"))
		}
	}
	if t.providersMoved {
		old := filepath.Join(t.backup, ".terraform")
		info, err := t.boundary.Lstat(filepath.Join(filepath.Base(t.backup), ".terraform"))
		records, members, re := terraformTreeInventory(context.Background(), filepath.Join(old, "providers"))
		if err != nil || re != nil || !os.SameFile(info, t.originalTerraformInfo) || !sameTerraformFiles(records, t.originalProviderRecords) || !sameTerraformMembers(members, t.originalProviderMembers) {
			result = errors.Join(result, errors.New("terraform original provider backup changed"))
		} else {
			result = errors.Join(result, renameRootNoReplace(t.boundary, filepath.Join(filepath.Base(t.backup), ".terraform"), ".terraform"))
		}
	}
	if t.metadataPublished {
		body, info, err := readGoTransactionControl(t.boundary, terraformTransactionMetadata)
		if err != nil || t.metadataInfo == nil || !os.SameFile(info, t.metadataInfo) || !bytes.Equal(body, t.metadataValue) || info.Mode() != t.metadataInfo.Mode() {
			result = errors.Join(result, errors.New("terraform metadata rollback identity is uncertain"))
		} else if err := t.boundary.Remove(terraformTransactionMetadata); err != nil {
			result = errors.Join(result, errors.New("remove Terraform transaction metadata"))
		}
	}
	if t.metadataMoved {
		name := filepath.Join(filepath.Base(t.backup), "terraform-transaction.json")
		body, info, err := readGoTransactionControl(t.boundary, name)
		if err != nil || !os.SameFile(info, t.originalMetadataInfo) || !bytes.Equal(body, t.originalMetadataValue) || info.Mode() != t.originalMetadataInfo.Mode() {
			result = errors.Join(result, errors.New("terraform original metadata backup changed"))
		} else if err := renameRootNoReplace(t.boundary, name, terraformTransactionMetadata); err != nil {
			result = errors.Join(result, errors.New("restore Terraform transaction metadata"))
		}
	}
	if t.published {
		body, info, err := readGoTransactionControl(t.boundary, ".terraform.lock.hcl")
		if err != nil || !os.SameFile(t.selectedInfo, info) || !bytes.Equal(body, t.expectedLock) || info.Mode() != t.selectedInfo.Mode() {
			result = errors.Join(result, errors.New("terraform lock rollback identity is uncertain"))
		} else if err := t.boundary.Remove(".terraform.lock.hcl"); err != nil {
			result = errors.Join(result, errors.New("remove Terraform lock file"))
		}
	}
	if t.moved {
		name := filepath.Join(filepath.Base(t.backup), ".terraform.lock.hcl")
		body, info, err := readGoTransactionControl(t.boundary, name)
		if err != nil || !os.SameFile(info, t.plan.lockInfo) || sha256.Sum256(body) != t.plan.lockHash || info.Mode() != t.plan.lockInfo.Mode() {
			result = errors.Join(result, errors.New("terraform original lock backup changed"))
		} else if err := renameRootNoReplace(t.boundary, name, ".terraform.lock.hcl"); err != nil {
			result = errors.Join(result, errors.New("restore Terraform lock file"))
		}
	}
	if t.metadataDirectoryCreated {
		info, err := t.boundary.Lstat(".heliopause")
		if err != nil || !os.SameFile(info, t.metadataDirectoryInfo) {
			result = errors.Join(result, errors.New("terraform created metadata boundary changed"))
		} else {
			removeErr := t.boundary.Remove(".heliopause")
			result = errors.Join(result, removeErr)
			if removeErr == nil {
				t.metadataDirectoryInfo = nil
			}
		}
	}
	if err := syncDirectory(t.plan.root); err != nil {
		result = errors.Join(result, errors.New("sync Terraform rollback"))
	}
	return result
}

func (t *terraformProjectTransaction) verifyBoundary() error {
	current, err := os.Lstat(t.plan.root)
	backup, be := t.boundary.Lstat(filepath.Base(t.backup))
	if err != nil || be != nil || !os.SameFile(t.plan.rootInfo, current) || !os.SameFile(t.backupInfo, backup) || current.Mode() != t.plan.rootInfo.Mode() || backup.Mode() != t.backupInfo.Mode() {
		return errors.New("terraform transaction boundary identity changed")
	}
	if t.metadataDirectoryInfo != nil {
		current, err := t.boundary.Lstat(".heliopause")
		if err != nil || !os.SameFile(t.metadataDirectoryInfo, current) || current.Mode() != t.metadataDirectoryInfo.Mode() {
			return errors.New("terraform metadata directory identity changed")
		}
	}
	return nil
}
func (t *terraformProjectTransaction) check(phase string) error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	if t.checkpoint != nil {
		return t.checkpoint(phase)
	}
	return nil
}
func sameTerraformFiles(a, b []terraformFileRecord) bool {
	left, le := json.Marshal(a)
	right, re := json.Marshal(b)
	return le == nil && re == nil && bytes.Equal(left, right)
}
func sameTerraformMembers(a, b map[string]os.FileInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for name, before := range a {
		after := b[name]
		if (before == nil) != (after == nil) || before != nil && (!os.SameFile(before, after) || before.Mode() != after.Mode()) {
			return false
		}
	}
	return true
}

// Recheck the complete private/rollback namespace before removing it. A file
// added by another actor is preserved, even after all visible controls restore.
func (t *terraformProjectTransaction) verifyBackupContents(originals bool) error {
	if err := t.verifyBoundary(); err != nil {
		return err
	}
	backup, err := t.boundary.OpenRoot(filepath.Base(t.backup))
	if err != nil {
		return err
	}
	defer backup.Close()
	selected, err := backup.OpenRoot("selected")
	if err != nil {
		return err
	}
	defer selected.Close()
	info, err := selected.Stat(".")
	if err != nil || !os.SameFile(info, t.selectedDirectoryInfo) || info.Mode() != t.selectedDirectoryInfo.Mode() {
		return errors.New("terraform selected backup directory changed")
	}
	checkNames := func(root *os.Root, want map[string]bool) error {
		f, e := root.Open(".")
		if e != nil {
			return e
		}
		entries, re := f.ReadDir(8)
		ce := f.Close()
		if re != nil && !errors.Is(re, io.EOF) || ce != nil || len(entries) != len(want) {
			return errors.New("terraform backup contains unexpected members")
		}
		for _, entry := range entries {
			if !want[entry.Name()] {
				return errors.New("terraform backup contains a foreign member")
			}
		}
		return nil
	}
	selectedNames := map[string]bool{}
	if !t.published {
		selectedNames[".terraform.lock.hcl"] = true
		body, info, err := readGoTransactionControl(selected, ".terraform.lock.hcl")
		if err != nil || !os.SameFile(info, t.selectedInfo) || !bytes.Equal(body, t.expectedLock) || info.Mode() != t.selectedInfo.Mode() {
			return errors.New("terraform selected backup lock changed")
		}
	}
	if t.providers && !t.providersPublished {
		selectedNames[".terraform"] = true
		info, files, members, err := inspectTerraformInstallation(context.Background(), selected, filepath.Join(t.backup, "selected"))
		if err != nil || !os.SameFile(info, t.selectedTerraformInfo) || !sameTerraformFiles(files, t.providerRecords) || !sameTerraformMembers(members, t.providerMembers) {
			return errors.New("terraform selected backup providers changed")
		}
	}
	if err := checkNames(selected, selectedNames); err != nil {
		return err
	}
	names := map[string]bool{"selected": true}
	if originals && t.moved {
		names[".terraform.lock.hcl"] = true
		body, info, err := readGoTransactionControl(backup, ".terraform.lock.hcl")
		if err != nil || !os.SameFile(info, t.plan.lockInfo) || sha256.Sum256(body) != t.plan.lockHash || info.Mode() != t.plan.lockInfo.Mode() {
			return errors.New("terraform original backup lock changed")
		}
	}
	if originals && t.metadataMoved {
		names["terraform-transaction.json"] = true
		body, info, err := readGoTransactionControl(backup, "terraform-transaction.json")
		if err != nil || !os.SameFile(info, t.originalMetadataInfo) || !bytes.Equal(body, t.originalMetadataValue) || info.Mode() != t.originalMetadataInfo.Mode() {
			return errors.New("terraform original backup metadata changed")
		}
	}
	if originals && t.providersMoved {
		names[".terraform"] = true
		info, files, members, err := inspectTerraformInstallation(context.Background(), backup, t.backup)
		if err != nil || !os.SameFile(info, t.originalTerraformInfo) || !sameTerraformFiles(files, t.originalProviderRecords) || !sameTerraformMembers(members, t.originalProviderMembers) {
			return errors.New("terraform original backup providers changed")
		}
	}
	return checkNames(backup, names)
}
