//go:build linux || darwin

package githubrelease

import (
	"context"
	"errors"
	"io"
	"os"
	"sort"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type replayedReleaseRun struct{ directory, asset os.FileInfo }

func (s *HeliopauseReleaseSet) captureReplayRun(run domain.RunID, artifact domain.AcquiredArtifact) (owned replayedReleaseRun, resultErr error) {
	info, err := s.root.Lstat(run.String())
	if err != nil || !privateIntakeDirectory(info) {
		return owned, errors.New("release replay directory identity unavailable")
	}
	directory, err := s.root.OpenRoot(run.String())
	if err != nil {
		return owned, err
	}
	defer func() { resultErr = errors.Join(resultErr, directory.Close()) }()
	opened, err := directory.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return owned, errors.New("release replay directory changed")
	}
	asset, err := directory.Lstat("asset")
	if err != nil || !ownedReplayAsset(asset) || uint64(asset.Size()) != artifact.SizeBytes() {
		return owned, errors.New("release replay asset identity unavailable")
	}
	if err := onlyReplayAsset(directory); err != nil {
		return owned, err
	}
	return replayedReleaseRun{info, asset}, nil
}

// DiscardReplayRuns is explicit retention after the controller has confirmed
// actual runtime drainage. Close alone keeps inspection Runs unchanged. This
// method removes only the exact copies made by this source, never Evidence.
func (s *HeliopauseReleaseSet) DiscardReplayRuns(ctx context.Context) error {
	if s == nil || ctx == nil {
		return errors.New("release replay retention requires source and context")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.replayMu.Lock()
	defer s.replayMu.Unlock()
	if len(s.replayed) == 0 {
		return ctx.Err()
	}
	if s.closed || s.root == nil {
		return errors.New("release replay retention source is closed")
	}
	current, err := os.Lstat(s.client.intakeRoot)
	if err != nil || !privateIntakeDirectory(current) || !os.SameFile(s.rootInfo, current) {
		return errors.New("release replay retention root changed")
	}
	names := make([]string, 0, len(s.replayed))
	for name := range s.replayed {
		names = append(names, name)
	}
	sort.Strings(names)
	var failures []error
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		if err := discardReplayRun(s.root, name, s.replayed[name]); err != nil {
			failures = append(failures, err)
			continue
		}
		delete(s.replayed, name)
	}
	return errors.Join(failures...)
}

func discardReplayRun(root *os.Root, name string, owned replayedReleaseRun) (resultErr error) {
	before, err := root.Lstat(name)
	if err != nil || !privateIntakeDirectory(before) || !os.SameFile(owned.directory, before) {
		return errors.New("release replay retention directory ownership uncertain")
	}
	directory, err := root.OpenRoot(name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, directory.Close()) }()
	opened, err := directory.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		return errors.New("release replay retention directory changed")
	}
	asset, err := directory.Lstat("asset")
	if err != nil || !ownedReplayAsset(asset) || !sameReplayFile(owned.asset, asset) {
		return errors.New("release replay retention asset ownership uncertain")
	}
	if err := onlyReplayAsset(directory); err != nil {
		return err
	}
	if err := directory.Remove("asset"); err != nil {
		return err
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(before, after) {
		return errors.New("release replay retention parent changed")
	}
	return root.Remove(name)
}

func ownedReplayAsset(info os.FileInfo) bool {
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return false
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == uint32(os.Geteuid()) && owner.Nlink == 1
}

func sameReplayFile(before, after os.FileInfo) bool {
	return os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

func onlyReplayAsset(root *os.Root) (resultErr error) {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, directory.Close()) }()
	entries, err := directory.ReadDir(2)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) != 1 || entries[0].Name() != "asset" {
		return errors.New("release replay contains unknown entries")
	}
	return nil
}
