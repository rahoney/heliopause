package sandbox

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"strings"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

type goResolverTestRunner struct {
	recordingRunner
	outputQueue [][]byte
	outputError error
}

func (r *goResolverTestRunner) RunOutput(_ context.Context, writer io.Writer, binary string, arguments ...string) error {
	r.timeline = append(r.timeline, commandCall{binary: binary, arguments: append([]string(nil), arguments...)})
	if r.outputError != nil {
		return r.outputError
	}
	if len(r.outputQueue) == 0 {
		return errors.New("missing output fixture")
	}
	body := r.outputQueue[0]
	r.outputQueue = r.outputQueue[1:]
	_, err := writer.Write(body)
	return err
}

type goResolverTestEndpoints struct{}

func (goResolverTestEndpoints) Resolve(context.Context, []string) (map[string][]netip.Addr, error) {
	return map[string][]netip.Addr{"proxy.golang.org": {netip.MustParseAddr("1.1.1.1")}, "sum.golang.org": {netip.MustParseAddr("1.1.1.2")}}, nil
}
func availableGoProbe(context.Context) (GoCapability, error) {
	return GoCapability{Available: true, Runtime: PinnedGoRuntime()}, nil
}

func TestIsolatedGoResolverFailsClosedBeforeControlPublication(t *testing.T) {
	mod := []byte("module example.com/project\n\ngo 1.26\n")
	sum := []byte("example.com/existing v1.0.0 h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n")
	for _, test := range []struct {
		name         string
		traceError   error
		record       string
		commandError bool
		removeError  bool
		wantError    bool
	}{
		{"complete", nil, "filesystem-workspace-access", false, false, false},
		{"observer incomplete", errors.New("unavailable"), "", false, false, true},
		{"unexpected child", nil, "process-exec-unexpected", false, false, true},
		{"outside filesystem", nil, "filesystem-outside-workspace", false, false, true},
		{"honeytoken", nil, "honeytoken-access", false, false, true},
		{"untrusted network", nil, "network-attempt", false, false, true},
		{"command failure", nil, "", true, false, true},
		{"cleanup failure", nil, "", false, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace, cleanup, err := privateGoProjectWorkspace(mod, sum)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			env, cleanupEnv, err := privateGoResolverEnvironment()
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupEnv()
			runner := &goResolverTestRunner{recordingRunner: recordingRunner{responses: [][]byte{[]byte("0123456789abcdef"), []byte("172.30.0.0/24"), []byte("0123456789abcdef"), nil, nil}}, outputQueue: [][]byte{[]byte("go version go" + PinnedGoRuntime().GoVersion + " linux/amd64\n"), []byte("frozen output"), mod, sum}}
			if test.commandError {
				runner.outputError = errors.New("untrusted command diagnostic")
			}
			if test.removeError {
				runner.errors = []error{nil, nil, nil, nil, nil, errors.New("cleanup denied")}
			}
			trace := &traceReader{err: test.traceError}
			if test.traceError != nil {
				trace.errAfter = 0
			}
			if test.record != "" {
				trace.records = []TraceRecord{{Kind: test.record, Bytes: 1}}
			}
			service := &recordingResolverPolicyService{}
			isolated, err := newIsolatedGoModuleRunner(runner, goResolverTestEndpoints{}, &recordingObserver{reader: trace}, availableGoProbe, service)
			if err != nil {
				t.Fatal(err)
			}
			got, err := isolated.RunGo(context.Background(), workspace, env, "mod", "graph")
			if (err != nil) != test.wantError || (test.wantError && len(got) != 0) {
				t.Fatalf("output=%q error=%v", got, err)
			}
			if service.remove != 1 {
				t.Fatal("network cleanup omitted")
			}
			if test.wantError {
				currentMod, currentSum, err := readGoProjectControlFiles(workspace)
				if err != nil || string(currentMod) != string(mod) || string(currentSum) != string(sum) {
					t.Fatal("failed resolver published controls")
				}
			}
		})
	}
}

func TestIsolatedGoResolverRejectsAmbientCommandsAndEndpoints(t *testing.T) {
	for _, command := range [][]string{{"run", "."}, {"test", "./..."}, {"generate", "./..."}, {"install", "example.com/tool@v1.0.0"}, {"get", "-toolexec=evil"}, {"get", "example.com/private@latest"}, {"mod", "download", "-json", "all", "extra"}} {
		runner := &goResolverTestRunner{}
		isolated, _ := newIsolatedGoModuleRunner(runner, goResolverTestEndpoints{}, &recordingObserver{}, availableGoProbe, &recordingResolverPolicyService{})
		if _, err := isolated.RunGo(context.Background(), "/invalid", artifactgomodule.ResolverEnvironment(), command...); err == nil || len(runner.timeline) != 0 {
			t.Fatal("unsupported command crossed runtime boundary")
		}
	}
	for _, endpoints := range []map[string][]netip.Addr{
		{"proxy.golang.org": {netip.MustParseAddr("1.1.1.1")}},
		{"proxy.golang.org": {netip.MustParseAddr("127.0.0.1")}, "sum.golang.org": {netip.MustParseAddr("1.1.1.2")}},
		{"proxy.golang.org": {netip.MustParseAddr("1.1.1.1")}, "evil.example": {netip.MustParseAddr("1.1.1.2")}},
	} {
		if _, _, err := goResolverNetworkArguments(endpoints); err == nil {
			t.Fatal("unsafe network input accepted")
		}
	}
	w := &goResolverBoundedOutput{limit: 4}
	if _, err := w.Write([]byte("12345")); err == nil || !w.exceeded || w.Len() != 0 {
		t.Fatal("excess output accepted")
	}
	arguments := strings.Join(goResolverCreateArguments("owned-network", nil), "\n")
	for _, unsafe := range []string{"--mount", "GOPROXY=direct", "GOVCS=*:all", "GOWORK=auto"} {
		if strings.Contains(arguments, unsafe) {
			t.Fatal("ambient runtime input inherited")
		}
	}
}

func TestGoObserverProfileRequiresRegisteredTopology(t *testing.T) {
	topology, ok := observerExpectedTopology(goResolverProfile)
	if !ok || !validObserverProfile(goResolverProfile) || len(topology) != 3 || !topology[0].ReadOnly || !topology[1].NoExec {
		t.Fatal("go profile topology and registration disagree")
	}
	for _, profile := range []string{"go", "go-build", "go-module-resolver-untrusted", "GO-MODULE-RESOLVER"} {
		if validObserverProfile(profile) {
			t.Fatal("artifact-selected profile admitted")
		}
	}
}

func TestGoModuleRunnerFailureRetainsOnlyTrustedDiagnostics(t *testing.T) {
	raw := errors.New("artifact-controlled stderr /private/secret")
	trusted := &isolatedGoResolverFailure{cause: raw, phase: "COMMAND", diagnostic: TraceDiagnostic{Reason: "EVENT_LIMIT", FaultSite: "EVENT_LIMIT", FaultBudget: FaultBudgetDiagnostic{Charged: 10000, Limit: 10000, Fcntl: 7379, Close: 2292, Other: 329}}}
	wrapped := goModuleRunnerFailure("private go module selection failed", trusted)
	if !strings.Contains(wrapped.Error(), "phase=COMMAND") || !strings.Contains(wrapped.Error(), "budget_charged=10000 budget_limit=10000") || strings.Contains(wrapped.Error(), "/private/secret") || !errors.Is(wrapped, raw) {
		t.Fatal("trusted first failure lost or raw diagnostic exposed")
	}
	untrusted := goModuleRunnerFailure("go module download failed", raw)
	if untrusted.Error() != "go module download failed" || errors.Is(untrusted, raw) {
		t.Fatal("Host/stub error acquired a trusted diagnostic channel")
	}
}
