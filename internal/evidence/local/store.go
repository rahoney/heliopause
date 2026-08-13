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

// Record atomically writes a batch of normalized Evidence and returns opaque references.
func (s *Store) Record(ctx context.Context, runID domain.RunID, evidence []domain.Evidence) ([]domain.EvidenceReference, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.root == "" || runID.String() == "" || len(evidence) == 0 {
		return nil, errors.New("evidence store requires root, Run ID, and a non-empty batch")
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return nil, fmt.Errorf("create Evidence root: %w", err)
	}
	runDirectory := filepath.Join(s.root, runID.String())
	if filepath.Dir(runDirectory) != s.root {
		return nil, errors.New("evidence Run directory escapes root")
	}
	if err := os.Mkdir(runDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("create Evidence Run directory: %w", err)
	}
	references := make([]domain.EvidenceReference, 0, len(evidence))
	seen := make(map[domain.EvidenceID]bool, len(evidence))
	for _, item := range evidence {
		if seen[item.ID()] {
			return nil, errors.New("evidence batch contains duplicate IDs")
		}
		seen[item.ID()] = true
		if err := s.write(runDirectory, runID, item); err != nil {
			return nil, err
		}
		reference, err := domain.NewEvidenceReference(item.ID(), "evidence:"+runID.String()+":"+item.ID().String())
		if err != nil {
			return nil, err
		}
		references = append(references, reference)
	}
	return references, nil
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
		temporary.Close()
		return fmt.Errorf("write Evidence record: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync Evidence record: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Evidence record: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, evidence.ID().String()+".json")); err != nil {
		return fmt.Errorf("finalize Evidence record: %w", err)
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
