package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"slices"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	elfWriterWork              = "/work"
	elfReaderWork              = "/tmp/haa-elf-work"
	elfWorkspaceCapacity int64 = 256 << 20
)

type elfEnvironmentTraceObserver interface {
	startELFEnvironmentProfile(context.Context, string, string, bool, *observationTraceLedger) (TraceReader, error)
}

// The existing ELF owner retains introduction and fixed terminal semantics.
// Workspace, termination and shared CPU/trace accounting reuse the lifecycle owner.
func (b *GitHubELFBackend) executeELFEnvironments(ctx context.Context, request domain.SandboxRequest, owner domain.SandboxSessionID, profile string, introduce func(context.Context, string, domain.AcquiredArtifact) error, probeTerminal func(context.Context, string) (string, error)) (domain.SandboxResult, error) {
	observer, ok := b.observer.(environmentTraceObserver)
	_, elfObserver := b.observer.(elfEnvironmentTraceObserver)
	_, input := b.runner.(inputCommandRunner)
	_, output := b.runner.(streamingDockerOutput)
	_, discard := b.runner.(discardCommandRunner)
	if !ok || !elfObserver || !input || !output || !discard || b.resources == nil {
		return incomplete(owner, "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE")
	}
	backend := &Backend{runner: b.runner, observer: b.observer, resources: b.resources, newSessionID: b.newSessionID, wallTimeout: sandboxWallTimeout}
	if profile == "terraform-provider" {
		backend.wallTimeout = 30 * time.Second
	}
	op, code, err := backend.newEnvironmentInspection(owner, observer, profile)
	if err != nil {
		result, e := incomplete(owner, code)
		return result, errors.Join(e, err)
	}
	run, cancel := context.WithDeadline(ctx, op.ledger.policy.wallDeadline)
	defer cancel()
	if err := op.initialize(run, cancel); err != nil {
		return op.finish(ctx, run, "M3_DYNAMIC_ACCOUNTING_UNAVAILABLE", err)
	}
	seen := map[string]bool{owner.String(): true}
	for _, kind := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
		session, err := b.newSessionID()
		if err != nil || session.String() == "" || seen[session.String()] {
			return op.finish(ctx, run, "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", errors.Join(errEnvironmentState, err))
		}
		seen[session.String()] = true
		environment, err := newInspectionEnvironmentProfile(kind, session)
		if err != nil {
			return op.finish(ctx, run, "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err)
		}
		code, err := op.elfExperiment(run, request.Artifact(), session, environment, introduce, probeTerminal)
		if code != "" || err != nil {
			return op.finish(ctx, run, code, err)
		}
	}
	completed, _ := domain.NewSandboxObservation(domain.ObservationProcess, "lifecycle-completed")
	op.observed = append(op.observed, completed)
	return op.finish(ctx, run, "", nil)
}

func (n *environmentInspection) elfExperiment(ctx context.Context, artifact domain.AcquiredArtifact, session domain.SandboxSessionID, profile inspectionEnvironmentProfile, introduce func(context.Context, string, domain.AcquiredArtifact) error, probeTerminal func(context.Context, string) (string, error)) (string, error) {
	var err error
	n.work, err = createInputVolume(ctx, n.backend.runner, session.String(), profile.identity, elfWorkspaceCapacity, false, projectBuildVolumeConfig{elfWriterWork, true})
	if err != nil {
		n.uncertain = n.work.name != ""
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	n.volume, err = createEnvironmentWorkspace(ctx, n.backend.runner, n.owner, profile, elfWorkspaceCapacity)
	if err != nil {
		n.uncertain = n.volume.name != ""
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	writer, err := n.start(ctx, session, profile, false)
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	seed, err := profile.payload()
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := n.backend.runner.(inputCommandRunner).RunInput(ctx, bytes.NewReader(seed), "docker", "cp", "--archive", "-", writer.id+":/tmp"); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	// Prepare this fixed DATA mountpoint without another guest executable.
	var mountpoint bytes.Buffer
	archive := tar.NewWriter(&mountpoint)
	if err := archive.WriteHeader(&tar.Header{Name: "haa-elf-work", Mode: 0700, Typeflag: tar.TypeDir, Uid: 1000, Gid: 1000}); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := archive.Close(); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := n.backend.runner.(inputCommandRunner).RunInput(ctx, bytes.NewReader(mountpoint.Bytes()), "docker", "cp", "--archive", "-", writer.id+":/tmp"); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := introduce(ctx, writer.id, artifact); err != nil {
		return "M6_DYNAMIC_ARTIFACT_INTRODUCTION_FAILED", err
	}
	anchor, err := n.start(ctx, session, profile, true)
	if err != nil {
		return "M3_DYNAMIC_SETUP_FAILED", err
	}
	attachments := map[string]bool{writer.id: false, anchor.id: true}
	if err := n.work.verifyAttachments(ctx, n.backend.runner, attachments); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := n.volume.verifyAttachments(ctx, n.backend.runner, attachments); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	limits := environmentStateLimits{entries: int(domain.MaximumObservationSummaryCount), bytes: 2 * elfWorkspaceCapacity, archiveBytes: 4 * elfWorkspaceCapacity}
	before, err := captureEnvironmentState(ctx, n.backend.runner, anchor.id, limits)
	if err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	if err := profile.verifyPlacement(before); err != nil {
		return "M3_DYNAMIC_ENVIRONMENT_PREPARATION_FAILED", err
	}
	n.ledger.mu.Lock()
	n.ledger.preparationOK, n.ledger.anchorAlive = true, true
	n.ledger.mu.Unlock()
	if err := n.ledger.beginUnit(string(profile.kind), writer.id, time.Now()); err != nil {
		return "M3_DYNAMIC_ACCOUNTING_INCOMPLETE", err
	}
	var terminal string
	if probeTerminal == nil {
		err = discardCommand(ctx, n.backend.runner, "docker", boundaryExecArguments(writer.id, boundaryELFHandoffMode, "/work/artifact")...)
	} else {
		terminal, err = probeTerminal(ctx, writer.id)
	}
	if terminal != "" {
		fact, e := domain.NewSandboxObservation(domain.ObservationProcess, terminal)
		if e != nil {
			return "M3_DYNAMIC_OBSERVER_FAILED", e
		}
		n.observed = append(n.observed, fact)
	}
	stopErr := n.stop(writer.id)
	if err != nil || ctx.Err() != nil {
		code := "M6_DYNAMIC_EXECUTION_FAILED"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			code = "M6_DYNAMIC_TIMEOUT"
		}
		return code, errors.Join(err, ctx.Err(), stopErr)
	}
	if stopErr != nil {
		return "M3_DYNAMIC_OBSERVER_FAILED", stopErr
	}
	return n.completeEnvironment(ctx, profile, writer, anchor, before, limits)
}

func (n *environmentInspection) elfContainerArguments(session domain.SandboxSessionID, profile inspectionEnvironmentProfile, readOnly bool) ([]string, error) {
	args, err := environmentContainerArguments(session, profile, n.volume, readOnly)
	if err != nil {
		return nil, err
	}
	work := n.work
	if readOnly {
		work.projectBuildDestination = elfReaderWork
	}
	mount, err := work.mountArgument(readOnly)
	image := slices.Index(args, nodeImageReference)
	if err != nil || image < 0 {
		return nil, errors.Join(errEnvironmentState, err)
	}
	return append(append(append([]string{}, args[:image]...), "--mount", mount), args[image:]...), nil
}

func elfEnvironmentExpectedTopology(profile string, readOnly bool) ([]observerMountExpectation, bool) {
	if profile != "github-elf" && profile != "terraform-provider" {
		return nil, false
	}
	topology, ok := environmentExpectedTopology(readOnly)
	root, mountRoot := elfWriterWork, "/"
	if readOnly {
		root, mountRoot = elfReaderWork, "/tmp"
	}
	return append(topology, observerMountExpectation{root, "workspace", mountRoot, "9p", readOnly, false, false, false}), ok
}

func (o *SharedObserver) startELFEnvironmentProfile(ctx context.Context, container, profile string, readOnly bool, budget *observationTraceLedger) (TraceReader, error) {
	topology, ok := elfEnvironmentExpectedTopology(profile, readOnly)
	if !ok || budget == nil {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	if readOnly {
		profile = "npm-lifecycle"
	}
	return o.startProfileWithBudget(ctx, container, profile, topology, budget)
}
