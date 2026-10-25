package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const uvProjectTransactionLock = ".heliopause-uv-project-transaction.lock"

// Controller project files are not distribution-owned installation outputs.
// Their publication shares the venv's existing journal/backup/rollback engine.
type uvProjectFileTransaction struct {
	root     string
	guard    goProjectGuard
	input    artifactpypi.UVProjectInput
	selected []pypiDestination
	cleanup  func() error
}

type uvProjectJournalControls struct {
	Root  string
	Files []pypiDestination
}

func (p *uvProjectFileTransaction) close(recovery bool) error {
	if !recovery {
		return p.guard.release()
	}
	return errors.Join(p.guard.verify(), p.guard.file.Close(), p.guard.root.Close())
}

func freezeUVProjectFiles(ctx context.Context, tx *pypiVenvTransaction, root string, intent artifactpypi.UVProjectIntent, selector string, trustedRoots []string) (_ artifactpypi.UVProjectInput, resultErr error) {
	if ctx == nil || tx == nil || tx.project != nil || len(trustedRoots) == 0 || !filepath.IsAbs(root) || filepath.Clean(root) != root || root == "/" {
		return artifactpypi.UVProjectInput{}, errors.New("UV project requires one guarded canonical source")
	}
	if err := ctx.Err(); err != nil {
		return artifactpypi.UVProjectInput{}, err
	}
	paths := []string{filepath.Join(root, artifactpypi.UVProjectInputControl), filepath.Join(root, artifactpypi.UVNativeLockControl)}
	roots := append(append([]string{tx.plan.root}, paths...), trustedRoots...)
	for _, path := range roots {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || rejectSymlinkPathAllowMissing(path) != nil {
			return artifactpypi.UVProjectInput{}, errors.New("UV project boundaries are not canonical")
		}
	}
	if !separateRoots(roots) {
		return artifactpypi.UVProjectInput{}, errors.New("UV project controls overlap the venv or trusted storage")
	}
	if err := tx.checkBefore(); err != nil {
		return artifactpypi.UVProjectInput{}, err
	}
	guard, err := acquireProjectTransactionGuard(root, uvProjectTransactionLock)
	if err != nil {
		return artifactpypi.UVProjectInput{}, err
	}
	accepted := false
	defer func() {
		if !accepted {
			resultErr = errors.Join(resultErr, guard.release())
		}
	}()
	if err := tx.freezeParents(root); err != nil {
		return artifactpypi.UVProjectInput{}, err
	}
	bodies := make([][]byte, 2)
	snapshots := make([]*pypiFileSnapshot, 2)
	for i, path := range paths {
		if _, exists := tx.before[path]; exists {
			return artifactpypi.UVProjectInput{}, errors.New("UV project control was already frozen")
		}
		if _, err := os.Lstat(path); i == 1 && errors.Is(err, os.ErrNotExist) {
			continue
		}
		snapshot, err := snapshotBoundedPromotionFile(path, 4<<20)
		if err != nil {
			return artifactpypi.UVProjectInput{}, err
		}
		body, err := readBoundedPromotionFile(path, 4<<20)
		if err != nil {
			return artifactpypi.UVProjectInput{}, err
		}
		sum := sha256.Sum256(body)
		if snapshot.Digest != hex.EncodeToString(sum[:]) || !snapshot.matches(path) {
			return artifactpypi.UVProjectInput{}, errors.New("UV project source changed during capture")
		}
		bodies[i], snapshots[i] = body, &snapshot
	}
	input, err := artifactpypi.ParseUVProjectInput(bodies[0], bodies[1], snapshots[1] != nil, intent, selector)
	if err != nil {
		return artifactpypi.UVProjectInput{}, err
	}
	for i, path := range paths {
		tx.before[path] = snapshots[i]
	}
	tx.project = &uvProjectFileTransaction{root: root, guard: guard, input: input}
	accepted = true
	if err := ctx.Err(); err != nil {
		return artifactpypi.UVProjectInput{}, err
	}
	return input, tx.checkBefore()
}

// The approved publication caller supplies exactly the two selected project
// files. This private filesystem primitive does not confer Policy approval.
func (t *pypiVenvTransaction) commitCompleteProject(destinations []pypiDestination, dependencyFree bool, controls []domain.ProjectControlFile) (resultErr error) {
	if t.project == nil || len(t.project.selected) != 0 || len(controls) != 2 {
		return errors.New("UV project commit requires its single frozen two-file selection")
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, c := range controls {
		if !c.Present() {
			return errors.New("UV project selected control is absent")
		}
		byName[c.Name()] = c
	}
	manifest, ok := byName[artifactpypi.UVProjectInputControl]
	lock, locked := byName[artifactpypi.UVNativeLockControl]
	if !ok || !locked {
		return errors.New("UV project selection changed control names")
	}
	input, err := t.project.input.SelectManifest(manifest.Body())
	if err != nil {
		return err
	}
	if _, err := artifactpypi.ParseUVNativeProjectLock(input, lock.Body()); err != nil {
		return err
	}
	if err := t.checkBefore(); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(t.plan.root, ".haa-uv-project-controls-")
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(t.plan.root)
	if err != nil {
		t.recovery = true
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(t.directories[t.plan.root], opened) {
		t.recovery = true
		return errors.New("project control temporary parent changed")
	}
	directory := filepath.Base(temporary)
	info, err := root.Lstat(directory)
	if err != nil {
		t.recovery = true
		return err
	}
	records := map[string]controlBackupRecord{}
	cleanupDone := false
	var cleanupErr error
	cleanup := func() error {
		if cleanupDone {
			return cleanupErr
		}
		cleanupDone = true
		// Moved outputs are absent here even after a successful rollback.
		for _, d := range t.published {
			if d.Scheme == "project-control" && filepath.Dir(d.Source) == temporary {
				delete(records, filepath.Base(d.Source))
			}
		}
		err := t.verifyPathIdentity(t.plan.root)
		if err == nil {
			err = removeControlBackup(root, directory, info, nil, records)
		}
		if err == nil {
			delete(t.directories, temporary)
			err = syncDirectory(t.plan.root)
		}
		if err != nil {
			t.recovery = true
			cleanupErr = errors.Join(err, errors.New("project control cleanup uncertain; guards and data retained"))
		}
		return cleanupErr
	}
	t.project.cleanup = cleanup
	defer func() {
		if !cleanupDone && !t.recovery {
			resultErr = errors.Join(resultErr, cleanup())
		}
	}()
	selected := []pypiDestination{}
	for _, c := range []domain.ProjectControlFile{manifest, lock} {
		path := filepath.Join(temporary, c.Name())
		if err := writePromotionFile(path, c.Body()); err != nil {
			return err
		}
		if before := t.before[filepath.Join(t.project.root, c.Name())]; before != nil {
			if err := os.Chmod(path, before.Mode.Perm()); err != nil {
				return err
			}
		}
		body, member, err := readGoTransactionControl(root, filepath.Join(directory, c.Name()))
		sum := sha256.Sum256(body)
		if err != nil || hex.EncodeToString(sum[:]) != c.Digest().String() {
			return errors.New("private project control copy changed")
		}
		records[c.Name()] = controlBackupRecord{member, sum}
		// Hash and inode are frozen again in the common prepare boundary.
		selected = append(selected, pypiDestination{Scheme: "project-control", Relative: c.Name(), Final: filepath.Join(t.project.root, c.Name()), Source: path, Digest: c.Digest().String(), Size: int64(len(c.Body()))})
	}
	t.project.selected = selected
	return t.commitSelectionWithProject(destinations, true, dependencyFree)
}

func (t *pypiVenvTransaction) prepareProjectControls() ([]pypiDestination, error) {
	if t.project == nil {
		return nil, nil
	}
	if len(t.project.selected) != 2 {
		return nil, errors.New("project controls require explicit complete project publication")
	}
	for _, d := range t.project.selected {
		if d.Scheme != "project-control" || d.Final != filepath.Join(t.project.root, d.Relative) || (d.Relative != artifactpypi.UVProjectInputControl && d.Relative != artifactpypi.UVNativeLockControl) {
			return nil, errors.New("project control destination changed")
		}
		actual, err := snapshotPyPIFile(d.Source)
		if err != nil || actual.Digest != d.Digest || actual.Size != d.Size {
			return nil, errors.New("private project control changed")
		}
		if _, frozen := t.before[d.Final]; !frozen {
			return nil, errors.New("project control lacks frozen presence")
		}
		if err := t.freezeParents(filepath.Dir(d.Source)); err != nil {
			return nil, err
		}
		t.outputs[d.Final] = actual
	}
	return append([]pypiDestination(nil), t.project.selected...), nil
}

func (t *pypiVenvTransaction) projectJournal() *uvProjectJournalControls {
	if t.project == nil {
		return nil
	}
	return &uvProjectJournalControls{Root: t.project.root, Files: append([]pypiDestination(nil), t.project.selected...)}
}
