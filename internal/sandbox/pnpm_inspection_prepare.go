package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMInspectionPreparation = errors.New("pnpm complete inspection layout preparation is unavailable")

// PNPMInspectionPreparer owns script-free native metadata work. It returns no
// layout until the existing native scope has proved complete observation and
// cleanup. Artifact execution and Policy are separate consumers.
type PNPMInspectionPreparer struct {
	native *PNPMResolver
	intake *DockerArtifactIntroducer
}

func NewPNPMInspectionPreparer(native *PNPMResolver, intakeRoot string) (*PNPMInspectionPreparer, error) {
	if native == nil || native.runner == nil || native.endpoints == nil || native.observer == nil || native.probe == nil || native.policy == nil || native.prepare == nil {
		return nil, errPNPMInspectionPreparation
	}
	intake, err := NewDockerArtifactIntroducer(intakeRoot, native.runner)
	if err != nil {
		return nil, errors.Join(errPNPMInspectionPreparation, err)
	}
	return &PNPMInspectionPreparer{native, intake}, nil
}

func (p *PNPMInspectionPreparer) PrepareInspection(ctx context.Context, input artifactnpm.PNPMInspectionInput, inventories []artifactnpm.ArchiveInventory) (artifactnpm.PNPMInspectionLayout, error) {
	if p == nil || p.native == nil || p.intake == nil || ctx == nil || !input.Valid() || input.Context().RuntimeIdentity() != PNPMResolverRuntimeIdentity() {
		return artifactnpm.PNPMInspectionLayout{}, errPNPMInspectionPreparation
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.PNPMInspectionLayout{}, errors.Join(errPNPMInspectionPreparation, err, context.Cause(ctx))
	}
	if err := input.CheckWorkspace(inventories); err != nil {
		return artifactnpm.PNPMInspectionLayout{}, errors.Join(errPNPMInspectionPreparation, err)
	}
	archives, err := p.intake.freezePNPMArchives(ctx, input)
	if err != nil {
		return artifactnpm.PNPMInspectionLayout{}, errors.Join(errPNPMInspectionPreparation, err)
	}
	var bound artifactnpm.PNPMInspectionLayout
	err = p.native.withNativeProject(ctx, true, func(ctx context.Context, container, network string, _ *closureVolume) error {
		controls := input.Context().Controls()
		if err := copyPNPMProjectControls(ctx, p.native.runner, container, controls); err != nil {
			return err
		}
		if err := copyPNPMInspectionArchives(ctx, p.native.runner, container, archives); err != nil {
			return err
		}
		files := make([]string, 0, len(archives))
		for _, archive := range archives {
			files = append(files, pnpmProjectRoot+"/"+archive.name)
		}
		if err := runPNPMInspectionCommand(ctx, p.native.runner, container, pnpmInspectionCommand("store-add", files)); err != nil {
			return err
		}
		if err := runPNPMInspectionCommand(ctx, p.native.runner, container, pnpmInspectionCommand("fetch", nil)); err != nil {
			return err
		}
		if err := disconnectPNPMInspectionNetwork(ctx, p.native.runner, container, network); err != nil {
			return err
		}
		if err := runPNPMInspectionCommand(ctx, p.native.runner, container, pnpmInspectionCommand("install", nil)); err != nil {
			return err
		}
		streams := p.native.runner.(streamingDockerOutput)
		for _, original := range controls {
			actual, err := readPNPMProjectControl(ctx, streams, container, original.Name())
			if err != nil || !bytes.Equal(original.Body(), actual.Body()) {
				return errors.Join(errPNPMInspectionPreparation, err)
			}
		}
		output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
		if err := streams.RunOutput(ctx, &output, "docker", "cp", container+":"+pnpmProjectDirectory+"/node_modules", "-"); err != nil || output.failed {
			return errors.Join(errPNPMInspectionPreparation, err)
		}
		layout, err := artifactnpm.ParsePNPMLayoutArchive(ctx, output.body)
		if err != nil {
			return err
		}
		bound, err = input.BindLayout(ctx, inventories, layout.RestrictDirectoryWrites(), pnpmProjectRoot+"/store")
		return err
	})
	if err != nil {
		return artifactnpm.PNPMInspectionLayout{}, errors.Join(errPNPMInspectionPreparation, err, ctx.Err(), context.Cause(ctx))
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.PNPMInspectionLayout{}, errors.Join(errPNPMInspectionPreparation, err, context.Cause(ctx))
	}
	if bound.Digest().String() == "" {
		return artifactnpm.PNPMInspectionLayout{}, errPNPMInspectionPreparation
	}
	return bound, nil
}

// Exact commands already qualified by the native offline primitive. Arguments
// are private guest data, never native option pass-through or registry choice.
func pnpmInspectionCommand(operation string, files []string) []string {
	if operation == "store-add" && len(files) == 0 || operation != "store-add" && len(files) != 0 {
		return nil
	}
	directory := pnpmProjectDirectory
	var command []string
	switch operation {
	case "store-add":
		directory = "/"
		command = []string{"store", "add", "--offline"}
	case "fetch":
		command = []string{"fetch"}
	case "install":
		command = []string{"install", "--offline", "--frozen-lockfile"}
	default:
		return nil
	}
	arguments := append([]string{pnpmToolGuestPath, "--dir", directory}, command...)
	arguments = append(arguments, "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot+"/store", "--state-dir", pnpmProjectRoot+"/state", "--reporter=silent")
	if operation != "store-add" {
		return append(arguments, "--node-linker=isolated")
	}
	seen := map[string]bool{}
	for _, file := range files {
		name := path.Base(file)
		if path.Clean(file) != file || path.Dir(file) != pnpmProjectRoot+"/artifacts" || !strings.HasSuffix(name, ".tgz") || seen[file] {
			return nil
		}
		if _, err := domain.NewSHA256Digest(strings.TrimSuffix(name, ".tgz")); err != nil {
			return nil
		}
		seen[file] = true
		arguments = append(arguments, "file:"+file)
	}
	return arguments
}

func runPNPMInspectionCommand(ctx context.Context, runner CommandRunner, container string, arguments []string) error {
	if ctx == nil || len(arguments) == 0 || !exactObservationContainerID(container) {
		return errPNPMInspectionPreparation
	}
	_, err := runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, arguments...)...)
	if err != nil || ctx.Err() != nil {
		return errors.Join(errPNPMInspectionPreparation, err, ctx.Err(), context.Cause(ctx))
	}
	return nil
}

func disconnectPNPMInspectionNetwork(ctx context.Context, runner CommandRunner, container, network string) error {
	if ctx == nil || network == "" || network == "none" || !exactObservationContainerID(container) {
		return errPNPMInspectionPreparation
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(errPNPMInspectionPreparation, err, context.Cause(ctx))
	}
	if _, err := runner.Output(ctx, "docker", "network", "disconnect", network, container); err != nil {
		return errors.Join(errPNPMInspectionPreparation, err)
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", "{{json .NetworkSettings.Networks}}", container)
	var remaining map[string]json.RawMessage
	if err != nil || len(body) > 64<<10 || json.Unmarshal(body, &remaining) != nil || remaining == nil || len(remaining) != 0 {
		return errors.Join(errPNPMInspectionPreparation, err)
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(errPNPMInspectionPreparation, err, context.Cause(ctx))
	}
	return nil
}
