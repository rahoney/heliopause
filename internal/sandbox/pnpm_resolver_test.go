package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pnpmCountingEndpoints struct{ calls int }

func (e *pnpmCountingEndpoints) Resolve(context.Context, []string) ([]netip.Addr, error) {
	e.calls++
	return []netip.Addr{netip.MustParseAddr("1.1.1.1")}, nil
}

func pnpmSelectionInputFixture(t *testing.T, intent artifactnpm.PNPMSelectionIntent, selector, manifest string) artifactnpm.PNPMSelectionInput {
	t.Helper()
	c, err := domain.NewProjectControlFile("package.json", []byte(manifest), true)
	if err != nil {
		t.Fatal(err)
	}
	input, err := artifactnpm.NewPNPMSelectionInput(intent, selector, []domain.ProjectControlFile{c}, runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestPNPMResolverRejectsUnsupportedInfrastructureAndPreflight(t *testing.T) {
	for _, runner := range []CommandRunner{nil, &recordingRunner{}, uvStreamOnlyRunner{}} {
		if resolver, err := NewPNPMResolver(runner, &pnpmCountingEndpoints{}, &recordingObserver{}, availableProbe, &recordingResolverPolicyService{}); err == nil || resolver != nil {
			t.Fatal("incomplete transport accepted")
		}
	}
	for _, mode := range []string{"zero input", "canceled", "probe unavailable", "tool unavailable"} {
		t.Run(mode, func(t *testing.T) {
			runner := &pnpmControlTransport{}
			endpoints := &pnpmCountingEndpoints{}
			probeCalls, toolCalls := 0, 0
			cause := errors.New("fixed tool unavailable")
			resolver, err := NewPNPMResolver(runner, endpoints, &recordingObserver{}, func(context.Context) (Capability, error) {
				probeCalls++
				return Capability{Available: mode != "probe unavailable"}, nil
			}, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			resolver.prepare = func(context.Context) (*pnpmPreparedTool, error) { toolCalls++; return nil, cause }
			input := pnpmSelectionInputFixture(t, artifactnpm.PNPMInstall, "", `{}`)
			if mode == "zero input" {
				input = artifactnpm.PNPMSelectionInput{}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "canceled" {
				cancel()
			}
			output, err := resolver.SelectProject(ctx, input)
			if err == nil || len(output.Controls()) != 0 || output.Selection().DependencyFree() || runner.calls != 0 || endpoints.calls != 0 {
				t.Fatal("refused request reached resources or produced selection", err)
			}
			if mode == "canceled" && (!errors.Is(err, context.Canceled) || probeCalls != 0 || toolCalls != 0) {
				t.Fatal("cancellation cause/preflight changed", err)
			}
			if mode == "tool unavailable" && !errors.Is(err, cause) {
				t.Fatal("tool cause lost", err)
			}
		})
	}
}

func TestPNPMSelectionArgumentsKeepFixedClientAndNativeIntents(t *testing.T) {
	for _, test := range []struct {
		intent             artifactnpm.PNPMSelectionIntent
		selector, manifest string
	}{
		{artifactnpm.PNPMAdd, "@scope/probe@^1.0.0", `{}`}, {artifactnpm.PNPMInstall, "", `{}`}, {artifactnpm.PNPMUpdate, "probe", `{"dependencies":{"probe":"^1.0.0"}}`}, {artifactnpm.PNPMUpdate, "", `{}`},
	} {
		input := pnpmSelectionInputFixture(t, test.intent, test.selector, test.manifest)
		args, err := pnpmSelectionArguments(input)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{pnpmToolGuestPath, "--dir", pnpmProjectDirectory, string(test.intent), "--lockfile-only", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state", "--reporter=silent"}
		if test.intent == artifactnpm.PNPMInstall {
			want = append(want, "--no-frozen-lockfile")
		}
		if test.selector != "" {
			want = append(want, "--", test.selector)
		}
		if !reflect.DeepEqual(args, want) {
			t.Fatal("native selection gained implicit arguments", args)
		}
	}
	manifest := pnpmTestControl(t, "package.json", `{}`, true)
	lock := pnpmTestControl(t, "pnpm-lock.yaml", "lockfileVersion: '9.0'\nsettings: {autoInstallPeers: true, excludeLinksFromLockfile: false}\nimporters: {'.': {}}\n", true)
	input, err := artifactnpm.NewPNPMSelectionInput(artifactnpm.PNPMInstall, "", []domain.ProjectControlFile{manifest, lock}, runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	args, err := pnpmSelectionArguments(input)
	if err != nil || args[len(args)-1] != "--frozen-lockfile" {
		t.Fatal("existing reviewed lock not frozen", err)
	}
	if _, err := pnpmSelectionArguments(artifactnpm.PNPMSelectionInput{}); err == nil {
		t.Fatal("zero input became a native invocation")
	}
}

func TestPNPMOwnedContainerCleanupCannotAdoptOtherNativeKind(t *testing.T) {
	const owner = "sbx_owned"
	const name = "haa-pnpm-resolver-" + owner
	id := strings.Repeat("a", 64)
	for _, mode := range []string{"owned", "unknown create", "absent unknown", "query error", "expected missing", "uv image", "uv label", "wrong name", "wrong ID", "ambiguous", "oversize", "malformed", "unknown kind"} {
		t.Run(mode, func(t *testing.T) {
			actual := map[string]any{"Id": id, "Name": "/" + name, "Config": map[string]any{"Image": nodeImageReference, "Labels": map[string]string{pnpmResolverOwnerLabel: owner}}}
			runner := &uvOwnedContainerRunner{inventory: []byte(id + "\n")}
			expected := id
			kind := nativeRuntimePNPM
			switch mode {
			case "unknown create":
				expected = ""
			case "absent unknown":
				expected = ""
				runner.inventory = nil
			case "query error":
				runner.queryErr = errors.New("query failed")
			case "expected missing":
				runner.inventory = nil
			case "uv image":
				actual["Config"].(map[string]any)["Image"] = pythonImageReference
			case "uv label":
				actual["Config"].(map[string]any)["Labels"] = map[string]string{uvResolverOwnerLabel: owner}
			case "wrong name":
				actual["Name"] = "/haa-uv-resolver-" + owner
			case "wrong ID":
				actual["Id"] = strings.Repeat("b", 64)
			case "ambiguous":
				runner.inventory = append(runner.inventory, []byte(strings.Repeat("b", 64))...)
			case "unknown kind":
				kind = nativeRuntimeToolKind(99)
			}
			body, err := json.Marshal(actual)
			if err != nil {
				t.Fatal(err)
			}
			runner.inspection = body
			if mode == "oversize" {
				runner.inspection = append(body, make([]byte, 32<<10)...)
			}
			if mode == "malformed" {
				runner.inspection = []byte(`{"Id":`)
			}
			err = removeOwnedNativeResolverContainer(context.Background(), runner, name, owner, expected, kind)
			accepted := mode == "owned" || mode == "unknown create" || mode == "absent unknown"
			if (err == nil) != accepted {
				t.Fatal("ownership decision changed", err)
			}
			if mode == "owned" || mode == "unknown create" {
				if runner.removed != id {
					t.Fatal("owned resource not removed")
				}
			} else if runner.removed != "" {
				t.Fatal("foreign/unknown resource was removed")
			}
		})
	}
}
