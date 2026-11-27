package domain_test

import (
	"math"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestStoragePolicyFiniteLimitsAndStableIdentity(t *testing.T) {
	baseline := domain.DefaultStoragePolicy()
	if !baseline.Valid() || baseline.Spec().TotalBytes != 64<<30 || baseline.Spec().EvidenceRuns != 1000 {
		t.Fatal("canonical defaults changed")
	}
	copy := baseline.Spec()
	copy.EvidenceBytes--
	changed, err := domain.NewStoragePolicy(copy)
	if err != nil || changed.Digest() == baseline.Digest() || baseline.Spec().EvidenceBytes != 256<<20 {
		t.Fatal("policy identity or value isolation lost", err)
	}
	for _, change := range []func(*domain.StoragePolicySpec){
		func(s *domain.StoragePolicySpec) { s.TotalBytes = 0 },
		func(s *domain.StoragePolicySpec) { s.TotalBytes = math.MaxUint64 },
		func(s *domain.StoragePolicySpec) { s.MinimumFreeBytes = 0 },
		func(s *domain.StoragePolicySpec) { s.EvidenceBytes = s.TotalBytes },
		func(s *domain.StoragePolicySpec) { s.EvidenceRuns = 0 },
		func(s *domain.StoragePolicySpec) { s.CacheLowWaterPercent = 100 },
		func(s *domain.StoragePolicySpec) { s.CacheTTLSeconds = 0 },
		func(s *domain.StoragePolicySpec) { s.EvidenceTTLSeconds = math.MaxUint64 },
		func(s *domain.StoragePolicySpec) { s.AdvisoryTTLSeconds = 0 },
	} {
		spec := baseline.Spec()
		change(&spec)
		if _, err := domain.NewStoragePolicy(spec); err == nil {
			t.Fatal("unbounded/contradictory storage policy accepted", spec)
		}
	}
}
