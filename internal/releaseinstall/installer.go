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
)

const (
	manifestName      = "helox-release-manifest.json"
	runtimeImagesName = "helox-runtime-images.json"
	sbomName          = "helox-release-sbom.cdx.json"
	installRecordName = "helox-install.json"
	manifestSchema    = "helox.release-manifest/v1"
	maxBundleFileSize = 1 << 30
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

// Request identifies the already-attested bundle and the user-owned install
// root. Both paths must be canonical absolute paths.
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
}

// New creates an installer for the running platform.
func New(verifier BundleVerifier) (*Installer, error) {
	return newInstaller(verifier, runtime.GOOS, runtime.GOARCH)
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
	if i == nil || i.verifier == nil || ctx == nil || ctx.Err() != nil {
		return Result{}, errors.New("verified release installation is unavailable")
	}
	bundle, err := canonicalDirectory(request.BundleDirectory)
	if err != nil {
		return Result{}, errors.New("release bundle directory is invalid")
	}
	root, err := canonicalAbsolute(request.InstallRoot)
	if err != nil {
		return Result{}, errors.New("release install root is invalid")
	}
	if err := i.verifier.Verify(ctx, bundle); err != nil {
		return Result{}, errors.New("release attestation verification failed")
	}
	manifest, manifestBody, err := readManifest(bundle)
	if err != nil {
		return Result{}, err
	}
	selected, err := selectFiles(manifest, i.goos, i.goarch)
	if err != nil {
		return Result{}, err
	}
	if err := verifyBundleFiles(bundle, manifest, selected); err != nil {
		return Result{}, err
	}
	if err := ensureInstallRoot(root); err != nil {
		return Result{}, err
	}
	return activate(root, manifest.Version, manifestBody, selected, bundle)
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
	var value manifest
	if json.Unmarshal(body, &value) != nil || value.Schema != manifestSchema || !versionPattern.MatchString(value.Version) || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(value.SourceCommit) || value.WorkflowRun == "" || len(value.Assets) == 0 {
		return manifest{}, nil, errors.New("release manifest is invalid")
	}
	if !validRuntimeLock(value.RuntimeLock) || !validRecord(value.SBOM, sbomName) || !validRecord(value.RuntimeImages, runtimeImagesName) {
		return manifest{}, nil, errors.New("release manifest binding is invalid")
	}
	return value, body, nil
}

func selectFiles(value manifest, goos, goarch string) ([]asset, error) {
	want := map[string]string{"helox-" + goos + "-" + goarch: "helox"}
	if goos == "linux" && goarch == "amd64" {
		want["haa_gvisor_observer-linux-amd64"] = "gvisor-observer"
		want["haa-network-policy-helper-linux-amd64"] = "network-policy-helper"
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

func activate(root, version string, manifestBody []byte, selected []asset, bundle string) (Result, error) {
	versions := filepath.Join(root, "versions")
	if err := ensureDirectory(versions); err != nil {
		return Result{}, err
	}
	final := filepath.Join(versions, version)
	if _, err := os.Lstat(final); !errors.Is(err, os.ErrNotExist) {
		return Result{}, errors.New("release version already exists")
	}
	stage, err := os.MkdirTemp(versions, ".pending-")
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
		if err := copyNew(filepath.Join(stage, name), filepath.Join(bundle, name), 0o600); err != nil {
			return Result{}, errors.New("stage release metadata")
		}
	}
	for _, item := range selected {
		if err := copyNew(filepath.Join(stage, item.Name), filepath.Join(bundle, item.Name), 0o700); err != nil {
			return Result{}, errors.New("stage release asset")
		}
	}
	manifestDigest := sha256.Sum256(manifestBody)
	recordBody, err := json.MarshalIndent(struct {
		Version        string `json:"version"`
		ManifestSHA256 string `json:"manifest_sha256"`
	}{Version: version, ManifestSHA256: hex.EncodeToString(manifestDigest[:])}, "", "  ")
	if err != nil || writeNew(filepath.Join(stage, installRecordName), append(recordBody, '\n'), 0o600) != nil {
		return Result{}, errors.New("write release installation record")
	}
	if err := syncDirectory(stage); err != nil || os.Rename(stage, final) != nil || syncDirectory(versions) != nil {
		return Result{}, errors.New("commit release installation")
	}
	cleanup = false
	if err := activateCurrent(root, version); err != nil {
		return Result{}, err
	}
	return Result{Version: version, InstallRoot: root, ActivePath: filepath.Join(root, "current"), ManifestSHA256: hex.EncodeToString(manifestDigest[:])}, nil
}

func activateCurrent(root, version string) error {
	current := filepath.Join(root, "current")
	if info, err := os.Lstat(current); err == nil {
		if info.Mode()&os.ModeSymlink == 0 || !validCurrentTarget(root, current) {
			return errors.New("existing active release pointer is invalid")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("inspect active release pointer")
	}
	next, err := os.CreateTemp(root, ".current-")
	if err != nil {
		return errors.New("create active release pointer")
	}
	nextPath := next.Name()
	if err := next.Close(); err != nil || os.Remove(nextPath) != nil || os.Symlink(filepath.Join("versions", version), nextPath) != nil {
		return errors.New("create active release pointer")
	}
	if err := os.Rename(nextPath, current); err != nil {
		_ = os.Remove(nextPath)
		return errors.New("activate release pointer")
	}
	if err := syncDirectory(root); err != nil {
		return errors.New("sync active release pointer")
	}
	return nil
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
	if err := os.MkdirAll(path, 0o700); err != nil {
		return errors.New("create release installation directory")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || os.Chmod(path, 0o700) != nil {
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
