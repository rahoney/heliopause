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

// This preparation adapter retains the actual native tool-install resolver
// identity. It does not claim UVx native-run equivalence or expose an alias.
type freshPythonToolExecutionResolver struct{ native *sandbox.UVResolver }

func (r freshPythonToolExecutionResolver) ResolveExecutionDependencies(ctx context.Context, reference domain.ArtifactReference) (resolution domain.DependencyResolution, err error) {
	if ctx == nil {
		return resolution, errors.New("python execution selection requires a context")
	}
	defer func() {
		err = errors.Join(err, ctx.Err(), context.Cause(ctx))
		if err != nil {
			resolution = domain.DependencyResolution{}
		}
	}()
	if r.native == nil || ctx.Err() != nil || reference.Source() != artifactpypi.PublicPyPIProfile().Source() {
		return resolution, errors.New("python execution selection requires the fresh public tool selector")
	}
	canonical, err := artifactpypi.ParseReference(reference.Locator())
	if err != nil || canonical != reference {
		return resolution, errors.New("python execution selection requires a canonical bare or exact public reference")
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(canonical.Locator())
	if err != nil {
		return resolution, err
	}
	selected, err := r.native.ResolveToolInstall(ctx, requirement)
	if err != nil {
		return resolution, err
	}
	return selected.Resolution(), nil
}

type pythonConsoleExecutionBinder struct {
	static *inspectionpypi.StaticInspector
}

func (b pythonConsoleExecutionBinder) BindExecutionEntry(ctx context.Context, inspection application.ExecutionInspection, command string) (domain.ExecutionContext, error) {
	if ctx == nil || b.static == nil || !inspection.Valid() {
		return domain.ExecutionContext{}, errors.New("python console binding requires current whole approval and controlled intake")
	}
	var primary domain.AcquiredArtifact
	for _, entry := range inspection.Set().Inspected().Inspections() {
		if entry.Node() == inspection.Resolution().Graph().Primary() {
			primary = entry.Artifact()
		}
	}
	inventory, err := b.static.ReadConsoleExecutionInventory(ctx, primary)
	if err != nil {
		return domain.ExecutionContext{}, err
	}
	selected, err := artifactpypi.SelectConsoleExecutionEntry(inventory, command)
	if err != nil {
		return domain.ExecutionContext{}, err
	}
	return newPythonConsoleExecutionContext(inspection.OperationID(), inspection.Resolution(), inspection.Set(), selected)
}

// The measured member is the source console declaration, not a module body
// or proof that its callable ran. A runtime must rebind the same declaration
// and complete approved closure before importing or invoking anything.
func newPythonConsoleExecutionContext(operation domain.OperationID, resolution domain.DependencyResolution, set domain.VerifiedSet, entry artifactpypi.ConsoleExecutionEntry) (domain.ExecutionContext, error) {
	if !entry.Valid() {
		return domain.ExecutionContext{}, errors.New("python execution preparation requires a measured console declaration")
	}
	member := entry.Metadata()
	digest, err := domain.NewSHA256Digest(member.SHA256)
	if err != nil || member.Size <= 0 {
		return domain.ExecutionContext{}, errors.New("python console declaration member is incomplete")
	}
	return domain.NewExecutionContext(domain.ExecutionContextData{
		OperationID: operation, Resolution: resolution, Set: set, EntryArtifact: entry.Artifact(), Command: entry.Command(),
		EntryPath: member.Path, EntryDigest: digest, EntrySize: uint64(member.Size), RuntimeIdentity: runtimeidentity.PythonImageReference,
	})
}

func newPythonExecutionPreparationService(native *sandbox.UVResolver, intake *artifactpypi.Intake, inspector *inspectionpypi.CompositeInspector, static *inspectionpypi.StaticInspector, store *local.Store, staging *promotion.LocalStaging) (*application.ExecutionPreparationService, error) {
	if native == nil || intake == nil || inspector == nil || static == nil || store == nil || staging == nil {
		return nil, errors.New("python execution preparation requires actual native selection, intake, current inspection and storage")
	}
	fresh, err := application.NewExecutionInspectService(intake, verificationpypi.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		return nil, err
	}
	return application.NewExecutionPreparationService(freshPythonToolExecutionResolver{native}, fresh, pythonConsoleExecutionBinder{static}, evidence.Generator{}, staging)
}
