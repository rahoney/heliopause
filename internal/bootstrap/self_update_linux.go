//go:build linux

package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/rahoney/heliopause/internal/application"
	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationgithub "github.com/rahoney/heliopause/internal/verification/githubrelease"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

func selfUpdatePlatform() error {
	if runtime.GOARCH != "amd64" {
		return ports.NewInstallUpdateFailure("SELF_UPDATE_PLATFORM_UNSUPPORTED", errors.New("native quarantine update requires Linux amd64"))
	}
	return nil
}
func selfUpdateFailure(err error) error { return promotion.NativeReleaseUpdateFailure(err) }
func selfUpdateRuntime(ctx context.Context) error {
	if err := selfUpdatePlatform(); err != nil {
		return err
	}
	if ctx == nil {
		return selfUpdateFailure(releaseactivation.ErrUntrusted)
	}
	lease, _ := ctx.Value(commandRuntimeKey{}).(*releaseactivation.RuntimeLease)
	old, err := lease.Installed()
	if err != nil {
		return promotion.NativeReleaseUpdateFailure(err)
	}
	return promotion.NativeReleaseUpdateFailure(releaseactivation.CheckRuntimeController(old))
}

func newSelfUpdater(ctx context.Context, executor *hosttool.Executor, supervisor *sandbox.ObserverSupervisor, observer sandbox.TraceObserver) (cli.Installer, error) {
	if err := selfUpdateRuntime(ctx); err != nil {
		return nil, err
	}
	lease, _ := ctx.Value(commandRuntimeKey{}).(*releaseactivation.RuntimeLease)
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(cache, "heliopause")
	intakeRoot, evidenceRoot, stagingRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
	baseSource, err := artifactgithub.NewPublicClient(intakeRoot)
	if err != nil {
		return nil, err
	}
	static, err := inspectiongithub.NewStaticInspector(intakeRoot)
	if err != nil {
		return nil, err
	}
	backend, err := sandbox.NewLinuxHeliopauseNativeBackendWithExecutor(intakeRoot, executor, observer)
	if err != nil {
		return nil, err
	}
	dynamic, err := inspectiongithub.NewDynamicInspector(backend)
	if err != nil {
		return nil, err
	}
	native, err := inspectiongithub.NewCompositeInspector(static, dynamic)
	if err != nil {
		return nil, err
	}
	pipxBackend, err := sandbox.NewLinuxPipxRuntimeBackendWithExecutor(executor, observer, newObservationResourceAdapter())
	if err != nil {
		return nil, err
	}
	pipxInspection, err := inspectiongithub.NewPipxRuntimeInspector(intakeRoot, pipxReleaseObservationRunner{backend: pipxBackend})
	if err != nil {
		return nil, err
	}
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		return nil, err
	}
	inspection, err := application.NewInstallInspectService(baseSource, baseSource, verificationgithub.IntegrityVerifier{}, native, store, policy.M6{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	staging, err := promotion.NewLocalStaging(intakeRoot, evidenceRoot, stagingRoot)
	if err != nil {
		return nil, err
	}
	basePromoter, err := promotion.NewGitHubReleasePromotion(stagingRoot)
	if err != nil {
		return nil, err
	}
	base, err := application.NewInstallService(inspection, evidence.Generator{}, staging, basePromoter)
	if err != nil {
		return nil, err
	}
	factory := func(version, lock string, install domain.InstallContext) (content promotion.NativeReleaseContent, resultErr error) {
		client, err := artifactgithub.NewPublicClient(intakeRoot)
		if err != nil {
			return content, err
		}
		source, err := artifactgithub.NewHeliopauseReleaseSet(client, version, lock, install)
		if err != nil {
			return content, err
		}
		content.Source = source
		content.Verification, err = releaseprovenance.NewResolvedArtifactVerifier(func(artifact domain.AcquiredArtifact) (releaseprovenance.Identity, [][]byte, error) {
			manifest, proof, err := source.ReleaseProof(artifact)
			return releaseprovenance.Identity{Version: manifest.Version(), SourceCommit: manifest.SourceCommit(), WorkflowRun: manifest.WorkflowRun()}, proof, err
		})
		if err != nil {
			return content, err
		}
		content.Inspection, err = inspectiongithub.NewReleaseSetInspector(intakeRoot, source.Manifest, native, pipxInspection)
		return content, err
	}
	var once sync.Once
	var cleanupErr error
	drain := func() error {
		once.Do(func() { cleanupErr = errors.Join(supervisor.Close(), executor.Close()) })
		return cleanupErr
	}
	preparation, err := promotion.NewNativeReleaseUpdatePreparation(lease, stagingRoot, factory, drain)
	if err != nil {
		return nil, err
	}
	updater, err := application.NewInstallUpdateService(base, preparation)
	if err != nil {
		return nil, err
	}
	return updater, nil
}

// This adapter preserves the existing transaction result without adding a
// dependency from the release inspector back into sandbox implementation.
type pipxReleaseObservationRunner struct{ backend *sandbox.PythonDynamicBackend }

func (r pipxReleaseObservationRunner) InspectPipxRuntime(ctx context.Context, artifact domain.AcquiredArtifact, plan artifactpypi.PipxRuntimePlan) (inspectiongithub.PipxRuntimeObservation, error) {
	observed, err := r.backend.InspectPipxRuntime(ctx, artifact, plan)
	if err != nil {
		return inspectiongithub.PipxRuntimeObservation{}, err
	}
	result := inspectiongithub.PipxRuntimeObservation{SandboxResult: observed.SandboxResult}
	for _, command := range observed.Commands {
		result.Commands = append(result.Commands, inspectiongithub.PipxCommandObservation{Module: command.Module, UnitID: command.UnitID, OwnerSHA256: command.OwnerSHA256, ZeroExit: command.ZeroExit})
	}
	return result, nil
}
