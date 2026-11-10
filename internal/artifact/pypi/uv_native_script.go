package pypi

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// ValidateUVNativeConsoleScript compares DATA with the reviewed POSIX,
// nonrelocatable UV wrapper recipe. It performs no IO or execution and grants
// no ownership: callers must authenticate the old wheel/entrypoint name and
// bind the interpreter, destination and complete original inventory separately.
func ValidateUVNativeConsoleScript(ep EntryPoint, python string, body []byte) error {
	if len(body) == 0 || len(body) > 32<<10 || len(ep.Module) > 512 || len(ep.Attr) > 512 ||
		ep.Group != "console_scripts" || len(ep.Extras) != 0 || !validToolCommand(ep.Name) ||
		!validToolAbsolutePath(python) || !validPythonImportName(ep.Module) || ep.Attr == "" {
		return errors.New("native UV console wrapper input is unsupported or exceeds bounds")
	}
	for _, component := range strings.Split(ep.Attr, ".") {
		if !validPythonImportComponent(component) {
			return errors.New("native UV console wrapper callable is invalid")
		}
	}
	shebang := "#!" + python
	// Pinned uv-install-wheel counts UTF-8 bytes, not Unicode codepoints.
	if len(python)+3 > 127 || strings.Contains(python, " ") {
		quoted := strings.ReplaceAll(python, "'", `'"'"'`)
		shebang = "#!/bin/sh\n'''exec' '" + quoted + "' \"$0\" \"$@\"\n' '''"
	}
	importName, _, _ := strings.Cut(ep.Attr, ".")
	expected := fmt.Sprintf(`%s
# -*- coding: utf-8 -*-
import sys
from %s import %s
if __name__ == "__main__":
    if sys.argv[0].endswith("-script.pyw"):
        sys.argv[0] = sys.argv[0][:-11]
    elif sys.argv[0].endswith(".exe"):
        sys.argv[0] = sys.argv[0][:-4]
    sys.exit(%s())
`, shebang, ep.Module, importName, ep.Attr)
	if !bytes.Equal(body, []byte(expected)) {
		return errors.New("native UV console wrapper differs from its exact entrypoint/runtime recipe")
	}
	return nil
}
