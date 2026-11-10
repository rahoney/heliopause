package pypi

import (
	"bytes"
	"errors"
	"strings"
)

// PipxNativeUVScriptRecordBody reconstructs only the candidate UV bytes named
// by the original RECORD. Pinned pipx app exposure adds -E to an eligible short
// Python shebang without updating that RECORD. This inert conversion grants no
// authority: the caller must independently authenticate the old wheel/entrypoint,
// validate the complete candidate with ValidateUVNativeConsoleScript, and bind
// the held interpreter, actual snapshot, RECORD and exposed command census.
func PipxNativeUVScriptRecordBody(python string, actual []byte) ([]byte, error) {
	if !validToolAbsolutePath(python) || len(actual) == 0 || len(actual) > 32<<10 {
		return nil, errors.New("native pipx UV script input is invalid or exceeds bounds")
	}
	prefix := []byte("#!" + python)
	// These are the exact Linux conditions in the fixed pipx exposure helper.
	if strings.Contains(strings.ToLower(python), "python") && !strings.ContainsAny(python, " \t") && len(prefix)+1+3 <= 127 {
		first, rest, found := bytes.Cut(actual, []byte("\n"))
		if !found || !bytes.Equal(first, append(bytes.Clone(prefix), []byte(" -E")...)) {
			return nil, errors.New("native pipx short script lacks its exact fixed -E suffix")
		}
		return append(append(bytes.Clone(prefix), '\n'), rest...), nil
	}
	return bytes.Clone(actual), nil
}
