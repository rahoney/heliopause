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
	if reference.Source().String() == "" || context.Target().String() == "" || !context.RequiresNewTarget() {
		return InstallRequest{}, errors.New("install request requires Artifact Reference and new-target Install Context")
	}
	return InstallRequest{reference: reference, context: context}, nil
}

func (r InstallRequest) Reference() domain.ArtifactReference { return r.reference }
func (r InstallRequest) Context() domain.InstallContext      { return r.context }
