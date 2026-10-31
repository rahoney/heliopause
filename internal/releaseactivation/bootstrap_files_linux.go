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
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const bootstrapDirectory = releaseinstall.SystemStagingRoot + "/bootstrap"
const bootstrapIntentPath = stateRoot + "/native-bootstrap.json"

type nativeBootstrapSet struct {
	record  registration
	release wireRelease
	proof   []byte
	configs [3]savedConfig
}
type bootstrapChecks struct {
	read      func(context.Context) (nativeBootstrapSet, error)
	self      func(nativeBootstrapSet) error
	installed func(context.Context, registration) (InstalledRelease, error)
	installer func(nativeBootstrapSet) (*releaseinstall.Installer, error)
	consumers func(context.Context, int, int) error
}

// BootstrapFiles prepares independently verified first-install bytes only.
// An administrator must authenticate this helper before its first root launch.
// Successful preparation is not registration, runtime health, or quarantine.
func BootstrapFiles(ctx context.Context) error {
	return prepareBootstrapFiles(ctx, bootstrapChecks{readBootstrapSet, verifyBootstrapExecutable, readRecordedRelease, func(set nativeBootstrapSet) (*releaseinstall.Installer, error) {
		return releaseinstall.NewSystemInstaller(initialBundleVerifier{set.record, set.proof})
	}, protectedConsumersAbsent})
}

func prepareBootstrapFiles(ctx context.Context, checks bootstrapChecks) (resultErr error) {
	if ctx == nil || runtime.GOARCH != "amd64" || os.Geteuid() != 0 || os.Getegid() != 0 {
		return ErrCaller
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if checks.read == nil || checks.self == nil || checks.installed == nil || checks.installer == nil || checks.consumers == nil {
		return ErrUntrusted
	}
	owner, err := acquireInstallerOwner(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, owner.Close()) }()
	set, err := checks.read(ctx)
	if err != nil {
		return err
	}
	if err := checks.self(set); err != nil {
		return err
	}
	intent, err := bootstrapState(ctx, set.record)
	if err != nil {
		return err
	}
	if !intent {
		body, err := json.Marshal(set.record)
		if err != nil {
			return ErrUntrusted
		}
		if err := replaceProtected(ctx, bootstrapIntentPath, nil, body, 0600, 0); err != nil {
			return err
		}
	}
	for _, path := range []string{runtimeLockPath, updateLockPath} {
		if err := prepareEnrollmentLock(path); err != nil {
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
	if err := checks.consumers(ctx, 0, 0); err != nil {
		return err
	}
	actual, err := checks.read(ctx)
	if err != nil || actual.record != set.record || !sameEnrollmentConfigs(actual.configs, set.configs) {
		return errors.Join(ErrUntrusted, err)
	}
	if err := checks.self(actual); err != nil {
		return err
	}
	if present, err := bootstrapState(ctx, set.record); err != nil || !present {
		return errors.Join(ErrTransition, err)
	}
	if err := validateHeldEnrollmentLock(runtimeLockPath, exclusive); err != nil {
		return err
	}
	if err := validateHeldEnrollmentLock(updateLockPath, update); err != nil {
		return err
	}
	installer, err := checks.installer(set)
	if err != nil {
		return err
	}
	request := releaseinstall.Request{BundleDirectory: bootstrapDirectory, InstallRoot: releaseinstall.SystemInstallRoot}
	name, err := releaseinstall.SystemCopyStageName(set.record.ManifestSHA256)
	if err != nil {
		return ErrUntrusted
	}
	pending := filepath.Join(releaseinstall.SystemInstallRoot, "versions", name)
	if _, err := os.Lstat(pending); err == nil {
		if !intent {
			return ErrTransition
		}
		if _, err := os.Lstat(filepath.Dir(nativeVersionPath(set.record.Version, "helox-linux-amd64"))); !errors.Is(err, os.ErrNotExist) {
			return ErrTransition
		}
		if err := installer.DiscardSystemCopy(ctx, request); err != nil {
			return errors.Join(ErrCleanup, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrUntrusted
	}
	target := filepath.Dir(nativeVersionPath(set.record.Version, "helox-linux-amd64"))
	if _, err := os.Lstat(target); err == nil {
		request.BundleDirectory = target
		if _, err := installer.ActivateExisting(ctx, request); err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if _, err := installer.Install(ctx, request); err != nil {
			return err
		}
	} else {
		return ErrUntrusted
	}
	record, configs, err := enrollmentIdentity(ctx)
	if err != nil || record != set.record || !sameEnrollmentConfigs(configs, set.configs) {
		return errors.Join(ErrUntrusted, err)
	}
	installed, err := checks.installed(ctx, set.record)
	if err != nil || installed.record != set.record {
		return errors.Join(ErrUntrusted, err)
	}
	if err := validateHeldEnrollmentLock(runtimeLockPath, exclusive); err != nil {
		return err
	}
	if err := validateHeldEnrollmentLock(updateLockPath, update); err != nil {
		return err
	}
	// Keep the source and small intent for explicit native configuration,
	// installed-helper registration and ordinary-user bootstrap acceptance.
	return ctx.Err()
}

func bootstrapState(ctx context.Context, record registration) (bool, error) {
	body, err := json.Marshal(record)
	if err != nil {
		return false, ErrUntrusted
	}
	if _, err := parseRegistration(body); err != nil {
		return false, err
	}
	for _, path := range []string{"/usr/libexec/heliopause", filepath.Join(releaseinstall.SystemInstallRoot, "releases"), registrationPath, journalPath, historyPath} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return false, ErrUnmanaged
		}
	}
	intent := false
	if _, err := os.Lstat(bootstrapIntentPath); err == nil {
		stored, err := readBootstrapIntent(ctx)
		if err != nil || stored != record {
			return false, errors.Join(ErrUntrusted, err)
		}
		intent = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, ErrUntrusted
	}
	for _, path := range []string{"/usr/lib", stateRoot} {
		if err := protectedDirectory(path); err != nil {
			return false, err
		}
	}
	if !intent {
		for _, path := range []string{runtimeLockPath, updateLockPath} {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return false, ErrTransition
			}
		}
	}
	root := releaseinstall.SystemInstallRoot
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return intent, nil
	} else if err != nil {
		return false, ErrUntrusted
	}
	if err := protectedDirectory(root); err != nil {
		return false, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, ErrUntrusted
	}
	for _, entry := range entries {
		if entry.Name() != "versions" && (!intent || entry.Name() != "current") {
			return false, ErrUnmanaged
		}
	}
	versions := filepath.Join(root, "versions")
	if _, err := os.Lstat(versions); err == nil {
		if err := protectedDirectory(versions); err != nil {
			return false, err
		}
		entries, err := os.ReadDir(versions)
		if err != nil {
			return false, ErrUntrusted
		}
		name, err := releaseinstall.SystemCopyStageName(record.ManifestSHA256)
		if err != nil {
			return false, ErrUntrusted
		}
		for _, entry := range entries {
			if !intent || entry.Name() != record.Version && entry.Name() != name {
				return false, ErrUnmanaged
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, ErrUntrusted
	}
	current := filepath.Join(root, "current")
	if info, err := os.Lstat(current); err == nil {
		if !intent || info.Mode()&os.ModeSymlink == 0 {
			return false, ErrUntrusted
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		target, err := os.Readlink(current)
		if !ok || owner.Uid != 0 || owner.Nlink != 1 || err != nil || target != filepath.Join("versions", record.Version) {
			return false, ErrUntrusted
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, ErrUntrusted
	}
	return intent, nil
}

func readBootstrapIntent(ctx context.Context) (registration, error) {
	body, err := readProtected(ctx, bootstrapIntentPath, 16<<10)
	if err != nil {
		return registration{}, err
	}
	record, err := parseRegistration(body)
	if err != nil {
		return registration{}, err
	}
	canonical, err := json.Marshal(record)
	if err != nil || !bytes.Equal(body, canonical) {
		return registration{}, ErrUntrusted
	}
	file, info, err := openProtectedFile(bootstrapIntentPath)
	if err != nil {
		return registration{}, err
	}
	closeErr := file.Close()
	owner, ok := info.Sys().(*syscall.Stat_t)
	if closeErr != nil || !ok || owner.Gid != 0 || info.Mode().Perm() != 0600 {
		return registration{}, ErrUntrusted
	}
	return record, nil
}

func readBootstrapSet(ctx context.Context) (nativeBootstrapSet, error) {
	var set nativeBootstrapSet
	if ctx == nil {
		return set, ErrUntrusted
	}
	if err := protectedDirectory(bootstrapDirectory); err != nil {
		return set, err
	}
	info, err := os.Lstat(bootstrapDirectory)
	if err != nil || info.Mode().Perm() != 0700 {
		return set, ErrUntrusted
	}
	body, err := readProtected(ctx, filepath.Join(bootstrapDirectory, "helox-release-manifest.json"), 1<<20)
	if err != nil {
		return set, err
	}
	manifest, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		return set, ErrUntrusted
	}
	proofs, err := readStoredProofs(ctx, manifest.SHA256())
	if err != nil {
		return set, err
	}
	release, err := newWireRelease(body, proofs)
	if err != nil {
		return set, err
	}
	if manifest.RuntimeLockSHA256() != runtimeidentity.RuntimeLockSHA256 || manifest.GVisorRelease() != runtimeidentity.GVisorRelease || manifest.GVisorCommit() != runtimeidentity.GVisorCommit {
		return set, ErrUntrusted
	}
	proof, err := release.verifyProvenance(ctx)
	if err != nil {
		return set, err
	}
	entries, err := os.ReadDir(bootstrapDirectory)
	if err != nil || len(entries) != len(release.files)+1 {
		return set, ErrUntrusted
	}
	expected := map[string]bool{"helox-release-manifest.json": true}
	for _, file := range release.files {
		expected[file.Name] = true
		mode := os.FileMode(0500)
		if file.Role == "release-metadata" || file.IsRuntimeData() {
			mode = 0600
		}
		if err := hashProtected(ctx, filepath.Join(bootstrapDirectory, file.Name), file.SHA256, file.Size, mode); err != nil {
			return set, err
		}
	}
	for _, entry := range entries {
		if !expected[entry.Name()] || entry.IsDir() {
			return set, ErrUntrusted
		}
	}
	if err := hashProtected(ctx, filepath.Join(bootstrapDirectory, "helox-release-manifest.json"), manifest.SHA256(), int64(len(body)), 0600); err != nil {
		return set, err
	}
	sbom, err := readProtected(ctx, filepath.Join(bootstrapDirectory, "helox-release-sbom.cdx.json"), 8<<20)
	if err != nil {
		return set, err
	}
	if err := releaseinstall.ValidateReleaseSBOM(sbom, manifest); err != nil {
		return set, errors.Join(ErrUntrusted, err)
	}
	images, err := readProtected(ctx, filepath.Join(bootstrapDirectory, "helox-runtime-images.json"), 1<<20)
	if err != nil {
		return set, err
	}
	if err := releaseinstall.ValidateRuntimeImagesForRelease(images, manifest); err != nil {
		return set, errors.Join(ErrUntrusted, err)
	}
	notices, err := readProtected(ctx, filepath.Join(bootstrapDirectory, "pnpm-NOTICE.txt"), 64<<10)
	if err != nil {
		return set, err
	}
	if err := releaseinstall.ValidatePNPMNotices(notices); err != nil {
		return set, errors.Join(ErrUntrusted, err)
	}
	bunNotices, err := readProtected(ctx, filepath.Join(bootstrapDirectory, "bun-NOTICE.txt"), 64<<10)
	if err != nil {
		return set, err
	}
	if err := releaseinstall.ValidateBunNotices(bunNotices); err != nil {
		return set, errors.Join(ErrUntrusted, err)
	}
	if manifest.HasPipxRuntime() {
		pipxNotices, err := readProtected(ctx, filepath.Join(bootstrapDirectory, "pipx-NOTICE.txt"), 64<<10)
		if err != nil {
			return set, errors.Join(ErrUntrusted, err)
		}
		if err := releaseinstall.ValidatePipxNotices(pipxNotices); err != nil {
			return set, errors.Join(ErrUntrusted, err)
		}
	}

	configs, err := readNativeConfigSnapshots(ctx)
	if err != nil {
		return set, err
	}
	record, err := nativeRegistration(manifest, configs)
	if err != nil {
		return set, err
	}
	return nativeBootstrapSet{record, release, proof, configs}, nil
}

func verifyBootstrapExecutable(set nativeBootstrapSet) error {
	if os.Geteuid() != 0 || os.Getegid() != 0 || set.record.Schema != registrationSchema || len(set.proof) == 0 {
		return ErrCaller
	}
	actual, err := os.Stat("/proc/self/exe")
	if err != nil {
		return ErrCaller
	}
	expected, err := os.Stat(filepath.Join(bootstrapDirectory, "haa-release-installer-linux-amd64"))
	if err != nil || !os.SameFile(actual, expected) {
		return ErrCaller
	}
	return nil
}

type initialBundleVerifier struct {
	record registration
	proof  []byte
}

func (v initialBundleVerifier) Verify(ctx context.Context, directory string) error {
	if ctx == nil || os.Geteuid() != 0 || len(v.proof) == 0 {
		return ErrCaller
	}
	bodyRecord, err := json.Marshal(v.record)
	if err != nil {
		return ErrUntrusted
	}
	if _, err := parseRegistration(bodyRecord); err != nil {
		return err
	}
	if directory != bootstrapDirectory && directory != filepath.Dir(nativeVersionPath(v.record.Version, "helox-linux-amd64")) {
		return ErrUntrusted
	}
	body, err := readProtected(ctx, filepath.Join(directory, "helox-release-manifest.json"), 1<<20)
	if err != nil {
		return err
	}
	release, err := newWireRelease(body, [][]byte{v.proof})
	if err != nil {
		return err
	}
	if release.manifest.RuntimeLockSHA256() != runtimeidentity.RuntimeLockSHA256 || release.manifest.Version() != v.record.Version || release.manifest.SHA256() != v.record.ManifestSHA256 || release.manifest.GVisorRelease() != runtimeidentity.GVisorRelease || release.manifest.GVisorCommit() != runtimeidentity.GVisorCommit {
		return ErrUntrusted
	}
	_, err = release.verifyProvenance(ctx)
	return err
}

func cleanupBootstrapSource(ctx context.Context, installed InstalledRelease) error {
	if ctx == nil || os.Geteuid() != 0 {
		return ErrCaller
	}
	if _, err := os.Lstat(bootstrapIntentPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return ErrCleanup
	}
	if _, err := os.Lstat(bootstrapDirectory); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return ErrCleanup
	}
	intent, err := readBootstrapIntent(ctx)
	if err != nil || intent != installed.record {
		return errors.Join(ErrCleanup, err)
	}
	checked, err := readRecordedRelease(ctx, installed.record)
	if err != nil || checked.record != installed.record {
		return errors.Join(ErrCleanup, err)
	}
	return discardBootstrapSource(ctx, checked)
}

// The production caller first re-verifies the installed SDK/intent. Interrupted
// source cleanup can then authenticate any remaining subset from retained bytes.
func discardBootstrapSource(ctx context.Context, installed InstalledRelease) error {
	if ctx == nil || os.Geteuid() != 0 {
		return ErrCleanup
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrCleanup, err)
	}
	if err := protectedDirectory(bootstrapDirectory); err != nil {
		return errors.Join(ErrCleanup, err)
	}
	identity, err := os.Lstat(bootstrapDirectory)
	if err != nil || identity.Mode().Perm() != 0700 {
		return ErrCleanup
	}
	body, err := readProtected(ctx, nativeVersionPath(installed.Version(), "helox-release-manifest.json"), 1<<20)
	if err != nil || bodySHA256(body) != installed.manifest.SHA256() {
		return errors.Join(ErrCleanup, err)
	}
	expected := map[string]releaseinstall.ReleaseFile{"helox-release-manifest.json": {Name: "helox-release-manifest.json", Role: "release-metadata", SHA256: installed.manifest.SHA256(), Size: int64(len(body))}}
	payloads, err := installed.manifest.NativeUpdatePayloads()
	if err != nil {
		return errors.Join(ErrCleanup, err)
	}
	for _, file := range payloads {
		expected[file.Name] = file
	}

	entries, err := os.ReadDir(bootstrapDirectory)
	if err != nil || len(entries) > len(expected) {
		return ErrCleanup
	}
	identities := map[string]os.FileInfo{}
	for _, entry := range entries {
		value, found := expected[entry.Name()]
		if !found || entry.IsDir() {
			return ErrCleanup
		}
		mode := os.FileMode(0500)
		if value.Role == "release-metadata" || value.IsRuntimeData() {
			mode = 0600
		}
		path := filepath.Join(bootstrapDirectory, entry.Name())
		if err := hashProtected(ctx, path, value.SHA256, value.Size, mode); err != nil {
			return errors.Join(ErrCleanup, err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return ErrCleanup
		}
		identities[entry.Name()] = info
	}
	current, err := os.Lstat(bootstrapDirectory)
	if err != nil || !os.SameFile(identity, current) {
		return ErrCleanup
	}
	actual, err := os.ReadDir(bootstrapDirectory)
	if err != nil || len(actual) != len(identities) {
		return ErrCleanup
	}
	for _, entry := range actual {
		if identities[entry.Name()] == nil {
			return ErrCleanup
		}
	}
	for _, entry := range actual {
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrCleanup, err)
		}
		path := filepath.Join(bootstrapDirectory, entry.Name())
		before := identities[entry.Name()]
		file, current, err := openProtectedFile(path)
		if err != nil {
			return errors.Join(ErrCleanup, err)
		}
		closeErr := file.Close()
		if closeErr != nil || !os.SameFile(before, current) || current.Size() != before.Size() || current.Mode() != before.Mode() || current.ModTime() != before.ModTime() {
			return ErrCleanup
		}
		if err := os.Remove(path); err != nil {
			return errors.Join(ErrCleanup, err)
		}
	}
	if err := syncProtectedDirectory(bootstrapDirectory); err != nil {
		return errors.Join(ErrCleanup, err)
	}
	current, err = os.Lstat(bootstrapDirectory)
	if err != nil || !os.SameFile(identity, current) {
		return ErrCleanup
	}
	if err := os.Remove(bootstrapDirectory); err != nil {
		return errors.Join(ErrCleanup, err)
	}
	return syncProtectedDirectory(releaseinstall.SystemStagingRoot)
}
