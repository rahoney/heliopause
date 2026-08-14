package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// InstallOutcome preserves completed security decisions even when a later
// storage or Promotion operation fails.
type InstallOutcome struct {
	inspection InspectedInstall
	bundle     domain.VerifiedBundle
	staged     domain.StagedSet
	promoted   domain.PromotedInstall
}

func (o InstallOutcome) Inspection() InspectedInstall     { return o.inspection }
func (o InstallOutcome) Bundle() domain.VerifiedBundle    { return o.bundle }
func (o InstallOutcome) Staged() domain.StagedSet         { return o.staged }
func (o InstallOutcome) Promoted() domain.PromotedInstall { return o.promoted }

// InstallService is the sole M4 ordering authority.
type InstallService struct {
	inspection *InstallInspectService
	manifest   ports.Manifest
	staging    ports.Staging
	promotion  ports.Promotion
}

func NewInstallService(inspection *InstallInspectService, manifest ports.Manifest, staging ports.Staging, promotion ports.Promotion) (*InstallService, error) {
	if inspection == nil || manifest == nil || staging == nil || promotion == nil {
		return nil, errors.New("install service requires inspection, Manifest, Staging, and Promotion")
	}
	return &InstallService{inspection: inspection, manifest: manifest, staging: staging, promotion: promotion}, nil
}

func (s *InstallService) Install(ctx context.Context, request InstallRequest) (InstallOutcome, error) {
	inspected, err := s.inspection.Inspect(ctx, request)
	if err != nil {
		return InstallOutcome{}, err
	}
	outcome := InstallOutcome{inspection: inspected}
	verified, err := domain.NewVerifiedSet(inspected.Set(), inspected.Decision())
	if err != nil {
		return outcome, fmt.Errorf("construct complete ALLOW set: %w", err)
	}
	bundle, err := s.manifest.Build(ctx, inspected.OperationID(), request.Context(), inspected.Resolution(), verified)
	if err != nil {
		return outcome, fmt.Errorf("build verified Manifest and SBOM: %w", err)
	}
	outcome.bundle = bundle
	staged, err := s.staging.Stage(ctx, bundle)
	if err != nil {
		return outcome, fmt.Errorf("stage verified set: %w", err)
	}
	outcome.staged = staged
	promoted, err := s.promotion.Promote(ctx, staged, bundle, request.Context())
	if err != nil {
		return outcome, fmt.Errorf("promote staged set: %w", err)
	}
	outcome.promoted = promoted
	return outcome, nil
}
