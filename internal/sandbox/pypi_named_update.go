package sandbox

import (
	"context"
	"errors"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type pyTorchUpgradeTraversal struct {
	input          artifactpypi.UpgradeInput
	final, started bool
	expected       map[string]artifactpypi.Candidate
	records        []artifactpypi.SourcePinnedUpgradeRecord
	rawBytes       int
}

// The original first call creates only frozen METADATA. Subsequent calls reuse
// that private data; pip is imported before changing its data-only search path.
const pyTorchUpgradeSelectionScript = `import sys
from pip._internal.cli.main import main
sys.path.insert(0,sys.argv[1])`

// Return only the bounded report after pip succeeds, in its own trusted process.
// Human pip output stays on stderr. A failed selection cannot return a stale
// report, and no additional observed Python process is needed just to read it.
const pyTorchUpgradeReportScript = `
import contextlib
limit=int(sys.argv[3])
if limit<=0 or limit>4194304: raise ValueError('invalid report bound')
if sys.argv[2] not in ('/tmp/haa-pypi-resolver/changes.json','/tmp/haa-pypi-resolver/report.json'): raise ValueError('invalid report path')
with contextlib.redirect_stdout(sys.stderr):
 result=main(sys.argv[4:])
if result: raise SystemExit(result)
with open(sys.argv[2],'rb') as report: body=report.read(limit+1)
if not body or len(body)>limit: raise ValueError('report exceeds bound')
sys.stdout.buffer.write(body)`

const pyTorchUpgradeFreshScript = "import sys\nfrom pip._internal.cli.main import main" + pyTorchUpgradeReportScript

func (s *pyTorchUpgradeTraversal) withRetainedExtras(project string, incoming []artifactpypi.BoundedRequirement, current map[string]artifactpypi.Candidate) (string, []string, error) {
	selected := current
	if s.final {
		selected = s.expected
	}
	values := namedUpgradeCandidateValues(selected)
	extras, err := s.input.RequiredUpgradeExtras(project, values)
	if err != nil {
		return "", nil, err
	}
	requests := append([]artifactpypi.BoundedRequirement(nil), incoming...)
	if len(extras) > 0 {
		if project != "cuda-toolkit" {
			return "", nil, errors.New("retained upgrade extras have an unsupported project")
		}
		request, err := artifactpypi.ParseBoundedRequirement("cuda-toolkit[" + strings.Join(extras, ",") + `]; platform_system == "Linux"`)
		if err != nil {
			return "", nil, err
		}
		requests = append(requests, request)
	}
	aggregated, err := artifactpypi.AggregateBoundedRequirements(project, requests)
	if err != nil {
		return "", nil, err
	}
	return aggregated.Request(), aggregated.Extras(), nil
}

func namedUpgradeCandidateValues(items map[string]artifactpypi.Candidate) []artifactpypi.Candidate {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	values := make([]artifactpypi.Candidate, 0, len(names))
	for _, name := range names {
		values = append(values, items[name])
	}
	return values
}

func (r *PyPIResolver) resolvePyTorchDependencyUpgrade(ctx context.Context, containerID string, runtime PythonRuntime, reference domain.ArtifactReference, input artifactpypi.UpgradeInput) (domain.DependencyResolution, error) {
	if reference != input.Requirement().Reference() || reference.Source() != r.profile.Source() {
		return domain.DependencyResolution{}, errors.New("named upgrade input does not bind its source")
	}
	control, err := input.Control()
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if err := r.writeUpgradeControl(ctx, containerID, "update-input.json", control.Body()); err != nil {
		return domain.DependencyResolution{}, err
	}
	first := &pyTorchUpgradeTraversal{input: input}
	candidates, _, err := r.resolvePyTorchGraphWithUpgrade(ctx, containerID, runtime, reference, first)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	proposal, err := input.ProposeSourcePinnedUpgrade(candidates)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if err := r.writeUpgradeControl(ctx, containerID, "upgrade-constraints.txt", proposal.Constraints()); err != nil {
		return domain.DependencyResolution{}, err
	}
	expected := map[string]artifactpypi.Candidate{}
	for _, item := range candidates {
		expected[item.Project()] = item
	}
	final := &pyTorchUpgradeTraversal{input: input, final: true, expected: expected}
	finalCandidates, _, err := r.resolvePyTorchGraphWithUpgrade(ctx, containerID, runtime, proposal.Reference(), final)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	if err := proposal.ValidateSourcePinnedFinal(finalCandidates); err != nil {
		return domain.DependencyResolution{}, err
	}
	selected, err := input.BuildSourcePinnedUpgrade(first.records, final.records, runtime.PipVersion)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	runtimeIdentity := "python:" + runtime.PythonVersion + ";pip:" + runtime.PipVersion + ";target:" + runtime.InterpreterTag + "/" + runtime.ABITag + "/" + runtime.PlatformTag + ";source:" + r.profile.Name()
	resolution, err := domain.NewDependencyResolution(selected.Graph(), runtimeIdentity, selected.Digest())
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return resolution.WithProjectControls(selected.Controls())
}

func (r *PyPIResolver) resolvePyTorchUpgradeCandidate(ctx context.Context, containerID string, runtime PythonRuntime, profile artifactpypi.SourceProfile, requirement string, extras []string, primary bool, reportLimit int, state *pyTorchUpgradeTraversal) (artifactpypi.Candidate, []byte, error) {
	project, err := artifactpypi.DependencyProject(requirement)
	if err != nil {
		return artifactpypi.Candidate{}, nil, err
	}
	var reference domain.ArtifactReference
	var choice artifactpypi.UpgradeNodeSelection
	var delta []byte
	request := requirement
	constraints := ""
	if state.final {
		expected, present := state.expected[project]
		if !present || expected.Source() != profile.Source() {
			return artifactpypi.Candidate{}, nil, errors.New("final named upgrade introduced an unselected dependency")
		}
		request, err = expected.ExactUpgradeRequirement()
		if err != nil {
			return artifactpypi.Candidate{}, nil, err
		}
		reference, err = artifactpypi.ParseReferenceForSource(project+"@"+expected.Version(), profile.Source())
		if err != nil {
			return artifactpypi.Candidate{}, nil, err
		}
		constraints = pypiResolverProjectDir + "/upgrade-constraints.txt"
	} else {
		delta, err = r.selectPyTorchUpgradeNode(ctx, containerID, profile, requirement, primary, state)
		if err != nil {
			return artifactpypi.Candidate{}, nil, err
		}
		choice, err = state.input.ParseSourcePinnedNodeSelection(requirement, profile.Source(), extras, delta, runtime.PipVersion)
		if err != nil {
			return artifactpypi.Candidate{}, nil, err
		}
		reference = choice.Reference()
		request, err = namedUpgradeExactRequest(reference, extras)
		if err != nil {
			return artifactpypi.Candidate{}, nil, err
		}
	}
	candidate, body, err := r.resolvePyTorchCandidateWithUpgrade(ctx, containerID, runtime, reference, profile, request, extras, reportLimit, true, constraints)
	if err != nil {
		return artifactpypi.Candidate{}, nil, err
	}
	if !state.final {
		if err := choice.ValidateFreshCandidate(candidate); err != nil {
			return artifactpypi.Candidate{}, nil, err
		}
	}
	if len(state.records) >= maxPyTorchReports || len(body) > maxPyTorchReportBytes-state.rawBytes || len(delta) > maxPyTorchReportBytes-state.rawBytes-len(body) {
		return artifactpypi.Candidate{}, nil, errors.New("named upgrade report traversal exceeds bound")
	}
	state.rawBytes += len(body) + len(delta)
	recordedRequest := requirement
	if state.final {
		recordedRequest = request
	}
	state.records = append(state.records, artifactpypi.SourcePinnedUpgradeRecord{SourceID: profile.Source().String(), Requirement: recordedRequest, RequestedExtras: append([]string(nil), extras...), SelectionReport: append([]byte(nil), delta...), CandidateReport: append([]byte(nil), body...)})
	return candidate, body, nil
}

func namedUpgradeExactRequest(reference domain.ArtifactReference, extras []string) (string, error) {
	request, err := pypiRequirement(reference)
	if err != nil {
		return "", err
	}
	if len(extras) == 0 {
		return request, nil
	}
	project, err := artifactpypi.RequestedProject(reference)
	if err != nil || project != "cuda-toolkit" {
		return "", errors.New("named upgrade candidate extras have an unsupported project")
	}
	parts := strings.SplitN(request, "==", 2)
	if len(parts) != 2 {
		return "", errors.New("named upgrade requires an exact selected version")
	}
	return project + "[" + strings.Join(extras, ",") + "]==" + parts[1] + `; platform_system == "Linux"`, nil
}

func (r *PyPIResolver) selectPyTorchUpgradeNode(ctx context.Context, containerID string, profile artifactpypi.SourceProfile, requirement string, primary bool, state *pyTorchUpgradeTraversal) ([]byte, error) {
	arguments := []string{"install", "--dry-run", "--report", pypiResolverProjectDir + "/changes.json", "--disable-pip-version-check", "--no-input", "--no-cache-dir", "--isolated", "--no-deps", "--index-url", profile.IndexURL(), requirement}
	if primary {
		arguments = append(arguments, "--upgrade", "--upgrade-strategy", "only-if-needed")
	}
	var launch []string
	if !state.started {
		if !primary {
			return nil, errors.New("named upgrade metadata must be prepared by its root")
		}
		launch = boundaryExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-c", pypiUpgradeMetadataScript+pyTorchUpgradeReportScript, pypiResolverProjectDir+"/update-input.json", pypiResolverProjectDir+"/changes.json", "4194304")
		state.started = true
	} else {
		launch = boundaryExecArguments(containerID, boundaryLaunchMode, "python", "-I", "-c", pyTorchUpgradeSelectionScript+pyTorchUpgradeReportScript, pypiResolverProjectDir+"/installed-metadata", pypiResolverProjectDir+"/changes.json", "4194304")
	}
	body, err := r.runner.Output(ctx, "docker", append(launch, arguments...)...)
	if err != nil {
		return nil, errors.New("select source-pinned only-if-needed upgrade failed")
	}
	if len(body) == 0 || len(body) > 4<<20 {
		return nil, errors.New("read source-pinned upgrade selection failed")
	}
	return body, nil
}
