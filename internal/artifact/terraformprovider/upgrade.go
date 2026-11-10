package terraformprovider

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/mod/semver"
)

// Selection uses one frozen bounded list and every current constraint. An
// unsupported newest version fails; it never silently falls back to an older
// platform/protocol. Exact init retains its separate reference semantics.
func selectUpgradeVersion(ctx context.Context, body []byte, constraints []string, platform Platform) (string, error) {
	if ctx == nil {
		return "", errors.New("terraform upgrade context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	for _, constraint := range constraints {
		if len(constraint) > 4096 || strings.Contains(constraint, "-") {
			return "", errors.New("terraform upgrade requires bounded stable-version constraints")
		}
	}
	document, err := parseVersionDocument(body)
	if err != nil {
		return "", err
	}
	selected := ""
	for _, entry := range document.Versions {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if semver.Prerelease("v"+entry.Version) != "" {
			continue
		}
		matches := true
		for _, constraint := range constraints {
			if !MatchesConstraint(entry.Version, constraint) {
				matches = false
				break
			}
		}
		if matches && (selected == "" || semver.Compare("v"+entry.Version, "v"+selected) > 0) {
			selected = entry.Version
		}
	}
	if selected == "" {
		return "", errors.New("terraform upgrade has no stable version satisfying all current constraints")
	}
	if err := ParseVersionResponse(body, selected, platform); err != nil {
		return "", err
	}
	return selected, nil
}
