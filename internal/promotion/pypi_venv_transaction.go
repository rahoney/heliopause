package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/sandbox"
)

const pypiVenvMetadata = ".heliopause/pypi-transaction.json"

type pypiVenvPlan struct {
	root string
	site string
}

func discoverPythonVenv(root string) (pypiVenvPlan, error) {
	if !filepath.IsAbs(root) || trustedExistingDirectory(root) != nil {
		return pypiVenvPlan{}, errors.New("active Python virtual environment is untrusted")
	}
	if err := rejectSymlinkPath(filepath.Join(root, "pyvenv.cfg")); err != nil {
		return pypiVenvPlan{}, errors.New("active Python virtual environment configuration is untrusted")
	}
	cfg, err := os.ReadFile(filepath.Join(root, "pyvenv.cfg"))
	if err != nil || !strings.Contains(string(cfg), "version = 3.14") {
		return pypiVenvPlan{}, errors.New("active Python virtual environment version is unsupported")
	}
	minor := strings.Join(strings.Split(sandbox.PinnedPythonRuntime().PythonVersion, ".")[:2], ".")
	site := filepath.Join(root, "lib", "python"+minor, "site-packages")
	if trustedExistingDirectory(site) != nil {
		return pypiVenvPlan{}, errors.New("active Python virtual environment site-packages is unavailable")
	}
	return pypiVenvPlan{root: root, site: site}, nil
}

type pypiVenvState struct {
	Files map[string]string `json:"files"`
}

func (p pypiVenvPlan) readState() (pypiVenvState, bool, error) {
	body, err := os.ReadFile(filepath.Join(p.root, pypiVenvMetadata))
	if errors.Is(err, os.ErrNotExist) {
		return pypiVenvState{Files: map[string]string{}}, false, nil
	}
	if err != nil {
		return pypiVenvState{}, false, errors.New("read PyPI virtual environment transaction metadata")
	}
	var state pypiVenvState
	if json.Unmarshal(body, &state) != nil || len(state.Files) == 0 {
		return pypiVenvState{}, false, errors.New("pypi virtual environment transaction metadata is malformed")
	}
	for path, digest := range state.Files {
		if filepath.IsAbs(path) || filepath.Clean(path) != path || strings.HasPrefix(path, "..") || len(digest) != 64 {
			return pypiVenvState{}, false, errors.New("pypi virtual environment transaction metadata is unsafe")
		}
	}
	return state, true, nil
}

func (p pypiVenvPlan) outputState(output string) (pypiVenvState, error) {
	state := pypiVenvState{Files: map[string]string{}}
	err := filepath.WalkDir(output, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("pypi virtual environment output contains symbolic link")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("pypi virtual environment output contains special file")
		}
		relative, err := filepath.Rel(output, path)
		if err != nil || filepath.IsAbs(relative) || strings.HasPrefix(relative, "..") {
			return errors.New("pypi virtual environment output escapes site")
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		state.Files[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil || len(state.Files) == 0 {
		return pypiVenvState{}, errors.New("pypi virtual environment output is empty or invalid")
	}
	return state, nil
}

func (p pypiVenvPlan) verifyState(state pypiVenvState) error {
	for relative, expected := range state.Files {
		path := filepath.Join(p.site, filepath.FromSlash(relative))
		if err := rejectSymlinkPathAllowMissing(path); err != nil {
			return errors.New("pypi virtual environment state contains an unsafe path")
		}
		if filepath.Dir(path) != p.site && !strings.HasPrefix(path, p.site+string(os.PathSeparator)) {
			return errors.New("pypi virtual environment state escapes site")
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return errors.New("haa-owned pypi virtual environment file is missing")
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != expected {
			return errors.New("haa-owned pypi virtual environment file changed")
		}
	}
	return nil
}

func (p pypiVenvPlan) commit(output string, desired pypiVenvState) (resultErr error) {
	guard, err := acquirePyPIVenvGuard(p.root)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, guard.release()) }()
	current, managed, err := p.readState()
	if err != nil {
		return err
	}
	if managed {
		if err := p.verifyState(current); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return errors.New("read pypi virtual environment transaction state")
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heliopause-pypi-commit-") {
			return errors.New("interrupted pypi virtual environment transaction requires explicit recovery")
		}
	}
	metadataPath := filepath.Join(p.root, pypiVenvMetadata)
	oldMetadata, metadataErr := os.ReadFile(metadataPath)
	if metadataErr != nil && !errors.Is(metadataErr, os.ErrNotExist) {
		return errors.New("read PyPI virtual environment metadata")
	}
	hadMetadata := metadataErr == nil
	for relative := range desired.Files {
		path := filepath.Join(p.site, filepath.FromSlash(relative))
		if err := rejectSymlinkPathAllowMissing(path); err != nil {
			return errors.New("pypi install destination contains an unsafe path")
		}
		if _, err := os.Lstat(path); err == nil {
			if !managed || current.Files[relative] == "" {
				return errors.New("pypi install would overwrite an unmanaged virtual environment file")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return errors.New("inspect PyPI virtual environment destination")
		}
	}
	tx, err := os.MkdirTemp(p.root, ".heliopause-pypi-commit-")
	if err != nil {
		return errors.New("create PyPI virtual environment transaction")
	}
	defer func() { _ = os.RemoveAll(tx) }()
	backed := map[string]bool{}
	published := map[string]bool{}
	rollback := func() error {
		var rollbackErr error
		if hadMetadata {
			if err := os.WriteFile(metadataPath, oldMetadata, 0o600); err != nil {
				rollbackErr = errors.Join(rollbackErr, errors.New("restore PyPI virtual environment metadata"))
			}
		} else if err := os.Remove(metadataPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			rollbackErr = errors.Join(rollbackErr, errors.New("remove uncommitted PyPI virtual environment metadata"))
		}
		for relative := range published {
			if err := os.Remove(filepath.Join(p.site, filepath.FromSlash(relative))); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, err)
			}
		}
		for relative := range backed {
			backup := filepath.Join(tx, filepath.FromSlash(relative))
			target := filepath.Join(p.site, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil || os.Rename(backup, target) != nil {
				rollbackErr = errors.Join(rollbackErr, errors.New("restore PyPI virtual environment file"))
			}
		}
		return rollbackErr
	}
	for relative := range desired.Files {
		target := filepath.Join(p.site, filepath.FromSlash(relative))
		if err := rejectSymlinkPathAllowMissing(target); err != nil {
			return errors.Join(errors.New("pypi install destination contains an unsafe path"), rollback())
		}
		if _, err := os.Lstat(target); err == nil {
			backup := filepath.Join(tx, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil || os.Rename(target, backup) != nil {
				_ = rollback()
				return errors.New("backup HAA-owned PyPI virtual environment file")
			}
			backed[relative] = true
		}
		source := filepath.Join(output, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil || os.Rename(source, target) != nil {
			return errors.Join(errors.New("publish PyPI virtual environment file"), rollback())
		}
		published[relative] = true
	}
	if err := writePypiVenvMetadata(p.root, desired); err != nil {
		return errors.Join(err, rollback())
	}
	if err := syncDirectory(p.site); err != nil {
		return errors.Join(err, rollback())
	}
	if err := os.RemoveAll(tx); err != nil {
		return errors.New("remove PyPI virtual environment rollback state")
	}
	return nil
}

type pypiVenvGuard struct{ path string }

func acquirePyPIVenvGuard(root string) (pypiVenvGuard, error) {
	path := filepath.Join(root, ".heliopause-pypi-transaction.lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return pypiVenvGuard{}, errors.New("python virtual environment is already being mutated")
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return pypiVenvGuard{}, errors.New("close PyPI virtual environment transaction lock")
	}
	return pypiVenvGuard{path: path}, nil
}

func (g pypiVenvGuard) release() error { return os.Remove(g.path) }

func writePypiVenvMetadata(root string, state pypiVenvState) error {
	directory := filepath.Join(root, ".heliopause")
	if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("create PyPI virtual environment metadata directory")
	}
	if err := trustedExistingDirectory(directory); err != nil {
		return err
	}
	body, err := json.Marshal(state)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".pypi-transaction-")
	if err != nil {
		return err
	}
	path := temporary.Name()
	if _, err := temporary.Write(body); err != nil || temporary.Sync() != nil || temporary.Close() != nil {
		_ = os.Remove(path)
		return errors.New("write PyPI virtual environment metadata")
	}
	if err := os.Rename(path, filepath.Join(root, pypiVenvMetadata)); err != nil {
		_ = os.Remove(path)
		return errors.New("publish PyPI virtual environment metadata")
	}
	return nil
}

func (p pypiVenvPlan) String() string { return fmt.Sprintf("%s:%s", p.root, p.site) }
