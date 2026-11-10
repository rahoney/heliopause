package cargo

import (
	"errors"
	"strings"
)

// ParseUpdateSelector accepts the bounded public package-id subset. An empty
// value requests the complete project, never an arbitrary primary crate.
func ParseUpdateSelector(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	name, version, qualified := strings.Cut(value, "@")
	if !crateNamePattern.MatchString(name) || (qualified && !crateVersionPattern.MatchString(version)) {
		return "", errors.New("cargo update selector must be name or name@locked-version")
	}
	return value, nil
}

// ValidateUpdateSelector binds the request to one original public lock entry.
// Lock declarations remain input, not independent checksum approval.
func ValidateUpdateSelector(selector string, lock []byte) error {
	value, err := ParseUpdateSelector(selector)
	if err != nil {
		return err
	}
	packages, err := parseLock(lock)
	if err != nil {
		return err
	}
	if value == "" {
		return nil
	}
	name, version, qualified := strings.Cut(value, "@")
	matches, public := 0, false
	for _, entry := range packages {
		if entry.Name == name && (!qualified || entry.Version == version) {
			matches++
			public = entry.Source == registrySourceURL
		}
	}
	if matches != 1 || !public {
		return errors.New("cargo update selector must identify one locked public crate")
	}
	return nil
}
