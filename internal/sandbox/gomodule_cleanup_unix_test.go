//go:build linux || darwin

package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type goCleanupFailureRunner struct {
	goModuleRunnerFixture
	cache string
	fail  bool
}

func (r *goCleanupFailureRunner) RunGo(ctx context.Context, directory string, environment []string, arguments ...string) ([]byte, error) {
	for _, entry := range environment {
		if value, ok := strings.CutPrefix(entry, "GOMODCACHE="); ok {
			r.cache = value
		}
	}
	locked := filepath.Join(r.cache, "locked")
	if err := os.Mkdir(locked, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	if err := os.Chmod(locked, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(locked, "content"), []byte("fixture"), 0o600); err != nil {
		return nil, err
	}
	if err := os.Chmod(locked, 0o500); err != nil {
		return nil, err
	}
	if r.fail {
		return nil, errors.New("runner fixture failed")
	}
	return r.goModuleRunnerFixture.RunGo(ctx, directory, environment, arguments...)
}

func TestGoModuleCleanupFailureCannotPublishResolution(t *testing.T) {
	for _, command := range []string{"get", "download"} {
		for _, fail := range []bool{false, true} {
			t.Run(command+map[bool]string{true: "/primary-failure", false: "/normal-work"}[fail], func(t *testing.T) {
				project := t.TempDir()
				for name, body := range map[string]string{
					"go.mod": "module example.com/app\ngo 1.25\nrequire example.com/mod v1.2.3\n",
					"go.sum": "fixture-sum\n",
				} {
					if err := os.WriteFile(filepath.Join(project, name), []byte(body), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				runner := &goCleanupFailureRunner{fail: fail}
				t.Cleanup(func() {
					if runner.cache != "" {
						_ = os.Chmod(filepath.Join(runner.cache, "locked"), 0o700)
						if err := os.RemoveAll(runner.cache); err != nil {
							t.Errorf("dispose test-owned failed cache: %v", err)
						}
					}
				})
				resolver, _ := NewGoModuleResolver(runner)
				target, _ := domain.NewInstallTarget(project)
				install, _ := domain.NewInstallContext(target)
				var err error
				if command == "get" {
					ref, _ := artifactgomodule.ParseReference("example.com/mod@v1.2.3")
					resolution, resolutionErr := resolver.ResolveDependencies(context.Background(), ref, install)
					err = resolutionErr
					if len(resolution.Graph().Nodes()) != 0 {
						t.Fatal("cleanup failure exposed an exact graph")
					}
				} else {
					snapshot, snapshotErr := resolver.ResolveProjectDependencies(context.Background(), install)
					err = snapshotErr
					if snapshot.Valid() {
						t.Fatal("cleanup failure exposed a project snapshot")
					}
				}
				if err == nil || !strings.Contains(err.Error(), "dispose private Go module cache failed") {
					t.Fatalf("cleanup failure was hidden: %v", err)
				}
				if fail && !strings.HasPrefix(err.Error(), map[string]string{"get": "private go module selection failed", "download": "go module download failed"}[command]) {
					t.Fatalf("cleanup replaced the primary failure: %v", err)
				}
			})
		}
	}
}
