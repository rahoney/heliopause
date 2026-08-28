package application

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// InstallRequest is the validated M4 request boundary before resolution or any
// target filesystem action is attempted.
type InstallRequest struct {
	reference domain.ArtifactReference
	context   domain.InstallContext
}

func NewInstallRequest(reference domain.ArtifactReference, context domain.InstallContext) (InstallRequest, error) {
	if reference.Source().String() == "" || !context.Valid() {
		return InstallRequest{}, errors.New("install request requires Artifact Reference and supported Install Context")
	}
	return InstallRequest{reference: reference, context: context}, nil
}

func (r InstallRequest) Reference() domain.ArtifactReference { return r.reference }
func (r InstallRequest) Context() domain.InstallContext      { return r.context }

// InspectedInstall preserves the exact operation and resolver context needed
// to build a Manifest after the complete set-level Policy decision.
type InspectedInstall struct {
	operationID        domain.OperationID
	request            InstallRequest
	resolution         domain.DependencyResolution
	set                domain.InspectedDependencySet
	decision           domain.PolicyDecision
	diagnostics        []GraphStaticDiagnostic
	dynamicDiagnostics []GraphDynamicInstallDiagnostic
}

// GraphStaticDiagnostic is the application-owned bounded qualification
// record for a node that prevented graph static readiness. It contains only
// validated Domain identity fields and the bounded diagnostic vocabulary.
type GraphStaticDiagnostic struct {
	node        domain.DependencyNodeID
	packageName string
	version     string
	source      domain.SourceID
	variant     string
	cause       domain.InspectionDiagnosticCause
	stage       domain.InspectionDiagnosticStage
}

func newGraphStaticDiagnostic(dependency domain.LockedDependency, diagnostic domain.InspectionDiagnostic) (GraphStaticDiagnostic, error) {
	identity := dependency.Artifact().Identity()
	if dependency.Node().String() == "" || identity.Name() == "" || identity.Version() == "" || identity.Source().String() == "" || identity.Variant() == "" {
		return GraphStaticDiagnostic{}, errors.New("graph static diagnostic identity is invalid")
	}
	if _, err := domain.NewInspectionDiagnostic(diagnostic.Cause(), diagnostic.Stage()); err != nil {
		return GraphStaticDiagnostic{}, err
	}
	return GraphStaticDiagnostic{node: dependency.Node(), packageName: identity.Name(), version: identity.Version(), source: identity.Source(), variant: identity.Variant(), cause: diagnostic.Cause(), stage: diagnostic.Stage()}, nil
}

func (d GraphStaticDiagnostic) Node() domain.DependencyNodeID           { return d.node }
func (d GraphStaticDiagnostic) Package() string                         { return d.packageName }
func (d GraphStaticDiagnostic) Version() string                         { return d.version }
func (d GraphStaticDiagnostic) Source() domain.SourceID                 { return d.source }
func (d GraphStaticDiagnostic) Variant() string                         { return d.variant }
func (d GraphStaticDiagnostic) Cause() domain.InspectionDiagnosticCause { return d.cause }
func (d GraphStaticDiagnostic) Stage() domain.InspectionDiagnosticStage { return d.stage }

// GraphDynamicInstallDiagnostic is a bounded qualification record for a
// graph node whose dynamic wheel operation did not complete.
// It deliberately carries no subprocess output or Host path.
type GraphDynamicInstallDiagnostic struct {
	node         domain.DependencyNodeID
	packageName  string
	version      string
	source       domain.SourceID
	variant      string
	reason       string
	phase        string
	failureClass string
}

func newGraphDynamicInstallDiagnostic(dependency domain.LockedDependency, reason, phase, failureClass string) (GraphDynamicInstallDiagnostic, error) {
	identity := dependency.Artifact().Identity()
	if dependency.Node().String() == "" || identity.Name() == "" || identity.Version() == "" || identity.Source().String() == "" || identity.Variant() == "" || !validDynamicFailure(reason, phase, failureClass) {
		return GraphDynamicInstallDiagnostic{}, errors.New("graph dynamic install diagnostic is invalid")
	}
	return GraphDynamicInstallDiagnostic{node: dependency.Node(), packageName: identity.Name(), version: identity.Version(), source: identity.Source(), variant: identity.Variant(), reason: reason, phase: phase, failureClass: failureClass}, nil
}

func (d GraphDynamicInstallDiagnostic) Node() domain.DependencyNodeID { return d.node }
func (d GraphDynamicInstallDiagnostic) Package() string               { return d.packageName }
func (d GraphDynamicInstallDiagnostic) Version() string               { return d.version }
func (d GraphDynamicInstallDiagnostic) Source() domain.SourceID       { return d.source }
func (d GraphDynamicInstallDiagnostic) Variant() string               { return d.variant }
func (d GraphDynamicInstallDiagnostic) Reason() string                { return d.reason }
func (d GraphDynamicInstallDiagnostic) Phase() string                 { return d.phase }
func (d GraphDynamicInstallDiagnostic) FailureClass() string          { return d.failureClass }

func newInspectedInstall(operationID domain.OperationID, request InstallRequest, resolution domain.DependencyResolution, set domain.InspectedDependencySet, decision domain.PolicyDecision) InspectedInstall {
	return InspectedInstall{operationID: operationID, request: request, resolution: resolution, set: set, decision: decision}
}

func newPartialInspectedInstall(operationID domain.OperationID, request InstallRequest, resolution domain.DependencyResolution) InspectedInstall {
	return InspectedInstall{operationID: operationID, request: request, resolution: resolution}
}

func (i InspectedInstall) withGraphStaticDiagnostics(diagnostics []GraphStaticDiagnostic) InspectedInstall {
	i.diagnostics = append([]GraphStaticDiagnostic(nil), diagnostics...)
	return i
}

func (i InspectedInstall) withGraphDynamicInstallDiagnostics(diagnostics []GraphDynamicInstallDiagnostic) InspectedInstall {
	i.dynamicDiagnostics = append([]GraphDynamicInstallDiagnostic(nil), diagnostics...)
	return i
}

func (i InspectedInstall) OperationID() domain.OperationID         { return i.operationID }
func (i InspectedInstall) Request() InstallRequest                 { return i.request }
func (i InspectedInstall) Resolution() domain.DependencyResolution { return i.resolution }
func (i InspectedInstall) Set() domain.InspectedDependencySet      { return i.set }
func (i InspectedInstall) Decision() domain.PolicyDecision         { return i.decision }
func (i InspectedInstall) GraphStaticDiagnostics() []GraphStaticDiagnostic {
	return append([]GraphStaticDiagnostic(nil), i.diagnostics...)
}
func (i InspectedInstall) GraphDynamicInstallDiagnostics() []GraphDynamicInstallDiagnostic {
	return append([]GraphDynamicInstallDiagnostic(nil), i.dynamicDiagnostics...)
}
