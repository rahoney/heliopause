//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/rahoney/heliopause/internal/releaseinstall"
)

const wireStagedMarker = "STAGED1\n"

var ErrCleanup = errors.New("SELF_UPDATE_OWNED_STAGE_CLEANUP_UNCERTAIN")

type preparedRelease struct {
	mu        sync.Mutex
	directory string
	identity  os.FileInfo
	release   wireRelease
	proof     []byte
	closed    bool
	closeErr  error
}

// prepareWireRelease writes only after actual old-controller authorization and
// independent official proof validation. PREPARED is not activation authority.
func prepareWireRelease(ctx context.Context, reader io.Reader, peer *controllerPeer, current InstalledRelease) (prepared *preparedRelease, resultErr error) {
	if ctx == nil || reader == nil || os.Geteuid() != 0 || peer == nil {
		return nil, ErrCaller
	}
	if peer.uid != current.ClientUID() || peer.gid != current.ClientGID() {
		return nil, ErrCaller
	}
	if err := peer.alive(); err != nil {
		return nil, err
	}
	release, err := readWirePreamble(ctx, reader)
	if err != nil {
		return nil, err
	}
	proof, err := release.verify(ctx, current)
	if err != nil {
		return nil, err
	}
	return stageCheckedWire(ctx, reader, peer, release, proof)
}

// The production entry above is the sole caller after auth and fixed SDK
// verification. The filesystem primitive is separately fault-tested.
func stageCheckedWire(ctx context.Context, reader io.Reader, peer *controllerPeer, release wireRelease, proof []byte) (prepared *preparedRelease, resultErr error) {
	if ctx == nil || reader == nil || peer == nil || os.Geteuid() != 0 {
		return nil, ErrCaller
	}
	if err := peer.alive(); err != nil {
		return nil, err
	}
	if err := ensureStagingDirectory(); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(releaseinstall.SystemStagingRoot, ".prepared-")
	if err != nil {
		return nil, ErrUntrusted
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, errors.Join(ErrUntrusted, ErrCleanup)
	}
	stage := &preparedRelease{directory: directory, identity: info, release: release, proof: proof}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, stage.Close())
		}
	}()
	if err := os.Chmod(directory, 0700); err != nil {
		return nil, ErrUntrusted
	}
	if err := copyStageFile(ctx, stage, "helox-release-manifest.json", release.manifest.SHA256(), int64(len(release.body)), bytes.NewReader(release.body)); err != nil {
		return nil, err
	}
	for _, file := range release.files {
		if err := copyStageFile(ctx, stage, file.Name, file.SHA256, file.Size, reader); err != nil {
			return nil, err
		}
	}
	marker := make([]byte, len(wireStagedMarker))
	if _, err := io.ReadFull(protectedContextReader{ctx, reader}, marker); err != nil {
		return nil, errors.Join(ErrProtocol, err, ctx.Err())
	}
	if !bytes.Equal(marker, []byte(wireStagedMarker)) {
		return nil, ErrProtocol
	}
	sbom, err := readProtected(ctx, filepath.Join(directory, "helox-release-sbom.cdx.json"), 8<<20)
	if err != nil {
		return nil, err
	}
	if err := releaseinstall.ValidateReleaseSBOM(sbom, release.manifest); err != nil {
		return nil, errors.Join(ErrProtocol, err)
	}
	images, err := readProtected(ctx, filepath.Join(directory, "helox-runtime-images.json"), 1<<20)
	if err != nil {
		return nil, err
	}
	if err := releaseinstall.ValidateRuntimeImages(images); err != nil {
		return nil, errors.Join(ErrProtocol, err)
	}
	notices, err := readProtected(ctx, filepath.Join(directory, "pnpm-NOTICE.txt"), 64<<10)
	if err != nil {
		return nil, errors.Join(ErrProtocol, err)
	}
	if err := releaseinstall.ValidatePNPMNotices(notices); err != nil {
		return nil, errors.Join(ErrProtocol, err)
	}
	bunNotices, err := readProtected(ctx, filepath.Join(directory, "bun-NOTICE.txt"), 64<<10)
	if err != nil {
		return nil, errors.Join(ErrProtocol, err)
	}
	if err := releaseinstall.ValidateBunNotices(bunNotices); err != nil {
		return nil, errors.Join(ErrProtocol, err)
	}
	if err := syncProtectedDirectory(directory); err != nil {
		return nil, err
	}
	if err := peer.alive(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stage, nil
}

func ensureStagingDirectory() error {
	if err := protectedDirectory(stateRoot); err != nil {
		return err
	}
	if _, err := os.Lstat(releaseinstall.SystemStagingRoot); err == nil {
		if err := protectedDirectory(releaseinstall.SystemStagingRoot); err != nil {
			return err
		}
		info, err := os.Lstat(releaseinstall.SystemStagingRoot)
		if err != nil || info.Mode().Perm() != 0700 {
			return ErrUntrusted
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrUntrusted
	}
	if err := os.Mkdir(releaseinstall.SystemStagingRoot, 0700); err != nil {
		return ErrUntrusted
	}
	if err := os.Chmod(releaseinstall.SystemStagingRoot, 0700); err != nil {
		return ErrUntrusted
	}
	return protectedDirectory(releaseinstall.SystemStagingRoot)
}

func copyStageFile(ctx context.Context, stage *preparedRelease, name, digest string, size int64, reader io.Reader) (resultErr error) {
	if ctx == nil || stage == nil || reader == nil || filepath.Base(name) != name || size <= 0 || size > 512<<20 || !sha256Text(digest) {
		return ErrProtocol
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(stage.directory, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return ErrUntrusted
	}
	defer func() {
		if file.Close() != nil {
			resultErr = errors.Join(resultErr, ErrUntrusted)
		}
	}()
	hash := sha256.New()
	count, err := io.CopyN(io.MultiWriter(file, hash), protectedContextReader{ctx, reader}, size)
	if err != nil {
		return errors.Join(ErrProtocol, err, ctx.Err())
	}
	if count != size || hex.EncodeToString(hash.Sum(nil)) != digest {
		return ErrUntrusted
	}
	if err := file.Chmod(0600); err != nil {
		return ErrUntrusted
	}
	if err := file.Sync(); err != nil {
		return ErrUntrusted
	}
	return ctx.Err()
}

// writeWirePayloads opens only names selected from the strict manifest. The
// Promotion adapter must bind open to its existing sealed staging membership.
func writeWirePayloads(ctx context.Context, writer io.Writer, release wireRelease, open func(string) (io.ReadCloser, error)) error {
	if ctx == nil || writer == nil || open == nil {
		return ErrProtocol
	}
	checked, err := newWireRelease(release.body, release.proofs)
	if err != nil {
		return err
	}
	for _, file := range checked.files {
		if err := ctx.Err(); err != nil {
			return err
		}
		input, err := open(file.Name)
		if err != nil {
			return err
		}
		if input == nil {
			return ErrProtocol
		}
		hash := sha256.New()
		count, copyErr := io.Copy(io.MultiWriter(writer, hash), io.LimitReader(protectedContextReader{ctx, input}, file.Size+1))
		closeErr := input.Close()
		if copyErr != nil || closeErr != nil {
			return errors.Join(ErrProtocol, copyErr, closeErr, ctx.Err())
		}
		if count != file.Size || hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
			return ErrUntrusted
		}
	}
	return writeWireBytes(ctx, writer, []byte(wireStagedMarker))
}

func syncProtectedDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return ErrUntrusted
	}
	syncErr, closeErr := directory.Sync(), directory.Close()
	if syncErr != nil || closeErr != nil {
		return ErrUntrusted
	}
	return nil
}

func (p *preparedRelease) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return p.closeErr
	}
	p.closed = true
	info, err := os.Lstat(p.directory)
	if err != nil || !os.SameFile(info, p.identity) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		p.closeErr = ErrCleanup
		return p.closeErr
	}
	if err := os.RemoveAll(p.directory); err != nil {
		p.closeErr = ErrCleanup
		return p.closeErr
	}
	if err := syncProtectedDirectory(releaseinstall.SystemStagingRoot); err != nil {
		p.closeErr = errors.Join(ErrCleanup, err)
	}
	return p.closeErr
}
