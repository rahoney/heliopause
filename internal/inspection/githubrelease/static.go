// Package githubrelease performs bounded, non-executing inspection of a
// GitHub Release asset held only in controlled intake.
package githubrelease

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
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
	if ctx == nil || ctx.Err() != nil || i == nil || artifact.Identity().Source().String() != "github-release" {
		return domain.InspectionReport{}, errors.New("GitHub Release static inspection request is invalid")
	}
	file, err := i.openAsset(artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	defer file.Close()
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
		return i.complete(artifact, "github-release-elf-static", "GitHub Release Linux amd64 ELF static header inspection completed.")
	case n >= 4 && (bytes.Equal(header[:4], []byte{'P', 'K', 3, 4}) || bytes.Equal(header[:4], []byte{'P', 'K', 5, 6})):
		if !strings.HasSuffix(filename, ".zip") {
			return i.finding(artifact, "M6_FORMAT_NAME_MISMATCH", "GitHub Release asset filename does not match ZIP bytes.")
		}
		if err := inspectZIP(file, int64(artifact.SizeBytes())); err != nil {
			return i.finding(artifact, "M6_ARCHIVE_INVALID", "GitHub Release ZIP archive is unsafe or exceeds bounds.")
		}
		return i.complete(artifact, "github-release-zip-static", "GitHub Release ZIP static archive inspection completed.")
	case n >= 2 && header[0] == 0x1f && header[1] == 0x8b:
		if !strings.HasSuffix(filename, ".tar.gz") {
			return i.finding(artifact, "M6_FORMAT_NAME_MISMATCH", "GitHub Release asset filename does not match tar.gz bytes.")
		}
		if err := inspectTarGZ(file); err != nil {
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
	assetPath := filepath.Join(i.intakeRoot, runID.String(), "asset")
	info, err := os.Lstat(assetPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || uint64(info.Size()) != artifact.SizeBytes() {
		return nil, errors.New("GitHub Release intake asset is invalid")
	}
	return os.Open(assetPath)
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

func inspectZIP(file *os.File, size int64) error {
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
	}
	return nil
}

func inspectTarGZ(file *os.File) error {
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(io.LimitReader(gzipReader, archiveTotalLimit+1))
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
	}
	return nil
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
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
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
