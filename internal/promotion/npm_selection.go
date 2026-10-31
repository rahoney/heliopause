package promotion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// applyNPMSelectionControls runs only after the local-tarball recipe and exact
// offline output have been validated. Selection bytes are covered by Manifest
// identity; they never replace the private recipe before installation.
func applyNPMSelectionControls(workspace string, bundle domain.VerifiedBundle) error {
	var manifest struct {
		Resolver struct {
			LockfileSHA256 string `json:"lockfile_sha256"`
		} `json:"resolver"`
		Controls []struct {
			Name    string `json:"name"`
			SHA256  string `json:"sha256"`
			Content []byte `json:"content"`
		} `json:"project_controls"`
	}
	if err := verifyDocuments(bundle); err != nil {
		return err
	}
	if json.Unmarshal(bundle.ManifestDocument(), &manifest) != nil {
		return errors.New("npm selection Manifest is invalid")
	}
	if len(manifest.Controls) == 0 {
		return nil
	} // Historical bundles remain valid.
	if len(manifest.Controls) != 2 {
		return errors.New("npm selection requires exactly two controls")
	}
	controls := map[string][]byte{}
	for _, control := range manifest.Controls {
		if control.Name != "package.json" && control.Name != "package-lock.json" || len(control.Content) == 0 || len(control.Content) > artifactnpm.ProjectControlLimit || controls[control.Name] != nil {
			return errors.New("npm selection control is unsupported")
		}
		sum := sha256.Sum256(control.Content)
		if hex.EncodeToString(sum[:]) != control.SHA256 {
			return errors.New("npm selection control digest differs from Manifest")
		}
		controls[control.Name] = control.Content
	}
	lockDigest := sha256.Sum256(controls["package-lock.json"])
	if hex.EncodeToString(lockDigest[:]) != manifest.Resolver.LockfileSHA256 {
		return errors.New("npm selection lock differs from resolved input")
	}
	reference, err := artifactnpm.ProjectRootReference(controls["package.json"])
	if err != nil {
		return err
	}
	graph, err := artifactnpm.ParseProjectControls(reference, controls["package.json"], controls["package-lock.json"])
	if err != nil || !artifactnpm.SameSelectionGraph(graph, bundle.Set().Inspected().Graph()) {
		return errors.New("npm selected graph differs from the inspected set")
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return errors.New("open npm private selection workspace")
	}
	defer root.Close()
	for _, name := range []string{"package.json", "package-lock.json"} {
		before, err := root.Lstat(name)
		if err != nil || !before.Mode().IsRegular() || !pypiSingleLink(before) {
			return errors.New("npm private recipe control is untrusted")
		}
		file, err := root.OpenFile(name, os.O_WRONLY, 0)
		if err != nil {
			return errors.New("open npm private recipe control")
		}
		now, statErr := file.Stat()
		if statErr != nil || !os.SameFile(before, now) || before.Mode() != now.Mode() || !pypiSingleLink(now) {
			_ = file.Close()
			return errors.New("npm private recipe control changed")
		}
		writeErr := file.Truncate(0)
		if writeErr == nil {
			_, writeErr = file.Write(controls[name])
		}
		if writeErr == nil {
			writeErr = file.Sync()
		}
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return errors.New("write npm selected public control")
		}
		actual, err := os.ReadFile(filepath.Join(workspace, name))
		if err != nil || !equalBytes(actual, controls[name]) {
			return errors.New("npm selected control publication differs from Manifest")
		}
	}
	return nil
}
