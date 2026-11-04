package content

import (
	"bytes"
	"path"
	"strings"
)

// SourceLanguage selects a parser from a checked name or bounded interpreter
// hint. It does not attest safety, executable role or unrecognized coverage.
func SourceLanguage(name string, prefixes ...[]byte) string {
	switch path.Ext(name) {
	case ".js", ".mjs", ".cjs", ".jsx":
		return "javascript"
	case ".ts", ".mts", ".cts", ".tsx":
		return "typescript"
	case ".py", ".pyi":
		return "python"
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".tf", ".hcl":
		return "hcl"
	}
	if len(prefixes) != 1 {
		return ""
	}
	prefix := prefixes[0][:min(len(prefixes[0]), 512)]
	if !bytes.HasPrefix(prefix, []byte("#!")) {
		return ""
	}
	if end := bytes.IndexAny(prefix, "\r\n"); end >= 0 {
		prefix = prefix[:end]
	}
	args := strings.Fields(string(prefix[2:]))
	if len(args) == 0 {
		return ""
	}
	command := path.Base(args[0])
	if command == "env" {
		args = args[1:]
		if len(args) > 0 && args[0] == "-S" {
			args = args[1:]
		}
		if len(args) == 0 {
			return ""
		}
		command = path.Base(args[0])
	}
	if command == "node" || command == "nodejs" {
		return "javascript"
	}
	if suffix, ok := strings.CutPrefix(command, "python"); ok {
		for index, character := range suffix {
			if character < '0' || character > '9' {
				if character != '.' || index == 0 {
					return ""
				}
			}
		}
		return "python"
	}
	return ""
}
