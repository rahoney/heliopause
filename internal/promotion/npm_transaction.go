package promotion

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
)

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
