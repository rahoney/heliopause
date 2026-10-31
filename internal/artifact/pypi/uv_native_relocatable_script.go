package pypi

import (
	"bytes"
	"errors"
)

// ValidateUVRelocatableConsoleScript compares the fixed UV0.13.4 POSIX
// relative-python recipe as DATA. Callers separately bind the original console
// declaration, exact filename, complete source and pinned interpreter alias.
// No native script is modified or executed, and no ownership/Policy is granted.
func ValidateUVRelocatableConsoleScript(ep EntryPoint, body []byte) error {
	const header = `#!/bin/sh
'''exec' "$(dirname -- "$(realpath -- "$0")")"/'python' "$0" "$@"
' '''
`
	if len(body) == 0 || len(body) > 32<<10 || !bytes.HasPrefix(body, []byte(header)) {
		return errors.New("native UV relocatable wrapper differs from its fixed relative-python header")
	}
	// Reuse the existing exact tail and canonical entrypoint grammar. The
	// projected header exists only in private comparison memory.
	projected := append([]byte("#!/usr/local/bin/python\n"), body[len(header):]...)
	return ValidateUVNativeConsoleScript(ep, "/usr/local/bin/python", projected)
}
