package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

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

type goBuildRunnerFixture struct {
	call        string
	environment []string
}

func (r *goBuildRunnerFixture) RunGo(_ context.Context, _ string, environment []string, arguments ...string) ([]byte, error) {
	if err := artifactgomodule.ValidateBuildEnvironmentForCache(environment, "/private/tmp/verified-cache"); err != nil {
		return nil, err
	}
	r.environment = environment
	r.call = strings.Join(arguments, " ")
	return nil, nil
}
