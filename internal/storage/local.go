// Package storage owns common durable resource accounting. Payload creation,
// publication and removal remain with the existing concrete storage owners.
package storage

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

const maxJournalBytes = 16 << 20

// Local keeps an indexed append journal. Ordinary reservation/commit work is
// independent of retained run count; another process's new tail is replayed once.
// This foundation accounts Evidence first. It does not claim cache GC or adopt
// old payloads as owned resources.
type Local struct {
	owned                                                      map[string]bool
	inspected                                                  map[domain.RunID]bool
	closed                                                     bool
	activeUses                                                 uint64
	uncertainUse                                               bool
	mu                                                         sync.Mutex
	root, evidence, intake, state                              *os.Root
	journal                                                    *os.File
	rootInfo, evidenceInfo, intakeInfo, stateInfo, journalInfo os.FileInfo
	path                                                       string
	policy                                                     domain.StoragePolicy
	offset                                                     int64
	entries                                                    map[string]resourceRecord
	charged, runs, pending                                     uint64
	legacyBytes, legacyRuns, legacyIntake                      uint64
	evidenceCharged, evidenceRuns                              uint64
	now                                                        func() time.Time
}

type fileIdentity struct {
	Device, Inode uint64
	UID           uint32
	Mode          uint32
}
type resourceRecord struct {
	Schema        string       `json:"schema"`
	Policy        string       `json:"policy_sha256"`
	Operation     string       `json:"operation"`
	Run           string       `json:"run"`
	Kind          string       `json:"kind"`
	Temporary     string       `json:"temporary"`
	Target        string       `json:"target"`
	Identity      fileIdentity `json:"identity"`
	ReservedBytes uint64       `json:"reserved_bytes"`
	ReservedFiles uint64       `json:"reserved_files"`
	BaseBytes     uint64       `json:"base_bytes,omitempty"`
	LogicalBytes  uint64       `json:"logical_bytes"`
	Content       string       `json:"content_sha256,omitempty"`
	Phase         string       `json:"phase"`
	Created       int64        `json:"created"`
	Expiry        int64        `json:"expiry"`
}
type journalEvent struct {
	Action   string         `json:"action"`
	Resource resourceRecord `json:"resource"`
}
type resourceLease struct {
	store     *Local
	directory *os.Root
	record    resourceRecord
}

// NewLocal configures an inert owned root and policy. Legacy Evidence contributes to admission
// accounting without gaining automatic deletion authority.
func NewLocal(ctx context.Context, path string, policy domain.StoragePolicy) (*Local, error) {
	if ctx == nil {
		return nil, errors.New("storage context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || !policy.Valid() {
		return nil, errors.New("storage requires an absolute owned root and validated policy")
	}
	return &Local{path: path, policy: policy, entries: make(map[string]resourceRecord), owned: make(map[string]bool), inspected: make(map[domain.RunID]bool), now: time.Now}, nil
}

// Composition is inert. Only an actual reservation opens the private ledger;
// help, invalid requests and unavailable runtime preparation create no files.
func (s *Local) open(ctx context.Context) (resultErr error) {
	if s == nil || ctx == nil {
		return errors.New("storage and context are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("storage is closed")
	}
	if s.journal != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	inventoryContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	path := s.path
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil || canonical != path {
		return errors.New("storage root must not contain symlinks")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	s.root = root
	ok := false
	defer func() {
		if !ok {
			s.closed = true
			resultErr = errors.Join(resultErr, s.closeDescriptors())
		}
	}()
	s.rootInfo, err = root.Stat(".")
	if err != nil || !ownedDirectory(s.rootInfo) {
		return errors.New("storage root is not owned and private")
	}
	for _, name := range []string{"evidence", "intake", ".storage"} {
		if err := root.Mkdir(name, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err := root.Lstat(name)
		if err != nil || !privateDirectory(info) {
			return errors.New("storage child is not an owned private directory")
		}
	}
	s.evidence, err = root.OpenRoot("evidence")
	if err != nil {
		return err
	}
	s.evidenceInfo, err = s.evidence.Stat(".")
	if err != nil {
		return err
	}
	s.intake, err = root.OpenRoot("intake")
	if err != nil {
		return err
	}
	s.intakeInfo, err = s.intake.Stat(".")
	if err != nil {
		return err
	}
	s.state, err = root.OpenRoot(".storage")
	if err != nil {
		return err
	}
	s.stateInfo, err = s.state.Stat(".")
	if err != nil {
		return err
	}
	s.journal, err = s.state.OpenFile("resources.jsonl", os.O_CREATE|os.O_RDWR|os.O_APPEND|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	s.journalInfo, err = s.journal.Stat()
	if err != nil || !privateFile(s.journalInfo) {
		return errors.New("storage journal is not an owned single-link private file")
	}
	if s.journalInfo.Sys().(*syscall.Stat_t).Dev != s.evidenceInfo.Sys().(*syscall.Stat_t).Dev {
		return errors.New("evidence and its reservation ledger must share a filesystem")
	}
	if s.journalInfo.Sys().(*syscall.Stat_t).Dev != s.intakeInfo.Sys().(*syscall.Stat_t).Dev {
		return errors.New("intake and its reservation ledger must share a filesystem")
	}
	if err := errors.Join(s.journal.Sync(), syncRoot(s.state), syncRoot(s.root)); err != nil {
		return err
	}
	if err := s.locked(inventoryContext, func() error {
		var inventoryErr error
		s.legacyBytes, s.legacyRuns, inventoryErr = s.legacyEvidenceUsage(inventoryContext)
		if inventoryErr != nil {
			return inventoryErr
		}
		s.legacyIntake, inventoryErr = s.legacyIntakeUsage(inventoryContext)
		return inventoryErr
	}); err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *Local) Policy() domain.StoragePolicy { return s.policy }

func (s *Local) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return s.closeDescriptors()
}
func (s *Local) closeDescriptors() error {
	var errs []error
	if s.journal != nil {
		errs = append(errs, s.journal.Close())
		s.journal = nil
	}
	if s.evidence != nil {
		errs = append(errs, s.evidence.Close())
		s.evidence = nil
	}
	if s.intake != nil {
		errs = append(errs, s.intake.Close())
		s.intake = nil
	}
	if s.state != nil {
		errs = append(errs, s.state.Close())
		s.state = nil
	}
	if s.root != nil {
		errs = append(errs, s.root.Close())
		s.root = nil
	}
	return errors.Join(errs...)
}

func (s *Local) withLock(ctx context.Context, work func() error) error {
	if err := s.open(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.journal == nil {
		return errors.New("storage is closed")
	}
	return s.locked(ctx, work)
}
func (s *Local) locked(ctx context.Context, work func() error) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := syscall.Flock(int(s.journal.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("storage journal is busy or unavailable: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, syscall.Flock(int(s.journal.Fd()), syscall.LOCK_UN)) }()
	current, err := os.Lstat(s.path)
	if err != nil || !ownedDirectory(current) || !os.SameFile(current, s.rootInfo) {
		return errors.New("storage root changed")
	}
	current, err = s.root.Lstat("evidence")
	if err != nil || !privateDirectory(current) || !os.SameFile(current, s.evidenceInfo) {
		return errors.New("evidence root changed")
	}
	current, err = s.root.Lstat("intake")
	if err != nil || !privateDirectory(current) || !os.SameFile(current, s.intakeInfo) {
		return errors.New("intake root changed")
	}
	current, err = s.root.Lstat(".storage")
	if err != nil || !privateDirectory(current) || !os.SameFile(current, s.stateInfo) {
		return errors.New("storage journal directory changed")
	}
	current, err = s.state.Lstat("resources.jsonl")
	if err != nil || !privateFile(current) || !os.SameFile(current, s.journalInfo) {
		return errors.New("storage journal changed or acquired a hardlink")
	}
	if err := s.refresh(); err != nil {
		return err
	}
	return work()
}

func (s *Local) refresh() error {
	info, err := s.journal.Stat()
	if err != nil {
		return err
	}
	if info.Size() < s.offset || info.Size() > maxJournalBytes {
		return errors.New("storage journal truncated or requires bounded maintenance")
	}
	if info.Size() == s.offset {
		return nil
	}
	reader := bufio.NewReader(io.NewSectionReader(s.journal, s.offset, info.Size()-s.offset))
	for {
		line, err := reader.ReadSlice('\n')
		if err == io.EOF && len(line) == 0 {
			break
		}
		if err != nil {
			return errors.New("storage journal has an incomplete or oversized event; accounting recovery is required")
		}
		var event journalEvent
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&event) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			return errors.New("storage journal event is invalid")
		}
		if err := s.apply(event); err != nil {
			return err
		}
		s.offset += int64(len(line))
	}
	return nil
}

func (s *Local) validate(event journalEvent) error {
	r := event.Resource
	if r.Schema != "helox.storage/v1" || !validResourceHandles(r.Kind, r.Temporary, r.Target, r.Run) || r.ReservedBytes == 0 || r.ReservedBytes > math.MaxInt64 || r.BaseBytes > r.ReservedBytes || r.ReservedFiles == 0 || r.ReservedFiles > 1<<20 || (r.Kind == "evidence" && r.BaseBytes != 0) || r.Created <= 0 || r.Expiry < r.Created || (r.Kind == "evidence" && r.Expiry == r.Created) || r.Identity.UID != uint32(os.Geteuid()) || r.Identity.Inode == 0 {
		return errors.New("storage resource event does not match the active policy or trusted handle contract")
	}
	if _, err := domain.NewSHA256Digest(r.Policy); err != nil {
		return err
	}
	if _, err := domain.ParseOperationID(r.Operation); err != nil {
		return err
	}
	if _, err := domain.ParseRunID(r.Run); err != nil {
		return err
	}
	before, exists := s.entries[resourceKey(r.Kind, r.Run)]
	switch event.Action {
	case "reserve":
		if exists || r.BaseBytes != 0 || r.Phase != "IN_USE" || r.LogicalBytes != 0 || r.Content != "" || s.charged > math.MaxUint64-r.ReservedBytes {
			return errors.New("invalid or duplicate storage reservation")
		}

	case "expand":
		expected := before
		expected.Phase, expected.LogicalBytes, expected.Content = "IN_USE", 0, ""
		expected.BaseBytes = before.LogicalBytes
		expected.ReservedBytes, expected.ReservedFiles = r.ReservedBytes, r.ReservedFiles
		if !exists || before.Kind != "intake" || before.Phase != "RETAINED" || r != expected || r.ReservedBytes <= r.BaseBytes || r.ReservedFiles <= before.ReservedFiles || s.charged > math.MaxUint64-(r.ReservedBytes-r.BaseBytes) {
			return errors.New("invalid intake expansion reservation")
		}

	case "commit":
		if !exists || before.Phase != "IN_USE" || r.Phase != "RETAINED" || !sameAllocation(before, r) || r.LogicalBytes > r.ReservedBytes {
			return errors.New("invalid storage publication")
		}
		if _, err := domain.NewSHA256Digest(r.Content); err != nil {
			return err
		}

	case "deleting":
		if !exists || r.Kind != "intake" || before.Phase != "RETAINED" || r.Phase != "DELETING" || !sameAllocation(before, r) || before.Content != r.Content || before.LogicalBytes != r.LogicalBytes {
			return errors.New("invalid intake deletion transition")
		}
	case "removed":
		if !exists || r.Kind != "intake" || before.Phase != "DELETING" || r.Phase != "REMOVED" || !sameAllocation(before, r) || before.Content != r.Content || before.LogicalBytes != r.LogicalBytes {
			return errors.New("invalid intake removal completion")
		}
	case "abort":
		if !exists || before != r || before.Phase != "IN_USE" {
			return errors.New("invalid storage reservation release")
		}

	default:
		return errors.New("unknown storage lifecycle event")
	}
	return nil
}

func (s *Local) apply(event journalEvent) error {
	if err := s.validate(event); err != nil {
		return err
	}
	r := event.Resource
	switch event.Action {
	case "reserve":
		if r.Kind == "evidence" {
			s.evidenceCharged += r.ReservedBytes
			s.evidenceRuns++
		}
		s.pending += r.ReservedBytes
		s.charged += r.ReservedBytes
		s.runs++
		s.entries[resourceKey(r.Kind, r.Run)] = r
	case "expand":
		s.pending += r.ReservedBytes - r.BaseBytes
		s.charged += r.ReservedBytes - r.BaseBytes
		s.entries[resourceKey(r.Kind, r.Run)] = r
	case "commit":
		if r.Kind == "evidence" {
			s.evidenceCharged -= r.ReservedBytes
			s.evidenceCharged += r.LogicalBytes
		}
		s.pending -= r.ReservedBytes - r.BaseBytes
		s.charged -= r.ReservedBytes
		s.charged += r.LogicalBytes
		s.entries[resourceKey(r.Kind, r.Run)] = r
	case "deleting":
		s.entries[resourceKey(r.Kind, r.Run)] = r
	case "removed":
		s.charged -= r.LogicalBytes
		s.runs--
		delete(s.entries, resourceKey(r.Kind, r.Run))
	case "abort":
		if r.Kind == "evidence" {
			s.evidenceCharged -= r.ReservedBytes
			s.evidenceRuns--
		}
		s.pending -= r.ReservedBytes - r.BaseBytes
		s.charged -= r.ReservedBytes
		s.runs--
		delete(s.entries, resourceKey(r.Kind, r.Run))
	}
	return nil
}

func sameAllocation(a, b resourceRecord) bool {
	a.Phase = b.Phase
	a.LogicalBytes = b.LogicalBytes
	a.Content = b.Content
	return a == b
}

func (s *Local) append(event journalEvent) error {
	if err := s.validate(event); err != nil {
		return err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if len(body) > 4096 || s.offset > maxJournalBytes-int64(len(body)) {
		return errors.New("storage journal capacity exhausted; maintenance is required")
	}
	n, writeErr := s.journal.Write(body)
	if writeErr != nil || n != len(body) {
		return errors.Join(errors.New("storage journal write is uncertain"), writeErr)
	}
	if err := s.journal.Sync(); err != nil {
		return err
	}
	if err := s.apply(event); err != nil {
		return err
	}
	s.offset += int64(len(body))
	return nil
}

func (s *Local) Reserve(ctx context.Context, request ports.StorageRequest) (lease ports.StorageLease, resultErr error) {
	if !validResourceHandles(request.Kind, request.Temporary, request.Target, request.Run.String()) || request.Operation.String() == "" || request.Run.String() == "" || request.Bytes == 0 || request.Files == 0 || request.Files > 1<<20 || request.Extend && request.Kind != "intake" {
		return nil, errors.New("invalid storage reservation")
	}
	var directory *os.Root
	defer func() {
		if resultErr != nil && directory != nil {
			resultErr = errors.Join(resultErr, directory.Close())
		}
	}()
	resultErr = s.withLock(ctx, func() error {
		payload := s.resourceRoot(request.Kind)
		key := resourceKey(request.Kind, request.Run.String())
		previous, exists := s.entries[key]
		if exists != request.Extend || request.Extend && (!s.owned[key] || previous.Operation != request.Operation.String() || previous.Phase != "RETAINED") {
			return errors.New("storage Run is already registered or expansion ownership is unavailable")
		}
		info, err := payload.Lstat(request.Temporary)
		if err != nil || !privateDirectory(info) {
			return errors.New("new storage directory is not owned")
		}
		if request.Temporary != request.Target {
			if _, err := payload.Lstat(request.Target); !errors.Is(err, os.ErrNotExist) {
				return errors.New("storage publication target already exists or is unknown")
			}
		}
		directory, err = payload.OpenRoot(request.Temporary)
		if err != nil {
			return err
		}
		opened, err := directory.Stat(".")
		if err != nil || !os.SameFile(opened, info) {
			return errors.New("evidence allocation identity changed")
		}
		if request.Extend {
			if !matchesIdentity(info, previous.Identity) || request.Bytes > math.MaxInt64-previous.LogicalBytes || request.Files > (1<<20)-previous.ReservedFiles {
				return errors.New("intake expansion exceeds its identity or finite reservation")
			}
			bytes, digest, err := measureResource(ctx, directory, "intake", previous.ReservedBytes, previous.ReservedFiles, nil)
			if err != nil || bytes != previous.LogicalBytes || digest != previous.Content {
				return errors.Join(errors.New("intake changed before expansion"), err)
			}
		} else {
			f, err := directory.Open(".")
			if err != nil {
				return err
			}
			items, readErr := f.ReadDir(1)
			closeErr := f.Close()
			if len(items) != 0 || readErr != io.EOF || closeErr != nil {
				return errors.New("storage registration must precede payload writes")
			}
		}
		// Existing unregistered runs are counted, never silently adopted/deleted.
		legacyBytes, legacyRuns := s.legacyBytes, s.legacyRuns
		limits := s.policy.Spec()
		if request.Kind == "evidence" && (s.evidenceCharged > limits.EvidenceBytes || legacyBytes > limits.EvidenceBytes-s.evidenceCharged || request.Bytes > limits.EvidenceBytes-s.evidenceCharged-legacyBytes || s.evidenceRuns+legacyRuns >= uint64(limits.EvidenceRuns)) {
			return errors.New("evidence retention quota is full; unexpired or unowned records are protected")
		}
		if s.charged > limits.TotalBytes || legacyBytes > limits.TotalBytes-s.charged || s.legacyIntake > limits.TotalBytes-s.charged-legacyBytes || request.Bytes > limits.TotalBytes-s.charged-legacyBytes-s.legacyIntake {
			return errors.New("managed storage quota is full")
		}

		var filesystem syscall.Statfs_t
		if err := syscall.Fstatfs(int(s.journal.Fd()), &filesystem); err != nil {
			return err
		}
		if filesystem.Bsize <= 0 || uint64(filesystem.Bavail) > math.MaxUint64/uint64(filesystem.Bsize) {
			return errors.New("filesystem available capacity is unknown")
		}
		available := uint64(filesystem.Bavail) * uint64(filesystem.Bsize)
		if s.pending > available || request.Bytes > available-s.pending || limits.MinimumFreeBytes > available-s.pending-request.Bytes {
			return errors.New("filesystem storage reserve would be exhausted")
		}
		stamp := s.now().Unix()
		stat := info.Sys().(*syscall.Stat_t)
		expiry := stamp
		if request.Kind == "evidence" {
			expiry += int64(limits.EvidenceTTLSeconds)
		}
		record := resourceRecord{Schema: "helox.storage/v1", Policy: s.policy.Digest().String(), Operation: request.Operation.String(), Run: request.Run.String(), Kind: request.Kind, Temporary: request.Temporary, Target: request.Target, Identity: fileIdentity{uint64(stat.Dev), uint64(stat.Ino), stat.Uid, uint32(info.Mode())}, ReservedBytes: request.Bytes, ReservedFiles: request.Files, Phase: "IN_USE", Created: stamp, Expiry: expiry}
		action := "reserve"
		if request.Extend {
			record = previous
			record.Phase, record.LogicalBytes, record.Content = "IN_USE", 0, ""
			record.BaseBytes = previous.LogicalBytes
			record.ReservedBytes = previous.LogicalBytes + request.Bytes
			record.ReservedFiles = previous.ReservedFiles + request.Files
			action = "expand"
		}
		if err := s.append(journalEvent{action, record}); err != nil {
			return err
		}
		s.owned[resourceKey(record.Kind, record.Run)] = true
		lease = &resourceLease{store: s, directory: directory, record: record}
		return nil
	})
	return lease, resultErr
}

func (l *resourceLease) Commit(ctx context.Context) error {
	if l == nil || l.store == nil || l.directory == nil {
		return errors.New("storage lease is closed")
	}
	return l.store.withLock(ctx, func() error {
		current, exists := l.store.entries[resourceKey(l.record.Kind, l.record.Run)]
		if !exists || current != l.record {
			return errors.New("storage lease changed")
		}
		info, err := l.store.resourceRoot(l.record.Kind).Lstat(l.record.Target)
		pinned, pinnedErr := l.directory.Stat(".")
		if err != nil || pinnedErr != nil || !privateDirectory(info) || !os.SameFile(info, pinned) {
			return errors.New("published Evidence identity differs from allocation")
		}
		bytes, digest, err := measureResource(ctx, l.directory, l.record.Kind, l.record.ReservedBytes, l.record.ReservedFiles, nil)
		if err != nil {
			return err
		}
		// Payload and publication entries must be durable before the journal
		// makes the reservation reusable as a sealed resource.
		if err := errors.Join(syncRoot(l.directory), syncRoot(l.store.resourceRoot(l.record.Kind))); err != nil {
			return err
		}
		committed := l.record
		committed.Phase = "RETAINED"
		committed.LogicalBytes = bytes
		committed.Content = digest
		if err := l.store.append(journalEvent{"commit", committed}); err != nil {
			return err
		}
		l.record = committed
		return nil
	})
}

func (l *resourceLease) Abort(ctx context.Context) error {
	if l == nil || l.store == nil {
		return errors.New("storage lease is unavailable")
	}
	if l.record.Phase == "RETAINED" {
		return errors.New("retained Evidence cannot be released as an aborted reservation")
	}
	return l.store.withLock(ctx, func() error {
		for _, name := range []string{l.record.Temporary, l.record.Target} {
			if _, err := l.store.resourceRoot(l.record.Kind).Lstat(name); !errors.Is(err, os.ErrNotExist) {
				return errors.New("storage reservation remains accounted: payload exists or its absence is unknown")
			}
		}
		current, exists := l.store.entries[resourceKey(l.record.Kind, l.record.Run)]
		if !exists {
			return nil
		}
		if current != l.record {
			return errors.New("storage reservation changed before release")
		}
		if err := syncRoot(l.store.resourceRoot(l.record.Kind)); err != nil {
			return err
		}
		return l.store.append(journalEvent{"abort", l.record})
	})
}

func (l *resourceLease) Close() error {
	if l == nil || l.directory == nil {
		return nil
	}
	err := l.directory.Close()
	l.directory = nil
	return err
}

func plainHandle(value string) bool {
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value && !strings.ContainsAny(value, "/\\\x00")
}
func ownedDirectory(info os.FileInfo) bool {
	if info == nil || !info.IsDir() || info.Mode().Perm()&0700 != 0700 || info.Mode().Perm()&0022 != 0 || info.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid) != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}
func privateDirectory(info os.FileInfo) bool {
	return ownedDirectory(info) && info.Mode().Perm() == 0700
}
func privateFile(info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && stat.Nlink == 1
}

// Intake also owns immutable 0400 metadata. Journal and Evidence retain their
// stricter 0600 contracts; both payload modes prohibit foreign access and links.
func privateResourceFile(info os.FileInfo) bool {
	if privateFile(info) {
		return true
	}
	if info == nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0400 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && stat.Nlink == 1
}

type measuredMember struct {
	info   os.FileInfo
	sha256 string
}

func measureResource(ctx context.Context, root *os.Root, kind string, maxBytes, maxFiles uint64, capture map[string]measuredMember) (total uint64, digest string, resultErr error) {
	f, err := root.Open(".")
	if err != nil {
		return 0, "", err
	}
	defer func() { resultErr = errors.Join(resultErr, f.Close()) }()
	entries, err := f.ReadDir(int(maxFiles) + 1)
	if err != nil && err != io.EOF {
		return 0, "", err
	}
	if len(entries) == 0 || uint64(len(entries)) > maxFiles {
		return 0, "", errors.New("evidence file count differs from reservation")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	hash := sha256.New()
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return 0, "", err
		}
		info, err := root.Lstat(entry.Name())
		if err != nil || !privateResourceFile(info) || (kind == "evidence" && (!privateFile(info) || !strings.HasSuffix(entry.Name(), ".json"))) || info.Size() < 0 || uint64(info.Size()) > maxBytes-total {
			return 0, "", errors.New("evidence member exceeds ownership or byte reservation")
		}
		memberDigest, err := hashResourceMember(ctx, root, entry.Name(), info)
		if err != nil {
			return 0, "", err
		}
		if capture != nil {
			capture[entry.Name()] = measuredMember{info, memberDigest}
		}
		total += uint64(info.Size())
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00%s\n", entry.Name(), info.Size(), memberDigest)
	}
	return total, hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *Local) legacyEvidenceUsage(ctx context.Context) (total, runs uint64, resultErr error) {
	f, err := s.evidence.Open(".")
	if err != nil {
		return 0, 0, err
	}
	defer func() { resultErr = errors.Join(resultErr, f.Close()) }()
	entries, err := f.ReadDir(int(s.policy.Spec().EvidenceRuns) + 2)
	if err != nil && err != io.EOF {
		return 0, 0, err
	}
	if len(entries) > int(s.policy.Spec().EvidenceRuns)+1 {
		return 0, 0, errors.New("evidence inventory exceeds bounded retention capacity")
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}
		if _, registered := s.entries[resourceKey("evidence", entry.Name())]; registered {
			continue
		}
		if _, err := domain.ParseRunID(entry.Name()); err != nil {
			continue
		}
		info, err := s.evidence.Lstat(entry.Name())
		if err != nil || !privateDirectory(info) {
			return 0, 0, errors.New("unregistered Evidence ownership is unknown")
		}
		root, err := s.evidence.OpenRoot(entry.Name())
		if err != nil {
			return 0, 0, err
		}
		size, _, measureErr := measureResource(ctx, root, "evidence", s.policy.Spec().EvidenceBytes-total, 1<<20, nil)
		closeErr := root.Close()
		if err := errors.Join(measureErr, closeErr); err != nil {
			return 0, 0, err
		}
		total += size
		runs++
	}
	return total, runs, nil
}

func syncRoot(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

func resourceKey(kind, run string) string { return kind + ":" + run }
func validResourceHandles(kind, temporary, target, run string) bool {
	if target != run || !plainHandle(temporary) {
		return false
	}
	return kind == "evidence" && temporary != target || kind == "intake" && temporary == target
}
func (s *Local) resourceRoot(kind string) *os.Root {
	if kind == "intake" {
		return s.intake
	}
	return s.evidence
}

// Legacy intake contributes logical bytes without acquiring deletion authority.
// Current intake creation is flat and bounded; unknown nested/query states stop
// admission rather than being guessed away or recursively removed.
func (s *Local) legacyIntakeUsage(ctx context.Context) (total uint64, resultErr error) {
	directory, err := s.intake.Open(".")
	if err != nil {
		return 0, err
	}
	defer func() { resultErr = errors.Join(resultErr, directory.Close()) }()
	const maximumInventoryEntries = 100000
	entries, err := directory.ReadDir(maximumInventoryEntries + 1)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if len(entries) > maximumInventoryEntries {
		return 0, errors.New("intake inventory exceeds bounded capacity")
	}
	remaining := maximumInventoryEntries - len(entries)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if _, registered := s.entries[resourceKey("intake", entry.Name())]; registered {
			continue
		}
		info, err := s.intake.Lstat(entry.Name())
		if err != nil || !privateDirectory(info) {
			return 0, errors.New("legacy intake directory is unknown")
		}
		root, err := s.intake.OpenRoot(entry.Name())
		if err != nil {
			return 0, err
		}
		file, err := root.Open(".")
		if err != nil {
			return 0, errors.Join(err, root.Close())
		}
		members, readErr := file.ReadDir(remaining + 1)
		closeErr := file.Close()
		if (readErr != nil && readErr != io.EOF) || closeErr != nil || len(members) > remaining {
			return 0, errors.Join(errors.New("legacy intake members are unknown"), readErr, closeErr, root.Close())
		}
		remaining -= len(members)
		for _, member := range members {
			if err := ctx.Err(); err != nil {
				return 0, errors.Join(err, root.Close())
			}
			info, err := root.Lstat(member.Name())
			if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || uint64(info.Size()) > s.policy.Spec().TotalBytes-total {
				return 0, errors.Join(errors.New("legacy intake usage is unknown or exceeds quota"), root.Close())
			}
			total += uint64(info.Size())
		}
		if err := root.Close(); err != nil {
			return 0, err
		}
	}
	return total, nil
}

func (s *Local) NoteInspection(ctx context.Context, run domain.RunID, complete bool) error {
	if s == nil || ctx == nil || run.String() == "" {
		return errors.New("storage inspection requires a Run and context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("storage is closed")
	}
	s.inspected[run] = complete
	return nil
}

func (s *Local) TrackUse(ctx context.Context) (func(error), error) {
	if s == nil || ctx == nil {
		return nil, errors.New("storage consumer requires context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.activeUses == math.MaxUint64 {
		return nil, errors.New("storage consumer tracking is unavailable")
	}
	s.activeUses++
	done := false
	return func(cause error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if done {
			return
		}
		done = true
		s.activeUses--
		if cause != nil {
			s.uncertainUse = true
		}
	}, nil
}

// Finish runs after the command and its outer runtime finalizers. Policy/guest
// exit failure does not itself imply active consumers. Required check completion
// and tracked guard/execution outcomes remain independent deletion conditions.
// Restart reclamation is a separate, still protected path.
func (s *Local) Finish(ctx context.Context, outerFinalizersClosed bool) error {
	if s == nil || ctx == nil {
		return errors.New("storage finalization requires context")
	}
	if !outerFinalizersClosed {
		return nil
	}
	s.mu.Lock()
	anyIntake := false
	for key := range s.owned {
		if strings.HasPrefix(key, "intake:") {
			anyIntake = true
			break
		}
	}
	s.mu.Unlock()
	if !anyIntake {
		return nil
	} // inert composition must remain inert
	return s.withLock(ctx, func() error {
		if s.activeUses != 0 || s.uncertainUse {
			return errors.New("intake remains accounted: consumer finalization is active or unproven")
		}
		var failures []error
		for key := range s.owned {
			record, exists := s.entries[key]
			if !exists || record.Kind != "intake" {
				continue
			}
			// Existing concrete owners can already have removed their short-lived
			// source proof. Confirm durable absence before releasing accounting;
			// a stat/query error never counts as absence.
			if _, err := s.intake.Lstat(record.Target); errors.Is(err, os.ErrNotExist) && record.Phase == "RETAINED" {
				if err := syncRoot(s.intake); err != nil {
					return err
				}
				deleting := record
				deleting.Phase = "DELETING"
				if err := s.append(journalEvent{"deleting", deleting}); err != nil {
					return err
				}
				removed := deleting
				removed.Phase = "REMOVED"
				if err := s.append(journalEvent{"removed", removed}); err != nil {
					return err
				}
				continue
			}
			run, err := domain.ParseRunID(record.Run)
			if err != nil {
				failures = append(failures, err)
				continue
			}
			if !s.inspected[run] || record.Phase != "RETAINED" {
				failures = append(failures, fmt.Errorf("intake %s remains accounted: complete observation/last-use proof is unavailable", record.Run))
				continue
			}
		}
		// Inputs can be shared by another node's runtime. An incomplete member
		// protects the command's entire set, including its supporting inputs.
		if len(failures) != 0 {
			return errors.Join(failures...)
		}
		for key := range s.owned {
			record, exists := s.entries[key]
			if !exists || record.Kind != "intake" {
				continue
			}
			if err := s.removeSealedIntake(ctx, record); err != nil {
				failures = append(failures, fmt.Errorf("intake %s cleanup: %w", record.Run, err))
			}
		}
		return errors.Join(failures...)
	})
}

func (s *Local) removeSealedIntake(ctx context.Context, record resourceRecord) (resultErr error) {
	current, err := s.intake.Lstat(record.Target)
	if err != nil || !matchesIdentity(current, record.Identity) {
		return errors.New("intake cleanup directory identity is unknown")
	}
	root, err := s.intake.OpenRoot(record.Target)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	held, err := root.Stat(".")
	if err != nil || !os.SameFile(current, held) {
		return errors.New("intake cleanup directory changed")
	}
	bytes, digest, err := measureResource(ctx, root, "intake", record.ReservedBytes, record.ReservedFiles, nil)
	if err != nil {
		return err
	}
	if bytes != record.LogicalBytes || digest != record.Content {
		return errors.New("intake cleanup content changed since sealing")
	}
	deleting := record
	deleting.Phase = "DELETING"
	if err := s.append(journalEvent{"deleting", deleting}); err != nil {
		return err
	}
	// Re-read after the durable transition, immediately before removing members.
	members := make(map[string]measuredMember)
	bytes, digest, err = measureResource(ctx, root, "intake", record.ReservedBytes, record.ReservedFiles, members)
	if err != nil || bytes != record.LogicalBytes || digest != record.Content {
		return errors.Join(errors.New("intake cleanup readback changed"), err)
	}
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := directory.ReadDir(int(record.ReservedFiles) + 1)
	closeErr := directory.Close()
	if (readErr != nil && readErr != io.EOF) || closeErr != nil || uint64(len(entries)) > record.ReservedFiles {
		return errors.Join(errors.New("intake cleanup inventory changed"), readErr, closeErr)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		expected, known := members[entry.Name()]
		info, err := root.Lstat(entry.Name())
		if !known || err != nil || !privateResourceFile(info) || info.Mode() != expected.info.Mode() || !os.SameFile(expected.info, info) || info.Size() != expected.info.Size() || !info.ModTime().Equal(expected.info.ModTime()) {
			return errors.New("intake cleanup member ownership changed")
		}
		digest, err := hashResourceMember(ctx, root, entry.Name(), info)
		if err != nil || digest != expected.sha256 {
			return errors.Join(errors.New("intake cleanup member changed after inventory"), err)
		}
		if err := root.Remove(entry.Name()); err != nil {
			return err
		}
	}
	current, err = s.intake.Lstat(record.Target)
	if err != nil || !matchesIdentity(current, record.Identity) {
		return errors.New("intake cleanup parent changed")
	}
	if err := s.intake.Remove(record.Target); err != nil {
		return err
	}
	if err := syncRoot(s.intake); err != nil {
		return err
	}
	removed := record
	removed.Phase = "REMOVED"
	return s.append(journalEvent{"removed", removed})
}
func matchesIdentity(info os.FileInfo, identity fileIdentity) bool {
	if !privateDirectory(info) {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && uint64(stat.Dev) == identity.Device && uint64(stat.Ino) == identity.Inode && stat.Uid == identity.UID && uint32(info.Mode()) == identity.Mode
}

func hashResourceMember(ctx context.Context, root *os.Root, name string, info os.FileInfo) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", err
	}
	opened, statErr := file.Stat()
	hash := sha256.New()
	count, readErr := io.Copy(hash, io.LimitReader(file, info.Size()+1))
	after, afterErr := file.Stat()
	closeErr := file.Close()
	current, currentErr := root.Lstat(name)
	if statErr != nil || afterErr != nil || currentErr != nil || readErr != nil || closeErr != nil || !os.SameFile(info, opened) || !os.SameFile(info, after) || !os.SameFile(info, current) || count != info.Size() || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) || current.Size() != info.Size() || !current.ModTime().Equal(info.ModTime()) || !privateResourceFile(after) || !privateResourceFile(current) || after.Mode() != info.Mode() || current.Mode() != info.Mode() {
		return "", errors.New("resource member changed during accounting/readback")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
