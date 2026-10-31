package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Python wheel copies can be large. Reuse streaming file fingerprints rather
// than reading entire wheel bodies into an in-memory cleanup inventory.
type toolPythonWorkspace struct {
	tx           *pypiVenvTransaction
	path         string
	root         *os.Root
	info         os.FileInfo
	files        map[string]pypiFileSnapshot
	directories  map[string]os.FileInfo
	closed       bool
	closeErr     error
	beforeRemove func() error // Instance-local diagnostic seam only.
}

type toolWorkspaceJournal struct {
	Root               string
	Files, Directories map[string]pypiFileSnapshot
}

func (w *toolPythonWorkspace) journal() *toolWorkspaceJournal {
	if w == nil {
		return nil
	}
	files := map[string]pypiFileSnapshot{}
	for name, snapshot := range w.files {
		files[name] = snapshot
	}
	directories := map[string]pypiFileSnapshot{}
	for name, info := range w.directories {
		device, inode, _ := pypiObjectIdentity(info)
		directories[name] = pypiFileSnapshot{Mode: info.Mode(), Device: device, Inode: inode}
	}
	return &toolWorkspaceJournal{w.path, files, directories}
}

func newToolPythonWorkspace(tx *pypiVenvTransaction) (*toolPythonWorkspace, error) {
	if tx == nil {
		return nil, errors.New("tool workspace requires the held venv transaction")
	}
	if err := tx.checkBefore(); err != nil {
		return nil, err
	}
	path, err := os.MkdirTemp(tx.plan.root, ".haa-tool-work-")
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		tx.recovery = true
		return nil, errors.New("tool workspace retained after descriptor failure")
	}
	info, err := root.Stat(".")
	if err != nil {
		tx.recovery = true
		return nil, errors.Join(err, root.Close())
	}
	workspace := &toolPythonWorkspace{tx: tx, path: path, root: root, info: info, files: map[string]pypiFileSnapshot{}, directories: map[string]os.FileInfo{".": info}}
	if err := tx.freezeParents(path); err != nil {
		return nil, errors.Join(err, workspace.Close())
	}
	return workspace, nil
}

func (w *toolPythonWorkspace) verifyBoundary() error {
	if w == nil || w.root == nil || w.closed {
		return errors.New("tool workspace is unavailable")
	}
	if err := w.tx.guard.verify(); err != nil {
		return err
	}
	info, err := os.Lstat(w.path)
	if err != nil || !os.SameFile(w.info, info) || info.Mode() != w.info.Mode() {
		return errors.New("tool workspace root changed")
	}
	opened, err := w.root.Stat(".")
	if err != nil || !os.SameFile(w.info, opened) {
		return errors.New("tool workspace descriptor changed")
	}
	return nil
}

func (w *toolPythonWorkspace) takeFile(name, digest string, limit int64) error {
	if err := w.verifyBoundary(); err != nil {
		return err
	}
	if name == "." || !fs.ValidPath(name) || filepath.IsAbs(name) {
		return errors.New("tool workspace member path is unsafe")
	}
	path := filepath.Join(w.path, filepath.FromSlash(name))
	snapshot, err := snapshotBoundedPromotionFile(path, limit)
	if err != nil || snapshot.Digest != digest {
		return errors.New("tool workspace file differs from known approved bytes")
	}
	if before, exists := w.files[name]; exists && !before.matches(path) {
		return errors.New("tool workspace input identity changed")
	}
	w.files[name] = snapshot
	for parent := filepath.ToSlash(filepath.Dir(name)); parent != "."; parent = filepath.ToSlash(filepath.Dir(parent)) {
		info, err := w.root.Lstat(parent)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("tool workspace member parent is unsafe")
		}
		if before, exists := w.directories[parent]; exists && (!os.SameFile(before, info) || before.Mode() != info.Mode()) {
			return errors.New("tool workspace member parent changed")
		}
		w.directories[parent] = info
	}
	return nil
}

func toolBytesDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (w *toolPythonWorkspace) takeInputs(bundle domain.VerifiedBundle, requirements []byte, expected map[string]pypiExpected, limits artifactpypi.WheelLimits) error {
	for _, record := range []struct {
		name string
		body []byte
	}{{manifestFilename, bundle.ManifestDocument()}, {sbomFilename, bundle.SBOMDocument()}, {"requirements.txt", requirements}} {
		if err := w.takeFile(record.name, toolBytesDigest(record.body), max(1, int64(len(record.body)))); err != nil {
			return err
		}
	}
	for _, node := range bundle.Set().Inspected().Graph().Nodes() {
		digest := expected[node.Artifact().Identity().Name()].digest
		if digest == "" {
			return errors.New("tool workspace wheel lacks inspected identity")
		}
		if err := w.takeFile("wheels/"+filepath.Base(node.RecordPath()), digest, limits.MaxCompressed); err != nil {
			return err
		}
	}
	return w.Verify()
}

func (w *toolPythonWorkspace) takeOutputs(destinations []pypiDestination) error {
	for _, d := range destinations {
		name, err := filepath.Rel(w.path, d.Source)
		if err != nil || !pathWithin(w.path, d.Source) {
			return errors.New("tool output source escapes private workspace")
		}
		if err := w.takeFile(filepath.ToSlash(name), d.Digest, max(1, d.Size)); err != nil {
			return err
		}
	}
	return w.Verify()
}

func (w *toolPythonWorkspace) Verify() error {
	if err := w.verifyBoundary(); err != nil {
		return err
	}
	count := 0
	err := fs.WalkDir(w.root.FS(), ".", func(name string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if before, exists := w.directories[name]; exists {
			info, err := w.root.Lstat(name)
			if err != nil || !os.SameFile(before, info) || before.Mode() != info.Mode() {
				return errors.New("tool workspace directory changed")
			}
		} else if before, exists := w.files[name]; !exists || !before.matches(filepath.Join(w.path, filepath.FromSlash(name))) {
			return errors.New("tool workspace contains unknown or changed data")
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if count != len(w.files)+len(w.directories) {
		return errors.New("tool workspace member is missing")
	}
	return nil
}

func (w *toolPythonWorkspace) retirePublishedSources() error {
	for name, before := range w.files {
		path := filepath.Join(w.path, filepath.FromSlash(name))
		if _, err := os.Lstat(path); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		moved := false
		for _, d := range w.tx.published {
			if d.Source == path {
				snapshot, ok := w.tx.outputs[d.Final]
				if ok && os.SameFile(before.info, snapshot.info) && before.Digest == snapshot.Digest && before.Size == snapshot.Size && before.Mode == snapshot.Mode {
					moved = true
					break
				}
			}
		}
		if !moved {
			return errors.New("tool workspace source disappeared without transaction publication")
		}
		delete(w.files, name)
	}
	return nil
}

func (w *toolPythonWorkspace) Close() (resultErr error) {
	if w == nil {
		return errors.New("tool workspace is unavailable")
	}
	if w.closed {
		return w.closeErr
	}
	defer func() {
		w.closed = true
		resultErr = errors.Join(resultErr, w.root.Close())
		w.closeErr = resultErr
		if resultErr != nil {
			w.tx.recovery = true
		}
	}()
	if err := w.retirePublishedSources(); err != nil {
		return err
	}
	if err := w.Verify(); err != nil {
		return err
	}
	if w.beforeRemove != nil {
		if err := w.beforeRemove(); err != nil {
			return err
		}
	}
	names := make([]string, 0, len(w.files)+len(w.directories))
	for name := range w.files {
		names = append(names, name)
	}
	for name := range w.directories {
		if name != "." {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, name := range names {
		if err := w.verifyBoundary(); err != nil {
			return err
		}
		if before, exists := w.files[name]; exists {
			if !before.matches(filepath.Join(w.path, filepath.FromSlash(name))) {
				return errors.New("tool workspace cleanup file changed")
			}
		} else {
			before := w.directories[name]
			info, err := w.root.Lstat(name)
			if err != nil || !os.SameFile(before, info) || before.Mode() != info.Mode() {
				return errors.New("tool workspace cleanup directory changed")
			}
		}
		if err := w.root.Remove(name); err != nil {
			return err
		}
		if _, directory := w.directories[name]; directory {
			delete(w.tx.directories, filepath.Join(w.path, filepath.FromSlash(name)))
		}
	}
	if err := w.verifyBoundary(); err != nil {
		return err
	}
	if err := w.tx.remove(w.path); err != nil {
		return err
	}
	delete(w.tx.directories, w.path)
	return syncDirectory(w.tx.plan.root)
}
