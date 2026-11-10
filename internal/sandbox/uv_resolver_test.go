package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvStreamOnlyRunner struct{}

func (uvStreamOnlyRunner) Output(context.Context, string, ...string) ([]byte, error)     { return nil, nil }
func (uvStreamOnlyRunner) RunOutput(context.Context, io.Writer, string, ...string) error { return nil }

type uvCountingEndpoints struct{ calls int }

func (e *uvCountingEndpoints) Resolve(ctx context.Context, names []string) (map[string][]netip.Addr, error) {
	e.calls++
	return (pypiStaticEndpoints{}).Resolve(ctx, names)
}

func TestUVResolverRequiresActualStreamCapabilities(t *testing.T) {
	for _, runner := range []CommandRunner{nil, &recordingRunner{}, uvStreamOnlyRunner{}} {
		if resolver, err := NewUVResolver(runner, pypiStaticEndpoints{}, &recordingObserver{}, availablePythonProbe, &recordingResolverPolicyService{}); err == nil || resolver != nil {
			t.Fatal("unsupported transport accepted")
		}
	}
	_, runner := uvReaderFixture(t)
	resolver, err := NewUVResolver(runner, pypiStaticEndpoints{}, &recordingObserver{}, availablePythonProbe, &recordingResolverPolicyService{})
	if err != nil || resolver == nil || resolver.Close() != nil {
		t.Fatal("complete injected infrastructure rejected", err)
	}
}

func TestUVResolverRefusesBeforeNetworkOrUntrustedToolExecution(t *testing.T) {
	for _, outcome := range []string{"source", "empty reference", "canceled", "runtime unavailable", "untrusted tool"} {
		t.Run(outcome, func(t *testing.T) {
			_, runner := uvReaderFixture(t)
			endpoints := &uvCountingEndpoints{}
			probe := availablePythonProbe
			if outcome == "runtime unavailable" {
				probe = func(context.Context) (PythonCapability, error) {
					return PythonCapability{}, errors.New("runtime unavailable")
				}
			}
			resolver, err := NewUVResolver(runner, endpoints, &recordingObserver{reader: &traceReader{}}, probe, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			reference, _ := artifactpypi.ParseReference("probe@1.0")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch outcome {
			case "source":
				profile, _ := artifactpypi.PyTorchProfile("cpu")
				reference, _ = artifactpypi.ParseReferenceForSource("torch@2.9.1+cpu", profile.Source())
			case "empty reference":
				reference = domain.ArtifactReference{}
			case "canceled":
				cancel()
			case "untrusted tool":
				runner.responses = [][]byte{[]byte("changed immutable image identity")}
			}
			result, err := resolver.ResolveDependencies(ctx, reference, domain.InstallContext{})
			if err == nil || len(result.Graph().Nodes()) != 0 || endpoints.calls != 0 || len(runner.inputCalls) != 0 {
				t.Fatal("invalid request produced graph or reached network/native execution", err)
			}
			if outcome == "untrusted tool" {
				if !errors.Is(err, errUVToolSupply) || len(runner.calls) != 1 || runner.calls[0].arguments[0] != "image" {
					t.Fatal("immutable tool preflight was bypassed", err)
				}
			} else if len(runner.calls) != 0 {
				t.Fatal("refused request mutated Docker")
			}
		})
	}
}

type uvOwnedContainerRunner struct {
	inventory  []byte
	inspection []byte
	queryErr   error
	removed    string
}

func (r *uvOwnedContainerRunner) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	switch args[0] {
	case "ps":
		return r.inventory, r.queryErr
	case "inspect":
		return r.inspection, nil
	case "rm":
		r.removed = args[len(args)-1]
		return nil, nil
	}
	return nil, errors.New("unexpected command")
}

func TestUVResolverCleanupNeverAdoptsUnknownOrForeignContainer(t *testing.T) {
	const owner = "sbx_owned"
	const name = "haa-uv-resolver-" + owner
	id := strings.Repeat("a", 64)
	for _, outcome := range []string{"owned", "creation output missing", "absent before identity", "query failed", "missing expected", "foreign owner", "wrong image", "different ID", "ambiguous inventory"} {
		t.Run(outcome, func(t *testing.T) {
			actual := map[string]any{"Id": id, "Name": "/" + name, "Config": map[string]any{"Image": pythonImageReference, "Labels": map[string]string{uvResolverOwnerLabel: owner}}}
			runner := &uvOwnedContainerRunner{inventory: []byte(id + "\n")}
			expected := id
			switch outcome {
			case "creation output missing":
				expected = ""
			case "absent before identity":
				expected = ""
				runner.inventory = nil
			case "query failed":
				runner.queryErr = errors.New("inventory unavailable")
			case "missing expected":
				runner.inventory = nil
			case "foreign owner":
				actual["Config"].(map[string]any)["Labels"] = map[string]string{uvResolverOwnerLabel: "other"}
			case "wrong image":
				actual["Config"].(map[string]any)["Image"] = "other-image"
			case "different ID":
				actual["Id"] = strings.Repeat("b", 64)
			case "ambiguous inventory":
				runner.inventory = append(runner.inventory, []byte(strings.Repeat("b", 64)+"\n")...)
			}
			var err error
			runner.inspection, err = json.Marshal(actual)
			if err != nil {
				t.Fatal(err)
			}
			err = removeOwnedUVResolverContainer(context.Background(), runner, name, owner, expected)
			accepted := outcome == "owned" || outcome == "creation output missing" || outcome == "absent before identity"
			if (err == nil) != accepted {
				t.Fatal("cleanup identity decision differs", err)
			}
			if outcome == "owned" || outcome == "creation output missing" {
				if runner.removed != id {
					t.Fatal("owned container not removed")
				}
			} else if runner.removed != "" {
				t.Fatal("uncertain or foreign container was removed")
			}
		})
	}
}

func TestUVNativeTargetUsesOnlyLockedPythonPlatform(t *testing.T) {
	runtime := PinnedPythonRuntime()
	if platform, err := uvTargetPlatform(runtime); err != nil || platform != "x86_64-manylinux_2_36" {
		t.Fatal("locked target conversion differs", platform, err)
	}
	runtime.PlatformTag = "win_amd64"
	if _, err := uvTargetPlatform(runtime); err == nil {
		t.Fatal("foreign target accepted")
	}
}
