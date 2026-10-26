package bootstrap

import (
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func newUVToolUpdateService(b pythonClientBindings, binRoot string) (*application.InstallUpdateService, error) {
	promoter, ok := b.promoter.(*promotion.PyPIPromotion)
	if !ok || promoter == nil || b.installer == nil {
		return nil, errors.New("UV tool upgrade requires the existing Python install pipeline")
	}
	native, ok := b.nativeLookup.(*sandbox.UVResolver)
	if !ok || native == nil {
		return nil, errors.New("UV tool upgrade requires the reviewed native UV selector")
	}
	preparation, err := promotion.NewUVToolUpdatePreparation(promoter, native, binRoot, []string{b.intakeRoot, b.evidenceRoot, b.stagingRoot})
	if err != nil {
		return nil, err
	}
	return application.NewInstallUpdateService(b.installer, preparation)
}
