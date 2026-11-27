package releaseprovenance

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"debug/elf"
	"errors"
	"io"
)

const maxUVToolSupplyBytes = 64 << 20

// UVToolLayer is untrusted compressed content in manifest order. Its size and
// entire bytes must match the already authenticated image descriptors.
type UVToolLayer struct {
	Reader io.Reader
	Size   int64
}

// UVToolReceipt binds the native file to an authenticated image. It grants no
// Policy decision, compatibility claim or permission to execute the file.
type UVToolReceipt struct {
	image  UVImageIdentity
	digest string
	size   int64
}

func (r UVToolReceipt) Image() UVImageIdentity { return r.image }
func (r UVToolReceipt) SHA256() string         { return r.digest }
func (r UVToolReceipt) Size() int64            { return r.size }

// VerifyToolLayers validates every compressed layer, its signed config diffID
// and the complete reviewed distroless layout. Later layers cannot replace or
// whiteout a previously checked tool. The native file is only parsed as data.
func (r UVImageReceipt) VerifyToolLayers(ctx context.Context, config []byte, inputs []UVToolLayer) (UVToolReceipt, error) {
	if ctx == nil {
		return UVToolReceipt{}, errors.New("uv tool context is required")
	}
	if err := ctx.Err(); err != nil {
		return UVToolReceipt{}, err
	}
	if r.bundleSHA256 == "" || len(r.layers) == 0 || len(inputs) != len(r.layers) || len(config) > 64<<10 || uvDigest(config) != r.configSHA256 {
		return UVToolReceipt{}, errors.New("uv tool lacks authenticated image inputs")
	}
	var configuration struct {
		RootFS struct {
			DiffIDs []string `json:"diff_ids"`
		} `json:"rootfs"`
	}
	if decodeUVJSON(config, &configuration) != nil || len(configuration.RootFS.DiffIDs) != len(inputs) {
		return UVToolReceipt{}, errors.New("uv tool layer configuration mismatch")
	}
	seen := map[string]bool{}
	var compressedTotal, expandedTotal int64
	entries := 0
	result := UVToolReceipt{image: r.identity}
	for i, input := range inputs {
		if err := ctx.Err(); err != nil {
			return UVToolReceipt{}, err
		}
		if input.Reader == nil || input.Size != r.layers[i].Size || input.Size <= 0 || compressedTotal > maxUVToolSupplyBytes-input.Size {
			return UVToolReceipt{}, errors.New("uv tool compressed layer budget mismatch")
		}
		compressedTotal += input.Size
		compressed, err := io.ReadAll(io.LimitReader(input.Reader, input.Size+1))
		if err != nil || int64(len(compressed)) != input.Size || uvDigest(compressed) != r.layers[i].SHA256 {
			return UVToolReceipt{}, errors.New("uv tool layer digest mismatch")
		}
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return UVToolReceipt{}, errors.New("uv tool layer compression is invalid")
		}
		expanded, readErr := io.ReadAll(io.LimitReader(reader, maxUVToolSupplyBytes-expandedTotal+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || int64(len(expanded)) > maxUVToolSupplyBytes-expandedTotal || "sha256:"+uvDigest(expanded) != configuration.RootFS.DiffIDs[i] {
			return UVToolReceipt{}, errors.New("uv tool layer expansion or diffID mismatch")
		}
		expandedTotal += int64(len(expanded))
		archive := tar.NewReader(bytes.NewReader(expanded))
		for {
			header, err := archive.Next()
			if err == io.EOF {
				break
			}
			if err != nil || entries >= 32 {
				return UVToolReceipt{}, errors.New("uv tool layer archive is invalid or excessive")
			}
			entries++
			if header.Uid != 0 || header.Gid != 0 || header.Mode != 0o755 || seen[header.Name] || header.Linkname != "" {
				return UVToolReceipt{}, errors.New("uv tool archive ownership or identity mismatch")
			}
			seen[header.Name] = true
			if header.Typeflag == tar.TypeDir && (header.Name == "io" || header.Name == "io/") && header.Size == 0 {
				continue
			}
			if header.Typeflag != tar.TypeReg || (header.Name != "uv" && header.Name != "uvx") || header.Size <= 0 || header.Size > maxUVToolSupplyBytes {
				return UVToolReceipt{}, errors.New("uv tool archive layout is unsupported")
			}
			body, err := io.ReadAll(io.LimitReader(archive, header.Size+1))
			if err != nil || int64(len(body)) != header.Size || !uvStaticELF(body) {
				return UVToolReceipt{}, errors.New("uv tool native file is invalid")
			}
			if header.Name == "uv" {
				result.digest, result.size = uvDigest(body), header.Size
			}
		}
	}
	if !seen["uv"] || !seen["uvx"] || result.size == 0 {
		return UVToolReceipt{}, errors.New("uv tool file set is incomplete")
	}
	if err := ctx.Err(); err != nil {
		return UVToolReceipt{}, err
	}
	return result, nil
}

func uvStaticELF(body []byte) bool {
	file, err := elf.NewFile(bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS64 || file.Data != elf.ELFDATA2LSB || file.Machine != elf.EM_X86_64 || file.Type != elf.ET_EXEC {
		return false
	}
	loads := 0
	for _, program := range file.Progs {
		if program.Type == elf.PT_INTERP || program.Off > uint64(len(body)) || program.Filesz > uint64(len(body))-program.Off || program.Type == elf.PT_LOAD && program.Memsz < program.Filesz {
			return false
		}
		if program.Type == elf.PT_LOAD {
			loads++
		}
	}
	return loads > 0
}
