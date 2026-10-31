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
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

type freshPythonUVRunSelectionResolver struct {
	native  *sandbox.UVResolver
	command string
	request artifactpypi.UVRunRequirements
}

func (r freshPythonUVRunSelectionResolver) ResolveExecutionSelection(ctx context.Context, reference domain.ArtifactReference) (resolution domain.ExecutionSelectionResolution, resultErr error) {
	if ctx == nil {
		return resolution, errors.New("UV root selection requires context")
	}
	defer func() {
		resultErr = errors.Join(resultErr, ctx.Err(), context.Cause(ctx))
		if resultErr != nil {
			resolution = domain.ExecutionSelectionResolution{}
		}
	}()
	if r.native == nil || !r.request.Valid() || reference != r.request.Primary().Reference() {
		return resolution, errors.New("UV root request differs from configured public primary")
	}
	selected, err := r.native.ResolveToolRunWithRequirements(ctx, r.request, r.command)
	if err != nil {
		return resolution, err
	}
	if !selected.Entry().Valid() || selected.Entry().Command() != r.command || !selected.Resolution().Valid() || selected.Resolution().RuntimeIdentity() != "uv-tool-run-isolated:"+sandbox.UVResolverRuntimeIdentity() {
		return resolution, errors.New("UV root selection differs from the actual configured selector")
	}
	return selected.Resolution(), nil
}

type pythonUVRunSelectionBinder struct {
	static  *inspectionpypi.StaticInspector
	command string
}

func (b pythonUVRunSelectionBinder) BindExecutionSelectionEntry(ctx context.Context, inspection application.ExecutionSelectionInspection, command string) (domain.ExecutionSelectionContext, error) {
	if ctx == nil || b.static == nil || !inspection.Valid() || inspection.Resolution().RuntimeIdentity() != "uv-tool-run-isolated:"+sandbox.UVResolverRuntimeIdentity() || command != "" && command != b.command {
		return domain.ExecutionSelectionContext{}, errors.New("console binding requires the current complete UV selection and requested primary command")
	}
	var primary domain.AcquiredArtifact
	for _, inspected := range inspection.Set().Inspected().Inspections() {
		if inspected.Node() == inspection.Resolution().Selection().Primary() {
			primary = inspected.Artifact()
		}
	}
	inventory, err := b.static.ReadConsoleExecutionInventory(ctx, primary)
	if err != nil {
		return domain.ExecutionSelectionContext{}, err
	}
	entry, err := artifactpypi.SelectConsoleExecutionEntry(inventory, command)
	if err != nil {
		return domain.ExecutionSelectionContext{}, err
	}
	if entry.Command() != b.command {
		return domain.ExecutionSelectionContext{}, errors.New("current primary console differs from native selection")
	}
	member := entry.Metadata()
	digest, err := domain.NewSHA256Digest(member.SHA256)
	if err != nil || member.Size <= 0 {
		return domain.ExecutionSelectionContext{}, errors.New("current primary console member is incomplete")
	}
	return domain.NewExecutionSelectionContext(domain.ExecutionSelectionContextData{OperationID: inspection.OperationID(), Resolution: inspection.Resolution(), Set: inspection.Set(), EntryArtifact: entry.Artifact(), Command: entry.Command(), EntryPath: member.Path, EntryDigest: digest, EntrySize: uint64(member.Size), RuntimeIdentity: runtimeidentity.PythonImageReference})
}

func newUVRunSelectionService(native *sandbox.UVResolver, command string, request artifactpypi.UVRunRequirements, intake *artifactpypi.Intake, inspector *inspectionpypi.CompositeInspector, static *inspectionpypi.StaticInspector, backend *sandbox.PythonDynamicBackend, store *local.Store, staging *promotion.LocalStaging) (*application.ExecutionSelectionService, error) {
	if native == nil || intake == nil || inspector == nil || static == nil || backend == nil || store == nil || staging == nil || !request.Valid() || artifactpypi.ValidateConsoleExecutionCommand(command) != nil {
		return nil, errors.New("UV root execution requires actual adapters and bounded public requests")
	}
	fresh, err := application.NewExecutionInspectService(intake, verificationpypi.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	preparation, err := application.NewExecutionSelectionPreparationService(freshPythonUVRunSelectionResolver{native, command, request}, fresh, pythonUVRunSelectionBinder{static, command}, evidence.Generator{}, staging)
	if err != nil {
		return nil, err
	}
	return application.NewExecutionSelectionService(preparation, pythonExecutionRuntime{backend, static})
}
