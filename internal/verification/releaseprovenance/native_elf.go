package releaseprovenance

import (
	"bytes"
	"debug/elf"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const gnuAMD64Interpreter = "/lib64/ld-linux-x86-64.so.2"

// Shared bounded GNU AMD64 ELF validation. It measures format data only;
// imported library names do not grant Host loader/runtime authority.
func gnuAMD64NativeELF(body []byte) ([]string, error) {
	file, err := elf.NewFile(bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("native executable is not a supported ELF")
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS64 || file.Data != elf.ELFDATA2LSB || file.Machine != elf.EM_X86_64 || (file.Type != elf.ET_DYN && file.Type != elf.ET_EXEC) || len(file.Progs) > 128 {
		return nil, errors.New("native executable architecture or type is unsupported")
	}
	loads, interpreters := 0, 0
	for _, p := range file.Progs {
		if p.Off > uint64(len(body)) || p.Filesz > uint64(len(body))-p.Off || p.Type == elf.PT_LOAD && (p.Memsz < p.Filesz || p.Flags&(elf.PF_W|elf.PF_X) == elf.PF_W|elf.PF_X) || p.Type == elf.PT_GNU_STACK && p.Flags&elf.PF_X != 0 {
			return nil, errors.New("native executable program range or permissions are invalid")
		}
		if p.Type == elf.PT_LOAD {
			loads++
		}
		if p.Type == elf.PT_INTERP {
			interpreters++
			if string(body[p.Off:p.Off+p.Filesz]) != gnuAMD64Interpreter+"\x00" {
				return nil, errors.New("native executable loader is unsupported")
			}
		}
	}
	libraries, err := file.ImportedLibraries()
	if err != nil || loads == 0 || interpreters != 1 || len(libraries) == 0 || len(libraries) > 32 {
		return nil, errors.New("native executable dynamic metadata is invalid")
	}
	seen := map[string]bool{}
	for _, library := range libraries {
		if library == "" || len(library) > 255 || !utf8.ValidString(library) || strings.ContainsAny(library, "/\\") || strings.IndexFunc(library, unicode.IsControl) >= 0 || seen[library] {
			return nil, errors.New("native executable library name is invalid")
		}
		seen[library] = true
	}
	if !seen["libc.so.6"] {
		return nil, errors.New("native executable requires the reviewed GNU libc ABI")
	}
	return libraries, nil
}
