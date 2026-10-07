package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const npmTransactionMetadata = ".heliopause/npm-transaction.json"

// npmProjectPlan freezes the small project-control surface before any later
// mutation. A changed or missing control file invalidates the transaction.
type npmProjectPlan struct {
	root        string
	rootInfo    os.FileInfo
	packageJSON [32]byte
	packageLock [32]byte
	controls    map[string]pypiFileSnapshot
}

type npmProjectGuard struct {
	path string
	info os.FileInfo
}

func acquireNPMProjectGuard(root string) (npmProjectGuard, error) {
	path := filepath.Join(root, ".heliopause-npm-transaction.lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return npmProjectGuard{}, errors.New("npm project is already being mutated or lock is unavailable")
	}
	info, statErr := file.Stat()
	if err := errors.Join(statErr, file.Close()); err != nil {
		_ = os.Remove(path)
		return npmProjectGuard{}, errors.New("close npm project transaction lock")
	}
	return npmProjectGuard{path: path, info: info}, nil
}

func (g npmProjectGuard) release() error {
	if g.path == "" {
		return errors.New("npm project transaction lock is unavailable")
	}
	now, err := os.Lstat(g.path)
	if err != nil || g.info == nil || !os.SameFile(g.info, now) || now.Mode() != g.info.Mode() || !pypiSingleLink(now) {
		return errors.New("npm project transaction lock changed")
	}
	if err := os.Remove(g.path); err != nil {
		return errors.New("remove npm project transaction lock")
	}
	return nil
}

func freezeNPMProject(root string) (npmProjectPlan, error) {
	if !filepath.IsAbs(root) || trustedExistingDirectory(root) != nil {
		return npmProjectPlan{}, errors.New("npm project root is untrusted")
	}
	manifest, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil || len(manifest) == 0 {
		return npmProjectPlan{}, errors.New("npm project package.json is unavailable")
	}
	lock, err := os.ReadFile(filepath.Join(root, "package-lock.json"))
	if err != nil || len(lock) == 0 {
		return npmProjectPlan{}, errors.New("npm project package-lock.json is unavailable")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return npmProjectPlan{}, err
	}
	controls := map[string]pypiFileSnapshot{}
	for _, name := range []string{"package.json", "package-lock.json"} {
		controls[name], err = snapshotPyPIFile(filepath.Join(root, name))
		if err != nil {
			return npmProjectPlan{}, err
		}
	}
	manifestDigest, lockDigest := sha256.Sum256(manifest), sha256.Sum256(lock)
	if controls["package.json"].Digest != hex.EncodeToString(manifestDigest[:]) || controls["package-lock.json"].Digest != hex.EncodeToString(lockDigest[:]) {
		return npmProjectPlan{}, errors.New("npm controls changed while freezing")
	}
	return npmProjectPlan{root: root, rootInfo: info, controls: controls, packageJSON: sha256.Sum256(manifest), packageLock: sha256.Sum256(lock)}, nil
}

func (p npmProjectPlan) verifyUnchanged() error {
	current, err := freezeNPMProject(p.root)
	if err != nil || !os.SameFile(p.rootInfo, current.rootInfo) || p.rootInfo.Mode() != current.rootInfo.Mode() || current.packageJSON != p.packageJSON || current.packageLock != p.packageLock {
		return errors.New("npm project changed during transaction")
	}
	for name, before := range p.controls {
		if !before.matches(filepath.Join(p.root, name)) {
			return errors.New("npm control identity changed")
		}
	}
	return nil
}

func (p npmProjectPlan) verifyManagedOrEmpty() error {
	body, err := os.ReadFile(filepath.Join(p.root, npmTransactionMetadata))
	if err == nil {
		want := hex.EncodeToString(p.packageJSON[:]) + ":" + hex.EncodeToString(p.packageLock[:])
		if string(body) == want+"\n" {
			return nil
		}
		return errors.New("npm managed project metadata does not match current state")
	}
	if !errors.Is(err, os.ErrNotExist) {
		return errors.New("read npm project transaction metadata")
	}
	if hasNPMDependencies(filepath.Join(p.root, "package.json")) || hasNPMDependencies(filepath.Join(p.root, "package-lock.json")) {
		return errors.New("npm project has unmanaged dependencies")
	}
	return nil
}

func (p npmProjectPlan) writeMetadata() error {
	directory, err := ensureNPMMetadataDirectory(p.root)
	if err != nil {
		return err
	}
	value := hex.EncodeToString(p.packageJSON[:]) + ":" + hex.EncodeToString(p.packageLock[:]) + "\n"
	return writeNPMTransactionMetadata(directory, filepath.Join(directory, "npm-transaction.json"), []byte(value))
}

func ensureNPMMetadataDirectory(root string) (string, error) {
	directory := filepath.Join(root, ".heliopause")
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", errors.New("create npm transaction metadata directory")
	}
	if err := trustedExistingDirectory(directory); err != nil {
		return "", errors.New("npm transaction metadata directory is untrusted")
	}
	return directory, nil
}

func hasNPMDependencies(path string) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	var document struct {
		Dependencies map[string]json.RawMessage `json:"dependencies"`
		Packages     map[string]json.RawMessage `json:"packages"`
	}
	if json.Unmarshal(body, &document) != nil {
		return true
	}
	return len(document.Dependencies) != 0 || len(document.Packages) > 1
}

func (p npmProjectPlan) privateWorkspace() (string, error) {
	parent := filepath.Dir(p.root)
	workspace, err := os.MkdirTemp(parent, "."+filepath.Base(p.root)+".haa-work-")
	if err != nil {
		return "", errors.New("create npm private transaction workspace")
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		body, readErr := os.ReadFile(filepath.Join(p.root, name))
		if readErr != nil || os.WriteFile(filepath.Join(workspace, name), body, 0o600) != nil {
			_ = os.RemoveAll(workspace)
			return "", errors.New("copy npm transaction control files")
		}
	}
	return workspace, nil
}

// npmProjectTransaction keeps every old project member until the new exact
// Verified Set and its binding metadata have both been made durable.  It never
// reports a successful promotion while a rollback backup remains.
type npmProjectTransaction struct {
	plan          npmProjectPlan
	workspace     string
	backup        string
	backupInfo    os.FileInfo
	workspaceInfo os.FileInfo
	moved         map[string]bool
	published     map[string]bool
	before        map[string]*npmTransactionMember
	outputs       map[string]*npmTransactionMember
}

func beginNPMProjectTransaction(plan npmProjectPlan, workspace string) (*npmProjectTransaction, error) {
	if err := plan.verifyUnchanged(); err != nil {
		return nil, err
	}
	if err := rejectInterruptedNPMTransaction(plan.root); err != nil {
		return nil, err
	}
	if err := trustedExistingDirectory(workspace); err != nil {
		return nil, errors.New("npm private transaction workspace is untrusted")
	}
	backup, err := os.MkdirTemp(plan.root, ".heliopause-npm-commit-")
	if err != nil {
		return nil, errors.New("create npm rollback transaction")
	}
	backupInfo, err := os.Lstat(backup)
	if err != nil {
		return nil, err
	}
	workspaceInfo, err := os.Lstat(workspace)
	if err != nil {
		return nil, err
	}
	return &npmProjectTransaction{plan: plan, workspace: workspace, backup: backup, backupInfo: backupInfo, workspaceInfo: workspaceInfo, moved: map[string]bool{}, published: map[string]bool{}, before: map[string]*npmTransactionMember{}, outputs: map[string]*npmTransactionMember{}}, nil
}

func rejectInterruptedNPMTransaction(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return errors.New("read npm project transaction state")
	}
	for _, entry := range entries {
		if len(entry.Name()) > len(".heliopause-npm-commit-") && entry.Name()[:len(".heliopause-npm-commit-")] == ".heliopause-npm-commit-" {
			return errors.New("interrupted npm transaction requires explicit recovery")
		}
	}
	return nil
}

func (t *npmProjectTransaction) commit() (resultErr error) {
	if err := t.plan.verifyUnchanged(); err != nil {
		return err
	}
	if err := syncTree(t.workspace); err != nil {
		return errors.New("sync verified npm transaction workspace")
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
		return t.fail(errors.New("sync committed npm project"))
	}
	for name, output := range t.outputs {
		if err := output.verify(t.target(name)); err != nil {
			return t.fail(err)
		}
	}
	if err := t.removeBackup(true); err != nil {
		return errors.Join(err, errors.New("remove committed npm rollback backup; project is fail-closed"))
	}
	if err := syncDirectory(t.plan.root); err != nil {
		return errors.New("sync completed npm transaction; project is fail-closed")
	}
	return nil
}

func (t *npmProjectTransaction) fail(cause error) error {
	rollbackErr := t.rollback()
	if rollbackErr != nil {
		return errors.Join(cause, rollbackErr, errors.New("npm transaction rollback is incomplete; project is fail-closed"))
	}
	if err := t.removeBackup(false); err != nil {
		return errors.Join(cause, err, errors.New("remove rolled back npm transaction; project is fail-closed"))
	}
	return cause
}

func (t *npmProjectTransaction) backupCurrent() error {
	if err := t.verifyBoundaries(); err != nil {
		return err
	}
	for _, name := range []string{"package.json", "package-lock.json", "node_modules", "haa"} {
		source := t.target(name)
		info, err := os.Lstat(source)
		if errors.Is(err, os.ErrNotExist) && (name == "node_modules" || name == "haa") {
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("inspect current npm transaction member")
		}
		if ((name == "node_modules" || name == "haa") && !info.IsDir()) || (name != "node_modules" && name != "haa" && !info.Mode().IsRegular()) {
			return errors.New("current npm transaction member has unsupported type")
		}
		record, err := snapshotNPMTransactionMember(source)
		if err != nil {
			return err
		}
		if err := record.verify(source); err != nil {
			return err
		}
		if err := renameNoReplace(source, filepath.Join(t.backup, name)); err != nil {
			return errors.New("backup current npm transaction member")
		}
		t.moved[name] = true
		t.before[name] = record
	}
	return nil
}

func (t *npmProjectTransaction) publishWorkspace() error {
	for _, name := range []string{"package.json", "package-lock.json", "node_modules", "haa"} {
		if err := t.verifyBoundaries(); err != nil {
			return err
		}
		source := filepath.Join(t.workspace, name)
		if name == "haa" {
			source = filepath.Join(t.workspace, ".heliopause")
		}
		info, err := os.Lstat(source)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || ((name == "node_modules" || name == "haa") && !info.IsDir()) || (name != "node_modules" && name != "haa" && !info.Mode().IsRegular()) {
			return errors.New("verified npm transaction output is unavailable")
		}
		record, err := snapshotNPMTransactionMember(source)
		if err != nil {
			return err
		}
		if err := record.verify(source); err != nil {
			return err
		}
		if err := renameNoReplace(source, t.target(name)); err != nil {
			return errors.New("publish verified npm transaction member")
		}
		t.published[name] = true
		t.outputs[name] = record
	}
	return nil
}

func (t *npmProjectTransaction) publishMetadata() error {
	directory, err := ensureNPMMetadataDirectory(t.plan.root)
	if err != nil {
		return err
	}
	current, err := freezeNPMProject(t.plan.root)
	if err != nil {
		return errors.New("freeze committed npm project state")
	}
	value := hex.EncodeToString(current.packageJSON[:]) + ":" + hex.EncodeToString(current.packageLock[:]) + "\n"
	if err := writeNPMTransactionMetadata(directory, t.target("metadata"), []byte(value)); err != nil {
		return err
	}
	snapshot, err := snapshotPyPIFile(t.target("metadata"))
	if err != nil {
		return err
	}
	t.outputs["haa"].files["npm-transaction.json"] = snapshot
	if err := syncDirectory(directory); err != nil {
		return errors.New("sync npm transaction metadata")
	}
	return nil
}

func writeNPMTransactionMetadata(directory, target string, value []byte) error {
	temporary, err := os.CreateTemp(directory, ".npm-transaction-")
	if err != nil {
		return errors.New("create npm transaction metadata")
	}
	path := temporary.Name()
	if _, err := temporary.Write(value); err != nil || temporary.Sync() != nil || temporary.Close() != nil {
		_ = os.Remove(path)
		return errors.New("write npm transaction metadata")
	}
	if err := renameNoReplace(path, target); err != nil {
		_ = os.Remove(path)
		return errors.New("publish npm transaction metadata")
	}
	return nil
}

func (t *npmProjectTransaction) rollback() error {
	if err := t.verifyBoundaries(); err != nil {
		return err
	}
	var result error
	for _, name := range []string{"haa", "node_modules", "package-lock.json", "package.json"} {
		if t.moved[name] {
			if err := t.before[name].verify(filepath.Join(t.backup, name)); err != nil {
				result = errors.Join(result, err)
				continue
			}
		}
		if t.published[name] {
			if err := t.outputs[name].remove(t.target(name)); err != nil {
				result = errors.Join(result, err, errors.New("remove uncommitted npm transaction member"))
				continue
			}
		}
		if t.moved[name] {
			if err := renameNoReplace(filepath.Join(t.backup, name), t.target(name)); err != nil {
				result = errors.Join(result, errors.New("restore npm transaction member"))
			}
		}
	}
	if err := syncDirectory(t.plan.root); err != nil {
		result = errors.Join(result, errors.New("sync rolled back npm transaction"))
	}
	return result
}

func (t *npmProjectTransaction) target(name string) string {
	if name == "metadata" {
		return filepath.Join(t.plan.root, npmTransactionMetadata)
	}
	if name == "haa" {
		return filepath.Join(t.plan.root, ".heliopause")
	}
	return filepath.Join(t.plan.root, name)
}
