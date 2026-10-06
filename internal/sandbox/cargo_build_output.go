package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	projectoutput "github.com/rahoney/heliopause/internal/artifact/projectbuild"
)

// Archive metadata is untrusted diagnostic data. Fixed scalar fields explain
// rejection without publishing names, link targets or granting permission.
type cargoOutputArchiveFailure struct {
	member   int
	kind     byte
	uid, gid int
	mode     int64
	link     bool
	pax      int
	rootDot  bool
}

func (e *cargoOutputArchiveFailure) Error() string {
	return fmt.Sprintf("ARCHIVE_BOUNDARY member=%d kind=%d uid=%d gid=%d mode=%o link=%t pax=%d root_dot=%t", e.member, e.kind, e.uid, e.gid, e.mode, e.link, e.pax, e.rootDot)
}

func readCargoBuildSourceFiles(ctx context.Context, input *goBuildInputFile, files map[string][]byte) error {
	if input == nil || input.prefix != "project" || input.limits.variant != "cargo-source" || ctx == nil || files == nil {
		return errors.New("cargo build source reader is invalid")
	}
	manifest := &goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}, storage: 16 << 20}
	if err := input.scan(ctx, manifest, nil); err != nil {
		return err
	}
	if _, err := input.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	hash := sha256.New()
	bounded := &io.LimitedReader{R: io.TeeReader(input.file, hash), N: int64(input.artifact.SizeBytes()) + 1}
	archive := tar.NewReader(bounded)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		// Only bounded control candidates are retained in memory. Selected
		// metadata paths are independently confined by BuildProjectSnapshot.
		if header.Typeflag == tar.TypeReg && (header.Name == "Cargo.lock" || path.Base(header.Name) == "Cargo.toml") && header.Size <= artifactcargo.MaxProjectControlBytes {
			body, err := io.ReadAll(io.LimitReader(archive, header.Size+1))
			if err != nil || int64(len(body)) != header.Size {
				return errors.New("cargo build manifest read is incomplete")
			}
			files[header.Name] = body
		}
	}
	if _, err := io.Copy(io.Discard, bounded); err != nil || bounded.N != 1 || hex.EncodeToString(hash.Sum(nil)) != input.artifact.Digest().String() {
		return errors.New("cargo build source changed during manifest read")
	}
	return nil
}

// filterCargoBuildOutputs consumes the entire bounded private debug archive.
// Archive-local hardlink aliases are resolved only to regular data in that same
// inventory. No link or untrusted destination is materialized on the host.
func filterCargoBuildOutputs(ctx context.Context, input io.Reader, output io.Writer) error {
	if ctx == nil || input == nil || output == nil {
		return errors.New("cargo output transport is invalid")
	}
	type member struct {
		header  *tar.Header
		ordinal int
		data    []byte
		target  string
	}
	boundary := func(m *member) error {
		h := m.header
		return &cargoOutputArchiveFailure{member: m.ordinal, kind: h.Typeflag, uid: h.Uid, gid: h.Gid, mode: h.Mode, link: h.Linkname != "", pax: len(h.PAXRecords), rootDot: h.Name == "." || h.Name == "./"}
	}
	bounded := &io.LimitedReader{R: input, N: artifactcargo.MaxCrateExpandedBytes + (4*artifactcargo.MaxCrateFiles+16)*512 + 1}
	archive := tar.NewReader(bounded)
	members := map[string]*member{}
	var ordered []*member
	var selected []string
	count, files := 0, 0
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.New("cargo output archive is incomplete")
		}
		count++
		m := &member{header: h, ordinal: count}
		name := strings.TrimSuffix(strings.TrimPrefix(h.Name, "./"), "/")
		rootDot := h.Name == "." || h.Name == "./"
		if rootDot {
			name = "."
		}
		if count > 2*artifactcargo.MaxCrateFiles || (name == "." && (!rootDot || count != 1 || h.Typeflag != tar.TypeDir || h.Size != 0)) ||
			(name != "." && !validClosureDestination(name)) || len(name) > 4096 || members[name] != nil || h.Uid != 1000 || h.Gid != 1000 ||
			h.Mode&^int64(0o777) != 0 || h.Mode&0o022 != 0 || len(h.PAXRecords) != 0 {
			return boundary(m)
		}
		members[name] = m
		ordered = append(ordered, m)
		if h.Typeflag == tar.TypeDir {
			if h.Size != 0 || h.Linkname != "" {
				return boundary(m)
			}
			continue
		}
		files++
		if files > artifactcargo.MaxCrateFiles {
			return errors.New("cargo output file count exceeds bound")
		}
		switch h.Typeflag {
		case tar.TypeReg:
			if h.Linkname != "" || h.Size < 0 || h.Size > artifactcargo.MaxCrateFileBytes || h.Size > artifactcargo.MaxCrateExpandedBytes-total {
				return boundary(m)
			}
			total += h.Size
			m.data, err = io.ReadAll(io.LimitReader(archive, h.Size+1))
			if err != nil || int64(len(m.data)) != h.Size {
				return errors.New("cargo output product is incomplete")
			}
		case tar.TypeLink:
			m.target = strings.TrimPrefix(h.Linkname, "./")
			if h.Size != 0 || !validClosureDestination(m.target) || len(m.target) > 4096 {
				return boundary(m)
			}
		default:
			return boundary(m)
		}
		if !strings.Contains(name, "/") && !strings.HasPrefix(name, ".") && !strings.HasSuffix(name, ".d") {
			selected = append(selected, name)
			if len(selected) > projectoutput.MaxBuildOutputFiles {
				return errors.New("cargo output selection exceeds bound")
			}
		}
	}
	tail, err := io.ReadAll(io.LimitReader(bounded, 4097))
	if err != nil || len(tail) > 4096 || bounded.N <= 0 || !bytes.Equal(tail, make([]byte, len(tail))) {
		return errors.New("cargo output archive has trailing data")
	}
	// Resolve every alias, including unselected entries, after the complete
	// inventory is validated. Reject cycles/chains, metadata drift and expansion
	// amplification. The sole backing object must be regular archive data.
	for _, m := range ordered {
		if m.header.Typeflag != tar.TypeLink {
			continue
		}
		target := members[m.target]
		if target == nil || target.header.Typeflag != tar.TypeReg || target.header.Mode != m.header.Mode ||
			target.header.Uid != m.header.Uid || target.header.Gid != m.header.Gid || int64(len(target.data)) > artifactcargo.MaxCrateExpandedBytes-total {
			return boundary(m)
		}
		total += int64(len(target.data))
		m.data = target.data
	}
	if len(selected) == 0 {
		return errors.New("cargo build retained no default-target output")
	}
	writer := tar.NewWriter(output)
	for _, name := range selected {
		if err := ctx.Err(); err != nil {
			return err
		}
		m := members[name]
		if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o500, Uid: 1000, Gid: 1000, Size: int64(len(m.data)), Format: tar.FormatUSTAR}); err != nil {
			return errors.New("cargo output product metadata cannot be reconstructed")
		}
		if n, err := writer.Write(m.data); err != nil || n != len(m.data) {
			return errors.New("cargo output product is incomplete")
		}
	}
	return writer.Close()
}
