package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type nodeInputObserver interface {
	nodeMaterializationObserver
	StartNodeExecutionInputProfile(context.Context, string) (TraceReader, error)
}

func (o *SharedObserver) StartNodeExecutionInputProfile(ctx context.Context, container string) (TraceReader, error) {
	topology, ok := nodeExecutionInputExpectedTopology(true)
	if !ok {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfile(ctx, container, "npm-lifecycle", topology)
}

type nodeInputContainer struct {
	id, name, owner string
	trace           TraceReader
}

type nodeExecutionInputs struct {
	backend        *Backend
	volume         closureVolume
	writer, reader *nodeInputContainer
	cancel         context.CancelFunc
	run            context.Context
}

// ValidateNodeExecutionInputs exercises only the complete DATA introduction
// and readonly view, then disposes it. No selected Artifact program is run.
func (b *Backend) ValidateNodeExecutionInputs(ctx context.Context, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout) (resultErr error) {
	defer func() {
		if ctx != nil && ctx.Err() != nil {
			resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		}
	}()
	inputs, err := b.prepareNodeExecutionInputs(ctx, execution, layout)
	if err != nil {
		return err
	}
	return inputs.close()
}

func (b *Backend) prepareNodeExecutionInputs(ctx context.Context, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout) (*nodeExecutionInputs, error) {
	return b.prepareNodeExecutionInputView(ctx, execution, layout, false)
}

func (b *Backend) prepareNodeExecutionInputView(ctx context.Context, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout, execute bool) (_ *nodeExecutionInputs, resultErr error) {
	if b == nil || ctx == nil || !layout.MatchesExecution(execution) || execution.RuntimeIdentity() != nodeImageReference || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil {
		return nil, errNodeMaterialization
	}
	input, inputOK := b.runner.(inputCommandRunner)
	stream, streamOK := b.runner.(streamingDockerOutput)
	observer, observerOK := b.observer.(nodeInputObserver)
	if !inputOK || !streamOK || !observerOK {
		return nil, errNodeMaterialization
	}
	if execute {
		if _, ok := b.observer.(nodeExecutionObserver); !ok {
			return nil, errNodeMaterialization
		}
	}
	payload, err := artifactnpm.RenderNodeExecutionInputs(ctx, execution, layout)
	if err != nil {
		return nil, err
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available {
		return nil, errors.Join(errNodeMaterialization, err)
	}
	owner, err := b.newSessionID()
	if err != nil || owner.String() == "" {
		return nil, errors.Join(errNodeMaterialization, err)
	}
	run, cancel := context.WithTimeout(ctx, b.wallTimeout)
	inputs := &nodeExecutionInputs{backend: b, cancel: cancel, run: run}
	ready := false
	defer func() {
		if !ready {
			resultErr = errors.Join(resultErr, inputs.close())
		}
	}()
	inputs.volume, err = createNodeMaterializationVolume(run, b.runner, owner.String(), execution.SourceFingerprint().String())
	if err != nil {
		return nil, errors.Join(errNodeMaterialization, err)
	}
	inputs.writer, err = b.startNodeInputContainer(run, inputs.volume, observer, false, nil, false)
	if err != nil {
		return nil, fmt.Errorf("start node input writer: %w", err)
	}
	if err := input.RunInput(run, bytes.NewReader(payload), "docker", "cp", "--archive", "-", inputs.writer.id+":"+nodeExecutionRoot); err != nil {
		return nil, fmt.Errorf("copy original node input tree: %w", errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run)))
	}
	if err := verifyNodeExecutionInputTree(run, stream, inputs.writer.id, execution, layout); err != nil {
		return nil, fmt.Errorf("verify original node input writer tree: %w", err)
	}
	// A named local-driver tmpfs loses its contents at the last unmount. Keep
	// the exact trusted writer until the RO reader is mounted and attested.
	// No Artifact program may run during this preparation-only overlap.
	inputs.reader, err = b.startNodeInputContainer(run, inputs.volume, observer, true, inputs.writer, execute)
	if err != nil {
		return nil, fmt.Errorf("start readonly node input reader: %w", err)
	}
	if err := inputs.writer.close(b); err != nil {
		return nil, fmt.Errorf("retire original node input writer: %w", err)
	}
	inputs.writer = nil
	if err := inputs.volume.verifyAttachments(run, b.runner, map[string]bool{inputs.reader.id: true}); err != nil {
		return nil, fmt.Errorf("verify sole readonly node input attachment: %w", err)
	}
	if err := verifyNodeExecutionInputTree(run, stream, inputs.reader.id, execution, layout); err != nil {
		return nil, fmt.Errorf("verify readonly node input reader tree: %w", err)
	}
	if err := inputs.volume.verifyAttachments(run, b.runner, map[string]bool{inputs.reader.id: true}); err != nil {
		return nil, err
	}
	if run.Err() != nil {
		return nil, errors.Join(run.Err(), context.Cause(run))
	}
	ready = true
	return inputs, nil
}

func (b *Backend) startNodeInputContainer(ctx context.Context, volume closureVolume, observer nodeInputObserver, readOnly bool, writer *nodeInputContainer, execute bool) (*nodeInputContainer, error) {
	session, err := b.newSessionID()
	if err != nil || session.String() == "" {
		return nil, errors.Join(errNodeMaterialization, err)
	}
	container := &nodeInputContainer{name: "heliopause-" + session.String(), owner: session.String()}
	args, err := nodeExecutionInputCreateArguments(session, volume, readOnly)
	if err != nil {
		return container, err
	}
	if execute {
		args, err = nodeExecutionWorkingDirectory(args)
		if err != nil {
			return container, err
		}
	}
	args = append(args[:1], append([]string{"--label", nodeMaterializationOwnerLabel + "=" + session.String()}, args[1:]...)...)
	body, err := b.runner.Output(ctx, "docker", args...)
	if exactObservationContainerID(strings.TrimSpace(string(body))) {
		container.id = strings.TrimSpace(string(body))
	}
	if err != nil || container.id == "" {
		return container, errors.Join(errNodeMaterialization, err)
	}
	if execute {
		if !readOnly {
			return container, errNodeMaterialization
		}
		container.trace, err = observer.(nodeExecutionObserver).StartNodeExecutionProfile(ctx, container.id)
	} else if readOnly {
		container.trace, err = observer.StartNodeExecutionInputProfile(ctx, container.id)
	} else {
		container.trace, err = observer.StartNodeMaterializationProfile(ctx, container.id)
	}
	if err != nil || container.trace == nil {
		return container, errors.Join(errNodeMaterialization, err)
	}
	if _, err := b.runner.Output(ctx, "docker", "start", container.id); err != nil {
		return container, errors.Join(errNodeMaterialization, err)
	}
	if err := awaitBoundaryHelper(ctx, b.runner, container.id); err != nil {
		return container, err
	}
	if err := awaitMountAnchors(ctx, b.observer, container.id); err != nil {
		return container, err
	}
	authorized := map[string]bool{container.id: readOnly}
	if writer != nil {
		if !readOnly || !exactObservationContainerID(writer.id) || writer.id == container.id {
			return container, errNodeMaterialization
		}
		authorized[writer.id] = false
	}
	if err := volume.verifyAttachments(ctx, b.runner, authorized); err != nil {
		return container, err
	}
	return container, verifyPNPMInstanceNetwork(ctx, b.runner, container.id)
}

func verifyNodeExecutionInputTree(ctx context.Context, stream streamingDockerOutput, container string, execution domain.ExecutionContext, layout artifactnpm.NodeExecutionLayout) error {
	output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
	if err := stream.RunOutput(ctx, &output, "docker", "cp", container+":"+nodeExecutionProject+"/node_modules", "-"); err != nil || output.failed {
		return errors.Join(errNodeMaterialization, err, ctx.Err(), context.Cause(ctx))
	}
	native, err := artifactnpm.ParsePNPMLayoutArchive(ctx, output.body)
	if err != nil {
		return errors.Join(errNodeMaterialization, err)
	}
	return artifactnpm.CompareNodeExecutionInputs(ctx, execution, layout, native)
}

func (c *nodeInputContainer) close(b *Backend) error {
	if c == nil {
		return nil
	}
	cleanup, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
	err := cleanupNodeMaterialization(cleanup, b.runner, c.name, c.owner, c.id)
	cancel()
	if c.trace != nil {
		collect, done := context.WithTimeout(context.Background(), b.cleanupWait)
		_, limitation, diagnostic := collectTraceDiagnostic(collect, c.trace)
		done()
		c.trace = nil
		if limitation != "" || len(retainedSuspiciousTraceFacts(diagnostic.KindCounts)) != 0 {
			err = errors.Join(err, fmt.Errorf("node input trace is incomplete: %s", diagnostic.String()))
		}
	}
	return err
}

func (i *nodeExecutionInputs) close() error {
	if i == nil {
		return nil
	}
	defer i.cancel()
	err := errors.Join(i.reader.close(i.backend), i.writer.close(i.backend))
	if i.volume.name != "" {
		cleanup, cancel := context.WithTimeout(context.Background(), i.backend.cleanupWait)
		err = errors.Join(err, closeWritableOutputVolume(cleanup, i.backend.runner, &i.volume, nodeExecutionRoot))
		cancel()
	}
	return err
}
