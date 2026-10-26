package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvToolNativeSelector interface {
	ResolveToolInstall(context.Context, artifactpypi.UpgradeRequirement) (sandbox.UVToolSelection, error)
}

// UVToolInstallPreparation uses ordinary current whole-set INSTALL approval.
// Native selection remains distinct from UV pip and opens no existing tool.
type UVToolInstallPreparation struct {
	promoter     *PyPIPromotion
	resolver     uvToolNativeSelector
	upgrade      uvToolUpgradeNativeSelector
	binRoot      string
	trustedRoots []string
}

func NewUVToolInstallPreparation(promoter *PyPIPromotion, resolver uvToolNativeSelector, binRoot string, trustedRoots []string) (*UVToolInstallPreparation, error) {
	if promoter == nil || promoter.runner == nil || resolver == nil || promoter.goos != "linux" || promoter.goarch != "amd64" || len(trustedRoots) == 0 {
		return nil, errors.New("UV tool installation requires supported existing promotion, native selection and actual storage")
	}
	stagingBound := false
	for _, root := range trustedRoots {
		if root == promoter.stagingRoot {
			stagingBound = true
		}
	}
	if !stagingBound {
		return nil, errors.New("UV tool guard omits actual trusted staging")
	}
	return &UVToolInstallPreparation{promoter: promoter, resolver: resolver, binRoot: binRoot, trustedRoots: append([]string(nil), trustedRoots...)}, nil
}

type uvToolInstallSession struct {
	owner                                         *UVToolInstallPreparation
	guard                                         *toolFreshTarget
	reference                                     domain.ArtifactReference
	install                                       domain.InstallContext
	resolution                                    domain.DependencyResolution
	receipt                                       domain.ProjectControlFile
	originalWheels                                []artifactpypi.UVInstalledWheelControls
	attempted, resolved, promoteAttempted, closed bool
	closeErr                                      error
}

func (p *UVToolInstallPreparation) BeginInstall(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (ports.InstallSession, error) {
	if p == nil || ctx == nil || reference.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return nil, errors.New("UV tool installation requires one public request")
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(reference.Locator())
	if err != nil || requirement.Requirement() != reference.Locator() {
		return nil, errors.New("UV tool request is not canonical")
	}
	guard, err := beginToolFreshTarget(ctx, install, requirement, p.binRoot, p.trustedRoots)
	if err != nil {
		return nil, err
	}
	return &uvToolInstallSession{owner: p, guard: guard, reference: reference, install: install}, nil
}

func (s *uvToolInstallSession) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	if s == nil || s.closed || s.attempted || s.promoteAttempted || reference != s.reference || install != s.install {
		return domain.DependencyResolution{}, errors.New("UV tool selection is unavailable, repeated or changed")
	}
	if err := s.guard.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	s.attempted = true
	var selected sandbox.UVToolSelection
	var err error
	if s.guard.original != nil {
		if s.owner.upgrade == nil {
			return domain.DependencyResolution{}, errors.New("UV tool original selector is unavailable")
		}
		selected, err = s.owner.upgrade.ResolveToolUpgrade(ctx, s.guard.original.input, s.originalWheels)
	} else {
		selected, err = s.owner.resolver.ResolveToolInstall(ctx, s.guard.requirement)
	}
	if err = errors.Join(err, s.guard.verify(ctx)); err != nil {
		return domain.DependencyResolution{}, err
	}
	resolution, controls := selected.Resolution(), selected.Controls()
	if err := s.validateSelection(resolution, controls); err != nil {
		return domain.DependencyResolution{}, err
	}
	var native artifactpypi.ToolReceipt
	for _, control := range controls {
		if control.Name() == "uv-tool-receipt.toml" {
			native, err = artifactpypi.ParseUVToolReceipt(control.Body())
			if err != nil {
				return domain.DependencyResolution{}, err
			}
		}
	}
	projected, err := native.WithUVLayout(install.Target().String(), s.owner.binRoot)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if err := s.guard.freezeCommands(ctx, projected); err != nil {
		return domain.DependencyResolution{}, err
	}
	receipt, err := domain.NewProjectControlFile(uvToolHostReceiptControl, projected.Body(), true)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	target, err := uvToolTargetControl(s.guard, resolution.LockfileDigest())
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	controls = append(controls, receipt, target)
	resolution, err = resolution.WithProjectControls(controls)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if err := s.guard.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	s.resolution, s.receipt, s.resolved = resolution, receipt, true
	return resolution, nil
}

func validateUVToolSelection(resolution domain.DependencyResolution, controls []domain.ProjectControlFile, requirement artifactpypi.UpgradeRequirement) error {
	return validateUVToolNativeSelection(resolution, controls, requirement, "uv-tool-install:"+sandbox.UVResolverRuntimeIdentity(), []string{"uv-tool-intent.json", "uv-tool-receipt.toml", "uv-tool-pylock.toml", "uv-tool-census.json"})
}

func validateUVToolNativeSelection(resolution domain.DependencyResolution, controls []domain.ProjectControlFile, requirement artifactpypi.UpgradeRequirement, runtime string, names []string) error {
	graph := resolution.Graph()
	if resolution.RuntimeIdentity() != runtime || resolution.LockfileDigest().String() == "" || len(resolution.ProjectControls()) != 0 || len(graph.Nodes()) == 0 || len(graph.Derivations()) != 0 || len(controls) != len(names) {
		return errors.New("UV tool selection lacks its exact native wheel-only input")
	}
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[name] = true
	}
	for _, control := range controls {
		if !wanted[control.Name()] || !control.Present() || len(control.Body()) == 0 {
			return errors.New("UV tool native controls are missing or changed")
		}
		delete(wanted, control.Name())
	}
	if len(wanted) != 0 {
		return errors.New("UV tool native controls are incomplete")
	}
	primary := false
	for _, node := range graph.Nodes() {
		identity := node.Artifact().Identity()
		if identity.Source() != artifactpypi.PublicPyPIProfile().Source() || identity.Variant() != "wheel" || node.HostInstallAction() {
			return errors.New("UV tool selection has an unsupported source or install action")
		}
		if node.Node() == graph.Primary() {
			if identity.Name() != requirement.Project() || !requirement.AcceptsVersion(identity.Version()) {
				return errors.New("UV tool native primary differs from requested constraint")
			}
			primary = true
		}
	}
	if !primary {
		return errors.New("UV tool selection has no matching primary")
	}
	return nil
}

func uvToolTargetControl(g *toolFreshTarget, nativeLock domain.ContentDigest) (domain.ProjectControlFile, error) {
	if g == nil || g.tx == nil || g.bin.root == nil || nativeLock.String() == "" {
		return domain.ProjectControlFile{}, errors.New("tool target control lacks held roots or native selection")
	}
	type root struct {
		Path          string
		Device, Inode uint64
		Mode          uint32
	}
	type interpreter struct {
		Path, Link, Digest string
		Device, Inode      uint64
		Mode               uint32
		Size               int64
	}
	rootInfo := g.tx.directories[g.tx.plan.root]
	device, inode, ok := pypiObjectIdentity(rootInfo)
	if !ok {
		return domain.ProjectControlFile{}, errors.New("tool target root identity unavailable")
	}
	bd, bi, ok := pypiObjectIdentity(g.bin.rootInfo)
	if !ok {
		return domain.ProjectControlFile{}, errors.New("tool exposed root identity unavailable")
	}
	var members []interpreter
	for _, member := range g.interpreter {
		d, i, ok := pypiObjectIdentity(member.info)
		if !ok {
			return domain.ProjectControlFile{}, errors.New("tool interpreter identity unavailable")
		}
		digest := ""
		if member.file != nil {
			digest = member.file.Digest
		}
		members = append(members, interpreter{member.path, member.link, digest, d, i, uint32(member.info.Mode()), member.info.Size()})
	}
	intent := "fresh-install"
	if g.original != nil {
		intent = "managed-upgrade"
	}
	body, err := json.Marshal(struct {
		Version                                                    int
		Intent, Requirement, NativeLock, ConfigSHA256, Interpreter string
		Target, Exposed                                            root
		InterpreterMembers                                         []interpreter
	}{1, intent, g.requirement.Requirement(), nativeLock.String(), g.tx.config.Digest, filepath.Join(g.tx.plan.root, "bin", "python"), root{g.tx.plan.root, device, inode, uint32(rootInfo.Mode())}, root{g.binRoot, bd, bi, uint32(g.bin.rootInfo.Mode())}, members})
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	return domain.NewProjectControlFile("uv-tool-target.json", body, true)
}

func (s *uvToolInstallSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	if s == nil || s.closed || !s.resolved || s.promoteAttempted || install != s.install || !bundle.Valid() || len(bundle.Set().Inspected().Graph().Derivations()) != 0 || !pypiUpgradeGraphExtends(s.resolution.Graph(), bundle.Set().Inspected()) {
		return domain.PromotedInstall{}, errors.New("UV tool publication differs from frozen native selection")
	}
	if err := s.guard.verify(ctx); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := verifyPythonSelectionManifest(bundle, install, s.resolution, "UV tool install"); err != nil {
		return domain.PromotedInstall{}, err
	}
	s.promoteAttempted = true
	return s.owner.promoter.promoteToolSelection(ctx, staged, bundle, install, s.guard, s.receipt)
}

func (s *uvToolInstallSession) Close() error {
	if s == nil {
		return errors.New("UV tool install session unavailable")
	}
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	s.closeErr = s.guard.Close()
	return s.closeErr
}
