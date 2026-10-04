package sandbox

import (
	"context"
	"errors"
	"path/filepath"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

// GoBuildRunner executes only a network-disabled, readonly Go build against a
// caller-provided verified cache. It does not fall back to resolver settings.
type GoBuildRunner struct{ runner GoModuleRunner }

func NewGoBuildRunner(runner GoModuleRunner) (*GoBuildRunner, error) {
	if runner == nil {
		return nil, errors.New("go build runner requires trusted Go runner")
	}
	return &GoBuildRunner{runner: runner}, nil
}

func (b *GoBuildRunner) Build(ctx context.Context, project, cache string, packageArgs ...string) error {
	if b == nil || b.runner == nil || ctx == nil || !filepath.IsAbs(project) || filepath.Clean(project) != project || project == "/" {
		return errors.New("valid Go build request is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(packageArgs) > 1 || (len(packageArgs) == 1 && artifactgomodule.ValidateBuildPackage(packageArgs[0]) != nil) {
		return errors.New("go build requires one package selector without command flags")
	}
	environment, err := artifactgomodule.BuildEnvironmentForCache(cache)
	if err != nil {
		return err
	}
	if err := artifactgomodule.ValidateBuildEnvironmentForCache(environment, cache); err != nil {
		return err
	}
	arguments := append([]string{"build", "-mod=readonly"}, packageArgs...)
	if _, err := b.runner.RunGo(ctx, project, environment, arguments...); err != nil {
		return errors.New("network-disabled Go build failed")
	}
	return nil
}
