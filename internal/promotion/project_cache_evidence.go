package promotion

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Both project caches re-read the same controller-owned Evidence records.
// Typed references alone are not sufficient retained approval.
func projectCacheEvidence(ctx context.Context, reader ProjectEvidenceReader, inspection domain.DependencyInspection) ([]goCacheEvidence, error) {
	var records []goCacheEvidence
	covered := map[domain.CheckID]bool{}
	artifact := inspection.Artifact()
	for _, ref := range inspection.Evidence() {
		item, digest, err := reader.ReadReference(ctx, inspection.RunID(), ref, artifact.Identity(), artifact.Digest())
		if err != nil {
			return nil, err
		}
		covered[item.CheckID()] = true
		records = append(records, goCacheEvidence{ref.ID().String(), digest.String()})
	}
	for _, check := range inspection.Checks() {
		if check.Required() && !covered[check.ID()] {
			return nil, errors.New("project cache approval has missing required Evidence")
		}
	}
	return records, nil
}
