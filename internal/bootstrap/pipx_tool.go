package bootstrap

import (
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// Use the existing current whole inspection/Evidence/Policy and staging owners.
func newPipxToolInstallService(b pythonClientBindings, native *sandbox.PipxResolver, binRoot string) (*application.PreparedInstallService, error) {
	promoter, ok := b.promoter.(*promotion.PyPIPromotion)
	if !ok || promoter == nil || b.installer == nil || native == nil {
		return nil, errors.New("pipx tool installation requires the existing Python pipeline and native selector")
	}
	preparation, err := promotion.NewPipxToolInstallPreparation(promoter, native, binRoot, []string{b.intakeRoot, b.evidenceRoot, b.stagingRoot})
	if err != nil {
		return nil, err
	}
	return application.NewPreparedInstallService(b.installer, preparation)
}

func newPipxToolUpdateService(b pythonClientBindings, native *sandbox.PipxResolver, binRoot string) (*application.InstallUpdateService, error) {
	promoter, ok := b.promoter.(*promotion.PyPIPromotion)
	if !ok || promoter == nil || b.installer == nil || native == nil {
		return nil, errors.New("pipx upgrade requires the existing Python pipeline and native selector")
	}
	preparation, err := promotion.NewPipxToolUpdatePreparation(promoter, native, binRoot, []string{b.intakeRoot, b.evidenceRoot, b.stagingRoot})
	if err != nil {
		return nil, err
	}
	preparation, err = preparation.WithNativeEnrollment(b.nativeLookup, b.intake, b.intakeRoot)
	if err != nil {
		return nil, err
	}
	return application.NewInstallUpdateService(b.installer, preparation)
}
