package releaseprovenance

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

const (
	maxPNPMArchiveBytes  = 64 << 20
	maxPNPMExpandedBytes = 96 << 20
	maxPNPMToolBytes     = 64 << 20
)

// PNPMToolReceipt binds one native file to an authenticated archive. Declared
// shared libraries are observations, not an ABI or execution qualification.
type PNPMToolReceipt struct {
	release   PNPMReleaseIdentity
	digest    string
	size      int64
	libraries []string
}

func (r PNPMToolReceipt) Release() PNPMReleaseIdentity { return r.release }
func (r PNPMToolReceipt) SHA256() string               { return r.digest }
func (r PNPMToolReceipt) Size() int64                  { return r.size }
func (r PNPMToolReceipt) Libraries() []string          { return append([]string(nil), r.libraries...) }

// VerifyToolArchive authenticates the entire archive before parsing its bounded
// layout. Only pnpm's regular-file bytes contribute to the native identity.
// Bundled build data and internal hardlinks are validated and discarded; no
// path is extracted, linked, executed or granted Host ownership.
func (r PNPMReleaseReceipt) VerifyToolArchive(ctx context.Context, compressed []byte) (PNPMToolReceipt, error) {
	receipt, _, err := r.ReadToolArchive(ctx, compressed)
	return receipt, err
}

// ReadToolArchive returns only the native file after the complete authenticated
// archive, including all trailing entries, has passed the same bounded parser.
// It performs no filesystem operation or execution.
func (r PNPMReleaseReceipt) ReadToolArchive(ctx context.Context, compressed []byte) (PNPMToolReceipt, []byte, error) {
	if ctx == nil {
		return PNPMToolReceipt{}, nil, errors.New("pnpm tool context is required")
	}
	if err := ctx.Err(); err != nil {
		return PNPMToolReceipt{}, nil, err
	}
	if r.bundleSHA256 == "" || len(compressed) == 0 || len(compressed) > maxPNPMArchiveBytes || uvDigest(compressed) != r.identity.ArchiveSHA256 {
		return PNPMToolReceipt{}, nil, errors.New("pnpm tool lacks the authenticated complete archive")
	}
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return PNPMToolReceipt{}, nil, errors.New("pnpm tool archive compression is invalid")
	}
	expanded, readErr := io.ReadAll(io.LimitReader(z, maxPNPMExpandedBytes+1))
	closeErr := z.Close()
	if readErr != nil || closeErr != nil || len(expanded) > maxPNPMExpandedBytes {
		return PNPMToolReceipt{}, nil, errors.New("pnpm tool archive expansion is invalid or excessive")
	}
	input := bytes.NewReader(expanded)
	archive := tar.NewReader(input)
	seen := map[string]byte{}
	var total int64
	result := PNPMToolReceipt{release: r.identity}
	var native []byte
	for {
		if err := ctx.Err(); err != nil {
			return PNPMToolReceipt{}, nil, err
		}
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil || len(seen) >= 1024 {
			return PNPMToolReceipt{}, nil, errors.New("pnpm tool tar is invalid or excessive")
		}
		name := header.Name
		if header.Typeflag == tar.TypeDir {
			name = strings.TrimSuffix(name, "/")
		}
		if !pnpmArchivePath(name) || seen[name] != 0 || header.Size < 0 || header.Size > maxPNPMExpandedBytes-total || header.Uid < 0 || header.Gid < 0 {
			return PNPMToolReceipt{}, nil, errors.New("pnpm tool tar path, identity or size is invalid")
		}
		total += header.Size
		switch header.Typeflag {
		case tar.TypeDir:
			if name == "pnpm" || header.Mode != 0o755 || header.Size != 0 || header.Linkname != "" {
				return PNPMToolReceipt{}, nil, errors.New("pnpm tool directory is unsupported")
			}
		case tar.TypeLink:
			if !strings.HasPrefix(name, "dist/") || !strings.HasPrefix(header.Linkname, "dist/") || !pnpmArchivePath(header.Linkname) || seen[header.Linkname] != tar.TypeReg || header.Size != 0 || !pnpmDataMode(header.Mode) {
				return PNPMToolReceipt{}, nil, errors.New("pnpm bundled data hardlink is invalid")
			}
		case tar.TypeReg:
			if header.Linkname != "" || !pnpmDataMode(header.Mode) || name == "dist" {
				return PNPMToolReceipt{}, nil, errors.New("pnpm tool regular file is invalid")
			}
			if name == "pnpm" {
				if header.Mode != 0o755 || header.Size <= 0 || header.Size > maxPNPMToolBytes {
					return PNPMToolReceipt{}, nil, errors.New("pnpm executable mode or size is invalid")
				}
				body, err := io.ReadAll(io.LimitReader(archive, header.Size+1))
				if err != nil || int64(len(body)) != header.Size {
					return PNPMToolReceipt{}, nil, errors.New("pnpm executable is incomplete")
				}
				libraries, err := gnuAMD64NativeELF(body)
				if err != nil {
					return PNPMToolReceipt{}, nil, err
				}
				result.digest, result.size, result.libraries = uvDigest(body), header.Size, libraries
				native = body
			}
		default:
			return PNPMToolReceipt{}, nil, errors.New("pnpm tool archive contains an unsupported file type")
		}
		seen[name] = header.Typeflag
	}
	for _, value := range expanded[len(expanded)-input.Len():] {
		if value != 0 {
			return PNPMToolReceipt{}, nil, errors.New("pnpm tool tar has trailing content")
		}
	}
	if result.size == 0 {
		return PNPMToolReceipt{}, nil, errors.New("pnpm native executable is missing")
	}
	if err := ctx.Err(); err != nil {
		return PNPMToolReceipt{}, nil, err
	}
	return result, native, nil
}

func pnpmArchivePath(name string) bool {
	return len(name) <= 1024 && utf8.ValidString(name) && path.Clean(name) == name && !strings.ContainsAny(name, "\\\x00") && (name == "pnpm" || name == "dist" || strings.HasPrefix(name, "dist/"))
}

func pnpmDataMode(mode int64) bool { return mode == 0o600 || mode == 0o644 || mode == 0o755 }
