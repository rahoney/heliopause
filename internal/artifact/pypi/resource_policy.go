package pypi

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ResourcePolicy is selected by an install request's canonical root
// SourceProfile. It never replaces the independent source identity frozen on
// each resolved graph node.
type ResourcePolicy struct {
	maxArtifactCompressed   int64
	maxArtifactUncompressed int64
	maxFilesPerArtifact     int64
	maxMetadataFile         int64
	maxGraphCompressed      int64
	maxGraphUncompressed    int64
	maxGraphArtifacts       int
	maxTemporaryDisk        int64
	qualificationDuration   time.Duration
	runtimeMemory           int64
	runtimeTmpfs            int64
	promotionTmpfs          int64
}

func defaultResourcePolicy() ResourcePolicy {
	limits := DefaultWheelLimits()
	return ResourcePolicy{
		maxArtifactCompressed: limits.MaxCompressed, maxArtifactUncompressed: limits.MaxUncompressed,
		maxFilesPerArtifact: limits.MaxFiles, maxMetadataFile: limits.MaxMetadata,
		maxGraphCompressed: limits.MaxCompressed, maxGraphUncompressed: limits.MaxUncompressed,
		maxGraphArtifacts: 64, maxTemporaryDisk: 512 << 20,
		qualificationDuration: 5 * time.Minute, runtimeMemory: 512 << 20,
		runtimeTmpfs: 256 << 20, promotionTmpfs: 128 << 20,
	}
}

func pyTorchCPUResourcePolicy() ResourcePolicy {
	return ResourcePolicy{
		maxArtifactCompressed: 256 << 20, maxArtifactUncompressed: 1 << 30,
		maxFilesPerArtifact: 20_000, maxMetadataFile: 2 << 20,
		maxGraphCompressed: 512 << 20, maxGraphUncompressed: 2 << 30,
		maxGraphArtifacts: 64, maxTemporaryDisk: 4 << 30,
		qualificationDuration: 15 * time.Minute, runtimeMemory: 2 << 30,
		runtimeTmpfs: 2 << 30, promotionTmpfs: 512 << 20,
	}
}

func pyTorchCU126ResourcePolicy() ResourcePolicy {
	return ResourcePolicy{
		maxArtifactCompressed: 1 << 30, maxArtifactUncompressed: (5 << 30) / 2,
		maxFilesPerArtifact: 20_000, maxMetadataFile: 2 << 20,
		maxGraphCompressed: (9 << 30) / 2, maxGraphUncompressed: 8 << 30,
		maxGraphArtifacts: 64, maxTemporaryDisk: 24 << 30,
		qualificationDuration: 40 * time.Minute, runtimeMemory: 4 << 30,
		runtimeTmpfs: 3 << 30, promotionTmpfs: 1 << 30,
	}
}

func (p ResourcePolicy) WheelLimits() WheelLimits {
	return WheelLimits{MaxCompressed: p.maxArtifactCompressed, MaxUncompressed: p.maxArtifactUncompressed, MaxFiles: p.maxFilesPerArtifact, MaxMetadata: p.maxMetadataFile}
}

func (p ResourcePolicy) MaxArtifactCompressed() int64 { return p.maxArtifactCompressed }
func (p ResourcePolicy) MaxGraphCompressed() int64    { return p.maxGraphCompressed }
func (p ResourcePolicy) MaxGraphUncompressed() int64  { return p.maxGraphUncompressed }
func (p ResourcePolicy) MaxFilesPerArtifact() int64   { return p.maxFilesPerArtifact }
func (p ResourcePolicy) MaxTemporaryDisk() int64      { return p.maxTemporaryDisk }
func (p ResourcePolicy) Duration() time.Duration      { return p.qualificationDuration }
func (p ResourcePolicy) RuntimeMemory() int64         { return p.runtimeMemory }
func (p ResourcePolicy) RuntimeTmpfs() int64          { return p.runtimeTmpfs }
func (p ResourcePolicy) PromotionTmpfs() int64        { return p.promotionTmpfs }

func (p ResourcePolicy) valid() bool {
	return p.maxArtifactCompressed > 0 && p.maxArtifactUncompressed > 0 && p.maxFilesPerArtifact > 0 && p.maxMetadataFile > 0 && p.maxGraphCompressed > 0 && p.maxGraphUncompressed > 0 && p.maxGraphArtifacts > 0 && p.maxTemporaryDisk > 0 && p.qualificationDuration > 0 && p.runtimeMemory > 0 && p.runtimeTmpfs > 0 && p.promotionTmpfs > 0
}

type resourceSession struct {
	policy   ResourcePolicy
	mu       sync.Mutex
	count    int
	bytes    int64
	expanded int64
}

func (s *resourceSession) chargeUncompressed(bytes int64) error {
	if s == nil || bytes < 0 {
		return errors.New("PyPI resource accounting is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.expanded > s.policy.maxGraphUncompressed-bytes {
		return errors.New("PyPI expanded graph resource budget exceeds bound")
	}
	s.expanded += bytes
	return nil
}

// CheckTemporaryDisk fail-closes before a large transaction starts when the
// selected root policy cannot fit on the filesystem that owns its workspace.
func CheckTemporaryDisk(directory string, policy ResourcePolicy) error {
	if directory == "" || !policy.valid() {
		return errors.New("PyPI temporary disk preflight is invalid")
	}
	var statistics syscall.Statfs_t
	if err := syscall.Statfs(filepath.Clean(directory), &statistics); err != nil {
		return errors.New("PyPI temporary disk capability is unavailable")
	}
	available := uint64(statistics.Bavail) * uint64(statistics.Bsize)
	if available < uint64(policy.maxTemporaryDisk) {
		return errors.New("PyPI temporary disk resource budget exceeds available space")
	}
	return nil
}

func (s *resourceSession) beginArtifact(contentLength int64) error {
	if s == nil || !s.policy.valid() || contentLength < -1 {
		return errors.New("PyPI resource policy is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count >= s.policy.maxGraphArtifacts || contentLength > s.policy.maxArtifactCompressed || contentLength >= 0 && s.bytes > s.policy.maxGraphCompressed-contentLength {
		return errors.New("PyPI resource budget exceeds bound")
	}
	s.count++
	return nil
}

func (s *resourceSession) charge(bytes int64) error {
	if s == nil || bytes < 0 {
		return errors.New("PyPI resource accounting is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bytes > s.policy.maxGraphCompressed-bytes {
		return errors.New("PyPI graph resource budget exceeds bound")
	}
	s.bytes += bytes
	return nil
}

type resourcePolicyContextKey struct{}

// ContextWithResourcePolicy creates transaction-local accounting for one
// canonical root profile. Bootstrap composition, not user input, selects it.
func ContextWithResourcePolicy(ctx context.Context, profile SourceProfile) (context.Context, error) {
	if ctx == nil || !profile.resourcePolicy.valid() {
		return nil, errors.New("PyPI root resource profile is invalid")
	}
	return context.WithValue(ctx, resourcePolicyContextKey{}, &resourceSession{policy: profile.resourcePolicy}), nil
}

func resourcePolicyFromContext(ctx context.Context) (ResourcePolicy, *resourceSession) {
	if ctx != nil {
		if session, ok := ctx.Value(resourcePolicyContextKey{}).(*resourceSession); ok && session != nil && session.policy.valid() {
			return session.policy, session
		}
	}
	policy := defaultResourcePolicy()
	return policy, &resourceSession{policy: policy}
}

// ResourcePolicyFromContext returns the root-profile policy selected by the
// bootstrap transaction, or the unchanged conservative default.
func ResourcePolicyFromContext(ctx context.Context) ResourcePolicy {
	policy, _ := resourcePolicyFromContext(ctx)
	return policy
}

// ChargeUncompressedFromContext accounts the bounded file-backed archive
// expansion observed by static inspection for the current transaction.
func ChargeUncompressedFromContext(ctx context.Context, bytes int64) error {
	_, session := resourcePolicyFromContext(ctx)
	return session.chargeUncompressed(bytes)
}
