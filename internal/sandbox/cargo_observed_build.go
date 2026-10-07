package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	projectoutput "github.com/rahoney/heliopause/internal/artifact/projectbuild"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type cargoBuildObserver interface {
	StartCargoBuildProfile(context.Context, string, bool, *observationTraceLedger) (TraceReader, error)
	AwaitMountAnchors(context.Context, string) error
}

type ObservedCargoBuilder struct {
	intake   string
	runner   CommandRunner
	observer cargoBuildObserver
	probe    func(context.Context) (CargoCapability, error)
}

func NewLinuxObservedCargoBuilder(intake string, executor TrustedExecutor, observer TraceObserver) (*ObservedCargoBuilder, error) {
	trusted, ok := observer.(cargoBuildObserver)
	if !ok || executor == nil || !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" {
		return nil, errors.New("cargo build requires bounded intake and trusted observation")
	}
	if _, ok := executor.(inputCommandRunner); !ok {
		return nil, errors.New("cargo build input transport is unavailable")
	}
	if _, ok := executor.(streamingDockerOutput); !ok {
		return nil, errors.New("cargo build output transport is unavailable")
	}
	return &ObservedCargoBuilder{intake, admissionAwareRunner(executor), trusted, func(ctx context.Context) (CargoCapability, error) { return ProbeCargo(ctx, executor) }}, nil
}

type isolatedCargoBuildFailure struct {
	cause         error
	phase         string
	diagnostic    TraceDiagnostic
	commandReason string
	outputReason  string
}

func (e *isolatedCargoBuildFailure) Error() string {
	return "observed cargo build failed: phase=" + e.phase + " command_status=" + e.commandReason + " " + e.diagnostic.String() + " output_diagnostic=" + e.outputReason
}
func (e *isolatedCargoBuildFailure) Unwrap() error { return e.cause }

type cargoBuildOperation struct {
	builder       *ObservedCargoBuilder
	input, target closureVolume
	inputs        *goBuildPreparedInputs
	budget        *observationTraceLedger
	runtimes      []*goBuildRuntime
	phase         string
	observations  []domain.SandboxObservation
	diagnostic    TraceDiagnostic
	commandReason string
	outputReason  string
	output        *projectoutput.CapturedBuildOutput
}

func (b *ObservedCargoBuilder) Build(ctx context.Context, inputs domain.ProjectBuildInputs) (observation domain.ProjectBuildObservation, resultErr error) {
	if b == nil || ctx == nil || !inputs.Valid() || inputs.Kind() != "cargo" || inputs.Selector() != "default" {
		return observation, errors.New("cargo build request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return observation, err
	}
	ctx, cancel := context.WithTimeout(ctx, cargoBuildTimeout)
	defer cancel()
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return observation, err
	}
	op := &cargoBuildOperation{builder: b, phase: "INPUT"}
	var output domain.AcquiredArtifact
	var binding domain.DerivationBinding
	defer func() {
		primary := resultErr
		resultErr = errors.Join(primary, op.dispose())
		if op.output != nil {
			if resultErr == nil {
				output = op.output.Artifact()
			}
			resultErr = errors.Join(resultErr, op.output.Close(resultErr == nil))
		}
		status, limitation := domain.SandboxCompleted, ""
		if resultErr != nil {
			status, limitation = domain.SandboxIncomplete, "M12_CARGO_BUILD_FAILED"
			output, binding = domain.AcquiredArtifact{}, domain.DerivationBinding{}
			if primary == nil {
				op.phase = "CLEANUP"
			}
			resultErr = &isolatedCargoBuildFailure{resultErr, op.phase, op.diagnostic, op.commandReason, op.outputReason}
		} else {
			completed, _ := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
			op.observations = append(op.observations, completed)
		}
		result, resultError := domain.NewSandboxResult(session, status, limitation, op.observations)
		observation, err = domain.NewProjectBuildObservation(inputs, result, output, binding)
		resultErr = errors.Join(resultErr, resultError, err)
	}()
	op.inputs, err = prepareCargoBuildInputs(ctx, b.intake, inputs)
	if err != nil {
		return observation, err
	}
	op.phase = "RUNTIME"
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedCargoRuntime() {
		return observation, errors.New("cargo build pinned runtime unavailable")
	}
	op.budget, err = newObservationTraceLedger(cargoBuildProfile)
	if err != nil {
		return observation, err
	}
	op.phase = "VOLUME"
	op.input, err = createInputVolume(ctx, b.runner, session.String(), op.inputs.manifest.identity, goBuildInputCapacity, false, projectBuildVolumeConfig{cargoBuildGuestInput, false})
	if err != nil {
		return observation, err
	}
	op.target, err = createInputVolume(ctx, b.runner, session.String()+"-target", op.inputs.manifest.identity, projectoutput.MaxBuildOutputBytes, false, projectBuildVolumeConfig{cargoBuildGuestTarget, true})
	if err != nil {
		return observation, err
	}
	op.phase = "PREPARATION"
	preparation, err := op.start(ctx, false)
	if err != nil {
		return observation, err
	}
	if err := op.inputs.introduceAt(ctx, b.runner, preparation.id, cargoBuildGuestInput); err != nil {
		return observation, err
	}
	if err := verifyProjectBuildMaterialization(ctx, b.runner, preparation.id, cargoBuildGuestInput, op.inputs.manifest); err != nil {
		return observation, err
	}
	op.phase = "READ_ONLY"
	build, err := op.start(ctx, true)
	if err != nil {
		return observation, err
	}
	if err := op.input.verifyAttachments(ctx, b.runner, map[string]bool{preparation.id: false, build.id: true}); err != nil {
		return observation, err
	}
	if err := op.target.verifyAttachments(ctx, b.runner, map[string]bool{preparation.id: false, build.id: false}); err != nil {
		return observation, err
	}
	if err := op.remove(preparation); err != nil {
		return observation, err
	}
	if err := op.verify(ctx, build); err != nil {
		return observation, err
	}
	op.phase = "OFFLINE_GRAPH"
	if err := op.checkGraph(ctx, build, inputs.Snapshot()); err != nil {
		return observation, err
	}
	op.phase = "BUILD"
	if _, err := op.artifactOutput(ctx, build, cargoBuildCompilerArguments()...); err != nil {
		return observation, err
	}
	op.phase = "POST_BUILD"
	if err := op.verify(ctx, build); err != nil {
		return observation, err
	}
	if err := op.checkGraph(ctx, build, inputs.Snapshot()); err != nil {
		return observation, err
	}
	op.phase = "OUTPUT_CAPTURE"
	op.output, err = op.captureOutput(ctx, build, inputs)
	if err != nil {
		return observation, err
	}
	if err := op.verify(ctx, build); err != nil {
		return observation, err
	}
	digest, err := cargoBuildRecipeDigest()
	if err != nil {
		return observation, err
	}
	binding, err = domain.NewDerivationBinding(inputs.Source().Digest(), []domain.ContentDigest{inputs.Cache().Digest(), inputs.Snapshot().GraphDigest()}, "cargo-build-linux-amd64", digest)
	return observation, err
}

// Per-operation volume IDs are opaque transport references. The recipe binds
// their two exact roles; actual IDs/attachment identities are independently
// checked before commands, after the build and during cleanup.
func cargoBuildRecipeDigest() (domain.ContentDigest, error) {
	inputMount := "type=volume,source=verified-cargo-input,target=" + cargoBuildGuestInput + ",volume-nocopy"
	targetMount := "type=volume,source=private-cargo-target,target=" + cargoBuildGuestTarget + ",volume-nocopy"
	preparation, _ := cargoBuildExpectedTopology(false)
	build, _ := cargoBuildExpectedTopology(true)
	runtime := PinnedCargoRuntime()
	recipe, err := json.Marshal(struct {
		Schema                                                                       string
		Runtime                                                                      CargoRuntime
		Configuration                                                                string
		MetadataCommand, BuildCommand                                                []string
		PreparationArguments, BuildArguments                                         []string
		PreparationTopology, BuildTopology                                           []observerMountExpectation
		InputCapacity, TargetCapacity, ArchiveExpandedBytes, FileBytes, ArchiveBytes int64
		ArchiveFiles, ArchiveEntries, OutputFiles                                    int
		OutputSelection, AliasHandling, Profile                                      string
		TimeoutNanoseconds                                                           int64
	}{"cargo-build-v3", runtime, cargoBuildSourceConfiguration,
		append([]string{runtime.CargoBinary()}, cargoBuildMetadataArguments()...), append([]string{runtime.CargoBinary()}, cargoBuildCompilerArguments()...),
		cargoBuildCreateArgumentsForMounts(inputMount, targetMount), cargoBuildCreateArgumentsForMounts(inputMount+",readonly", targetMount), preparation, build,
		goBuildInputCapacity, projectoutput.MaxBuildOutputBytes, artifactcargo.MaxCrateExpandedBytes, artifactcargo.MaxCrateFileBytes, projectoutput.MaxBuildOutputArchiveBytes,
		artifactcargo.MaxCrateFiles, 2 * artifactcargo.MaxCrateFiles, projectoutput.MaxBuildOutputFiles,
		"top-level nonhidden default products excluding .d", "same-archive regular backing data only", cargoBuildProfile, int64(cargoBuildTimeout)})
	if err != nil {
		return domain.ContentDigest{}, err
	}
	hash := sha256.Sum256(recipe)
	return domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
}

func (o *cargoBuildOperation) start(ctx context.Context, readOnly bool) (*goBuildRuntime, error) {
	args, err := cargoBuildCreateArguments(o.input, o.target, readOnly)
	if err != nil {
		return nil, err
	}
	created, err := o.builder.runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		return nil, errors.New("cargo build runtime creation failed")
	}
	runtime := &goBuildRuntime{id: id}
	o.runtimes = append(o.runtimes, runtime)
	if err := o.input.verifyContainerMount(ctx, o.builder.runner, id, readOnly); err != nil {
		return nil, err
	}
	if err := o.target.verifyContainerMount(ctx, o.builder.runner, id, false); err != nil {
		return nil, err
	}
	runtime.trace, err = o.builder.observer.StartCargoBuildProfile(ctx, id, readOnly, o.budget)
	if err != nil {
		return nil, err
	}
	if _, err := o.builder.runner.Output(ctx, "docker", "start", id); err != nil {
		return nil, errors.New("cargo build runtime start failed")
	}
	if awaitBoundaryHelper(ctx, o.builder.runner, id) != nil || o.builder.observer.AwaitMountAnchors(ctx, id) != nil || verifyObservationRuntimeAlive(ctx, o.builder.runner, id) != nil {
		return nil, errors.New("cargo build runtime boundary is incomplete")
	}
	return runtime, nil
}

func (o *cargoBuildOperation) verify(ctx context.Context, runtime *goBuildRuntime) error {
	if runtime == nil || verifyObservationRuntimeAlive(ctx, o.builder.runner, runtime.id) != nil {
		return errors.New("cargo build readonly runtime is unavailable")
	}
	if err := o.input.verifyAttachments(ctx, o.builder.runner, map[string]bool{runtime.id: true}); err != nil {
		return err
	}
	if err := o.target.verifyAttachments(ctx, o.builder.runner, map[string]bool{runtime.id: false}); err != nil {
		return err
	}
	return verifyProjectBuildMaterialization(ctx, o.builder.runner, runtime.id, cargoBuildGuestInput, o.inputs.manifest)
}

func (o *cargoBuildOperation) remove(runtime *goBuildRuntime) error {
	if runtime == nil || runtime.collected {
		return nil
	}
	ctx, cancel := resolverCleanupContext()
	_, removeErr := o.builder.runner.Output(ctx, "docker", "rm", "--force", runtime.id)
	cancel()
	var failure error
	if removeErr != nil {
		failure = errors.New("cargo build runtime cleanup failed")
	}
	if runtime.trace != nil {
		ctx, cancel := resolverCleanupContext()
		facts, limitation, diagnostic := collectTraceDiagnostic(ctx, runtime.trace)
		cancel()
		o.observations = append(o.observations, facts...)
		if o.diagnostic.Reason == "" {
			o.diagnostic = diagnostic
		}
		if limitation != "" {
			failure = errors.Join(failure, errors.New("cargo build observation is incomplete"))
		}
	}
	if removeErr == nil {
		runtime.collected = true
	}
	return failure
}

func (o *cargoBuildOperation) dispose() error {
	var failure error
	for i := len(o.runtimes) - 1; i >= 0; i-- {
		failure = errors.Join(failure, o.remove(o.runtimes[i]))
	}
	for _, volume := range []*closureVolume{&o.target, &o.input} {
		if volume.name != "" {
			ctx, cancel := resolverCleanupContext()
			if volume.verify(ctx, o.builder.runner) != nil {
				failure = errors.Join(failure, errors.New("cargo build volume cleanup identity unavailable"))
			} else if _, err := o.builder.runner.Output(ctx, "docker", "volume", "rm", volume.name); err != nil {
				failure = errors.Join(failure, errors.New("cargo build volume cleanup failed"))
			}
			cancel()
		}
	}
	if o.inputs != nil {
		failure = errors.Join(failure, o.inputs.close())
	}
	return failure
}

func (o *cargoBuildOperation) artifactOutput(ctx context.Context, runtime *goBuildRuntime, args ...string) ([]byte, error) {
	w := &goResolverBoundedOutput{limit: goResolverOutputLimit}
	command := append([]string{PinnedCargoRuntime().CargoBinary()}, args...)
	if err := o.builder.runner.(streamingDockerOutput).RunOutput(ctx, w, "docker", boundaryExecArguments(runtime.id, boundaryELFHandoffMode, command...)...); err != nil {
		reason := pythonCommandErrorReason(err)
		if o.commandReason == "" {
			o.commandReason = reason
		}
		return nil, fmt.Errorf("offline cargo command failed: %s", reason)
	}
	if w.exceeded {
		return nil, errors.New("offline cargo output exceeded bound")
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func (o *cargoBuildOperation) checkGraph(ctx context.Context, runtime *goBuildRuntime, expected domain.ProjectDependencySnapshot) error {
	metadata, err := o.artifactOutput(ctx, runtime, cargoBuildMetadataArguments()...)
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	// Complete source bytes are authenticated by the private reader. This
	// metadata consumer needs selected local manifest bytes, not new host reads.
	if err := readCargoBuildSourceFiles(ctx, o.inputs.source, files); err != nil {
		return err
	}
	snapshot, err := artifactcargo.BuildProjectSnapshot(expected.Context(), metadata, files["Cargo.toml"], files["Cargo.lock"], cargoBuildGuestProject, PinnedCargoRuntime().ImageReference, files)
	if err != nil || !reflect.DeepEqual(snapshot, expected) {
		return errors.New("offline cargo graph differs from independent retained approval")
	}
	return nil
}

func (o *cargoBuildOperation) captureOutput(ctx context.Context, runtime *goBuildRuntime, inputs domain.ProjectBuildInputs) (*projectoutput.CapturedBuildOutput, error) {
	reader, writer := io.Pipe()
	command := make(chan error, 1)
	go func() {
		err := o.builder.runner.(streamingDockerOutput).RunOutput(ctx, writer, "docker", "cp", runtime.id+":"+cargoBuildGuestTarget+"/"+PinnedCargoRuntime().Target+"/debug/.", "-")
		_ = writer.CloseWithError(err)
		command <- err
	}()
	filtered, filterWriter := io.Pipe()
	filter := make(chan error, 1)
	go func() {
		err := filterCargoBuildOutputs(ctx, reader, filterWriter)
		_ = filterWriter.CloseWithError(err)
		_ = reader.CloseWithError(err)
		filter <- err
	}()
	output, err := projectoutput.CaptureBuildOutput(ctx, o.builder.intake, inputs, filtered)
	_ = filtered.CloseWithError(err)
	filterErr, commandErr := <-filter, <-command
	if errors.Join(err, filterErr, commandErr, ctx.Err()) != nil {
		var boundary *cargoOutputArchiveFailure
		if errors.As(filterErr, &boundary) {
			o.outputReason = boundary.Error()
		} else if filterErr != nil {
			o.outputReason = "FILTER_FAILED"
		} else if commandErr != nil {
			o.outputReason = "TRANSPORT_" + pythonCommandErrorReason(commandErr)
		} else {
			o.outputReason = "CAPTURE_FAILED"
		}
		if output != nil {
			_ = output.Close(false)
		}
		return nil, errors.Join(errors.New("cargo build output capture failed"), filterErr, commandErr, ctx.Err())
	}
	return output, nil
}
