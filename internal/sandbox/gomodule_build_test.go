package sandbox

import (
	"context"
	"strings"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

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
