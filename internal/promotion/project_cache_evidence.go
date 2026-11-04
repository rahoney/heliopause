package promotion

import (
	"context"
	"errors"
	"slices"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/inspection/content"
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

// Current approval requires the same readonly content declarations at Stage and
// readback. Historical original inspection is not a current approval shortcut.
func verifyArchiveContentEvidence(facts []domain.Evidence) error {
	required, err := content.ArchiveEvidenceRequirements()
	if err != nil {
		return err
	}
	byCheck := make(map[domain.CheckID]content.EvidenceRequirement, len(required))
	for _, requirement := range required {
		byCheck[requirement.Check.ID()] = requirement
	}
	seen := make(map[domain.EvidenceID]bool, len(facts))
	for _, fact := range facts {
		if seen[fact.ID()] {
			return errors.New("archive content approval contains duplicate Evidence")
		}
		seen[fact.ID()] = true
		if requirement, known := byCheck[fact.CheckID()]; known {
			if fact.ProfileDigest() != requirement.Check.ProfileDigest() || fact.ID() == requirement.ID && fact.Kind() != requirement.Kind || fact.ID() != requirement.ID && !slices.Contains(requirement.ExtraKinds, fact.Kind()) {
				return errors.New("archive content Evidence differs from current profile or kind")
			}
		}
	}
	for _, requirement := range required {
		found := false
		for _, fact := range facts {
			if fact.ID() == requirement.ID && fact.CheckID() == requirement.Check.ID() {
				found = true
				break
			}
		}
		if !found {
			return errors.New("archive approval lacks current source/native/inventory Evidence")
		}
	}
	return nil
}
