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
func (r *UVResolver) ResolveToolRun(ctx context.Context, requirement artifactpypi.UpgradeRequirement, command string) (UVRunSelection, error) {
	return r.resolveToolRun(ctx, requirement, artifactpypi.UVRunRequirement{}, command)
}

// ResolveToolRunWithExtras selects the same connected primary graph with its
// explicitly requested root extras. It does not approve native selection DATA.
func (r *UVResolver) ResolveToolRunWithExtras(ctx context.Context, request artifactpypi.UVRunRequirement, command string) (UVRunSelection, error) {
	if !request.Valid() || len(request.Extras()) == 0 {
		return UVRunSelection{}, errors.New("UV extra selection requires a canonical root-extra request")
	}
	return r.resolveToolRun(ctx, request.Base(), request, command)
}

func (r *UVResolver) resolveToolRun(ctx context.Context, requirement artifactpypi.UpgradeRequirement, request artifactpypi.UVRunRequirement, command string) (result UVRunSelection, resultErr error) {
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
	var frozen *pipxNativeInput
	var err error
	if len(request.Extras()) != 0 {
		frozen, err = r.preparePythonRunOriginalWithExtras(ctx, requirement, request)
		if err == nil {
			_, _, err = uvRunConsoleOriginal(frozen, command)
		}
	} else {
		frozen, err = r.prepareUVRunOriginal(ctx, requirement, command)
	}
	if err != nil {
		return UVRunSelection{}, err
	}
	return r.resolveFrozenUVToolRun(ctx, requirement, frozen, command)
}

// The validated caller retains the context/error boundary. Frozen originals
// own their bytes independently of the native sessions and archive readers.
func (r *UVResolver) prepareUVRunOriginal(ctx context.Context, requirement artifactpypi.UpgradeRequirement, command string) (*pipxNativeInput, error) {
	frozen, err := r.preparePythonRunOriginal(ctx, requirement)
	if err != nil {
		return nil, err
	}
	if _, _, err := uvRunConsoleOriginal(frozen, command); err != nil {
		return nil, err
	}
	return frozen, nil
}

func (r *UVResolver) preparePythonRunOriginal(ctx context.Context, requirement artifactpypi.UpgradeRequirement) (*pipxNativeInput, error) {
	return r.preparePythonRunOriginalWithExtras(ctx, requirement, artifactpypi.UVRunRequirement{})
}

func (r *UVResolver) preparePythonRunOriginalWithExtras(ctx context.Context, requirement artifactpypi.UpgradeRequirement, request artifactpypi.UVRunRequirement) (*pipxNativeInput, error) {
	reference, text := requirement.Reference(), requirement.Requirement()
	if len(request.Extras()) != 0 {
		if !request.Valid() || request.Base().Requirement() != requirement.Requirement() {
			return nil, errPipxNativeInput
		}
		reference, text = request.Reference(), request.Requirement()
	}
	native, err := r.resolveNativeSelection(ctx, reference, text)
	if err != nil {
		return nil, err
	}
	runtime := PinnedPythonRuntime()
	lock, err := artifactpypi.ParseUVLock(reference, native.lock, runtime.PythonVersion)
	if err != nil {
		return nil, err
	}
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	selected, err := lock.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return nil, err
	}
	var frozen *pipxNativeInput
	if err := r.reader.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		var freezeErr error
		frozen, freezeErr = freezePythonRunNativeInput(ctx, lock, inputs, pages, request)
		return freezeErr
	}); err != nil {
		return nil, err
	}
	if !pipxNativeRequirementMatches(requirement, frozen) {
		return nil, errPipxNativeInput
	}
	return frozen, nil
}

func (r *UVResolver) resolveFrozenUVToolRun(ctx context.Context, requirement artifactpypi.UpgradeRequirement, frozen *pipxNativeInput, command string) (UVRunSelection, error) {
	args, err := uvRunPreparationArguments(requirement, command)
	if err != nil {
		return UVRunSelection{}, err
	}
	if frozen.uvRunRequest != "" {
		request, err := artifactpypi.ParseUVRunRequirement(frozen.uvRunRequest)
		if err != nil || !frozen.valid() || request.Base().Requirement() != requirement.Requirement() {
			return UVRunSelection{}, errPipxNativeInput
		}
		args[5] = request.Requirement()
	}
	facts, err := r.collectUVRunCensus(ctx, frozen, command, args)
	if err != nil {
		return UVRunSelection{}, err
	}
	return uvRunSelectionFromCensus(ctx, requirement, frozen, command, facts)
}

func (r *UVResolver) collectUVRunCensus(ctx context.Context, frozen *pipxNativeInput, command string, args []string) ([]uvRunCensusFact, error) {
	var facts []uvRunCensusFact
	_, err := r.withUVNativeSessionForToolInstall(ctx, func(selection context.Context, session uvNativeSession) (uvNativeOutput, error) {
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
	return facts, err
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
	resolution, err := pythonRunSelectionResolution(ctx, requirement, frozen, entry, facts, false)
	if err != nil {
		return UVRunSelection{}, err
	}
	return UVRunSelection{resolution, entry}, nil
}

func pythonRunSelectionResolution(ctx context.Context, requirement artifactpypi.UpgradeRequirement, frozen *pipxNativeInput, entry artifactpypi.ConsoleExecutionEntry, facts []uvRunCensusFact, pipx bool) (domain.DependencyResolution, error) {
	// Hash the complete observed census as unapproved DATA, including native
	// RECORD/scaffold. This does not authenticate it as original ownership.
	canonical := append([]uvRunCensusFact(nil), facts...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Name < canonical[j].Name })
	data, err := json.Marshal(canonical)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	schema, runtime := "helox.uv-isolated-run-selection/v1", "uv-tool-run-isolated:"+UVResolverRuntimeIdentity()
	if pipx {
		schema, runtime = "helox.pipx-isolated-run-selection/v1", "pipx-run-isolated:"+PipxResolverRuntimeIdentity()
	}
	requestText := requirement.Requirement()
	if frozen.uvRunRequest != "" {
		request, err := artifactpypi.ParseUVRunRequirement(frozen.uvRunRequest)
		if err != nil || pipx || request.Base().Requirement() != requestText || !frozen.valid() {
			return domain.DependencyResolution{}, errPipxNativeInput
		}
		schema, requestText = "helox.uv-isolated-run-selection/v2", request.Requirement()
	}
	census := sha256.Sum256(data)
	descriptor := struct {
		Schema, Request, Original, Command, NativeUnapprovedDataSHA256 string
		Packages                                                       []artifactpypi.UVLockedPackage
		Console                                                        artifactpypi.WheelFile
	}{schema, requestText, frozen.id, entry.Command(), hex.EncodeToString(census[:]), frozen.lock.Packages(), entry.Metadata()}
	body, err := json.Marshal(descriptor)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digest := sha256.Sum256(body)
	measured, err := domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	resolution, err := domain.NewDependencyResolution(frozen.graph, runtime, measured)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, errors.Join(err, context.Cause(ctx))
	}
	return resolution, nil
}
