package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// NewRecordedGitHubReleasePromotion extends the existing publisher with an
// independent ownership record. Its old constructor remains strictly new-only.
func NewRecordedGitHubReleasePromotion(staging, evidence, state string, reader ProjectEvidenceReader) (*GitHubReleasePromotion, error) {
	if reader == nil || !separateRoots([]string{staging, evidence, state}) {
		return nil, errors.New("GitHub ownership requires separate staging/Evidence/state roots")
	}
	for _, path := range []string{staging, evidence, state} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
			return nil, errors.New("GitHub ownership root is invalid")
		}
	}
	p, err := NewGitHubReleasePromotion(staging)
	if err != nil {
		return nil, err
	}
	p.stateRoot, p.evidenceRoot, p.evidence = state, evidence, reader
	return p, nil
}

type GitHubReleaseUpdatePreparation struct {
	promoter *GitHubReleasePromotion
	resolver ports.DependencyResolver
}

func NewGitHubReleaseUpdatePreparation(p *GitHubReleasePromotion, resolver ports.DependencyResolver) (*GitHubReleaseUpdatePreparation, error) {
	if p == nil || p.stateRoot == "" || p.evidence == nil || resolver == nil {
		return nil, errors.New("GitHub replacement requires recorded publisher and existing resolver")
	}
	return &GitHubReleaseUpdatePreparation{p, resolver}, nil
}

func (p *GitHubReleaseUpdatePreparation) BeginInstallUpdate(ctx context.Context, source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (ports.InstallUpdateSession, error) {
	if p == nil || source.String() != artifactgithub.SourceName || install.Mode() != domain.InstallGitHubAsset || adopt {
		return nil, errors.New("GitHub replacement requires explicit managed-asset intent without adoption")
	}
	s, err := p.promoter.begin(ctx, install, selector, true)
	if err != nil {
		return nil, err
	}
	s.resolver = p.resolver
	return s, nil
}

type githubReplacementSession struct {
	lock                                                                                          *os.File
	owner                                                                                         *GitHubReleasePromotion
	install                                                                                       domain.InstallContext
	reference                                                                                     domain.ArtifactReference
	resolver                                                                                      ports.DependencyResolver
	resolution                                                                                    domain.DependencyResolution
	guard                                                                                         goProjectGuard
	state                                                                                         *os.Root
	stateInfo                                                                                     os.FileInfo
	stateName                                                                                     string
	originalState, expectedState                                                                  []byte
	expectedStateInfo                                                                             os.FileInfo
	original, selected                                                                            githubAssetApproval
	selectedBody                                                                                  []byte
	originalTree, selectedTree                                                                    *githubDirectorySnapshot
	selectedPath, backupPath                                                                      string
	selectedInfo, backupInfo                                                                      os.FileInfo
	replace, resolved, attempted, originalMoved, targetPublished, stateChanged, committed, closed bool
	closeErr                                                                                      error
}

func (p *GitHubReleasePromotion) begin(ctx context.Context, install domain.InstallContext, selector string, replace bool) (result *githubReplacementSession, resultErr error) {
	if p == nil || p.stateRoot == "" || p.evidence == nil || ctx == nil || !install.Valid() || replace && install.Mode() != domain.InstallGitHubAsset || !replace && !install.RequiresNewTarget() {
		return nil, errors.New("GitHub installation intent is invalid")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target := install.Target().String()
	if !separateRoots([]string{target, p.stagingRoot, p.evidenceRoot, p.stateRoot}) {
		return nil, errors.New("GitHub target overlaps trusted storage")
	}
	s := &githubReplacementSession{owner: p, install: install, replace: replace}
	hash := sha256.Sum256([]byte(target))
	id := hex.EncodeToString(hash[:])
	s.stateName = id + ".json"
	var err error
	s.guard, err = acquireProjectTransactionGuard(filepath.Dir(target), ".heliopause-github-"+id+".lock")
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, s.Close())
			result = nil
		}
	}()
	// Keeping the original inode open prevents unlink/recreate ABA reuse.
	s.lock, err = s.guard.root.OpenFile(filepath.Base(s.guard.path), os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	if err := s.verifyLock(); err != nil {
		return nil, err
	}
	if err := s.checkRecovery(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(p.stateRoot, 0o700); err != nil {
		return nil, err
	}
	if trustedExistingDirectory(p.stateRoot) != nil {
		return nil, errors.New("GitHub independent state root is untrusted")
	}
	s.stateInfo, err = os.Lstat(p.stateRoot)
	if err != nil || s.stateInfo.Mode().Perm() != 0o700 {
		return nil, errors.New("GitHub independent state root permissions are invalid")
	}
	s.state, err = os.OpenRoot(p.stateRoot)
	if err != nil {
		return nil, err
	}
	body, info, err := readGoTransactionControl(s.state, s.stateName)
	if errors.Is(err, os.ErrNotExist) {
		body, info, err = nil, nil, nil
	}
	if err != nil || info != nil && (!pypiSingleLink(info) || info.Mode().Perm() != 0o600) {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", errors.New("GitHub independent approval is unavailable"))
	}
	s.originalState, s.expectedState, s.expectedStateInfo = body, body, info
	if replace {
		if info == nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", errors.New("GitHub target has no independent ownership approval"))
		}
		s.original, err = decodeGitHubApproval(body)
		if err != nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
		}
		if err := p.verifyApproval(ctx, s.original); err != nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
		}
		if s.original.Target != target {
			return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", errors.New("GitHub approval belongs to a different target"))
		}
		s.originalTree, err = snapshotGitHubDirectory(target, s.original, body)
		if err != nil {
			return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
		}
		s.reference, err = artifactgithub.ParseReference(selector)
		if err != nil {
			return nil, err
		}
		old, err := artifactgithub.ParseSelector(s.original.Reference)
		if err != nil {
			return nil, err
		}
		next, err := artifactgithub.ParseSelector(s.reference.Locator())
		if err != nil {
			return nil, err
		}
		if old.Owner() != next.Owner() || old.Repo() != next.Repo() {
			return nil, ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("GitHub replacement cannot change repository"))
		}
	} else {
		if info != nil {
			return nil, errors.New("GitHub target has an existing or incomplete independent approval")
		}
		if _, err := s.guard.root.Lstat(filepath.Base(target)); !errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("GitHub Release target already exists")
		}
	}
	if err := s.verify(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *githubReplacementSession) Reference() domain.ArtifactReference { return s.reference }
func (s *githubReplacementSession) recoveryPrefix() string {
	return "." + filepath.Base(s.install.Target().String()) + ".haa-github-commit-"
}

func (s *githubReplacementSession) checkRecovery() error {
	f, err := s.guard.root.Open(".")
	if err != nil {
		return err
	}
	entries, re := f.ReadDir(10001)
	ce := f.Close()
	if re != nil && !errors.Is(re, io.EOF) || ce != nil || len(entries) > 10000 {
		return errors.New("GitHub parent recovery inventory unavailable or exceeds bounds")
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), s.recoveryPrefix()) {
			return errors.New("GitHub target has incomplete transaction recovery state")
		}
	}
	return nil
}

func (s *githubReplacementSession) verifyState() error {
	if s.state == nil || trustedExistingDirectory(s.owner.stateRoot) != nil {
		return errors.New("GitHub state boundary unavailable")
	}
	current, err := os.Lstat(s.owner.stateRoot)
	opened, oe := s.state.Stat(".")
	if err != nil || oe != nil || !os.SameFile(s.stateInfo, current) || !os.SameFile(current, opened) || current.Mode() != s.stateInfo.Mode() {
		return errors.New("GitHub independent state root changed")
	}
	body, info, err := readGoTransactionControl(s.state, s.stateName)
	if s.expectedStateInfo == nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errors.New("GitHub independent approval appeared")
	}
	if err != nil || !pypiSingleLink(info) || !os.SameFile(s.expectedStateInfo, info) || info.Mode() != s.expectedStateInfo.Mode() || !info.ModTime().Equal(s.expectedStateInfo.ModTime()) || !bytes.Equal(body, s.expectedState) {
		return errors.New("GitHub independent approval changed")
	}
	return nil
}

func (s *githubReplacementSession) verifyLock() error {
	if s.lock == nil || s.guard.root == nil {
		return errors.New("GitHub original lock descriptor unavailable")
	}
	current, err := s.guard.root.Lstat(filepath.Base(s.guard.path))
	held, heldErr := s.lock.Stat()
	if err != nil || heldErr != nil || !os.SameFile(s.guard.fileInfo, current) || !os.SameFile(current, held) || !pypiSingleLink(held) || held.Size() != 0 || held.Mode().Perm() != 0o600 || !held.ModTime().Equal(s.guard.fileInfo.ModTime()) {
		return errors.New("GitHub original lock identity or content changed")
	}
	return s.guard.verify()
}

func (s *githubReplacementSession) verify(ctx context.Context) error {
	if s == nil || s.closed || ctx == nil {
		return errors.New("GitHub original-state session is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.verifyLock(); err != nil {
		return err
	}
	parent, err := os.Lstat(filepath.Dir(s.install.Target().String()))
	if err != nil || parent.Mode() != s.guard.rootInfo.Mode() {
		return errors.New("GitHub target parent permissions changed")
	}
	if err := s.verifyState(); err != nil {
		return err
	}
	if s.replace {
		if err := s.owner.verifyApproval(ctx, s.original); err != nil {
			return err
		}
	}
	if s.targetPublished {
		if err := s.selectedTree.verify(s.install.Target().String()); err != nil {
			return err
		}
	} else if s.originalMoved {
		if _, err := s.guard.root.Lstat(filepath.Base(s.install.Target().String())); !errors.Is(err, os.ErrNotExist) {
			return errors.New("GitHub target appeared while original is backed up")
		}
	} else if s.replace {
		if err := s.originalTree.verify(s.install.Target().String()); err != nil {
			return err
		}
	} else {
		if _, err := s.guard.root.Lstat(filepath.Base(s.install.Target().String())); !errors.Is(err, os.ErrNotExist) {
			return errors.New("GitHub new target appeared")
		}
	}
	if s.originalMoved {
		if err := s.verifyBackup(); err != nil {
			return err
		}
	}
	return nil
}

func (s *githubReplacementSession) ResolveDependencies(ctx context.Context, ref domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	if s == nil || s.resolver == nil || s.resolved || ref != s.reference || install != s.install {
		return domain.DependencyResolution{}, errors.New("GitHub replacement selection intent changed")
	}
	if err := s.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	selected, err := s.resolver.ResolveDependencies(ctx, ref, install)
	if err != nil {
		return selected, err
	}
	if err := s.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	nodes := selected.Graph().Nodes()
	if len(nodes) != 1 || len(selected.Graph().Edges()) != 0 {
		return domain.DependencyResolution{}, errors.New("GitHub replacement requires one frozen standalone selection")
	}
	selector, _, err := artifactgithub.SelectionForResolved(nodes[0].Artifact())
	if err != nil || selector.Locator() != ref.Locator() {
		return domain.DependencyResolution{}, errors.New("GitHub resolver changed requested repository/tag/asset")
	}
	s.resolution, s.resolved = selected, true
	return selected, nil
}

func (s *githubReplacementSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	if s == nil || !s.resolved || s.attempted || s.closed || install != s.install || !bundle.Valid() {
		return domain.PromotedInstall{}, errors.New("GitHub replacement requires its held frozen selection")
	}
	nodes := bundle.Set().Inspected().Graph().Nodes()
	selected := s.resolution.Graph().Nodes()
	if len(nodes) != 1 || len(selected) != 1 || nodes[0] != selected[0] {
		return domain.PromotedInstall{}, errors.New("GitHub replacement bundle differs from frozen selection")
	}
	s.attempted = true
	return s.owner.promote(ctx, staged, bundle, install, s)
}

func (s *githubReplacementSession) check(phase string) error {
	if s.owner.checkpoint != nil {
		return s.owner.checkpoint(phase)
	}
	return nil
}

func (s *githubReplacementSession) recordSelected() error {
	root, err := os.OpenRoot(s.selectedPath)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.OpenFile(githubOwnershipMarker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, we := f.Write(s.selectedBody)
	se := f.Sync()
	ce := f.Close()
	if we != nil || se != nil || ce != nil {
		return errors.New("persist GitHub installed ownership marker")
	}
	s.selectedTree, err = snapshotGitHubDirectory(s.selectedPath, s.selected, s.selectedBody)
	return err
}

func (s *githubReplacementSession) beforePublish(ctx context.Context) error {
	if err := s.check("before-original-move"); err != nil {
		return err
	}
	if err := s.verify(ctx); err != nil {
		return err
	}
	if err := s.selectedTree.verify(s.selectedPath); err != nil {
		return err
	}
	if !s.replace {
		return nil
	}
	path, err := os.MkdirTemp(filepath.Dir(s.install.Target().String()), s.recoveryPrefix())
	if err != nil {
		return err
	}
	s.backupPath = path
	s.backupInfo, err = os.Lstat(path)
	if err != nil {
		return err
	}
	if err := s.verify(ctx); err != nil {
		return err
	}
	if err := renameRootNoReplace(s.guard.root, filepath.Base(s.install.Target().String()), filepath.Base(path)+"/original"); err != nil {
		return err
	}
	s.originalMoved = true
	if err := syncDirectory(path); err != nil {
		return err
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	if err := s.check("after-original-move"); err != nil {
		return err
	}
	return s.verify(ctx)
}

func (s *githubReplacementSession) verifyBackup() error {
	if s.backupPath == "" {
		return errors.New("GitHub original backup unavailable")
	}
	info, err := os.Lstat(s.backupPath)
	if err != nil || !os.SameFile(s.backupInfo, info) || info.Mode() != s.backupInfo.Mode() {
		return errors.New("GitHub backup boundary changed")
	}
	root, err := os.OpenRoot(s.backupPath)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, re := f.ReadDir(2)
	ce := f.Close()
	if re != nil && !errors.Is(re, io.EOF) || ce != nil || len(entries) != 1 || entries[0].Name() != "original" {
		return errors.New("GitHub backup contains foreign or missing members")
	}
	return s.originalTree.verify(filepath.Join(s.backupPath, "original"))
}

func (s *githubReplacementSession) publishState(body []byte) (resultErr error) {
	if err := s.verifyState(); err != nil {
		return err
	}
	id, err := domain.NewRunID()
	if err != nil {
		return err
	}
	name := ".github-approval-" + id.String()
	f, err := s.state.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, we := f.Write(body)
	se := f.Sync()
	info, ie := f.Stat()
	ce := f.Close()
	if we != nil || se != nil || ie != nil || ce != nil {
		return errors.New("persist GitHub independent approval")
	}
	defer func() {
		if current, err := s.state.Lstat(name); err == nil {
			if !os.SameFile(info, current) {
				resultErr = errors.Join(resultErr, errors.New("private GitHub approval changed"))
				return
			}
			resultErr = errors.Join(resultErr, s.state.Remove(name))
		} else if !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	if err := s.verifyState(); err != nil {
		return err
	}
	if s.expectedStateInfo == nil {
		if err := s.state.Link(name, s.stateName); err != nil {
			return err
		}
		s.stateChanged = true
		s.expectedStateInfo = info
		s.expectedState = append([]byte(nil), body...)
		if err := s.state.Remove(name); err != nil {
			return err
		}
	} else {
		if err := s.state.Rename(name, s.stateName); err != nil {
			return err
		}
		s.stateChanged = true
		s.expectedStateInfo = info
		s.expectedState = append([]byte(nil), body...)
	}
	return syncDirectory(s.owner.stateRoot)
}

func (s *githubReplacementSession) afterPublish(ctx context.Context) error {
	if err := s.check("after-target-publish"); err != nil {
		return err
	}
	if err := s.verify(ctx); err != nil {
		return err
	}
	if err := s.owner.verifyApproval(ctx, s.selected); err != nil {
		return err
	}
	if err := s.check("before-approval-publish"); err != nil {
		return err
	}
	if err := s.verify(ctx); err != nil {
		return err
	}
	if err := s.publishState(s.selectedBody); err != nil {
		return err
	}
	if err := s.check("after-approval-publish"); err != nil {
		return err
	}
	if err := s.verify(ctx); err != nil {
		return err
	}
	if err := s.check("before-backup-cleanup"); err != nil {
		return err
	}
	if err := s.verify(ctx); err != nil {
		return err
	}
	s.committed = true // Old cleanup is separate from confirmed publication/proof.
	if s.originalMoved {
		if err := s.originalTree.remove(filepath.Join(s.backupPath, "original")); err != nil {
			return err
		}
		if err := s.removeEmptyBackup(); err != nil {
			return err
		}
		s.originalMoved = false
	}
	if err := s.check("after-backup-cleanup"); err != nil {
		return err
	}
	return s.verify(ctx)
}

func (s *githubReplacementSession) disposeSelected() error {
	if s.selectedPath == "" || s.targetPublished {
		return nil
	}
	if s.selectedTree != nil {
		if err := s.selectedTree.remove(s.selectedPath); err != nil {
			return err
		}
		s.selectedPath = ""
		return nil
	}
	info, err := os.Lstat(s.selectedPath)
	if err != nil || s.selectedInfo == nil || !os.SameFile(s.selectedInfo, info) {
		return errors.New("GitHub incomplete selected directory cannot be safely removed")
	}
	root, err := os.OpenRoot(s.selectedPath)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, re := f.ReadDir(1)
	ce := f.Close()
	if re != nil && !errors.Is(re, io.EOF) || ce != nil || len(entries) != 0 {
		return errors.New("GitHub incomplete selected members retained for recovery")
	}
	return s.guard.root.Remove(filepath.Base(s.selectedPath))
}

func (s *githubReplacementSession) removeEmptyBackup() error {
	if s.backupPath == "" {
		return nil
	}
	info, err := s.guard.root.Lstat(filepath.Base(s.backupPath))
	if err != nil || !os.SameFile(s.backupInfo, info) {
		return errors.New("GitHub backup identity changed before disposal")
	}
	if err := s.guard.root.Remove(filepath.Base(s.backupPath)); err != nil {
		return err
	}
	s.backupPath = ""
	return syncDirectory(filepath.Dir(s.install.Target().String()))
}

func (s *githubReplacementSession) abort() error {
	if s == nil || s.committed {
		return nil
	}
	if err := s.verify(context.Background()); err != nil {
		return errors.Join(err, errors.New("GitHub transaction preserved for recovery"))
	}
	if s.selectedTree != nil && s.selectedPath != "" {
		if err := s.selectedTree.verify(s.selectedPath); err != nil {
			return err
		}
	}
	if s.stateChanged {
		if len(s.originalState) == 0 {
			if err := s.state.Remove(s.stateName); err != nil {
				return err
			}
			s.expectedState, s.expectedStateInfo = nil, nil
		} else if err := s.publishState(s.originalState); err != nil {
			return err
		}
		if err := syncDirectory(s.owner.stateRoot); err != nil {
			return err
		}
	}
	if s.targetPublished {
		if err := s.selectedTree.remove(s.install.Target().String()); err != nil {
			return err
		}
		s.targetPublished = false
		s.selectedPath = ""
	}
	if s.originalMoved {
		if err := renameRootNoReplace(s.guard.root, filepath.Base(s.backupPath)+"/original", filepath.Base(s.install.Target().String())); err != nil {
			return err
		}
		s.originalMoved = false
		if err := syncDirectory(filepath.Dir(s.install.Target().String())); err != nil {
			return err
		}
	}
	if err := s.disposeSelected(); err != nil {
		return err
	}
	if err := s.removeEmptyBackup(); err != nil {
		return err
	}
	return s.verify(context.Background())
}

func (s *githubReplacementSession) Close() error {
	if s == nil {
		return errors.New("GitHub session unavailable")
	}
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	if s.state != nil {
		s.closeErr = errors.Join(s.closeErr, s.state.Close())
	}
	if s.guard.root != nil {
		if err := s.verifyLock(); err != nil {
			s.closeErr = errors.Join(s.closeErr, err, s.guard.root.Close())
		} else {
			s.closeErr = errors.Join(s.closeErr, s.guard.release())
		}
	}
	if s.lock != nil {
		s.closeErr = errors.Join(s.closeErr, s.lock.Close())
	}
	return s.closeErr
}
