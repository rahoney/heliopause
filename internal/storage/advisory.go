package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

var errAdvisoryCapacity = errors.New("optional advisory cache capacity is unavailable")

// Called under the existing ledger lock. Composition/help remains inert. This
// namespace contains only opaque DATA; no consumer receives an executable path.
func (s *Local) openAdvisoryRoot() error {
	if s.advisory != nil {
		return nil
	}
	if err := s.root.Mkdir("advisory-cache", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := s.root.Lstat("advisory-cache")
	if err != nil || !privateDirectory(info) {
		return errors.New("advisory cache root is not owned and private")
	}
	root, err := s.root.OpenRoot("advisory-cache")
	if err != nil {
		return err
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) || opened.Sys().(*syscall.Stat_t).Dev != s.journalInfo.Sys().(*syscall.Stat_t).Dev {
		return errors.Join(errors.New("advisory cache root identity or filesystem changed"), root.Close())
	}
	for _, record := range s.entries {
		if record.Kind == "advisory" && !matchesIdentity(opened, record.CacheRoot) {
			return errors.Join(errors.New("advisory cache ancestor changed since registration"), root.Close())
		}
	}
	s.advisory, s.advisoryInfo = root, opened
	return errors.Join(syncRoot(root), syncRoot(s.root))
}

func (s *Local) ReadAdvisory(ctx context.Context, key domain.ContentDigest, limit uint64) (body []byte, hit bool, resultErr error) {
	if key.String() == "" || limit == 0 || limit >= uint64(^uint64(0)>>1) {
		return nil, false, errors.New("advisory cache read is invalid")
	}
	resultErr = s.withLock(ctx, func() error {
		if err := s.openAdvisoryRoot(); err != nil {
			return err
		}
		s.sweepAdvisories(ctx, 0, s.advisoryIndex[key.String()])
		run, ok := s.advisoryIndex[key.String()]
		if !ok {
			return nil
		}
		record, ok := s.entries[resourceKey("advisory", run)]
		if !ok || record.CacheKey != key.String() {
			return errors.New("advisory cache index is incomplete")
		}
		if record.Phase == "DELETING" {
			return s.removeAdvisory(ctx, record)
		}
		if record.Phase != "RETAINED" {
			return errors.New("advisory cache is not sealed")
		}
		now := s.now().Unix()
		if now < record.Created {
			return errors.New("advisory cache clock moved backwards")
		}
		if now >= record.Expiry || now-record.Created >= int64(s.policy.Spec().AdvisoryTTLSeconds) {
			return s.removeAdvisory(ctx, record)
		}
		var err error
		body, err = s.readAdvisory(ctx, record, limit)
		if err == nil {
			hit = true
		}
		return err
	})
	return
}

// The lock is held until every descriptor is closed and only bytes are returned.
func (s *Local) readAdvisory(ctx context.Context, record resourceRecord, limit uint64) (body []byte, resultErr error) {
	info, err := s.advisory.Lstat(record.Target)
	if !matchesIdentity(s.advisoryInfo, record.CacheRoot) || err != nil || !privateDirectory(info) || !matchesIdentity(info, record.Identity) {
		return nil, errors.New("advisory cache allocation changed")
	}
	root, err := s.advisory.OpenRoot(record.Target)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("advisory cache directory changed")
	}
	if record.LogicalBytes > limit {
		return nil, errors.New("advisory cache exceeds the current read bound")
	}
	members := map[string]measuredMember{}
	size, digest, err := measureResource(ctx, root, "advisory", record.ReservedBytes, 1, members)
	if err != nil || size != record.LogicalBytes || digest != record.Content {
		return nil, errors.Join(errors.New("advisory cache content changed"), err)
	}
	f, err := root.OpenFile("response.json", os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, f.Close()) }()
	actual, err := f.Stat()
	member := members["response.json"]
	if err != nil || !privateFile(actual) || !os.SameFile(member.info, actual) {
		return nil, errors.New("advisory cache member changed")
	}
	body, err = io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	after, statErr := root.Lstat("response.json")
	if uint64(len(body)) != size || hex.EncodeToString(sum[:]) != member.sha256 || statErr != nil || !privateFile(after) || !os.SameFile(actual, after) {
		return nil, errors.New("advisory cache changed during read")
	}
	return body, ctx.Err()
}

func (s *Local) WriteAdvisory(ctx context.Context, key domain.ContentDigest, body []byte) (stored bool, resultErr error) {
	if ctx == nil || key.String() == "" || len(body) == 0 {
		return false, errors.New("advisory cache write is invalid")
	}
	operation := ports.StorageOperationFromContext(ctx)
	if operation.String() == "" {
		return false, errors.New("advisory cache requires its controller operation")
	}
	// This is an allocation generation only. It never creates an InspectionRun,
	// Evidence reference or Policy decision for an uninspected subject.
	run, err := domain.NewRunID()
	if err != nil {
		return false, err
	}
	temporary := ".advisory-" + run.String()
	var directory *os.Root
	var identity os.FileInfo
	var lease ports.StorageLease
	published := false
	commitAttempted := false
	var member os.FileInfo
	defer func() {
		if directory == nil {
			return
		}
		// Close the reservation's duplicate descriptor before the private writer
		// removes its own failed allocation. Partial/unknown state stays charged.
		if lease != nil {
			resultErr = errors.Join(resultErr, lease.Close())
		}
		if !stored && !commitAttempted {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cleanupErr := s.withLock(cleanup, func() error {
				name := temporary
				if published {
					name = run.String()
				}
				actual, err := s.advisory.Lstat(name)
				if err != nil || !privateDirectory(actual) || !os.SameFile(identity, actual) {
					return errors.New("failed advisory allocation changed; preserved")
				}
				if member != nil {
					actual, err := directory.Lstat("response.json")
					if err != nil || !privateFile(actual) || !os.SameFile(member, actual) {
						return errors.New("failed advisory member changed; preserved")
					}
					if err := directory.Remove("response.json"); err != nil {
						return err
					}
				}
				if err := syncRoot(directory); err != nil {
					return err
				}
				if err := s.advisory.Remove(name); err != nil {
					return err
				}
				return syncRoot(s.advisory)
			})
			resultErr = errors.Join(resultErr, cleanupErr)
			if cleanupErr == nil && lease != nil {
				resultErr = errors.Join(resultErr, lease.Abort(cleanup))
			}
		}
		resultErr = errors.Join(resultErr, directory.Close())
	}()
	if err := s.withLock(ctx, func() error {
		if err := s.openAdvisoryRoot(); err != nil {
			return err
		}
		if uint64(len(body)) > s.policy.Spec().CacheBytes {
			return errAdvisoryCapacity
		}
		s.sweepAdvisories(ctx, uint64(len(body)), "")
		if err := s.advisory.Mkdir(temporary, 0700); err != nil {
			return err
		}
		var err error
		identity, err = s.advisory.Lstat(temporary)
		if err != nil {
			return err
		}
		directory, err = s.advisory.OpenRoot(temporary)
		return err
	}); err != nil {
		if errors.Is(err, errAdvisoryCapacity) {
			return false, nil
		}
		return false, err
	}
	lease, err = s.Reserve(ctx, ports.StorageRequest{Operation: operation, Run: run, Kind: "advisory", Temporary: temporary, Target: run.String(), Bytes: uint64(len(body)), Files: 1, CacheKey: key})
	if err != nil {
		if errors.Is(err, errAdvisoryCapacity) {
			return false, nil
		}
		return false, err
	}
	if concrete, ok := lease.(*resourceLease); !ok || !matchesIdentity(identity, concrete.record.Identity) {
		return false, errors.New("advisory reservation differs from writer allocation")
	}
	f, err := directory.OpenFile("response.json", os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return false, err
	}
	member, err = f.Stat()
	if err != nil || !privateFile(member) {
		return false, errors.Join(errors.New("advisory writer member is not private"), err, f.Close())
	}
	if err := ctx.Err(); err != nil {
		return false, errors.Join(err, f.Close())
	}
	n, writeErr := f.Write(body)
	if writeErr == nil && n != len(body) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return false, err
	}
	if err := s.withLock(ctx, func() error {
		actual, err := s.advisory.Lstat(temporary)
		if err != nil || !privateDirectory(actual) || !os.SameFile(identity, actual) {
			return errors.New("advisory publication allocation changed")
		}
		if _, err := s.advisory.Lstat(run.String()); !errors.Is(err, os.ErrNotExist) {
			return errors.New("advisory publication target exists or is unknown")
		}
		if err := s.advisory.Rename(temporary, run.String()); err != nil {
			return err
		}
		published = true
		return syncRoot(s.advisory)
	}); err != nil {
		return false, err
	}
	// A failed commit can already have written a durable event. Never remove
	// that allocation as an aborted write; retain accounting for recovery.
	commitAttempted = true
	if err := lease.Commit(ctx); err != nil {
		return false, err
	}
	// Superseded copies are DATA with no external path/FD references. A cleanup
	// failure belongs to the outer finalization axis, not this completed query.
	if previous := lease.(*resourceLease).previousAdvisory; previous != "" {
		cleanupErr := s.withLock(ctx, func() error {
			record, exists := s.entries[resourceKey("advisory", previous)]
			if !exists || s.advisoryIndex[key.String()] == previous {
				return nil
			}
			return s.removeAdvisory(ctx, record)
		})
		if cleanupErr != nil {
			s.mu.Lock()
			s.advisoryCleanupErr = errors.Join(s.advisoryCleanupErr, cleanupErr)
			s.mu.Unlock()
		}
	}
	return true, nil
}

// Only the opaque DATA reader above can hold a cache payload, and it closes all
// descriptors before releasing this same ledger lock. Artifact/runtime handles
// and IN_USE allocations never enter this deletion path.
func (s *Local) removeAdvisory(ctx context.Context, record resourceRecord) (resultErr error) {
	if record.Kind != "advisory" || (record.Phase != "RETAINED" && record.Phase != "DELETING") || !matchesIdentity(s.advisoryInfo, record.CacheRoot) {
		return errors.New("advisory cache is not eligible for removal")
	}
	info, err := s.advisory.Lstat(record.Target)
	if errors.Is(err, os.ErrNotExist) && record.Phase == "DELETING" {
		if err := syncRoot(s.advisory); err != nil {
			return err
		}
		removed := record
		removed.Phase = "REMOVED"
		return s.append(journalEvent{"removed", removed})
	}
	if err != nil || !privateDirectory(info) || !matchesIdentity(info, record.Identity) {
		return errors.New("advisory deletion identity is unknown")
	}
	root, err := s.advisory.OpenRoot(record.Target)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("advisory deletion directory changed")
	}
	memberPresent := true
	if record.Phase == "DELETING" {
		f, err := root.Open(".")
		if err != nil {
			return err
		}
		entries, readErr := f.ReadDir(2)
		closeErr := f.Close()
		if readErr != nil && readErr != io.EOF || closeErr != nil {
			return errors.Join(readErr, closeErr)
		}
		if len(entries) == 0 {
			memberPresent = false
		}
	}
	members := map[string]measuredMember{}
	if memberPresent {
		size, digest, err := measureResource(ctx, root, "advisory", record.ReservedBytes, 1, members)
		if err != nil || size != record.LogicalBytes || digest != record.Content {
			return errors.Join(errors.New("advisory deletion content changed"), err)
		}
	}
	if record.Phase == "RETAINED" {
		deleting := record
		deleting.Phase = "DELETING"
		if err := s.append(journalEvent{"deleting", deleting}); err != nil {
			return err
		}
	}
	parent, err := s.advisory.Lstat(record.Target)
	if err != nil || !privateDirectory(parent) || !os.SameFile(info, parent) {
		return errors.New("advisory deletion directory changed")
	}
	if memberPresent {
		actual, err := root.Lstat("response.json")
		if err != nil || !privateFile(actual) || !os.SameFile(actual, members["response.json"].info) {
			return errors.New("advisory deletion member changed")
		}
		if err := root.Remove("response.json"); err != nil {
			return err
		}
	}
	if err := syncRoot(root); err != nil {
		return err
	}
	if err := s.advisory.Remove(record.Target); err != nil {
		return err
	}
	if err := syncRoot(s.advisory); err != nil {
		return err
	}
	removed := record
	removed.Phase = "REMOVED"
	return s.append(journalEvent{"removed", removed})
}

// One bounded metadata pass per operation; ordinary key lookup remains O(1).
// Global sweeps report independent cleanup errors at Close. A requested entry's
// own integrity/absence is still checked by ReadAdvisory before any cache hit.
func (s *Local) sweepAdvisories(ctx context.Context, reserve uint64, skipRun string) {
	operation := ports.StorageOperationFromContext(ctx).String()
	limit := s.policy.Spec().CacheBytes
	pressure := reserve <= limit && (s.cacheCharged > limit-reserve)
	if operation == "" || operation == s.advisorySweepOperation && !pressure {
		return
	}
	s.advisorySweepOperation = operation
	percent := uint64(s.policy.Spec().CacheLowWaterPercent)
	lowWater := limit/100*percent + limit%100*percent/100
	if reserve <= limit && limit-reserve < lowWater {
		lowWater = limit - reserve
	}
	remainingEntries, remainingBytes := 64, uint64(64<<20)
	now := s.now().Unix()
	for _, record := range s.entries {
		if err := ctx.Err(); err != nil {
			s.advisoryCleanupErr = errors.Join(s.advisoryCleanupErr, err)
			break
		}
		if record.Kind != "advisory" || record.Run == skipRun || (record.Phase != "RETAINED" && record.Phase != "DELETING") {
			continue
		}
		expired := now >= record.Expiry || now >= record.Created && now-record.Created >= int64(s.policy.Spec().AdvisoryTTLSeconds)
		obsolete := s.advisoryIndex[record.CacheKey] != record.Run
		if record.Phase != "DELETING" && !expired && !obsolete && !(pressure && s.cacheCharged > lowWater) {
			continue
		}
		if remainingEntries == 0 || record.LogicalBytes > remainingBytes {
			continue
		}
		remainingEntries--
		remainingBytes -= record.LogicalBytes
		if err := s.removeAdvisory(ctx, record); err != nil {
			s.advisoryCleanupErr = errors.Join(s.advisoryCleanupErr, err)
		}
	}
}

var _ ports.AdvisoryStorage = (*Local)(nil)
