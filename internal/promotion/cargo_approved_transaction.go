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

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// CargoProjectPromotion publishes only an inspected frozen graph; it has no SDK execution capability.
type CargoProjectPromotion struct {
	cache      *CargoVerifiedCache
	stateRoot  string
	checkpoint func(string) error
}

type cargoProjectApproval struct {
	Schema         int                `json:"schema"`
	Cache          string             `json:"cache"`
	CacheDigest    string             `json:"cache_digest"`
	Approval       cargoCacheDocument `json:"approval"`
	LocalManifests map[string]string  `json:"local_manifests"`
}

// NewCargoProjectPromotion supplies the existing transaction with the
// current Cargo cache and an independent controller-owned retained-approval root.
func NewCargoProjectPromotion(cache *CargoVerifiedCache, stateRoot string) (*CargoProjectPromotion, error) {
	if cache == nil || !filepath.IsAbs(stateRoot) || filepath.Clean(stateRoot) != stateRoot || stateRoot == "/" || !separateRoots([]string{cache.intakeRoot, cache.evidenceRoot, cache.cacheRoot, stateRoot}) {
		return nil, errors.New("cargo project promotion requires separate trusted cache and state roots")
	}
	return &CargoProjectPromotion{cache: cache, stateRoot: stateRoot}, nil
}

type approvedCargoProjectGuard struct {
	owner             *CargoProjectPromotion
	context           domain.InstallContext
	guard             goProjectGuard
	plan              cargoProjectPlan
	state             *os.Root
	stateInfo         os.FileInfo
	stateName         string
	originalState     []byte
	originalStateInfo os.FileInfo
	closed            bool
	committed         bool
	source            *sandbox.CargoProjectSource
}

func (p *CargoProjectPromotion) Begin(ctx context.Context, install domain.InstallContext) (result *approvedCargoProjectGuard, resultErr error) {
	if ctx == nil || p == nil || p.cache == nil || p.stateRoot == "" || !install.Valid() {
		return nil, errors.New("cargo project mutation requires configured approval boundaries")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	project := install.Target().String()
	if !separateRoots([]string{project, p.cache.intakeRoot, p.cache.evidenceRoot, p.cache.cacheRoot, p.stateRoot}) {
		return nil, errors.New("cargo project overlaps HAA trusted state")
	}
	guard, err := acquireProjectTransactionGuard(project, ".heliopause-cargo-transaction.lock")
	if err != nil {
		return nil, err
	}
	g := &approvedCargoProjectGuard{owner: p, context: install, guard: guard}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, g.Close())
			result = nil
		}
	}()
	if err := g.checkRecoveryState(); err != nil {
		return nil, err
	}
	g.plan, err = freezeCargoProject(project)
	if err != nil {
		return nil, err
	}
	g.source, err = sandbox.CaptureCargoProjectSource(ctx, project)
	if err != nil {
		return nil, err
	}
	if err := ensureTrustedRoot(p.stateRoot); err != nil {
		return nil, err
	}
	g.stateInfo, err = os.Lstat(p.stateRoot)
	if err != nil {
		return nil, errors.New("cargo retained approval root unavailable")
	}
	g.state, err = os.OpenRoot(p.stateRoot)
	if err != nil {
		return nil, errors.New("open Cargo retained approval root")
	}
	opened, err := g.state.Stat(".")
	if err != nil || !os.SameFile(g.stateInfo, opened) {
		return nil, errors.New("cargo retained approval root identity changed")
	}
	projectHash := sha256.Sum256([]byte(project))
	g.stateName = hex.EncodeToString(projectHash[:]) + ".json"
	g.originalState, g.originalStateInfo, err = readGoTransactionControl(g.state, g.stateName)
	if errors.Is(err, os.ErrNotExist) {
		// Unmanaged input is data only. A pre-existing project marker cannot restore
		// missing controller-owned approval, and Begin never authorizes publication.
		if _, markerErr := g.guard.root.Lstat(cargoTransactionMetadata); !errors.Is(markerErr, os.ErrNotExist) {
			return nil, errors.New("cargo marker has no independent retained approval")
		}
	} else if err != nil || !pypiSingleLink(g.originalStateInfo) {
		return nil, errors.New("cargo retained approval is unavailable or aliased")
	} else if err := g.verifyRetainedApproval(ctx); err != nil {
		return nil, err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *approvedCargoProjectGuard) checkRecoveryState() error {
	directory, err := g.guard.root.Open(".")
	if err != nil {
		return errors.New("open Cargo project recovery inventory")
	}
	entries, readErr := directory.ReadDir(10001)
	closeErr := directory.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) > 10000 {
		return errors.New("cargo project recovery inventory is unavailable or exceeds bounds")
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heliopause-cargo-commit-") {
			return errors.New("cargo project has an incomplete transaction requiring recovery")
		}
	}
	if info, err := g.guard.root.Lstat(".heliopause"); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("cargo transaction metadata directory is untrusted")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("cargo transaction metadata directory is unavailable")
	}
	return nil
}

func (g *approvedCargoProjectGuard) verifyRetainedApproval(ctx context.Context) error {
	var doc cargoProjectApproval
	decoder := json.NewDecoder(bytes.NewReader(g.originalState))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&doc) != nil {
		return errors.New("cargo retained approval document is invalid")
	}
	canonical, err := json.Marshal(doc)
	if err != nil || !bytes.Equal(canonical, g.originalState) || doc.Schema != 1 || doc.Approval.Schema != 1 || doc.Approval.Project+".json" != g.stateName || doc.Approval.Policy == "" || doc.Approval.PolicyVersion == 0 || len(doc.Approval.Controls) != 2 {
		return errors.New("cargo retained approval binding is invalid")
	}
	if _, err := domain.NewSHA256Digest(doc.Approval.Graph); err != nil {
		return errors.New("cargo retained graph identity is invalid")
	}
	for _, control := range g.plan.controls {
		matched := false
		for _, recorded := range doc.Approval.Controls {
			if recorded.Path == control.Name() && recorded.SHA256 == control.Digest().String() {
				matched = true
			}
		}
		if !control.Present() || !matched {
			return errors.New("cargo retained approval does not match current controls")
		}
	}
	manifests, err := cargoLocalManifestDigests(g.source.Files())
	if err != nil || !sameCargoManifests(manifests, doc.LocalManifests) {
		return errors.New("cargo retained local manifests changed")
	}
	if err := g.owner.cache.verifyRecordedApproval(ctx, doc.Approval); err != nil {
		return err
	}
	digest, err := domain.NewSHA256Digest(doc.CacheDigest)
	if err != nil {
		return errors.New("cargo retained cache identity is invalid")
	}
	_, err = g.owner.cache.openDocument(ctx, doc.Cache, digest, doc.Approval)
	return err
}

func (g *approvedCargoProjectGuard) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), g.plan.controls...)
}

func (g *approvedCargoProjectGuard) VerifyUnchanged(ctx context.Context) error {
	if g == nil || ctx == nil || g.closed || g.committed {
		return errors.New("cargo project guard is not active")
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
	if err := g.source.Verify(ctx); err != nil {
		return err
	}
	info, err := os.Lstat(g.owner.stateRoot)
	if err != nil || !os.SameFile(g.stateInfo, info) || g.stateInfo.Mode() != info.Mode() {
		return errors.New("cargo retained approval root changed")
	}
	current, stateInfo, err := readGoTransactionControl(g.state, g.stateName)
	if len(g.originalState) == 0 && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !pypiSingleLink(stateInfo) || !os.SameFile(g.originalStateInfo, stateInfo) || g.originalStateInfo.Mode() != stateInfo.Mode() || !bytes.Equal(current, g.originalState) {
		return errors.New("cargo retained approval changed during transaction")
	}
	return nil
}

func sameCargoTransactionControls(left, right []domain.ProjectControlFile) bool {
	return artifactcargo.EqualControls(left, right)
}

func (g *approvedCargoProjectGuard) Commit(ctx context.Context, update domain.ProjectDependencyUpdate, staged domain.StagedProjectSet) (resultErr error) {
	if g == nil || !update.Valid() || !staged.Valid() || update.Snapshot().Context() != g.context || update.Snapshot().Source() != artifactcargo.Source() || !sameCargoTransactionControls(g.plan.controls, update.OriginalControls()) {
		return errors.New("cargo transaction update differs from guarded project")
	}
	return g.commitSelection(ctx, update.Snapshot(), update.SelectedControls(), staged)
}

// CommitSnapshot retains approval for the complete current project without
// inventing a primary artifact or selecting dependencies again.
func (g *approvedCargoProjectGuard) CommitSnapshot(ctx context.Context, snapshot domain.ProjectDependencySnapshot, staged domain.StagedProjectSet) error {
	if g == nil || !snapshot.Valid() || snapshot.Context() != g.context || snapshot.Source() != artifactcargo.Source() {
		return errors.New("cargo snapshot differs from guarded project")
	}
	controls := g.Controls()
	if len(controls) != len(snapshot.ControlDigests()) {
		return errors.New("cargo snapshot control coverage differs from guard")
	}
	for _, file := range snapshot.ControlDigests() {
		matched := false
		for _, control := range controls {
			if control.Present() && control.Name() == file.Name() && control.Digest() == file.Digest() {
				matched = true
			}
		}
		if !matched {
			return errors.New("cargo snapshot controls differ from guard")
		}
	}

	return g.commitSelection(ctx, snapshot, controls, staged)
}

func (g *approvedCargoProjectGuard) commitSelection(ctx context.Context, selected domain.ProjectDependencySnapshot, controls []domain.ProjectControlFile, staged domain.StagedProjectSet) (resultErr error) {
	if !staged.Valid() {
		return errors.New("cargo project requires an approved staged cache")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	snapshot := staged.Set().Inspected().Snapshot()
	if snapshot.Context() != selected.Context() || snapshot.Source() != selected.Source() || snapshot.GraphDigest() != selected.GraphDigest() || len(snapshot.Dependencies()) != len(selected.Dependencies()) {
		return errors.New("cargo transaction cache approval differs from selected graph")
	}
	selectedDependencies := selected.Dependencies()
	for i, artifact := range snapshot.Dependencies() {
		if artifact != selectedDependencies[i] {
			return errors.New("cargo transaction cache dependency differs from selected graph")
		}
	}
	if _, err := g.owner.cache.OpenProjectCache(ctx, staged); err != nil {
		return err
	}
	doc, err := g.owner.cache.approval(ctx, staged.Set())
	if err != nil {
		return err
	}
	if len(doc.Controls) != len(controls) {
		return errors.New("cargo transaction approval controls are incomplete")
	}
	selectedBodies := map[string][]byte{}
	for _, control := range controls {
		matched := false
		for _, recorded := range doc.Controls {
			if recorded.Path == control.Name() && recorded.SHA256 == control.Digest().String() {
				matched = true
			}
		}
		if !matched || !control.Present() || (control.Name() != "Cargo.toml" && control.Name() != "Cargo.lock") {
			return errors.New("cargo approval controls differ from selection")
		}
		selectedBodies[control.Name()] = control.Body()
	}
	if len(selectedBodies) != 2 || artifactcargo.ValidateProjectManifest(selectedBodies["Cargo.toml"], "Cargo.toml") != nil || artifactcargo.ValidateProjectLock(selectedBodies["Cargo.lock"]) != nil {
		return errors.New("cargo selected controls are incomplete or invalid")
	}
	selectedFiles := g.source.Files()
	for name, body := range selectedBodies {
		selectedFiles[name] = body
	}
	localManifests, err := cargoLocalManifestDigests(selectedFiles)
	if err != nil {
		return err
	}
	approval, err := json.Marshal(cargoProjectApproval{Schema: 1, Cache: staged.ContentHandle(), CacheDigest: staged.Digest().String(), Approval: doc, LocalManifests: localManifests})
	if err != nil || len(approval) > artifactcargo.MaxProjectControlBytes {
		return errors.New("cargo transaction approval exceeds bounds")
	}
	workspace, err := g.plan.privateWorkspace()
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(workspace); err != nil {
			resultErr = errors.Join(resultErr, errors.New("dispose Cargo transaction workspace"))
		}
	}()
	for _, control := range controls {
		if !control.Present() || os.WriteFile(filepath.Join(workspace, control.Name()), control.Body(), 0o600) != nil {
			return errors.New("materialize selected Cargo transaction controls")
		}
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	plan := g.plan
	plan.authorized = true // Complete typed ALLOW/cache/Evidence were revalidated above.
	transaction, err := beginCargoProjectTransaction(plan, workspace, controls...)
	if err != nil {
		return err
	}
	transaction.checkpoint = func(phase string) error {
		if g.owner.checkpoint != nil {
			if err := g.owner.checkpoint(phase); err != nil {
				return err
			}
		}
		if phase == "before-backup" {
			return g.VerifyUnchanged(ctx)
		}
		if phase == "before-approval" {
			return g.source.VerifyPublishedControls(ctx, selectedBodies)
		}
		return ctx.Err()
	}
	published := false
	var publishedInfo os.FileInfo
	transaction.publishApproval = func() error {
		current, info, err := readGoTransactionControl(g.state, g.stateName)
		if g.originalStateInfo == nil {
			if !errors.Is(err, os.ErrNotExist) {
				return errors.New("cargo retained approval appeared before publication")
			}
		} else if err != nil || !os.SameFile(g.originalStateInfo, info) || !bytes.Equal(current, g.originalState) {
			return errors.New("cargo retained approval changed before publication")
		}
		if _, err := g.owner.cache.OpenProjectCache(ctx, staged); err != nil {
			return err
		}
		err = g.writeState(approval, &published)
		if published {
			publishedInfo, _ = g.state.Lstat(g.stateName)
		}
		return err
	}
	transaction.rollbackApproval = func() error {
		if !published {
			return nil
		}
		current, info, err := readGoTransactionControl(g.state, g.stateName)
		if err != nil || publishedInfo == nil || !os.SameFile(publishedInfo, info) || !pypiSingleLink(info) || !bytes.Equal(current, approval) {
			return errors.New("cargo approval rollback identity is uncertain")
		}
		if len(g.originalState) == 0 {
			if err := g.state.Remove(g.stateName); err != nil {
				return errors.New("remove uncommitted Cargo approval")
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

func (g *approvedCargoProjectGuard) writeState(body []byte, published *bool) (resultErr error) {
	info, err := os.Lstat(g.owner.stateRoot)
	if err != nil || !os.SameFile(g.stateInfo, info) || g.stateInfo.Mode() != info.Mode() {
		return errors.New("cargo retained approval root changed before publication")
	}
	id, err := domain.NewRunID()
	if err != nil {
		return err
	}
	name := ".cargo-approval-" + id.String()
	f, err := g.state.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create private Cargo approval record")
	}
	defer func() {
		if err := g.state.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, errors.New("dispose private Cargo approval record"))
		}
	}()
	_, writeErr := f.Write(body)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("persist private Cargo approval record")
	}
	if len(g.originalState) == 0 {
		if err := g.state.Link(name, g.stateName); err != nil {
			return errors.New("publish initial trusted Cargo project approval")
		}
	} else if err := g.state.Rename(name, g.stateName); err != nil {
		return errors.New("publish trusted Cargo project approval")
	}
	*published = true
	return syncDirectory(g.owner.stateRoot)
}

func (g *approvedCargoProjectGuard) Close() error {
	if g == nil || g.closed {
		return errors.New("cargo project guard already closed or unavailable")
	}
	g.closed = true
	var result error
	if g.source != nil {
		result = g.source.Close()
	}
	if g.state != nil {
		result = errors.Join(result, g.state.Close())
	}
	return errors.Join(result, g.guard.release())
}

func cargoLocalManifestDigests(files map[string][]byte) (map[string]string, error) {
	names, err := artifactcargo.SelectedLocalManifests(files)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, name := range names {
		hash := sha256.Sum256(files[name])
		result[name] = hex.EncodeToString(hash[:])
	}
	return result, nil
}
func sameCargoManifests(left, right map[string]string) bool {
	if len(left) == 0 || len(left) != len(right) {
		return false
	}
	for name, digest := range left {
		if right[name] != digest {
			return false
		}
	}
	return true
}
