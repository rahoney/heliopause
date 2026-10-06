package promotion

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Cache bytes are streamed; the 512 MiB payload bound is not a memory allocation.
const maxGoBuildCacheArchiveBytes = maxGoProjectCacheBytes + (4*maxGoProjectCacheFiles+2)*512

// SnapshotBuildCache copies only the inventory authenticated by the independent
// retained cache receipt. New inventory or a cache path alone cannot authorize
// copied bytes. Every consumed file is rehashed against that fixed inventory.
func (g *approvedGoProjectGuard) SnapshotBuildCache(ctx context.Context) (result domain.AcquiredArtifact, resultErr error) {
	if g == nil || g.buildCache != nil {
		return result, errors.New("go build cache is unavailable or already frozen")
	}
	defer func() {
		if resultErr == nil && result.ContentHandle() != "" {
			frozen := result
			g.buildCache = &frozen
		}
	}()
	snapshot, cachePath, err := g.OpenBuildInputs(ctx)
	if err != nil {
		return result, err
	}
	var approval goProjectApproval
	if json.Unmarshal(g.originalState, &approval) != nil {
		return result, errors.New("go build cache approval is invalid")
	}
	cacheParent, err := os.OpenRoot(filepath.Dir(cachePath))
	if err != nil {
		return result, errors.New("open approved Go cache root")
	}
	defer func() {
		resultErr = errors.Join(resultErr, cacheParent.Close())
		if resultErr != nil {
			result = domain.AcquiredArtifact{}
		}
	}()
	receiptInfo, err := cacheParent.Lstat(goCacheReceipt)
	if err != nil || !receiptInfo.Mode().IsRegular() || !pypiSingleLink(receiptInfo) || receiptInfo.Size() <= 0 || receiptInfo.Size() > 4<<20 {
		return result, errors.New("go build cache receipt is not bounded regular data")
	}
	receipt, err := cacheParent.OpenFile(goCacheReceipt, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return result, errors.New("open Go build cache receipt")
	}
	opened, statErr := receipt.Stat()
	if statErr != nil || !os.SameFile(receiptInfo, opened) || !pypiSingleLink(opened) {
		_ = receipt.Close()
		return result, errors.New("go build cache receipt identity changed")
	}
	body, readErr := io.ReadAll(io.LimitReader(receipt, receiptInfo.Size()+1))
	closeErr := receipt.Close()
	hash := sha256.Sum256(body)
	if readErr != nil || closeErr != nil || int64(len(body)) != receiptInfo.Size() || hex.EncodeToString(hash[:]) != approval.CacheDigest {
		return result, errors.New("go build cache receipt differs from retained approval")
	}
	var doc goCacheDocument
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&doc) != nil {
		return result, errors.New("go build cache receipt is invalid")
	}
	expected := approval.Approval
	expected.Files = doc.Files
	canonical, err := json.Marshal(expected)
	if err != nil || !bytes.Equal(body, canonical) || len(doc.Files) == 0 || len(doc.Files) > 2*maxGoProjectCacheFiles {
		return result, errors.New("go build cache inventory or binding is invalid")
	}
	rootInfo, err := cacheParent.Lstat("modcache")
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode() != os.ModeDir|0o755 {
		return result, errors.New("go build cache root is invalid")
	}
	cache, err := cacheParent.OpenRoot("modcache")
	if err != nil {
		return result, errors.New("open anchored Go build cache")
	}
	roots := []*os.Root{cache}
	defer func() {
		for i := len(roots) - 1; i >= 0; i-- {
			resultErr = errors.Join(resultErr, roots[i].Close())
		}
		if resultErr != nil {
			result = domain.AcquiredArtifact{}
		}
	}()
	openedRoot, err := cache.Stat(".")
	if err != nil || !os.SameFile(rootInfo, openedRoot) {
		return result, errors.New("go build cache root identity changed")
	}

	if err := ensureTrustedRoot(g.owner.cache.intakeRoot); err != nil {
		return result, err
	}
	intakeInfo, err := os.Lstat(g.owner.cache.intakeRoot)
	if err != nil {
		return result, errors.New("go build cache intake unavailable")
	}
	intake, err := os.OpenRoot(g.owner.cache.intakeRoot)
	if err != nil {
		return result, errors.New("open Go build cache intake")
	}
	defer func() {
		resultErr = errors.Join(resultErr, intake.Close())
		if resultErr != nil {
			result = domain.AcquiredArtifact{}
		}
	}()
	openedIntake, err := intake.Stat(".")
	if err != nil || !os.SameFile(intakeInfo, openedIntake) {
		return result, errors.New("go build cache intake identity changed")
	}
	run, err := domain.NewRunID()
	if err != nil {
		return result, err
	}
	if err := intake.Mkdir(run.String(), 0o700); err != nil {
		return result, errors.New("create Go build cache intake")
	}
	ownedInfo, err := intake.Lstat(run.String())
	if err != nil || !ownedInfo.IsDir() {
		return result, errors.New("go build cache intake identity unavailable")
	}
	var owned *os.Root
	defer func() {
		if owned != nil {
			resultErr = errors.Join(resultErr, owned.Close())
		}
		if resultErr != nil {
			current, err := intake.Lstat(run.String())
			if err != nil || !current.IsDir() || !os.SameFile(ownedInfo, current) {
				resultErr = errors.Join(resultErr, errors.New("go build cache cleanup identity unavailable"))
			} else {
				resultErr = errors.Join(resultErr, intake.RemoveAll(run.String()))
			}
			result = domain.AcquiredArtifact{}
		}
	}()
	owned, err = intake.OpenRoot(run.String())
	if err != nil {
		return result, errors.New("open Go build cache intake")
	}
	openedOwned, err := owned.Stat(".")
	if err != nil || !os.SameFile(ownedInfo, openedOwned) {
		return result, errors.New("go build cache intake identity changed")
	}
	file, err := owned.OpenFile("go-cache.tar", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return result, errors.New("create Go build cache archive")
	}
	archiveHash := sha256.New()
	bounded := &goBuildCacheArchiveWriter{writer: io.MultiWriter(file, archiveHash), remaining: maxGoBuildCacheArchiveBytes}
	writer := tar.NewWriter(bounded)
	copyErr := copyApprovedGoBuildCache(ctx, cache, doc.Files, writer, &roots)
	tarErr := writer.Close()
	syncErr := file.Sync()
	fileErr := file.Close()
	if err := errors.Join(copyErr, tarErr, syncErr, fileErr); err != nil {
		return result, err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return result, err
	}
	currentSnapshot, currentPath, err := g.OpenBuildInputs(ctx)
	if err != nil || currentPath != cachePath || currentSnapshot.GraphDigest() != snapshot.GraphDigest() {
		return result, errors.New("go build cache changed during snapshot")
	}
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(archiveHash.Sum(nil)))
	if err != nil {
		return result, err
	}
	source, err := domain.NewSourceID("project-local")
	if err != nil {
		return result, err
	}
	identity, err := domain.NewResolvedArtifactIdentity(source, "cache-"+approval.Approval.Project, snapshot.GraphDigest().String(), "go-cache")
	if err != nil {
		return result, err
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":go-cache", uint64(maxGoBuildCacheArchiveBytes-bounded.remaining), "sha256:"+digest.String())
}

type goBuildCacheArchiveWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *goBuildCacheArchiveWriter) Write(body []byte) (int, error) {
	if int64(len(body)) > w.remaining {
		return 0, errors.New("go build cache archive exceeds bound")
	}
	n, err := w.writer.Write(body)
	w.remaining -= int64(n)
	return n, err
}

func copyApprovedGoBuildCache(ctx context.Context, root *os.Root, inventory []goCacheFile, writer *tar.Writer, roots *[]*os.Root) error {
	return copyApprovedProjectBuildCache(ctx, root, inventory, writer, roots, maxGoProjectCacheFiles, maxGoProjectCacheBytes)
}

func copyApprovedProjectBuildCache(ctx context.Context, root *os.Root, inventory []cacheFileRecord, writer *tar.Writer, roots *[]*os.Root, maxFiles int, maxBytes int64) error {
	if maxFiles <= 0 || maxBytes <= 0 || len(inventory) == 0 || len(inventory) > 2*maxFiles {
		return errors.New("project build cache inventory exceeds bounds")
	}
	directories := map[string]*os.Root{".": root}
	seen := map[string]bool{}
	var total int64
	fileCount := 0
	for i, entry := range inventory {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := entry.Path
		if name == "." {
			if i != 0 || entry.Kind != "directory" || entry.Size != 0 || entry.SHA256 != "" {
				return errors.New("go build cache inventory root is invalid")
			}
			seen[name] = true
			continue
		}
		if name == "" || len(name) > 1024 || path.Clean(name) != name || strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\\x00:\r\n") || seen[name] {
			return errors.New("go build cache inventory member is invalid or repeated")
		}
		parent, ok := directories[path.Dir(name)]
		if !ok {
			return errors.New("go build cache inventory parent is unavailable")
		}
		leaf := path.Base(name)
		info, err := parent.Lstat(leaf)
		if err != nil {
			return errors.New("go build cache member identity unavailable")
		}
		seen[name] = true
		if entry.Kind == "directory" {
			if !info.IsDir() || info.Mode() != os.ModeDir|0o755 || entry.Size != 0 || entry.SHA256 != "" {
				return errors.New("go build cache directory is substituted")
			}
			child, err := parent.OpenRoot(leaf)
			if err != nil {
				return errors.New("open anchored Go build cache directory")
			}
			*roots = append(*roots, child)
			opened, err := child.Stat(".")
			if err != nil || !os.SameFile(info, opened) {
				return errors.New("go build cache directory identity changed")
			}
			directories[name] = child
			if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o700}); err != nil {
				return err
			}
			continue
		}
		fileCount++
		if entry.Kind != "file" || !info.Mode().IsRegular() || info.Mode() != 0o444 || !pypiSingleLink(info) || entry.Size < 0 || info.Size() != entry.Size || entry.Size > 64<<20 || total+entry.Size > maxBytes || fileCount > maxFiles {
			return errors.New("go build cache file content or identity exceeds bounds")
		}
		digest, err := domain.NewSHA256Digest(entry.SHA256)
		if err != nil {
			return errors.New("go build cache file digest is invalid")
		}
		file, err := parent.OpenFile(leaf, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return errors.New("open anchored Go build cache file")
		}
		opened, err := file.Stat()
		if err != nil || !opened.Mode().IsRegular() || opened.Mode() != info.Mode() || !pypiSingleLink(opened) || !os.SameFile(info, opened) {
			_ = file.Close()
			return errors.New("go build cache file identity changed")
		}
		headerErr := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o400, Size: entry.Size})
		hash := sha256.New()
		n, copyErr := io.Copy(io.MultiWriter(writer, hash), io.LimitReader(file, entry.Size+1))
		after, statErr := file.Stat()
		closeErr := file.Close()
		if headerErr != nil || copyErr != nil || statErr != nil || closeErr != nil || n != entry.Size || after.Size() != entry.Size || after.Mode() != info.Mode() || !after.ModTime().Equal(info.ModTime()) || !pypiSingleLink(after) || hex.EncodeToString(hash.Sum(nil)) != digest.String() {
			return errors.New("go build cache consumed bytes differ from approved inventory")
		}
		total += n
	}
	if !seen["."] {
		return errors.New("go build cache root inventory is missing")
	}
	return nil
}
