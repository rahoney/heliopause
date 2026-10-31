package application

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// PreparedInstallService adds a pre-resolution target guard while retaining
// ordinary INSTALL intent and the existing inspection/Policy/staging order.
type PreparedInstallService struct {
	base        *InstallService
	preparation ports.InstallPreparation
}

func NewPreparedInstallService(base *InstallService, preparation ports.InstallPreparation) (*PreparedInstallService, error) {
	if base == nil || base.inspection == nil || preparation == nil {
		return nil, errors.New("prepared install requires the install pipeline and target preparation")
	}
	return &PreparedInstallService{base, preparation}, nil
}

func (s *PreparedInstallService) Install(ctx context.Context, request InstallRequest) (outcome InstallOutcome, resultErr error) {
	if s == nil || ctx == nil || request.update || request.reference.Source().String() == "" || !request.context.Valid() {
		return outcome, errors.New("validated ordinary install request is required")
	}
	operation, err := s.base.inspection.newOperationID()
	if err != nil {
		return outcome, err
	}
	outcome = InstallOutcome{inspection: newPartialInspectedInstall(operation, request, domain.DependencyResolution{}), failureCode: "INSTALL_PREPARATION_FAILED"}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	session, err := s.preparation.BeginInstall(ctx, request.reference, request.context)
	if err != nil {
		return outcome, err
	}
	if session == nil {
		return outcome, errors.New("install preparation returned no target guard")
	}
	defer func() {
		closeErr := session.Close()
		if closeErr != nil && outcome.failureCode == "" {
			outcome.failureCode = "INSTALL_GUARD_CLOSE_FAILED"
		}
		resultErr = errors.Join(resultErr, closeErr)
	}()
	return s.base.installWithSession(ctx, request, operation, session)
}

// Both ordinary prepared installs and updates bind per-request copies only.
// The base service remains safe for simultaneous independent operations.
func (s *InstallService) installWithSession(ctx context.Context, request InstallRequest, operation domain.OperationID, session ports.InstallSession) (InstallOutcome, error) {
	inspection := *s.inspection
	inspection.resolver = session
	if content, ok := session.(ports.InstallUpdateContentSession); ok {
		inspection.artifact, inspection.verification, inspection.inspection = content, content, content
	}
	inspection.newOperationID = func() (domain.OperationID, error) { return operation, nil }
	pipeline := *s
	pipeline.inspection, pipeline.promotion = &inspection, session
	return pipeline.Install(ctx, request)
}
