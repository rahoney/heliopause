package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// UVUpdatePreparation binds the native UV selection format to the existing
// Python ownership snapshot and held publication transaction. Native UV
// installations require explicitly configured independent enrollment and adoption.
type UVUpdatePreparation struct{ shared *PyPIUpdatePreparation }

func NewUVUpdatePreparation(promoter *PyPIPromotion, resolver ports.DependencyUpdateResolver, intent artifactpypi.UVSelectionIntent) (*UVUpdatePreparation, error) {
	intentControl, err := artifactpypi.NewUVSelectionIntentControl(intent)
	if err != nil {
		return nil, err
	}
	shared, err := NewPyPIUpdatePreparation(promoter, resolver)
	if err != nil {
		return nil, err
	}
	shared.discover = discoverUVPythonVenv
	shared.selectionControls = func(input artifactpypi.UpgradeInput) ([]domain.ProjectControlFile, error) {
		controls, err := pypiUpdateSelectionControls(input)
		if err != nil {
			return nil, err
		}
		return append(controls, intentControl), nil
	}
	shared.validateSelection = func(input artifactpypi.UpgradeInput, resolution domain.DependencyResolution) (domain.LockedDependencyGraph, error) {
		if err := validateUVUpdateSelection(input, intentControl, resolution); err != nil {
			return domain.LockedDependencyGraph{}, err
		}
		return resolution.Graph(), nil
	}
	shared.graphExtends = func(original domain.LockedDependencyGraph, inspected domain.InspectedDependencySet) bool {
		return reflect.DeepEqual(original, inspected.Graph())
	}
	return &UVUpdatePreparation{shared}, nil
}

func (p *UVUpdatePreparation) BeginInstallUpdate(ctx context.Context, source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (ports.InstallUpdateSession, error) {
	if p == nil || p.shared == nil || source != artifactpypi.PublicPyPIProfile().Source() || !install.Valid() {
		return nil, errors.New("UV update requires public PyPI and existing verified HAA ownership")
	}
	if !separateRoots([]string{install.Target().String(), p.shared.promoter.stagingRoot}) {
		return nil, errors.New("UV update virtual environment overlaps trusted staging")
	}
	if adopt && p.shared.native == nil {
		return nil, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", errors.New("UV native ownership collaborators are unavailable"))
	}
	return p.shared.BeginInstallUpdate(ctx, source, selector, install, adopt)
}

func validateUVUpdateSelection(input artifactpypi.UpgradeInput, intent domain.ProjectControlFile, resolution domain.DependencyResolution) error {
	original, err := input.Control()
	if err != nil {
		return err
	}
	controls := map[string][]byte{}
	for _, control := range resolution.ProjectControls() {
		if !control.Present() || controls[control.Name()] != nil {
			return errors.New("UV update selection controls are absent or ambiguous")
		}
		controls[control.Name()] = control.Body()
	}
	if len(controls) != 4 || !bytes.Equal(controls[artifactpypi.UpgradeInputControlName], original.Body()) || !bytes.Equal(controls[artifactpypi.UVSelectionIntentControlName], intent.Body()) {
		return errors.New("UV update resolver changed original input or native intent")
	}
	proposal, err := input.ParseUVSelection(controls[artifactpypi.UVSelectionDeltaControlName])
	if err != nil {
		return err
	}
	if err := proposal.ValidateGraph(resolution.Graph()); err != nil {
		return err
	}
	lockBody := controls[artifactpypi.UVSelectionLockControlName]
	lock, err := artifactpypi.ParseUVLock(proposal.Reference(), lockBody, sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(lockBody)
	if resolution.RuntimeIdentity() != sandbox.UVResolverRuntimeIdentity() || resolution.LockfileDigest().String() != hex.EncodeToString(sum[:]) || len(lock.Packages()) != len(resolution.Graph().Nodes()) {
		return errors.New("UV update native runtime, lock digest or complete graph differs")
	}
	selected := map[string]domain.ResolvedArtifact{}
	for _, node := range resolution.Graph().Nodes() {
		selected[node.Artifact().Identity().Name()] = node.Artifact()
	}
	for _, project := range lock.Packages() {
		artifact := selected[project.Name]
		if artifact.Identity().Version() != project.Version {
			return errors.New("UV update graph differs from native lock package")
		}
		matches := 0
		for _, wheel := range project.Wheels {
			if wheel.URL == artifact.AcquisitionLocator() && "sha256:"+wheel.SHA256 == artifact.DeclaredIntegrity() {
				matches++
			}
		}
		if matches != 1 {
			return errors.New("UV update graph archive differs from native lock")
		}
	}
	return nil
}

// Existing old-wheel acquisition/ownership is reused with a distinct, bounded
// native UV format. Selection itself continues to use the configured UV guard.
func (p *UVUpdatePreparation) WithNativeEnrollment(resolver ports.DependencyResolver, intake ports.Artifact, intakeRoot string) (*UVUpdatePreparation, error) {
	if p == nil || p.shared == nil {
		return nil, errors.New("UV native enrollment requires the existing guard")
	}
	shared, err := p.shared.WithNativeEnrollment(resolver, intake, intakeRoot)
	if err != nil {
		return nil, err
	}
	shared.native.client = nativeUVClient
	return &UVUpdatePreparation{shared}, nil
}
