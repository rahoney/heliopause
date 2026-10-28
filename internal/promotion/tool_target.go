package promotion

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

const toolBinTransactionLock = ".heliopause-tool-bin-transaction.lock"
const uvToolHostReceipt = "uv-receipt.toml"

// Tool targets retain both roots before native selection. Original targets use
// a separate managed-state proof; neither path executes the Host interpreter.
type toolFreshTarget struct {
	client      toolClient
	work        *toolPythonWorkspace
	original    *toolOriginalState
	tx          *pypiVenvTransaction
	bin         goProjectGuard
	binRoot     string
	requirement artifactpypi.UpgradeRequirement
	interpreter []toolInterpreterMember
	commands    []artifactpypi.ToolCommand
	closed      bool
	closeErr    error
}

type toolInterpreterMember struct {
	path string
	info os.FileInfo
	link string
	file *pypiFileSnapshot
}

func toolOwnedDirectory(path string) error {
	if trustedExistingDirectory(path) != nil {
		return errors.New("tool root is not a canonical existing directory")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return errors.New("tool root permissions are unsafe")
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return errors.New("tool root owner is unavailable")
	}
	uid := value.Elem().FieldByName("Uid")
	if !uid.IsValid() || !uid.CanUint() || uid.Uint() != uint64(os.Geteuid()) {
		return errors.New("tool root is not owned by current user")
	}
	return nil
}

func snapshotToolInterpreter(path string) ([]toolInterpreterMember, error) {
	seen := map[string]bool{}
	var members []toolInterpreterMember
	for len(members) < 12 {
		if seen[path] || !filepath.IsAbs(path) || filepath.Clean(path) != path || rejectSymlinkPath(filepath.Dir(path)) != nil {
			return nil, errors.New("tool interpreter alias chain is unsafe")
		}
		seen[path] = true
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		member := toolInterpreterMember{path: path, info: info}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil || link == "" || strings.ContainsAny(link, "\\\x00\r\n") {
				return nil, errors.New("tool interpreter alias is unsupported")
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(info, after) || info.Mode() != after.Mode() {
				return nil, errors.New("tool interpreter alias changed")
			}
			member.link = link
			members = append(members, member)
			if filepath.IsAbs(link) {
				path = filepath.Clean(link)
			} else {
				path = filepath.Clean(filepath.Join(filepath.Dir(path), link))
			}
			continue
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return nil, errors.New("tool interpreter is not an executable regular file")
		}
		snapshot, err := snapshotBoundedPromotionFile(path, 64<<20)
		if err != nil || !os.SameFile(info, snapshot.info) {
			return nil, errors.New("tool interpreter file changed or exceeds bounds")
		}
		member.file = &snapshot
		members = append(members, member)
		return members, nil
	}
	return nil, errors.New("tool interpreter alias chain exceeds bounds")
}

func verifyToolInterpreterSnapshot(members []toolInterpreterMember) error {
	for _, member := range members {
		info, err := os.Lstat(member.path)
		if err != nil || !os.SameFile(member.info, info) || member.info.Mode() != info.Mode() {
			return errors.New("tool interpreter identity changed")
		}
		if member.file != nil {
			if !member.file.matches(member.path) {
				return errors.New("tool interpreter bytes changed")
			}
		} else {
			link, err := os.Readlink(member.path)
			if err != nil || link != member.link {
				return errors.New("tool interpreter alias changed")
			}
		}
	}
	return nil
}

func beginToolFreshTarget(ctx context.Context, install domain.InstallContext, requirement artifactpypi.UpgradeRequirement, binRoot string, trustedRoots []string) (result *toolFreshTarget, resultErr error) {
	return beginToolTarget(ctx, install, requirement, binRoot, trustedRoots, false)
}

func beginToolTarget(ctx context.Context, install domain.InstallContext, requirement artifactpypi.UpgradeRequirement, binRoot string, trustedRoots []string, original bool) (result *toolFreshTarget, resultErr error) {
	return beginToolTargetWithNative(ctx, install, requirement, binRoot, trustedRoots, original, nil)
}

func beginToolTargetWithNative(ctx context.Context, install domain.InstallContext, requirement artifactpypi.UpgradeRequirement, binRoot string, trustedRoots []string, original bool, native *pypiNativeEnrollment) (result *toolFreshTarget, resultErr error) {
	return beginToolTargetForClient(ctx, install, requirement, binRoot, trustedRoots, original, native, toolClientUV)
}

func beginToolTargetForClient(ctx context.Context, install domain.InstallContext, requirement artifactpypi.UpgradeRequirement, binRoot string, trustedRoots []string, original bool, native *pypiNativeEnrollment, client toolClient) (result *toolFreshTarget, resultErr error) {
	if !client.valid() || client == toolClientPipx && (original || native != nil) {
		return nil, errors.New("tool client or original mode is unavailable")
	}
	if ctx == nil || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || !install.Valid() || install.Mode() != domain.InstallPythonVenv || requirement.Project() == "" || requirement.Reference().Source() != artifactpypi.PublicPyPIProfile().Source() || len(trustedRoots) == 0 {
		return nil, errors.New("tool target requires explicit Linux amd64 venv, requirement and trusted storage")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := install.Target().String()
	if filepath.Base(root) != requirement.Project() {
		return nil, errors.New("tool target basename differs from normalized package")
	}
	roots := append([]string{root, binRoot}, trustedRoots...)
	for _, path := range roots {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || strings.ContainsAny(path, ",:\\\x00\r\n") {
			return nil, errors.New("tool target boundary is unsupported")
		}
	}
	if !separateRoots(roots) {
		return nil, errors.New("tool venv, exposed bin and storage overlap")
	}
	if err := toolOwnedDirectory(root); err != nil {
		return nil, err
	}
	if err := toolOwnedDirectory(binRoot); err != nil {
		return nil, err
	}
	plan, err := discoverUVPythonVenv(root)
	if err != nil {
		return nil, err
	}
	body, err := readBoundedPromotionFile(filepath.Join(root, "pyvenv.cfg"), 64<<10)
	if err != nil {
		return nil, err
	}
	version, err := uvPythonVenvConfigVersion(body)
	if err != nil || version != sandbox.PinnedPythonRuntime().PythonVersion {
		return nil, errors.New("tool target requires the pinned Python version")
	}
	tx, err := beginPyPIVenvTransactionWithDiscovery(plan, discoverUVPythonVenv)
	if err != nil {
		return nil, err
	}
	result = &toolFreshTarget{client: client, tx: tx, binRoot: binRoot, requirement: requirement}
	accepted := false
	guarded := result
	defer func() {
		if !accepted {
			resultErr = errors.Join(resultErr, guarded.Close())
			result = nil
		}
	}()
	guard, err := acquireProjectTransactionGuard(binRoot, toolBinTransactionLock)
	if err != nil {
		return nil, err
	}
	result.bin = guard
	if err := tx.freezeParents(binRoot); err != nil {
		return nil, err
	}
	receipt := filepath.Join(root, client.receiptName())
	if original {
		if native != nil && result.tx.metadata == nil {
			result.original, err = captureUVToolNativeOriginal(ctx, result, receipt, install, native)
		} else {
			result.original, err = captureToolOriginal(ctx, result, receipt)
		}
		if err != nil {
			return nil, err
		}
	} else {
		for _, name := range []string{uvToolHostReceipt, pipxToolHostReceipt} {
			path := filepath.Join(root, name)
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return nil, errors.New("fresh tool target already has native receipt or uncertain data")
			}
			tx.before[path] = nil
		}
	}
	result.interpreter, err = snapshotToolInterpreter(filepath.Join(root, "bin", "python"))
	if err != nil {
		return nil, err
	}
	if err := result.verify(ctx); err != nil {
		return nil, err
	}
	accepted = true
	return result, nil
}

func (g *toolFreshTarget) verify(ctx context.Context) error {
	if g == nil || g.closed || ctx == nil || g.tx == nil {
		return errors.New("tool target guard is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := g.bin.verify(); err != nil {
		return err
	}
	if err := toolOwnedDirectory(g.binRoot); err != nil {
		return err
	}
	if err := toolOwnedDirectory(g.tx.plan.root); err != nil {
		return err
	}
	if err := g.tx.checkBefore(); err != nil {
		return err
	}
	if g.original != nil {
		if err := g.original.verify(ctx, g); err != nil {
			return err
		}
	} else {
		site, err := os.Open(g.tx.plan.site)
		if err != nil {
			return err
		}
		entries, readErr := site.ReadDir(1)
		closeErr := site.Close()
		if !errors.Is(readErr, io.EOF) || closeErr != nil || len(entries) != 0 || len(g.tx.current.Files) != 0 {
			return errors.New("fresh tool requires an empty distribution and ownership namespace")
		}
	}
	if err := verifyToolInterpreterSnapshot(g.interpreter); err != nil {
		return err
	}
	if err := g.verifyCommandCaseCollisions(); err != nil {
		return err
	}
	return g.tx.checkBefore()
}

func (g *toolFreshTarget) freezeCommands(ctx context.Context, receipt artifactpypi.ToolReceipt) error {
	if err := g.verify(ctx); err != nil {
		return err
	}
	if g.original != nil {
		return g.freezeOriginalCommands(ctx, receipt)
	}
	if len(g.commands) != 0 || receipt.Client() != g.client.name() || !receipt.Requirement().SameConstraints(g.requirement) {
		return errors.New("tool command selection is repeated or differs from request")
	}
	root := g.tx.plan.root
	if err := receipt.ValidateLayout(root, g.binRoot, filepath.Join(root, "bin", "python")); err != nil {
		return err
	}
	commands := g.client.exposedCommands(receipt, g.binRoot)
	wanted := map[string]bool{}
	for _, command := range commands {
		if command.Name == toolBinTransactionLock {
			return errors.New("tool command collides with controller guard")
		}
		wanted[strings.ToLower(command.Name)] = true
	}
	file, err := os.Open(g.binRoot)
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(100001)
	closeErr := file.Close()
	if errors.Join(readErr, closeErr) != nil || len(entries) > 100000 {
		return errors.New("tool command directory inventory is unavailable or exceeds bounds")
	}
	for _, entry := range entries {
		if wanted[strings.ToLower(entry.Name())] {
			return errors.New("tool command destination is already occupied")
		}
	}
	for _, command := range commands {
		if _, exists := g.tx.before[command.Path]; exists {
			return errors.New("tool command destination was already frozen")
		}
		internal := filepath.Join(root, "bin", command.Name)
		if _, err := os.Lstat(internal); !errors.Is(err, os.ErrNotExist) {
			return errors.New("tool script would replace existing venv data")
		}
		g.tx.before[command.Path] = nil
		g.tx.before[internal] = nil
	}
	g.commands = commands
	return g.verify(ctx)
}

func (g *toolFreshTarget) Close() error {
	if g == nil {
		return errors.New("tool target guard is unavailable")
	}
	if g.closed {
		return g.closeErr
	}
	if g.original != nil && !g.original.committed {
		if err := g.original.verifyLinks(g); err != nil {
			g.tx.recovery = true
			g.closeErr = errors.Join(g.closeErr, err)
		}
	}
	g.closed = true
	// Keep both locks if either root's identity is uncertain. Independent FDs
	// are still closed, and foreign data is never removed to release a lock.
	if g.bin.root != nil {
		if err := g.bin.verify(); err != nil {
			g.tx.recovery = true
			g.closeErr = errors.Join(g.closeErr, err)
		}
	}
	if err := g.verifyReleaseIdentity(); err != nil {
		g.tx.recovery = true
		g.closeErr = errors.Join(g.closeErr, err)
	}
	if g.tx.recovery && g.bin.root != nil {
		g.closeErr = errors.Join(g.closeErr, g.bin.file.Close(), g.bin.root.Close())
	} else if g.bin.root != nil {
		g.closeErr = errors.Join(g.closeErr, g.bin.release())
	}
	if g.closeErr != nil {
		g.tx.recovery = true
	}
	g.closeErr = errors.Join(g.closeErr, g.tx.close())
	return g.closeErr
}

// The common engine has already removed its owned backup after publication or
// successful rollback. Release checks persistent roots, not retired temporary
// directories from that engine's mutation inventory.
func (g *toolFreshTarget) verifyReleaseIdentity() error {
	if err := g.tx.guard.verify(); err != nil {
		return err
	}
	if !g.tx.config.matches(filepath.Join(g.tx.plan.root, "pyvenv.cfg")) {
		return errors.New("tool venv configuration changed before guard release")
	}
	roots := []string{g.tx.plan.root, g.tx.plan.site, filepath.Join(g.tx.plan.root, "bin")}
	if g.bin.root != nil {
		roots = append(roots, g.binRoot)
	}
	for _, path := range roots {
		before, ok := g.tx.directories[path]
		info, err := os.Lstat(path)
		if !ok || err != nil || !os.SameFile(before, info) || before.Mode() != info.Mode() {
			return errors.New("tool persistent root changed before guard release")
		}
	}
	return nil
}
