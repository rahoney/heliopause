package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type pythonPipxRunCLI struct {
	native    *sandbox.PipxResolver
	intake    *artifactpypi.Intake
	inspector *inspectionpypi.CompositeInspector
	static    *inspectionpypi.StaticInspector
	backend   *sandbox.PythonDynamicBackend
	store     *local.Store
	staging   *promotion.LocalStaging
}

func (r pythonPipxRunCLI) Run(ctx context.Context, request application.ExecutionPreparationRequest) (result application.ExecutionResult, resultErr error) {
	if ctx == nil {
		return result, errors.New("pipx CLI execution requires context")
	}
	defer func() { resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx)) }()
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	command := request.Command()
	project, err := artifactpypi.RequestedProject(request.Reference())
	if err != nil {
		return result, err
	}
	// A primary-name request is pipx's default choice. Keep that choice in
	// the concrete selector while the application binds its measured console.
	if command == project {
		request, err = application.NewExecutionPreparationRequest(request.Reference(), "", request.Arguments())
		if err != nil {
			return result, err
		}
	}
	service, err := newPipxRunExecutionService(r.native, command, r.intake, r.inspector, r.static, r.backend, r.store, r.staging)
	if err != nil {
		return result, err
	}
	return service.Run(ctx, request)
}

func pipxRunExecutionPlatform(goos, arch string) error {
	if goos != "linux" || arch != "amd64" {
		return errors.New("pipx execution requires Linux amd64")
	}
	return nil
}
