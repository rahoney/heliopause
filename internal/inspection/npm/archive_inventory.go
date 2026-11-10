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
	"path"
	"path/filepath"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/content"
)

// InspectArchive reuses static inspection and additionally measures every file
// and the complete compressed source. A finding or error yields no inventory.
// The original Inspect API and its result semantics remain unchanged.
func (i *Inspector) InspectArchive(ctx context.Context, artifact domain.AcquiredArtifact) (artifactnpm.ArchiveInventory, domain.InspectionReport, error) {
	if i != nil && i.analyzer != nil {
		inventory, _, report, err := i.InspectArchiveContent(ctx, artifact)
		return inventory, report, err
	}
	return i.ReadArchiveInventory(ctx, artifact)
}

// ReadArchiveInventory remeasures the complete original archive for entry and
// layout binding. It provides no content inspection or current Policy approval.
func (i *Inspector) ReadArchiveInventory(ctx context.Context, artifact domain.AcquiredArtifact) (artifactnpm.ArchiveInventory, domain.InspectionReport, error) {
	capture := &archiveCapture{}
	return i.captureArchive(ctx, artifact, capture)
}

// InspectArchiveSources measures the same archive once and retains only bounded
// source DATA for the fixed parsers. No files are returned before the complete
// archive, compressed digest and all member identities have been verified.
// An empty selection is not a completed content inspection.
func (i *Inspector) InspectArchiveSources(ctx context.Context, artifact domain.AcquiredArtifact) (artifactnpm.ArchiveInventory, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	capture := &archiveCapture{captureSources: true}
	inventory, report, err := i.captureArchive(ctx, artifact, capture)
	if err != nil || !inventory.Valid() {
		return artifactnpm.ArchiveInventory{}, nil, report, err
	}
	return inventory, capture.sources, report, nil
}

// InspectArchiveContent binds native facts to the same measured archive. Facts
// remain private until its complete compressed digest/trailer is verified.
func (i *Inspector) InspectArchiveContent(ctx context.Context, artifact domain.AcquiredArtifact) (artifactnpm.ArchiveInventory, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	native, err := content.NewBinaryInspection(artifact)
	if err != nil {
		return artifactnpm.ArchiveInventory{}, nil, domain.InspectionReport{}, err
	}
	capture := &archiveCapture{captureSources: true, native: native, metadata: make(map[string][]byte)}
	inventory, static, err := i.captureArchive(ctx, artifact, capture)
	if err != nil || !inventory.Valid() {
		return artifactnpm.ArchiveInventory{}, nil, static, err
	}
	declared, err := inventory.DeclaredSourcePaths(capture.metadata)
	if err != nil {
		return artifactnpm.ArchiveInventory{}, nil, static, err
	}
	missing := make(map[string]bool)
	indices := make(map[string]int)
	for index, member := range capture.members {
		if declared[member.Path] && member.Language == "" && !member.Binary {
			missing[member.Path] = true
			indices[member.Path] = index
		}
	}
	if len(missing) != 0 {
		// A late manifest may name an earlier extensionless member. Reuse
		// the complete authenticated reader, retaining only these targets;
		// do not copy existing source bodies or charge the graph twice.
		additional := &archiveCapture{sourceTargets: missing}
		verified, _, readErr := i.captureArchive(ctx, artifact, additional)
		if readErr != nil || !verified.Matches(artifact) {
			return artifactnpm.ArchiveInventory{}, nil, static, errors.Join(errors.New("declared npm source read is incomplete"), readErr)
		}
		for _, source := range additional.sources {
			index, present := indices[source.Path]
			if !present {
				return artifactnpm.ArchiveInventory{}, nil, static, errors.New("declared npm source is outside its original inventory")
			}
			member := &capture.members[index]
			digest := sha256.Sum256(source.Body)
			if member.Size != uint64(len(source.Body)) || member.Digest.String() != hex.EncodeToString(digest[:]) {
				return artifactnpm.ArchiveInventory{}, nil, static, errors.New("declared npm source differs from its original member")
			}
			member.Language, member.UnsupportedSource = source.Language, false
			capture.sources = append(capture.sources, source)
		}
	}
	inventoryFacts, err := content.NewInventory(artifact, capture.members)
	if err != nil {
		return artifactnpm.ArchiveInventory{}, nil, static, err
	}
	nativeReport, nativeErr := native.Report(ctx, inventoryFacts)
	if len(nativeReport.Executions()) == 0 {
		return artifactnpm.ArchiveInventory{}, nil, static, nativeErr
	}
	census, censusErr := inventoryFacts.Report(ctx)
	if censusErr != nil {
		return artifactnpm.ArchiveInventory{}, nil, static, errors.Join(nativeErr, censusErr)
	}
	reports := []domain.InspectionReport{static, nativeReport, census}
	var sourceErr error
	if i.analyzer != nil {
		var source domain.InspectionReport
		source, sourceErr = content.InspectSources(ctx, i.analyzer, artifact, capture.sources, inventoryFacts)
		if len(source.Executions()) != 0 {
			reports = append(reports, source)
		}
	}
	report, err := domain.NewCompositeInspectionReport(reports)
	return inventory, capture.sources, report, errors.Join(nativeErr, sourceErr, err)
}

func (i *Inspector) captureArchive(ctx context.Context, artifact domain.AcquiredArtifact, capture *archiveCapture) (artifactnpm.ArchiveInventory, domain.InspectionReport, error) {
	report, err := i.inspect(ctx, artifact, capture)
	if err != nil || len(report.Findings()) != 0 {
		return artifactnpm.ArchiveInventory{}, report, err
	}
	return capture.inventory, report, nil
}

type archiveCapture struct {
	compressed     *io.LimitedReader
	buffered       *bufio.Reader
	decoded        *io.LimitedReader
	digest         hash.Hash
	entries        []artifactnpm.ArchiveEntry
	manifest       []byte
	inventory      artifactnpm.ArchiveInventory
	selectedPath   string
	selectedBody   []byte
	captureSources bool
	sourceTargets  map[string]bool
	sources        []ports.SourceAnalysisFile
	native         *content.BinaryInspection
	members        []content.MemberFact
	metadata       map[string][]byte
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
		var prefix []byte
		language := ""
		if c.captureSources || c.sourceTargets[item.Path] {
			if manifest == nil {
				prefix = make([]byte, min(header.Size, 512))
				if _, err := io.ReadFull(reader, prefix); err != nil {
					return err
				}
			}
			language = content.SourceLanguage(item.Path, prefix)
			if language == "" && c.sourceTargets[item.Path] && !bytes.HasPrefix(prefix, []byte("#!")) {
				language = "javascript"
			}
		}
		// The existing npm limits (20MiB/file, 200MiB decoded, 10000
		// entries including the manifest) already bound this subset more
		// tightly than the parser input. Do not add a second archive budget.
		retainNative := c.native != nil && content.IsBinary(item.Path, prefix)
		retainSource := language != "" && !retainNative
		retainMetadata := c.metadata != nil && path.Base(item.Path) == "package.json"
		var nativeBody []byte
		if manifest != nil {
			_, _ = digest.Write(manifest) // hash.Hash.Write never fails.
			c.manifest = append([]byte(nil), manifest...)
			if retainMetadata {
				c.metadata[item.Path] = c.manifest
			}
			if retainSource {
				c.sources = append(c.sources, ports.SourceAnalysisFile{Path: item.Path, Language: language, Body: c.manifest})
			}
			if item.Path == c.selectedPath {
				c.selectedBody = bytes.Clone(manifest)
			}
		} else {
			var selected bytes.Buffer
			var writer io.Writer = digest
			if item.Path == c.selectedPath || retainSource || retainNative || retainMetadata {
				writer = io.MultiWriter(digest, &selected)
			}
			if _, err := writer.Write(prefix); err != nil {
				return err
			}
			if err := copyArchivePayload(ctx, writer, reader, header.Size-int64(len(prefix))); err != nil {
				return err
			}
			if item.Path == c.selectedPath {
				c.selectedBody = selected.Bytes()
			}
			if retainSource {
				c.sources = append(c.sources, ports.SourceAnalysisFile{Path: item.Path, Language: language, Body: selected.Bytes()})
			}
			if retainNative {
				nativeBody = selected.Bytes()
			}
			if retainMetadata {
				c.metadata[item.Path] = selected.Bytes()
			}
		}
		var err error
		item.Digest, err = domain.NewSHA256Digest(hex.EncodeToString(digest.Sum(nil)))
		if err != nil {
			return err
		}
		if retainNative {
			_ = c.native.Inspect(ctx, content.BinaryMember{Path: item.Path, Body: nativeBody, Mode: uint32(item.Mode), Digest: item.Digest})
		}
		if c.native != nil {
			if retainNative {
				language = ""
			}
			c.members = append(c.members, content.MemberFact{Path: item.Path, Size: uint64(item.Size), Digest: item.Digest, Language: language, Binary: retainNative, UnsupportedSource: !retainNative && language == "" && content.UnsupportedSource(item.Path, prefix, uint32(item.Mode))})
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
