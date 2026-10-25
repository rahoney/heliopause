package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var errInstalledNativeTool = errors.New("immutable installed native tool supply is unavailable or changed")

// readInstalledNativeTool accepts only compiled installed-tool identities. It
// reads data through a held root; it never executes the native file.
func readInstalledNativeTool(ctx context.Context, installation string, kind nativeRuntimeToolKind) (prepared []byte, resultErr error) {
	spec, known := nativeRuntimeToolIdentity(kind)
	if !known || spec.installedName == "" {
		return nil, errInstalledNativeTool
	}
	if ctx == nil {
		return nil, errInstalledNativeTool
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errInstalledNativeTool, err)
	}
	canonical, err := filepath.EvalSymlinks(installation)
	if err != nil || canonical != installation || !filepath.IsAbs(installation) {
		return nil, errInstalledNativeTool
	}
	beforeRoot, err := os.Lstat(installation)
	if err != nil || !nativeRootDirectory(beforeRoot) {
		return nil, errInstalledNativeTool
	}
	root, err := os.OpenRoot(installation)
	if err != nil {
		return nil, errInstalledNativeTool
	}
	defer func() {
		if err := root.Close(); err != nil {
			prepared = nil
			resultErr = errors.Join(resultErr, errInstalledNativeTool, err)
		}
	}()
	openedRoot, err := root.Stat(".")
	if err != nil || !os.SameFile(beforeRoot, openedRoot) || !nativeRootDirectory(openedRoot) {
		return nil, errInstalledNativeTool
	}
	current, err := root.Lstat("current")
	if err != nil || !nativeRootCurrent(current) {
		return nil, errInstalledNativeTool
	}
	target, err := root.Readlink("current")
	if err != nil || !nativeCurrentTarget(target) {
		return nil, errInstalledNativeTool
	}
	for _, path := range []string{"versions", target} {
		info, err := root.Lstat(path)
		if err != nil || !nativeRootDirectory(info) {
			return nil, errInstalledNativeTool
		}
	}
	name := target + "/" + spec.installedName
	beforeFile, err := root.Lstat(name)
	if err != nil || !nativePinnedFile(beforeFile, spec.size) {
		return nil, errInstalledNativeTool
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, errInstalledNativeTool
	}
	defer func() {
		if err := file.Close(); err != nil {
			prepared = nil
			resultErr = errors.Join(resultErr, errInstalledNativeTool, err)
		}
	}()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(beforeFile, opened) || !nativePinnedFile(opened, spec.size) {
		return nil, errInstalledNativeTool
	}
	body, err := io.ReadAll(io.LimitReader(file, spec.size+1))
	if err != nil || ctx.Err() != nil || int64(len(body)) != spec.size {
		return nil, errors.Join(errInstalledNativeTool, err, ctx.Err())
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != spec.sha256 {
		return nil, errInstalledNativeTool
	}
	afterFile, fileErr := root.Lstat(name)
	afterCurrent, currentErr := root.Lstat("current")
	afterTarget, targetErr := root.Readlink("current")
	afterRoot, rootErr := os.Lstat(installation)
	if fileErr != nil || currentErr != nil || targetErr != nil || rootErr != nil ||
		!os.SameFile(beforeFile, afterFile) || !nativePinnedFile(afterFile, spec.size) ||
		!os.SameFile(current, afterCurrent) || !nativeRootCurrent(afterCurrent) || afterTarget != target ||
		!os.SameFile(beforeRoot, afterRoot) || !nativeRootDirectory(afterRoot) {
		return nil, errInstalledNativeTool
	}
	return body, nil
}

func nativeCurrentTarget(target string) bool {
	if !strings.HasPrefix(target, "versions/v") || filepath.ToSlash(filepath.Clean(target)) != target {
		return false
	}
	version := strings.TrimPrefix(target, "versions/")
	if len(version) < 2 || len(version) > 128 || strings.Contains(version, "/") {
		return false
	}
	for _, value := range version {
		if value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '.' || value == '-' || value == '+' {
			continue
		}
		return false
	}
	return true
}

func nativeRootOwned(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == 0 && owner.Gid == 0
}

func nativeRootDirectory(info os.FileInfo) bool {
	return nativeRootOwned(info) && info.IsDir() && info.Mode().Perm() == 0755 && info.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0
}

func nativeRootCurrent(info os.FileInfo) bool {
	if !nativeRootOwned(info) || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	return info.Sys().(*syscall.Stat_t).Nlink == 1
}

func nativePinnedFile(info os.FileInfo, size int64) bool {
	return nativeRootOwned(info) && info.Mode().IsRegular() && info.Mode().Perm() == 0555 &&
		info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0 && info.Size() == size &&
		info.Sys().(*syscall.Stat_t).Nlink == 1
}
