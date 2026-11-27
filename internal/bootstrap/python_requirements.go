package bootstrap

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

type pythonRequirementsService struct {
	bindings pythonClientBindings
	client   string
}

func (s *pythonRequirementsService) InstallRequirements(ctx context.Context, request cli.PythonRequirementsRequest) (cli.PythonRequirementsOutcome, error) {
	var outcome cli.PythonRequirementsOutcome
	if s == nil || ctx == nil || request.Requirements.Client != s.client {
		return outcome, errors.New("requirements installation service differs from the selected client")
	}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	b := s.bindings
	promoter, ok := b.promoter.(*promotion.PyPIPromotion)
	if !ok {
		return outcome, errors.New("requirements installation needs the existing Python transaction")
	}
	resolver, ok := b.nativeLookup.(ports.ProjectLockUpdateResolver)
	if !ok {
		return outcome, errors.New("requirements installation lacks a complete native resolver")
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		return outcome, err
	}
	ctx = ports.WithStorageOperation(ctx, operation)
	cache, err := promotion.NewPythonProjectCache(b.intakeRoot, b.evidenceRoot, b.stagingRoot, b.evidence, evidence.Generator{}, operation)
	if err != nil {
		return outcome, err
	}
	preparation, err := promotion.NewPythonRequirementsPreparation(promoter, resolver, request.Requirements, []string{b.intakeRoot, b.evidenceRoot, b.stagingRoot})
	if err != nil {
		return outcome, err
	}
	preparation, err = preparation.WithNativeEnrollment(b.nativeLookup, b.intake, b.intakeRoot)
	if err != nil {
		return outcome, err
	}
	publication, err := promotion.NewPythonRequirementsPromotion(preparation, cache)
	if err != nil {
		return outcome, err
	}
	inspection, err := application.NewProjectInspectService(b.intake, verificationpypi.IntegrityVerifier{}, b.inspector, b.evidence, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return operation, nil }, domain.NewRunID)
	if err != nil {
		return outcome, err
	}
	service, err := application.NewPythonRequirementsService(preparation, publication, inspection, publication)
	if err != nil {
		return outcome, err
	}
	outcome.Verified, outcome.Noop, err = service.Install(ctx, request.Install)
	if err != nil || outcome.Noop {
		return outcome, err
	}
	id, manifest, sbom, ok := publication.PublishedRecords()
	if !ok {
		return outcome, errors.New("requirements installation lacks completed publication records")
	}
	outcome.ManifestID, outcome.ManifestHandle, outcome.SBOMHandle = id, manifest, sbom
	return outcome, nil
}
