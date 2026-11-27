//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// The journal reserved this exact basename while it was absent, before any
// system copier ran. Never glob anonymous pending directories or adopt them.
func (a *rootActivation) discardOwnedCopy(ctx context.Context) error {
	if ctx == nil || a == nil || a.journal == nil || a.checks.recorded == nil {
		return ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.journal.value.validate(); err != nil {
		return err
	}
	name, err := releaseinstall.SystemCopyStageName(a.journal.value.ManifestSHA256)
	if err != nil {
		return ErrUntrusted
	}
	versions := filepath.Join(releaseinstall.SystemInstallRoot, "versions")
	if err := protectedDirectory(versions); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(versions, name)); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return errors.Join(ErrCleanup, err)
	}
	if a.journal.value.State == "PREPARED" {
		return ErrCleanup
	}
	old, err := a.checks.recorded(ctx, a.journal.value.Old)
	if err != nil || old.record != a.journal.value.Old {
		return errors.Join(ErrCleanup, ErrUntrusted, err)
	}
	var bundle string
	var proof []byte
	if candidate, err := a.checks.recorded(ctx, candidateRegistration(a.journal.value)); err == nil && candidate.record == candidateRegistration(a.journal.value) {
		bundle, proof = filepath.Dir(candidate.ClientPath()), candidate.proof
	} else if a.stage != nil && !a.stage.closed && a.stage.release.manifest.SHA256() == a.journal.value.ManifestSHA256 {
		bundle, proof = a.stage.directory, a.stage.proof
	} else {
		return ErrCleanup
	}
	installer, err := releaseinstall.NewSystemInstaller(protectedBundleVerifier{old, proof})
	if err != nil {
		return errors.Join(ErrCleanup, err)
	}
	if err := installer.DiscardSystemCopy(ctx, releaseinstall.Request{BundleDirectory: bundle, InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
		return errors.Join(ErrCleanup, err)
	}
	return nil
}
