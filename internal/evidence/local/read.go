package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// ReadReference verifies an exact recorded subject. The opaque reference is
// checked against the configured trusted store; caller paths are not accepted.
func (s *Store) ReadReference(ctx context.Context, run domain.RunID, ref domain.EvidenceReference, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error) {
	if s == nil || ctx == nil || run.String() == "" || ref.ID().String() == "" || digest.String() == "" || ref.Handle() != "evidence:"+run.String()+":"+ref.ID().String() {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("recorded Evidence reference is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.Evidence{}, domain.ContentDigest{}, err
	}
	// Reject symlinks in each store path component before confined opening.
	path := filepath.VolumeName(s.root) + string(filepath.Separator)
	var rootInfo os.FileInfo
	for _, part := range strings.Split(strings.TrimPrefix(s.root, path), string(filepath.Separator)) {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence store path is unavailable")
		}
		rootInfo = info
	}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("open Evidence store")
	}
	defer root.Close()
	openedStore, err := root.Stat(".")
	if err != nil || rootInfo == nil || !os.SameFile(rootInfo, openedStore) {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence store identity changed")
	}
	runInfo, err := root.Lstat(run.String())
	if err != nil || !runInfo.IsDir() || runInfo.Mode()&os.ModeSymlink != 0 {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence Run is unavailable")
	}
	runRoot, err := root.OpenRoot(run.String())
	if err != nil {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("open Evidence Run")
	}
	defer runRoot.Close()
	openedRun, err := runRoot.Stat(".")
	if err != nil || !os.SameFile(runInfo, openedRun) {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence Run identity changed")
	}
	name := ref.ID().String() + ".json"
	info, err := runRoot.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 1<<20 {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence record is unavailable or exceeds bounds")
	}
	f, err := runRoot.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("open Evidence record")
	}
	opened, statErr := f.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		_ = f.Close()
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence record identity changed")
	}
	body, readErr := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	after, afterErr := f.Stat()
	closeErr := f.Close()
	current, currentErr := runRoot.Lstat(name)
	currentRun, currentRunErr := root.Lstat(run.String())
	currentStore, currentStoreErr := os.Lstat(s.root)
	if readErr != nil || afterErr != nil || closeErr != nil || currentErr != nil || currentRunErr != nil || currentStoreErr != nil || !os.SameFile(rootInfo, currentStore) || !os.SameFile(runInfo, currentRun) || !os.SameFile(info, current) || int64(len(body)) != info.Size() || after.Size() != info.Size() || after.ModTime() != info.ModTime() {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence record changed during read")
	}
	var doc record
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&doc) != nil {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence record is invalid")
	}
	canonical, err := json.Marshal(doc)
	if err != nil || !bytes.Equal(canonical, body) {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence record is not canonical")
	}
	claimed := doc.RecordSHA256
	doc.RecordSHA256 = ""
	unsigned, err := json.Marshal(doc)
	if err != nil {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence record cannot be verified")
	}
	hash := sha256.Sum256(unsigned)
	if claimed != hex.EncodeToString(hash[:]) || doc.RunID != run.String() || doc.EvidenceID != ref.ID().String() || doc.SourceID != identity.Source().String() || doc.Name != identity.Name() || doc.Version != identity.Version() || doc.Variant != identity.Variant() || doc.SHA256 != digest.String() {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence record integrity or subject differs")
	}
	check, err := domain.NewCheckID(doc.CheckID)
	if err != nil {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence check is invalid")
	}
	evidence, err := domain.NewEvidence(ref.ID(), check, identity, digest, doc.Kind, doc.Summary)
	if err != nil {
		return domain.Evidence{}, domain.ContentDigest{}, errors.New("evidence content is invalid")
	}
	content := sha256.Sum256(body)
	contentDigest, err := domain.NewSHA256Digest(hex.EncodeToString(content[:]))
	return evidence, contentDigest, err
}
