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
	"time"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type goBuildInputObserver interface {
	StartGoBuildInputProfile(context.Context, string, bool, *observationTraceLedger) (TraceReader, error)
	AwaitMountAnchors(context.Context, string) error
}

type goBuildOutputObserver interface {
	StartGoBuildOutputProfile(context.Context, string, bool, *observationTraceLedger) (TraceReader, error)
}

// ObservedGoBuilder consumes private, frozen source/cache data. It owns the
// disposable offline runtime; it cannot approve or publish project outputs.
type ObservedGoBuilder struct {
	intake   string
	runner   CommandRunner
	observer goBuildInputObserver
	probe    func(context.Context) (GoCapability, error)
}

type isolatedGoBuildFailure struct {
	cause         error
	phase         string
	diagnostic    TraceDiagnostic
	commandReason string
}

func (e *isolatedGoBuildFailure) Error() string {
	status := ""
	if e.commandReason != "" {
		status = " command_status=" + e.commandReason
	}
	return "observed go build failed: phase=" + e.phase + status + " " + e.diagnostic.String()
}
func (e *isolatedGoBuildFailure) Unwrap() error { return e.cause }

func NewLinuxObservedGoBuilder(intake string, executor TrustedExecutor, observer TraceObserver) (*ObservedGoBuilder, error) {
	trusted, ok := observer.(goBuildInputObserver)
	if !ok || executor == nil {
		return nil, errors.New("go build requires complete observation and trusted executor")
	}
	return newObservedGoBuilder(intake, executor, trusted, func(ctx context.Context) (GoCapability, error) { return ProbeGo(ctx, executor) })
}

func newObservedGoBuilder(intake string, runner CommandRunner, observer goBuildInputObserver, probe func(context.Context) (GoCapability, error)) (*ObservedGoBuilder, error) {
	if !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" || runner == nil || observer == nil || probe == nil {
		return nil, errGoBuildInput
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errGoBuildInput
	}
	if _, ok := runner.(streamingDockerOutput); !ok {
		return nil, errGoBuildInput
	}
	return &ObservedGoBuilder{intake, admissionAwareRunner(runner), observer, probe}, nil
}

type goBuildRuntime struct {
	id        string
	trace     TraceReader
	collected bool
}

type goBuildOperation struct {
	builder         *ObservedGoBuilder
	volume          closureVolume
	outputVolume    closureVolume
	inputs          *goBuildPreparedInputs
	budget          *observationTraceLedger
	runtimes        []*goBuildRuntime
	phase           string
	observations    []domain.SandboxObservation
	firstDiagnostic TraceDiagnostic
	commandReason   string
	output          *artifactgo.CapturedBuildOutput
}

// ObserveBuild covers materialization, exact offline graph reconciliation and
// compilation. The output remains disposable scratch; this facet provides no
// publication authority and does not execute the resulting program.
func (b *ObservedGoBuilder) ObserveBuild(ctx context.Context, snapshot domain.ProjectDependencySnapshot, source, cache domain.AcquiredArtifact, selector string) (domain.SandboxResult, error) {
	result, _, err := b.observe(ctx, snapshot, source, cache, selector, nil)
	return result, err
}

// Build retains private output only after complete runtime/observer cleanup.
// A structural input is not retained approval; the Application guard supplies it.
func (b *ObservedGoBuilder) Build(ctx context.Context, inputs domain.ProjectBuildInputs) (domain.ProjectBuildObservation, error) {
	if !inputs.Valid() {
		return domain.ProjectBuildObservation{}, errGoBuildInput
	}
	result, output, err := b.observe(ctx, inputs.Snapshot(), inputs.Source(), inputs.Cache(), inputs.Selector(), &inputs)
	var binding domain.DerivationBinding
	if err == nil {
		recipe, marshalErr := json.Marshal(struct {
			Schema                              string
			Runtime                             GoRuntime
			CreateArguments                     []string
			ValidationArguments, RoleArguments  []string
			BuildArguments                      []string
			PreparationTopology, BuildTopology  []observerMountExpectation
			InputCapacity, OutputCapacity       int64
			OutputFiles                         int
			OutputFileBytes, OutputArchiveBytes int64
			Profile                             string
		}{"go-build-v2", PinnedGoRuntime(), goBuildCreateArgumentsForCache(goBuildGuestCache), goBuildValidationArguments(inputs.Selector()), goBuildRoleArguments(inputs.Selector()), goBuildCompilerArguments(inputs.Selector()),
			goBuildRecipeTopology(false), goBuildRecipeTopology(true), goBuildInputCapacity, goBuildOutputCapacity,
			artifactgo.MaxBuildOutputFiles, artifactgo.MaxBuildOutputFileBytes, artifactgo.MaxBuildOutputArchiveBytes, goBuildProfile})
		if marshalErr != nil {
			return domain.ProjectBuildObservation{}, errGoBuildInput
		}
		hash := sha256.Sum256(recipe)
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
		binding, err = domain.NewDerivationBinding(inputs.Source().Digest(), []domain.ContentDigest{inputs.Cache().Digest(), inputs.Snapshot().GraphDigest()}, "go-build-linux-amd64", digest)
	}
	observation, bindErr := domain.NewProjectBuildObservation(inputs, result, output, binding)
	return observation, errors.Join(err, bindErr)
}

func goBuildRecipeTopology(readOnly bool) []observerMountExpectation {
	topology, _ := goBuildOutputExpectedTopology(readOnly)
	return topology
}

func goBuildCompilerArguments(selector string) []string {
	arguments := []string{"build", "-mod=readonly", "-o", goBuildGuestOutput + "/"}
	if selector != "" {
		arguments = append(arguments, selector)
	}
	return arguments
}

// The pinned Go directory -o mode builds only main packages. Its official
// null-output mode compiles every selected package without writing into the
// read-only project. No command failure is converted into successful absence.
func goBuildValidationArguments(selector string) []string {
	arguments := []string{"build", "-mod=readonly", "-o", "/dev/null"}
	if selector != "" {
		arguments = append(arguments, selector)
	}
	return arguments
}

func goBuildRoleArguments(selector string) []string {
	arguments := []string{"list", "-mod=readonly", "-f", `{{if eq .Name "main"}}main{{else}}library{{end}}`}
	if selector != "" {
		arguments = append(arguments, selector)
	}
	return arguments
}

// These bounded role values select only fixed output commands, never process
// privileges, approval, executable paths or observation requiredness. The
// complete selected compile is required independently of this query.
func goBuildHasMain(roles []byte) (bool, error) {
	if len(roles) == 0 || len(roles) > 80000 || roles[len(roles)-1] != '\n' {
		return false, errors.New("go build package roles are incomplete")
	}
	lines := strings.Split(string(roles[:len(roles)-1]), "\n")
	if len(lines) > 10000 {
		return false, errors.New("go build package roles exceed bound")
	}
	main := false
	for _, role := range lines {
		switch role {
		case "main":
			main = true
		case "library":
		default:
			return false, errors.New("go build package roles are invalid")
		}
	}
	return main, nil
}

func (b *ObservedGoBuilder) observe(ctx context.Context, snapshot domain.ProjectDependencySnapshot, source, cache domain.AcquiredArtifact, selector string, capture *domain.ProjectBuildInputs) (result domain.SandboxResult, output domain.AcquiredArtifact, resultErr error) {
	if b == nil || ctx == nil || !snapshot.Valid() || artifactgo.ValidateBuildPackage(selector) != nil {
		return result, output, errGoBuildInput
	}
	if err := ctx.Err(); err != nil {
		return result, output, err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return result, output, err
	}
	operation := &goBuildOperation{builder: b, phase: "INPUT"}
	defer func() {
		primary := resultErr
		cleanup := operation.dispose()
		resultErr = errors.Join(primary, cleanup)
		if operation.output != nil {
			if resultErr == nil {
				output = operation.output.Artifact()
			}
			resultErr = errors.Join(resultErr, operation.output.Close(resultErr == nil))
			if resultErr != nil {
				output = domain.AcquiredArtifact{}
			}
		}
		status, limitation := domain.SandboxCompleted, ""
		if resultErr != nil {
			status, limitation = domain.SandboxIncomplete, "M12_GO_BUILD_FAILED"
			if primary == nil {
				operation.phase = "CLEANUP"
			}
			resultErr = &isolatedGoBuildFailure{cause: resultErr, phase: operation.phase, diagnostic: operation.firstDiagnostic, commandReason: operation.commandReason}
		} else {
			completed, _ := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
			operation.observations = append(operation.observations, completed)
		}
		var resultError error
		result, resultError = domain.NewSandboxResult(session, status, limitation, operation.observations)
		resultErr = errors.Join(resultErr, resultError)
	}()
	operation.inputs, err = prepareGoBuildInputs(ctx, b.intake, snapshot, source, cache)
	if err != nil {
		return result, output, err
	}
	operation.phase = "RUNTIME"
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedGoRuntime() {
		return result, output, errors.New("go build pinned runtime unavailable")
	}
	operation.budget, err = newObservationTraceLedger(goBuildProfile)
	if err != nil {
		return result, output, err
	}
	operation.phase = "VOLUME"
	operation.volume, err = createGoBuildInputVolume(ctx, b.runner, session.String(), operation.inputs.manifest.identity)
	if err != nil {
		return result, output, err
	}
	if capture != nil {
		if _, ok := b.observer.(goBuildOutputObserver); !ok {
			return result, output, errors.New("go build output topology observer unavailable")
		}
		operation.outputVolume, err = createGoBuildOutputVolume(ctx, b.runner, session.String(), operation.inputs.manifest.identity)
		if err != nil {
			return result, output, err
		}
	}
	operation.phase = "PREPARATION"
	preparation, err := operation.start(ctx, false)
	if err != nil {
		return result, output, err
	}
	if err := operation.inputs.introduce(ctx, b.runner, preparation.id); err != nil {
		return result, output, err
	}
	if err := verifyGoBuildMaterialization(ctx, b.runner, preparation.id, operation.inputs.manifest); err != nil {
		return result, output, err
	}
	operation.phase = "READ_ONLY"
	build, err := operation.start(ctx, true)
	if err != nil {
		return result, output, err
	}
	if err := operation.volume.verifyAttachments(ctx, b.runner, map[string]bool{preparation.id: false, build.id: true}); err != nil {
		return result, output, err
	}
	if operation.outputVolume.name != "" {
		if err := operation.outputVolume.verifyAttachments(ctx, b.runner, map[string]bool{preparation.id: false, build.id: false}); err != nil {
			return result, output, err
		}
	}
	if err := operation.remove(preparation); err != nil {
		return result, output, err
	}
	if err := operation.verify(ctx, build); err != nil {
		return result, output, err
	}
	operation.phase = "CONFIG"
	input := b.runner.(inputCommandRunner)
	if err := input.RunInput(ctx, strings.NewReader("off\n"), "docker", boundaryInputExecArguments(build.id, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; mkdir -p /tmp/.config/go/telemetry /tmp/haa-go-output; cat > /tmp/.config/go/telemetry/mode")...); err != nil {
		return result, output, errors.New("go build configuration failed")
	}
	operation.phase = "OFFLINE_GRAPH"
	if err := operation.checkGraph(ctx, build, snapshot); err != nil {
		return result, output, err
	}
	operation.phase = "BUILD"
	runner := &goBuildArtifactRunner{operation: operation, runtime: build}
	wrapper, err := NewGoBuildRunner(runner)
	if err != nil {
		return result, output, err
	}
	if err := wrapper.Build(ctx, goBuildGuestProject, goBuildGuestCache, selector); err != nil {
		return result, output, err
	}
	operation.phase = "POST_BUILD"
	if err := operation.verify(ctx, build); err != nil {
		return result, output, err
	}
	if err := operation.checkGraph(ctx, build, snapshot); err != nil {
		return result, output, err
	}
	if capture != nil {
		operation.phase = "OUTPUT_CAPTURE"
		operation.output, err = operation.captureOutput(ctx, build, *capture)
		if err != nil {
			return result, output, err
		}
		if err := operation.verify(ctx, build); err != nil {
			return result, output, err
		}
	}
	return result, output, nil
}

func (o *goBuildOperation) captureOutput(ctx context.Context, runtime *goBuildRuntime, inputs domain.ProjectBuildInputs) (*artifactgo.CapturedBuildOutput, error) {
	reader, writer := io.Pipe()
	command := make(chan error, 1)
	go func() {
		err := o.builder.runner.(streamingDockerOutput).RunOutput(ctx, writer, "docker", "cp", runtime.id+":"+goBuildGuestOutput+"/.", "-")
		_ = writer.CloseWithError(err)
		command <- err
	}()
	output, err := artifactgo.CaptureBuildOutput(ctx, o.builder.intake, inputs, reader)
	_ = reader.CloseWithError(err)
	commandErr := <-command
	if err != nil || commandErr != nil || ctx.Err() != nil {
		if commandErr != nil && o.commandReason == "" {
			o.commandReason = pythonCommandErrorReason(commandErr)
		}
		if output != nil {
			_ = output.Close(false)
		}
		return nil, errors.Join(errors.New("go build output capture failed"), ctx.Err())
	}
	return output, nil
}

func (o *goBuildOperation) start(ctx context.Context, readOnly bool) (*goBuildRuntime, error) {
	var outputVolume *closureVolume
	if o.outputVolume.name != "" {
		outputVolume = &o.outputVolume
	}
	args, err := goBuildVolumesCreateArguments(o.volume, outputVolume, readOnly)
	if err != nil {
		return nil, err
	}
	created, err := o.builder.runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		return nil, errors.New("go build runtime creation failed")
	}
	runtime := &goBuildRuntime{id: id}
	o.runtimes = append(o.runtimes, runtime)
	if err := o.volume.verifyContainerMount(ctx, o.builder.runner, id, readOnly); err != nil {
		return nil, err
	}
	if outputVolume != nil {
		if err := outputVolume.verifyContainerMount(ctx, o.builder.runner, id, false); err != nil {
			return nil, err
		}
		runtime.trace, err = o.builder.observer.(goBuildOutputObserver).StartGoBuildOutputProfile(ctx, id, readOnly, o.budget)
	} else {
		runtime.trace, err = o.builder.observer.StartGoBuildInputProfile(ctx, id, readOnly, o.budget)
	}
	if err != nil {
		return nil, err
	}
	if _, err := o.builder.runner.Output(ctx, "docker", "start", id); err != nil {
		return nil, errors.New("go build runtime start failed")
	}
	if awaitBoundaryHelper(ctx, o.builder.runner, id) != nil || o.builder.observer.AwaitMountAnchors(ctx, id) != nil || verifyObservationRuntimeAlive(ctx, o.builder.runner, id) != nil {
		return nil, errors.New("go build runtime boundary is incomplete")
	}
	return runtime, nil
}

func (o *goBuildOperation) verify(ctx context.Context, build *goBuildRuntime) error {
	if build == nil || verifyObservationRuntimeAlive(ctx, o.builder.runner, build.id) != nil {
		return errors.New("go build readonly runtime is unavailable")
	}
	if err := o.volume.verifyAttachments(ctx, o.builder.runner, map[string]bool{build.id: true}); err != nil {
		return err
	}
	if o.outputVolume.name != "" {
		if err := o.outputVolume.verifyAttachments(ctx, o.builder.runner, map[string]bool{build.id: false}); err != nil {
			return err
		}
	}
	return verifyGoBuildMaterialization(ctx, o.builder.runner, build.id, o.inputs.manifest)
}

func (o *goBuildOperation) remove(runtime *goBuildRuntime) error {
	if runtime == nil || runtime.collected {
		return nil
	}
	ctx, cancel := resolverCleanupContext()
	_, err := o.builder.runner.Output(ctx, "docker", "rm", "--force", runtime.id)
	cancel()
	var failure error
	if err != nil {
		failure = errors.New("go build runtime cleanup failed")
	}
	if runtime.trace != nil {
		ctx, cancel = resolverCleanupContext()
		observed, limitation, diagnostic := collectTraceDiagnostic(ctx, runtime.trace)
		cancel()
		o.observations = append(o.observations, observed...)
		if limitation != "" {
			if o.firstDiagnostic.Reason == "" {
				o.firstDiagnostic = diagnostic
			}
			failure = errors.Join(failure, errors.New("go build observation is incomplete"))
		}
	}
	if err == nil {
		runtime.collected = true
	}
	return failure
}

func (o *goBuildOperation) dispose() error {
	var err error
	for i := len(o.runtimes) - 1; i >= 0; i-- {
		err = errors.Join(err, o.remove(o.runtimes[i]))
	}
	for _, volume := range []*closureVolume{&o.outputVolume, &o.volume} {
		if volume.name != "" {
			ctx, cancel := resolverCleanupContext()
			if volume.verify(ctx, o.builder.runner) != nil {
				err = errors.Join(err, errors.New("go build volume cleanup identity unavailable"))
			} else if _, failure := o.builder.runner.Output(ctx, "docker", "volume", "rm", volume.name); failure != nil {
				err = errors.Join(err, errors.New("go build volume cleanup failed"))
			}
			cancel()
		}
	}
	if o.inputs != nil {
		err = errors.Join(err, o.inputs.close())
	}
	return err
}

func (o *goBuildOperation) artifactOutput(ctx context.Context, runtime *goBuildRuntime, arguments ...string) ([]byte, error) {
	writer := &goResolverBoundedOutput{limit: goResolverOutputLimit}
	err := o.builder.runner.(streamingDockerOutput).RunOutput(ctx, writer, "docker", boundaryExecArguments(runtime.id, boundaryELFHandoffMode, append([]string{goResolverBinary, "-C", goBuildGuestProject}, arguments...)...)...)
	if err != nil {
		reason := pythonCommandErrorReason(err)
		if o.commandReason == "" {
			o.commandReason = reason
		}
		return nil, fmt.Errorf("offline go build command failed: %s", reason)
	}
	if writer.exceeded {
		return nil, errors.New("offline go build output exceeded bound")
	}
	return append([]byte(nil), writer.Bytes()...), nil
}

func (o *goBuildOperation) checkGraph(ctx context.Context, runtime *goBuildRuntime, expected domain.ProjectDependencySnapshot) error {
	json, err := o.artifactOutput(ctx, runtime, "mod", "download", "-json", "all")
	if err != nil {
		return err
	}
	records, err := artifactgo.ParseProjectDownloadJSON(json)
	if err != nil {
		return errors.New("offline go build module output is invalid")
	}
	graph, err := o.artifactOutput(ctx, runtime, "mod", "graph")
	if err != nil {
		return err
	}
	snapshot, err := artifactgo.BuildProjectSnapshot(expected.Context(), records, graph, o.inputs.manifest.controls["go.mod"], o.inputs.manifest.controls["go.sum"])
	if err != nil || snapshot.GraphDigest() != expected.GraphDigest() || !reflect.DeepEqual(snapshot.Dependencies(), expected.Dependencies()) || !reflect.DeepEqual(snapshot.ControlDigests(), expected.ControlDigests()) || snapshot.DependencyFree() != expected.DependencyFree() {
		return errors.New("offline go build graph differs from retained approval")
	}
	return nil
}

type goBuildArtifactRunner struct {
	operation *goBuildOperation
	runtime   *goBuildRuntime
}

func (r *goBuildArtifactRunner) RunGo(ctx context.Context, project string, environment []string, arguments ...string) ([]byte, error) {
	if r == nil || r.operation == nil || r.runtime == nil || project != goBuildGuestProject || artifactgo.ValidateBuildEnvironmentForCache(environment, goBuildGuestCache) != nil || len(arguments) < 2 || len(arguments) > 3 || arguments[0] != "build" || arguments[1] != "-mod=readonly" {
		return nil, errGoBuildInput
	}
	selector := ""
	if len(arguments) == 3 {
		selector = arguments[2]
	}
	r.operation.phase = "BUILD_ALL"
	_, err := r.operation.artifactOutput(ctx, r.runtime, goBuildValidationArguments(selector)...)
	if err == nil {
		r.operation.phase = "PACKAGE_ROLES"
		var roles []byte
		roles, err = r.operation.artifactOutput(ctx, r.runtime, goBuildRoleArguments(selector)...)
		if err == nil {
			var main bool
			main, err = goBuildHasMain(roles)
			if err == nil && main {
				r.operation.phase = "BUILD_OUTPUT"
				_, err = r.operation.artifactOutput(ctx, r.runtime, goBuildCompilerArguments(selector)...)
			}
		}
	}
	if err != nil {
		return nil, &isolatedGoBuildFailure{cause: err, phase: r.operation.phase, commandReason: r.operation.commandReason}
	}
	return nil, nil
}

func goBuildRunnerFailure(err error) error {
	var trusted *isolatedGoBuildFailure
	if errors.As(err, &trusted) {
		return fmt.Errorf("network-disabled Go build failed: %w", trusted)
	}
	return errors.New("network-disabled Go build failed")
}
