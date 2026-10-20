//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// Recover uses only the fixed root journal and independently signed retained
// versions. It never accepts a caller-selected snapshot, UID, target or PID.
func Recover(ctx context.Context) (resultErr error) {
	if ctx == nil || runtime.GOARCH != "amd64" || os.Geteuid() != 0 || os.Getegid() != 0 {
		return ErrCaller
	}
	owner, err := acquireInstallerOwner(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, owner.Close()) }()
	journal, err := readDurableJournal(ctx)
	if err != nil {
		return err
	}
	old, err := readRecordedRelease(ctx, journal.value.Old)
	if err != nil {
		return err
	}
	var candidate *InstalledRelease
	if next, err := readRecordedRelease(ctx, candidateRegistration(journal.value)); err == nil {
		candidate = &next
	}
	if err := verifyInstallerController(old); err != nil {
		if candidate == nil || verifyInstallerController(*candidate) != nil {
			return ErrCaller
		}
	}
	wait, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	exclusive, err := acquireExclusiveRuntime(wait)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, exclusive.Close()) }()
	if err := restorableRuntimeConfigs(ctx, journal.value); err != nil {
		return err
	}
	services, err := newServiceController(ctx)
	if err != nil {
		return err
	}
	activation := &rootActivation{old: old, journal: journal, services: services, checks: activationChecks{ReadInstalledRelease, readRecordedRelease, runCompatibilityHealth}}
	defer func() { resultErr = errors.Join(resultErr, activation.Close()) }()
	activation.restore, err = releaseinstall.NewSystemInstaller(protectedBundleVerifier{old, old.proof})
	if err != nil {
		return err
	}
	activation.stage, err = recoverPreparedStage(ctx, journal.value, old, candidate)
	if err != nil {
		return err
	}
	activation.workspace, err = recoverHealthWorkspace(journal.value.Stage, old)
	if err != nil {
		return err
	}
	if journal.value.State == "COMMITTED" || journal.value.State == "ROLLED_BACK" {
		expected := old.record
		oldConfig := true
		if journal.value.State == "COMMITTED" {
			expected = candidateRegistration(journal.value)
			oldConfig = false
		}
		installed, err := ReadInstalledRelease(ctx)
		if err != nil || installed.record != expected {
			return errors.Join(ErrUntrusted, err)
		}
		if err := verifyRuntimeConfigs(ctx, journal.value, oldConfig); err != nil {
			return err
		}
		process, err := recoverTerminalService(ctx, services, installed)
		if err != nil {
			return err
		}
		defer func() { resultErr = errors.Join(resultErr, process.Close()) }()
		if err := services.runtimeHealthy(ctx); err != nil {
			return err
		}
		if err := services.resourcesDrained(ctx, old.ClientUID(), 0, process.pid); err != nil {
			return err
		}
		if err := activation.cleanupOwnedStage(ctx); err != nil {
			return err
		}
		return activation.retire(ctx)
	}
	if err := activation.rollback(ctx); err != nil {
		return errors.Join(err, activation.requireRecovery(ctx))
	}
	return nil
}

func recoverPreparedStage(ctx context.Context, journal updateJournal, old InstalledRelease, candidate *InstalledRelease) (*preparedRelease, error) {
	if err := journal.validate(); err != nil {
		return nil, err
	}
	directory := filepath.Join(releaseinstall.SystemStagingRoot, journal.Stage)
	identity, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !identity.IsDir() || identity.Mode().Perm() != 0700 {
		return nil, ErrCleanup
	}
	if err := protectedDirectory(directory); err != nil {
		return nil, err
	}
	var body []byte
	var proofs [][]byte
	partial := candidate != nil
	if candidate != nil {
		body, err = readProtected(ctx, filepath.Join(filepath.Dir(candidate.ClientPath()), "helox-release-manifest.json"), 1<<20)
		proofs = [][]byte{candidate.proof}
	} else {
		body, err = readProtected(ctx, filepath.Join(directory, "helox-release-manifest.json"), 1<<20)
		if err == nil {
			proofs, err = readStoredProofs(ctx, journal.ManifestSHA256)
		}
	}
	if err != nil {
		return nil, errors.Join(ErrCleanup, err)
	}
	release, err := newWireRelease(body, proofs)
	if err != nil {
		return nil, err
	}
	if release.manifest.SHA256() != journal.ManifestSHA256 || release.manifest.Version() != journal.Version {
		return nil, ErrUntrusted
	}
	proof, err := release.verify(ctx, old)
	if err != nil {
		return nil, err
	}
	expected := map[string]releaseinstall.ReleaseFile{"helox-release-manifest.json": {Name: "helox-release-manifest.json", SHA256: journal.ManifestSHA256, Size: int64(len(body))}}
	for _, file := range release.files {
		expected[file.Name] = file
	}
	entries, err := os.ReadDir(directory)
	if err != nil || !partial && len(entries) != len(expected) {
		return nil, ErrCleanup
	}
	for _, entry := range entries {
		file, found := expected[entry.Name()]
		if !found || entry.IsDir() {
			return nil, ErrCleanup
		}
		if err := hashProtected(ctx, filepath.Join(directory, file.Name), file.SHA256, file.Size, 0600); err != nil {
			return nil, errors.Join(ErrCleanup, err)
		}
	}
	return &preparedRelease{directory: directory, identity: identity, release: release, proof: proof}, nil
}

func recoverHealthWorkspace(stage string, installed InstalledRelease) (*healthWorkspace, error) {
	if os.Geteuid() != 0 {
		return nil, ErrCaller
	}
	path, err := healthWorkspacePath(stage)
	if err != nil {
		return nil, err
	}
	if err := protectedDirectory(stateRoot); err != nil {
		return nil, err
	}
	identity, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !identity.IsDir() || identity.Mode().Perm() != 0700 || identity.Mode()&os.ModeSymlink != 0 {
		return nil, ErrCleanup
	}
	owner, ok := identity.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, ErrCleanup
	}
	if owner.Uid == 0 {
		entries, err := os.ReadDir(path)
		if err != nil || len(entries) != 0 || owner.Gid != 0 {
			return nil, ErrCleanup
		}
	} else if owner.Uid != installed.ClientUID() || owner.Gid != installed.ClientGID() {
		return nil, ErrCleanup
	}
	return &healthWorkspace{path: path, identity: identity, uid: owner.Uid}, nil
}

type terminalRecoveryServices interface {
	active(context.Context, string, uint32) (*serviceProcess, error)
	stopped(context.Context, *serviceProcess) error
	resourcesDrained(context.Context, uint32, int, int) error
	start(context.Context, string, uint32) (*serviceProcess, error)
}

// A durable terminal state can survive a boot where the known native unit has
// not started. Only independent stopped and global-drain facts permit restart.
func recoverTerminalService(ctx context.Context, services terminalRecoveryServices, installed InstalledRelease) (*serviceProcess, error) {
	if ctx == nil || services == nil {
		return nil, ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	process, activeErr := services.active(ctx, installed.Version(), installed.ClientGID())
	if activeErr == nil {
		if process == nil {
			return nil, ErrUntrusted
		}
		return process, nil
	}
	if process != nil {
		activeErr = errors.Join(activeErr, process.Close())
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(activeErr, err)
	}
	if err := services.stopped(ctx, nil); err != nil {
		return nil, errors.Join(activeErr, err)
	}
	if err := services.resourcesDrained(ctx, installed.ClientUID(), 0, 0); err != nil {
		return nil, errors.Join(activeErr, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	process, err := services.start(ctx, installed.Version(), installed.ClientGID())
	if err != nil {
		if process != nil {
			err = errors.Join(err, process.Close())
		}
		return nil, err
	}
	if process == nil {
		return nil, ErrUntrusted
	}
	return process, nil
}
