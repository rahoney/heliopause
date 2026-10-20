//go:build linux

package githubrelease

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// BootstrapMaterials returns frozen source facts, never a provenance or Policy
// verdict. The ordinary bootstrap adapter must verify all selected subjects.
func (s *HeliopauseReleaseSet) BootstrapMaterials() (releaseinstall.ReleaseManifest, [][]byte, error) {
	if s == nil {
		return releaseinstall.ReleaseManifest{}, nil, errors.New("native release set is unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.view == nil {
		return releaseinstall.ReleaseManifest{}, nil, errors.New("native release set is unresolved or closed")
	}
	proofs := make([][]byte, len(s.bundles))
	for index, body := range s.bundles {
		proofs[index] = append([]byte(nil), body...)
	}
	return s.manifest, proofs, nil
}

type bootstrapMemberReader struct {
	reader   io.Reader
	file     *os.File
	once     sync.Once
	closeErr error
}

func (r *bootstrapMemberReader) Read(body []byte) (int, error) { return r.reader.Read(body) }
func (r *bootstrapMemberReader) Close() error {
	r.once.Do(func() { r.closeErr = r.file.Close() })
	return r.closeErr
}

// OpenBootstrapMember duplicates a held FD, not an acquired pathname. Its
// independent lifetime survives Source.Close; copied bytes still need hashing.
func (s *HeliopauseReleaseSet) OpenBootstrapMember(ctx context.Context, name string) (io.ReadCloser, uint64, error) {
	if ctx == nil || s == nil {
		return nil, 0, errors.New("native member context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, found := s.entries[name]
	if s.closed || s.view == nil || !found {
		return nil, 0, errors.New("native member is absent from the frozen set")
	}
	s.view.mu.RLock()
	defer s.view.mu.RUnlock()
	held, found := s.view.files[entry.Reference.Locator()]
	if s.view.closed || !found || held.entry.Acquired.Identity() != entry.Acquired.Identity() || held.entry.Acquired.Digest() != entry.Acquired.Digest() {
		return nil, 0, errors.New("native member binding changed")
	}
	// os.Open adds CLOEXEC atomically. /proc/self/fd refers to the locked live
	// descriptor even if another process replaced or unlinked its old pathname.
	file, err := os.Open(fmt.Sprintf("/proc/self/fd/%d", held.file.Fd()))
	if err != nil {
		return nil, 0, err
	}
	original, err := held.file.Stat()
	opened, statErr := file.Stat()
	if err != nil || statErr != nil || !os.SameFile(original, opened) {
		_ = file.Close()
		return nil, 0, errors.New("native member descriptor changed")
	}
	size := entry.Acquired.SizeBytes()
	reader := frozenContextReader{ctx: ctx, reader: io.NewSectionReader(file, 0, int64(size)+1)}
	return &bootstrapMemberReader{reader: reader, file: file}, size, nil
}
