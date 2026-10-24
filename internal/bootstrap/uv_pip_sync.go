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
	"github.com/spf13/cobra"
)

type pythonClientEvidence interface {
	ports.Evidence
	promotion.ProjectEvidenceReader
}

type uvPipSyncService struct {
	promoter *promotion.PyPIPromotion
	resolver ports.ProjectLockUpdateResolver
	bindings pythonClientBindings
}

func bindUVPipSync(command *cobra.Command, bindings pythonClientBindings) error {
	promoter, ok := bindings.promoter.(*promotion.PyPIPromotion)
	if !ok || bindings.intake == nil || bindings.inspector == nil || bindings.evidence == nil {
		return errors.New("UV pip sync requires the existing Python inspection, Evidence and transaction")
	}
	resolver, ok := bindings.nativeLookup.(ports.ProjectLockUpdateResolver)
	if !ok {
		return errors.New("UV pip sync resolver lacks complete native selection")
	}
	return cli.AddUVPipSync(command, &uvPipSyncService{promoter, resolver, bindings})
}

func (s *uvPipSyncService) Sync(ctx context.Context, request cli.UVPipSyncRequest) (cli.UVPipSyncOutcome, error) {
	var outcome cli.UVPipSyncOutcome
	if s == nil || ctx == nil {
		return outcome, errors.New("UV pip sync service is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		return outcome, err
	}
	b := s.bindings
	cache, err := promotion.NewPythonProjectCache(b.intakeRoot, b.evidenceRoot, b.stagingRoot, b.evidence, evidence.Generator{}, operation)
	if err != nil {
		return outcome, err
	}
	preparation, err := promotion.NewUVSyncPreparation(s.promoter, s.resolver, request.SourceFile, request.Format, request.AllowEmpty, request.Adopt, []string{b.intakeRoot, b.evidenceRoot, b.stagingRoot})
	if err != nil {
		return outcome, err
	}
	preparation, err = preparation.WithNativeEnrollment(b.nativeLookup, b.intake, b.intakeRoot)
	if err != nil {
		return outcome, err
	}
	promoter, err := promotion.NewUVSyncProjectPromotion(preparation, cache)
	if err != nil {
		return outcome, err
	}
	inspection, err := application.NewProjectInspectService(b.intake, verificationpypi.IntegrityVerifier{}, b.inspector, b.evidence, policy.M3{}, policy.M4{}, func() (domain.OperationID, error) { return operation, nil }, domain.NewRunID)
	if err != nil {
		return outcome, err
	}
	service, err := application.NewUVPipSyncService(preparation, promoter, inspection, promoter)
	if err != nil {
		return outcome, err
	}
	outcome.Verified, err = service.Sync(ctx, request.Install)
	if err != nil {
		return outcome, err
	}
	id, manifest, sbom, ok := promoter.PublishedRecords()
	if !ok {
		return outcome, errors.New("UV pip sync completed without verified publication records")
	}
	outcome.ManifestID, outcome.ManifestHandle, outcome.SBOMHandle = id, manifest, sbom
	return outcome, nil
}
