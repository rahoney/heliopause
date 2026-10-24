//go:build linux || darwin

package releaseinstall

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// DiscardSystemCopy removes only a fully checked interrupted copy. Its caller
// must own the durable manifest reservation, installer owner and runtime EX,
// and independently confirm resource drainage. Naming alone grants no authority.
func (i *Installer) DiscardSystemCopy(ctx context.Context, request Request) (resultErr error) {
	if i == nil || !i.system || ctx == nil || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || os.Geteuid() != 0 {
		return errors.New("system copy cleanup requires protected installer")
	}
	existing := filepath.Dir(request.BundleDirectory) == filepath.Join(SystemInstallRoot, "versions")
	input, err := i.validatedInput(ctx, request, existing)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(input.body)
	name, err := SystemCopyStageName(hex.EncodeToString(sum[:]))
	if err != nil {
		return err
	}
	directory := filepath.Join(SystemInstallRoot, "versions", name)
	if err := verifySystemDirectory(filepath.Dir(directory), false); err != nil {
		return err
	}
	before, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return errors.New("untrusted copy stage")
	}
	owner, ok := before.Sys().(*syscall.Stat_t)
	mode := before.Mode().Perm()
	if !ok || owner.Uid != 0 || owner.Gid != 0 || (mode&^0700 != 0 && mode != 0755) || before.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return errors.New("untrusted copy stage owner or mode")
	}
	record, err := installationRecord(input.manifest.Version, input.body)
	if err != nil {
		return err
	}
	type expectedCopy struct {
		size int64
		mode os.FileMode
	}
	expected := map[string]expectedCopy{
		manifestName:      {int64(len(input.body)), 0444},
		sbomName:          {input.manifest.SBOM.Size, 0444},
		runtimeImagesName: {input.manifest.RuntimeImages.Size, 0444},
		installRecordName: {int64(len(record)), 0444},
	}
	for _, file := range input.files {
		expected[file.Name] = expectedCopy{file.Size, 0555}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) > len(expected) {
		return errors.New("unknown copy stage membership")
	}
	identities := make(map[string]os.FileInfo, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		value, found := expected[entry.Name()]
		if !found || entry.IsDir() {
			return errors.New("unknown copy stage member")
		}
		path := filepath.Join(directory, entry.Name())
		if err := verifySystemRegularFile(path); err != nil {
			return err
		}
		identity, err := os.Lstat(path)
		if err != nil {
			return err
		}
		owner, ok := identity.Sys().(*syscall.Stat_t)
		// Open(mode) is affected by umask until the copier's explicit Chmod. Only
		// subsets of the intended read/execute permissions can be interrupted here.
		if !ok || owner.Gid != 0 || identity.Mode().Perm()&^value.mode != 0 || identity.Size() < 0 || identity.Size() > value.size {
			return errors.New("untrusted partial copy metadata")
		}
		file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		opened, statErr := file.Stat()
		if statErr != nil || !os.SameFile(identity, opened) {
			_ = file.Close()
			return errors.New("copy member replaced")
		}
		var compareErr error
		if entry.Name() == installRecordName {
			compareErr = compareCopyPrefix(ctx, file, bytes.NewReader(record), identity.Size())
		} else {
			sourcePath := filepath.Join(input.bundle, entry.Name())
			if err := verifySystemRegularFile(sourcePath); err != nil {
				_ = file.Close()
				return err
			}
			source, err := os.OpenFile(sourcePath, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
			if err != nil {
				_ = file.Close()
				return err
			}
			compareErr = compareCopyPrefix(ctx, file, source, identity.Size())
			compareErr = errors.Join(compareErr, source.Close())
		}
		closeErr := file.Close()
		after, statErr := os.Lstat(path)
		if compareErr != nil || closeErr != nil || statErr != nil || !samePendingFile(identity, after) {
			return errors.Join(errors.New("partial copy differs from verified source"), compareErr, closeErr, statErr)
		}
		identities[entry.Name()] = identity
	}
	// Validate the complete set before deleting the first member. After a crash
	// during cleanup, the same journal can safely validate the remaining subset.
	current, err := os.Lstat(directory)
	if err != nil || !os.SameFile(before, current) {
		return errors.New("copy stage replaced")
	}
	actual, err := os.ReadDir(directory)
	if err != nil || len(actual) != len(identities) {
		return errors.New("copy membership changed")
	}
	for _, entry := range actual {
		if identities[entry.Name()] == nil {
			return errors.New("copy membership changed")
		}
	}
	for _, entry := range actual {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(directory, entry.Name())
		identity := identities[entry.Name()]
		current, err := os.Lstat(path)
		if err != nil || !samePendingFile(identity, current) {
			return errors.New("copy member changed before removal")
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	if err := syncDirectory(directory); err != nil {
		return err
	}
	current, err = os.Lstat(directory)
	if err != nil || !os.SameFile(before, current) {
		return errors.New("copy stage replaced before removal")
	}
	if err := os.Remove(directory); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(directory))
}

func compareCopyPrefix(ctx context.Context, partial, source io.Reader, size int64) error {
	var actual, expected [64 << 10]byte
	for size > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		count := int64(len(actual))
		if size < count {
			count = size
		}
		if _, err := io.ReadFull(partial, actual[:count]); err != nil {
			return err
		}
		if _, err := io.ReadFull(source, expected[:count]); err != nil {
			return err
		}
		if !bytes.Equal(actual[:count], expected[:count]) {
			return errors.New("partial copy prefix mismatch")
		}
		size -= count
	}
	return ctx.Err()
}

func samePendingFile(before, after os.FileInfo) bool {
	owner, ok := after.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == 0 && owner.Gid == 0 && owner.Nlink == 1 && os.SameFile(before, after) && before.Size() == after.Size() && before.ModTime() == after.ModTime() && before.Mode() == after.Mode()
}
