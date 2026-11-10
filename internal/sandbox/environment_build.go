package sandbox

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const projectBuildEnvironmentDescriptor = runtimeidentity.NodeImageReference + "|" + runtimeidentity.GVisorPatchSHA256 + "|" + domain.SandboxEnvironmentStateSchema + "|" + string(domain.SandboxEnvironmentMinimal) + "|" + string(domain.SandboxEnvironmentDevelopment) + "|" + environmentWorkspacePolicy + "|" + runtimeidentity.SandboxResponderInspectionIdentity

// Matches the existing Go/Cargo private /tmp limit; experiments are sequential.
const projectBuildWorkspaceCapacity int64 = 512 << 20

type projectBuildEnvironmentObserver interface {
	startProjectBuildEnvironmentProfile(context.Context, string, string, bool, bool, *observationTraceLedger, ...*responderSupply) (TraceReader, error)
	AwaitMountAnchors(context.Context, string) error
}

// Go/Cargo retain their own input, compiler and output owners. This shared
// state holder covers only the fresh mutable workspace and its DATA keeper.
type projectBuildEnvironment struct {
	session   domain.SandboxSessionID
	profile   inspectionEnvironmentProfile
	workspace closureVolume
	reader    *goBuildRuntime
	before    environmentState
	limits    environmentStateLimits
}

func (e *projectBuildEnvironment) begin(ctx context.Context, runner CommandRunner, observer projectBuildEnvironmentObserver, kind, writer string, output *closureVolume, budget *observationTraceLedger) error {
	if e == nil || ctx == nil || runner == nil || observer == nil || budget == nil || e.reader != nil || !exactObservationContainerID(writer) {
		return errEnvironmentState
	}
	if _, ok := projectBuildEnvironmentExpectedTopology(kind, output != nil, true); !ok {
		return errEnvironmentState
	}
	seed, err := e.profile.payload()
	if err != nil {
		return err
	}
	input, ok := runner.(inputCommandRunner)
	if !ok {
		return errEnvironmentState
	}
	if err := input.RunInput(ctx, bytes.NewReader(seed), "docker", "cp", "--archive", "-", writer+":/tmp"); err != nil {
		return err
	}
	args, err := environmentContainerArguments(e.session, e.profile, e.workspace, true)
	if err != nil {
		return err
	}
	if output != nil {
		mount, err := output.mountArgument(true)
		image := slices.Index(args, nodeImageReference)
		if err != nil || image < 0 {
			return errors.Join(errEnvironmentState, err)
		}
		args = append(append(append([]string{}, args[:image]...), "--mount", mount), args[image:]...)
	}
	created, err := runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if exactObservationContainerID(id) {
		e.reader = &goBuildRuntime{id: id}
	}
	if err != nil || e.reader == nil {
		return errors.Join(errEnvironmentState, err)
	}
	if err := e.workspace.verifyContainerMount(ctx, runner, id, true); err != nil {
		return err
	}
	if output != nil {
		if err := output.verifyContainerMount(ctx, runner, id, true); err != nil {
			return err
		}
	}
	e.reader.trace, err = observer.startProjectBuildEnvironmentProfile(ctx, id, kind, output != nil, true, budget)
	if err != nil {
		return err
	}
	if err := discardCommand(ctx, runner, "docker", "start", id); err != nil {
		return err
	}
	if err := awaitBoundaryHelper(ctx, runner, id); err != nil {
		return err
	}
	if err := observer.AwaitMountAnchors(ctx, id); err != nil {
		return err
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, id); err != nil {
		return err
	}
	if err := verifyEnvironmentResolver(ctx, runner, id); err != nil {
		return err
	}
	attachments := map[string]bool{writer: false, id: true}
	if err := e.workspace.verifyAttachments(ctx, runner, attachments); err != nil {
		return err
	}
	capacity := e.workspace.capacity
	if output != nil {
		if err := output.verifyAttachments(ctx, runner, attachments); err != nil {
			return err
		}
		capacity += output.capacity
	}
	e.limits = environmentStateLimits{entries: int(domain.MaximumObservationSummaryCount) + len(e.profile.fixtures) + 4, bytes: capacity, archiveBytes: 2 * capacity}
	e.before, err = captureEnvironmentState(ctx, runner, id, e.limits)
	if err != nil {
		return err
	}
	return e.profile.verifyPlacement(e.before)
}

// The caller has already terminated and drained its Artifact writer. The
// remaining keeper is readonly and omits immutable source/cache mounts.
func (e *projectBuildEnvironment) final(ctx context.Context, runner CommandRunner, output *closureVolume) (domain.SandboxEnvironmentState, error) {
	var state domain.SandboxEnvironmentState
	if e.reader == nil || e.reader.collected {
		return state, errEnvironmentState
	}
	attachments := map[string]bool{e.reader.id: true}
	if err := e.workspace.verifyAttachments(ctx, runner, attachments); err != nil {
		return state, err
	}
	if output != nil {
		if err := output.verifyAttachments(ctx, runner, attachments); err != nil {
			return state, err
		}
	}
	after, err := captureEnvironmentState(ctx, runner, e.reader.id, e.limits)
	if err != nil {
		return state, err
	}
	changes, err := compareEnvironmentStates(e.before, after)
	if err != nil {
		return state, err
	}
	fixture, _ := domain.NewSHA256Digest(e.profile.identity)
	before, _ := domain.NewSHA256Digest(e.before.digest)
	final, _ := domain.NewSHA256Digest(after.digest)
	state = domain.SandboxEnvironmentState{Profile: e.profile.kind, FixtureDigest: fixture, Before: before, After: final, Created: changes.created, Deleted: changes.deleted, Modified: changes.modified, VFSOperationsCovered: true}
	_, err = state.Summary()
	return state, err
}

// Build writers keep the original immutable inputs and executable output role.
// A fixed Node DATA keeper sees only mutable roots, never the dependency cache.
func projectBuildEnvironmentExpectedTopology(kind string, output, reader bool) ([]observerMountExpectation, bool) {
	if kind != "go" && kind != "cargo" || kind == "cargo" && !output {
		return nil, false
	}
	var topology []observerMountExpectation
	var ok bool
	if reader {
		topology, ok = environmentExpectedTopology(true)
		if output {
			root := goBuildGuestOutput
			if kind == "cargo" {
				root = cargoBuildGuestTarget
			}
			topology = append(topology, observerMountExpectation{root, "workspace", "/tmp", "9p", true, false, false, false})
		}
		return topology, ok
	}
	if kind == "cargo" {
		topology, ok = cargoBuildExpectedTopology(true)
	} else if output {
		topology, ok = goBuildOutputExpectedTopology(true)
	} else {
		topology, ok = goBuildInputExpectedTopology(true)
	}
	if !ok {
		return nil, false
	}
	for index, mount := range topology {
		if mount.Mountpoint == environmentWorkspaceRoot {
			topology[index] = observerMountExpectation{environmentWorkspaceRoot, "workspace", "/", "9p", false, true, true, false}
			return topology, true
		}
	}
	return nil, false
}

func (o *SharedObserver) startProjectBuildEnvironmentProfile(ctx context.Context, container, kind string, output, reader bool, budget *observationTraceLedger, supplies ...*responderSupply) (TraceReader, error) {
	topology, ok := projectBuildEnvironmentExpectedTopology(kind, output, reader)
	if !ok || budget == nil || len(supplies) > 1 || len(supplies) == 1 && reader {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if len(supplies) == 1 {
		var err error
		topology, err = supplies[0].expectedTopology(topology)
		if err != nil {
			return nil, err
		}
	}
	profile := goBuildProfile
	if kind == "cargo" {
		profile = cargoBuildProfile
	}
	if reader {
		profile = "npm-lifecycle"
	}
	return o.startProfileWithBudget(ctx, container, profile, topology, budget)
}
