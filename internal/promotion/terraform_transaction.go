package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

const terraformTransactionMetadata = ".heliopause/terraform-transaction.json"

type terraformProjectPlan struct {
	root     string
	lockHash [32]byte
}

func freezeTerraformProject(root string) (terraformProjectPlan, error) {
	if !filepath.IsAbs(root) || trustedExistingDirectory(root) != nil {
		return terraformProjectPlan{}, errors.New("Terraform project root is untrusted")
	}
	body, err := os.ReadFile(filepath.Join(root, ".terraform.lock.hcl"))
	if err != nil || len(body) == 0 {
		return terraformProjectPlan{}, errors.New("Terraform lock file is unavailable")
	}
	return terraformProjectPlan{root: root, lockHash: sha256.Sum256(body)}, nil
}
func (p terraformProjectPlan) verifyUnchanged() error {
	current, err := freezeTerraformProject(p.root)
	if err != nil || current.lockHash != p.lockHash {
		return errors.New("Terraform lock file changed during transaction")
	}
	return nil
}
func (p terraformProjectPlan) verifyManaged() error {
	body, err := os.ReadFile(filepath.Join(p.root, terraformTransactionMetadata))
	if err != nil {
		return errors.New("Terraform project is not HAA-managed")
	}
	want := "{\"lock_sha256\":\"" + hex.EncodeToString(p.lockHash[:]) + "\"}\n"
	if string(body) != want {
		return errors.New("Terraform managed metadata does not match current state")
	}
	return nil
}
func (p terraformProjectPlan) privateWorkspace() (string, error) {
	workspace, err := os.MkdirTemp(filepath.Dir(p.root), "."+filepath.Base(p.root)+".haa-terraform-work-")
	if err != nil {
		return "", errors.New("create Terraform private transaction workspace")
	}
	body, readErr := os.ReadFile(filepath.Join(p.root, ".terraform.lock.hcl"))
	if readErr != nil || os.WriteFile(filepath.Join(workspace, ".terraform.lock.hcl"), body, 0o600) != nil {
		_ = os.RemoveAll(workspace)
		return "", errors.New("copy Terraform lock file")
	}
	return workspace, nil
}

type terraformProjectTransaction struct {
	plan                                               terraformProjectPlan
	workspace, backup                                  string
	moved, published, metadataMoved, metadataPublished bool
}

func beginTerraformProjectTransaction(plan terraformProjectPlan, workspace string) (*terraformProjectTransaction, error) {
	if err := plan.verifyUnchanged(); err != nil {
		return nil, err
	}
	if err := plan.verifyManaged(); err != nil {
		return nil, err
	}
	if err := trustedExistingDirectory(workspace); err != nil {
		return nil, errors.New("Terraform private transaction workspace is untrusted")
	}
	backup, err := os.MkdirTemp(plan.root, ".heliopause-terraform-commit-")
	if err != nil {
		return nil, errors.New("create Terraform rollback transaction")
	}
	return &terraformProjectTransaction{plan: plan, workspace: workspace, backup: backup}, nil
}
func (t *terraformProjectTransaction) commit() error {
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
		return t.fail(errors.New("sync committed Terraform project"))
	}
	if err := os.RemoveAll(t.backup); err != nil {
		return errors.New("remove committed Terraform rollback backup; project is fail-closed")
	}
	return nil
}
func (t *terraformProjectTransaction) fail(cause error) error {
	if rollbackErr := t.rollback(); rollbackErr != nil {
		return errors.Join(cause, rollbackErr, errors.New("Terraform transaction rollback is incomplete; project is fail-closed"))
	}
	if err := os.RemoveAll(t.backup); err != nil {
		return errors.Join(cause, errors.New("remove rolled back Terraform transaction; project is fail-closed"))
	}
	return cause
}
func (t *terraformProjectTransaction) backupCurrent() error {
	metadata := filepath.Join(t.plan.root, terraformTransactionMetadata)
	if info, err := os.Lstat(metadata); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || os.Rename(metadata, filepath.Join(t.backup, "terraform-transaction.json")) != nil {
			return errors.New("backup Terraform transaction metadata")
		}
		t.metadataMoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect Terraform transaction metadata")
	}
	if err := os.Rename(filepath.Join(t.plan.root, ".terraform.lock.hcl"), filepath.Join(t.backup, ".terraform.lock.hcl")); err != nil {
		return errors.New("backup Terraform lock file")
	}
	t.moved = true
	return nil
}
func (t *terraformProjectTransaction) publishWorkspace() error {
	info, err := os.Lstat(filepath.Join(t.workspace, ".terraform.lock.hcl"))
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || os.Rename(filepath.Join(t.workspace, ".terraform.lock.hcl"), filepath.Join(t.plan.root, ".terraform.lock.hcl")) != nil {
		return errors.New("publish Terraform lock file")
	}
	t.published = true
	return nil
}
func (t *terraformProjectTransaction) publishMetadata() error {
	directory := filepath.Join(t.plan.root, ".heliopause")
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("create Terraform transaction metadata directory")
	}
	if err := trustedExistingDirectory(directory); err != nil {
		return errors.New("Terraform transaction metadata directory is untrusted")
	}
	current, err := freezeTerraformProject(t.plan.root)
	if err != nil {
		return err
	}
	value := "{\"lock_sha256\":\"" + hex.EncodeToString(current.lockHash[:]) + "\"}\n"
	if err := writeNPMTransactionMetadata(directory, filepath.Join(directory, "terraform-transaction.json"), []byte(value)); err != nil {
		return errors.New("publish Terraform transaction metadata")
	}
	t.metadataPublished = true
	return nil
}
func (t *terraformProjectTransaction) rollback() error {
	var result error
	if t.metadataPublished {
		if err := os.Remove(filepath.Join(t.plan.root, terraformTransactionMetadata)); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, errors.New("remove Terraform transaction metadata"))
		}
	}
	if t.metadataMoved {
		if err := os.Rename(filepath.Join(t.backup, "terraform-transaction.json"), filepath.Join(t.plan.root, terraformTransactionMetadata)); err != nil {
			result = errors.Join(result, errors.New("restore Terraform transaction metadata"))
		}
	}
	if t.published {
		if err := os.Remove(filepath.Join(t.plan.root, ".terraform.lock.hcl")); err != nil {
			result = errors.Join(result, errors.New("remove Terraform lock file"))
		}
	}
	if t.moved {
		if err := os.Rename(filepath.Join(t.backup, ".terraform.lock.hcl"), filepath.Join(t.plan.root, ".terraform.lock.hcl")); err != nil {
			result = errors.Join(result, errors.New("restore Terraform lock file"))
		}
	}
	if err := syncDirectory(t.plan.root); err != nil {
		result = errors.Join(result, errors.New("sync Terraform rollback"))
	}
	return result
}
