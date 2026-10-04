package gomodule

import (
	"errors"
	"strings"

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

// ValidateProjectSums binds the current control declaration to the complete
// selected content pair. This agreement is separate from independent SumDB
// authentication and never makes a project declaration an authority.
func ValidateProjectSums(body []byte, records []DownloadRecord) error {
	if len(body) == 0 || len(body) > MaxProjectControlBytes || len(records) == 0 || len(records) > 4096 {
		return errors.New("go project checksum controls exceed bounds")
	}
	sums := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 {
			return errors.New("go project checksum line is invalid")
		}
		version := strings.TrimSuffix(fields[1], "/go.mod")
		if !validModuleVersion(fields[0], version) {
			return errors.New("go project checksum identity is invalid")
		}
		if _, err := h1Digest(fields[2]); err != nil {
			return errors.New("go project checksum is invalid")
		}
		key := fields[0] + "@" + fields[1]
		if _, exists := sums[key]; exists {
			return errors.New("go project checksum contains duplicate identity")
		}
		sums[key] = fields[2]
	}
	for _, record := range records {
		if !validModuleVersion(record.Path, record.Version) {
			return errors.New("go selected checksum identity is invalid")
		}
		if _, err := h1Digest(record.Sum); err != nil {
			return errors.New("go selected archive checksum is invalid")
		}
		if _, err := h1Digest(record.GoModSum); err != nil {
			return errors.New("go selected module checksum is invalid")
		}
		if sums[recordKey(record.Path, record.Version)] != record.Sum || sums[recordKey(record.Path, record.Version)+"/go.mod"] != record.GoModSum {
			return errors.New("go project checksum pair differs from selected content")
		}
	}
	return nil
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
