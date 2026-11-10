//go:build linux

package promotion

import (
	"errors"
	"io"
	"os"
	"sort"
	"syscall"

	"github.com/rahoney/heliopause/internal/releaseactivation"
)

type nativeStagePayload struct {
	path                 string
	directory, artifacts os.FileInfo
	files                map[string]os.FileInfo
}

// Close discards this operation's payload only after the session has confirmed
// actual runtime drainage. Manifest/SBOM records and Evidence remain available.
func (p *NativeReleasePromotion) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return p.closeErr
	}
	p.closed = true
	if p.payload != nil {
		p.closeErr = p.payload.discard()
	}
	return p.closeErr
}

func (p *nativeStagePayload) discard() (resultErr error) {
	root, err := os.OpenRoot(p.path)
	if err != nil {
		return errors.Join(releaseactivation.ErrCleanup, err)
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	info, err := root.Stat(".")
	if err != nil || !os.SameFile(p.directory, info) || nativeStageDirectory(root, ".") != nil {
		return releaseactivation.ErrCleanup
	}
	info, err = root.Lstat("artifacts")
	if err != nil || !os.SameFile(p.artifacts, info) || nativeStageDirectory(root, "artifacts") != nil {
		return releaseactivation.ErrCleanup
	}
	artifacts, err := root.OpenRoot("artifacts")
	if err != nil {
		return errors.Join(releaseactivation.ErrCleanup, err)
	}
	defer func() { resultErr = errors.Join(resultErr, artifacts.Close()) }()
	opened, err := artifacts.Stat(".")
	if err != nil || !os.SameFile(p.artifacts, opened) {
		return releaseactivation.ErrCleanup
	}
	list, err := artifacts.Open(".")
	if err != nil {
		return errors.Join(releaseactivation.ErrCleanup, err)
	}
	entries, readErr := list.ReadDir(len(p.files) + 1)
	closeErr := list.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) != len(p.files) {
		return errors.Join(releaseactivation.ErrCleanup, closeErr)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		owned, ok := p.files[entry.Name()]
		if !ok {
			return releaseactivation.ErrCleanup
		}
		actual, err := artifacts.Lstat(entry.Name())
		if err != nil || !sameNativeStageFile(owned, actual) {
			return releaseactivation.ErrCleanup
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		actual, err := artifacts.Lstat(name)
		if err != nil || !sameNativeStageFile(p.files[name], actual) {
			return releaseactivation.ErrCleanup
		}
		if err := artifacts.Remove(name); err != nil {
			return errors.Join(releaseactivation.ErrCleanup, err)
		}
	}
	current, err := root.Lstat("artifacts")
	if err != nil || !os.SameFile(p.artifacts, current) {
		return releaseactivation.ErrCleanup
	}
	if err := root.Remove("artifacts"); err != nil {
		return errors.Join(releaseactivation.ErrCleanup, err)
	}
	return nil
}

func sameNativeStageFile(before, after os.FileInfo) bool {
	owner, ok := after.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == uint32(os.Geteuid()) && owner.Nlink == 1 && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}
