package promotion

import (
	"context"
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// UVProjectPreparation holds original project files and the existing venv
// through one complete selection. Native output never supplies ownership.
type UVProjectPreparation struct {
	promoter     *PyPIPromotion
	resolver     ports.ProjectLockUpdateResolver
	root         string
	intent       artifactpypi.UVProjectIntent
	selector     string
	adopt        bool
	native       *pypiNativeEnrollment
	trustedRoots []string
	started      bool
	active       *uvProjectSession
}

func NewUVProjectPreparation(promoter *PyPIPromotion, resolver ports.ProjectLockUpdateResolver, root string, intent artifactpypi.UVProjectIntent, selector string, adopt bool, trustedRoots []string) (*UVProjectPreparation, error) {
	if promoter == nil || resolver == nil || len(trustedRoots) == 0 || len(trustedRoots) > 32 || (intent != artifactpypi.UVProjectAdd && intent != artifactpypi.UVProjectSync) || (intent == artifactpypi.UVProjectSync && selector != "") {
		return nil, errors.New("UV project installation requires guarded add/sync, storage and native selection")
	}
	found := false
	for _, r := range trustedRoots {
		found = found || r == promoter.stagingRoot
	}
	if !found {
		return nil, errors.New("UV project guard omits actual staging")
	}
	return &UVProjectPreparation{promoter: promoter, resolver: resolver, root: root, intent: intent, selector: selector, adopt: adopt, trustedRoots: append([]string(nil), trustedRoots...)}, nil
}

func (p *UVProjectPreparation) WithNativeEnrollment(resolver ports.DependencyResolver, intake ports.Artifact, intakeRoot string) (*UVProjectPreparation, error) {
	if p == nil || p.started || resolver == nil || intake == nil {
		return nil, errors.New("UV project original enrollment is unavailable")
	}
	found := false
	for _, root := range p.trustedRoots {
		found = found || root == intakeRoot
	}
	if !found {
		return nil, errors.New("UV project original intake is outside frozen storage boundaries")
	}
	copy := *p
	copy.native = &pypiNativeEnrollment{resolver: resolver, intake: intake, intakeRoot: intakeRoot, runner: p.promoter.runner, client: nativeUVClient}
	return &copy, nil
}

type uvProjectSession struct {
	install                      domain.InstallContext
	transaction                  *pypiVenvTransaction
	original                     pypiInstalledSnapshot
	controls                     []domain.ProjectControlFile
	baseControls                 []domain.ProjectControlFile
	proof                        domain.ProjectControlFile
	baseSelected                 domain.ProjectDependencyUpdate
	selected                     domain.ProjectDependencyUpdate
	attempted, committed, closed bool
	closeErr                     error
}

func (p *UVProjectPreparation) Begin(ctx context.Context, install domain.InstallContext) (result *uvProjectSession, resultErr error) {
	if p == nil || p.started || ctx == nil || !install.Valid() || install.Mode() != domain.InstallPythonVenv || p.promoter.goos != "linux" || p.promoter.goarch != "amd64" || strings.Contains(install.Target().String(), ",") {
		return nil, errors.New("UV project requires one existing Linux amd64 venv transaction")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.started = true
	plan, err := discoverUVPythonVenv(install.Target().String())
	if err != nil {
		return nil, err
	}
	tx, err := beginPyPIVenvTransactionWithDiscovery(plan, discoverUVPythonVenv)
	if err != nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	s := &uvProjectSession{install: install, transaction: tx}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, s.Close())
			result = nil
		}
	}()
	input, err := freezeUVProjectFiles(ctx, tx, p.root, p.intent, p.selector, p.trustedRoots)
	if err != nil {
		return nil, err
	}
	s.original, err = capturePyPIInstalledSnapshot(tx)
	if err != nil {
		return nil, err
	}
	s.proof, err = prepareUVCompleteOwnership(ctx, tx, s.original, install, p.native, p.adopt, s.VerifyUnchanged)
	if err != nil {
		return nil, err
	}
	s.baseControls, err = sandbox.NewUVProjectOriginalControls(input)
	if err != nil {
		return nil, err
	}
	s.controls = append([]domain.ProjectControlFile(nil), s.baseControls...)
	if s.proof.Present() {
		s.controls = append(s.controls, s.proof)
	}
	if err := s.VerifyUnchanged(ctx); err != nil {
		return nil, err
	}
	p.active = s
	return s, nil
}

func (s *uvProjectSession) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}
func (s *uvProjectSession) VerifyUnchanged(ctx context.Context) error {
	if s == nil || ctx == nil || s.closed || s.committed {
		return errors.New("UV project original guard is unavailable")
	}
	return s.original.verify(ctx, s.transaction)
}
func (s *uvProjectSession) Close() error {
	if s == nil {
		return nil
	}
	if !s.closed {
		s.closed = true
		s.closeErr = s.transaction.close()
	}
	return s.closeErr
}

func (p *UVProjectPreparation) ResolveProjectLockUpdate(ctx context.Context, install domain.InstallContext, original []domain.ProjectControlFile, selector string) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.active == nil {
		return domain.ProjectDependencyUpdate{}, errors.New("UV project selection requires its held original guard")
	}
	s := p.active
	if s.attempted || selector != "" || install != s.install || !sameUVSyncControls(original, s.controls) {
		return domain.ProjectDependencyUpdate{}, errors.New("UV project selection changed or was repeated")
	}
	if err := s.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	s.attempted = true
	update, err := p.resolver.ResolveProjectLockUpdate(ctx, install, append([]domain.ProjectControlFile(nil), s.baseControls...), "")
	if err = errors.Join(err, s.VerifyUnchanged(ctx)); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	if update.Snapshot().Context() != install || !sameUVSyncControls(update.OriginalControls(), s.baseControls) {
		return domain.ProjectDependencyUpdate{}, errors.New("UV project resolver changed original target or controls")
	}
	if err := sandbox.ValidateUVProjectUpdate(update); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	s.baseSelected = update
	if s.proof.Present() {
		controls := append(update.SelectedControls(), s.proof)
		snapshot, err := bindPythonOriginalProof(update.Snapshot(), controls, "uv-project-original-proof/v1")
		if err != nil {
			return domain.ProjectDependencyUpdate{}, err
		}
		update, err = domain.NewProjectDependencyUpdateFromSnapshot(s.controls, controls, snapshot)
		if err != nil {
			return domain.ProjectDependencyUpdate{}, err
		}
	}
	s.selected = update
	return update, nil
}

func (p *UVProjectPreparation) SelectedUpdate(ctx context.Context) (domain.ProjectDependencyUpdate, error) {
	if p == nil || p.active == nil || !p.active.selected.Valid() {
		return domain.ProjectDependencyUpdate{}, errors.New("UV project has no frozen complete selection")
	}
	if err := p.active.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencyUpdate{}, err
	}
	return p.active.selected, nil
}
