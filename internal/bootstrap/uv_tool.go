package bootstrap

import (
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// Reuse the current primary inspection/Policy/Manifest/staging pipeline. Its
// per-request resolver and promotion are supplied by the held tool session.
func newUVToolInstallService(b pythonClientBindings, binRoot string) (*application.PreparedInstallService, error) {
	promoter, ok := b.promoter.(*promotion.PyPIPromotion)
	if !ok || promoter == nil || b.installer == nil {
		return nil, errors.New("UV tool installation requires the existing Python install pipeline")
	}
	native, ok := b.nativeLookup.(*sandbox.UVResolver)
	if !ok || native == nil {
		return nil, errors.New("UV tool installation requires the reviewed native UV selector")
	}
	preparation, err := promotion.NewUVToolInstallPreparation(promoter, native, binRoot, []string{b.intakeRoot, b.evidenceRoot, b.stagingRoot})
	if err != nil {
		return nil, err
	}
	return application.NewPreparedInstallService(b.installer, preparation)
}
