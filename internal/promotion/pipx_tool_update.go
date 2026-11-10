package promotion

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type pipxToolUpgradeNativeSelector interface {
	ResolveToolUpgrade(context.Context, artifactpypi.PipxToolUpgradeInput, []artifactpypi.UVInstalledWheelControls) (sandbox.PipxToolSelection, error)
}
type pipxToolUpdateNativeSelector interface {
	pipxToolNativeSelector
	pipxToolUpgradeNativeSelector
}

// Managed state is mandatory by default. Native adoption needs explicit source
// capabilities and opt-in; old recipe/interpreter claims never become argv.
type PipxToolUpdatePreparation struct{ shared *UVToolInstallPreparation }

func NewPipxToolUpdatePreparation(promoter *PyPIPromotion, resolver pipxToolUpdateNativeSelector, binRoot string, trustedRoots []string) (*PipxToolUpdatePreparation, error) {
	shared, err := newToolInstallPreparation(promoter, nil, resolver, binRoot, trustedRoots)
	if err != nil {
		return nil, err
	}
	shared.pipxUpgrade = resolver
	return &PipxToolUpdatePreparation{shared: shared}, nil
}

func (p *PipxToolUpdatePreparation) BeginInstallUpdate(ctx context.Context, source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (result ports.InstallUpdateSession, resultErr error) {
	if p == nil || p.shared == nil || ctx == nil || source != artifactpypi.PublicPyPIProfile().Source() || !install.Valid() {
		return nil, errors.New("pipx upgrade requires one supported existing public tool")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if adopt && p.shared.native == nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", errors.New("pipx native adoption is unavailable"))
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(selector)
	if err != nil || requirement.Requirement() != requirement.Project() || selector != requirement.Project() {
		return nil, errors.New("pipx upgrade accepts one canonical bare project")
	}
	reference, err := domain.NewArtifactReference(source, requirement.Requirement())
	if err != nil {
		return nil, err
	}
	var guard *toolFreshTarget
	if adopt {
		guard, err = beginToolNativeOriginalTargetForClient(ctx, install, requirement, p.shared.binRoot, p.shared.trustedRoots, p.shared.native, toolClientPipx)
	} else {
		guard, err = beginToolTargetForClient(ctx, install, requirement, p.shared.binRoot, p.shared.trustedRoots, true, nil, toolClientPipx)
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
	wheels, err := captureToolOriginalWheels(ctx, guard)
	if err != nil {
		return nil, err
	}
	return &uvToolInstallSession{owner: p.shared, guard: guard, reference: reference, install: install, originalWheels: wheels}, nil
}

func (s *toolOriginalState) receipt() artifactpypi.ToolReceipt {
	if s.pipx != nil {
		return s.pipx.Receipt()
	}
	return s.input.Receipt()
}
func (s *toolOriginalState) selectionRequirement(request artifactpypi.UpgradeRequirement) artifactpypi.UpgradeRequirement {
	if s.pipx != nil {
		return request
	}
	return s.input.Receipt().Requirement()
}
func (s *toolOriginalState) metadataFiles() []artifactpypi.InstalledMetadataFile {
	if s.pipx != nil {
		return s.pipx.Original().MetadataFiles()
	}
	return s.input.Original().MetadataFiles()
}
func (s *toolOriginalState) replayWheelControls(wheels []artifactpypi.UVInstalledWheelControls, target artifactpypi.WheelTarget) ([]artifactpypi.UVInstalledWheelControls, error) {
	if s.pipx != nil {
		return s.pipx.ReplayWheelControls(wheels, target)
	}
	return s.input.ReplayWheelControls(wheels, target)
}

func (s *uvToolInstallSession) validatePipxSelection(resolution domain.DependencyResolution, controls []domain.ProjectControlFile) error {
	if s.guard.client != toolClientPipx {
		return errors.New("pipx selection has a foreign held client")
	}
	names := []string{"pipx-tool-intent.json", "pipx-tool-receipt.json", "pipx-tool-pylock.toml", "pipx-tool-census.json"}
	operation := "install"
	var original, wheels domain.ProjectControlFile
	if s.guard.original != nil {
		input := s.guard.original.pipx
		if input == nil || s.owner.pipxUpgrade == nil {
			return errors.New("pipx original provenance unavailable")
		}
		operation = "upgrade"
		names = append(names, artifactpypi.PipxToolUpgradeInputControlName, artifactpypi.PipxToolUpgradeWheelsControlName, "pipx-tool-selection-delta.json")
		var err error
		original, err = input.Control()
		if err != nil {
			return err
		}
		runtime := sandbox.PinnedPythonRuntime()
		wheels, err = input.ReplayWheelsControl(s.originalWheels, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
		if err != nil {
			return err
		}
	}
	if err := validateUVToolNativeSelection(resolution, controls, s.guard.requirement, "pipx-tool-"+operation+":"+sandbox.PipxResolverRuntimeIdentity(), names); err != nil {
		return err
	}
	for _, control := range controls {
		if control.Name() == original.Name() && !bytes.Equal(control.Body(), original.Body()) || control.Name() == wheels.Name() && !bytes.Equal(control.Body(), wheels.Body()) {
			return errors.New("pipx native selection changed held original controls")
		}
	}
	return nil
}

// The explicit capability authenticates only UV-generated native originals.
// Current whole-set inspection and approval remain mandatory after enrollment.
func (p *PipxToolUpdatePreparation) WithNativeEnrollment(resolver ports.DependencyResolver, intake ports.Artifact, intakeRoot string) (*PipxToolUpdatePreparation, error) {
	if p == nil || p.shared == nil || resolver == nil || intake == nil || !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot {
		return nil, errors.New("native pipx enrollment requires actual old-source capabilities")
	}
	bound := false
	for _, root := range p.shared.trustedRoots {
		bound = bound || root == intakeRoot
	}
	if !bound {
		return nil, errors.New("native pipx original intake is outside held trusted storage")
	}
	shared := *p.shared
	shared.native = &pypiNativeEnrollment{resolver: resolver, intake: intake, intakeRoot: intakeRoot, runner: shared.promoter.runner, client: nativeUVClient}
	return &PipxToolUpdatePreparation{shared: &shared}, nil
}
