package releaseprovenance

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"debug/elf"
	"errors"
	"io"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxPNPMArchiveBytes  = 64 << 20
	maxPNPMExpandedBytes = 96 << 20
	maxPNPMToolBytes     = 64 << 20
	pnpmGNUInterpreter   = "/lib64/ld-linux-x86-64.so.2"
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
	if ctx == nil {
		return PNPMToolReceipt{}, errors.New("pnpm tool context is required")
	}
	if err := ctx.Err(); err != nil {
		return PNPMToolReceipt{}, err
	}
	if r.bundleSHA256 == "" || len(compressed) == 0 || len(compressed) > maxPNPMArchiveBytes || uvDigest(compressed) != r.identity.ArchiveSHA256 {
		return PNPMToolReceipt{}, errors.New("pnpm tool lacks the authenticated complete archive")
	}
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return PNPMToolReceipt{}, errors.New("pnpm tool archive compression is invalid")
	}
	expanded, readErr := io.ReadAll(io.LimitReader(z, maxPNPMExpandedBytes+1))
	closeErr := z.Close()
	if readErr != nil || closeErr != nil || len(expanded) > maxPNPMExpandedBytes {
		return PNPMToolReceipt{}, errors.New("pnpm tool archive expansion is invalid or excessive")
	}
	input := bytes.NewReader(expanded)
	archive := tar.NewReader(input)
	seen := map[string]byte{}
	var total int64
	result := PNPMToolReceipt{release: r.identity}
	for {
		if err := ctx.Err(); err != nil {
			return PNPMToolReceipt{}, err
		}
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil || len(seen) >= 1024 {
			return PNPMToolReceipt{}, errors.New("pnpm tool tar is invalid or excessive")
		}
		name := header.Name
		if header.Typeflag == tar.TypeDir {
			name = strings.TrimSuffix(name, "/")
		}
		if !pnpmArchivePath(name) || seen[name] != 0 || header.Size < 0 || header.Size > maxPNPMExpandedBytes-total || header.Uid < 0 || header.Gid < 0 {
			return PNPMToolReceipt{}, errors.New("pnpm tool tar path, identity or size is invalid")
		}
		total += header.Size
		switch header.Typeflag {
		case tar.TypeDir:
			if name == "pnpm" || header.Mode != 0o755 || header.Size != 0 || header.Linkname != "" {
				return PNPMToolReceipt{}, errors.New("pnpm tool directory is unsupported")
			}
		case tar.TypeLink:
			if !strings.HasPrefix(name, "dist/") || !strings.HasPrefix(header.Linkname, "dist/") || !pnpmArchivePath(header.Linkname) || seen[header.Linkname] != tar.TypeReg || header.Size != 0 || !pnpmDataMode(header.Mode) {
				return PNPMToolReceipt{}, errors.New("pnpm bundled data hardlink is invalid")
			}
		case tar.TypeReg:
			if header.Linkname != "" || !pnpmDataMode(header.Mode) || name == "dist" {
				return PNPMToolReceipt{}, errors.New("pnpm tool regular file is invalid")
			}
			if name == "pnpm" {
				if header.Mode != 0o755 || header.Size <= 0 || header.Size > maxPNPMToolBytes {
					return PNPMToolReceipt{}, errors.New("pnpm executable mode or size is invalid")
				}
				body, err := io.ReadAll(io.LimitReader(archive, header.Size+1))
				if err != nil || int64(len(body)) != header.Size {
					return PNPMToolReceipt{}, errors.New("pnpm executable is incomplete")
				}
				libraries, err := pnpmNativeELF(body)
				if err != nil {
					return PNPMToolReceipt{}, err
				}
				result.digest, result.size, result.libraries = uvDigest(body), header.Size, libraries
			}
		default:
			return PNPMToolReceipt{}, errors.New("pnpm tool archive contains an unsupported file type")
		}
		seen[name] = header.Typeflag
	}
	for _, value := range expanded[len(expanded)-input.Len():] {
		if value != 0 {
			return PNPMToolReceipt{}, errors.New("pnpm tool tar has trailing content")
		}
	}
	if result.size == 0 {
		return PNPMToolReceipt{}, errors.New("pnpm native executable is missing")
	}
	if err := ctx.Err(); err != nil {
		return PNPMToolReceipt{}, err
	}
	return result, nil
}

func pnpmArchivePath(name string) bool {
	return len(name) <= 1024 && utf8.ValidString(name) && path.Clean(name) == name && !strings.ContainsAny(name, "\\\x00") && (name == "pnpm" || name == "dist" || strings.HasPrefix(name, "dist/"))
}

func pnpmDataMode(mode int64) bool { return mode == 0o600 || mode == 0o644 || mode == 0o755 }

func pnpmNativeELF(body []byte) ([]string, error) {
	file, err := elf.NewFile(bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("pnpm executable is not a supported ELF")
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS64 || file.Data != elf.ELFDATA2LSB || file.Machine != elf.EM_X86_64 || (file.Type != elf.ET_DYN && file.Type != elf.ET_EXEC) || len(file.Progs) > 128 {
		return nil, errors.New("pnpm executable architecture or type is unsupported")
	}
	loads, interpreters := 0, 0
	for _, p := range file.Progs {
		if p.Off > uint64(len(body)) || p.Filesz > uint64(len(body))-p.Off || p.Type == elf.PT_LOAD && (p.Memsz < p.Filesz || p.Flags&(elf.PF_W|elf.PF_X) == elf.PF_W|elf.PF_X) || p.Type == elf.PT_GNU_STACK && p.Flags&elf.PF_X != 0 {
			return nil, errors.New("pnpm executable program range or permissions are invalid")
		}
		if p.Type == elf.PT_LOAD {
			loads++
		}
		if p.Type == elf.PT_INTERP {
			interpreters++
			if string(body[p.Off:p.Off+p.Filesz]) != pnpmGNUInterpreter+"\x00" {
				return nil, errors.New("pnpm executable loader is unsupported")
			}
		}
	}
	libraries, err := file.ImportedLibraries()
	if err != nil || loads == 0 || interpreters != 1 || len(libraries) == 0 || len(libraries) > 32 {
		return nil, errors.New("pnpm executable dynamic metadata is invalid")
	}
	seen := map[string]bool{}
	for _, library := range libraries {
		if library == "" || len(library) > 255 || !utf8.ValidString(library) || strings.ContainsAny(library, "/\\") || strings.IndexFunc(library, unicode.IsControl) >= 0 || seen[library] {
			return nil, errors.New("pnpm executable library name is invalid")
		}
		seen[library] = true
	}
	if !seen["libc.so.6"] {
		return nil, errors.New("pnpm executable requires the reviewed GNU libc ABI")
	}
	return libraries, nil
}
