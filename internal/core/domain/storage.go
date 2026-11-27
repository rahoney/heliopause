package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"time"
)

// StoragePolicySpec uses bytes and seconds explicitly. These retention limits
// cannot enlarge any artifact, runtime or inspection security budget.
type StoragePolicySpec struct {
	TotalBytes           uint64 `json:"total_bytes"`
	MinimumFreeBytes     uint64 `json:"minimum_free_bytes"`
	CacheBytes           uint64 `json:"cache_bytes"`
	CacheTTLSeconds      uint64 `json:"cache_ttl_seconds"`
	CacheLowWaterPercent uint8  `json:"cache_low_water_percent"`
	EvidenceBytes        uint64 `json:"evidence_bytes"`
	EvidenceRuns         uint32 `json:"evidence_runs"`
	EvidenceTTLSeconds   uint64 `json:"evidence_ttl_seconds"`
	AdvisoryTTLSeconds   uint64 `json:"advisory_ttl_seconds"`
}

// StoragePolicy is a validated immutable value shared by storage consumers.
type StoragePolicy struct {
	spec   StoragePolicySpec
	digest ContentDigest
}

func DefaultStoragePolicy() StoragePolicy {
	p, _ := NewStoragePolicy(StoragePolicySpec{
		TotalBytes: 64 << 30, MinimumFreeBytes: 2 << 30,
		CacheBytes: 16 << 30, CacheTTLSeconds: 7 * 24 * 60 * 60, CacheLowWaterPercent: 80,
		EvidenceBytes: 256 << 20, EvidenceRuns: 1000, EvidenceTTLSeconds: 30 * 24 * 60 * 60,
		AdvisoryTTLSeconds: 24 * 60 * 60,
	})
	return p
}

func NewStoragePolicy(s StoragePolicySpec) (StoragePolicy, error) {
	if s.TotalBytes == 0 || s.TotalBytes > math.MaxInt64 || s.MinimumFreeBytes == 0 || s.MinimumFreeBytes > math.MaxInt64 || s.CacheBytes == 0 || s.EvidenceBytes == 0 || s.EvidenceRuns == 0 || s.CacheBytes > s.TotalBytes || s.EvidenceBytes > s.TotalBytes-s.CacheBytes || s.CacheLowWaterPercent == 0 || s.CacheLowWaterPercent >= 100 {
		return StoragePolicy{}, errors.New("storage policy requires finite byte/count limits and cache/evidence within total quota")
	}
	for _, seconds := range []uint64{s.CacheTTLSeconds, s.EvidenceTTLSeconds, s.AdvisoryTTLSeconds} {
		if seconds == 0 || seconds > uint64(math.MaxInt64/int64(time.Second)) {
			return StoragePolicy{}, errors.New("storage retention must be finite positive seconds")
		}
	}
	body, err := json.Marshal(struct {
		Version uint8             `json:"version"`
		Policy  StoragePolicySpec `json:"policy"`
	}{1, s})
	if err != nil {
		return StoragePolicy{}, err
	}
	sum := sha256.Sum256(body)
	digest, err := NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return StoragePolicy{}, err
	}
	return StoragePolicy{spec: s, digest: digest}, nil
}

func (p StoragePolicy) Spec() StoragePolicySpec { return p.spec }
func (p StoragePolicy) Digest() ContentDigest   { return p.digest }
func (p StoragePolicy) Valid() bool             { return p.digest.String() != "" }
