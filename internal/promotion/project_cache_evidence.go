package promotion

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Both project caches re-read the same controller-owned Evidence records.
// Typed references alone are not sufficient retained approval.
func projectCacheEvidence(ctx context.Context, reader ProjectEvidenceReader, inspection domain.DependencyInspection) ([]goCacheEvidence, error) {
	records, _, err := projectCacheEvidenceFacts(ctx, reader, inspection)
	return records, err
}

func projectCacheEvidenceFacts(ctx context.Context, reader ProjectEvidenceReader, inspection domain.DependencyInspection) ([]goCacheEvidence, []domain.Evidence, error) {
	plan := inspection.CheckPlan()
	if !plan.Valid() {
		return nil, nil, errors.New("project cache approval requires its frozen check declaration")
	}
	if err := plan.Validate(inspection.RunID(), inspection.Artifact(), inspection.Checks()); err != nil {
		return nil, nil, err
	}
	checks := make(map[domain.CheckID]domain.CheckExecution)
	for _, check := range inspection.Checks() {
		checks[check.ID()] = check
	}
	var facts []domain.Evidence
	var records []goCacheEvidence
	covered := map[domain.CheckID]bool{}
	artifact := inspection.Artifact()
	for _, ref := range inspection.Evidence() {
		item, digest, err := reader.ReadReference(ctx, inspection.RunID(), ref, artifact.Identity(), artifact.Digest())
		if err != nil {
			return nil, nil, err
		}
		check, known := checks[item.CheckID()]
		if !known || item.ProfileDigest().String() == "" || item.ProfileDigest() != check.ProfileDigest() {
			return nil, nil, errors.New("project cache Evidence is legacy or differs from its declared check profile")
		}
		covered[item.CheckID()] = true
		facts = append(facts, item)
		records = append(records, goCacheEvidence{ref.ID().String(), digest.String()})
	}
	for _, check := range inspection.Checks() {
		if check.Required() && !covered[check.ID()] {
			return nil, nil, errors.New("project cache approval has missing required Evidence")
		}
	}
	return records, facts, nil
}
