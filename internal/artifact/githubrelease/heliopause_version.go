package githubrelease

import (
	"errors"
	"strings"

	"golang.org/x/mod/semver"
)

// ValidateHeliopauseVersion is an inert exact-version input check shared by
// CLI intent, session construction and frozen acquisition. It performs no I/O.
func ValidateHeliopauseVersion(version string) error {
	core, _, _ := strings.Cut(version, "-")
	core, _, _ = strings.Cut(core, "+")
	if len(version) > 128 || !semver.IsValid(version) || strings.Count(core, ".") != 2 {
		return errors.New("an exact official release version is required")
	}
	return nil
}
