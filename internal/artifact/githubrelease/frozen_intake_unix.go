//go:build linux || darwin

package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// FrozenEntry binds only existing acquired bytes. It carries no Policy or
// provenance approval; the normal inspection workflow must still run.
type FrozenEntry struct {
	Reference domain.ArtifactReference
	Resolved  domain.ResolvedArtifact
	Acquired  domain.AcquiredArtifact
}

type frozenFile struct {
	entry FrozenEntry
	file  *os.File
}

// FrozenIntake implements Artifact without a network client. Each new Run gets
// a rehashed copy from the held original FD, never a resolve/download fallback.
type FrozenIntake struct {
	mu       sync.RWMutex
	root     *os.Root
	rootPath string
	rootInfo os.FileInfo
	files    map[string]frozenFile
	closed   bool
	closeErr error
}

func NewFrozenIntake(ctx context.Context, intakeRoot string, entries []FrozenEntry) (*FrozenIntake, error) {
	if ctx == nil {
		return nil, errors.New("frozen intake context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || len(entries) == 0 || len(entries) > maximumFrozenAssets {
		return nil, errors.New("frozen intake configuration is invalid")
	}
	canonical, err := filepath.EvalSymlinks(intakeRoot)
	info, statErr := os.Lstat(intakeRoot)
	if err != nil || canonical != intakeRoot || statErr != nil || !privateIntakeDirectory(info) {
		return nil, errors.New("frozen intake root is untrusted")
	}
	root, err := os.OpenRoot(intakeRoot)
	if err != nil {
		return nil, errors.New("frozen intake root is unavailable")
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = root.Close()
		return nil, errors.New("frozen intake root changed")
	}
	view := &FrozenIntake{root: root, rootPath: intakeRoot, rootInfo: info, files: make(map[string]frozenFile, len(entries))}
	fail := func(cause error) (*FrozenIntake, error) { return nil, errors.Join(cause, view.Close()) }
	seenHandles := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		selector, size, err := SelectionForResolved(entry.Resolved)
		declared, hasDeclared := entry.Acquired.DeclaredIntegrity()
		if err != nil || entry.Reference.Source().String() != SourceName || entry.Reference.Locator() != selector.Locator() || entry.Acquired.Identity() != entry.Resolved.Identity() || entry.Acquired.SizeBytes() != size || !hasDeclared || declared != entry.Resolved.DeclaredIntegrity() || declared != "sha256:"+entry.Acquired.Digest().String() || seenHandles[entry.Acquired.ContentHandle()] {
			return fail(errors.New("frozen intake identity differs from acquired selection"))
		}
		if _, exists := view.files[entry.Reference.Locator()]; exists {
			return fail(errors.New("frozen intake reference is duplicated"))
		}
		file, err := openFrozenArtifact(root, entry.Acquired)
		if err != nil {
			return fail(err)
		}
		view.files[entry.Reference.Locator()] = frozenFile{entry: entry, file: file}
		seenHandles[entry.Acquired.ContentHandle()] = true
		hash := sha256.New()
		count, err := io.Copy(hash, frozenContextReader{ctx: ctx, reader: io.NewSectionReader(file, 0, int64(size)+1)})
		if err != nil || uint64(count) != size || hex.EncodeToString(hash.Sum(nil)) != entry.Acquired.Digest().String() {
			return fail(errors.Join(errors.New("frozen intake content differs from acquired digest"), ctx.Err()))
		}
	}
	return view, nil
}

func privateIntakeDirectory(info os.FileInfo) bool {
	if info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 {
		return false
	}
	identity, ok := info.Sys().(*syscall.Stat_t)
	return ok && identity.Uid == uint32(os.Geteuid())
}

func openFrozenArtifact(root *os.Root, artifact domain.AcquiredArtifact) (result *os.File, resultErr error) {
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != SourceName {
		return nil, errors.New("frozen intake handle is invalid")
	}
	run, err := domain.ParseRunID(parts[1])
	if err != nil {
		return nil, errors.New("frozen intake Run is invalid")
	}
	directoryInfo, err := root.Lstat(run.String())
	if err != nil || !privateIntakeDirectory(directoryInfo) {
		return nil, errors.New("frozen intake Run directory is untrusted")
	}
	directory, err := root.OpenRoot(run.String())
	if err != nil {
		return nil, errors.New("frozen intake Run directory is unavailable")
	}
	defer func() {
		if directory.Close() != nil {
			if result != nil {
				if result.Close() != nil {
					resultErr = errors.Join(resultErr, errors.New("close frozen intake file failed"))
				}
				result = nil
			}
			resultErr = errors.Join(resultErr, errors.New("close frozen intake Run directory failed"))
		}
	}()
	opened, err := directory.Stat(".")
	if err != nil || !os.SameFile(directoryInfo, opened) {
		return nil, errors.New("frozen intake Run directory changed")
	}
	before, err := directory.Lstat("asset")
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0077 != 0 {
		return nil, errors.New("frozen intake file is untrusted")
	}
	file, err := directory.OpenFile("asset", os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, errors.New("frozen intake file is unavailable")
	}
	info, err := file.Stat()
	if err != nil || !os.SameFile(before, info) || !info.Mode().IsRegular() || uint64(info.Size()) != artifact.SizeBytes() {
		_ = file.Close()
		return nil, errors.New("frozen intake file identity changed")
	}
	identity, ok := info.Sys().(*syscall.Stat_t)
	if !ok || identity.Uid != uint32(os.Geteuid()) || identity.Nlink != 1 {
		_ = file.Close()
		return nil, errors.New("frozen intake file ownership or link count is invalid")
	}
	return file, nil
}

func (v *FrozenIntake) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if v == nil || ctx == nil {
		return domain.ResolvedArtifact{}, errors.New("frozen intake request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.ResolvedArtifact{}, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	entry, found := v.files[reference.Locator()]
	if v.closed || !found || reference != entry.entry.Reference {
		return domain.ResolvedArtifact{}, errors.New("reference is absent from frozen intake")
	}
	return entry.entry.Resolved, nil
}

func (v *FrozenIntake) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if v == nil || ctx == nil || run.String() == "" {
		return domain.AcquiredArtifact{}, errors.New("frozen intake acquisition is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	selector, _, err := SelectionForResolved(resolved)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	entry, found := v.files[selector.Locator()]
	if v.closed || !found || resolved != entry.entry.Resolved {
		return domain.AcquiredArtifact{}, errors.New("selection is absent from frozen intake")
	}
	current, err := os.Lstat(v.rootPath)
	if err != nil || !privateIntakeDirectory(current) || !os.SameFile(v.rootInfo, current) {
		return domain.AcquiredArtifact{}, errors.New("frozen intake root changed")
	}
	if err := v.root.Mkdir(run.String(), 0700); err != nil {
		return domain.AcquiredArtifact{}, errors.New("frozen intake destination already exists or is unavailable")
	}
	cleanup := func(cause error) (domain.AcquiredArtifact, error) {
		if err := v.root.RemoveAll(run.String()); err != nil {
			return domain.AcquiredArtifact{}, errors.Join(cause, errors.New("frozen intake partial copy cleanup failed"))
		}
		return domain.AcquiredArtifact{}, cause
	}
	output, err := v.root.OpenFile(run.String()+"/asset", os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return cleanup(errors.New("frozen intake destination file is unavailable"))
	}
	expected := entry.entry.Acquired
	hash := sha256.New()
	reader := ports.DownloadReader(ctx, frozenContextReader{ctx: ctx, reader: io.NewSectionReader(entry.file, 0, int64(expected.SizeBytes())+1)}, int64(expected.SizeBytes()))
	count, copyErr := io.Copy(io.MultiWriter(output, hash), reader)
	syncErr, closeErr := output.Sync(), output.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil || uint64(count) != expected.SizeBytes() || hex.EncodeToString(hash.Sum(nil)) != expected.Digest().String() {
		return cleanup(errors.Join(errors.New("frozen intake copied bytes differ from inspected selection"), ctx.Err()))
	}
	if err := ctx.Err(); err != nil {
		return cleanup(err)
	}
	current, err = os.Lstat(v.rootPath)
	if err != nil || !privateIntakeDirectory(current) || !os.SameFile(v.rootInfo, current) {
		return cleanup(errors.New("frozen intake root changed during copy"))
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), expected.Digest(), "intake:"+run.String()+":"+SourceName, expected.SizeBytes(), resolved.DeclaredIntegrity())
	if err != nil {
		return cleanup(err)
	}
	return artifact, nil
}

// Close owns only held descriptors, never the caller's source intake directory.
func (v *FrozenIntake) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return v.closeErr
	}
	v.closed = true
	for _, entry := range v.files {
		if entry.file.Close() != nil {
			v.closeErr = errors.Join(v.closeErr, errors.New("close frozen intake source failed"))
		}
	}
	if v.root != nil && v.root.Close() != nil {
		v.closeErr = errors.Join(v.closeErr, errors.New("close frozen intake root failed"))
	}
	return v.closeErr
}

type frozenContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r frozenContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
