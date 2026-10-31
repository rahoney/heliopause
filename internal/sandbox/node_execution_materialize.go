package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	nodeExecutionRoot             = "/tmp/haa-node-execution"
	nodeExecutionProject          = nodeExecutionRoot + "/project"
	nodeExecutionWorkspaceLimit   = artifactnpm.PNPMInspectionWorkspaceLimit
	nodeMaterializationOwnerLabel = "io.heliopause.node-materialization"
)

var errNodeMaterialization = errors.New("node execution materialization is unavailable or incomplete")

type nodeMaterializationObserver interface {
	StartNodeMaterializationProfile(context.Context, string) (TraceReader, error)
}

// This scoped entry adds one separately attested DATA workspace. Ordinary npm
// consumers retain their topology and no Artifact execution role is granted.
func (o *SharedObserver) StartNodeMaterializationProfile(ctx context.Context, container string) (TraceReader, error) {
	topology, ok := nodeMaterializationExpectedTopology()
	if !ok {
		return nil, observerFault{reason: "LIFECYCLE_ERROR"}
	}
	return o.startProfile(ctx, container, "npm-lifecycle", topology)
}

func nodeMaterializationExpectedTopology() ([]observerMountExpectation, bool) {
	return nodeExecutionInputExpectedTopology(false)
}

func nodeExecutionInputExpectedTopology(readOnly bool) ([]observerMountExpectation, bool) {
	topology, ok := observerExpectedTopology("npm-lifecycle")
	if !ok {
		return nil, false
	}
	return append(topology, observerMountExpectation{nodeExecutionRoot, "workspace", "/tmp", "9p", readOnly, false, false, false}), true
}

func createNodeMaterializationVolume(ctx context.Context, runner CommandRunner, owner, source string) (closureVolume, error) {
	return createInputVolume(ctx, runner, owner+"-node-materialization", source, nodeExecutionWorkspaceLimit, false, projectBuildVolumeConfig{nodeExecutionRoot, false})
}

func nodeMaterializationCreateArguments(session domain.SandboxSessionID, volume closureVolume) ([]string, error) {
	return nodeExecutionInputCreateArguments(session, volume, false)
}

func nodeExecutionInputCreateArguments(session domain.SandboxSessionID, volume closureVolume, readOnly bool) ([]string, error) {
	if volume.destination() != nodeExecutionRoot || volume.projectBuildExecutable || volume.capacity != nodeExecutionWorkspaceLimit {
		return nil, errNodeMaterialization
	}
	mount, err := volume.mountArgument(readOnly)
	if err != nil {
		return nil, err
	}
	args := createArguments(session)
	image := slices.Index(args, nodeImageReference)
	if image < 0 {
		return nil, errNodeMaterialization
	}
	return append(append(append([]string{}, args[:image]...), "--mount", mount), args[image:]...), nil
}

// MaterializeNodeExecution runs only the pinned npm controller with scripts
// disabled. The requested artifact command/argv are never executed here.
func (b *Backend) MaterializeNodeExecution(ctx context.Context, execution domain.ExecutionContext, controls []domain.ProjectControlFile, inventories []artifactnpm.ArchiveInventory) (layout artifactnpm.NodeExecutionLayout, resultErr error) {
	if b == nil || ctx == nil || !execution.Valid() || execution.RuntimeIdentity() != nodeImageReference || b.runner == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil {
		return layout, errNodeMaterialization
	}
	if ctx.Err() != nil {
		return layout, errors.Join(errNodeMaterialization, ctx.Err(), context.Cause(ctx))
	}
	intake, ok := b.introducer.(*DockerArtifactIntroducer)
	streams, streamOK := b.runner.(streamingDockerOutput)
	input, inputOK := b.runner.(inputCommandRunner)
	observer, observerOK := b.observer.(nodeMaterializationObserver)
	if !ok || !streamOK || !inputOK || !observerOK {
		return layout, errNodeMaterialization
	}
	var subjects []domain.AcquiredArtifact
	for _, inspection := range execution.Set().Inspected().Inspections() {
		subjects = append(subjects, inspection.Artifact())
	}
	if len(inventories) != len(subjects) {
		return layout, errNodeMaterialization
	}
	archives, err := intake.freezeNPMProjectArchives(ctx, subjects)
	if err != nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	payload, err := nodeMaterializationPayload(ctx, archives, controls)
	if err != nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	capability, err := b.probe(ctx)
	if err != nil || !capability.Available {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	session, err := b.newSessionID()
	if err != nil || session.String() == "" {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	name := "heliopause-" + session.String()
	var container string
	var trace TraceReader
	var project closureVolume
	attempted := false
	defer func() {
		if attempted {
			cleanup, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
			err := cleanupNodeMaterialization(cleanup, b.runner, name, session.String(), container)
			cancel()
			if err != nil {
				resultErr = errors.Join(resultErr, errNodeMaterialization, err)
			}
		}
		if trace != nil {
			collect, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
			_, limitation, diagnostic := collectTraceDiagnostic(collect, trace)
			cancel()
			if limitation != "" || len(retainedSuspiciousTraceFacts(diagnostic.KindCounts)) != 0 {
				resultErr = errors.Join(resultErr, errNodeMaterialization, fmt.Errorf("node materialization trace is incomplete or unexpected: %s", diagnostic.String()))
			}
		}
		if project.name != "" {
			cleanup, cancel := context.WithTimeout(context.Background(), b.cleanupWait)
			err := closeWritableOutputVolume(cleanup, b.runner, &project, nodeExecutionRoot)
			cancel()
			if err != nil {
				resultErr = errors.Join(resultErr, errNodeMaterialization, err)
			}
		}
		if ctx.Err() != nil {
			resultErr = errors.Join(resultErr, errNodeMaterialization, ctx.Err(), context.Cause(ctx))
		}
		if resultErr != nil {
			layout = artifactnpm.NodeExecutionLayout{}
		}
	}()
	project, err = createNodeMaterializationVolume(ctx, b.runner, session.String(), execution.SourceFingerprint().String())
	if err != nil {
		return layout, fmt.Errorf("create node materialization workspace: %w", errors.Join(errNodeMaterialization, err))
	}
	arguments, err := nodeMaterializationCreateArguments(session, project)
	if err != nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	arguments = append(arguments[:1], append([]string{"--label", nodeMaterializationOwnerLabel + "=" + session.String()}, arguments[1:]...)...)
	attempted = true
	created, err := b.runner.Output(ctx, "docker", arguments...)
	if exactObservationContainerID(strings.TrimSpace(string(created))) {
		container = strings.TrimSpace(string(created))
	}
	if err != nil || container == "" {
		return layout, fmt.Errorf("create node materialization runtime: %w", errors.Join(errNodeMaterialization, err))
	}
	trace, err = observer.StartNodeMaterializationProfile(ctx, container)
	if err != nil || trace == nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	run, cancel := context.WithTimeout(ctx, b.wallTimeout)
	defer cancel()
	if _, err := b.runner.Output(run, "docker", "start", container); err != nil {
		return layout, fmt.Errorf("start node materialization runtime: %w", errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run)))
	}
	if err := awaitBoundaryHelper(run, b.runner, container); err != nil {
		return layout, errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run))
	}
	if err := awaitMountAnchors(run, b.observer, container); err != nil {
		return layout, errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run))
	}
	if err := project.verifyAttachments(run, b.runner, map[string]bool{container: false}); err != nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	if err := input.RunInput(run, bytes.NewReader(payload), "docker", "cp", "--archive", "-", container+":"+nodeExecutionRoot); err != nil {
		return layout, fmt.Errorf("copy frozen node materialization input: %w", errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run)))
	}
	if err := verifyPNPMInstanceNetwork(run, b.runner, container); err != nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	if err := discardCommand(run, b.runner, "docker", nodeMaterializationArguments(container)...); err != nil {
		return layout, fmt.Errorf("run fixed npm materialization controller: %w", errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run)))
	}
	if err := project.verifyAttachments(run, b.runner, map[string]bool{container: false}); err != nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
	if err := streams.RunOutput(run, &output, "docker", "cp", container+":"+nodeExecutionProject+"/node_modules", "-"); err != nil || output.failed {
		return layout, fmt.Errorf("read materialized node tree: %w", errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run)))
	}
	native, err := artifactnpm.ParsePNPMLayoutArchive(run, output.body)
	if err != nil {
		return layout, errors.Join(errNodeMaterialization, err)
	}
	layout, err = artifactnpm.BindMaterializedNodeExecutionLayout(run, execution, inventories, native)
	if err != nil || run.Err() != nil {
		return artifactnpm.NodeExecutionLayout{}, errors.Join(errNodeMaterialization, err, run.Err(), context.Cause(run))
	}
	return layout, nil
}

func nodeMaterializationArguments(container string) []string {
	// Only HAA's fixed controller source enters the shell. Its immutable Node
	// child uses the existing observed CONTROL clone lineage, never Artifact
	// code. The trailing builtin keeps the controller waiting for its child;
	// -e preserves a failing child status before the builtin can run.
	return boundaryExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", nodeMaterializationCommand)
}

const nodeMaterializationCommand = "/usr/local/bin/node -- /usr/local/lib/node_modules/npm/bin/npm-cli.js " +
	"--prefix /tmp/haa-node-execution/project ci --offline --ignore-scripts=true --bin-links=false " +
	"--workspaces=false --no-audit --no-fund --no-update-notifier --cache=/tmp/.npm " +
	"--userconfig=/tmp/haa-user.npmrc --globalconfig=/tmp/haa-global.npmrc --script-shell=/haa-runtime/haa-boundary\n:"

func nodeMaterializationPayload(ctx context.Context, archives []pnpmInspectionArchive, controls []domain.ProjectControlFile) ([]byte, error) {
	if len(archives) == 0 || len(controls) != 2 {
		return nil, errNodeMaterialization
	}
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	emit := func(name string, kind byte, body []byte) error {
		if ctx.Err() != nil {
			return errors.Join(errNodeMaterialization, ctx.Err(), context.Cause(ctx))
		}
		if output.Len() > artifactnpm.PNPMInspectionWorkspaceLimit-len(body) {
			return errNodeMaterialization
		}
		header := &tar.Header{Name: name, Typeflag: kind, Mode: 0o600, Uid: 1000, Gid: 1000, Size: int64(len(body))}
		if kind == tar.TypeDir {
			header.Mode = 0o700
		}
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if _, err := writer.Write(body); err != nil {
			return err
		}
		if output.Len() > artifactnpm.PNPMInspectionWorkspaceLimit {
			return errNodeMaterialization
		}
		return nil
	}
	for _, name := range []string{"project", "project/.heliopause", "project/.heliopause/artifacts"} {
		if err := emit(name, tar.TypeDir, nil); err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{}
	for _, file := range controls {
		if !file.Present() || file.Name() != "package.json" && file.Name() != "package-lock.json" || seen[file.Name()] {
			return nil, errNodeMaterialization
		}
		seen[file.Name()] = true
		if err := emit("project/"+file.Name(), tar.TypeReg, file.Body()); err != nil {
			return nil, err
		}
	}
	for _, archive := range archives {
		if archive.name != "artifacts/"+archive.subject.Digest().String()+".tgz" || uint64(len(archive.body)) != archive.subject.SizeBytes() {
			return nil, errNodeMaterialization
		}
		if seen[archive.name] {
			continue
		}
		seen[archive.name] = true
		if err := emit("project/.heliopause/"+archive.name, tar.TypeReg, archive.body); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if output.Len() > artifactnpm.PNPMInspectionWorkspaceLimit {
		return nil, errNodeMaterialization
	}
	return output.Bytes(), nil
}

func cleanupNodeMaterialization(ctx context.Context, runner CommandRunner, name, owner, expectedID string) error {
	query := []string{"ps", "-aq", "--no-trunc", "--filter", "name=^/" + name + "$", "--filter", "label=" + nodeMaterializationOwnerLabel + "=" + owner}
	output, err := runner.Output(ctx, "docker", query...)
	if err != nil {
		return errors.Join(errNodeMaterialization, err)
	}
	ids := strings.Fields(string(output))
	if len(ids) == 0 {
		return nil
	}
	if len(ids) != 1 || !exactObservationContainerID(ids[0]) || expectedID != "" && ids[0] != expectedID {
		return errNodeMaterialization
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--type", "container", ids[0])
	if err != nil {
		return errors.Join(errNodeMaterialization, err)
	}
	var records []struct {
		ID     string `json:"Id"`
		Name   string
		Config struct {
			Image  string
			Labels map[string]string
		}
	}
	if json.Unmarshal(body, &records) != nil || len(records) != 1 || records[0].ID != ids[0] || records[0].Name != "/"+name || records[0].Config.Image != nodeImageReference || records[0].Config.Labels[nodeMaterializationOwnerLabel] != owner {
		return errNodeMaterialization
	}
	if _, err := runner.Output(ctx, "docker", "rm", "--force", ids[0]); err != nil {
		return errors.Join(errNodeMaterialization, err)
	}
	output, err = runner.Output(ctx, "docker", query...)
	if err != nil || strings.TrimSpace(string(output)) != "" {
		return errors.Join(errNodeMaterialization, err)
	}
	return nil
}
