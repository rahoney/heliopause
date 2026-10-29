package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

// The configured command is immutable per service. Native DATA is not a
// current inspection, and no previous approval or installed receipt is kept.
type freshPythonUVRunExecutionResolver struct {
	native  *sandbox.UVResolver
	command string
}

func (r freshPythonUVRunExecutionResolver) ResolveExecutionDependencies(ctx context.Context, reference domain.ArtifactReference) (resolution domain.DependencyResolution, resultErr error) {
	if ctx == nil {
		return resolution, errors.New("native run selection requires context")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			resolution = domain.DependencyResolution{}
		}
	}()
	if r.native == nil || ctx.Err() != nil || artifactpypi.ValidateConsoleExecutionCommand(r.command) != nil || reference.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return resolution, errors.New("native run selection requires actual public selector and configured console")
	}
	canonical, err := artifactpypi.ParseReference(reference.Locator())
	if err != nil || canonical != reference {
		return resolution, errors.New("native run selection requires canonical bare/exact public reference")
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(canonical.Locator())
	if err != nil {
		return resolution, err
	}
	selected, err := r.native.ResolveToolRun(ctx, requirement, r.command)
	if err != nil {
		return resolution, err
	}
	if !selected.Entry().Valid() || selected.Entry().Command() != r.command || selected.Resolution().RuntimeIdentity() != "uv-tool-run-isolated:"+sandbox.UVResolverRuntimeIdentity() {
		return resolution, errors.New("native run selection differs from configured command or actual selector")
	}
	return selected.Resolution(), nil
}

type pythonUVRunExecutionBinder struct {
	static  *inspectionpypi.StaticInspector
	command string
}

func (b pythonUVRunExecutionBinder) BindExecutionEntry(ctx context.Context, inspection application.ExecutionInspection, command string) (execution domain.ExecutionContext, resultErr error) {
	if ctx == nil {
		return execution, errors.New("native run current binding requires context")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			execution = domain.ExecutionContext{}
		}
	}()
	if ctx.Err() != nil || b.static == nil || !inspection.Valid() || artifactpypi.ValidateConsoleExecutionCommand(b.command) != nil || inspection.Resolution().RuntimeIdentity() != "uv-tool-run-isolated:"+sandbox.UVResolverRuntimeIdentity() {
		return execution, errors.New("native run binding requires actual current whole approval and its selector")
	}
	if command != "" && command != b.command {
		return execution, errors.New("current command differs from native source/console selection")
	}
	measured, err := (pythonConsoleExecutionBinder{static: b.static}).BindExecutionEntry(ctx, inspection, command)
	if err != nil {
		return execution, err
	}
	if measured.Command() != b.command {
		return execution, errors.New("canonical default command differs from native source/console selection")
	}
	return measured, nil
}

// The same current integrity/inspection/Policy/records and readonly runtime
// are used with the real native-run selector. The old tool-install factory
// remains a separate consumer with its original resolver identity.
func newUVRunExecutionService(native *sandbox.UVResolver, command string, intake *artifactpypi.Intake, inspector *inspectionpypi.CompositeInspector, static *inspectionpypi.StaticInspector, backend *sandbox.PythonDynamicBackend, store *local.Store, staging *promotion.LocalStaging) (*application.ExecutionService, error) {
	if native == nil || intake == nil || inspector == nil || static == nil || backend == nil || store == nil || staging == nil || artifactpypi.ValidateConsoleExecutionCommand(command) != nil {
		return nil, errors.New("native run execution requires actual adapters and an explicit canonical console")
	}
	fresh, err := application.NewExecutionInspectService(intake, verificationpypi.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	preparation, err := application.NewExecutionPreparationService(freshPythonUVRunExecutionResolver{native, command}, fresh, pythonUVRunExecutionBinder{static, command}, evidence.Generator{}, staging)
	if err != nil {
		return nil, err
	}
	return application.NewExecutionService(preparation, pythonExecutionRuntime{backend, static})
}
