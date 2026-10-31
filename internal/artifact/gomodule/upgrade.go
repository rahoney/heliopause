package gomodule

import (
	"errors"
	"reflect"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// ValidateUpgradeSelector admits a module path, never a package pattern,
// version query, SDK selector or arbitrary Go command argument.
func ValidateUpgradeSelector(selector string) error {
	if selector == "" || strings.ContainsAny(selector, "@\\: \t\r\n") || strings.Contains(selector, "...") || module.CheckPath(selector) != nil {
		return errors.New("go upgrade requires one exact public module path")
	}
	return nil
}

// ValidateUpgradeMembership prevents native get from silently adding a module.
func ValidateUpgradeMembership(original []byte, selector string) error {
	if err := ValidateUpgradeSelector(selector); err != nil {
		return err
	}
	file, err := parseProjectMod(original)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	found := false
	for _, requirement := range file.Require {
		if seen[requirement.Mod.Path] {
			return errors.New("go upgrade original requirements are ambiguous")
		}
		seen[requirement.Mod.Path] = true
		found = found || requirement.Mod.Path == selector
	}
	if !found {
		return errors.New("go upgrade module is not an existing requirement")
	}
	return nil
}

// FreezeUpgradeReference binds the real requested module to one exact selected
// record and rejects any non-require directive drift. This does not authenticate
// module content; independent acquisition/SumDB verification remain required.
func FreezeUpgradeReference(original, selected []byte, selector string, records []DownloadRecord) (domain.ArtifactReference, error) {
	if err := ValidateUpgradeMembership(original, selector); err != nil {
		return domain.ArtifactReference{}, err
	}
	before, err := parseProjectMod(original)
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	after, err := parseProjectMod(selected)
	if err != nil || !reflect.DeepEqual(nonRequireTokens(before), nonRequireTokens(after)) {
		return domain.ArtifactReference{}, errors.New("go upgrade changed a non-require directive")
	}
	if err := ValidateUpgradeMembership(selected, selector); err != nil {
		return domain.ArtifactReference{}, err
	}
	version := ""
	seen := map[string]bool{}
	for _, record := range records {
		if !validModuleVersion(record.Path, record.Version) || seen[record.Path] {
			return domain.ArtifactReference{}, errors.New("go upgrade selection is invalid or ambiguous")
		}
		seen[record.Path] = true
		if record.Path == selector {
			version = record.Version
		}
	}
	if version == "" {
		return domain.ArtifactReference{}, errors.New("go upgrade selected module is missing")
	}
	return ParseReference(selector + "@" + version)
}

func nonRequireTokens(file *modfile.File) [][]string {
	if file == nil || file.Syntax == nil {
		return nil
	}
	var result [][]string
	for _, statement := range file.Syntax.Stmt {
		switch line := statement.(type) {
		case *modfile.Line:
			if len(line.Token) > 0 && line.Token[0] != "require" {
				result = append(result, append([]string(nil), line.Token...))
			}
		case *modfile.LineBlock:
			if len(line.Token) > 0 && line.Token[0] != "require" {
				result = append(result, append(append([]string(nil), line.Token...), "("))
				for _, member := range line.Line {
					result = append(result, append([]string(nil), member.Token...))
				}
				result = append(result, []string{")"})
			}
		}
	}
	return result
}
