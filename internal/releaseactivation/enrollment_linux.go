//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// Register records an already independently verified native bootstrap. It does
// not download, copy, replace current, start a service, or adopt another layout.
// A bootstrap must authenticate this executable before invoking it as root.
func Register(ctx context.Context) error {
	return registerNative(ctx, readRecordedRelease, verifyInstallerController, func(ctx context.Context) (enrollmentServices, error) { return newServiceController(ctx) })
}

type enrollmentServices interface {
	active(context.Context, string, uint32) (*serviceProcess, error)
	resourcesDrained(context.Context, uint32, int, int) error
	runtimeHealthy(context.Context) error
	Close() error
}

func registerNative(ctx context.Context, recorded func(context.Context, registration) (InstalledRelease, error), controller func(InstalledRelease) error, services func(context.Context) (enrollmentServices, error)) (resultErr error) {
	if ctx == nil || runtime.GOARCH != "amd64" || os.Geteuid() != 0 || os.Getegid() != 0 {
		return ErrCaller
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	owner, err := acquireInstallerOwner(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, owner.Close()) }()
	record, configs, err := enrollmentIdentity(ctx)
	if err != nil {
		return err
	}
	previous, err := enrollmentState(ctx, record)
	if err != nil {
		return err
	}
	installed, err := recorded(ctx, record)
	if err != nil {
		return err
	}
	if installed.record != record {
		return ErrUntrusted
	}
	if err := controller(installed); err != nil {
		return err
	}
	if err := publicNativeLayout(ctx, installed); err != nil {
		return err
	}
	for _, path := range []string{runtimeLockPath, updateLockPath} {
		check := validateEnrollmentLock
		if len(previous) == 0 {
			check = prepareEnrollmentLock
		}
		if err := check(path); err != nil {
			return err
		}
	}
	update, err := lockPersistentFile(ctx, updateLockPath, false, false)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, update.Close()) }()
	wait, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	exclusive, err := acquireExclusiveRuntime(wait)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, exclusive.Close()) }()
	tools, err := services(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, tools.Close()) }()
	process, err := tools.active(ctx, record.Version, record.ClientGID)
	if err != nil {
		return err
	}
	if process == nil {
		return ErrUntrusted
	}
	defer func() { resultErr = errors.Join(resultErr, process.Close()) }()
	if err := tools.runtimeHealthy(ctx); err != nil {
		return err
	}
	if err := tools.resourcesDrained(ctx, record.ClientUID, 0, process.pid); err != nil {
		return err
	}
	// Re-read every authority after locks and live queries. Neither earlier
	// success nor a root-owned path alone permits publishing a changed identity.
	current, actualConfigs, err := enrollmentIdentity(ctx)
	if err != nil || current != record || !sameEnrollmentConfigs(configs, actualConfigs) {
		return errors.Join(ErrUntrusted, err)
	}
	actualPrevious, err := enrollmentState(ctx, record)
	if err != nil || !bytes.Equal(previous, actualPrevious) {
		return errors.Join(ErrTransition, err)
	}
	checked, err := recorded(ctx, record)
	if err != nil || checked.record != record {
		return errors.Join(ErrUntrusted, err)
	}
	if err := controller(checked); err != nil {
		return err
	}
	if err := publicNativeLayout(ctx, checked); err != nil {
		return err
	}
	if err := validateHeldEnrollmentLock(runtimeLockPath, exclusive); err != nil {
		return err
	}
	if err := validateHeldEnrollmentLock(updateLockPath, update); err != nil {
		return err
	}
	live, err := tools.active(ctx, record.Version, record.ClientGID)
	if err != nil {
		return err
	}
	if live == nil {
		return ErrUntrusted
	}
	defer func() { resultErr = errors.Join(resultErr, live.Close()) }()
	if live.pid != process.pid {
		return ErrTransition
	}
	for _, p := range []*serviceProcess{process, live} {
		if exited, err := p.exited(); err != nil || exited {
			return errors.Join(ErrTransition, err)
		}
	}

	if err := tools.resourcesDrained(ctx, record.ClientUID, 0, live.pid); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	body, err := json.Marshal(record)
	if err != nil {
		return ErrUntrusted
	}
	if len(previous) != 0 {
		return nil
	}
	return replaceProtected(ctx, registrationPath, nil, body, 0644, 0)
}

func enrollmentIdentity(ctx context.Context) (registration, [3]savedConfig, error) {
	var record registration
	var configs [3]savedConfig
	if err := protectedDirectory(releaseinstall.SystemInstallRoot); err != nil {
		return record, configs, err
	}
	path := filepath.Join(releaseinstall.SystemInstallRoot, "current")
	before, err := os.Lstat(path)
	if err != nil || before.Mode()&os.ModeSymlink == 0 {
		return record, configs, ErrUntrusted
	}
	owner, ok := before.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 {
		return record, configs, ErrUntrusted
	}
	target, err := os.Readlink(path)
	version := filepath.Base(target)
	if err != nil || !exactVersion(version) || target != filepath.Join("versions", version) {
		return record, configs, ErrUntrusted
	}
	manifestBody, err := readProtected(ctx, nativeVersionPath(version, "helox-release-manifest.json"), 1<<20)
	if err != nil {
		return record, configs, err
	}
	manifest, err := releaseinstall.ParseReleaseManifest(manifestBody)
	if err != nil || manifest.Version() != version {
		return record, configs, ErrUntrusted
	}
	for index, path := range configPaths() {
		configs[index], err = snapshotConfig(ctx, path)
		if err != nil {
			return record, configs, err
		}
		// These fixed configs must also be readable by the ordinary controller.
		if configs[index].Mode&0004 == 0 {
			return record, configs, ErrUntrusted
		}
	}
	var policy nativePolicyConfig
	if err := canonicalConfig(configs[0].Body, &policy); err != nil {
		return record, configs, err
	}
	record = registration{registrationSchema, "NATIVE", version, manifest.SHA256(), policy.ClientUID, policy.ClientGID}
	if _, _, err := parseNativeInstallationConfigs(record, configs); err != nil {
		return record, configs, err
	}
	after, err := os.Lstat(path)
	actual, linkErr := os.Readlink(path)
	if err != nil || linkErr != nil || !os.SameFile(before, after) || actual != target {
		return record, configs, ErrUntrusted
	}
	return record, configs, nil
}

func sameEnrollmentConfigs(a, b [3]savedConfig) bool {
	for i := range a {
		if a[i].Mode != b[i].Mode || a[i].GID != b[i].GID || a[i].SHA256 != b[i].SHA256 || !bytes.Equal(a[i].Body, b[i].Body) {
			return false
		}
	}
	return true
}

func enrollmentState(ctx context.Context, record registration) ([]byte, error) {
	if _, err := os.Lstat(journalPath); !errors.Is(err, os.ErrNotExist) {
		return nil, ErrTransition
	}
	var previous []byte
	if _, err := os.Lstat(registrationPath); err == nil {
		body, err := readProtected(ctx, registrationPath, 16<<10)
		if err != nil {
			return nil, err
		}
		existing, err := parseRegistration(body)
		canonical, marshalErr := json.Marshal(record)
		if err != nil || marshalErr != nil || existing != record || !bytes.Equal(body, canonical) {
			return nil, ErrUntrusted
		}
		file, info, err := openProtectedFile(registrationPath)
		if err != nil {
			return nil, err
		}
		closeErr := file.Close()
		owner, ok := info.Sys().(*syscall.Stat_t)
		if closeErr != nil || !ok || owner.Gid != 0 || info.Mode().Perm() != 0644 {
			return nil, ErrUntrusted
		}
		previous = body
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, ErrUntrusted
	}
	if _, err := os.Lstat(historyPath); err == nil {
		if len(previous) == 0 {
			return nil, ErrTransition
		}
		body, err := readProtected(ctx, historyPath, 128<<10)
		if err != nil {
			return nil, err
		}
		history, err := parseJournal(body)
		if err != nil {
			return nil, err
		}
		expected := history.Old
		switch history.State {
		case "COMMITTED":
			expected = candidateRegistration(history)
		case "ROLLED_BACK":
		default:
			return nil, ErrTransition
		}
		if expected != record {
			return nil, ErrTransition
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, ErrTransition
	}
	return previous, nil
}

func publicNativeLayout(ctx context.Context, installed InstalledRelease) error {
	directory := filepath.Dir(installed.ClientPath())
	for _, path := range []string{releaseinstall.SystemInstallRoot, filepath.Join(releaseinstall.SystemInstallRoot, "versions"), directory, stateRoot, proofRoot, filepath.Join(proofRoot, installed.manifest.SHA256()), filepath.Dir(policyConfigPath), filepath.Dir(policyUnitPath)} {
		if err := protectedDirectory(path); err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil || info.Mode().Perm() != 0755 {
			return ErrUntrusted
		}
	}
	// The shared recorded reader hashes the native payload and metadata. Proof
	// bytes are also published read-only for ordinary SDK verification.
	entries, err := os.ReadDir(filepath.Join(proofRoot, installed.manifest.SHA256()))
	if err != nil || len(entries) == 0 || len(entries) > 16 {
		return ErrUntrusted
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		file, info, err := openProtectedFile(filepath.Join(proofRoot, installed.manifest.SHA256(), entry.Name()))
		if err != nil {
			return err
		}
		if err := file.Close(); err != nil || info.Mode().Perm() != 0444 {
			return ErrUntrusted
		}
	}
	return nil
}

func prepareEnrollmentLock(path string) error {
	if path != runtimeLockPath && path != updateLockPath {
		return ErrUntrusted
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0444)
	if err == nil {
		modeErr, syncErr, closeErr := file.Chmod(0444), file.Sync(), file.Close()
		if modeErr != nil || syncErr != nil || closeErr != nil {
			return ErrUntrusted
		}
		if err := syncProtectedDirectory(stateRoot); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrExist) {
		return ErrUntrusted
	}
	return validateEnrollmentLock(path)
}

func validateEnrollmentLock(path string) error {
	if path != runtimeLockPath && path != updateLockPath {
		return ErrUntrusted
	}
	file, info, err := openProtectedFile(path)
	if err != nil {
		return err
	}
	closeErr := file.Close()
	owner, ok := info.Sys().(*syscall.Stat_t)
	if closeErr != nil || !ok || owner.Gid != 0 || info.Mode().Perm() != 0444 || info.Size() != 0 {
		return ErrUntrusted
	}
	return nil
}

func validateHeldEnrollmentLock(path string, owner *UpdateOwner) error {
	if err := validateEnrollmentLock(path); err != nil {
		return err
	}
	if owner == nil || owner.file == nil {
		return ErrUntrusted
	}
	held, err := owner.file.Stat()
	if err != nil {
		return ErrUntrusted
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(held, current) {
		return ErrUntrusted
	}
	return nil
}
