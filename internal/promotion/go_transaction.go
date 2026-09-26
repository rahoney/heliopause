package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

const goTransactionMetadata = ".heliopause/go-transaction.json"

// goProjectPlan freezes the complete Go control-file transaction surface.
type goProjectPlan struct {
	root      string
	goModHash [32]byte
	goSumHash [32]byte
}

type goProjectGuard struct{ path string }

func acquireGoProjectGuard(root string) (goProjectGuard, error) {
	path := filepath.Join(root, ".heliopause-go-transaction.lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return goProjectGuard{}, errors.New("go project is already being mutated or lock is unavailable")
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return goProjectGuard{}, errors.New("close Go project transaction lock")
	}
	return goProjectGuard{path: path}, nil
}

func (g goProjectGuard) release() error {
	if g.path == "" {
		return errors.New("go project transaction lock is unavailable")
	}
	if err := os.Remove(g.path); err != nil {
		return errors.New("remove Go project transaction lock")
	}
	return nil
}

func freezeGoProject(root string) (goProjectPlan, error) {
	if !filepath.IsAbs(root) || trustedExistingDirectory(root) != nil {
		return goProjectPlan{}, errors.New("go project root is untrusted")
	}
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || len(mod) == 0 {
		return goProjectPlan{}, errors.New("go project go.mod is unavailable")
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil || len(sum) == 0 {
		return goProjectPlan{}, errors.New("go project go.sum is unavailable")
	}
	return goProjectPlan{root: root, goModHash: sha256.Sum256(mod), goSumHash: sha256.Sum256(sum)}, nil
}

func (p goProjectPlan) verifyUnchanged() error {
	current, err := freezeGoProject(p.root)
	if err != nil || current.goModHash != p.goModHash || current.goSumHash != p.goSumHash {
		return errors.New("go project changed during transaction")
	}
	return nil
}

func (p goProjectPlan) verifyManaged() error {
	body, err := os.ReadFile(filepath.Join(p.root, goTransactionMetadata))
	if err != nil {
		return errors.New("go project is not HAA-managed")
	}
	want := "{\"go_mod_sha256\":\"" + hex.EncodeToString(p.goModHash[:]) + "\",\"go_sum_sha256\":\"" + hex.EncodeToString(p.goSumHash[:]) + "\"}\n"
	if string(body) != want {
		return errors.New("go managed project metadata does not match current state")
	}
	return nil
}

func (p goProjectPlan) privateWorkspace() (string, error) {
	workspace, err := os.MkdirTemp(filepath.Dir(p.root), "."+filepath.Base(p.root)+".haa-go-work-")
	if err != nil {
		return "", errors.New("create Go private transaction workspace")
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		body, readErr := os.ReadFile(filepath.Join(p.root, name))
		if readErr != nil || os.WriteFile(filepath.Join(workspace, name), body, 0o600) != nil {
			_ = os.RemoveAll(workspace)
			return "", errors.New("copy Go transaction control files")
		}
	}
	return workspace, nil
}

type goProjectTransaction struct {
	plan              goProjectPlan
	workspace         string
	backup            string
	moved             map[string]bool
	published         map[string]bool
	metadataMoved     bool
	metadataPublished bool
}

func beginGoProjectTransaction(plan goProjectPlan, workspace string) (*goProjectTransaction, error) {
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
	return &goProjectTransaction{plan: plan, workspace: workspace, backup: backup, moved: map[string]bool{}, published: map[string]bool{}}, nil
}

func (t *goProjectTransaction) commit() (resultErr error) {
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
		return t.fail(errors.New("sync committed Go project"))
	}
	if err := os.RemoveAll(t.backup); err != nil {
		return errors.New("remove committed Go rollback backup; project is fail-closed")
	}
	return syncDirectory(t.plan.root)
}

func (t *goProjectTransaction) fail(cause error) error {
	rollbackErr := t.rollback()
	if rollbackErr != nil {
		return errors.Join(cause, rollbackErr, errors.New("go transaction rollback is incomplete; project is fail-closed"))
	}
	if err := os.RemoveAll(t.backup); err != nil {
		return errors.Join(cause, errors.New("remove rolled back Go transaction; project is fail-closed"))
	}
	return cause
}

func (t *goProjectTransaction) backupCurrent() error {
	metadataPath := filepath.Join(t.plan.root, goTransactionMetadata)
	if info, err := os.Lstat(metadataPath); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("inspect current Go transaction metadata")
		}
		if err := os.Rename(metadataPath, filepath.Join(t.backup, "go-transaction.json")); err != nil {
			return errors.New("backup current Go transaction metadata")
		}
		t.metadataMoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect current Go transaction metadata")
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		info, err := os.Lstat(filepath.Join(t.plan.root, name))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("inspect current Go transaction member")
		}
		if err := os.Rename(filepath.Join(t.plan.root, name), filepath.Join(t.backup, name)); err != nil {
			return errors.New("backup current Go transaction member")
		}
		t.moved[name] = true
	}
	return nil
}

func (t *goProjectTransaction) publishWorkspace() error {
	for _, name := range []string{"go.mod", "go.sum"} {
		info, err := os.Lstat(filepath.Join(t.workspace, name))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("verified Go transaction output is unavailable")
		}
		if err := os.Rename(filepath.Join(t.workspace, name), filepath.Join(t.plan.root, name)); err != nil {
			return errors.New("publish verified Go transaction member")
		}
		t.published[name] = true
	}
	return nil
}

func (t *goProjectTransaction) publishMetadata() error {
	directory := filepath.Join(t.plan.root, ".heliopause")
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("create Go transaction metadata directory")
	}
	if err := trustedExistingDirectory(directory); err != nil {
		return errors.New("go transaction metadata directory is untrusted")
	}
	current, err := freezeGoProject(t.plan.root)
	if err != nil {
		return err
	}
	value := "{\"go_mod_sha256\":\"" + hex.EncodeToString(current.goModHash[:]) + "\",\"go_sum_sha256\":\"" + hex.EncodeToString(current.goSumHash[:]) + "\"}\n"
	if err := writeNPMTransactionMetadata(directory, filepath.Join(directory, "go-transaction.json"), []byte(value)); err != nil {
		return errors.New("publish Go transaction metadata")
	}
	t.metadataPublished = true
	return nil
}

func (t *goProjectTransaction) rollback() error {
	var result error
	if t.metadataPublished {
		if err := os.Remove(filepath.Join(t.plan.root, goTransactionMetadata)); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, errors.New("remove uncommitted Go transaction metadata"))
		}
	}
	if t.metadataMoved {
		if err := os.Rename(filepath.Join(t.backup, "go-transaction.json"), filepath.Join(t.plan.root, goTransactionMetadata)); err != nil {
			result = errors.Join(result, errors.New("restore Go transaction metadata"))
		}
	}
	for _, name := range []string{"go.sum", "go.mod"} {
		if t.published[name] {
			if err := os.Remove(filepath.Join(t.plan.root, name)); err != nil {
				result = errors.Join(result, errors.New("remove uncommitted Go transaction member"))
			}
		}
		if t.moved[name] {
			if err := os.Rename(filepath.Join(t.backup, name), filepath.Join(t.plan.root, name)); err != nil {
				result = errors.Join(result, errors.New("restore npm transaction member"))
			}
		}
	}
	if err := syncDirectory(t.plan.root); err != nil {
		result = errors.Join(result, errors.New("sync rolled back Go transaction"))
	}
	return result
}
