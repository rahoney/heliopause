//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

func storeReleaseProof(ctx context.Context, release wireRelease, current InstalledRelease) error {
	if ctx == nil || os.Geteuid() != 0 {
		return ErrCaller
	}
	proof, err := release.verify(ctx, current)
	if err != nil {
		return err
	}
	return storeCheckedProof(ctx, release.manifest.SHA256(), proof)
}

// The production caller above first uses the fixed SDK against all nine
// subjects. This storage primitive grants no provenance or Policy authority.
func storeCheckedProof(ctx context.Context, digest string, proof []byte) error {
	if ctx == nil || os.Geteuid() != 0 || !sha256Text(digest) || len(proof) == 0 || len(proof) > 1<<20 {
		return ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, directory := range []string{proofRoot, filepath.Join(proofRoot, digest)} {
		if err := protectedDirectory(filepath.Dir(directory)); err != nil {
			return err
		}
		if _, err := os.Lstat(directory); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(directory, 0755); err != nil {
				return ErrUntrusted
			}
			if err := os.Chmod(directory, 0755); err != nil {
				return ErrUntrusted
			}
			if err := syncProtectedDirectory(filepath.Dir(directory)); err != nil {
				return err
			}
		} else if err != nil {
			return ErrUntrusted
		}
		if err := protectedDirectory(directory); err != nil {
			return err
		}
		identity, err := os.Lstat(directory)
		if err != nil || identity.Mode().Perm() != 0755 {
			return ErrUntrusted
		}
	}
	directory := filepath.Join(proofRoot, digest)
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) > 16 {
		return ErrUntrusted
	}
	if len(entries) > 0 {
		if _, err := readStoredProofs(ctx, digest); err != nil {
			return err
		}
	}
	path := filepath.Join(directory, bodySHA256(proof)+".json")
	if _, err := os.Lstat(path); err == nil {
		body, err := readProtected(ctx, path, 1<<20)
		if err != nil || !bytes.Equal(body, proof) {
			return errors.Join(ErrUntrusted, err)
		}
		identity, err := os.Lstat(path)
		if err != nil || identity.Mode().Perm()&0444 != 0444 {
			return ErrUntrusted
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrUntrusted
	}
	if len(entries) >= 16 {
		return ErrUntrusted
	}
	return replaceProtected(ctx, path, nil, proof, 0444, 0)
}

// protectedBundleVerifier remains mandatory when calling the existing root
// installer. No mutable caller receipt or configurable signing root is used.
type protectedBundleVerifier struct {
	current InstalledRelease
	proof   []byte
}

func (v protectedBundleVerifier) Verify(ctx context.Context, directory string) error {
	if ctx == nil || os.Geteuid() != 0 {
		return ErrCaller
	}
	if filepath.Dir(directory) != releaseinstall.SystemStagingRoot && filepath.Dir(directory) != filepath.Join(releaseinstall.SystemInstallRoot, "versions") {
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
	if _, err := release.verify(ctx, v.current); err != nil {
		return err
	}
	if filepath.Dir(directory) == filepath.Join(releaseinstall.SystemInstallRoot, "versions") && filepath.Base(directory) != release.manifest.Version() {
		return ErrUntrusted
	}
	return nil
}

func candidateRegistration(journal updateJournal) registration {
	record := journal.Old
	record.Version = journal.Version
	record.ManifestSHA256 = journal.ManifestSHA256
	return record
}

func switchRegistration(ctx context.Context, journal updateJournal, restore bool) error {
	if err := journal.validate(); err != nil {
		return err
	}
	old, err := json.Marshal(journal.Old)
	if err != nil {
		return ErrUntrusted
	}
	next, err := json.Marshal(candidateRegistration(journal))
	if err != nil {
		return ErrUntrusted
	}
	actual, err := readProtected(ctx, registrationPath, 16<<10)
	if err != nil {
		return err
	}
	if restore {
		if bytes.Equal(actual, old) {
			return nil
		}
		if !bytes.Equal(actual, next) {
			return ErrUntrusted
		}
		old, next = next, old
	}
	return replaceProtected(ctx, registrationPath, old, next, 0644, 0)
}
