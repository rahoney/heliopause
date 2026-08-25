package releaseinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// Check is a bounded doctor result. Detail is a stable code, never an
// inherited command error or a Host path supplied by an untrusted source.
type Check struct {
	Name    string
	Healthy bool
	Detail  string
}

// DefaultRoot returns the per-user versioned release root. System runtime
// helpers intentionally live outside this unprivileged location.
func DefaultRoot() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" || !filepath.IsAbs(base) {
		return "", errors.New("resolve user release installation root")
	}
	return filepath.Join(base, "heliopause", "releases"), nil
}

// CheckInstallation verifies only the active user installation. It does not
// create directories, repair pointers, download artifacts, or treat absence as
// healthy.
func CheckInstallation(root string) Check {
	return checkInstallation(root, runtime.GOOS, runtime.GOARCH)
}

func checkInstallation(root, goos, goarch string) Check {
	clean, err := canonicalDirectory(root)
	if err != nil {
		return Check{Name: "release-installation", Detail: "RELEASE_INSTALLATION_UNAVAILABLE"}
	}
	current := filepath.Join(clean, "current")
	info, err := os.Lstat(current)
	if err != nil || info.Mode()&os.ModeSymlink == 0 || !validCurrentTarget(clean, current) {
		return Check{Name: "release-installation", Detail: "RELEASE_ACTIVE_POINTER_UNAVAILABLE"}
	}
	target, err := os.Readlink(current)
	if err != nil {
		return Check{Name: "release-installation", Detail: "RELEASE_ACTIVE_POINTER_UNAVAILABLE"}
	}
	versionDirectory := filepath.Join(clean, target)
	if _, err := canonicalDirectory(versionDirectory); err != nil {
		return Check{Name: "release-installation", Detail: "RELEASE_ACTIVE_VERSION_UNAVAILABLE"}
	}
	recordBody, err := readRegular(filepath.Join(versionDirectory, installRecordName))
	if err != nil {
		return Check{Name: "release-installation", Detail: "RELEASE_INSTALL_RECORD_UNAVAILABLE"}
	}
	var installed struct {
		Version        string `json:"version"`
		ManifestSHA256 string `json:"manifest_sha256"`
	}
	if json.Unmarshal(recordBody, &installed) != nil || !versionPattern.MatchString(installed.Version) || !digestPattern.MatchString(installed.ManifestSHA256) || filepath.Base(versionDirectory) != installed.Version {
		return Check{Name: "release-installation", Detail: "RELEASE_INSTALL_RECORD_INVALID"}
	}
	manifest, body, err := readManifest(versionDirectory)
	if err != nil || manifest.Version != installed.Version {
		return Check{Name: "release-installation", Detail: "RELEASE_MANIFEST_UNAVAILABLE"}
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != installed.ManifestSHA256 {
		return Check{Name: "release-installation", Detail: "RELEASE_MANIFEST_MISMATCH"}
	}
	selected, err := selectFiles(manifest, goos, goarch)
	if err != nil || verifyBundleFiles(versionDirectory, manifest, selected) != nil {
		return Check{Name: "release-installation", Detail: "RELEASE_ASSET_MISMATCH"}
	}
	return Check{Name: "release-installation", Healthy: true, Detail: "OK"}
}
