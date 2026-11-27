package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/artifact/pypi"
)

// pipxNotices preserves the reviewed distribution license texts, never similarly
// named Python modules. The archive must come from the typed closure generator.
func pipxNotices(closure pypi.PipxSupplyClosure, body []byte) ([]byte, error) {
	if !closure.Valid() || len(body) == 0 || len(body) > 8<<20 {
		return nil, errors.New("pipx notices require bounded closure data")
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	members := map[string]*zip.File{}
	for _, file := range archive.File {
		if members[file.Name] != nil {
			return nil, errors.New("pipx notice archive duplicates a member")
		}
		members[file.Name] = file
	}
	var result bytes.Buffer
	result.WriteString("Pipx wheel closure notices\n")
	for _, wheel := range closure.Wheels() {
		prefix := wheel.Project + "-" + wheel.Version + ".dist-info/licenses/"
		count := 0
		for _, fact := range closure.Files() {
			if fact.Project != wheel.Project || fact.Scheme != pypi.SchemeDistInfo || !strings.HasPrefix(fact.Path, prefix) {
				continue
			}
			if fact.Size <= 0 || fact.Size > 64<<10 {
				return nil, errors.New("pipx license size exceeds bound")
			}
			member := members[fact.Path]
			if member == nil || member.UncompressedSize64 != uint64(fact.Size) {
				return nil, errors.New("pipx license member differs")
			}
			reader, err := member.Open()
			if err != nil {
				return nil, err
			}
			text, readErr := io.ReadAll(io.LimitReader(reader, fact.Size+1))
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil || int64(len(text)) != fact.Size || hash(text) != fact.SHA256 || !utf8.Valid(text) {
				return nil, errors.New("pipx license bytes changed or are invalid")
			}
			for _, r := range string(text) {
				if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
					return nil, errors.New("pipx license contains unsupported control text")
				}
			}
			fmt.Fprintf(&result, "\n=== %s %s: %s ===\n\n", wheel.Project, wheel.Version, fact.Path)
			result.Write(text)
			if !bytes.HasSuffix(text, []byte{'\n'}) {
				result.WriteByte('\n')
			}
			count++
			if result.Len() > 64<<10 {
				return nil, errors.New("pipx combined notices exceed bound")
			}
		}
		if count == 0 {
			return nil, errors.New("pipx distribution has no reviewed license member")
		}
	}
	return result.Bytes(), nil
}
