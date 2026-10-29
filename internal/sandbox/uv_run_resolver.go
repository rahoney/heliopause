package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// This cache exists only in the fresh owned native output volume.
const uvRunCensusCache = "/haa-site/uv-run-cache"

// UVRunSelection is measured source/native-selection DATA. It grants neither
// current inspection/Policy nor program admission, ownership or installation.
type UVRunSelection struct {
	resolution domain.DependencyResolution
	entry      artifactpypi.ConsoleExecutionEntry
}

func (s UVRunSelection) Resolution() domain.DependencyResolution   { return s.resolution }
func (s UVRunSelection) Entry() artifactpypi.ConsoleExecutionEntry { return s.entry }

func validateUVRunRequest(requirement artifactpypi.UpgradeRequirement, command string) error {
	reference, err := artifactpypi.ParseReference(requirement.Reference().Locator())
	if err != nil || reference != requirement.Reference() || reference.Source() != artifactpypi.PublicPyPIProfile().Source() || artifactpypi.ValidateConsoleExecutionCommand(command) != nil {
		return errors.New("UV isolated run requires a canonical public bare/exact reference and explicit console command")
	}
	canonical, err := artifactpypi.ParseUpgradeRequirement(requirement.Requirement())
	if err != nil || canonical.Requirement() != requirement.Requirement() || canonical.Reference() != reference {
		return errors.New("UV isolated run requirement is outside the bounded public request")
	}
	if requirement.Requirement() != requirement.Project() {
		// The shared upgrade reference intentionally names only the project.
		// Reuse the exact reference parser, rather than guessing from ==.
		exact, err := artifactpypi.ParseReference(strings.Replace(requirement.Requirement(), "==", "@", 1))
		if err != nil {
			return errors.New("UV isolated run requires one bare or exact public request")
		}
		roundtrip, err := artifactpypi.ParseUpgradeRequirement(exact.Locator())
		if err != nil || roundtrip.Requirement() != requirement.Requirement() {
			return errors.New("UV isolated run requirement differs from its canonical exact reference")
		}
	}
	return nil
}

func uvRunPreparationArguments(requirement artifactpypi.UpgradeRequirement, command string) ([]string, error) {
	if err := validateUVRunRequest(requirement, command); err != nil {
		return nil, err
	}
	// Never pass the selected console or user argv to native UV. It prepares
	// from whole startup-safe originals and reaches only this inert terminal.
	return []string{uvToolGuestPath, "tool", "run", "--isolated", "--from", requirement.Requirement(), "--offline", "--no-index", "--find-links", pipxNativeWheels, "--no-build", "--no-config", "--cache-dir", uvRunCensusCache, "--no-python-downloads", "--no-managed-python", "--python", "/usr/local/bin/python", "--link-mode", "copy", "--no-sources", "/dev/null"}, nil
}

func uvRunPreparedTerminal(output []byte, err error) bool {
	var terminal *exec.ExitError
	return len(output) == 0 && errors.As(err, &terminal) && terminal.Exited() && terminal.ExitCode() > 0 && len(terminal.Stderr) > 0 && len(terminal.Stderr) <= 64<<10 && strings.Contains(string(terminal.Stderr), "/dev/null") && strings.Contains(string(terminal.Stderr), "Permission denied")
}

// ResolveToolRun independently binds original archives before native Python
// queries. Native installed files stay DATA; later inspection and execution
// must consume the same original graph through their existing current owners.
func (r *UVResolver) ResolveToolRun(ctx context.Context, requirement artifactpypi.UpgradeRequirement, command string) (result UVRunSelection, resultErr error) {
	if ctx == nil || r == nil || r.reader == nil {
		return UVRunSelection{}, errors.New("UV isolated run resolver is unavailable")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			result = UVRunSelection{}
		}
	}()
	if err := ctx.Err(); err != nil {
		return UVRunSelection{}, err
	}
	if err := validateUVRunRequest(requirement, command); err != nil {
		return UVRunSelection{}, err
	}
	native, err := r.resolveNativeSelection(ctx, requirement.Reference(), requirement.Requirement())
	if err != nil {
		return UVRunSelection{}, err
	}
	runtime := PinnedPythonRuntime()
	lock, err := artifactpypi.ParseUVLock(requirement.Reference(), native.lock, runtime.PythonVersion)
	if err != nil {
		return UVRunSelection{}, err
	}
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selected, err := lock.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return UVRunSelection{}, err
	}
	var frozen *pipxNativeInput
	if err := r.reader.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		var freezeErr error
		frozen, freezeErr = freezePipxNativeInput(ctx, lock, inputs, pages)
		return freezeErr
	}); err != nil {
		return UVRunSelection{}, err
	}
	if !pipxNativeRequirementMatches(requirement, frozen) {
		return UVRunSelection{}, errPipxNativeInput
	}
	if _, _, err := uvRunConsoleOriginal(frozen, command); err != nil {
		return UVRunSelection{}, err
	}
	var facts []uvRunCensusFact
	_, err = r.withUVNativeSessionForToolInstall(ctx, func(selection context.Context, session uvNativeSession) (uvNativeOutput, error) {
		args, err := uvRunPreparationArguments(requirement, command)
		if err != nil {
			return uvNativeOutput{}, err
		}
		output, runErr := r.reader.runner.Output(selection, "docker", boundaryExecArguments(session.container, boundaryLaunchMode, args...)...)
		if !uvRunPreparedTerminal(output, runErr) {
			return uvNativeOutput{}, errors.Join(errors.New("UV isolated preparation did not reach its fixed inert terminal"), runErr)
		}
		return readUVNativeTar(selection, r.reader.runner, session.container, pythonSitePath+"/.", func(input io.Reader) (uvNativeOutput, error) {
			var scanErr error
			facts, scanErr = scanUVRunCensus(input, command)
			return uvNativeOutput{}, scanErr
		})
	}, false, frozen)
	if err != nil {
		return UVRunSelection{}, err
	}
	return uvRunSelectionFromCensus(ctx, requirement, frozen, command, facts)
}

func uvRunSelectionFromCensus(ctx context.Context, requirement artifactpypi.UpgradeRequirement, frozen *pipxNativeInput, command string, facts []uvRunCensusFact) (UVRunSelection, error) {
	if ctx == nil {
		return UVRunSelection{}, errors.New("UV run selection requires context")
	}
	if err := ctx.Err(); err != nil {
		return UVRunSelection{}, errors.Join(err, context.Cause(ctx))
	}
	if err := validateUVRunRequest(requirement, command); err != nil {
		return UVRunSelection{}, err
	}
	if !pipxNativeRequirementMatches(requirement, frozen) {
		return UVRunSelection{}, errPipxNativeInput
	}
	entry, err := bindUVRunConsoleData(frozen, command, facts)
	if err != nil {
		return UVRunSelection{}, err
	}
	// Hash the complete observed census as unapproved DATA, including native
	// RECORD/scaffold. This does not authenticate it as original ownership.
	canonical := append([]uvRunCensusFact(nil), facts...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Name < canonical[j].Name })
	data, err := json.Marshal(canonical)
	if err != nil {
		return UVRunSelection{}, err
	}
	census := sha256.Sum256(data)
	descriptor := struct {
		Schema, Request, Original, Command, NativeUnapprovedDataSHA256 string
		Packages                                                       []artifactpypi.UVLockedPackage
		Console                                                        artifactpypi.WheelFile
	}{"helox.uv-isolated-run-selection/v1", requirement.Requirement(), frozen.id, entry.Command(), hex.EncodeToString(census[:]), frozen.lock.Packages(), entry.Metadata()}
	body, err := json.Marshal(descriptor)
	if err != nil {
		return UVRunSelection{}, err
	}
	digest := sha256.Sum256(body)
	measured, err := domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
	if err != nil {
		return UVRunSelection{}, err
	}
	resolution, err := domain.NewDependencyResolution(frozen.graph, "uv-tool-run-isolated:"+UVResolverRuntimeIdentity(), measured)
	if err != nil {
		return UVRunSelection{}, err
	}
	if err := ctx.Err(); err != nil {
		return UVRunSelection{}, errors.Join(err, context.Cause(ctx))
	}
	return UVRunSelection{resolution, entry}, nil
}
