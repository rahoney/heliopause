package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type pypiUpgradeRecordingRunner struct {
	recordingRunner
	inputs [][]byte
}

func (r *pypiUpgradeRecordingRunner) RunInput(ctx context.Context, input io.Reader, binary string, args ...string) error {
	body, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	r.inputs = append(r.inputs, append([]byte(nil), body...))
	return r.recordingRunner.RunInput(ctx, bytes.NewReader(body), binary, args...)
}

func frozenPyPIUpgradeFixture(t *testing.T) (domain.ArtifactReference, domain.InstallContext, domain.ProjectControlFile) {
	t.Helper()
	requirement, err := artifactpypi.ParseUpgradeRequirement("primary")
	if err != nil {
		t.Fatal(err)
	}
	files := []artifactpypi.InstalledMetadataFile{
		{Directory: "primary-1.0.dist-info", Body: []byte("Name: primary\nVersion: 1.0\nRequires-Dist: child>=2\n\n")},
		{Directory: "child-2.0.dist-info", Body: []byte("Name: child\nVersion: 2.0\n\n")},
		{Directory: "other-9.0.dist-info", Body: []byte("Name: other\nVersion: 9.0\nRequires-Dist: child<3\n\n")},
	}
	input, err := artifactpypi.NewUpgradeInput(requirement, files, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	control, err := input.Control()
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/haa-pypi-upgrade-original")
	install, _ := domain.NewPythonVenvInstallContext(target)
	return requirement.Reference(), install, control
}

func emptyPyPIUpgradeDelta(t *testing.T) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(pypiResolverReportJSON()), &body); err != nil {
		t.Fatal(err)
	}
	body["install"] = []any{}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func pypiUpgradeResponses(t *testing.T) [][]byte {
	t.Helper()
	return [][]byte{
		[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789abcdef"), nil,
		[]byte("3.14.7\n"), []byte("pip 26.2.1 from pip\n"), []byte("cp314-cp314-manylinux_2_36_x86_64\n"), nil,
		nil, emptyPyPIUpgradeDelta(t), nil, []byte(pypiResolverReportJSON()),
		[]byte(pypiResolverSimpleJSON("child", "child-2.0-py3-none-any.whl", "")),
		[]byte(pypiResolverSimpleJSON("primary", "primary-1.0-py3-none-any.whl", ">=3.14")),
	}
}

func TestPyPIUpgradeResolverCopiesOnlyInertMetadataAndReturnsCompleteNoopSelection(t *testing.T) {
	reference, install, control := frozenPyPIUpgradeFixture(t)
	runner := &pypiUpgradeRecordingRunner{recordingRunner: recordingRunner{responses: pypiUpgradeResponses(t)}}
	observer := &recordingObserver{reader: &traceReader{records: []TraceRecord{{Kind: "network-attempt", Bytes: 1}}}}
	resolver, err := NewPyPIResolver(runner, pypiStaticEndpoints{}, observer, availablePythonProbe, &recordingResolverPolicyService{})
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := resolver.ResolveDependencyUpdate(context.Background(), reference, install, []domain.ProjectControlFile{control})
	if err != nil || len(resolution.Graph().Nodes()) != 2 || len(resolution.ProjectControls()) != 3 || len(runner.inputs) != 2 || !bytes.Equal(runner.inputs[0], control.Body()) {
		t.Fatal("upgrade did not emit a complete root closure and frozen selection", resolution, err)
	}
	if !strings.Contains(string(runner.inputs[1]), "child==2.0\n") || !strings.Contains(string(runner.inputs[1]), "other==9.0\n") || !strings.Contains(string(runner.inputs[1]), "child<3\n") {
		t.Fatal("unchanged environment constraints lost", string(runner.inputs[1]))
	}
	first, second, removed := false, false, false
	for _, call := range runner.calls {
		joined := strings.Join(call.arguments, " ")
		if strings.Contains(joined, install.Target().String()) {
			t.Fatal("original environment exposed to resolver", joined)
		}
		if strings.Contains(joined, pypiUpgradeSelectionScript) {
			first = strings.Contains(joined, "--upgrade --upgrade-strategy only-if-needed") && strings.Contains(joined, "--isolated") && strings.Contains(joined, "changes.json")
		}
		if strings.Contains(joined, "-I -m pip install") {
			second = strings.Contains(joined, "primary==1.0") && strings.Contains(joined, "--ignore-installed --constraint "+pypiResolverProjectDir+"/upgrade-constraints.txt")
		}
		if joined == "rm --force 0123456789abcdef" {
			removed = true
		}
	}
	if !first || !second || !removed || strings.Contains(pypiUpgradeSelectionScript, "site.addsitedir") || strings.Index(pypiUpgradeSelectionScript, "from pip") > strings.Index(pypiUpgradeSelectionScript, "sys.path.insert") {
		t.Fatal("fixed two-phase selection/inert path/cleanup contract missing", first, second, removed)
	}
	selected := map[string][]byte{}
	for _, file := range resolution.ProjectControls() {
		selected[file.Name()] = file.Body()
	}
	if !bytes.Equal(selected[artifactpypi.UpgradeInputControlName], control.Body()) || !bytes.Equal(selected["pypi-update-report.json"], []byte(pypiResolverReportJSON())) || !bytes.Equal(selected["pypi-update-delta.json"], emptyPyPIUpgradeDelta(t)) {
		t.Fatal("final resolution lost exact input/delta/report bytes")
	}
}

func TestPyPIUpgradeResolverRejectsCancelledInvalidAndWrongSourceBeforeResources(t *testing.T) {
	for _, mode := range []string{"cancelled", "wrong-input", "wrong-root", "new-target", "named-source"} {
		t.Run(mode, func(t *testing.T) {
			reference, install, control := frozenPyPIUpgradeFixture(t)
			runner := &pypiUpgradeRecordingRunner{}
			resolver, _ := NewPyPIResolver(runner, pypiStaticEndpoints{}, &recordingObserver{}, availablePythonProbe, &recordingResolverPolicyService{})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "cancelled":
				cancel()
			case "wrong-input":
				control, _ = domain.NewProjectControlFile("wrong.json", control.Body(), true)
			case "wrong-root":
				reference, _ = artifactpypi.ParseReference("different")
			case "new-target":
				install, _ = domain.NewInstallContext(install.Target())
			case "named-source":
				resolver.profile, _ = artifactpypi.PyTorchProfile("cpu")
			}
			resolution, err := resolver.ResolveDependencyUpdate(ctx, reference, install, []domain.ProjectControlFile{control})
			if err == nil || len(runner.calls) != 0 || len(runner.inputCalls) != 0 || len(resolution.Graph().Nodes()) != 0 || mode == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("invalid update started resources or lost cancellation", err, runner.calls)
			}
		})
	}
}

func TestPyTorchUpgradeResolverKeepsNamedDeadlineAndRejectsMissingResourceContext(t *testing.T) {
	for _, name := range []string{"cpu", "cu126", "cu130", "cu132"} {
		for _, present := range []bool{false, true} {
			profile, _ := artifactpypi.PyTorchProfile(name)
			requirement, _ := artifactpypi.ParseUpgradeRequirementForSource("torch", profile.Source())
			input, err := artifactpypi.NewUpgradeInput(requirement, []artifactpypi.InstalledMetadataFile{{Directory: "torch-1.0.dist-info", Body: []byte("Name: torch\nVersion: 1.0\n\n")}}, PinnedPythonRuntime().PythonVersion)
			if err != nil {
				t.Fatal(err)
			}
			control, _ := input.Control()
			runner := &pypiUpgradeRecordingRunner{}
			probed := false
			probe := func(ctx context.Context) (PythonCapability, error) {
				probed = true
				deadline, ok := ctx.Deadline()
				remaining := time.Until(deadline)
				if !ok || remaining > profile.ResourcePolicy().Duration() || remaining < profile.ResourcePolicy().Duration()-time.Second {
					t.Fatal("named update inherited public timeout", remaining)
				}
				return PythonCapability{}, errors.New("controlled pre-resource stop")
			}
			resolver, err := NewPyTorchResolver(runner, pypiStaticEndpoints{}, &recordingObserver{}, probe, &recordingResolverPolicyService{}, profile)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if present {
				ctx, err = artifactpypi.ContextWithResourcePolicy(ctx, profile)
				if err != nil {
					t.Fatal(err)
				}
			}
			target, _ := domain.NewInstallTarget("/tmp/haa-named-deadline-original")
			install, _ := domain.NewPythonVenvInstallContext(target)
			_, err = resolver.ResolveDependencyUpdate(ctx, requirement.Reference(), install, []domain.ProjectControlFile{control})
			if err == nil || probed != present || len(runner.calls) != 0 {
				t.Fatal("profile/deadline preflight failed", err, probed)
			}
		}
	}
}

func TestPyPIUpgradeResolverDeadlinePreservesCauseAndConfirmedCleanup(t *testing.T) {
	reference, install, control := frozenPyPIUpgradeFixture(t)
	runner := &pypiUpgradeRecordingRunner{recordingRunner: recordingRunner{responses: pypiUpgradeResponses(t), waitForContext: true, waitForContextAt: 8}}
	observer := &recordingObserver{reader: &traceReader{records: []TraceRecord{{Kind: "network-attempt", Bytes: 1}}}}
	resolver, _ := NewPyPIResolver(runner, pypiStaticEndpoints{}, observer, availablePythonProbe, &recordingResolverPolicyService{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	resolution, err := resolver.ResolveDependencyUpdate(ctx, reference, install, []domain.ProjectControlFile{control})
	if !errors.Is(err, context.DeadlineExceeded) || len(resolution.Graph().Nodes()) != 0 || len(resolution.ProjectControls()) != 0 {
		t.Fatal("deadline produced a qualifying selection or lost cause", resolution, err)
	}
	removed := false
	for _, call := range runner.calls {
		if strings.Join(call.arguments, " ") == "rm --force 0123456789abcdef" {
			removed = call.bounded
		}
	}
	if !removed {
		t.Fatal("deadline skipped bounded independent cleanup", runner.calls)
	}
}
