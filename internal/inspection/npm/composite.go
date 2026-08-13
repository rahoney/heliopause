package npm

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// CompositeInspector runs the required static and dynamic npm inspections for one exact Artifact.
type CompositeInspector struct{ inspectors []ports.Inspection }

func NewCompositeInspector(inspectors ...ports.Inspection) (*CompositeInspector, error) {
	if len(inspectors) == 0 {
		return nil, errors.New("at least one npm inspector is required")
	}
	for _, inspector := range inspectors {
		if inspector == nil {
			return nil, errors.New("npm inspector is required")
		}
	}
	return &CompositeInspector{inspectors: append([]ports.Inspection(nil), inspectors...)}, nil
}

func (i *CompositeInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || len(i.inspectors) == 0 {
		return domain.InspectionReport{}, errors.New("npm composite inspector is required")
	}
	reports := make([]domain.InspectionReport, 0, len(i.inspectors))
	for _, inspector := range i.inspectors {
		report, err := inspector.Inspect(ctx, artifact)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		reports = append(reports, report)
	}
	return domain.NewCompositeInspectionReport(reports)
}
