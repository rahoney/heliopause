package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const pythonEnvironmentPreparationPrefix = "/tmp/haa-environment-"

// A profile owns one finite workspace. Each observation mounts only its own
// opaque unit directory; the trusted readonly keeper retains the whole tree.
type pythonEnvironmentWorkspace struct {
	profile inspectionEnvironmentProfile
	volume  closureVolume
	anchor  string
	before  environmentState
}

func pythonEnvironmentUnitRoot(id string) string {
	sum := sha256.Sum256([]byte(id))
	return "unit-" + hex.EncodeToString(sum[:])
}

func pythonEnvironmentUnits(original []observationUnit) []observationUnit {
	units := make([]observationUnit, 0, 2*len(original))
	for _, unit := range original {
		for _, profile := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
			experiment := unit
			experiment.originalID = unit.id
			experiment.environment = profile
			sum := sha256.Sum256([]byte(string(profile) + "\x00" + unit.id))
			experiment.id = string(unit.kind) + ":" + hex.EncodeToString(sum[:])
			units = append(units, experiment)
		}
	}
	return units
}

func (t *pythonTransactionExecutor) createEnvironmentWorkspaces(ctx context.Context) error {
	// Both profiles share the former writable scratch ceiling. They do not
	// multiply the request's CPU, wall, trace, launch or retained tmpfs budget.
	capacity := (t.policy.RuntimeTmpfs() - responderSupplyCapacity) / 2
	if capacity <= 0 {
		return errEnvironmentState
	}
	t.environments = make([]pythonEnvironmentWorkspace, 0, 2)
	seen := map[string]bool{t.transaction: true}
	for _, kind := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
		session, err := t.backend.newSessionID()
		if err != nil || seen[session.String()] {
			return errors.Join(errEnvironmentState, err)
		}
		seen[session.String()] = true
		profile, err := newInspectionEnvironmentProfile(kind, session, t.responder)
		if err != nil {
			return err
		}
		volume, err := createEnvironmentWorkspace(ctx, t.backend.runner, session, profile, capacity)
		if err != nil {
			return err
		}
		t.environments = append(t.environments, pythonEnvironmentWorkspace{profile: profile, volume: volume})
		index := len(t.environments) - 1
		t.activeEnvironment = &t.environments[index]
		reader, err := t.startPhase(ctx, phaseAnchor, index+2)
		t.activeEnvironment = nil
		if err != nil {
			return err
		}
		t.environments[index].anchor = reader.id
	}
	return nil
}

func (t *pythonTransactionExecutor) seedEnvironmentWorkspaces(ctx context.Context, preparation string, units []observationUnit) error {
	input, ok := t.backend.runner.(inputCommandRunner)
	if !ok {
		return errEnvironmentState
	}
	roots := make([]string, len(units))
	for index, unit := range units {
		roots[index] = pythonEnvironmentUnitRoot(unit.id)
	}
	for index := range t.environments {
		environment := &t.environments[index]
		payload, err := environment.profile.payload(roots...)
		if err != nil {
			return err
		}
		destination := pythonEnvironmentPreparationPrefix + string(environment.profile.kind)
		if err := input.RunInput(ctx, bytes.NewReader(payload), "docker", "cp", "--archive", "-", preparation+":"+destination); err != nil {
			return err
		}
	}
	return nil
}

func (t *pythonTransactionExecutor) captureEnvironmentBaselines(ctx context.Context, units []observationUnit) error {
	roots := make([]string, len(units))
	for index, unit := range units {
		roots[index] = pythonEnvironmentUnitRoot(unit.id)
	}
	for index := range t.environments {
		environment := &t.environments[index]
		if err := verifyObservationRuntimeAlive(ctx, t.backend.runner, environment.anchor); err != nil {
			return err
		}
		if err := environment.volume.verifyAttachments(ctx, t.backend.runner, map[string]bool{environment.anchor: true}); err != nil {
			return err
		}
		state, err := captureEnvironmentState(ctx, t.backend.runner, environment.anchor, environmentStateLimits{entries: int(domain.MaximumObservationSummaryCount), bytes: environment.volume.capacity, archiveBytes: 2 * environment.volume.capacity})
		if err != nil {
			return err
		}
		if err := environment.profile.verifyPlacement(state, roots...); err != nil {
			return err
		}
		environment.before = state
	}
	return nil
}

func (t *pythonTransactionExecutor) finishEnvironmentWorkspaces(ctx context.Context) error {
	for index := range t.environments {
		environment := &t.environments[index]
		if err := verifyObservationRuntimeAlive(ctx, t.backend.runner, environment.anchor); err != nil {
			return err
		}
		if err := environment.volume.verifyAttachments(ctx, t.backend.runner, map[string]bool{environment.anchor: true}); err != nil {
			return err
		}
		after, err := captureEnvironmentState(ctx, t.backend.runner, environment.anchor, environmentStateLimits{entries: int(domain.MaximumObservationSummaryCount), bytes: environment.volume.capacity, archiveBytes: 2 * environment.volume.capacity})
		if err != nil {
			return err
		}
		changes, err := compareEnvironmentStates(environment.before, after)
		if err != nil {
			return err
		}
		fixture, _ := domain.NewSHA256Digest(environment.profile.identity)
		before, _ := domain.NewSHA256Digest(environment.before.digest)
		final, _ := domain.NewSHA256Digest(after.digest)
		t.environmentStates = append(t.environmentStates, domain.SandboxEnvironmentState{Profile: environment.profile.kind, FixtureDigest: fixture, Before: before, After: final, Created: changes.created, Deleted: changes.deleted, Modified: changes.modified, VFSOperationsCovered: true})
		fact, err := domain.NewSandboxObservation(domain.ObservationProcess, "environment-"+string(environment.profile.kind)+"-completed")
		if err != nil {
			return err
		}
		t.allObserved = append(t.allObserved, fact)
	}
	return nil
}

func (t *pythonTransactionExecutor) environmentRuntimeArguments(args []string, phase pythonObservationPhase) ([]string, error) {
	if len(t.environments) == 0 {
		return args, nil
	}
	if phase == phasePreparation {
		mounts := make([]string, 0, 4)
		for _, environment := range t.environments {
			volume := environment.volume
			volume.projectBuildDestination = pythonEnvironmentPreparationPrefix + string(environment.profile.kind)
			mount, err := volume.mountArgument(false)
			if err != nil {
				return nil, err
			}
			mounts = append(mounts, "--mount", mount)
		}
		for index, arg := range args {
			if arg == pythonImageReference {
				return append(args[:index], append(mounts, args[index:]...)...), nil
			}
		}
		return nil, errEnvironmentState
	}
	environment := t.activeEnvironment
	if environment == nil {
		return args, nil
	} // Existing authenticated closure anchor.
	if phase == phaseAnchor {
		// The profile keeper reads only /tmp; it does not attach to the code closure.
		mount, err := t.volume.mountArgument(true)
		if err != nil {
			return nil, err
		}
		removed := false
		for index := 0; index < len(args); index++ {
			if args[index] == "--mount" && index+1 < len(args) && args[index+1] == mount {
				args = append(args[:index], args[index+2:]...)
				removed = true
				break
			}
		}
		if !removed {
			return nil, errEnvironmentState
		}
		args, err = applyEnvironmentWorkspace(args, pythonImageReference, environment.profile, environment.volume, true, "")
		if err != nil {
			return nil, err
		}
		return args, nil
	}
	if phase != phaseObservation || t.activeUnitOriginal == "" {
		return nil, errEnvironmentState
	}
	return applyEnvironmentWorkspace(args, pythonImageReference, environment.profile, environment.volume, false, pythonEnvironmentUnitRoot(t.activeUnitOriginal))
}

func (t *pythonTransactionExecutor) verifyEnvironmentRuntimeMounts(ctx context.Context, id string, phase pythonObservationPhase) error {
	if phase == phasePreparation {
		for _, environment := range t.environments {
			volume := environment.volume
			volume.projectBuildDestination = pythonEnvironmentPreparationPrefix + string(environment.profile.kind)
			if err := volume.verifyContainerMount(ctx, t.backend.runner, id, false); err != nil {
				return err
			}
		}
		return nil
	}
	if t.activeEnvironment == nil {
		return nil
	}
	if phase == phaseObservation {
		environment := t.activeEnvironment
		if err := verifyObservationRuntimeAlive(ctx, t.backend.runner, environment.anchor); err != nil {
			return err
		}
		return environment.volume.verifyAttachments(ctx, t.backend.runner, map[string]bool{environment.anchor: true, id: false}, map[string]string{id: pythonEnvironmentUnitRoot(t.activeUnitOriginal)})
	}
	return t.activeEnvironment.volume.verifyContainerMount(ctx, t.backend.runner, id, true)
}
