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

// UVInstallPreparation retains ordinary INSTALL intent for fresh and installed-
// aware native selection under the same original-state guard.
type UVInstallPreparation struct {
	promoter *PyPIPromotion
	resolver ports.DependencyResolver
}

func NewUVInstallPreparation(promoter *PyPIPromotion, resolver ports.DependencyResolver) (*UVInstallPreparation, error) {
	if promoter == nil || resolver == nil {
		return nil, errors.New("UV installation requires existing Promotion and native resolution")
	}
	return &UVInstallPreparation{promoter, resolver}, nil
}

type uvInstallSession struct {
	owner                                         *UVInstallPreparation
	reference                                     domain.ArtifactReference
	install                                       domain.InstallContext
	transaction                                   *pypiVenvTransaction
	resolution                                    domain.DependencyResolution
	attempted, resolved, promoteAttempted, closed bool
	closeErr                                      error
}

func (p *UVInstallPreparation) BeginInstall(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (result ports.InstallSession, resultErr error) {
	if p == nil || ctx == nil || reference.Source() != artifactpypi.PublicPyPIProfile().Source() || !install.Valid() || install.Mode() != domain.InstallPythonVenv || p.promoter.goos != "linux" || p.promoter.goarch != "amd64" {
		return nil, errors.New("UV installation requires a public request and supported Linux amd64 venv")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parsed, err := artifactpypi.ParseReference(reference.Locator())
	if err != nil || parsed != reference || strings.Contains(install.Target().String(), ",") {
		return nil, errors.New("UV installation request or target is unsupported")
	}
	if !separateRoots([]string{install.Target().String(), p.promoter.stagingRoot}) {
		return nil, errors.New("UV virtual environment overlaps trusted staging")
	}
	plan, err := discoverUVPythonVenv(install.Target().String())
	if err != nil {
		return nil, err
	}
	transaction, err := beginPyPIVenvTransactionWithDiscovery(plan, discoverUVPythonVenv)
	if err != nil {
		return nil, err
	}
	session := &uvInstallSession{owner: p, reference: reference, install: install, transaction: transaction}
	transferred := false
	defer func() {
		if resultErr != nil && !transferred {
			resultErr = errors.Join(resultErr, session.Close())
			result = nil
		}
	}()
	directories, err := pypiUpgradeMetadataDirectories(plan.site)
	if err != nil {
		return nil, err
	}
	if len(directories) != 0 || len(transaction.current.Files) != 0 {
		resolver, ok := p.resolver.(ports.DependencyUpdateResolver)
		if !ok {
			return nil, errors.New("UV installed state requires native installed-aware selection")
		}
		preparation, err := NewUVUpdatePreparation(p.promoter, resolver, artifactpypi.UVInstallSelection)
		if err != nil {
			return nil, err
		}
		requirement, err := artifactpypi.ParseUpgradeRequirement(reference.Locator())
		if err != nil {
			return nil, err
		}
		transferred = true
		guarded, err := preparation.shared.prepareInstallUpdate(ctx, requirement, install, false, transaction)
		if err != nil {
			return nil, err
		}
		return &uvInstalledInstallSession{session: guarded, reference: reference, install: install}, nil
	}
	if err := session.verify(ctx); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *uvInstallSession) verify(ctx context.Context) error {
	if s.promoteAttempted {
		return errors.New("UV install publication was already attempted")
	}
	return s.verifyOriginal(ctx)
}

func (s *uvInstallSession) verifyOriginal(ctx context.Context) error {
	if s.closed || ctx == nil {
		return errors.New("UV install original-state guard is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.transaction.checkBefore(); err != nil {
		return err
	}
	plan, err := discoverUVPythonVenv(s.install.Target().String())
	if err != nil || plan != s.transaction.plan {
		return errors.New("UV virtual environment configuration or scheme changed")
	}
	directories, err := pypiUpgradeMetadataDirectories(plan.site)
	if err != nil || len(directories) != 0 || len(s.transaction.current.Files) != 0 {
		return errors.New("UV installed distributions require installed-aware guarded selection")
	}
	return s.transaction.checkBefore()
}

func (s *uvInstallSession) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	if s.attempted || reference != s.reference || install != s.install {
		return domain.DependencyResolution{}, errors.New("UV install selection changed or was repeated")
	}
	if err := s.verify(ctx); err != nil {
		return domain.DependencyResolution{}, err
	}
	s.attempted = true
	resolution, err := s.owner.resolver.ResolveDependencies(ctx, reference, install)
	if err = errors.Join(err, s.verify(ctx)); err != nil {
		return domain.DependencyResolution{}, err
	}
	root, err := artifactpypi.RequestedProject(reference)
	version, exact, versionErr := artifactpypi.RequestedVersion(reference)
	if err != nil || versionErr != nil || resolution.RuntimeIdentity() != sandbox.UVResolverRuntimeIdentity() || resolution.LockfileDigest().String() == "" || len(resolution.Graph().Nodes()) == 0 || len(resolution.Graph().Derivations()) != 0 || len(resolution.ProjectControls()) != 0 {
		return domain.DependencyResolution{}, errors.New("UV install resolution is not bound to the native runtime and wheel-only selection")
	}
	for _, node := range resolution.Graph().Nodes() {
		identity := node.Artifact().Identity()
		if identity.Source() != reference.Source() || identity.Variant() != "wheel" || node.HostInstallAction() || node.Node() == resolution.Graph().Primary() && (identity.Name() != root || exact && identity.Version() != version) {
			return domain.DependencyResolution{}, errors.New("UV install graph has an unsupported source, primary or artifact")
		}
	}
	s.resolution, s.resolved = resolution, true
	return resolution, nil
}

func (s *uvInstallSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	if !s.resolved || install != s.install || !bundle.Valid() || len(bundle.Set().Inspected().Graph().Derivations()) != 0 || !pypiUpgradeGraphExtends(s.resolution.Graph(), bundle.Set().Inspected()) {
		return domain.PromotedInstall{}, errors.New("UV install publication differs from the frozen native selection")
	}
	if err := s.verify(ctx); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := verifyPythonSelectionManifest(bundle, install, s.resolution, "UV install"); err != nil {
		return domain.PromotedInstall{}, err
	}
	s.promoteAttempted = true
	return s.owner.promoter.promote(ctx, staged, bundle, install, s.transaction, s.verifyOriginal)
}

func (s *uvInstallSession) Close() error {
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	s.closeErr = s.transaction.close()
	return s.closeErr
}

func discoverUVPythonVenv(root string) (pypiVenvPlan, error) {
	return discoverPythonVenvWithVersion(root, uvPythonVenvConfigVersion)
}

func uvPythonVenvConfigVersion(body []byte) (string, error) {
	values := map[string]string{}
	allowed := map[string]bool{"home": true, "implementation": true, "uv": true, "version_info": true, "include-system-site-packages": true, "version": true, "executable": true, "command": true, "prompt": true}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if _, duplicate := values[key]; !ok || duplicate || !allowed[key] {
			return "", errors.New("UV virtual environment configuration is ambiguous or unsupported")
		}
		values[key] = value
	}
	if implementation := values["implementation"]; implementation != "" && implementation != "CPython" {
		return "", errors.New("UV target requires CPython")
	}
	if system := values["include-system-site-packages"]; system != "" && system != "false" {
		return "", errors.New("UV target cannot expose system site packages")
	}
	version, standard := values["version"]
	_, uv := values["uv"]
	_, versionInfo := values["version_info"]
	if standard {
		if uv || versionInfo {
			return "", errors.New("UV target has conflicting version formats")
		}
	} else {
		version = values["version_info"]
		// This is creation history, never authentication of the current tool.
		parts := strings.Split(values["uv"], ".")
		if !uv || !versionInfo || len(parts) != 3 || values["implementation"] != "CPython" || values["include-system-site-packages"] != "false" {
			return "", errors.New("UV target format is unsupported")
		}
		for _, part := range parts {
			if len(part) == 0 || len(part) > 10 || strings.Trim(part, "0123456789") != "" {
				return "", errors.New("UV creation version is malformed")
			}
		}
	}
	if version != sandbox.PinnedPythonRuntime().PythonVersion {
		return "", errors.New("UV target differs from the pinned Python version")
	}
	return version, nil
}

// The update marker is deliberately not embedded or forwarded. The public
// ordinary request stays INSTALL while the inner guard binds a bare reference.
type uvInstalledInstallSession struct {
	session   ports.InstallUpdateSession
	reference domain.ArtifactReference
	install   domain.InstallContext
}

func (s *uvInstalledInstallSession) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	if reference != s.reference || install != s.install {
		return domain.DependencyResolution{}, errors.New("UV installed selection request changed")
	}
	return s.session.ResolveDependencies(ctx, s.session.Reference(), install)
}
func (s *uvInstalledInstallSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	return s.session.Promote(ctx, staged, bundle, install)
}
func (s *uvInstalledInstallSession) Close() error { return s.session.Close() }
