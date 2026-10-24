package npm

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// InspectArchive reuses static inspection and additionally measures every file
// and the complete compressed source. A finding or error yields no inventory.
// The original Inspect API and its result semantics remain unchanged.
func (i *Inspector) InspectArchive(ctx context.Context, artifact domain.AcquiredArtifact) (artifactnpm.ArchiveInventory, domain.InspectionReport, error) {
	capture := &archiveCapture{}
	report, err := i.inspect(ctx, artifact, capture)
	if err != nil || len(report.Findings()) != 0 {
		return artifactnpm.ArchiveInventory{}, report, err
	}
	return capture.inventory, report, nil
}

type archiveCapture struct {
	compressed *io.LimitedReader
	buffered   *bufio.Reader
	decoded    *io.LimitedReader
	digest     hash.Hash
	entries    []artifactnpm.ArchiveEntry
	manifest   []byte
	inventory  artifactnpm.ArchiveInventory
}

func (i *Inspector) openInventorySource(name string) (*os.File, error) {
	root, err := os.OpenRoot(i.intakeRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	relative, err := filepath.Rel(i.intakeRoot, name)
	if err != nil {
		return nil, err
	}
	before, err := root.Lstat(relative)
	if err != nil || !before.Mode().IsRegular() {
		return nil, errors.New("npm inventory source is not a controlled regular file")
	}
	file, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, errors.Join(errors.New("npm inventory source identity changed while opening"), err, file.Close())
	}
	return file, nil
}

func (c *archiveCapture) input(file *os.File, artifact domain.AcquiredArtifact) (io.Reader, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat npm inventory source: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || uint64(info.Size()) != artifact.SizeBytes() || artifact.SizeBytes() > decodedLimit {
		return nil, errors.New("npm inventory source size or type differs from its acquired subject")
	}
	c.compressed = &io.LimitedReader{R: file, N: int64(artifact.SizeBytes()) + 1}
	c.digest = sha256.New()
	c.buffered = bufio.NewReader(io.TeeReader(c.compressed, c.digest))
	return c.buffered, nil
}

func (c *archiveCapture) contents(reader *gzip.Reader) io.Reader {
	reader.Multistream(false)
	c.decoded = &io.LimitedReader{R: reader, N: decodedLimit + 1}
	return c.decoded
}

func (c *archiveCapture) entry(ctx context.Context, header *tar.Header, name string, reader io.Reader, manifest []byte) error {
	item := artifactnpm.ArchiveEntry{Path: strings.TrimPrefix(name, "package/"), Directory: header.Typeflag == tar.TypeDir, Size: header.Size, Mode: header.Mode}
	if !item.Directory {
		digest := sha256.New()
		if manifest != nil {
			_, _ = digest.Write(manifest) // hash.Hash.Write never fails.
			c.manifest = append([]byte(nil), manifest...)
		} else if err := copyArchivePayload(ctx, digest, reader, header.Size); err != nil {
			return err
		}
		var err error
		item.Digest, err = domain.NewSHA256Digest(hex.EncodeToString(digest.Sum(nil)))
		if err != nil {
			return err
		}
	}
	c.entries = append(c.entries, item)
	return nil
}

func copyArchivePayload(ctx context.Context, writer io.Writer, reader io.Reader, size int64) error {
	buffer := make([]byte, 32<<10)
	remaining := size
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := io.ReadFull(reader, buffer[:min(int64(len(buffer)), remaining)])
		if err != nil {
			return err
		}
		if written, err := writer.Write(buffer[:n]); err != nil {
			return err
		} else if written != n {
			return io.ErrShortWrite
		}
		remaining -= int64(n)
	}
	return ctx.Err()
}

func (c *archiveCapture) finish(ctx context.Context, artifact domain.AcquiredArtifact) error {
	buffer := make([]byte, 32<<10)
	zero := make([]byte, len(buffer))
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := c.decoded.Read(buffer)
		if !bytes.Equal(buffer[:n], zero[:n]) || c.decoded.N == 0 {
			return errors.New("npm inventory archive has excess or nonzero trailing decoded content")
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("finish npm inventory gzip stream: %w", err)
		}
	}
	if _, err := c.buffered.Peek(1); err != io.EOF {
		return errors.New("npm inventory archive has trailing compressed content or an incomplete source read")
	}
	if c.compressed.N != 1 || hex.EncodeToString(c.digest.Sum(nil)) != artifact.Digest().String() {
		return errors.New("npm inventory source bytes differ from their acquired subject")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	inventory, err := artifactnpm.NewArchiveInventory(artifact, c.entries, c.manifest)
	if err != nil {
		return err
	}
	c.inventory = inventory
	return nil
}
