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
	"strings"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type goProjectApproval struct {
	Schema      int             `json:"schema"`
	Cache       string          `json:"cache"`
	CacheDigest string          `json:"cache_digest"`
	Approval    goCacheDocument `json:"approval"`
}

// NewGoProjectPromotion supplies the existing transaction with the
// current Go cache and an independent controller-owned retained-approval root.
func NewGoProjectPromotion(cache *GoVerifiedCache, stateRoot string) (*GoProjectPromotion, error) {
	if cache == nil || !filepath.IsAbs(stateRoot) || filepath.Clean(stateRoot) != stateRoot || stateRoot == "/" || !separateRoots([]string{cache.intakeRoot, cache.evidenceRoot, cache.cacheRoot, stateRoot}) {
		return nil, errors.New("go project promotion requires separate trusted cache and state roots")
	}
	return &GoProjectPromotion{cache: cache, stateRoot: stateRoot}, nil
}

type approvedGoProjectGuard struct {
	owner             *GoProjectPromotion
	context           domain.InstallContext
	guard             goProjectGuard
	plan              goProjectPlan
	state             *os.Root
	stateInfo         os.FileInfo
	stateName         string
	originalState     []byte
	originalStateInfo os.FileInfo
	closed            bool
	committed         bool
}

func (p *GoProjectPromotion) Begin(ctx context.Context, install domain.InstallContext) (result *approvedGoProjectGuard, resultErr error) {
	if ctx == nil || p == nil || p.cache == nil || p.stateRoot == "" || !install.Valid() {
		return nil, errors.New("go project mutation requires configured approval boundaries")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	project := install.Target().String()
	if !separateRoots([]string{project, p.cache.intakeRoot, p.cache.evidenceRoot, p.cache.cacheRoot, p.stateRoot}) {
		return nil, errors.New("go project overlaps HAA trusted state")
	}
	guard, err := acquireGoProjectGuard(project)
	if err != nil {
		return nil, err
	}
	g := &approvedGoProjectGuard{owner: p, context: install, guard: guard}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, g.Close())
			result = nil
		}
	}()
	if err := g.checkRecoveryState(); err != nil {
		return nil, err
	}
	g.plan, err = freezeGoProject(project)
	if err != nil {
		return nil, err
	}
	if err := ensureTrustedRoot(p.stateRoot); err != nil {
		return nil, err
	}
	g.stateInfo, err = os.Lstat(p.stateRoot)
	if err != nil {
		return nil, errors.New("go retained approval root unavailable")
	}
	g.state, err = os.OpenRoot(p.stateRoot)
	if err != nil {
		return nil, errors.New("open Go retained approval root")
	}
	opened, err := g.state.Stat(".")
	if err != nil || !os.SameFile(g.stateInfo, opened) {
		return nil, errors.New("go retained approval root identity changed")
	}
	projectHash := sha256.Sum256([]byte(project))
	g.stateName = hex.EncodeToString(projectHash[:]) + ".json"
	g.originalState, g.originalStateInfo, err = readGoTransactionControl(g.state, g.stateName)
	if errors.Is(err, os.ErrNotExist) {
		free, parseErr := artifactgo.ProjectDependencyFree(g.plan.controls[0].Body())
		if parseErr != nil || !free || len(g.plan.controls[1].Body()) != 0 {
			return nil, errors.New("unmanaged Go adoption requires a dependency-free project")
		}
	} else if err != nil {
		return nil, errors.New("go retained approval is unavailable")
	} else if err := g.verifyRetainedApproval(ctx); err != nil {
		return nil, err
	}
	g.plan.authorized = true
	if err := g.VerifyUnchanged(ctx); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *approvedGoProjectGuard) checkRecoveryState() error {
	directory, err := g.guard.root.Open(".")
	if err != nil {
		return errors.New("open Go project recovery inventory")
	}
	entries, readErr := directory.ReadDir(10001)
	closeErr := directory.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) > 10000 {
		return errors.New("go project recovery inventory is unavailable or exceeds bounds")
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heliopause-go-commit-") {
			return errors.New("go project has an incomplete transaction requiring recovery")
		}
	}
	if info, err := g.guard.root.Lstat(".heliopause"); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("go transaction metadata directory is untrusted")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("go transaction metadata directory is unavailable")
	}
	return nil
}

func (g *approvedGoProjectGuard) verifyRetainedApproval(ctx context.Context) error {
	var doc goProjectApproval
	decoder := json.NewDecoder(bytes.NewReader(g.originalState))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&doc) != nil {
		return errors.New("go retained approval document is invalid")
	}
	canonical, err := json.Marshal(doc)
	if err != nil || !bytes.Equal(canonical, g.originalState) || doc.Schema != 1 || doc.Approval.Schema != 1 || doc.Approval.Project+".json" != g.stateName || doc.Approval.Policy == "" || doc.Approval.PolicyVersion == 0 || len(doc.Approval.Controls) != 2 {
		return errors.New("go retained approval binding is invalid")
	}
	if _, err := domain.NewSHA256Digest(doc.Approval.Graph); err != nil {
		return errors.New("go retained graph identity is invalid")
	}
	for i, control := range g.plan.controls {
		if !control.Present() || doc.Approval.Controls[i].Path != control.Name() || doc.Approval.Controls[i].SHA256 != control.Digest().String() {
			return errors.New("go retained approval does not match current controls")
		}
	}
	if err := g.owner.cache.verifyRecordedApproval(ctx, doc.Approval); err != nil {
		return err
	}
	digest, err := domain.NewSHA256Digest(doc.CacheDigest)
	if err != nil {
		return errors.New("go retained cache identity is invalid")
	}
	_, err = g.owner.cache.openProjectCacheDocument(ctx, doc.Cache, digest, doc.Approval)
	return err
}

func (g *approvedGoProjectGuard) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), g.plan.controls...)
}

func (g *approvedGoProjectGuard) VerifyUnchanged(ctx context.Context) error {
	if g == nil || ctx == nil || g.closed || g.committed {
		return errors.New("go project guard is not active")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := g.guard.verify(); err != nil {
		return err
	}
	if err := g.plan.verifyUnchanged(); err != nil {
		return err
	}
	info, err := os.Lstat(g.owner.stateRoot)
	if err != nil || !os.SameFile(g.stateInfo, info) {
		return errors.New("go retained approval root changed")
	}
	current, stateInfo, err := readGoTransactionControl(g.state, g.stateName)
	if len(g.originalState) == 0 && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !os.SameFile(g.originalStateInfo, stateInfo) || !bytes.Equal(current, g.originalState) {
		return errors.New("go retained approval changed during transaction")
	}
	return nil
}

func sameGoTransactionControls(left, right []domain.ProjectControlFile) bool {
	if len(left) != 2 || len(right) != 2 {
		return false
	}
	for i, file := range left {
		if file.Name() != right[i].Name() || file.Present() != right[i].Present() || file.Digest() != right[i].Digest() {
			return false
		}
	}
	return true
}

func (g *approvedGoProjectGuard) Commit(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) (resultErr error) {
	if g == nil || !update.Valid() || !staged.Valid() || update.Snapshot().Context() != g.context || update.Snapshot().Source() != artifactgo.Source() || !sameGoTransactionControls(g.plan.controls, update.OriginalControls()) {
		return errors.New("go transaction update differs from guarded project")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	snapshot := staged.Set().Inspected().Snapshot()
	if snapshot.Context() != update.Snapshot().Context() || snapshot.Source() != update.Snapshot().Source() || snapshot.GraphDigest() != update.Snapshot().GraphDigest() || len(snapshot.Dependencies()) != len(update.Snapshot().Dependencies()) {
		return errors.New("go transaction cache approval differs from selected graph")
	}
	selectedDependencies := update.Snapshot().Dependencies()
	for i, artifact := range snapshot.Dependencies() {
		if artifact != selectedDependencies[i] {
			return errors.New("go transaction cache dependency differs from selected graph")
		}
	}
	if _, err := g.owner.cache.OpenProjectCache(ctx, staged); err != nil {
		return err
	}
	doc, err := g.owner.cache.goCacheApproval(ctx, staged.Set())
	if err != nil {
		return err
	}
	if len(doc.Controls) != len(update.SelectedControls()) {
		return errors.New("go transaction approval controls are incomplete")
	}
	for i, control := range update.SelectedControls() {
		if doc.Controls[i].Path != control.Name() || doc.Controls[i].SHA256 != control.Digest().String() {
			return errors.New("go transaction approval controls differ from selection")
		}
	}
	approval, err := json.Marshal(goProjectApproval{1, staged.ContentHandle(), staged.Digest().String(), doc})
	if err != nil || len(approval) > artifactgo.MaxProjectControlBytes {
		return errors.New("go transaction approval exceeds bounds")
	}
	workspace, err := g.plan.privateWorkspace()
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(workspace); err != nil {
			resultErr = errors.Join(resultErr, errors.New("dispose Go transaction workspace"))
		}
	}()
	for _, control := range update.SelectedControls() {
		if !control.Present() || os.WriteFile(filepath.Join(workspace, control.Name()), control.Body(), 0o600) != nil {
			return errors.New("materialize selected Go transaction controls")
		}
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	transaction, err := beginGoProjectTransaction(g.plan, workspace, update.SelectedControls()...)
	if err != nil {
		return err
	}
	published := false
	transaction.publishApproval = func() error {
		current, info, err := readGoTransactionControl(g.state, g.stateName)
		if g.originalStateInfo == nil {
			if !errors.Is(err, os.ErrNotExist) {
				return errors.New("go retained approval appeared before publication")
			}
		} else if err != nil || !os.SameFile(g.originalStateInfo, info) || !bytes.Equal(current, g.originalState) {
			return errors.New("go retained approval changed before publication")
		}
		return g.writeState(approval, &published)
	}
	transaction.rollbackApproval = func() error {
		if !published {
			return nil
		}
		current, _, err := readGoTransactionControl(g.state, g.stateName)
		if err != nil || !bytes.Equal(current, approval) {
			return errors.New("go approval rollback identity is uncertain")
		}
		if len(g.originalState) == 0 {
			if err := g.state.Remove(g.stateName); err != nil {
				return errors.New("remove uncommitted Go approval")
			}
			return syncDirectory(g.owner.stateRoot)
		}
		var restored bool
		return g.writeState(g.originalState, &restored)
	}
	if err := transaction.commit(); err != nil {
		return err
	}
	g.committed = true
	return nil
}

func (g *approvedGoProjectGuard) writeState(body []byte, published *bool) (resultErr error) {
	info, err := os.Lstat(g.owner.stateRoot)
	if err != nil || !os.SameFile(g.stateInfo, info) {
		return errors.New("go retained approval root changed before publication")
	}
	id, err := domain.NewRunID()
	if err != nil {
		return err
	}
	name := ".go-approval-" + id.String()
	f, err := g.state.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create private Go approval record")
	}
	defer func() {
		if err := g.state.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, errors.New("dispose private Go approval record"))
		}
	}()
	_, writeErr := f.Write(body)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("persist private Go approval record")
	}
	if len(g.originalState) == 0 {
		if err := g.state.Link(name, g.stateName); err != nil {
			return errors.New("publish initial trusted Go project approval")
		}
	} else if err := g.state.Rename(name, g.stateName); err != nil {
		return errors.New("publish trusted Go project approval")
	}
	*published = true
	return syncDirectory(g.owner.stateRoot)
}

func (g *approvedGoProjectGuard) Close() error {
	if g == nil || g.closed {
		return errors.New("go project guard already closed or unavailable")
	}
	g.closed = true
	var result error
	if g.state != nil {
		result = g.state.Close()
	}
	return errors.Join(result, g.guard.release())
}
