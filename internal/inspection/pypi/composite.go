package pypi

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// CompositeInspector keeps PyPI archive parsing before its dependent dynamic
// wheel import check. Source distributions remain static until the dedicated
// PEP 517 build boundary supplies a derived wheel.
type CompositeInspector struct {
	static  *StaticInspector
	dynamic *DynamicInspector
}

func NewCompositeInspector(static *StaticInspector, dynamic *DynamicInspector) (*CompositeInspector, error) {
	if static == nil || dynamic == nil {
		return nil, errors.New("PyPI composite inspector requires static and dynamic inspectors")
	}
	return &CompositeInspector{static: static, dynamic: dynamic}, nil
}

func (i *CompositeInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || i.static == nil || i.dynamic == nil {
		return domain.InspectionReport{}, errors.New("PyPI composite inspector is unavailable")
	}
	if artifact.Identity().Variant() == "sdist" {
		return i.static.Inspect(ctx, artifact)
	}
	wheel, static, err := i.static.InspectWheel(ctx, artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if hasBlockingFinding(static) {
		return static, nil
	}
	dynamic, err := i.dynamic.InspectWheel(ctx, artifact, wheel)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewCompositeInspectionReport([]domain.InspectionReport{static, dynamic})
}

func hasBlockingFinding(report domain.InspectionReport) bool { return len(report.Findings()) != 0 }
