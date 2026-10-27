// Package releaseinstall owns the fail-closed local half of verified Heliopause
// release activation. Remote acquisition and attestation verification stay at
// the caller boundary; this package never upgrades trust from a path or flag.
package releaseinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const (
	manifestName      = "helox-release-manifest.json"
	runtimeImagesName = "helox-runtime-images.json"
	sbomName          = "helox-release-sbom.cdx.json"
	installRecordName = "helox-install.json"
	manifestSchema    = "helox.release-manifest/v1"
	maxBundleFileSize = 1 << 30
	SystemInstallRoot = runtimeidentity.NativeInstallRoot
	SystemStagingRoot = "/var/lib/heliopause/install-staging"
)

var (
	versionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	digestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// BundleVerifier is the mandatory upstream trust boundary. An implementation
// must validate expected repository, protected workflow, commit and artifact
// attestation before this installer reads or activates bundle content.
type BundleVerifier interface {
	Verify(context.Context, string) error
}

// Request identifies the already-attested bundle and versioned install root.
// Both paths must be canonical absolute paths; system mode fixes both roots.
type Request struct {
	BundleDirectory string
	InstallRoot     string
}

// Result describes an activation that has completed atomically.
type Result struct {
	Version        string
	InstallRoot    string
	ActivePath     string
	ManifestSHA256 string
}

// Installer activates only a bundle accepted by its verifier.
type Installer struct {
	verifier BundleVerifier
	goos     string
	goarch   string
	system   bool
}

// New creates an installer for the running platform.
func New(verifier BundleVerifier) (*Installer, error) {
	return newInstaller(verifier, runtime.GOOS, runtime.GOARCH)
}

// NewSystemInstaller is a copy/pointer primitive for the narrow privileged
// installation adapter. It grants no IPC, Policy, lease or service authority.
// Its target and staging roots are fixed; the mandatory verifier is retained.
func NewSystemInstaller(verifier BundleVerifier) (*Installer, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || os.Geteuid() != 0 {
		return nil, errors.New("protected release installation requires Linux amd64 root")
	}
	installer, err := newInstaller(verifier, "linux", "amd64")
	if err != nil {
		return nil, err
	}
	installer.system = true
	return installer, nil
}

func newInstaller(verifier BundleVerifier, goos, goarch string) (*Installer, error) {
	if verifier == nil || !supportedPlatform(goos, goarch) {
		return nil, errors.New("verified release installation is unavailable for this platform")
	}
	return &Installer{verifier: verifier, goos: goos, goarch: goarch}, nil
}

// Install validates the release bundle before creating any install state. A
// failed stage or activation preserves the prior current pointer.
func (i *Installer) Install(ctx context.Context, request Request) (Result, error) {
	input, err := i.validatedInput(ctx, request, false)
	if err != nil {
		return Result{}, err
	}
	if err := i.ensureRoot(input.root); err != nil {
		return Result{}, err
	}
	return activate(ctx, input.root, input.manifest.Version, input.body, input.files, input.bundle, i.system)
}

// ActivateExisting re-verifies an existing version in this installation before
// using the same atomic pointer primitive. The lifecycle owner must hold its
// original-state guard and confirm service drainage before invoking rollback.
func (i *Installer) ActivateExisting(ctx context.Context, request Request) (Result, error) {
	input, err := i.validatedInput(ctx, request, true)
	if err != nil {
		return Result{}, err
	}
	if err := i.ensureRoot(input.root); err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := activateCurrent(input.root, input.manifest.Version); err != nil {
		return Result{}, err
	}
	sum := sha256.Sum256(input.body)
	return Result{Version: input.manifest.Version, InstallRoot: input.root, ActivePath: filepath.Join(input.root, "current"), ManifestSHA256: hex.EncodeToString(sum[:])}, nil
}

type installationInput struct {
	bundle, root string
	manifest     manifest
	body         []byte
	files        []asset
}

func (i *Installer) validatedInput(ctx context.Context, request Request, existing bool) (installationInput, error) {
	if i == nil || i.verifier == nil || ctx == nil {
		return installationInput{}, errors.New("verified release installation is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return installationInput{}, err
	}
	bundle, err := canonicalDirectory(request.BundleDirectory)
	if err != nil {
		return installationInput{}, errors.New("release bundle directory is invalid")
	}
	root, err := canonicalAbsolute(request.InstallRoot)
	if err != nil {
		return installationInput{}, errors.New("release install root is invalid")
	}
	if existing && (filepath.Dir(bundle) != filepath.Join(root, "versions") || !versionPattern.MatchString(filepath.Base(bundle))) {
		return installationInput{}, errors.New("rollback bundle is not an existing installation version")
	}
	if i.system {
		if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || os.Geteuid() != 0 || root != SystemInstallRoot || (!existing && filepath.Dir(bundle) != SystemStagingRoot) {
			return installationInput{}, errors.New("protected installation paths or caller are invalid")
		}
		if err := verifySystemDirectory(root, !existing); err != nil {
			return installationInput{}, err
		}
		if err := verifySystemDirectory(filepath.Join(root, "versions"), !existing); err != nil {
			return installationInput{}, err
		}
		if err := verifySystemDirectory(bundle, false); err != nil {
			return installationInput{}, err
		}
		if err := verifySystemRegularFile(filepath.Join(bundle, manifestName)); err != nil {
			return installationInput{}, err
		}
		info, err := os.Lstat(filepath.Join(bundle, manifestName))
		if err != nil || info.Size() <= 0 || info.Size() > maximumManifestBytes {
			return installationInput{}, errors.New("protected release manifest size is invalid")
		}
	}
	if err := i.verifier.Verify(ctx, bundle); err != nil {
		return installationInput{}, errors.Join(errors.New("release attestation verification failed"), err, ctx.Err())
	}
	value, body, err := readManifest(bundle)
	if err != nil {
		return installationInput{}, err
	}
	if existing && value.Version != filepath.Base(bundle) {
		return installationInput{}, errors.New("rollback manifest differs from the existing version")
	}
	files, err := selectFiles(value, i.goos, i.goarch)
	if err != nil {
		return installationInput{}, err
	}
	var strict ReleaseManifest
	if i.system {
		strict, err = ParseReleaseManifest(body)
		if err != nil {
			return installationInput{}, err
		}
		if _, err := strict.NativeUpdatePayloads(); err != nil {
			return installationInput{}, err
		}
		for _, name := range append([]string{manifestName, sbomName, runtimeImagesName}, selectedNames(files)...) {
			if err := verifySystemRegularFile(filepath.Join(bundle, name)); err != nil {
				return installationInput{}, err
			}
		}
		for _, file := range strict.Files() {
			if file.Role == "release-metadata" {
				info, err := os.Lstat(filepath.Join(bundle, file.Name))
				if err != nil || info.Size() != file.Size {
					return installationInput{}, errors.New("protected metadata size differs from its manifest")
				}
			}
		}
	}
	if err := verifyBundleFiles(bundle, value, files); err != nil {
		return installationInput{}, err
	}
	if i.system {
		sbom, err := readRegular(filepath.Join(bundle, sbomName))
		if err != nil {
			return installationInput{}, err
		}
		if err := ValidateReleaseSBOM(sbom, strict); err != nil {
			return installationInput{}, err
		}
		images, err := readRegular(filepath.Join(bundle, runtimeImagesName))
		if err != nil {
			return installationInput{}, err
		}
		if err := ValidateRuntimeImages(images); err != nil {
			return installationInput{}, err
		}
		notices, err := readRegular(filepath.Join(bundle, "pnpm-NOTICE.txt"))
		if err != nil {
			return installationInput{}, err
		}
		if err := ValidatePNPMNotices(notices); err != nil {
			return installationInput{}, err
		}
		bunNotices, err := readRegular(filepath.Join(bundle, "bun-NOTICE.txt"))
		if err != nil {
			return installationInput{}, err
		}
		if err := ValidateBunNotices(bunNotices); err != nil {
			return installationInput{}, err
		}
		if strict.HasPipxRuntime() {
			pipxNotices, err := readRegular(filepath.Join(bundle, "pipx-NOTICE.txt"))
			if err != nil {
				return installationInput{}, err
			}
			if err := ValidatePipxNotices(pipxNotices); err != nil {
				return installationInput{}, err
			}
		}

	}
	if err := ctx.Err(); err != nil {
		return installationInput{}, err
	}
	return installationInput{bundle, root, value, body, files}, nil
}

func selectedNames(files []asset) []string {
	names := make([]string, len(files))
	for index, file := range files {
		names[index] = file.Name
	}
	return names
}

func (i *Installer) ensureRoot(root string) error {
	if i.system {
		return ensureDirectoryMode(root, 0755)
	}
	return ensureInstallRoot(root)
}

type manifest struct {
	Schema        string      `json:"schema"`
	Version       string      `json:"version"`
	SourceCommit  string      `json:"source_commit"`
	WorkflowRun   string      `json:"workflow_run"`
	RuntimeLock   runtimeLock `json:"runtime_lock"`
	SBOM          record      `json:"sbom"`
	RuntimeImages record      `json:"runtime_images"`
	Assets        []asset     `json:"assets"`
}

type record struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type runtimeLock struct {
	SHA256        string `json:"sha256"`
	GVisorRelease string `json:"gvisor_release"`
	GVisorCommit  string `json:"gvisor_commit"`
}

type asset struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Role     string `json:"role"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

func readManifest(bundle string) (manifest, []byte, error) {
	body, err := readRegular(filepath.Join(bundle, manifestName))
	if err != nil {
		return manifest{}, nil, errors.New("release manifest is unavailable")
	}
	value, err := parseManifest(body)
	return value, body, err
}

func selectFiles(value manifest, goos, goarch string) ([]asset, error) {
	want := map[string]string{"helox-" + goos + "-" + goarch: "helox"}
	if goos == "linux" && goarch == "amd64" {
		want["haa_gvisor_observer-linux-amd64"] = "gvisor-observer"
		want["haa-network-policy-helper-linux-amd64"] = "network-policy-helper"
		for _, item := range value.Assets {
			if item.Name == "pnpm-linux-amd64" || item.Name == "pnpm-NOTICE.txt" {
				role := "pnpm"
				if item.Name == "pnpm-NOTICE.txt" {
					role = "release-metadata"
				}
				want[item.Name] = role
			}
		}
		for _, item := range value.Assets {
			if item.Name == "bun-linux-amd64" || item.Name == "bun-NOTICE.txt" {
				role := "bun"
				if item.Name == "bun-NOTICE.txt" {
					role = "release-metadata"
				}
				want[item.Name] = role
			}
		}
		for _, item := range value.Assets {
			if item.Name == runtimeidentity.PipxRuntimeAssetName {
				want[item.Name] = "pipx-runtime"
			}
			if item.Name == "pipx-NOTICE.txt" {
				want[item.Name] = "release-metadata"
			}
		}
		for _, item := range value.Assets {
			if item.Name == "haa-release-installer-linux-amd64" {
				want[item.Name] = "release-installer"
				break
			}
		}
	}
	if goos == "linux" && goarch == "amd64" {
		_, zip := want[runtimeidentity.PipxRuntimeAssetName]
		_, notices := want["pipx-NOTICE.txt"]
		if zip != notices {
			return nil, errors.New("pipx DATA and notices must be selected together")
		}
		if zip {
			for name, role := range nativeUpdateRoleSet() {
				if want[name] != role {
					return nil, errors.New("pipx DATA requires the complete native role set")
				}
			}
			if want["pnpm-NOTICE.txt"] != "release-metadata" || want["bun-NOTICE.txt"] != "release-metadata" {
				return nil, errors.New("pipx DATA requires existing client notices")
			}
		}
	}
	selected := make([]asset, 0, len(want))
	seen := make(map[string]bool, len(want))
	for _, item := range value.Assets {
		role, wanted := want[item.Name]
		if !wanted {
			continue
		}
		if seen[item.Name] || item.Role != role || item.Platform != goos+"/"+goarch || !validRecord(record{Name: item.Name, SHA256: item.SHA256, Size: item.Size}, item.Name) {
			return nil, errors.New("release asset binding is invalid")
		}
		if ReleaseFile(item).IsRuntimeData() && item.Size > 8<<20 || item.Name == "pipx-NOTICE.txt" && item.Size > 64<<10 {
			return nil, errors.New("pipx DATA or notices exceed their fixed role")
		}
		seen[item.Name] = true
		selected = append(selected, item)
	}
	if len(selected) != len(want) {
		return nil, errors.New("release asset set is incomplete for this platform")
	}
	return selected, nil
}

func verifyBundleFiles(bundle string, value manifest, selected []asset) error {
	if err := verifyRecord(filepath.Join(bundle, sbomName), value.SBOM); err != nil {
		return errors.New("release SBOM binding is invalid")
	}
	if err := verifyRecord(filepath.Join(bundle, runtimeImagesName), value.RuntimeImages); err != nil {
		return errors.New("runtime image manifest binding is invalid")
	}
	for _, item := range selected {
		if err := verifyRecord(filepath.Join(bundle, item.Name), record{Name: item.Name, SHA256: item.SHA256, Size: item.Size}); err != nil {
			return errors.New("release asset binding is invalid")
		}
	}
	return nil
}

func activate(ctx context.Context, root, version string, manifestBody []byte, selected []asset, bundle string, system bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	directoryMode, metadataMode, executableMode := os.FileMode(0700), os.FileMode(0600), os.FileMode(0700)
	if system {
		directoryMode, metadataMode, executableMode = 0755, 0444, 0555
	}
	versions := filepath.Join(root, "versions")
	if err := ensureDirectoryMode(versions, directoryMode); err != nil {
		return Result{}, err
	}
	final := filepath.Join(versions, version)
	if _, err := os.Lstat(final); !errors.Is(err, os.ErrNotExist) {
		return Result{}, errors.New("release version already exists")
	}
	var stage string
	var err error
	if system {
		sum := sha256.Sum256(manifestBody)
		name, nameErr := SystemCopyStageName(hex.EncodeToString(sum[:]))
		if nameErr != nil {
			return Result{}, nameErr
		}
		stage = filepath.Join(versions, name)
		err = os.Mkdir(stage, 0700)
	} else {
		stage, err = os.MkdirTemp(versions, ".pending-")
	}
	if err != nil {
		return Result{}, errors.New("create release installation stage")
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(stage)
		}
	}()
	if err := os.Chmod(stage, 0o700); err != nil {
		return Result{}, errors.New("protect release installation stage")
	}
	for _, name := range []string{manifestName, sbomName, runtimeImagesName} {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		if err := copyNew(filepath.Join(stage, name), filepath.Join(bundle, name), metadataMode); err != nil {
			return Result{}, errors.New("stage release metadata")
		}
	}
	for _, item := range selected {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		mode := executableMode
		if item.Role == "release-metadata" || ReleaseFile(item).IsRuntimeData() {
			mode = metadataMode
		}
		if err := copyNew(filepath.Join(stage, item.Name), filepath.Join(bundle, item.Name), mode); err != nil {
			return Result{}, errors.New("stage release asset")
		}
	}
	manifestDigest := sha256.Sum256(manifestBody)
	recordBody, err := installationRecord(version, manifestBody)
	if err != nil || writeNew(filepath.Join(stage, installRecordName), recordBody, metadataMode) != nil {
		return Result{}, errors.New("write release installation record")
	}
	if err := os.Chmod(stage, directoryMode); err != nil {
		return Result{}, errors.New("protect committed release directory")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := syncDirectory(stage); err != nil || os.Rename(stage, final) != nil || syncDirectory(versions) != nil {
		return Result{}, errors.New("commit release installation")
	}
	cleanup = false
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := activateCurrent(root, version); err != nil {
		return Result{}, err
	}
	return Result{Version: version, InstallRoot: root, ActivePath: filepath.Join(root, "current"), ManifestSHA256: hex.EncodeToString(manifestDigest[:])}, nil
}

// SystemCopyStageName binds only a basename to the durable manifest identity.
// It grants no ownership or permission to delete an existing directory.
func SystemCopyStageName(manifestSHA256 string) (string, error) {
	if !digestPattern.MatchString(manifestSHA256) {
		return "", errors.New("invalid system copy identity")
	}
	return ".pending-" + manifestSHA256, nil
}

func installationRecord(version string, manifestBody []byte) ([]byte, error) {
	sum := sha256.Sum256(manifestBody)
	body, err := json.MarshalIndent(struct {
		Version        string `json:"version"`
		ManifestSHA256 string `json:"manifest_sha256"`
	}{version, hex.EncodeToString(sum[:])}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func activateCurrent(root, version string) error {
	return activateCurrentWithSync(root, version, syncDirectory)
}

func activateCurrentWithSync(root, version string, syncFn func(string) error) error {
	if syncFn == nil {
		return errors.New("sync active release pointer")
	}
	current := filepath.Join(root, "current")
	previous := ""
	hadPrevious := false
	if info, err := os.Lstat(current); err == nil {
		if info.Mode()&os.ModeSymlink == 0 || !validCurrentTarget(root, current) {
			return errors.New("existing active release pointer is invalid")
		}
		previous, err = os.Readlink(current)
		if err != nil {
			return errors.New("inspect active release pointer")
		}
		hadPrevious = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect active release pointer")
	}
	nextPath, err := temporaryPointer(root, filepath.Join("versions", version), ".current-")
	if err != nil {
		return errors.New("create active release pointer")
	}
	if err := os.Rename(nextPath, current); err != nil {
		_ = os.Remove(nextPath)
		return errors.New("activate release pointer")
	}
	if err := syncFn(root); err != nil {
		if rollbackActivePointer(root, current, previous, hadPrevious, syncFn) != nil {
			return errors.New("active release pointer state is uncertain")
		}
		return errors.New("sync active release pointer")
	}
	return nil
}

func temporaryPointer(root, target, prefix string) (string, error) {
	next, err := os.CreateTemp(root, prefix)
	if err != nil {
		return "", err
	}
	nextPath := next.Name()
	if err := next.Close(); err != nil || os.Remove(nextPath) != nil || os.Symlink(target, nextPath) != nil {
		_ = os.Remove(nextPath)
		return "", errors.New("create pointer")
	}
	return nextPath, nil
}

func rollbackActivePointer(root, current, previous string, hadPrevious bool, syncFn func(string) error) error {
	if hadPrevious {
		rollback, err := temporaryPointer(root, previous, ".rollback-")
		if err != nil {
			return err
		}
		if err := os.Rename(rollback, current); err != nil {
			_ = os.Remove(rollback)
			return err
		}
	} else if err := os.Remove(current); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return syncFn(root)
}

func validCurrentTarget(root, current string) bool {
	target, err := os.Readlink(current)
	if err != nil || !strings.HasPrefix(target, "versions/") || filepath.IsAbs(target) {
		return false
	}
	resolved := filepath.Clean(filepath.Join(root, target))
	relative, err := filepath.Rel(filepath.Join(root, "versions"), resolved)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func supportedPlatform(goos, goarch string) bool {
	return (goos == "linux" || goos == "darwin") && (goarch == "amd64" || goarch == "arm64")
}

func canonicalAbsolute(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", errors.New("path is not canonical and absolute")
	}
	return path, nil
}

func canonicalDirectory(path string) (string, error) {
	clean, err := canonicalAbsolute(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(clean)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("directory is unavailable")
	}
	return clean, nil
}

func ensureInstallRoot(path string) error { return ensureDirectory(path) }

func ensureDirectory(path string) error {
	return ensureDirectoryMode(path, 0700)
}

func ensureDirectoryMode(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return errors.New("create release installation directory")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || os.Chmod(path, mode) != nil {
		return errors.New("release installation directory is invalid")
	}
	return nil
}

func validRecord(value record, name string) bool {
	return value.Name == name && value.Size > 0 && value.Size <= maxBundleFileSize && digestPattern.MatchString(value.SHA256)
}

func validRuntimeLock(value runtimeLock) bool {
	return digestPattern.MatchString(value.SHA256) && value.GVisorRelease != "" && regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(value.GVisorCommit)
}

func verifyRecord(path string, value record) error {
	body, err := readRegular(path)
	if err != nil || int64(len(body)) != value.Size {
		return errors.New("record differs")
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != value.SHA256 {
		return errors.New("record differs")
	}
	return nil
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maxBundleFileSize {
		return nil, errors.New("file is unavailable")
	}
	return os.ReadFile(path)
}

func copyNew(destination, source string, mode os.FileMode) error {
	body, err := readRegular(source)
	if err != nil {
		return err
	}
	return writeNew(destination, body, mode)
}

func writeNew(path string, body []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	err = directory.Sync()
	closeErr := directory.Close()
	if err != nil {
		return err
	}
	return closeErr
}
