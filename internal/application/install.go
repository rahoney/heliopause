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
	if reference.Source().String() == "" || context.Target().String() == "" || (context.Mode() != domain.InstallNewTarget && context.Mode() != domain.InstallNPMProject && context.Mode() != domain.InstallPythonVenv) {
		return InstallRequest{}, errors.New("install request requires Artifact Reference and supported Install Context")
	}
	return InstallRequest{reference: reference, context: context}, nil
}

func (r InstallRequest) Reference() domain.ArtifactReference { return r.reference }
func (r InstallRequest) Context() domain.InstallContext      { return r.context }

// InspectedInstall preserves the exact operation and resolver context needed
// to build a Manifest after the complete set-level Policy decision.
type InspectedInstall struct {
	operationID domain.OperationID
	request     InstallRequest
	resolution  domain.DependencyResolution
	set         domain.InspectedDependencySet
	decision    domain.PolicyDecision
}

func newInspectedInstall(operationID domain.OperationID, request InstallRequest, resolution domain.DependencyResolution, set domain.InspectedDependencySet, decision domain.PolicyDecision) InspectedInstall {
	return InspectedInstall{operationID: operationID, request: request, resolution: resolution, set: set, decision: decision}
}

func newPartialInspectedInstall(operationID domain.OperationID, request InstallRequest, resolution domain.DependencyResolution) InspectedInstall {
	return InspectedInstall{operationID: operationID, request: request, resolution: resolution}
}

func (i InspectedInstall) OperationID() domain.OperationID         { return i.operationID }
func (i InspectedInstall) Request() InstallRequest                 { return i.request }
func (i InspectedInstall) Resolution() domain.DependencyResolution { return i.resolution }
func (i InspectedInstall) Set() domain.InspectedDependencySet      { return i.set }
func (i InspectedInstall) Decision() domain.PolicyDecision         { return i.decision }
