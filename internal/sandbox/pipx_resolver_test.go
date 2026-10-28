package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

// Embed only the Output port so the underlying test runner's RunBounded is
// deliberately absent while every other required transport remains present.
type pipxWithoutBoundedRunner struct{ CommandRunner }

func (r pipxWithoutBoundedRunner) RunInput(ctx context.Context, input io.Reader, name string, args ...string) error {
	return r.CommandRunner.(*pipxNativeNoCommands).RunInput(ctx, input, name, args...)
}
func (r pipxWithoutBoundedRunner) RunOutput(ctx context.Context, output io.Writer, name string, args ...string) error {
	return r.CommandRunner.(*pipxNativeNoCommands).RunOutput(ctx, output, name, args...)
}
func (r pipxWithoutBoundedRunner) RunDiscard(ctx context.Context, name string, args ...string) error {
	_, err := r.Output(ctx, name, args...)
	return err
}

func TestPipxResolverRequiresNativeCapabilitiesBeforeCommands(t *testing.T) {
	for _, scenario := range []string{"resources", "runner", "profile", "discard", "bounded"} {
		t.Run(scenario, func(t *testing.T) {
			count := &pipxNativeNoCommands{}
			var runner CommandRunner = &pipxNativeDiscardCommands{count}
			var observer TraceObserver = pipxNativeUnusedObserver{}
			var resources ObservationResourceClient = pipxNativeUnusedResources{}
			switch scenario {
			case "resources":
				resources = nil
			case "runner":
				runner = nil
			case "profile":
				observer = &pipxNativeUnprofiledObserver{}
			case "discard":
				runner = count
			case "bounded":
				runner = pipxWithoutBoundedRunner{count}
			}
			resolver, err := NewPipxResolver(runner, pypiStaticEndpoints{}, observer, availablePythonProbe, &recordingResolverPolicyService{}, resources)
			if err == nil || resolver != nil || count.calls != 0 {
				t.Fatal("incomplete native infrastructure admitted", err, count.calls)
			}
		})
	}
}

func TestPipxPublicResolverRefusesInvalidRequestAndCancellationBeforeIO(t *testing.T) {
	for _, scenario := range []string{"empty", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			count := &pipxNativeNoCommands{}
			endpoints := &uvCountingEndpoints{}
			resolver, err := NewPipxResolver(&pipxNativeDiscardCommands{count}, endpoints, pipxNativeUnusedObserver{}, availablePythonProbe, &recordingResolverPolicyService{}, pipxNativeUnusedResources{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var requirement artifactpypi.UpgradeRequirement
			if scenario == "cancelled" {
				requirement, _ = artifactpypi.ParseUpgradeRequirement("probe==1.0")
				cancel()
			}
			selection, err := resolver.ResolveToolInstall(ctx, requirement)
			if err == nil || len(selection.Controls()) != 0 || len(selection.Resolution().Graph().Nodes()) != 0 || endpoints.calls != 0 || count.calls != 0 {
				t.Fatal("refusal performed IO or retained a selection", err)
			}
			if scenario == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost", err)
			}
			if err := resolver.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPipxSelectionBindsFrozenSourceNativeReceiptAndCopiedControls(t *testing.T) {
	frozen, entries := pipxNativeOutputFixture(t)
	// This fixture originally requests probe<2,>=1. Match the exact request
	// under test without changing the shared original/layout fixture.
	name := "pipx/venvs/probe/pipx_metadata.json"
	entry := entries[name]
	var wire map[string]any
	if err := json.Unmarshal(entry.body, &wire); err != nil {
		t.Fatal(err)
	}
	wire["main_package"].(map[string]any)["package_or_url"] = "probe==1.0"
	var marshalErr error
	entry.body, marshalErr = json.Marshal(wire)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	entry.header.Size = int64(len(entry.body))
	entries[name] = entry
	output, err := readPipxNativeOutput(context.Background(), &pipxNativeOutputTransport{body: uvToolFixtureTar(t, entries)}, strings.Repeat("a", 64), "probe", frozen)
	if err != nil {
		t.Fatal(err)
	}
	file := frozen.files[0]
	lock := []byte(fmt.Sprintf("lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=%s'\n[[packages]]\nname='probe'\nversion='1.0'\nwheels=[{url=%q,size=%d,hashes={sha256=%q}}]\n", PinnedPythonRuntime().PythonVersion, file.url, len(file.body), file.digest))
	requirement, _ := artifactpypi.ParseUpgradeRequirement("probe==1.0")
	// Production uses the source-only Reference and retains constraints in the
	// typed requirement. Refreeze this shared exact-reference fixture likewise.
	parsed, err := artifactpypi.ParseUVLock(requirement.Reference(), lock, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	var inputs []artifactpypi.UVWheelInput
	for _, file := range frozen.files {
		inputs = append(inputs, artifactpypi.UVWheelInput{URL: file.url, Reader: bytes.NewReader(file.body), Size: int64(len(file.body))})
	}
	frozen, err = freezePipxNativeInput(context.Background(), parsed, inputs, frozen.pages)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := pipxSelectionForOutput(requirement, lock, frozen, output)
	if err != nil {
		t.Fatal(err)
	}
	controls := selection.Controls()
	names := []string{"pipx-tool-intent.json", "pipx-tool-receipt.json", "pipx-tool-pylock.toml", "pipx-tool-census.json"}
	var digests []string
	for i, control := range controls {
		if i >= len(names) || control.Name() != names[i] || !control.Present() || len(control.Body()) == 0 {
			t.Fatal("native provenance incomplete")
		}
		digests = append(digests, control.Name()+":"+control.Digest().String())
	}
	if len(controls) != 4 || len(selection.Resolution().Graph().Nodes()) != 1 || selection.Resolution().RuntimeIdentity() != "pipx-tool-install:"+PipxResolverRuntimeIdentity() {
		t.Fatal("native selection identity differs")
	}
	body, _ := json.Marshal(digests)
	sum := sha256.Sum256(body)
	if selection.Resolution().LockfileDigest().String() != hex.EncodeToString(sum[:]) {
		t.Fatal("ordered provenance digest differs")
	}
	original := controls[0].Body()
	changed := controls[0].Body()
	changed[0] ^= 1
	controls[0] = controls[1]
	if string(selection.Controls()[0].Body()) != string(original) {
		t.Fatal("caller mutated held provenance")
	}
	for _, scenario := range []string{"lock", "metadata", "request", "frozen"} {
		t.Run(scenario, func(t *testing.T) {
			candidate := output
			candidate.controls = append([]artifactpypi.UVInstalledWheelControls(nil), output.controls...)
			candidateLock := append([]byte(nil), lock...)
			candidateFrozen := frozen
			request := requirement
			switch scenario {
			case "lock":
				candidateLock = []byte("invalid")
			case "metadata":
				candidate.controls[0].Metadata = append(append([]byte(nil), candidate.controls[0].Metadata...), []byte("Summary: forged\n")...)
			case "request":
				request, _ = artifactpypi.ParseUpgradeRequirement("probe==2.0")
			case "frozen":
				candidateFrozen = nil
			}
			refused, err := pipxSelectionForOutput(request, candidateLock, candidateFrozen, candidate)
			if err == nil || len(refused.Controls()) != 0 || len(refused.Resolution().Graph().Nodes()) != 0 {
				t.Fatal("substituted provenance retained native selection", err)
			}
		})
	}
}
