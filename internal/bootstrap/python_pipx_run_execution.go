package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type freshPythonPipxRunExecutionResolver struct {
	native  *sandbox.PipxResolver
	command string
}

func (r freshPythonPipxRunExecutionResolver) ResolveExecutionDependencies(ctx context.Context, reference domain.ArtifactReference) (resolution domain.DependencyResolution, resultErr error) {
	if ctx == nil {
		return resolution, errors.New("pipx run selection requires context")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			resolution = domain.DependencyResolution{}
		}
	}()
	if r.native == nil || ctx.Err() != nil || artifactpypi.ValidateConsoleExecutionCommand(r.command) != nil || reference.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return resolution, errors.New("pipx run requires its public selector and configured command")
	}
	canonical, err := artifactpypi.ParseReference(reference.Locator())
	if err != nil || canonical != reference {
		return resolution, errors.New("pipx run requires a canonical bare/exact reference")
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(canonical.Locator())
	if err != nil {
		return resolution, err
	}
	selected, err := r.native.ResolveToolRun(ctx, requirement, r.command)
	if err != nil {
		return resolution, err
	}
	entry := selected.Entry()
	if !entry.Valid() || entry.Artifact().Identity().Name() != requirement.Project() || selected.Resolution().RuntimeIdentity() != "pipx-run-isolated:"+sandbox.PipxResolverRuntimeIdentity() || entry.Command() != r.command && r.command != requirement.Project() {
		return resolution, errors.New("pipx native choice differs from its original request or client")
	}
	return selected.Resolution(), nil
}

type pythonPipxRunExecutionBinder struct {
	static  *inspectionpypi.StaticInspector
	command string
}

func (b pythonPipxRunExecutionBinder) BindExecutionEntry(ctx context.Context, inspection application.ExecutionInspection, command string) (execution domain.ExecutionContext, resultErr error) {
	if ctx == nil {
		return execution, errors.New("pipx current binding requires context")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			execution = domain.ExecutionContext{}
		}
	}()
	if ctx.Err() != nil || b.static == nil || !inspection.Valid() || artifactpypi.ValidateConsoleExecutionCommand(b.command) != nil || inspection.Resolution().RuntimeIdentity() != "pipx-run-isolated:"+sandbox.PipxResolverRuntimeIdentity() {
		return execution, errors.New("pipx binding requires fresh whole approval and its native selector")
	}
	if command != "" && command != b.command {
		return execution, errors.New("current command differs from configured pipx selection")
	}
	inventory, err := readPythonConsoleExecutionInventory(ctx, inspection, b.static)
	if err != nil {
		return execution, err
	}
	// Recompute the same client choice from the approved original. Native cache,
	// metadata and DATA entries never become current execution authority.
	entry, err := artifactpypi.SelectPipxConsoleExecutionEntry(inventory, b.command)
	if err != nil {
		return execution, err
	}
	return newPythonConsoleExecutionContext(inspection.OperationID(), inspection.Resolution(), inspection.Set(), entry)
}

func newPipxRunExecutionService(native *sandbox.PipxResolver, command string, intake *artifactpypi.Intake, inspector *inspectionpypi.CompositeInspector, static *inspectionpypi.StaticInspector, backend *sandbox.PythonDynamicBackend, store *local.Store, staging *promotion.LocalStaging) (*application.ExecutionService, error) {
	if native == nil || artifactpypi.ValidateConsoleExecutionCommand(command) != nil {
		return nil, errors.New("pipx current execution requires its actual selector and configured command")
	}
	return newPythonNativeRunExecutionService(freshPythonPipxRunExecutionResolver{native, command}, pythonPipxRunExecutionBinder{static, command}, intake, inspector, static, backend, store, staging)
}
