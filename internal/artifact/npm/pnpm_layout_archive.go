package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"path"
	"strconv"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMLayoutArchiveLimit bounds both native layout transport and parsing.
const PNPMLayoutArchiveLimit = 200 << 20

const (
	pnpmLayoutEntryLimit = 100000
	pnpmLayoutAliasLimit = 64
)

var errPNPMLayoutArchive = errors.New("pnpm layout archive is unavailable or structurally invalid")

// PNPMLayoutEntry describes unverified native layout data. Size is the tar
// header size (zero for links); Digest also describes a resolved hardlink's
// bytes. Neither field binds the entry to an acquired package.
type PNPMLayoutEntry struct {
	Name, Link string
	Kind       byte
	Mode, Size int64
	UID, GID   int
	Digest     domain.ContentDigest
}

type pnpmLayoutFile struct{ start, end int }

// PNPMLayoutArchive is parsed, unverified format data. It has no acquisition,
// graph, safety, execution or promotion authority. Full inventory/edge/bin and
// generated metadata validation must follow. Its zero value contains no files.
type PNPMLayoutArchive struct {
	body    []byte
	entries []PNPMLayoutEntry
	files   map[string]pnpmLayoutFile
}

// ParsePNPMLayoutArchive reads one bounded, uncompressed node_modules tar
// envelope without extracting content or following Host filesystem links.
func ParsePNPMLayoutArchive(ctx context.Context, body []byte) (PNPMLayoutArchive, error) {
	return parsePNPMTreeArchive(ctx, body, "node_modules")
}

// ParsePNPMInspectionPayloadArchive parses the fixed project root returned by
// inspection-payload readback. It is unverified format data, not a native lock,
// source binding or permission to execute. No caller-supplied root is accepted.
func ParsePNPMInspectionPayloadArchive(ctx context.Context, body []byte) (PNPMLayoutArchive, error) {
	return parsePNPMTreeArchive(ctx, body, "project")
}

func parsePNPMTreeArchive(ctx context.Context, body []byte, rootName string) (PNPMLayoutArchive, error) {
	if ctx == nil || len(body) == 0 || len(body) > PNPMLayoutArchiveLimit {
		return PNPMLayoutArchive{}, errPNPMLayoutArchive
	}
	if err := ctx.Err(); err != nil {
		return PNPMLayoutArchive{}, errors.Join(errPNPMLayoutArchive, err)
	}
	layout := PNPMLayoutArchive{body: bytes.Clone(body), files: map[string]pnpmLayoutFile{}}
	input := bytes.NewReader(layout.body)
	reader := tar.NewReader(input)
	byName := map[string]int{}
	lastEntryEnd := 0
	for {
		if err := ctx.Err(); err != nil {
			return PNPMLayoutArchive{}, errors.Join(errPNPMLayoutArchive, err)
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return PNPMLayoutArchive{}, errors.Join(errPNPMLayoutArchive, err)
		}
		if len(layout.entries) >= pnpmLayoutEntryLimit || !validPNPMLayoutHeader(header) {
			return PNPMLayoutArchive{}, errPNPMLayoutArchive
		}
		name := header.Name
		if header.Typeflag == tar.TypeDir {
			name = strings.TrimSuffix(name, "/")
		}
		if !validInventoryPath(name) || name != rootName && !strings.HasPrefix(name, rootName+"/") {
			return PNPMLayoutArchive{}, errPNPMLayoutArchive
		}
		if _, exists := byName[name]; exists {
			return PNPMLayoutArchive{}, errPNPMLayoutArchive
		}
		item := PNPMLayoutEntry{Name: name, Link: header.Linkname, Kind: header.Typeflag, Mode: header.Mode, Size: header.Size, UID: header.Uid, GID: header.Gid}
		switch item.Kind {
		case tar.TypeReg:
			start := len(layout.body) - input.Len()
			digest := sha256.New()
			copied, err := io.CopyN(digest, reader, item.Size)
			if err != nil || copied != item.Size || len(layout.body)-input.Len()-start != int(copied) {
				return PNPMLayoutArchive{}, errors.Join(errPNPMLayoutArchive, err)
			}
			item.Digest, err = domain.NewSHA256Digest(hex.EncodeToString(digest.Sum(nil)))
			if err != nil {
				return PNPMLayoutArchive{}, err
			}
			layout.files[name] = pnpmLayoutFile{start, start + int(copied)}
		case tar.TypeLink:
			if !validInventoryPath(item.Link) || !strings.HasPrefix(item.Link, rootName+"/") {
				return PNPMLayoutArchive{}, errPNPMLayoutArchive
			}
		case tar.TypeSymlink:
			if item.Link == "" || len(item.Link) > 4096 || path.Clean(item.Link) != item.Link || strings.HasPrefix(item.Link, "/") || strings.ContainsAny(item.Link, "\\:\x00\r\n") {
				return PNPMLayoutArchive{}, errPNPMLayoutArchive
			}
			target := path.Join(path.Dir(name), item.Link)
			if !validInventoryPath(target) || target != rootName && !strings.HasPrefix(target, rootName+"/") {
				return PNPMLayoutArchive{}, errPNPMLayoutArchive
			}
		}
		lastEntryEnd = (len(layout.body) - input.Len() + 511) / 512 * 512
		byName[name] = len(layout.entries)
		layout.entries = append(layout.entries, item)
	}
	if len(layout.entries) == 0 || len(layout.body)-input.Len() < lastEntryEnd+1024 {
		return PNPMLayoutArchive{}, errPNPMLayoutArchive
	}
	root, present := byName[rootName]
	if !present || layout.entries[root].Kind != tar.TypeDir {
		return PNPMLayoutArchive{}, errPNPMLayoutArchive
	}
	for _, item := range layout.entries {
		for parent := path.Dir(item.Name); parent != "."; parent = path.Dir(parent) {
			index, exists := byName[parent]
			if !exists || layout.entries[index].Kind != tar.TypeDir {
				return PNPMLayoutArchive{}, errPNPMLayoutArchive
			}
		}
		if item.Kind == tar.TypeLink {
			if err := layout.resolveAlias(item.Name, byName); err != nil {
				return PNPMLayoutArchive{}, err
			}
		}
	}
	for _, value := range layout.body[len(layout.body)-input.Len():] {
		if value != 0 {
			return PNPMLayoutArchive{}, errPNPMLayoutArchive
		}
	}
	if err := ctx.Err(); err != nil {
		return PNPMLayoutArchive{}, errors.Join(errPNPMLayoutArchive, err)
	}
	return layout, nil
}

func validPNPMLayoutHeader(header *tar.Header) bool {
	if header.Uid != 1000 || header.Gid != 1000 || header.Mode < 0 || header.Mode > 0o777 || header.Size < 0 || header.Size > ArchiveFileLimit {
		return false
	}
	if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir && header.Typeflag != tar.TypeLink && header.Typeflag != tar.TypeSymlink {
		return false
	}
	if header.Typeflag != tar.TypeReg && header.Size != 0 || header.Typeflag != tar.TypeLink && header.Typeflag != tar.TypeSymlink && header.Linkname != "" {
		return false
	}
	for key, value := range header.PAXRecords {
		switch key {
		case "path", "linkpath":
			if len(value) > 4096 {
				return false
			}
		case "mtime", "atime", "ctime":
			if len(value) > 64 {
				return false
			}
			number, err := strconv.ParseFloat(value, 64)
			if err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (l *PNPMLayoutArchive) resolveAlias(name string, byName map[string]int) error {
	var chain []string
	current := name
	for {
		if file, present := l.files[current]; present {
			digest := l.entries[byName[current]].Digest
			for _, alias := range chain {
				l.files[alias] = file
				l.entries[byName[alias]].Digest = digest
			}
			return nil
		}
		index, exists := byName[current]
		if !exists || l.entries[index].Kind != tar.TypeLink || len(chain) >= pnpmLayoutAliasLimit {
			return errPNPMLayoutArchive
		}
		chain = append(chain, current)
		current = l.entries[index].Link
	}
}

// Entries returns copied, unverified header facts.
func (l PNPMLayoutArchive) Entries() []PNPMLayoutEntry {
	return append([]PNPMLayoutEntry(nil), l.entries...)
}

// RestrictDirectoryWrites creates an unverified inspection view that removes
// group/other directory write bits. It grants no permission, changes no file
// bytes or links, and is not layout/source validation. Native raw headers stay
// in the original value; only the restricted entries may be rendered later.
func (l PNPMLayoutArchive) RestrictDirectoryWrites() PNPMLayoutArchive {
	entries := l.Entries()
	for index := range entries {
		if entries[index].Kind == tar.TypeDir {
			entries[index].Mode &^= 0o022
		}
	}
	return PNPMLayoutArchive{l.body, entries, l.files}
}

// ReadFile copies unverified regular bytes, including a structurally closed
// hardlink alias. Symlinks are not followed, and no Host file is opened.
func (l PNPMLayoutArchive) ReadFile(name string) ([]byte, error) {
	file, exists := l.files[name]
	if !exists {
		return nil, errPNPMLayoutArchive
	}
	return bytes.Clone(l.body[file.start:file.end]), nil
}
