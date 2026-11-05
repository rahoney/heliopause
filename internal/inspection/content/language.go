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
	case ".sh", ".bash":
		return "bash"
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
	if command == "sh" || command == "bash" || command == "dash" {
		return "bash"
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

// UnsupportedSource identifies executable/source hints outside the fixed parser
// set. Opaque DATA alone does not become a source target or a safety verdict.
func UnsupportedSource(name string, prefix []byte, mode uint32) bool {
	if SourceLanguage(name, prefix) != "" {
		return false
	}
	if SourceLanguage(strings.ToLower(name), prefix) != "" {
		return true // An unselected case variant is not proof of absent source.
	}
	if bytes.HasPrefix(prefix, []byte("#!")) || mode&0o111 != 0 {
		return true
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".zsh", ".fish", ".ps1", ".bat", ".cmd", ".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".s", ".asm", ".rb", ".pl", ".php", ".lua", ".java":
		return true
	}
	return false
}
