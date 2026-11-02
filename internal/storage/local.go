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
	closed                                         bool
	mu                                             sync.Mutex
	root, evidence, state                          *os.Root
	journal                                        *os.File
	rootInfo, evidenceInfo, stateInfo, journalInfo os.FileInfo
	path                                           string
	policy                                         domain.StoragePolicy
	offset                                         int64
	entries                                        map[string]resourceRecord
	charged, runs, pending                         uint64
	legacyBytes, legacyRuns                        uint64
	now                                            func() time.Time
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
	return &Local{path: path, policy: policy, entries: make(map[string]resourceRecord), now: time.Now}, nil
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
	for _, name := range []string{"evidence", ".storage"} {
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
	if err := errors.Join(s.journal.Sync(), syncRoot(s.state), syncRoot(s.root)); err != nil {
		return err
	}
	if err := s.locked(inventoryContext, func() error {
		var inventoryErr error
		s.legacyBytes, s.legacyRuns, inventoryErr = s.legacyEvidenceUsage(inventoryContext)
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
	if r.Schema != "helox.storage/v1" || r.Kind != "evidence" || r.Target != r.Run || !plainHandle(r.Temporary) || r.Temporary == r.Target || r.ReservedBytes == 0 || r.ReservedBytes > math.MaxInt64 || r.ReservedFiles == 0 || r.Created <= 0 || r.Expiry <= r.Created || r.Identity.UID != uint32(os.Geteuid()) || r.Identity.Inode == 0 {
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
	before, exists := s.entries[r.Run]
	switch event.Action {
	case "reserve":
		if exists || r.Phase != "IN_USE" || r.LogicalBytes != 0 || r.Content != "" || s.charged > math.MaxUint64-r.ReservedBytes {
			return errors.New("invalid or duplicate storage reservation")
		}

	case "commit":
		if !exists || before.Phase != "IN_USE" || r.Phase != "RETAINED" || !sameAllocation(before, r) || r.LogicalBytes > r.ReservedBytes {
			return errors.New("invalid storage publication")
		}
		if _, err := domain.NewSHA256Digest(r.Content); err != nil {
			return err
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
		s.pending += r.ReservedBytes
		s.charged += r.ReservedBytes
		s.runs++
		s.entries[r.Run] = r
	case "commit":
		s.pending -= r.ReservedBytes
		s.charged -= r.ReservedBytes
		s.charged += r.LogicalBytes
		s.entries[r.Run] = r
	case "abort":
		s.pending -= r.ReservedBytes
		s.charged -= r.ReservedBytes
		s.runs--
		delete(s.entries, r.Run)
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
	if request.Kind != "evidence" || request.Operation.String() == "" || request.Run.String() == "" || request.Target != request.Run.String() || !plainHandle(request.Temporary) || request.Temporary == request.Target || request.Bytes == 0 || request.Files == 0 || request.Files > 1<<20 {
		return nil, errors.New("invalid Evidence storage reservation")
	}
	var directory *os.Root
	defer func() {
		if resultErr != nil && directory != nil {
			resultErr = errors.Join(resultErr, directory.Close())
		}
	}()
	resultErr = s.withLock(ctx, func() error {
		if _, exists := s.entries[request.Run.String()]; exists {
			return errors.New("storage Run is already registered")
		}
		info, err := s.evidence.Lstat(request.Temporary)
		if err != nil || !privateDirectory(info) {
			return errors.New("new Evidence temporary directory is not owned")
		}
		if _, err := s.evidence.Lstat(request.Target); !errors.Is(err, os.ErrNotExist) {
			return errors.New("evidence publication target already exists or is unknown")
		}
		directory, err = s.evidence.OpenRoot(request.Temporary)
		if err != nil {
			return err
		}
		opened, err := directory.Stat(".")
		if err != nil || !os.SameFile(opened, info) {
			return errors.New("evidence allocation identity changed")
		}
		f, err := directory.Open(".")
		if err != nil {
			return err
		}
		items, readErr := f.ReadDir(1)
		closeErr := f.Close()
		if len(items) != 0 || readErr != io.EOF || closeErr != nil {
			return errors.New("storage registration must precede payload writes")
		}
		// Existing unregistered runs are counted, never silently adopted/deleted.
		legacyBytes, legacyRuns := s.legacyBytes, s.legacyRuns
		limits := s.policy.Spec()
		if s.charged > limits.EvidenceBytes || legacyBytes > limits.EvidenceBytes-s.charged || request.Bytes > limits.EvidenceBytes-s.charged-legacyBytes || s.runs+legacyRuns >= uint64(limits.EvidenceRuns) {
			return errors.New("evidence retention quota is full; unexpired or unowned records are protected")
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
		record := resourceRecord{Schema: "helox.storage/v1", Policy: s.policy.Digest().String(), Operation: request.Operation.String(), Run: request.Run.String(), Kind: "evidence", Temporary: request.Temporary, Target: request.Target, Identity: fileIdentity{uint64(stat.Dev), uint64(stat.Ino), stat.Uid, uint32(info.Mode())}, ReservedBytes: request.Bytes, ReservedFiles: request.Files, Phase: "IN_USE", Created: stamp, Expiry: stamp + int64(limits.EvidenceTTLSeconds)}
		if err := s.append(journalEvent{"reserve", record}); err != nil {
			return err
		}
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
		current, exists := l.store.entries[l.record.Run]
		if !exists || current != l.record {
			return errors.New("storage lease changed")
		}
		info, err := l.store.evidence.Lstat(l.record.Target)
		pinned, pinnedErr := l.directory.Stat(".")
		if err != nil || pinnedErr != nil || !privateDirectory(info) || !os.SameFile(info, pinned) {
			return errors.New("published Evidence identity differs from allocation")
		}
		bytes, digest, err := measureEvidence(ctx, l.directory, l.record.ReservedBytes, l.record.ReservedFiles)
		if err != nil {
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
			if _, err := l.store.evidence.Lstat(name); !errors.Is(err, os.ErrNotExist) {
				return errors.New("storage reservation remains accounted: payload exists or its absence is unknown")
			}
		}
		current, exists := l.store.entries[l.record.Run]
		if !exists {
			return nil
		}
		if current != l.record {
			return errors.New("storage reservation changed before release")
		}
		if err := syncRoot(l.store.evidence); err != nil {
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

func measureEvidence(ctx context.Context, root *os.Root, maxBytes, maxFiles uint64) (total uint64, digest string, resultErr error) {
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
		if err != nil || !privateFile(info) || !strings.HasSuffix(entry.Name(), ".json") || info.Size() < 0 || uint64(info.Size()) > maxBytes-total {
			return 0, "", errors.New("evidence member exceeds ownership or byte reservation")
		}
		file, err := root.OpenFile(entry.Name(), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return 0, "", err
		}
		opened, statErr := file.Stat()
		memberHash := sha256.New()
		count, readErr := io.Copy(memberHash, io.LimitReader(file, info.Size()+1))
		after, afterErr := file.Stat()
		closeErr := file.Close()
		if statErr != nil || afterErr != nil || readErr != nil || closeErr != nil || !os.SameFile(info, opened) || !os.SameFile(info, after) || count != info.Size() || !info.ModTime().Equal(after.ModTime()) || !privateFile(after) {
			return 0, "", errors.New("evidence member changed during accounting")
		}
		total += uint64(count)
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00%x\n", entry.Name(), count, memberHash.Sum(nil))
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
		if _, registered := s.entries[entry.Name()]; registered {
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
		size, _, measureErr := measureEvidence(ctx, root, s.policy.Spec().EvidenceBytes-total, 1<<20)
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
