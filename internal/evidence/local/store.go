// Package local records normalized Evidence under a trusted local root.
package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rahoney/heliopause/internal/core/ports"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Store writes one immutable-style normalized JSON record per Evidence item.
type Store struct{ root string }

// NewStore constructs a Store for an explicit Evidence root, separate from Intake.
func NewStore(root string) (*Store, error) {
	if !filepath.IsAbs(root) {
		return nil, errors.New("evidence root must be absolute")
	}
	return &Store{root: filepath.Clean(root)}, nil
}

// Record atomically publishes one complete normalized Evidence batch and
// returns opaque references only after its Run directory is durable.
func (s *Store) Record(ctx context.Context, runID domain.RunID, evidence []domain.Evidence, checks ...domain.CheckExecution) (references []domain.EvidenceReference, resultErr error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.root == "" || runID.String() == "" || len(evidence) == 0 {
		return nil, errors.New("evidence store requires root, Run ID, and a non-empty batch")
	}
	// Profile bindings are supplied by the actual normalized checks. Legacy
	// callers remain readable as legacy; they cannot create v2 approval evidence.
	bound, bindingErr := domain.BindEvidenceProfiles(checks, evidence)
	if bindingErr != nil {
		return nil, bindingErr
	}
	evidence = bound
	retention := domain.DefaultStoragePolicy()
	if storage := ports.StorageFromContext(ctx); storage != nil {
		retention = storage.Policy()
	}
	if !retention.Valid() {
		return nil, errors.New("storage policy is unavailable")
	}
	documents := make([][]byte, 0, len(evidence))
	var totalBytes uint64
	seen := make(map[domain.EvidenceID]bool, len(evidence))
	for _, item := range evidence {
		if seen[item.ID()] {
			return nil, errors.New("evidence batch contains duplicate IDs")
		}
		seen[item.ID()] = true
		document, err := encodeRecord(runID, item)
		if err != nil {
			return nil, err
		}
		if uint64(len(document)) > retention.Spec().EvidenceBytes-totalBytes {
			return nil, errors.New("evidence batch exceeds finite retention capacity")
		}
		totalBytes += uint64(len(document))
		documents = append(documents, document)
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return nil, fmt.Errorf("create Evidence root: %w", err)
	}
	root, _, err := s.openRoot()
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	final := runID.String()
	if _, err := root.Lstat(final); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("evidence Run already exists or cannot be verified")
	}
	// The Run ID is already unpredictable and unique. Exclusive creation
	// preserves existing data without an additional pathname-only temp owner.
	temporary := "." + final + ".tmp"
	if err := root.Mkdir(temporary, 0700); err != nil {
		return nil, err
	}
	identity, err := root.Lstat(temporary)
	if err != nil {
		return nil, err
	}
	cleanup := true
	committed := false
	var lease ports.StorageLease
	var allocation *os.Root
	expected := make(map[string][]byte, len(evidence))
	for index, item := range evidence {
		expected[item.ID().String()+".json"] = documents[index]
	}
	defer func() {
		finishContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if cleanup {
			resultErr = errors.Join(resultErr, removeCreatedRun(finishContext, root, temporary, identity, expected))
		}
		if allocation != nil {
			resultErr = errors.Join(resultErr, allocation.Close())
		}
		if lease != nil {
			if !committed {
				resultErr = errors.Join(resultErr, lease.Abort(finishContext))
			}
			resultErr = errors.Join(resultErr, lease.Close())
		}
	}()
	allocation, err = root.OpenRoot(temporary)
	if err != nil {
		return nil, err
	}
	opened, err := allocation.Stat(".")
	if err != nil || !os.SameFile(identity, opened) {
		return nil, errors.New("new Evidence allocation changed")
	}
	if storage := ports.StorageFromContext(ctx); storage != nil {
		lease, err = storage.Reserve(ctx, ports.StorageRequest{Operation: ports.StorageOperationFromContext(ctx), Run: runID, Kind: "evidence", Temporary: temporary, Target: final, Bytes: totalBytes, Files: uint64(len(evidence))})
		if err != nil {
			return nil, err
		}
	}

	references = make([]domain.EvidenceReference, 0, len(evidence))
	for index, item := range evidence {
		if err := s.write(allocation, item.ID(), documents[index]); err != nil {
			return nil, err
		}
		reference, err := domain.NewEvidenceReference(item.ID(), "evidence:"+runID.String()+":"+item.ID().String())
		if err != nil {
			return nil, err
		}
		references = append(references, reference)
	}
	if err := syncDirectory(allocation); err != nil {
		return nil, err
	}
	if err := root.Rename(temporary, final); err != nil {
		return nil, fmt.Errorf("finalize Evidence Run: %w", err)
	}
	cleanup = false
	if err := syncDirectory(root); err != nil {
		finishContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		cleanupErr := removeCreatedRun(finishContext, root, final, identity, expected)
		cancel()
		if cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("remove uncertain Evidence Run: %w", cleanupErr))
		}
		return nil, err
	}
	if lease != nil {
		if err := lease.Commit(ctx); err != nil {
			return nil, err
		}
		committed = true
	}
	return references, nil
}

// DeleteRun removes one committed Evidence Run as an explicit retention action.
// It refuses symlinks, non-directories and paths outside the configured root.
func (s *Store) DeleteRun(ctx context.Context, runID domain.RunID) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.root == "" || runID.String() == "" {
		return errors.New("evidence retention requires root and Run ID")
	}
	root, _, err := s.openRoot()
	if err != nil {
		return err
	}
	defer root.Close()

	info, err := root.Lstat(runID.String())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect Evidence Run for deletion: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("evidence retention target is not a trusted Run directory")
	}
	if err := root.RemoveAll(runID.String()); err != nil {
		return fmt.Errorf("remove Evidence Run: %w", err)
	}
	if err := syncDirectory(root); err != nil {
		return fmt.Errorf("sync Evidence root after deletion: %w", err)
	}
	return nil
}

func encodeRecord(runID domain.RunID, evidence domain.Evidence) ([]byte, error) {
	payload := record{EvidenceID: evidence.ID().String(), RunID: runID.String(), CheckID: evidence.CheckID().String(), Kind: evidence.Kind(), Summary: evidence.Summary(), SourceID: evidence.Identity().Source().String(), Name: evidence.Identity().Name(), Version: evidence.Identity().Version(), Variant: evidence.Identity().Variant(), SHA256: evidence.Digest().String()}
	if evidence.ProfileDigest().String() != "" {
		payload.SchemaVersion = "helox.evidence/v2"
		payload.ProfileSHA256 = evidence.ProfileDigest().String()
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	payload.RecordSHA256 = hex.EncodeToString(sum[:])
	document, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return document, nil
}

func (s *Store) write(directory *os.Root, id domain.EvidenceID, document []byte) error {
	temporary, err := directory.OpenFile("."+id.String()+".tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create Evidence temporary record: %w", err)
	}
	temporaryPath := "." + id.String() + ".tmp"
	defer directory.Remove(temporaryPath)
	if _, err := temporary.Write(document); err != nil {
		return errors.Join(fmt.Errorf("write Evidence record: %w", err), closeEvidenceTemporary(temporary))
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync Evidence record: %w", err), closeEvidenceTemporary(temporary))
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Evidence record: %w", err)
	}
	if err := directory.Rename(temporaryPath, id.String()+".json"); err != nil {
		return fmt.Errorf("finalize Evidence record: %w", err)
	}
	return nil
}

func closeEvidenceTemporary(file *os.File) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("close Evidence temporary record: %w", err)
	}
	return nil
}

type record struct {
	SchemaVersion string `json:"schema_version,omitempty"`
	ProfileSHA256 string `json:"profile_sha256,omitempty"`
	EvidenceID    string `json:"evidence_id"`
	RunID         string `json:"run_id"`
	CheckID       string `json:"check_id"`
	Kind          string `json:"kind"`
	Summary       string `json:"summary"`
	SourceID      string `json:"source_id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Variant       string `json:"variant"`
	SHA256        string `json:"sha256"`
	RecordSHA256  string `json:"record_sha256"`
}

func syncDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("open Evidence directory for sync: %w", err)
	}
	err = directory.Sync()
	closeErr := directory.Close()
	if err != nil {
		return fmt.Errorf("sync Evidence directory: %w", err)
	}
	return closeErr
}

// Reuse the reader's existing ancestor and descriptor identity checks for
// writing and explicit retention as well. Each caller keeps this root pinned.
func (s *Store) openRoot() (*os.Root, os.FileInfo, error) {
	path := filepath.VolumeName(s.root) + string(filepath.Separator)
	var info os.FileInfo
	for _, part := range strings.Split(strings.TrimPrefix(s.root, path), string(filepath.Separator)) {
		path = filepath.Join(path, part)
		var err error
		info, err = os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, nil, errors.New("evidence store path is unavailable")
		}
	}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, nil, err
	}
	opened, err := root.Stat(".")
	if err != nil || info == nil || !os.SameFile(info, opened) {
		return nil, nil, errors.Join(errors.New("evidence store identity changed"), root.Close())
	}
	return root, info, nil
}

// Failed publication can remove only this newly created directory and exact
// documents already written by this batch. Unexpected members are preserved.
func removeCreatedRun(ctx context.Context, root *os.Root, name string, identity os.FileInfo, expected map[string][]byte) (resultErr error) {
	current, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || identity == nil || !current.IsDir() || !os.SameFile(identity, current) {
		return errors.New("evidence cleanup ownership is unknown")
	}
	owned, err := root.OpenRoot(name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, owned.Close()) }()
	opened, err := owned.Stat(".")
	if err != nil || !os.SameFile(identity, opened) {
		return errors.New("evidence cleanup directory changed")
	}
	directory, err := owned.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := directory.ReadDir(len(expected) + 1)
	closeErr := directory.Close()
	if (readErr != nil && readErr != io.EOF) || closeErr != nil || len(entries) > len(expected) {
		return errors.Join(errors.New("evidence cleanup inventory is unknown"), readErr, closeErr)
	}
	identities := make(map[string]os.FileInfo, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		document, known := expected[entry.Name()]
		info, err := owned.Lstat(entry.Name())
		if !known || err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(document)) || info.Mode().Perm() != 0600 {
			return errors.New("evidence cleanup contains an unknown member")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 {
			return errors.New("evidence cleanup member ownership is unknown")
		}
		file, err := owned.OpenFile(entry.Name(), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		held, statErr := file.Stat()
		body, readErr := io.ReadAll(io.LimitReader(file, int64(len(document))+1))
		after, afterErr := file.Stat()
		closeErr := file.Close()
		if statErr != nil || afterErr != nil || readErr != nil || closeErr != nil || !os.SameFile(info, held) || !os.SameFile(info, after) || !bytes.Equal(body, document) || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
			return errors.New("evidence cleanup member changed")
		}
		identities[entry.Name()] = info
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := owned.Lstat(entry.Name())
		if err != nil || !os.SameFile(identities[entry.Name()], info) {
			return errors.New("evidence cleanup member replaced")
		}
		if err := owned.Remove(entry.Name()); err != nil {
			return err
		}
	}
	current, err = root.Lstat(name)
	if err != nil || !os.SameFile(identity, current) {
		return errors.New("evidence cleanup parent replaced")
	}
	if err := root.Remove(name); err != nil {
		return err
	}
	return syncDirectory(root)
}
