package gomodule

import (
	"errors"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const MaxProjectControlBytes = 4 << 20

// ValidateProjectMod rejects project directives that can redirect the public
// resolver to local files or a different module source. Parsing uses Go's
// module grammar, including quoted paths and block directives.
func ValidateProjectMod(body []byte) error {
	_, err := parseProjectMod(body)
	return err
}

func parseProjectMod(body []byte) (*modfile.File, error) {
	if len(body) == 0 || len(body) > MaxProjectControlBytes {
		return nil, errors.New("go project go.mod exceeds bound")
	}
	file, err := modfile.Parse("go.mod", body, nil)
	// The main module is local project identity, not an acquired public module.
	// Go permits names without a domain here; dependency paths remain public.
	if err != nil || file.Module == nil || module.CheckImportPath(file.Module.Mod.Path) != nil {
		return nil, errors.New("go project go.mod is invalid")
	}
	if len(file.Replace) != 0 {
		return nil, errors.New("go module replacement sources are unsupported")
	}
	for _, requirement := range file.Require {
		if !validModuleVersion(requirement.Mod.Path, requirement.Mod.Version) {
			return nil, errors.New("go project module requirement is invalid")
		}
	}
	return file, nil
}
