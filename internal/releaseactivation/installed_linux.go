//go:build linux

package releaseactivation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// InstalledRelease contains independently checked installation facts, not a
// Policy decision. The caller must hold the runtime lease while using them.
type InstalledRelease struct {
	record   registration
	manifest releaseinstall.ReleaseManifest
	proof    []byte
}

func (r InstalledRelease) Version() string                          { return r.record.Version }
func (r InstalledRelease) ClientUID() uint32                        { return r.record.ClientUID }
func (r InstalledRelease) ClientGID() uint32                        { return r.record.ClientGID }
func (r InstalledRelease) Manifest() releaseinstall.ReleaseManifest { return r.manifest }
func (r InstalledRelease) Proof() []byte                            { return append([]byte(nil), r.proof...) }
func (r InstalledRelease) ClientPath() string {
	if r.record.Schema != registrationSchema {
		return ""
	}
	return filepath.Join(releaseinstall.SystemInstallRoot, "versions", r.record.Version, "helox-linux-amd64")
}

// ReadInstalledRelease validates the fixed registration, current pointer,
// signed manifest/proof and every selected byte identity without networking.
func ReadInstalledRelease(ctx context.Context) (InstalledRelease, error) {
	if ctx == nil {
		return InstalledRelease{}, ErrUntrusted
	}
	body, err := readProtected(ctx, registrationPath, 16<<10)
	if err != nil {
		return InstalledRelease{}, err
	}
	record, err := parseRegistration(body)
	if err != nil {
		return InstalledRelease{}, err
	}
	if err := protectedDirectory(releaseinstall.SystemInstallRoot); err != nil {
		return InstalledRelease{}, err
	}
	current := filepath.Join(releaseinstall.SystemInstallRoot, "current")
	info, err := os.Lstat(current)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return InstalledRelease{}, ErrUntrusted
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 {
		return InstalledRelease{}, ErrUntrusted
	}
	target, err := os.Readlink(current)
	if err != nil || target != filepath.Join("versions", record.Version) {
		return InstalledRelease{}, ErrUntrusted
	}
	installed, err := readRecordedRelease(ctx, record)
	if err != nil {
		return InstalledRelease{}, err
	}
	after, err := os.Lstat(current)
	targetAfter, linkErr := os.Readlink(current)
	if err != nil || linkErr != nil || !os.SameFile(info, after) || targetAfter != target {
		return InstalledRelease{}, ErrUntrusted
	}
	return installed, nil
}

// readRecordedRelease shares the exact installed-byte/proof checks without
// requiring the active pointer. Recovery must authenticate the retained old
// version before changing current. Records come only from protected state.
func readRecordedRelease(ctx context.Context, record registration) (InstalledRelease, error) {
	if ctx == nil {
		return InstalledRelease{}, ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return InstalledRelease{}, err
	}
	body, err := json.Marshal(record)
	if err != nil {
		return InstalledRelease{}, ErrUntrusted
	}
	if _, err := parseRegistration(body); err != nil {
		return InstalledRelease{}, err
	}
	directory := filepath.Join(releaseinstall.SystemInstallRoot, "versions", record.Version)
	if err := protectedDirectory(directory); err != nil {
		return InstalledRelease{}, err
	}
	manifestBody, err := readProtected(ctx, filepath.Join(directory, "helox-release-manifest.json"), 1<<20)
	if err != nil {
		return InstalledRelease{}, err
	}
	manifest, err := releaseinstall.ParseReleaseManifest(manifestBody)
	if err != nil || manifest.Version() != record.Version || manifest.SHA256() != record.ManifestSHA256 || manifest.GVisorRelease() != runtimeidentity.GVisorRelease || manifest.GVisorCommit() != runtimeidentity.GVisorCommit {
		return InstalledRelease{}, ErrUntrusted
	}
	native, err := manifest.NativeUpdateFiles()
	if err != nil {
		return InstalledRelease{}, errors.Join(ErrUntrusted, err)
	}
	selected := append([]releaseinstall.ReleaseFile(nil), native...)
	for _, file := range manifest.Files() {
		if file.Role == "release-metadata" {
			selected = append(selected, file)
		}
	}
	subjects := []releaseprovenance.Subject{{Name: "helox-release-manifest.json", SHA256: manifest.SHA256()}}
	var total int64 = int64(len(manifestBody))
	for _, file := range selected {
		mode := os.FileMode(0555)
		if file.Role == "release-metadata" {
			mode = 0444
		}
		if file.Size <= 0 || file.Size > 512<<20 || file.Size > (1<<30)-total {
			return InstalledRelease{}, ErrUntrusted
		}
		total += file.Size
		if err := hashProtected(ctx, filepath.Join(directory, file.Name), file.SHA256, file.Size, mode); err != nil {
			return InstalledRelease{}, err
		}
		subjects = append(subjects, releaseprovenance.Subject{Name: file.Name, SHA256: file.SHA256})
	}
	if err := hashProtected(ctx, filepath.Join(directory, "helox-release-manifest.json"), manifest.SHA256(), int64(len(manifestBody)), 0444); err != nil {
		return InstalledRelease{}, err
	}
	sbom, err := readProtected(ctx, filepath.Join(directory, "helox-release-sbom.cdx.json"), 8<<20)
	if err != nil {
		return InstalledRelease{}, err
	}
	if err := releaseinstall.ValidateReleaseSBOM(sbom, manifest); err != nil {
		return InstalledRelease{}, errors.Join(ErrUntrusted, err)
	}
	images, err := readProtected(ctx, filepath.Join(directory, "helox-runtime-images.json"), 1<<20)
	if err != nil {
		return InstalledRelease{}, err
	}
	if err := releaseinstall.ValidateRuntimeImages(images); err != nil {
		return InstalledRelease{}, errors.Join(ErrUntrusted, err)
	}
	notices, err := readProtected(ctx, filepath.Join(directory, "pnpm-NOTICE.txt"), 64<<10)
	if err != nil {
		return InstalledRelease{}, errors.Join(ErrUntrusted, err)
	}
	if err := releaseinstall.ValidatePNPMNotices(notices); err != nil {
		return InstalledRelease{}, errors.Join(ErrUntrusted, err)
	}
	bunNotices, err := readProtected(ctx, filepath.Join(directory, "bun-NOTICE.txt"), 64<<10)
	if err != nil {
		return InstalledRelease{}, errors.Join(ErrUntrusted, err)
	}
	if err := releaseinstall.ValidateBunNotices(bunNotices); err != nil {
		return InstalledRelease{}, errors.Join(ErrUntrusted, err)
	}
	proofs, err := readStoredProofs(ctx, manifest.SHA256())
	if err != nil {
		return InstalledRelease{}, err
	}
	expected := releaseprovenance.Identity{Version: manifest.Version(), SourceCommit: manifest.SourceCommit(), WorkflowRun: manifest.WorkflowRun()}
	for _, proof := range proofs {
		if _, err := releaseprovenance.Verify(ctx, proof, expected, subjects); err == nil {
			return InstalledRelease{record, manifest, append([]byte(nil), proof...)}, nil
		}
		if err := ctx.Err(); err != nil {
			return InstalledRelease{}, err
		}
	}
	return InstalledRelease{}, ErrUntrusted
}

func readStoredProofs(ctx context.Context, digest string) ([][]byte, error) {
	if !sha256Text(digest) {
		return nil, ErrUntrusted
	}
	directory := filepath.Join(proofRoot, digest)
	if err := protectedDirectory(directory); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) == 0 || len(entries) > 16 {
		return nil, ErrUntrusted
	}
	proofs := make([][]byte, 0, len(entries))
	for _, entry := range entries {
		name, found := strings.CutSuffix(entry.Name(), ".json")
		if !found || !sha256Text(name) || entry.IsDir() {
			return nil, ErrUntrusted
		}
		body, err := readProtected(ctx, filepath.Join(directory, entry.Name()), 1<<20)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != name {
			return nil, ErrUntrusted
		}
		proofs = append(proofs, body)
	}
	return proofs, nil
}

func hashProtected(ctx context.Context, path, digest string, size int64, mode os.FileMode) (resultErr error) {
	file, info, err := openProtectedFile(path)
	if err != nil {
		return err
	}
	defer func() {
		if file.Close() != nil {
			resultErr = errors.Join(resultErr, ErrUntrusted)
		}
	}()
	if info.Size() != size || info.Mode().Perm() != mode {
		return ErrUntrusted
	}
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(protectedContextReader{ctx, file}, size+1))
	if err != nil {
		return errors.Join(ErrUntrusted, ctx.Err())
	}
	if count != size || hex.EncodeToString(hash.Sum(nil)) != digest {
		return ErrUntrusted
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, current) {
		return ErrUntrusted
	}
	return ctx.Err()
}
