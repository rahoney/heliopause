package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// GoProjectRunner is the narrow trusted Go execution capability required for
// private project mutation. It must not inherit the caller environment.
type GoProjectRunner interface {
	RunGo(context.Context, string, []string, ...string) ([]byte, error)
}

// GoProjectPromotion runs an exact go get in a disposable workspace and
// publishes go.mod/go.sum atomically only for an HAA-managed project.
type GoProjectPromotion struct{ runner GoProjectRunner }

func NewGoProjectPromotion(runner GoProjectRunner) (*GoProjectPromotion, error) {
	if runner == nil {
		return nil, errors.New("go project Promotion requires trusted Go runner")
	}
	return &GoProjectPromotion{runner: runner}, nil
}

func (p *GoProjectPromotion) PromoteGet(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (resultErr error) {
	if p == nil || p.runner == nil || ctx == nil || reference.Source() != artifactgomodule.Source() || !installContext.Valid() {
		return errors.New("valid Go project Promotion request is required")
	}
	root := installContext.Target().String()
	guard, err := acquireGoProjectGuard(root)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, guard.release()) }()
	plan, err := freezeGoProject(root)
	if err != nil {
		return err
	}
	workspace, err := plan.privateWorkspace()
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	cache, err := os.MkdirTemp("", "haa-go-get-cache-")
	if err != nil {
		return errors.New("create private Go get cache")
	}
	defer os.RemoveAll(cache)
	environment, err := artifactgomodule.ResolverEnvironmentForCache(cache)
	if err != nil {
		return err
	}
	if _, err := p.runner.RunGo(ctx, filepath.Clean(workspace), environment, "get", reference.Locator()); err != nil {
		return errors.New("private go get failed")
	}
	if err := plan.verifyUnchanged(); err != nil {
		return err
	}
	transaction, err := beginGoProjectTransaction(plan, workspace)
	if err != nil {
		return err
	}
	return transaction.commit()
}

func (p *GoProjectPromotion) PromoteProjectDependency(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) error {
	return p.PromoteGet(ctx, reference, installContext)
}
