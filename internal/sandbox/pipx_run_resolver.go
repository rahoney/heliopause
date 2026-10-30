package sandbox

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"path"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const pipxRunPreparationCommand = "helox-pipx-inert-query"

// Never pass a selected console or user argv to native pipx. Its fixed missing
// app must differ from the whole original's executable roles and primary name.
func pipxRunPreparationArguments(frozen *pipxNativeInput, requirement artifactpypi.UpgradeRequirement, command string) ([]string, error) {
	if validateUVRunRequest(requirement, command) != nil || !pipxNativeRequirementMatches(requirement, frozen) || requirement.Project() == pipxRunPreparationCommand {
		return nil, errPipxNativeInput
	}
	_, inspections, err := uvRunConsoleOriginal(frozen, command)
	if err != nil {
		return nil, err
	}
	for _, info := range inspections {
		for _, ep := range info.Surface.EntryPointDetails {
			if ep.Name == pipxRunPreparationCommand && (ep.Group == "pipx.run" || ep.Group == "console_scripts" || ep.Group == "gui_scripts") {
				return nil, errors.New("pipx inert query collides with an original entry point")
			}
			if info.Project == requirement.Project() && ep.Name == command && ep.Group == "pipx.run" {
				return nil, errors.New("pipx.run entry precedence is outside the supported console execution contract")
			}
		}
		for _, script := range info.Surface.Scripts {
			if path.Base(script) == pipxRunPreparationCommand {
				return nil, errors.New("pipx inert query collides with an original script")
			}
		}
	}
	// Resolve a bare request from the same already owned graph. Native replay
	// receives this exact pin, never another source lookup or ambient cache.
	for _, node := range frozen.graph.Nodes() {
		if node.Role() == domain.DependencyPrimary {
			identity := node.Artifact().Identity()
			requirement, err = artifactpypi.ParseUpgradeRequirement(identity.Name() + "==" + identity.Version())
			if err != nil {
				return nil, err
			}
		}
	}
	installed, err := pipxNativeArguments(pipxNativeInstall, requirement)
	if err != nil {
		return nil, err
	}
	args := append([]string(nil), installed[:6]...)
	return append(args, "run", "--backend", "uv", "--python", "/usr/local/bin/python", "--no-cache", "--no-path-check", "--spec", requirement.Requirement(), installed[len(installed)-1], pipxRunPreparationCommand), nil
}

func pipxRunPreparedTerminal(output []byte, err error, project string) bool {
	var terminal *exec.ExitError
	expected := "'" + pipxRunPreparationCommand + "' executable script not found in package '" + project + "'."
	return len(output) > 0 && len(output) <= 64<<10 && errors.As(err, &terminal) && terminal.Exited() && terminal.ExitCode() > 0 && strings.Contains(string(output), expected)
}

// Returns bound original console DATA only, after the existing session's trace,
// accounting and cleanup finish. It creates no installed ToolReceipt or Policy.
func runPipxFrozenToolRun(ctx context.Context, runner CommandRunner, observer TraceObserver, resources ObservationResourceClient, tool *pipxPreparedTool, frozen *pipxNativeInput, requirement artifactpypi.UpgradeRequirement, command string) (PipxRunSelection, error) {
	args, err := pipxRunPreparationArguments(frozen, requirement, command)
	if err != nil {
		return PipxRunSelection{}, err
	}
	var facts []uvRunCensusFact
	var nativeErr error
	_, err = withPipxNativeSession(ctx, runner, observer, resources, tool, frozen, func(run context.Context, session *pipxNativeSession) (pipxNativeOutput, error) {
		var output []byte
		output, nativeErr = session.runner.(boundedCommandRunner).RunBounded(run, "docker", boundaryExecArguments(session.reader.id, boundaryPythonHandoffMode, args...)...)
		if !pipxRunPreparedTerminal(output, nativeErr, requirement.Project()) {
			return pipxNativeOutput{}, errors.Join(errors.New("native pipx run did not reach its fixed missing-app terminal"), nativeErr)
		}
		if err := session.data.verifyReader(run, session.runner, session.reader.id); err != nil {
			return pipxNativeOutput{}, err
		}
		if err := session.wheels.verifyReader(run, session.runner, session.reader.id); err != nil {
			return pipxNativeOutput{}, err
		}
		if err := session.output.verifyAttachments(run, session.runner, map[string]bool{session.reader.id: false}); err != nil {
			return pipxNativeOutput{}, err
		}
		_, err := readUVNativeTar(run, session.runner, session.reader.id, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
			var scanErr error
			facts, scanErr = scanUVRunCensus(input, command)
			return uvNativeOutput{}, scanErr
		})
		return pipxNativeOutput{}, err
	})
	if err != nil {
		return PipxRunSelection{}, errors.Join(nativeErr, err)
	}
	return pipxRunSelectionFromCensus(ctx, requirement, frozen, command, facts)
}

// PipxRunSelection is native selection/source DATA; later current inspection,
// Policy and approved console execution remain separate existing authorities.
type PipxRunSelection struct {
	resolution domain.DependencyResolution
	entry      artifactpypi.ConsoleExecutionEntry
}

func (s PipxRunSelection) Resolution() domain.DependencyResolution   { return s.resolution }
func (s PipxRunSelection) Entry() artifactpypi.ConsoleExecutionEntry { return s.entry }

func (r *PipxResolver) ResolveToolRun(ctx context.Context, requirement artifactpypi.UpgradeRequirement, command string) (result PipxRunSelection, resultErr error) {
	if ctx == nil || r == nil || r.native == nil || r.native.reader == nil || r.resources == nil {
		return result, errors.New("pipx isolated run resolver is unavailable")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			result = PipxRunSelection{}
		}
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := validateUVRunRequest(requirement, command); err != nil {
		return result, err
	}
	// Missing installed supply fails before DNS or public source selection.
	tool, err := preparePipxTool(ctx)
	if err != nil {
		return result, err
	}
	frozen, err := r.native.prepareUVRunOriginal(ctx, requirement, command)
	if err != nil {
		return result, err
	}
	reader := r.native.reader
	return runPipxFrozenToolRun(ctx, reader.runner, reader.observer, r.resources, tool, frozen, requirement, command)
}

func pipxRunSelectionFromCensus(ctx context.Context, requirement artifactpypi.UpgradeRequirement, frozen *pipxNativeInput, command string, facts []uvRunCensusFact) (PipxRunSelection, error) {
	if ctx == nil {
		return PipxRunSelection{}, errors.New("pipx run selection requires context")
	}
	if err := ctx.Err(); err != nil {
		return PipxRunSelection{}, errors.Join(err, context.Cause(ctx))
	}
	if _, err := pipxRunPreparationArguments(frozen, requirement, command); err != nil {
		return PipxRunSelection{}, err
	}
	entry, err := bindPipxRunConsoleData(frozen, command, facts)
	if err != nil {
		return PipxRunSelection{}, err
	}
	resolution, err := pythonRunSelectionResolution(ctx, requirement, frozen, entry, facts, true)
	if err != nil {
		return PipxRunSelection{}, err
	}
	return PipxRunSelection{resolution, entry}, nil
}
