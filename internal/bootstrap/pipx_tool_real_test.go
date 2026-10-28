//go:build linux && realcorpus

package bootstrap

import (
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This actual-only composition qualifies the exported preparation before the
// separate production CLI binding. Selection remains unapproved until the existing whole inspection/Evidence/
// Policy service has approved and staged the same wheel graph and controls.
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
