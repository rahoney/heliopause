package content

import "path"

// SourceLanguage selects the parser for a checked member name. It does not
// attest safety, executable role or coverage of files without these suffixes.
func SourceLanguage(name string) string {
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
	default:
		return ""
	}
}
