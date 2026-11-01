package sandbox

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// PrepareNPMProjectInspection reuses the script-free Node data materializer.
// The frozen source controls and measured archives are unapproved inputs here;
// native preparation never runs the project or its dependencies' scripts.
func (b *Backend) PrepareNPMProjectInspection(ctx context.Context, input artifactnpm.NPMProjectInspectionInput, inventories []artifactnpm.ArchiveInventory) (artifactnpm.NPMProjectInspectionLayout, error) {
	if b == nil || ctx == nil || !input.Valid() || input.Context().RuntimeIdentity() != NPMProjectRuntimeIdentity() || input.Context().ClientVersion() != resolverNPMVersion {
		return artifactnpm.NPMProjectInspectionLayout{}, errors.New("npm project preparation lacks current native source context")
	}
	if err := input.CheckWorkspace(inventories); err != nil {
		return artifactnpm.NPMProjectInspectionLayout{}, err
	}
	controls, err := input.OfflineControls()
	if err != nil {
		return artifactnpm.NPMProjectInspectionLayout{}, err
	}
	var layout artifactnpm.NPMProjectInspectionLayout
	err = b.withMaterializedNodeInputs(ctx, input.Context().Snapshot().GraphDigest().String(), input.Artifacts(), controls, true, func(run context.Context, native artifactnpm.PNPMLayoutArchive) error {
		var err error
		layout, err = input.BindMaterializedLayout(run, inventories, native)
		return err
	})
	if err != nil {
		return artifactnpm.NPMProjectInspectionLayout{}, err
	}
	return layout, nil
}

const npmProjectTargetLifecycleCommand = `mkdir -p /tmp/.npm; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm --prefix /tmp/haa-node-execution/project rebuild --ignore-scripts=false --bin-links=false --foreground-scripts --offline --no-update-notifier --workspaces=false --no-audit --no-fund --dangerously-allow-all-scripts "$1"`

func validNPMProjectTargetLifecyclePath(target string) bool {
	return validHoistedTargetLifecyclePath(target, nodeExecutionProject)
}

func npmProjectTargetLifecycleExecArguments(container, target string) ([]string, error) {
	if !exactObservationContainerID(container) || !validNPMProjectTargetLifecyclePath(target) {
		return nil, errors.New("npm project lifecycle requires a fixed current instance and exact container")
	}
	return boundaryExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", npmProjectTargetLifecycleCommand, "helox-npm-project-target", target), nil
}

func (b *Backend) ExecuteNPMProjectInstance(ctx context.Context, layout artifactnpm.NPMProjectInspectionLayout, subject domain.AcquiredArtifact, instance string) (domain.SandboxResult, error) {
	if b == nil || ctx == nil || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || layout.RuntimeIdentity() != NPMProjectRuntimeIdentity() {
		return domain.SandboxResult{}, errors.New("npm project instance lacks its current bound runtime")
	}
	if err := ctx.Err(); err != nil {
		return domain.SandboxResult{}, errors.Join(err, context.Cause(ctx))
	}
	observer, ok := b.observer.(interface {
		StartNPMProjectTargetLifecycleProfile(context.Context, string) (TraceReader, error)
	})
	if !ok {
		return domain.SandboxResult{}, errors.New("npm project instance has no typed observer")
	}
	relative, err := layout.Target(subject, instance)
	target := nodeExecutionProject + "/" + relative
	if err != nil || !validNPMProjectTargetLifecyclePath(target) {
		return domain.SandboxResult{}, errors.Join(errors.New("npm project instance target differs"), err)
	}
	payload, err := prepareNPMInspectionPayload(ctx, layout, nativeRuntimeNPMProject)
	if err != nil {
		return domain.SandboxResult{}, err
	}
	return b.executeNPMInstance(ctx, nativeRuntimeNPMProject, target, payload, observer.StartNPMProjectTargetLifecycleProfile)
}
