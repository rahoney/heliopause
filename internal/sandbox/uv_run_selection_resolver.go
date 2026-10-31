package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// UVRunRootSelection is independently bound source/native DATA for explicitly
// requested roots. Current inspection and whole Policy approval remain required.
type UVRunRootSelection struct {
	resolution domain.ExecutionSelectionResolution
	entry      artifactpypi.ConsoleExecutionEntry
}

func (s UVRunRootSelection) Resolution() domain.ExecutionSelectionResolution { return s.resolution }
func (s UVRunRootSelection) Entry() artifactpypi.ConsoleExecutionEntry       { return s.entry }

func uvRunSelectionPreparationArguments(request artifactpypi.UVRunRequirements, command string) ([]string, error) {
	if !request.Valid() {
		return nil, errPipxNativeInput
	}
	args, err := uvRunPreparationArguments(request.Primary().Base(), command)
	if err != nil {
		return nil, err
	}
	args[5] = request.Primary().Requirement()
	// Every option value comes from the bounded immutable public request.
	// The only terminal stays /dev/null; no selected command or program argv.
	for _, supplemental := range request.Supplemental() {
		args = append(args[:len(args)-1], "--with", supplemental.Requirement(), "/dev/null")
	}
	return args, nil
}

func (r *UVResolver) ResolveToolRunWithRequirements(ctx context.Context, request artifactpypi.UVRunRequirements, command string) (result UVRunRootSelection, resultErr error) {
	if ctx == nil || r == nil || r.reader == nil || !request.Valid() {
		return result, errors.New("UV run selection requires a configured resolver, context and explicit roots")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			result = UVRunRootSelection{}
		}
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	args, err := uvRunSelectionPreparationArguments(request, command)
	if err != nil {
		return result, err
	}
	native, err := r.resolveNativeSelection(ctx, request.Primary().Reference(), strings.Join(request.Requirements(), "\n"))
	if err != nil {
		return result, err
	}
	runtime := PinnedPythonRuntime()
	lock, err := artifactpypi.ParseUVLock(request.Primary().Reference(), native.lock, runtime.PythonVersion)
	if err != nil {
		return result, err
	}
	target := artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}
	controls, err := lock.SelectInstalledWheelControls(native.controls, target)
	if err != nil {
		return result, err
	}
	var frozen *pipxNativeInput
	if err := r.reader.withUVArchiveInputs(ctx, controls, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		var freezeErr error
		frozen, freezeErr = freezeUVRunSelectionInput(ctx, lock, inputs, pages, request)
		return freezeErr
	}); err != nil {
		return result, err
	}
	if _, _, err := uvRunConsoleOriginal(frozen, command); err != nil {
		return result, err
	}
	facts, err := r.collectUVRunCensus(ctx, frozen, command, args)
	if err != nil {
		return result, err
	}
	return uvRunRootSelectionFromCensus(ctx, request, frozen, command, facts)
}

func uvRunRootSelectionFromCensus(ctx context.Context, request artifactpypi.UVRunRequirements, frozen *pipxNativeInput, command string, facts []uvRunCensusFact) (UVRunRootSelection, error) {
	if ctx == nil || !request.Valid() || frozen == nil || !frozen.valid() || !frozen.selection.Valid() {
		return UVRunRootSelection{}, errPipxNativeInput
	}
	if err := ctx.Err(); err != nil {
		return UVRunRootSelection{}, errors.Join(err, context.Cause(ctx))
	}
	if err := artifactpypi.ValidateConsoleExecutionCommand(command); err != nil {
		return UVRunRootSelection{}, err
	}
	if !slices.Equal(request.Requirements(), frozen.requests.Requirements()) {
		return UVRunRootSelection{}, errPipxNativeInput
	}
	entry, err := bindUVRunConsoleData(frozen, command, facts)
	if err != nil {
		return UVRunRootSelection{}, err
	}
	canonical := append([]uvRunCensusFact(nil), facts...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Name < canonical[j].Name })
	body, err := json.Marshal(canonical)
	if err != nil {
		return UVRunRootSelection{}, err
	}
	census := sha256.Sum256(body)
	descriptor := struct {
		Schema                                        string
		Requests                                      []string
		Original, Command, NativeUnapprovedDataSHA256 string
		Packages                                      []artifactpypi.UVLockedPackage
		Console                                       artifactpypi.WheelFile
	}{"helox.uv-isolated-run-root-selection/v1", request.Requirements(), frozen.id, entry.Command(), hex.EncodeToString(census[:]), frozen.lock.Packages(), entry.Metadata()}
	body, err = json.Marshal(descriptor)
	if err != nil {
		return UVRunRootSelection{}, err
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return UVRunRootSelection{}, err
	}
	resolution, err := domain.NewExecutionSelectionResolution(frozen.selection, "uv-tool-run-isolated:"+UVResolverRuntimeIdentity(), digest)
	if err != nil {
		return UVRunRootSelection{}, err
	}
	if err := ctx.Err(); err != nil {
		return UVRunRootSelection{}, errors.Join(err, context.Cause(ctx))
	}
	return UVRunRootSelection{resolution, entry}, nil
}
