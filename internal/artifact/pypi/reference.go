// Package pypi owns PyPI-specific reference normalization.
package pypi

import (
	"errors"
	"regexp"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var (
	projectNamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?$`)
	separatorPattern   = regexp.MustCompile(`[-_.]+`)
	localVersionSuffix = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`)
	versionPattern     = regexp.MustCompile(`(?i)^(?:v)?(?:(\d+)!)?(\d+(?:\.\d+)*)(?:(?:[-_.]?)(a|b|c|rc|alpha|beta|pre|preview)(?:[-_.]?)(\d*)?)?(?:(?:[-_.]?)(post|rev|r)(?:[-_.]?)(\d*)?|-(\d+))?(?:(?:[-_.]?)(dev)(?:[-_.]?)(\d*)?)?$`)
)

// ParseReference parses one public PyPI project with an optional exact PEP 440
// version. It deliberately excludes extras, specifier ranges, markers, direct
// URLs, local versions and all pip option syntax from the automatic path.
func ParseReference(input string) (domain.ArtifactReference, error) {
	return ParseReferenceForSource(input, PublicPyPIProfile().Source())
}

// ParseReferenceForSource parses a package reference for one canonical PyPI
// or named PyTorch profile. The profile source is part of the Artifact
// identity and cannot be supplied as an arbitrary URL.
func ParseReferenceForSource(input string, source domain.SourceID) (domain.ArtifactReference, error) {
	profile, ok := ProfileForSource(source)
	if !ok {
		return domain.ArtifactReference{}, errors.New("unsupported Python source profile")
	}
	project, version, hasVersion, err := parseReferenceForProfile(input, IsPyTorchSource(profile.source))
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	locator := project
	if hasVersion {
		locator += "@" + version
	}
	return domain.NewArtifactReference(source, locator)
}

// RequestedProject returns the normalized project name from a validated PyPI
// reference without exposing parser internals to Application.
func RequestedProject(reference domain.ArtifactReference) (string, error) {
	project, _, _, err := parsePypiReference(reference)
	return project, err
}

// RequestedVersion returns the normalized exact version when the caller
// supplied one. An absent version remains unresolved until M5-003.
func RequestedVersion(reference domain.ArtifactReference) (string, bool, error) {
	_, version, present, err := parsePypiReference(reference)
	return version, present, err
}

// NormalizeProjectName applies the PyPA normalized-name algorithm to an
// externally supplied project name without constructing an ArtifactReference.
// Resolver metadata uses this helper before it can enter a Domain graph.
func NormalizeProjectName(value string) (string, error) {
	if !projectNamePattern.MatchString(value) {
		return "", errors.New("PyPI project name is invalid")
	}
	return strings.ToLower(separatorPattern.ReplaceAllString(value, "-")), nil
}

// NormalizeVersion accepts one exact public PEP 440 version and returns its
// canonical form. It deliberately excludes local versions and specifiers.
func NormalizeVersion(value string) (string, error) { return normalizeVersion(value) }

// IsFinalVersion reports whether an already canonical public PEP 440 version
// has no pre-release or development-release segment.
func IsFinalVersion(value string) bool {
	canonical, err := normalizeVersion(value)
	return err == nil && !strings.Contains(canonical, "a") && !strings.Contains(canonical, "b") && !strings.Contains(canonical, "rc") && !strings.Contains(canonical, ".dev")
}

func parsePypiReference(reference domain.ArtifactReference) (string, string, bool, error) {
	if _, ok := ProfileForSource(reference.Source()); !ok {
		return "", "", false, errors.New("PyPI project reference is required")
	}
	return parseReferenceForProfile(reference.Locator(), IsPyTorchSource(reference.Source()))
}

func parseReference(input string) (string, string, bool, error) {
	return parseReferenceForProfile(input, false)
}

func parseReferenceForProfile(input string, allowLocal bool) (string, string, bool, error) {
	if input == "" || input != strings.TrimSpace(input) || strings.Count(input, "@") > 1 || strings.ContainsAny(input, "[]<>~=;\\/:?") || !allowLocal && strings.Contains(input, "+") {
		return "", "", false, errors.New("PyPI project reference is invalid")
	}
	project, version, hasVersion := input, "", false
	if at := strings.LastIndexByte(input, '@'); at >= 0 {
		project, version, hasVersion = input[:at], input[at+1:], true
	}
	normalizedProject, err := NormalizeProjectName(project)
	if err != nil {
		return "", "", false, err
	}
	project = normalizedProject
	if !hasVersion {
		return project, "", false, nil
	}
	canonical, err := normalizeVersionForProfile(version, allowLocal)
	if err != nil {
		return "", "", false, err
	}
	return project, canonical, true, nil
}

func normalizeVersionForProfile(value string, allowLocal bool) (string, error) {
	if !allowLocal || !strings.Contains(value, "+") {
		return normalizeVersion(value)
	}
	if strings.Count(value, "+") != 1 {
		return "", errors.New("Python local version is invalid")
	}
	parts := strings.SplitN(value, "+", 2)
	base, err := normalizeVersion(parts[0])
	if err != nil || parts[1] == "" || !localVersionSuffix.MatchString(parts[1]) {
		return "", errors.New("PyTorch local version is invalid")
	}
	return base + "+" + strings.ToLower(parts[1]), nil
}

func normalizeVersion(value string) (string, error) {
	if value == "" || value != strings.TrimSpace(value) || strings.Contains(value, "+") {
		return "", errors.New("PyPI version must be an exact public PEP 440 version")
	}
	matches := versionPattern.FindStringSubmatch(value)
	if matches == nil {
		return "", errors.New("PyPI version must be an exact public PEP 440 version")
	}
	epoch := canonicalNumber(matches[1])
	releaseParts := strings.Split(matches[2], ".")
	for index, part := range releaseParts {
		releaseParts[index] = canonicalNumber(part)
	}
	canonical := ""
	if epoch != "" && epoch != "0" {
		canonical += epoch + "!"
	}
	canonical += strings.Join(releaseParts, ".")
	if label := strings.ToLower(matches[3]); label != "" {
		switch label {
		case "alpha":
			label = "a"
		case "beta":
			label = "b"
		case "c", "pre", "preview":
			label = "rc"
		}
		canonical += label + canonicalOptionalNumber(matches[4])
	}
	if label := strings.ToLower(matches[5]); label != "" {
		canonical += ".post" + canonicalOptionalNumber(matches[6])
	} else if implicitPost := matches[7]; implicitPost != "" {
		canonical += ".post" + canonicalNumber(implicitPost)
	}
	if matches[8] != "" {
		canonical += ".dev" + canonicalOptionalNumber(matches[9])
	}
	return canonical, nil
}

func canonicalOptionalNumber(value string) string {
	if value == "" {
		return "0"
	}
	return canonicalNumber(value)
}

func canonicalNumber(value string) string {
	value = strings.TrimLeft(value, "0")
	if value == "" {
		return "0"
	}
	return value
}
