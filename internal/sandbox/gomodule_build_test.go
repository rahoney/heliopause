package sandbox

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

type goBuildSequenceRunner struct {
	calls  [][]string
	roles  []byte
	failAt int
}

func (r *goBuildSequenceRunner) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected command")
}
func (r *goBuildSequenceRunner) RunOutput(_ context.Context, output io.Writer, _ string, arguments ...string) error {
	r.calls = append(r.calls, append([]string(nil), arguments...))
	if len(r.calls) == r.failAt {
		return errors.New("injected command failure")
	}
	if len(r.calls) == 2 {
		_, err := output.Write(r.roles)
		return err
	}
	return nil
}

func TestGoBuildSelectedPackageCoveragePrecedesOutput(t *testing.T) {
	for _, test := range []struct {
		name, roles       string
		failAt, wantCalls int
		wantError         bool
	}{
		{"main", "main\n", 0, 3, false}, {"library", "library\n", 0, 2, false}, {"mixed", "main\nlibrary\n", 0, 3, false},
		{"compile failure", "main\n", 1, 1, true}, {"query failure", "main\n", 2, 2, true}, {"output failure", "main\n", 3, 3, true},
		{"empty selection", "", 0, 2, true}, {"invalid role", "main\nallow\n", 0, 2, true}, {"incomplete query", "main", 0, 2, true},
		{"too many packages", strings.Repeat("main\n", 10001), 0, 2, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &goBuildSequenceRunner{roles: []byte(test.roles), failAt: test.failAt}
			operation := &goBuildOperation{builder: &ObservedGoBuilder{runner: runner}}
			runtime := &goBuildRuntime{id: strings.Repeat("a", 64)}
			backend := &goBuildArtifactRunner{operation: operation, runtime: runtime}
			environment, _ := artifactgomodule.BuildEnvironmentForCache(goBuildGuestCache)
			_, err := backend.RunGo(context.Background(), goBuildGuestProject, environment, "build", "-mod=readonly", "./...")
			if (err != nil) != test.wantError || len(runner.calls) != test.wantCalls {
				t.Fatalf("selected package coverage: calls=%d error=%v", len(runner.calls), err)
			}
			commands := [][]string{goBuildValidationArguments("./..."), goBuildRoleArguments("./..."), goBuildCompilerArguments("./...")}
			for i, call := range runner.calls {
				want := boundaryExecArguments(runtime.id, boundaryELFHandoffMode, append([]string{goResolverBinary, "-C", goBuildGuestProject}, commands[i]...)...)
				if !reflect.DeepEqual(call, want) {
					t.Fatal("fixed offline command or package selection changed")
				}
			}
		})
	}
}

func TestGoBuildRunnerRejectsCommandRedirectionBeforeExecution(t *testing.T) {
	for _, arguments := range [][]string{
		{"-toolexec=/tmp/tool"}, {"-o=/outside"}, {"-C=/outside"},
		{"../outside"}, {"/outside"}, {"example.com/mod@v1.0.0"},
		{"./pkg\n"}, {"./pkg", "-toolexec=/tmp/tool"}, {"./one", "./two"},
	} {
		t.Run(strings.Join(arguments, " "), func(t *testing.T) {
			runner := &goBuildRunnerFixture{}
			build, err := NewGoBuildRunner(runner)
			if err != nil {
				t.Fatal(err)
			}
			if err := build.Build(context.Background(), "/workspace/project", "/private/tmp/verified-cache", arguments...); err == nil || runner.call != "" {
				t.Fatalf("redirected build reached runner: call=%q error=%v", runner.call, err)
			}
		})
	}
}

func TestGoBuildRunnerCancelledRequestDoesNotExecute(t *testing.T) {
	runner := &goBuildRunnerFixture{}
	build, err := NewGoBuildRunner(runner)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := build.Build(ctx, "/workspace/project", "/private/tmp/verified-cache", "."); !errors.Is(err, context.Canceled) || runner.call != "" {
		t.Fatalf("cancelled build reached runner: call=%q error=%v", runner.call, err)
	}
}

func TestGoBuildRunnerUsesReadonlyNetworkDisabledEnvironment(t *testing.T) {
	runner := &goBuildRunnerFixture{}
	build, err := NewGoBuildRunner(runner)
	if err != nil {
		t.Fatal(err)
	}
	if err := build.Build(context.Background(), "/workspace/project", "/private/tmp/verified-cache", "./..."); err != nil {
		t.Fatal(err)
	}
	if runner.call != "build -mod=readonly ./..." || !strings.Contains(strings.Join(runner.environment, "\n"), "GOPROXY=off") {
		t.Fatalf("build invocation = %q %#v", runner.call, runner.environment)
	}
}

func TestGoBuildRunnerAcceptsPackageSelectors(t *testing.T) {
	for _, arguments := range [][]string{nil, {"."}, {"./..."}, {"./pkg"}, {"example.com/module/pkg"}, {"example.com/module/..."}} {
		runner := &goBuildRunnerFixture{}
		build, err := NewGoBuildRunner(runner)
		if err != nil {
			t.Fatal(err)
		}
		if err := build.Build(context.Background(), "/workspace/project", "/private/tmp/verified-cache", arguments...); err != nil || runner.call == "" {
			t.Fatalf("package selector %v was rejected: %v", arguments, err)
		}
	}
}

func TestGoBuildRuntimeKeepsArtifactsOfflineAndExistingBounds(t *testing.T) {
	arguments := strings.Join(goBuildCreateArguments(), "\n")
	for _, required := range []string{"--network\nnone", "--read-only", "--memory\n512m", "--cpus\n1", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=readonly", PinnedGoRuntime().ImageReference} {
		if !strings.Contains(arguments, required) {
			t.Fatalf("build runtime omitted %s", required)
		}
	}
	for _, forbidden := range []string{"--add-host", "GOPROXY=https:", "GOPROXY=direct", "GOPRIVATE=example", "GOVCS=*:all", "--privileged"} {
		if strings.Contains(arguments, forbidden) {
			t.Fatalf("build runtime admitted %s", forbidden)
		}
	}
	if !validObserverProfile(goBuildProfile) || validObserverProfile("go-module-build-untrusted") {
		t.Fatal("build profile registration drift")
	}
	budget := traceBudgetForProfile(goBuildProfile)
	if budget.events != 10000 || budget.bytes != 2<<20 {
		t.Fatal("build collector acquired an unapproved budget")
	}
}

type goBuildRunnerFixture struct {
	call        string
	environment []string
	failure     error
}

func (r *goBuildRunnerFixture) RunGo(_ context.Context, _ string, environment []string, arguments ...string) ([]byte, error) {
	if err := artifactgomodule.ValidateBuildEnvironmentForCache(environment, "/private/tmp/verified-cache"); err != nil {
		return nil, err
	}
	r.environment = environment
	r.call = strings.Join(arguments, " ")
	return nil, r.failure
}

func TestGoBuildRunnerKeepsOnlyTrustedFailureDiagnostics(t *testing.T) {
	trusted := &isolatedGoBuildFailure{cause: errors.New("bounded primary"), phase: "BUILD", diagnostic: TraceDiagnostic{Reason: "EVENT_LIMIT"}}
	for _, failure := range []error{trusted, errors.New("artifact controlled secret path")} {
		runner := &goBuildRunnerFixture{failure: failure}
		build, err := NewGoBuildRunner(runner)
		if err != nil {
			t.Fatal(err)
		}
		err = build.Build(context.Background(), "/workspace/project", "/private/tmp/verified-cache", ".")
		if err == nil {
			t.Fatal("failed build became success")
		}
		if failure == trusted {
			if !errors.Is(err, trusted) || !strings.Contains(err.Error(), "EVENT_LIMIT") {
				t.Fatal("trusted observer failure was erased")
			}
		} else if strings.Contains(err.Error(), "secret") || errors.Is(err, failure) {
			t.Fatal("untrusted runner text became diagnostic authority")
		}
	}
}

func TestGoBuildCommandFailureRetainsFirstTrustedStatus(t *testing.T) {
	operation := &goBuildOperation{builder: &ObservedGoBuilder{runner: &goBuildFailureOutput{failure: context.DeadlineExceeded}}}
	runtime := &goBuildRuntime{id: strings.Repeat("a", 64)}
	if _, err := operation.artifactOutput(context.Background(), runtime, "build"); err == nil {
		t.Fatal("failed command became success")
	}
	operation.builder.runner = &goBuildFailureOutput{failure: errors.New("artifact controlled secret")}
	if _, err := operation.artifactOutput(context.Background(), runtime, "build"); err == nil {
		t.Fatal("second failed command became success")
	}
	failure := &isolatedGoBuildFailure{cause: errors.New("private host path"), phase: "BUILD", commandReason: operation.commandReason}
	if !strings.Contains(failure.Error(), "command_status=DEADLINE_EXCEEDED") || strings.Contains(failure.Error(), "private") || strings.Contains(failure.Error(), "secret") {
		t.Fatalf("first trusted command status was not preserved safely: %s", failure)
	}
}

type goBuildFailureOutput struct{ failure error }

func (r *goBuildFailureOutput) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, r.failure
}
func (r *goBuildFailureOutput) Run(context.Context, string, ...string) error { return r.failure }
func (r *goBuildFailureOutput) RunOutput(context.Context, io.Writer, string, ...string) error {
	return r.failure
}
