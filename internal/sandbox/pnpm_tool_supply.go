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

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const pnpmToolGuestPath = "/haa-runtime/pnpm"

var errPNPMToolSupply = errors.New("pnpm immutable native tool supply is unavailable or changed")

type pnpmPreparedTool struct{ body []byte }

// preparePNPMTool reads the fixed protected installation as data. The pin was
// adopted through the upstream provenance/archive verifier and signed release
// producer. This reader neither executes a Host tool nor accepts a user path.
func preparePNPMTool(ctx context.Context) (*pnpmPreparedTool, error) {
	return readInstalledPNPMTool(ctx, runtimeidentity.NativeInstallRoot)
}

func readInstalledPNPMTool(ctx context.Context, installation string) (prepared *pnpmPreparedTool, resultErr error) {
	if ctx == nil {
		return nil, errPNPMToolSupply
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errPNPMToolSupply, err)
	}
	canonical, err := filepath.EvalSymlinks(installation)
	if err != nil || canonical != installation || !filepath.IsAbs(installation) {
		return nil, errPNPMToolSupply
	}
	beforeRoot, err := os.Lstat(installation)
	if err != nil || !pnpmRootDirectory(beforeRoot) {
		return nil, errPNPMToolSupply
	}
	root, err := os.OpenRoot(installation)
	if err != nil {
		return nil, errPNPMToolSupply
	}
	defer func() {
		if err := root.Close(); err != nil {
			prepared = nil
			resultErr = errors.Join(resultErr, errPNPMToolSupply, err)
		}
	}()
	openedRoot, err := root.Stat(".")
	if err != nil || !os.SameFile(beforeRoot, openedRoot) || !pnpmRootDirectory(openedRoot) {
		return nil, errPNPMToolSupply
	}
	current, err := root.Lstat("current")
	if err != nil || !pnpmRootCurrent(current) {
		return nil, errPNPMToolSupply
	}
	target, err := root.Readlink("current")
	if err != nil || !pnpmCurrentTarget(target) {
		return nil, errPNPMToolSupply
	}
	for _, path := range []string{"versions", target} {
		info, err := root.Lstat(path)
		if err != nil || !pnpmRootDirectory(info) {
			return nil, errPNPMToolSupply
		}
	}
	name := target + "/" + runtimeidentity.PNPMNativeAssetName
	beforeFile, err := root.Lstat(name)
	if err != nil || !pnpmPinnedFile(beforeFile) {
		return nil, errPNPMToolSupply
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, errPNPMToolSupply
	}
	defer func() {
		if err := file.Close(); err != nil {
			prepared = nil
			resultErr = errors.Join(resultErr, errPNPMToolSupply, err)
		}
	}()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(beforeFile, opened) || !pnpmPinnedFile(opened) {
		return nil, errPNPMToolSupply
	}
	body, err := io.ReadAll(io.LimitReader(file, runtimeidentity.PNPMToolSize+1))
	if err != nil || ctx.Err() != nil || int64(len(body)) != runtimeidentity.PNPMToolSize {
		return nil, errors.Join(errPNPMToolSupply, err, ctx.Err())
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != runtimeidentity.PNPMToolSHA256 {
		return nil, errPNPMToolSupply
	}
	afterFile, fileErr := root.Lstat(name)
	afterCurrent, currentErr := root.Lstat("current")
	afterTarget, targetErr := root.Readlink("current")
	afterRoot, rootErr := os.Lstat(installation)
	if fileErr != nil || currentErr != nil || targetErr != nil || rootErr != nil ||
		!os.SameFile(beforeFile, afterFile) || !pnpmPinnedFile(afterFile) ||
		!os.SameFile(current, afterCurrent) || !pnpmRootCurrent(afterCurrent) || afterTarget != target ||
		!os.SameFile(beforeRoot, afterRoot) || !pnpmRootDirectory(afterRoot) {
		return nil, errPNPMToolSupply
	}
	return &pnpmPreparedTool{body: body}, nil
}

func pnpmCurrentTarget(target string) bool {
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

func pnpmRootOwned(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == 0 && owner.Gid == 0
}

func pnpmRootDirectory(info os.FileInfo) bool {
	return pnpmRootOwned(info) && info.IsDir() && info.Mode().Perm() == 0755 && info.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0
}

func pnpmRootCurrent(info os.FileInfo) bool {
	if !pnpmRootOwned(info) || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	return info.Sys().(*syscall.Stat_t).Nlink == 1
}

func pnpmPinnedFile(info os.FileInfo) bool {
	return pnpmRootOwned(info) && info.Mode().IsRegular() && info.Mode().Perm() == 0555 &&
		info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0 && info.Size() == runtimeidentity.PNPMToolSize &&
		info.Sys().(*syscall.Stat_t).Nlink == 1
}

func (p *pnpmPreparedTool) introduce(ctx context.Context, runner CommandRunner, container string) error {
	if p == nil {
		return errPNPMToolSupply
	}
	if err := introduceNativeTool(ctx, runner, container, nativeRuntimePNPM, p.body); err != nil {
		return errors.Join(errPNPMToolSupply, err)
	}
	return nil
}
