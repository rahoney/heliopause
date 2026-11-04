// Package githubrelease performs bounded, non-executing inspection of a
// GitHub Release asset held only in controlled intake.
package githubrelease

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/content"
)

const (
	archiveEntryLimit = 10000
	archiveFileLimit  = 64 << 20
	archiveTotalLimit = 200 << 20
)

// StaticInspector never executes or extracts the acquired asset.
type StaticInspector struct{ intakeRoot string }

func NewStaticInspector(intakeRoot string) (*StaticInspector, error) {
	if !filepath.IsAbs(intakeRoot) {
		return nil, errors.New("GitHub Release static inspector intake root must be absolute")
	}
	return &StaticInspector{intakeRoot: filepath.Clean(intakeRoot)}, nil
}

func (i *StaticInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.inspect(ctx, artifact, nil, nil)
}

// InspectSources returns only bounded DATA bound to the acquired compressed
// subject after every member and trailer has been checked. An empty selection
// is not a source or binary content-analysis verdict.
func (i *StaticInspector) InspectSources(ctx context.Context, artifact domain.AcquiredArtifact) ([]ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var sources []ports.SourceAnalysisFile
	report, err := i.inspect(ctx, artifact, &sources, nil)
	if err != nil || len(report.Findings()) != 0 {
		return nil, report, err
	}
	if err := ctx.Err(); err != nil {
		return nil, report, err
	}
	return sources, report, nil
}

// InspectContent adds native facts from the same held raw asset/member pass.
// Source DATA remains input for the separate fixed analyzer, never executed.
func (i *StaticInspector) InspectContent(ctx context.Context, artifact domain.AcquiredArtifact) ([]ports.SourceAnalysisFile, domain.InspectionReport, error) {
	native, err := content.NewBinaryInspection(artifact)
	if err != nil {
		return nil, domain.InspectionReport{}, err
	}
	var sources []ports.SourceAnalysisFile
	static, err := i.inspect(ctx, artifact, &sources, native)
	if err != nil || len(static.Findings()) != 0 {
		return nil, static, err
	}
	nativeReport, nativeErr := native.Report(ctx)
	if len(nativeReport.Executions()) == 0 {
		return nil, static, nativeErr
	}
	report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{static, nativeReport})
	return sources, report, errors.Join(nativeErr, err)
}

func (i *StaticInspector) inspect(ctx context.Context, artifact domain.AcquiredArtifact, sources *[]ports.SourceAnalysisFile, native *content.BinaryInspection) (domain.InspectionReport, error) {
	if ctx == nil || ctx.Err() != nil || i == nil || artifact.Identity().Source().String() != "github-release" {
		return domain.InspectionReport{}, errors.New("GitHub Release static inspection request is invalid")
	}
	file, err := i.openAsset(artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	defer file.Close()
	if sources != nil {
		digest := sha256.New()
		info, err := file.Stat()
		if err != nil {
			return domain.InspectionReport{}, err
		}
		n, err := io.Copy(digest, io.NewSectionReader(file, 0, info.Size()))
		after, statErr := file.Stat()
		if err != nil || statErr != nil || n != info.Size() || after.Size() != info.Size() || after.ModTime() != info.ModTime() || hex.EncodeToString(digest.Sum(nil)) != artifact.Digest().String() {
			return domain.InspectionReport{}, errors.New("GitHub Release source subject differs from acquired bytes")
		}
		if err := ctx.Err(); err != nil {
			return domain.InspectionReport{}, err
		}
	}
	header := make([]byte, 64)
	n, _ := io.ReadFull(file, header)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return domain.InspectionReport{}, err
	}
	filename := artifact.Identity().Variant()
	switch {
	case n >= 4 && bytes.Equal(header[:4], []byte{0x7f, 'E', 'L', 'F'}):
		if strings.HasSuffix(filename, ".zip") || strings.HasSuffix(filename, ".tar.gz") {
			return i.finding(artifact, "M6_FORMAT_NAME_MISMATCH", "GitHub Release asset filename does not match ELF bytes.")
		}
		if err := inspectELF(header[:n], artifact.SizeBytes()); err != nil {
			return i.finding(artifact, "M6_ELF_PLATFORM_UNSUPPORTED", "GitHub Release ELF is not a supported Linux amd64 candidate.")
		}
		if native != nil {
			var body []byte
			if artifact.SizeBytes() <= archiveFileLimit {
				body, err = io.ReadAll(io.LimitReader(file, archiveFileLimit+1))
				if err != nil || uint64(len(body)) != artifact.SizeBytes() {
					return domain.InspectionReport{}, errors.Join(errors.New("GitHub Release native asset read is incomplete"), err)
				}
			}
			// A declared size above the fixed bound records unanalysed
			// coverage without allocation or a false integrity accusation.
			_ = native.Inspect(ctx, content.BinaryMember{Path: filename, Body: body, Size: artifact.SizeBytes(), Digest: artifact.Digest()})
		}
		return i.complete(artifact, "github-release-elf-static", "GitHub Release Linux amd64 ELF static header inspection completed.")
	case n >= 4 && (bytes.Equal(header[:4], []byte{'P', 'K', 3, 4}) || bytes.Equal(header[:4], []byte{'P', 'K', 5, 6})):
		if !strings.HasSuffix(filename, ".zip") {
			return i.finding(artifact, "M6_FORMAT_NAME_MISMATCH", "GitHub Release asset filename does not match ZIP bytes.")
		}
		if err := inspectZIP(ctx, file, int64(artifact.SizeBytes()), sources, native); err != nil {
			return i.finding(artifact, "M6_ARCHIVE_INVALID", "GitHub Release ZIP archive is unsafe or exceeds bounds.")
		}
		return i.complete(artifact, "github-release-zip-static", "GitHub Release ZIP static archive inspection completed.")
	case n >= 2 && header[0] == 0x1f && header[1] == 0x8b:
		if !strings.HasSuffix(filename, ".tar.gz") {
			return i.finding(artifact, "M6_FORMAT_NAME_MISMATCH", "GitHub Release asset filename does not match tar.gz bytes.")
		}
		if err := inspectTarGZ(ctx, file, sources, native); err != nil {
			return i.finding(artifact, "M6_ARCHIVE_INVALID", "GitHub Release tar.gz archive is unsafe or exceeds bounds.")
		}
		return i.complete(artifact, "github-release-targz-static", "GitHub Release tar.gz static archive inspection completed.")
	default:
		return i.finding(artifact, "M6_FORMAT_UNSUPPORTED", "GitHub Release asset format is unsupported.")
	}
}

func (i *StaticInspector) openAsset(artifact domain.AcquiredArtifact) (*os.File, error) {
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != "github-release" {
		return nil, errors.New("GitHub Release intake handle is invalid")
	}
	runID, err := domain.ParseRunID(parts[1])
	if err != nil {
		return nil, errors.New("GitHub Release intake handle is invalid")
	}
	root, err := os.OpenRoot(i.intakeRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	assetPath := filepath.Join(runID.String(), "asset")
	info, err := root.Lstat(assetPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || uint64(info.Size()) != artifact.SizeBytes() {
		return nil, errors.New("GitHub Release intake asset is invalid")
	}
	file, err := root.OpenFile(assetPath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) || uint64(opened.Size()) != artifact.SizeBytes() {
		return nil, errors.Join(errors.New("GitHub Release intake asset changed while opening"), err, file.Close())
	}
	return file, nil
}

func inspectELF(header []byte, size uint64) error {
	if len(header) < 64 || header[4] != 2 || header[5] != 1 || header[6] != 1 || binary.LittleEndian.Uint16(header[18:20]) != 62 || binary.LittleEndian.Uint16(header[52:54]) != 64 {
		return errors.New("unsupported ELF")
	}
	phoff, phents, phnum := binary.LittleEndian.Uint64(header[32:40]), binary.LittleEndian.Uint16(header[54:56]), binary.LittleEndian.Uint16(header[56:58])
	if phents != 56 || phnum > 128 || phoff > size || uint64(phnum)*56 > size-phoff {
		return errors.New("invalid ELF program headers")
	}
	return nil
}

func inspectZIP(ctx context.Context, file *os.File, size int64, sources *[]ports.SourceAnalysisFile, native *content.BinaryInspection) error {
	reader, err := zip.NewReader(file, size)
	if err != nil || len(reader.File) > archiveEntryLimit {
		return errors.New("invalid ZIP")
	}
	var total uint64
	seen := map[string]bool{}
	for _, entry := range reader.File {
		mode := entry.Mode()
		if !safeArchivePath(entry.Name) || seen[entry.Name] || entry.UncompressedSize64 > archiveFileLimit || (mode&os.ModeSymlink != 0) || (!entry.FileInfo().IsDir() && !mode.IsRegular()) {
			return errors.New("unsafe ZIP entry")
		}
		seen[entry.Name] = true
		total += entry.UncompressedSize64
		if total > archiveTotalLimit {
			return errors.New("ZIP exceeds bounds")
		}
		if entry.FileInfo().IsDir() {
			if entry.UncompressedSize64 != 0 {
				return errors.New("ZIP directory has payload")
			}
			continue
		}
		stream, err := entry.Open()
		if err != nil {
			return err
		}
		readErr := readArchiveMember(ctx, stream, entry.Name, int64(entry.UncompressedSize64), uint32(mode.Perm()), sources, native)
		closeErr := stream.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
	}
	return nil
}

func inspectTarGZ(ctx context.Context, file *os.File, sources *[]ports.SourceAnalysisFile, native *content.BinaryInspection) error {
	compressed := bufio.NewReader(file)
	gzipReader, err := gzip.NewReader(compressed)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	gzipReader.Multistream(false)
	decoded := &io.LimitedReader{R: gzipReader, N: archiveTotalLimit + 1}
	reader := tar.NewReader(decoded)
	seen := map[string]bool{}
	entries := 0
	var total int64
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		entries++
		if header.Size < 0 || header.Size > archiveFileLimit || total > archiveTotalLimit-header.Size {
			return errors.New("unsafe tar entry")
		}
		total += header.Size
		if entries > archiveEntryLimit || total > archiveTotalLimit || !safeArchivePath(header.Name) || seen[header.Name] || (header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir) {
			return errors.New("unsafe tar entry")
		}
		seen[header.Name] = true
		if header.Typeflag == tar.TypeDir {
			if header.Size != 0 {
				return errors.New("tar directory has payload")
			}
		} else if err := readArchiveMember(ctx, reader, header.Name, header.Size, uint32(header.Mode), sources, native); err != nil {
			return err
		}
	}
	buffer := make([]byte, 32<<10)
	for {
		n, readErr := decoded.Read(buffer)
		if len(bytes.Trim(buffer[:n], "\x00")) != 0 || decoded.N <= 0 {
			return errors.New("tar has uninspected decoded content or exceeds bounds")
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if _, err := compressed.Peek(1); err != io.EOF {
		return errors.New("tar.gz has trailing compressed content")
	}
	return nil
}

// Existing archive bounds are tighter than the fixed parser's input bounds.
// Integrity is checked for every member, whether or not its name selects a parser.
func readArchiveMember(ctx context.Context, reader io.Reader, name string, size int64, mode uint32, sources *[]ports.SourceAnalysisFile, native *content.BinaryInspection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var prefix []byte
	language := ""
	if sources != nil || native != nil {
		prefix = make([]byte, min(size, 512))
		if _, err := io.ReadFull(reader, prefix); err != nil {
			return err
		}
		language = content.SourceLanguage(name, prefix)
	}
	binaryMember := native != nil && content.IsBinary(name, prefix)
	var body bytes.Buffer
	var destination io.Writer = io.Discard
	if language != "" || binaryMember {
		destination = &body
	}
	if _, err := destination.Write(prefix); err != nil {
		return err
	}
	n, err := io.Copy(destination, io.LimitReader(reader, size-int64(len(prefix))+1))
	if err != nil || n+int64(len(prefix)) != size {
		return errors.Join(errors.New("archive member size or integrity is invalid"), err)
	}
	if sources != nil && language != "" && !binaryMember {
		*sources = append(*sources, ports.SourceAnalysisFile{Path: name, Language: language, Body: body.Bytes()})
	}
	if binaryMember {
		sum := sha256.Sum256(body.Bytes())
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		_ = native.Inspect(ctx, content.BinaryMember{Path: name, Body: body.Bytes(), Mode: mode, Digest: digest})
	}
	return ctx.Err()
}

func safeArchivePath(value string) bool {
	return value != "" && !strings.HasPrefix(value, "/") && path.Clean(value) == value && !strings.HasPrefix(value, "../") && value != "."
}

func (i *StaticInspector) complete(artifact domain.AcquiredArtifact, kind, summary string) (domain.InspectionReport, error) {
	return i.report(artifact, kind, summary, "")
}
func (i *StaticInspector) finding(artifact domain.AcquiredArtifact, code, summary string) (domain.InspectionReport, error) {
	return i.report(artifact, "github-release-static", summary, code)
}
func (i *StaticInspector) report(artifact domain.AcquiredArtifact, kind, summary, code string) (domain.InspectionReport, error) {
	checkID, _ := domain.NewCheckID("github-release-static")
	profile, _ := domain.CheckProfileDigest("github-release-static-v2", "")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	evidenceID, _ := domain.NewEvidenceID("github-release-static-result")
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), kind, summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if code == "" {
		return domain.NewInspectionReport(check, nil, []domain.Evidence{evidence})
	}
	finding, _ := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
	return domain.NewInspectionReport(check, []domain.Finding{finding}, []domain.Evidence{evidence})
}

func (i *StaticInspector) ExpectedChecks(ctx context.Context, a domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if i == nil || ctx == nil || ctx.Err() != nil || kind != domain.CheckInspection || a.Identity().Source().String() != "github-release" {
		return nil, errors.New("github release static declaration is invalid")
	}
	id, _ := domain.NewCheckID("github-release-static")
	profile, _ := domain.CheckProfileDigest("github-release-static-v2", "")
	requirement, err := domain.NewCheckRequirement(id, kind, true, profile)
	return []domain.CheckRequirement{requirement}, err
}
