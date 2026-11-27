package sandbox

import (
	"bytes"
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

var errBunInspectionPreparation = errors.New("bun complete offline inspection preparation is unavailable")

// BunInspectionPreparer owns script-free data preparation. Observation and
// cleanup must finish before its unapproved bound layout is returned.
type BunInspectionPreparer struct {
	native *BunResolver
	intake *DockerArtifactIntroducer
}

func NewBunInspectionPreparer(native *BunResolver, intakeRoot string) (*BunInspectionPreparer, error) {
	if native == nil || native.scope == nil || native.scope.kind != nativeRuntimeBun || native.scope.runner == nil || native.scope.endpoints == nil || native.scope.observer == nil || native.scope.probe == nil || native.scope.policy == nil || native.scope.prepare == nil {
		return nil, errBunInspectionPreparation
	}
	if _, ok := native.scope.runner.(inputCommandRunner); !ok {
		return nil, errBunInspectionPreparation
	}
	if _, ok := native.scope.runner.(streamingDockerOutput); !ok {
		return nil, errBunInspectionPreparation
	}
	intake, err := NewDockerArtifactIntroducer(intakeRoot, native.scope.runner)
	if err != nil {
		return nil, errors.Join(errBunInspectionPreparation, err)
	}
	return &BunInspectionPreparer{native, intake}, nil
}

func (p *BunInspectionPreparer) PrepareInspection(ctx context.Context, input artifactnpm.BunInspectionInput, inventories []artifactnpm.ArchiveInventory) (artifactnpm.BunInspectionLayout, error) {
	if p == nil || p.native == nil || p.native.scope == nil || p.intake == nil || ctx == nil || !input.Valid() || input.Context().RuntimeIdentity() != BunResolverRuntimeIdentity() || input.Context().ClientVersion() != runtimeidentity.BunVersion {
		return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err, context.Cause(ctx))
	}
	if err := input.CheckNativeCacheInputs(inventories); err != nil {
		return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err)
	}
	archives, err := p.intake.freezeBunArchives(ctx, input)
	if err != nil {
		return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err)
	}
	var bound artifactnpm.BunInspectionLayout
	runner := p.native.scope.runner
	err = p.native.scope.run(ctx, false, func(ctx context.Context, container, network string, _ *closureVolume) error {
		if err := disconnectPNPMInspectionNetwork(ctx, runner, container, network); err != nil {
			return err
		}
		if err := copyBunProjectControls(ctx, runner, container, input.Context().Controls()); err != nil {
			return err
		}
		if err := copyBunInspectionSeed(ctx, runner, container, input, archives); err != nil {
			return err
		}
		if err := runBunInspectionCommand(ctx, runner, container, "seed"); err != nil {
			return err
		}
		var entries []artifactnpm.BunNativeCacheEntry
		for _, inventory := range inventories {
			entry, err := readBunInspectionCache(ctx, runner.(streamingDockerOutput), container, input, inventory)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
		}
		registry, err := input.RenderRegistryCache(ctx, entries)
		if err != nil {
			return err
		}
		if err := runner.(inputCommandRunner).RunInput(ctx, bytes.NewReader(registry), "docker", "cp", "--archive", "-", container+":"+bunProjectRoot); err != nil {
			return err
		}
		if err := runBunInspectionCommand(ctx, runner, container, "install"); err != nil {
			return err
		}
		output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
		if err := runner.(streamingDockerOutput).RunOutput(ctx, &output, "docker", "cp", container+":"+bunProjectDirectory, "-"); err != nil || output.failed {
			return errors.Join(errBunInspectionPreparation, err)
		}
		bound, err = bindBunInspectionProject(ctx, input, inventories, output.body)
		return err
	})
	if err != nil {
		return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err, ctx.Err(), context.Cause(ctx))
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.BunInspectionLayout{}, errors.Join(errBunInspectionPreparation, err, context.Cause(ctx))
	}
	if !bound.MatchesInput(input) {
		return artifactnpm.BunInspectionLayout{}, errBunInspectionPreparation
	}
	return bound, nil
}

func bunInspectionCommand(operation string) []string {
	directory, mode, tail := bunProjectDirectory, "--offline", "--frozen-lockfile"
	if operation == "seed" {
		directory = bunProjectRoot + "/seed"
		mode = "--lockfile-only"
		tail = "--offline"
	} else if operation != "install" {
		return nil
	}
	return []string{bunToolGuestPath, "--cwd=" + directory, "install", mode, "--ignore-scripts", "--no-env-file", "--registry=https://registry.npmjs.org/", "--cache-dir=" + bunProjectRoot + "/cache", "--backend=copyfile", "--linker=hoisted", "--save-text-lockfile", "--no-progress", "--no-summary", tail}
}

func runBunInspectionCommand(ctx context.Context, runner CommandRunner, container, operation string) error {
	arguments := bunInspectionCommand(operation)
	if ctx == nil || runner == nil || !exactObservationContainerID(container) || len(arguments) != 14 {
		return errBunInspectionPreparation
	}
	_, err := runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, arguments...)...)
	if err != nil || ctx.Err() != nil {
		return errors.Join(errBunInspectionPreparation, err, ctx.Err(), context.Cause(ctx))
	}
	return nil
}
