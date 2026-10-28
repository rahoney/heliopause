package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvToolUpgradeNativeSelector interface {
	ResolveToolUpgrade(context.Context, artifactpypi.UVToolUpgradeInput, []artifactpypi.UVInstalledWheelControls) (sandbox.UVToolSelection, error)
}

type uvToolUpdateNativeSelector interface {
	uvToolNativeSelector
	uvToolUpgradeNativeSelector
}

// UVToolUpdatePreparation reuses the held tool session and current whole-set
// update pipeline. Native receipts never supply implicit original ownership.
type UVToolUpdatePreparation struct{ shared *UVToolInstallPreparation }

func NewUVToolUpdatePreparation(promoter *PyPIPromotion, resolver uvToolUpdateNativeSelector, binRoot string, trustedRoots []string) (*UVToolUpdatePreparation, error) {
	shared, err := NewUVToolInstallPreparation(promoter, resolver, binRoot, trustedRoots)
	if err != nil {
		return nil, err
	}
	shared.upgrade = resolver
	return &UVToolUpdatePreparation{shared}, nil
}

func (p *UVToolUpdatePreparation) WithNativeEnrollment(resolver ports.DependencyResolver, intake ports.Artifact, intakeRoot string) (*UVToolUpdatePreparation, error) {
	if p == nil || p.shared == nil || resolver == nil || intake == nil || !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot {
		return nil, errors.New("UV tool native enrollment requires actual original-source capabilities")
	}
	bound := false
	for _, root := range p.shared.trustedRoots {
		bound = bound || root == intakeRoot
	}
	if !bound {
		return nil, errors.New("UV tool original intake is outside frozen storage boundaries")
	}
	shared := *p.shared
	shared.native = &pypiNativeEnrollment{resolver: resolver, intake: intake, intakeRoot: intakeRoot, runner: shared.promoter.runner, client: nativeUVClient}
	return &UVToolUpdatePreparation{shared: &shared}, nil
}

func (p *UVToolUpdatePreparation) BeginInstallUpdate(ctx context.Context, source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (result ports.InstallUpdateSession, resultErr error) {
	if p == nil || p.shared == nil || ctx == nil || source != artifactpypi.PublicPyPIProfile().Source() || !install.Valid() {
		return nil, errors.New("UV tool upgrade requires one supported existing public tool")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if adopt && p.shared.native == nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", errors.New("UV native tool adoption requires independent original enrollment"))
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(selector)
	if err != nil {
		return nil, err
	}
	reference, err := domain.NewArtifactReference(source, requirement.Requirement())
	if err != nil {
		return nil, err
	}
	var guard *toolFreshTarget
	if adopt {
		guard, err = beginToolNativeOriginalTarget(ctx, install, requirement, p.shared.binRoot, p.shared.trustedRoots, p.shared.native)
	} else {
		guard, err = beginToolOriginalTarget(ctx, install, requirement, p.shared.binRoot, p.shared.trustedRoots)
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, guard.Close())
			result = nil
		}
	}()
	wheels, err := captureUVToolOriginalWheels(ctx, guard)
	if err != nil {
		return nil, err
	}
	return &uvToolInstallSession{owner: p.shared, guard: guard, reference: reference, install: install, originalWheels: wheels}, nil
}

func (s *uvToolInstallSession) Reference() domain.ArtifactReference { return s.reference }

func captureUVToolOriginalWheels(ctx context.Context, g *toolFreshTarget) ([]artifactpypi.UVInstalledWheelControls, error) {
	if g == nil || g.original == nil {
		return nil, errors.New("UV tool original guard is unavailable")
	}
	if err := g.verify(ctx); err != nil {
		return nil, err
	}
	var wheels []artifactpypi.UVInstalledWheelControls
	total := len(g.original.input.Receipt().Body())
	for _, metadata := range g.original.input.Original().MetadataFiles() {
		key := "site/" + metadata.Directory + "/"
		main, ok := g.tx.current.Files[key+"METADATA"]
		wheel, present := g.tx.current.Files[key+"WHEEL"]
		path := filepath.Join(g.tx.plan.site, metadata.Directory, "WHEEL")
		before := g.tx.before[path]
		if !ok || !present || len(main.owners()) != 1 || len(wheel.owners()) != 1 ||
			main.Distribution != wheel.Distribution || main.Version != wheel.Version || main.ArtifactDigest == "" || main.ArtifactDigest != wheel.ArtifactDigest ||
			before == nil || before.Digest != wheel.Digest || before.Size > 1<<20 || !before.matches(path) {
			return nil, errors.New("UV tool original WHEEL lacks the same held archive owner as METADATA")
		}
		body, err := readBoundedPromotionFile(path, 1<<20)
		if err != nil || !before.matches(path) {
			return nil, errors.New("UV tool original WHEEL changed while reading")
		}
		sum := sha256.Sum256(body)
		if int64(len(body)) != before.Size || hex.EncodeToString(sum[:]) != before.Digest {
			return nil, errors.New("UV tool original WHEEL read differs from the held snapshot")
		}
		total += len(metadata.Body) + len(body)
		if total > 4<<20 {
			return nil, errors.New("UV tool original controls exceed combined bounds")
		}
		wheels = append(wheels, artifactpypi.UVInstalledWheelControls{DistInfo: metadata.Directory, Metadata: metadata.Body, Wheel: body})
	}
	runtime := sandbox.PinnedPythonRuntime()
	checked, err := g.original.input.ReplayWheelControls(wheels, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		return nil, err
	}
	if err := g.verify(ctx); err != nil {
		return nil, err
	}
	return checked, nil
}

func (s *uvToolInstallSession) validateSelection(resolution domain.DependencyResolution, controls []domain.ProjectControlFile) error {
	if s.owner.pipx != nil {
		if s.guard.original != nil || s.guard.client != toolClientPipx {
			return errors.New("pipx fresh selection cannot consume an original tool")
		}
		return validateUVToolNativeSelection(resolution, controls, s.guard.requirement, "pipx-tool-install:"+sandbox.PipxResolverRuntimeIdentity(), []string{"pipx-tool-intent.json", "pipx-tool-receipt.json", "pipx-tool-pylock.toml", "pipx-tool-census.json"})
	}
	if s.guard.original == nil {
		return validateUVToolSelection(resolution, controls, s.guard.requirement)
	}
	names := []string{"uv-tool-intent.json", "uv-tool-receipt.toml", "uv-tool-pylock.toml", "uv-tool-census.json", artifactpypi.UVToolUpgradeInputControlName, artifactpypi.UVToolUpgradeWheelsControlName}
	if err := validateUVToolNativeSelection(resolution, controls, s.guard.requirement, "uv-tool-upgrade:"+sandbox.UVResolverRuntimeIdentity(), names); err != nil {
		return err
	}
	original, err := s.guard.original.input.Control()
	if err != nil {
		return err
	}
	runtime := sandbox.PinnedPythonRuntime()
	wheels, err := s.guard.original.input.ReplayWheelsControl(s.originalWheels, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		return err
	}
	for _, control := range controls {
		if control.Name() == original.Name() && !bytes.Equal(control.Body(), original.Body()) ||
			control.Name() == wheels.Name() && !bytes.Equal(control.Body(), wheels.Body()) {
			return errors.New("UV tool native selection changed the held original controls")
		}
	}
	return nil
}
