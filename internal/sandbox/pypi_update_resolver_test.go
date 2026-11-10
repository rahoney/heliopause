package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
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
		[]byte(runtimeidentity.PythonVersion + "\n"), []byte("pip " + runtimeidentity.PipVersion + " from pip\n"), []byte("cp314-cp314-manylinux_2_36_x86_64\n"), nil,
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

func TestRequirementsParserCopiesFrozenDataWithoutUserProgramArguments(t *testing.T) {
	input, err := artifactpypi.NewRequirementsInput(nil, nil, []string{"alpha==1.0"})
	if err != nil {
		t.Fatal(err)
	}
	control, _ := input.Control()
	parserControl, _ := input.ParserControl()
	textHash := sha256.Sum256([]byte("alpha==1.0"))
	body, err := json.Marshal(map[string]any{
		"version": 1, "input_sha256": control.Digest().String(), "parser_sha256": parserControl.Digest().String(), "python": runtimeidentity.PythonVersion, "pip": runtimeidentity.PipVersion, "require_hashes": false,
		"clauses": []any{map[string]any{"requirement": "alpha==1.0", "project": "alpha", "extras": []string{}, "constraint": false, "hashes": map[string][]string{}, "active": true, "source": "@positional", "line": 1, "text_sha256": hex.EncodeToString(textHash[:])}},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{responses: [][]byte{body}}
	resolver := &PyPIResolver{runner: runner}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	parsed, err := resolver.parseRequirementsInSession(ctx, "0123456789abcdef", input)
	if err != nil || !parsed.Valid() || parsed.InputDigest() != control.Digest() {
		t.Fatal("frozen parser transport failed", err)
	}
	if len(runner.inputCalls) != 1 || !bytes.Equal(runner.input, parserControl.Body()) || len(runner.calls) != 1 {
		t.Fatal("parser did not consume one frozen input")
	}
	if strings.Contains(strings.Join(runner.calls[0].arguments, "\n"), "alpha==1.0") {
		t.Fatal("user requirement became executable program arguments")
	}
}

func TestRequirementsParserRejectsBeforeCommands(t *testing.T) {
	input, err := artifactpypi.NewRequirementsInput(nil, nil, []string{"alpha"})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"nil context", "canceled", "missing container", "missing input"} {
		runner := &recordingRunner{}
		resolver := &PyPIResolver{runner: runner}
		ctx := context.Background()
		container := "0123456789abcdef"
		source := input
		switch kind {
		case "nil context":
			ctx = nil
		case "canceled":
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			ctx = canceled
		case "missing container":
			container = ""
		case "missing input":
			source = artifactpypi.RequirementsInput{}
		}
		if _, err := resolver.parseRequirementsInSession(ctx, container, source); err == nil {
			t.Fatal("invalid parser request accepted", kind)
		}
		if len(runner.calls) != 0 || len(runner.inputCalls) != 0 {
			t.Fatal("invalid parser request ran a command", kind)
		}
	}
}

func TestRequirementsParserPreservesCancellationWithoutReplayingDiagnostic(t *testing.T) {
	input, err := artifactpypi.NewRequirementsInput(nil, nil, []string{"alpha"})
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{errors: []error{context.Canceled}}
	resolver := &PyPIResolver{runner: runner}
	_, err = resolver.parseRequirementsInSession(context.Background(), "0123456789abcdef", input)
	if !errors.Is(err, context.Canceled) || err.Error() != "pinned requirements grammar validation failed" {
		t.Fatal("parser lost its cancellation cause or replayed diagnostics", err)
	}
}

func TestRequirementsPipSelectionKeepsMultipleRootsInOneNativeRequest(t *testing.T) {
	for _, kind := range []string{"ordinary", "upgrade", "second-resolution-canceled", "omitted-final-root"} {
		t.Run(kind, func(t *testing.T) {
			source, err := artifactpypi.NewRequirementsInput(nil, nil, []string{"primary==1.0", "child==2.0"})
			if err != nil {
				t.Fatal(err)
			}
			source, err = source.WithInstallation("pip", kind == "upgrade", nil)
			if err != nil {
				t.Fatal(err)
			}
			control, _ := source.Control()
			parserControl, _ := source.ParserControl()
			clauses := []any{}
			for index, entry := range []struct{ text, project string }{{"primary==1.0", "primary"}, {"child==2.0", "child"}} {
				sum := sha256.Sum256([]byte(entry.text))
				clauses = append(clauses, map[string]any{"active": true, "source": "@positional", "line": index + 1, "text_sha256": hex.EncodeToString(sum[:]), "requirement": entry.text, "project": entry.project, "extras": []string{}, "constraint": false, "hashes": map[string][]string{}})
			}
			grammar, _ := json.Marshal(map[string]any{"version": 1, "input_sha256": control.Digest().String(), "parser_sha256": parserControl.Digest().String(), "python": runtimeidentity.PythonVersion, "pip": runtimeidentity.PipVersion, "require_hashes": false, "clauses": clauses})
			report := []byte(strings.ReplaceAll(pypiResolverReportJSON(), `"requested":false`, `"requested":true`))
			final := report
			if kind == "omitted-final-root" {
				final = []byte(pypiResolverReportJSON())
			}
			runner := &recordingRunner{responses: [][]byte{grammar, nil, report, nil, final}}
			if kind == "second-resolution-canceled" {
				runner.errors = []error{nil, nil, nil, context.Canceled}
			}
			resolver := &PyPIResolver{runner: runner, profile: artifactpypi.PublicPyPIProfile()}
			result, err := resolver.selectPipRequirements(context.Background(), "0123456789abcdef", source)
			if kind == "second-resolution-canceled" {
				if !errors.Is(err, context.Canceled) || len(result.report) != 0 {
					t.Fatal("failed complete selection returned data or lost cause", err)
				}
				return
			}
			if kind == "omitted-final-root" {
				if err == nil {
					t.Fatal("missing real root accepted")
				}
				return
			}
			if err != nil || !bytes.Equal(result.report, report) || !result.input.HasActiveRoots() {
				t.Fatal("multi-root native selection failed", err)
			}
			first := strings.Join(runner.calls[1].arguments, "\n")
			second := strings.Join(runner.calls[3].arguments, "\n")
			if strings.Contains(first, "primary==") || strings.Contains(second, "child==") || !strings.Contains(first, "requirements-roots.txt") || !strings.Contains(second, "--ignore-installed") {
				t.Fatal("requirements escaped data files or complete selection was omitted")
			}
			if strings.Contains(first, "--upgrade") != (kind == "upgrade") {
				t.Fatal("ordinary install changed to upgrade")
			}
			if len(runner.inputCalls) != 4 {
				t.Fatal("roots were split into separate native installations")
			}
		})
	}
}
