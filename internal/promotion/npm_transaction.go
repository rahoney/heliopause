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
	packageJSON [32]byte
	packageLock [32]byte
}

type npmProjectGuard struct{ path string }

func acquireNPMProjectGuard(root string) (npmProjectGuard, error) {
	path := filepath.Join(root, ".heliopause-npm-transaction.lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return npmProjectGuard{}, errors.New("npm project is already being mutated or lock is unavailable")
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return npmProjectGuard{}, errors.New("close npm project transaction lock")
	}
	return npmProjectGuard{path: path}, nil
}

func (g npmProjectGuard) release() error {
	if g.path == "" {
		return errors.New("npm project transaction lock is unavailable")
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
	return npmProjectPlan{root: root, packageJSON: sha256.Sum256(manifest), packageLock: sha256.Sum256(lock)}, nil
}

func (p npmProjectPlan) verifyUnchanged() error {
	current, err := freezeNPMProject(p.root)
	if err != nil || current.packageJSON != p.packageJSON || current.packageLock != p.packageLock {
		return errors.New("npm project changed during transaction")
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
	directory := filepath.Join(p.root, ".heliopause")
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("create npm transaction metadata directory")
	}
	value := hex.EncodeToString(p.packageJSON[:]) + ":" + hex.EncodeToString(p.packageLock[:]) + "\n"
	if err := os.WriteFile(filepath.Join(p.root, npmTransactionMetadata), []byte(value), 0o600); err != nil {
		return errors.New("write npm transaction metadata")
	}
	return nil
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

func (p npmProjectPlan) commitControlFiles(workspace string) error {
	if err := p.verifyUnchanged(); err != nil {
		return err
	}
	original := make(map[string][]byte, 2)
	for _, name := range []string{"package.json", "package-lock.json"} {
		body, err := os.ReadFile(filepath.Join(p.root, name))
		if err != nil {
			return errors.New("backup npm control files")
		}
		original[name] = body
	}
	committed := []string{}
	rollback := func() error {
		for _, name := range committed {
			if err := os.WriteFile(filepath.Join(p.root, name), original[name], 0o600); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		body, err := os.ReadFile(filepath.Join(workspace, name))
		if err != nil || len(body) == 0 {
			return errors.New("read verified npm transaction output")
		}
		temporary, err := os.CreateTemp(p.root, "."+name+".haa-")
		if err != nil {
			return errors.New("create npm control file transaction")
		}
		path := temporary.Name()
		if _, err := temporary.Write(body); err != nil || temporary.Sync() != nil || temporary.Close() != nil {
			_ = os.Remove(path)
			return errors.New("write npm control file transaction")
		}
		if err := os.Rename(path, filepath.Join(p.root, name)); err != nil {
			_ = os.Remove(path)
			_ = rollback()
			return errors.New("commit npm control file transaction")
		}
		committed = append(committed, name)
	}
	return nil
}

func (p npmProjectPlan) swapNodeModules(workspace string) error {
	staged := filepath.Join(workspace, "node_modules")
	if err := trustedExistingDirectory(staged); err != nil {
		return errors.New("verified npm node_modules is unavailable")
	}
	target := filepath.Join(p.root, "node_modules")
	backup := filepath.Join(p.root, ".node_modules.haa-backup")
	if _, err := os.Lstat(backup); !errors.Is(err, os.ErrNotExist) {
		return errors.New("npm node_modules backup already exists")
	}
	hadTarget := false
	if _, err := os.Lstat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return errors.New("backup npm node_modules")
		}
		hadTarget = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect npm node_modules")
	}
	if err := os.Rename(staged, target); err != nil {
		if hadTarget {
			_ = os.Rename(backup, target)
		}
		return errors.New("commit npm node_modules")
	}
	if hadTarget && os.RemoveAll(backup) != nil {
		return errors.New("remove npm node_modules rollback backup")
	}
	return nil
}
