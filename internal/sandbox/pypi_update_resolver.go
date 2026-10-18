package sandbox

import (
	"bytes"
	"context"
	"errors"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// ResolveDependencyUpdate reuses the existing complete public PyPI resolver
// lifecycle with frozen inert metadata. Named-source upgrades have a separate
// selection contract and are not implicitly redirected to public PyPI.
func (r *PyPIResolver) ResolveDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, controls []domain.ProjectControlFile) (resolution domain.DependencyResolution, resultErr error) {
	if r == nil || ctx == nil || !install.Valid() || install.Mode() != domain.InstallPythonVenv || r.profile.Source() != artifactpypi.PublicPyPIProfile().Source() || reference.Source() != r.profile.Source() {
		return domain.DependencyResolution{}, errors.New("public PyPI upgrade requires a supported existing virtual environment")
	}
	ctx, cancel := context.WithTimeout(ctx, pypiResolverTimeout)
	defer cancel()
	defer func() {
		if err := ctx.Err(); err != nil {
			resolution = domain.DependencyResolution{}
			resultErr = errors.Join(resultErr, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, err
	}
	input, err := artifactpypi.ParseUpgradeInput(controls, PinnedPythonRuntime().PythonVersion)
	if err != nil || reference != input.Requirement().Reference() {
		return domain.DependencyResolution{}, errors.New("PyPI upgrade input does not bind the requested root and runtime")
	}
	runner := r.runner
	if admitted, ok := runner.(*admissionAwareCommandRunner); ok {
		runner = admitted.runner
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return domain.DependencyResolution{}, errors.New("PyPI upgrade input stream is unavailable")
	}
	return r.resolveDependencies(ctx, reference, &input)
}

type pypiUpgradeSelection struct {
	report   artifactpypi.InstallationReport
	body     []byte
	controls []domain.ProjectControlFile
}

// pip is imported from the immutable runtime before adding the data-only path.
// sys.path.insert does not process .pth files. No target package/interpreter,
// entry_points file, executable or editable loader is copied into this directory.
const pypiUpgradeSelectionScript = `import base64,json,pathlib,sys
from pip._internal.cli.main import main
path=pathlib.Path(sys.argv[1])
wire=json.loads(path.read_bytes())
installed=path.parent/'installed-metadata'
installed.mkdir(mode=0o700)
for item in wire['installed']:
 name=item['dist_info']
 if pathlib.PurePath(name).name!=name or not name.endswith('.dist-info'): raise ValueError('invalid inert metadata directory')
 body=base64.b64decode(item['metadata'],validate=True)
 if len(body)>1048576: raise ValueError('metadata exceeds bound')
 directory=installed/name
 directory.mkdir(mode=0o700)
 (directory/'METADATA').write_bytes(body)
sys.path.insert(0,str(installed))
raise SystemExit(main(sys.argv[2:]))`

const pypiUpgradeWriteScript = `import sys
body=sys.stdin.buffer.read(4194305)
if len(body)>4194304: raise ValueError('upgrade control exceeds bound')
with open(sys.argv[1],'xb') as output: output.write(body)`

func (r *PyPIResolver) writeUpgradeControl(ctx context.Context, containerID, name string, body []byte) error {
	if (name != "update-input.json" && name != "upgrade-constraints.txt") || len(body) == 0 || len(body) > 4<<20 {
		return errors.New("PyPI upgrade private control is unsupported")
	}
	runner, ok := r.runner.(inputCommandRunner)
	if !ok {
		return errors.New("PyPI upgrade input stream is unavailable")
	}
	arguments := boundaryInputExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-c", pypiUpgradeWriteScript, pypiResolverProjectDir+"/"+name)
	if err := runner.RunInput(ctx, bytes.NewReader(body), "docker", arguments...); err != nil {
		return errors.New("copy frozen PyPI upgrade control failed")
	}
	return nil
}

func (r *PyPIResolver) resolveUpgradeSelection(ctx context.Context, containerID string, runtime PythonRuntime, input artifactpypi.UpgradeInput) (pypiUpgradeSelection, error) {
	control, err := input.Control()
	if err != nil {
		return pypiUpgradeSelection{}, err
	}
	if err := r.writeUpgradeControl(ctx, containerID, "update-input.json", control.Body()); err != nil {
		return pypiUpgradeSelection{}, err
	}
	arguments := pypiResolveArguments(r.profile, input.Requirement().Requirement())
	for i := range arguments {
		if arguments[i] == pypiResolverProjectDir+"/report.json" {
			arguments[i] = pypiResolverProjectDir + "/changes.json"
		}
	}
	arguments = append(arguments, "--upgrade", "--upgrade-strategy", "only-if-needed")
	launch := boundaryExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-c", pypiUpgradeSelectionScript, pypiResolverProjectDir+"/update-input.json")
	if _, err := r.runner.Output(ctx, "docker", append(launch, arguments...)...); err != nil {
		return pypiUpgradeSelection{}, errors.New("select only-if-needed PyPI upgrade failed")
	}
	delta, err := r.runner.Output(ctx, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-c", boundedReadScript, pypiResolverProjectDir+"/changes.json", "4194304")...)
	if err != nil {
		return pypiUpgradeSelection{}, errors.New("read PyPI upgrade delta failed")
	}
	proposal, err := input.ParseUpgradeDelta(delta, runtime.PipVersion)
	if err != nil {
		return pypiUpgradeSelection{}, err
	}
	if err := r.writeUpgradeControl(ctx, containerID, "upgrade-constraints.txt", proposal.Constraints()); err != nil {
		return pypiUpgradeSelection{}, err
	}
	request, err := pypiRequirement(proposal.Reference())
	if err != nil {
		return pypiUpgradeSelection{}, err
	}
	arguments = pypiResolveArguments(r.profile, request)
	arguments = append(arguments, "--ignore-installed", "--constraint", pypiResolverProjectDir+"/upgrade-constraints.txt")
	if _, err := r.runner.Output(ctx, "docker", append(boundaryExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-m", "pip"), arguments...)...); err != nil {
		return pypiUpgradeSelection{}, errors.New("resolve complete constrained PyPI upgrade failed")
	}
	body, err := r.runner.Output(ctx, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-c", boundedReadScript, pypiResolverProjectDir+"/report.json", "4194304")...)
	if err != nil {
		return pypiUpgradeSelection{}, errors.New("read complete PyPI upgrade selection failed")
	}
	report, err := artifactpypi.ParseInstallationReport(proposal.Reference(), body, runtime.PipVersion, runtime.PythonVersion)
	if err != nil {
		return pypiUpgradeSelection{}, err
	}
	if err := proposal.ValidateFinal(report); err != nil {
		return pypiUpgradeSelection{}, err
	}
	controls := []domain.ProjectControlFile{control}
	for _, item := range []struct {
		name string
		body []byte
	}{{"pypi-update-delta.json", delta}, {"pypi-update-report.json", body}} {
		selected, err := domain.NewProjectControlFile(item.name, item.body, true)
		if err != nil {
			return pypiUpgradeSelection{}, err
		}
		controls = append(controls, selected)
	}
	return pypiUpgradeSelection{report, body, controls}, nil
}
