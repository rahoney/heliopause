package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type PNPMProjectPromotion struct {
	intakeRoot, evidenceRoot, cacheRoot, stateRoot string
	evidence                                       ProjectEvidenceReader
	operation                                      domain.OperationID
	selectedCache                                  func(domain.ProjectDependencySnapshot) (*PNPMVerifiedCache, error)
	checkpoint                                     func(string) error
	published                                      *approvedPNPMProjectGuard
}

func NewPNPMProjectPromotion(intake, evidence, cache, state string, reader ProjectEvidenceReader, operation domain.OperationID, selectedCache func(domain.ProjectDependencySnapshot) (*PNPMVerifiedCache, error)) (*PNPMProjectPromotion, error) {
	roots := []string{intake, evidence, cache, state}
	if reader == nil || operation.String() == "" || selectedCache == nil || !separateRoots(roots) {
		return nil, errors.New("pnpm mutation requires independent trusted roots, actual operation and selected cache")
	}
	for _, root := range roots {
		if root == "/" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return nil, errors.New("pnpm mutation roots must be canonical")
		}
	}
	return &PNPMProjectPromotion{intakeRoot: intake, evidenceRoot: evidence, cacheRoot: cache, stateRoot: state, evidence: reader, operation: operation, selectedCache: selectedCache}, nil
}

// Successful publication records are controller-bound opaque cache references.
// A target marker or a previous request cannot authorize a new completion.
func (p *PNPMProjectPromotion) PublishedRecords() (domain.ContentDigest, string, string, bool) {
	if p == nil || p.published == nil || !p.published.committed || p.published.recordManifest.String() == "" || p.published.recordHandle == "" {
		return domain.ContentDigest{}, "", "", false
	}
	g := p.published
	return g.recordManifest, g.recordHandle + ":manifest.json", g.recordHandle + ":sbom.cdx.json", true
}

type pnpmMemberApproval struct{ Name, SHA256 string }
type pnpmProjectApproval struct {
	Schema             int
	Cache, CacheDigest string
	Approval           pnpmCacheDocument
	Members            []pnpmMemberApproval
}

type approvedPNPMProjectGuard struct {
	owner                       *PNPMProjectPromotion
	ctx                         context.Context
	install                     domain.InstallContext
	guard                       goProjectGuard
	controls                    []domain.ProjectControlFile
	controlInfo                 map[string]os.FileInfo
	members                     map[string]*npmTransactionMember
	state                       *os.Root
	stateInfo                   os.FileInfo
	stateName                   string
	originalState, currentState []byte
	currentStateInfo            os.FileInfo
	previous                    *pnpmProjectApproval
	closed, committed           bool
	recordManifest              domain.ContentDigest
	recordHandle                string
}

func (p *PNPMProjectPromotion) Begin(ctx context.Context, install domain.InstallContext) (*approvedPNPMProjectGuard, error) {
	return p.begin(ctx, install, false)
}
func (p *PNPMProjectPromotion) BeginUpgrade(ctx context.Context, install domain.InstallContext, adopt bool) (*approvedPNPMProjectGuard, error) {
	return p.begin(ctx, install, adopt)
}

func (p *PNPMProjectPromotion) begin(ctx context.Context, install domain.InstallContext, adopt bool) (result *approvedPNPMProjectGuard, resultErr error) {
	if p != nil {
		p.published = nil
	}
	if ctx == nil || p == nil || !install.Valid() || install.Mode() != domain.InstallNPMProject || !separateRoots([]string{install.Target().String(), p.intakeRoot, p.evidenceRoot, p.cacheRoot, p.stateRoot}) {
		return nil, errors.New("pnpm mutation requires an existing bounded project")
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	guard, err := acquireProjectTransactionGuard(install.Target().String(), ".heliopause-npm-transaction.lock")
	if err != nil {
		return nil, err
	}
	g := &approvedPNPMProjectGuard{owner: p, ctx: ctx, install: install, guard: guard, controlInfo: map[string]os.FileInfo{}, members: map[string]*npmTransactionMember{}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, g.Close())
			result = nil
		}
	}()
	if err := g.rejectRecovery(); err != nil {
		return nil, err
	}
	for _, name := range []string{"package.json", "pnpm-lock.yaml"} {
		body, info, err := readGoTransactionControl(guard.root, name)
		present := true
		if name == "pnpm-lock.yaml" && errors.Is(err, os.ErrNotExist) {
			body, info, err, present = nil, nil, nil, false
		}
		if err != nil || present && (info == nil || !pypiSingleLink(info) || info.Mode()&^os.ModePerm != 0) {
			return nil, errors.Join(errors.New("pnpm original control is unavailable or aliased"), err)
		}
		control, err := domain.NewProjectControlFile(name, body, present)
		if err != nil {
			return nil, err
		}
		g.controls = append(g.controls, control)
		g.controlInfo[name] = info
		if present {
			snapshot, err := snapshotBoundedPromotionFile(filepath.Join(install.Target().String(), name), artifactnpm.ProjectControlLimit)
			if err != nil {
				return nil, err
			}
			if snapshot.Digest != control.Digest().String() || !os.SameFile(info, snapshot.info) || info.Mode() != snapshot.Mode {
				return nil, errors.New("pnpm control changed while freezing original bytes")
			}
			g.members[name] = &npmTransactionMember{files: map[string]pypiFileSnapshot{".": snapshot}, directories: map[string]os.FileInfo{}}
		} else {
			g.members[name] = nil
		}
	}
	if !json.Valid(g.controls[0].Body()) || len(g.controls[0].Body()) == 0 {
		return nil, errors.New("pnpm original manifest is invalid")
	}
	if err := ensureTrustedRoot(p.stateRoot); err != nil {
		return nil, err
	}
	g.stateInfo, err = os.Lstat(p.stateRoot)
	if err != nil {
		return nil, err
	}
	g.state, err = os.OpenRoot(p.stateRoot)
	if err != nil {
		return nil, err
	}
	g.stateName = pnpmCacheHash([]byte(install.Target().String())) + ".json"
	g.originalState, g.currentStateInfo, err = readGoTransactionControl(g.state, g.stateName)
	if errors.Is(err, os.ErrNotExist) {
		g.originalState = nil
		g.currentStateInfo = nil
	} else if err != nil {
		return nil, err
	} else if !pypiSingleLink(g.currentStateInfo) || g.currentStateInfo.Mode() != 0o600 {
		return nil, errors.New("pnpm independent approval is aliased or has unsafe mode")
	}
	g.currentState = bytes.Clone(g.originalState)
	for _, name := range []string{"node_modules", "haa"} {
		location := g.memberPath(name)
		_, err := g.guard.root.Lstat(pnpmDiskMember(name))
		if errors.Is(err, os.ErrNotExist) {
			g.members[name] = nil
			continue
		}
		if err != nil {
			return nil, err
		}
		if len(g.originalState) == 0 {
			return nil, errors.New("pnpm existing payload requires independently proven native adoption")
		}
		if name == "node_modules" {
			native, err := capturePNPMOwnedMember(ctx, location)
			if err != nil {
				return nil, err
			}
			native.boundary = g.verifyPinned
			g.members[name] = &npmTransactionMember{pnpm: native}
		} else {
			member, err := snapshotNPMTransactionMember(location)
			if err != nil {
				return nil, err
			}
			g.members[name] = member
		}
	}
	if len(g.originalState) != 0 {
		if err := g.verifyPrevious(ctx, adopt); err != nil {
			return nil, err
		}
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return nil, err
	}
	return g, nil
}

func pnpmDiskMember(name string) string {
	if name == "haa" {
		return ".heliopause"
	}
	return name
}
func (g *approvedPNPMProjectGuard) memberPath(name string) string {
	return filepath.Join(g.install.Target().String(), pnpmDiskMember(name))
}

func (g *approvedPNPMProjectGuard) rejectRecovery() error {
	f, err := g.guard.root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := f.ReadDir(10001)
	closeErr := f.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) > 10000 {
		return errors.Join(errors.New("pnpm recovery inventory is unavailable or exceeds its bound"), readErr, closeErr)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heliopause-npm-commit-") {
			return errors.New("pnpm project has an interrupted shared transaction requiring recovery")
		}
	}
	return nil
}

func (g *approvedPNPMProjectGuard) verifyPinned() error {
	if g == nil || g.closed {
		return errors.New("pnpm project guard is not active")
	}
	if err := g.guard.verify(); err != nil {
		return err
	}
	if g.state == nil {
		return nil
	}
	info, err := os.Lstat(g.owner.stateRoot)
	if err != nil || !os.SameFile(g.stateInfo, info) || g.stateInfo.Mode() != info.Mode() || !pnpmSameOwner(g.stateInfo, info) {
		return errors.Join(errors.New("pnpm independent approval root changed"), err)
	}
	body, current, err := readGoTransactionControl(g.state, g.stateName)
	if g.currentStateInfo == nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || g.currentStateInfo == nil || !os.SameFile(g.currentStateInfo, current) || current.Mode() != 0o600 || !pypiSingleLink(current) || !bytes.Equal(body, g.currentState) {
		return errors.Join(errors.New("pnpm independent approval changed during mutation"), err)
	}
	return nil
}

func (g *approvedPNPMProjectGuard) verifyPrevious(ctx context.Context, adopt bool) error {
	var doc pnpmProjectApproval
	decoder := json.NewDecoder(bytes.NewReader(g.originalState))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return err
	}
	canonical, err := json.Marshal(doc)
	if err != nil || !bytes.Equal(canonical, g.originalState) || doc.Schema != 1 || doc.Approval.Project+".json" != g.stateName || len(doc.Members) != 4 {
		return errors.New("pnpm retained target approval is invalid")
	}
	if err := verifyPNPMRecordedApproval(ctx, g.owner.evidence, doc.Approval); err != nil {
		return err
	}
	digest, err := domain.NewSHA256Digest(doc.CacheDigest)
	if err != nil {
		return err
	}
	if _, err := openPNPMApprovedCache(ctx, g.owner.cacheRoot, doc.Cache, digest, doc.Approval, nil); err != nil {
		return err
	}
	for index, name := range []string{"package.json", "pnpm-lock.yaml", "node_modules", "haa"} {
		if doc.Members[index].Name != name {
			return errors.New("pnpm retained member coverage differs")
		}
		if _, err := domain.NewSHA256Digest(doc.Members[index].SHA256); err != nil {
			return err
		}
		if adopt && (name == "package.json" || name == "pnpm-lock.yaml") {
			continue
		}
		actual, err := npmPNPMMemberDigest(g.members[name])
		if err != nil || actual != doc.Members[index].SHA256 {
			return errors.Join(errors.New("pnpm original owned target differs from independent approval"), err)
		}
	}
	g.previous = &doc
	return nil
}

func (g *approvedPNPMProjectGuard) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), g.controls...)
}
func (g *approvedPNPMProjectGuard) VerifyUnchanged(ctx context.Context) error {
	if ctx == nil || g == nil || g.closed || g.committed {
		return errors.New("pnpm project mutation is not active")
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	if err := g.verifyPinned(); err != nil {
		return err
	}
	for name, before := range g.members {
		if before == nil {
			if _, err := g.guard.root.Lstat(pnpmDiskMember(name)); !errors.Is(err, os.ErrNotExist) {
				return errors.New("pnpm original absent member appeared")
			}
		} else if err := before.verify(g.memberPath(name)); err != nil {
			return err
		}
	}
	return nil
}

func (g *approvedPNPMProjectGuard) Close() (resultErr error) {
	if g == nil || g.closed {
		return errors.New("pnpm project guard is already closed")
	}
	g.closed = true
	if g.state != nil {
		resultErr = g.state.Close()
	}
	return errors.Join(resultErr, g.guard.release())
}
