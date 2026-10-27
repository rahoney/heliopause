package pypi

import (
	"archive/zip"
	"bytes"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// PipxRuntimeZIP holds selected acquisition bytes, not installed-tool authority
// or an inspection result. Its zero value is invalid.
type PipxRuntimeZIP struct {
	body  []byte
	files []PipxRuntimeZIPFile
}

// PipxRuntimeZIPFile describes bytes in the original ZIP namespace. It does not
// assign an installed destination or an execution role.
type PipxRuntimeZIPFile struct {
	Path   string
	Size   int64
	SHA256 string
}

// ParsePipxRuntimeZIP accepts only the selected immutable runtime asset. Release
// identity, publisher verification and dynamic inspection remain separate.
func ParsePipxRuntimeZIP(body []byte) (PipxRuntimeZIP, error) {
	if len(body) != runtimeidentity.PipxToolSize || len(body) > pipxSupplyZIPLimit {
		return PipxRuntimeZIP{}, errors.New("pipx runtime ZIP size differs from selected asset")
	}
	owned := bytes.Clone(body)
	if pipxSupplyDigest(owned) != runtimeidentity.PipxToolSHA256 {
		return PipxRuntimeZIP{}, errors.New("pipx runtime ZIP digest differs from selected asset")
	}
	files, err := pipxRuntimeZIPFiles(owned)
	if err != nil {
		return PipxRuntimeZIP{}, err
	}
	return PipxRuntimeZIP{body: owned, files: files}, nil
}

func (z PipxRuntimeZIP) Valid() bool {
	return len(z.body) == runtimeidentity.PipxToolSize && len(z.files) > 0
}
func (z PipxRuntimeZIP) Bytes() []byte { return bytes.Clone(z.body) }
func (z PipxRuntimeZIP) Files() []PipxRuntimeZIPFile {
	return append([]PipxRuntimeZIPFile(nil), z.files...)
}

func pipxRuntimeZIPFiles(body []byte) ([]PipxRuntimeZIPFile, error) {
	if len(body) == 0 || len(body) > pipxSupplyZIPLimit {
		return nil, errors.New("pipx runtime ZIP exceeds bound")
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil || len(archive.File) == 0 || len(archive.File) > 4096 || archive.Comment != "" {
		return nil, errors.New("pipx runtime ZIP directory is invalid")
	}
	files := make([]PipxRuntimeZIPFile, 0, len(archive.File))
	seen := make(map[string]bool, len(archive.File))
	previous := ""
	var total int64
	fixedTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, member := range archive.File {
		name, directory, ok := wheelEntryPath(member)
		if !ok || directory || name <= previous || strings.ContainsAny(name, "\x00\r\n") || member.Mode().Perm() != 0444 || member.Method != zip.Store || !member.Modified.Equal(fixedTime) || member.Comment != "" || member.UncompressedSize64 > uint64(pipxSupplyZIPLimit-total) {
			return nil, errors.New("pipx runtime ZIP member metadata is invalid")
		}
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if seen[parent] {
				return nil, errors.New("pipx runtime ZIP file and directory collide")
			}
		}
		data, err := readZipEntry(member, pipxSupplyZIPLimit-total)
		if err != nil || uint64(len(data)) != member.UncompressedSize64 {
			return nil, errors.New("pipx runtime ZIP member bytes are invalid")
		}
		total += int64(len(data))
		files = append(files, PipxRuntimeZIPFile{Path: name, Size: int64(len(data)), SHA256: pipxSupplyDigest(data)})
		seen[name] = true
		previous = name
	}
	return files, nil
}
