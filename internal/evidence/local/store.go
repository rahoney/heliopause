// Package local records normalized Evidence under a trusted local root.
package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
func (s *Store) Record(ctx context.Context, runID domain.RunID, evidence []domain.Evidence) (references []domain.EvidenceReference, resultErr error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.root == "" || runID.String() == "" || len(evidence) == 0 {
		return nil, errors.New("evidence store requires root, Run ID, and a non-empty batch")
	}
	seen := make(map[domain.EvidenceID]bool, len(evidence))
	for _, item := range evidence {
		if seen[item.ID()] {
			return nil, errors.New("evidence batch contains duplicate IDs")
		}
		seen[item.ID()] = true
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return nil, fmt.Errorf("create Evidence root: %w", err)
	}
	runDirectory := filepath.Join(s.root, runID.String())
	if filepath.Dir(runDirectory) != s.root {
		return nil, errors.New("evidence Run directory escapes root")
	}
	if _, err := os.Lstat(runDirectory); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("evidence Run directory already exists or cannot be verified")
	}
	temporary, err := os.MkdirTemp(s.root, "."+runID.String()+".tmp-")
	if err != nil {
		return nil, fmt.Errorf("create Evidence temporary Run directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			if err := os.RemoveAll(temporary); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove incomplete Evidence Run: %w", err))
			}
		}
	}()
	if err := os.Chmod(temporary, 0o700); err != nil {
		return nil, fmt.Errorf("protect Evidence temporary Run directory: %w", err)
	}
	references = make([]domain.EvidenceReference, 0, len(evidence))
	for _, item := range evidence {
		if err := s.write(temporary, runID, item); err != nil {
			return nil, err
		}
		reference, err := domain.NewEvidenceReference(item.ID(), "evidence:"+runID.String()+":"+item.ID().String())
		if err != nil {
			return nil, err
		}
		references = append(references, reference)
	}
	if err := syncDirectory(temporary); err != nil {
		return nil, err
	}
	if err := os.Rename(temporary, runDirectory); err != nil {
		return nil, fmt.Errorf("finalize Evidence Run: %w", err)
	}
	cleanup = false
	if err := syncDirectory(s.root); err != nil {
		cleanupErr := os.RemoveAll(runDirectory)
		if cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("remove uncertain Evidence Run: %w", cleanupErr))
		}
		return nil, err
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
	runDirectory := filepath.Join(s.root, runID.String())
	if filepath.Dir(runDirectory) != s.root {
		return errors.New("evidence retention path escapes root")
	}
	info, err := os.Lstat(runDirectory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect Evidence Run for deletion: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("evidence retention target is not a trusted Run directory")
	}
	if err := os.RemoveAll(runDirectory); err != nil {
		return fmt.Errorf("remove Evidence Run: %w", err)
	}
	if err := syncDirectory(s.root); err != nil {
		return fmt.Errorf("sync Evidence root after deletion: %w", err)
	}
	return nil
}

func (s *Store) write(directory string, runID domain.RunID, evidence domain.Evidence) error {
	payload := record{EvidenceID: evidence.ID().String(), RunID: runID.String(), CheckID: evidence.CheckID().String(), Kind: evidence.Kind(), Summary: evidence.Summary(), SourceID: evidence.Identity().Source().String(), Name: evidence.Identity().Name(), Version: evidence.Identity().Version(), Variant: evidence.Identity().Variant(), SHA256: evidence.Digest().String()}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canonical)
	payload.RecordSHA256 = hex.EncodeToString(sum[:])
	document, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	temporary, err := os.OpenFile(filepath.Join(directory, "."+evidence.ID().String()+".tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create Evidence temporary record: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(document); err != nil {
		return errors.Join(fmt.Errorf("write Evidence record: %w", err), closeEvidenceTemporary(temporary))
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync Evidence record: %w", err), closeEvidenceTemporary(temporary))
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Evidence record: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, evidence.ID().String()+".json")); err != nil {
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
	EvidenceID   string `json:"evidence_id"`
	RunID        string `json:"run_id"`
	CheckID      string `json:"check_id"`
	Kind         string `json:"kind"`
	Summary      string `json:"summary"`
	SourceID     string `json:"source_id"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Variant      string `json:"variant"`
	SHA256       string `json:"sha256"`
	RecordSHA256 string `json:"record_sha256"`
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
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
