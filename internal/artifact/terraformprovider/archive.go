package terraformprovider

import (
	"bytes"
	"context"
	"debug/elf"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"
)

const providerExpandedLimit = 200 << 20

// PackageContents contains validated, flat provider installation bytes. It
// conveys no signer or Policy approval and never executes archive content.
type PackageContents struct {
	Files            map[string][]byte
	Executable       string
	ExecutableDigest string
	H1               string
	ZH               string
}

// InspectPackage reads every ZIP member through its CRC-checked bounded reader,
// then calculates Terraform's standard content hash and the ELF identity.
func InspectPackage(ctx context.Context, bundle Bundle, reference string) (PackageContents, error) {
	ref, err := ParseReference(reference)
	if err != nil || ctx == nil || ctx.Err() != nil {
		return PackageContents{}, errors.New("terraform package inspection request is invalid")
	}
	nameVersion := strings.Split(ref.Locator(), "@")
	provider := strings.Split(nameVersion[0], "/")[1]
	executableName := regexp.MustCompile(`^terraform-provider-` + regexp.QuoteMeta(provider) + `_v` + regexp.QuoteMeta(nameVersion[1]) + `(?:_x[1-9][0-9]?)?$`)
	reader, err := bundle.ZipReader()
	if err != nil || len(reader.File) == 0 || len(reader.File) > 10000 {
		return PackageContents{}, errors.New("terraform provider ZIP is invalid or exceeds entry bounds")
	}
	result := PackageContents{Files: map[string][]byte{}, ZH: "zh:" + bundle.ArchiveDigest()}
	seen := map[string]bool{}
	names := make([]string, 0, len(reader.File))
	var total uint64
	for _, entry := range reader.File {
		if ctx.Err() != nil {
			return PackageContents{}, ctx.Err()
		}
		name, mode := entry.Name, entry.Mode()
		// Terraform's supported package layout is flat. No filesystem alias,
		// directory, special file, encrypted member or permission ambiguity.
		if name == "" || len(name) > 255 || strings.ContainsAny(name, "/\\:\x00\r\n") || name == "." || name == ".." || strings.HasSuffix(name, ".") || strings.TrimSpace(name) != name || !mode.IsRegular() || mode.Type() != 0 || (mode.Perm() != 0o644 && mode.Perm() != 0o755) || mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || entry.Flags&1 != 0 || seen[strings.ToLower(name)] || entry.UncompressedSize64 > MaxProviderArchiveBytes || entry.UncompressedSize64 > providerExpandedLimit-total {
			return PackageContents{}, errors.New("terraform provider ZIP member is unsafe or exceeds bounds")
		}
		seen[strings.ToLower(name)] = true
		total += entry.UncompressedSize64
		stream, err := entry.Open()
		if err != nil {
			return PackageContents{}, errors.New("open terraform ZIP member")
		}
		body, readErr := io.ReadAll(io.LimitReader(stream, int64(entry.UncompressedSize64)+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || uint64(len(body)) != entry.UncompressedSize64 {
			return PackageContents{}, errors.New("terraform ZIP member size or CRC mismatch")
		}
		isELF := bytes.HasPrefix(body, []byte{0x7f, 'E', 'L', 'F'})
		if executableName.MatchString(name) {
			if result.Executable != "" || mode.Perm() != 0o755 || !isELF || inspectProviderELF(body) != nil {
				return PackageContents{}, errors.New("terraform provider executable is ambiguous or unsupported")
			}
			result.Executable = name
			result.ExecutableDigest = sha256Hex(body)
		} else if mode.Perm()&0o111 != 0 || isELF || strings.HasPrefix(name, "terraform-provider-") {
			return PackageContents{}, errors.New("terraform provider package has an unexpected execution surface")
		}
		result.Files[name] = body
		names = append(names, name)
	}
	if result.Executable == "" {
		return PackageContents{}, errors.New("terraform provider executable is missing")
	}
	result.H1, err = dirhash.Hash1(names, func(name string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(result.Files[name])), nil
	})
	if err != nil {
		return PackageContents{}, errors.New("hash terraform provider content")
	}
	return result, nil
}

func inspectProviderELF(body []byte) error {
	if len(body) < 64 || body[4] != 2 || body[5] != 1 || body[6] != 1 {
		return errors.New("unsupported provider ELF")
	}
	file, err := elf.NewFile(bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS64 || file.Data != elf.ELFDATA2LSB || file.Machine != elf.EM_X86_64 || (file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN) || len(file.Progs) == 0 || len(file.Progs) > 128 || len(file.Sections) > 4096 {
		return errors.New("unsupported provider ELF platform or layout")
	}
	for _, program := range file.Progs {
		if program.Off > uint64(len(body)) || program.Filesz > uint64(len(body))-program.Off || program.Filesz > program.Memsz {
			return errors.New("invalid provider ELF program bounds")
		}
	}
	return nil
}
