package sandbox

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type cargoResolverTestEndpoints struct{}

func (cargoResolverTestEndpoints) Resolve(context.Context, []string) (map[string][]netip.Addr, error) {
	return map[string][]netip.Addr{"index.crates.io": {netip.MustParseAddr("1.1.1.1")}, "static.crates.io": {netip.MustParseAddr("1.1.1.2")}}, nil
}
func availableCargoProbe(context.Context) (CargoCapability, error) {
	return CargoCapability{Available: true, Runtime: PinnedCargoRuntime()}, nil
}

func TestIsolatedCargoResolverFailsClosedBeforeControlPublication(t *testing.T) {
	for _, test := range []struct {
		name, kind                 string
		commandError, cleanupError bool
	}{
		{"complete", "", false, false},
		{"unexpected child", "process-exec-unexpected", false, false},
		{"outside filesystem", "filesystem-outside-workspace", false, false},
		{"honeytoken", "honeytoken-access", false, false},
		{"network attempt", "network-attempt", false, false},
		{"command failure", "", true, false},
		{"cleanup failure", "", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := cargoSourceFixture(t)
			manifest, err := os.ReadFile(filepath.Join(workspace, "Cargo.toml"))
			if err != nil {
				t.Fatal(err)
			}
			selected := append(append([]byte(nil), manifest...), []byte("[dependencies]\nitoa='=1.0.17'\n")...)
			lock := []byte("version=4\n[[package]]\nname='local'\nversion='0.1.0'\n")
			runner := &goResolverTestRunner{recordingRunner: recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789abcdef"), nil, nil}}, outputQueue: [][]byte{[]byte("cargo " + PinnedCargoRuntime().RustVersion + " (fixed-query)\n"), []byte(""), selected, lock}}
			if test.commandError {
				runner.outputError = errors.New("artifact stderr /private/secret")
			}
			if test.cleanupError {
				runner.errors = []error{nil, nil, nil, nil, nil, errors.New("cleanup denied")}
			}
			trace := &traceReader{}
			if test.kind != "" {
				trace.records = []TraceRecord{{Kind: test.kind, Bytes: 1}}
			}
			service := &recordingResolverPolicyService{}
			isolated, err := newIsolatedCargoRunner(runner, cargoResolverTestEndpoints{}, &recordingObserver{reader: trace}, availableCargoProbe, service)
			if err != nil {
				t.Fatal(err)
			}
			env, _ := CargoResolverEnvironmentForHome("/private/operation-home")
			body, err := isolated.RunCargo(context.Background(), workspace, env, "add", "itoa@=1.0.17")
			wantError := test.name != "complete"
			if (err != nil) != wantError || (wantError && body != nil) {
				t.Fatalf("body=%q error=%v", body, err)
			}
			if service.remove != 1 {
				t.Fatal("network cleanup omitted")
			}
			current, readErr := os.ReadFile(filepath.Join(workspace, "Cargo.toml"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if wantError {
				if string(current) != string(manifest) {
					t.Fatal("failed selection published manifest")
				}
				if _, err := os.Lstat(filepath.Join(workspace, "Cargo.lock")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed selection published lock")
				}
				if strings.Contains(err.Error(), "/private/secret") {
					t.Fatal("raw command stderr escaped trusted diagnostic boundary")
				}
			} else if string(current) != string(selected) {
				t.Fatal("selected manifest missing")
			}
		})
	}
}

func TestIsolatedCargoResolverRejectsAmbientCommandsAndEndpoints(t *testing.T) {
	for _, command := range [][]string{{"build"}, {"test"}, {"run"}, {"metadata", "--format-version", "1"}, {"add", "itoa@1.0.17"}, {"add", "itoa@=latest"}, {"add", "--git=https://example.invalid/repo"}, {"metadata", "--locked", "--format-version", "1", "--config", "evil"}} {
		runner := &goResolverTestRunner{}
		isolated, _ := newIsolatedCargoRunner(runner, cargoResolverTestEndpoints{}, &recordingObserver{}, availableCargoProbe, &recordingResolverPolicyService{})
		if _, err := isolated.RunCargo(context.Background(), "/invalid", CargoResolverEnvironment(), command...); err == nil || len(runner.timeline) != 0 {
			t.Fatal("unsupported command reached runtime")
		}
	}
	for _, endpoints := range []map[string][]netip.Addr{
		{"index.crates.io": {netip.MustParseAddr("1.1.1.1")}},
		{"index.crates.io": {netip.MustParseAddr("127.0.0.1")}, "static.crates.io": {netip.MustParseAddr("1.1.1.2")}},
		{"index.crates.io": {netip.MustParseAddr("1.1.1.1")}, "evil.example": {netip.MustParseAddr("1.1.1.2")}},
		{"index.crates.io": {netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.1.1.1")}, "static.crates.io": {netip.MustParseAddr("1.1.1.2")}},
	} {
		if _, _, err := cargoResolverNetworkArguments(endpoints); err == nil {
			t.Fatal("unsafe endpoints accepted")
		}
	}
	args := strings.Join(cargoResolverCreateArguments("owned-network", nil), "\n")
	for _, required := range []string{PinnedCargoRuntime().ImageReference, "--read-only", "--runtime\nrunsc", "--cpus\n1", "--memory\n512m", "RUSTC=" + PinnedCargoRuntime().RustcBinary(), "CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse"} {
		if !strings.Contains(args, required) {
			t.Fatalf("missing fixed input %q", required)
		}
	}
	for _, unsafe := range []string{"--mount", "--volume", "--privileged", "RUSTC_WRAPPER", "CARGO_HOME=/root"} {
		if strings.Contains(args, unsafe) {
			t.Fatalf("ambient input %q", unsafe)
		}
	}
}

func TestCargoRunnerFailureRetainsOnlyTrustedDiagnostics(t *testing.T) {
	raw := errors.New("artifact stderr /private/secret")
	trusted := &isolatedCargoResolverFailure{cause: raw, phase: "COMMAND", diagnostic: TraceDiagnostic{Reason: "STREAM_FAULT", FaultSite: "OPEN_RESULT_CLASSIFICATION_IMAGE"}}
	if !strings.Contains(trusted.Error(), "phase=COMMAND") || !strings.Contains(trusted.Error(), "STREAM_FAULT") || strings.Contains(trusted.Error(), "/private/secret") || !errors.Is(trusted, raw) {
		t.Fatal("first trusted failure lost or raw text exposed")
	}
}

func TestCargoObserverProfileRequiresRegisteredTopology(t *testing.T) {
	topology, ok := observerExpectedTopology(cargoResolverProfile)
	if !ok || !validObserverProfile(cargoResolverProfile) || len(topology) != 3 || !topology[0].ReadOnly || !topology[1].NoExec {
		t.Fatal("cargo profile topology and registration disagree")
	}
	for _, profile := range []string{"cargo", "cargo-build-untrusted", "cargo-resolver-untrusted", "CARGO-RESOLVER"} {
		if validObserverProfile(profile) {
			t.Fatal("artifact profile alias admitted")
		}
	}
	body, err := os.ReadFile("../../tools/gvisor-observer/observer.cc")
	if err != nil || !strings.Contains(string(body), `constexpr char kCargoBinary[] = "`+PinnedCargoRuntime().CargoBinary()+`";`) {
		t.Fatal("observer Cargo image differs from runtime lock")
	}
}
