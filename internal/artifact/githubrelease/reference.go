// Package githubrelease contains the GitHub Releases adapter boundary. GitHub
// REST payloads and selectors are normalized here before any common Artifact
// contract is constructed.
package githubrelease

import (
	"errors"
	"strings"
	"unicode"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	SourceName = "github-release"
	APIBaseURL = "https://api.github.com"
	APIVersion = "2026-03-10"
)

// Selector is the exact public GitHub Release asset requested by a caller.
// It intentionally has no latest/release-ID/direct-URL form.
type Selector struct {
	owner string
	repo  string
	tag   string
	asset string
}

func (s Selector) Owner() string { return s.owner }
func (s Selector) Repo() string  { return s.repo }
func (s Selector) Tag() string   { return s.tag }
func (s Selector) Asset() string { return s.asset }
func (s Selector) Locator() string {
	return s.owner + "/" + s.repo + "@" + s.tag + "#" + s.asset
}

// ParseReference accepts only owner/repo@exact-tag#exact-asset. It returns a
// normalized common reference without retaining raw user input.
func ParseReference(input string) (domain.ArtifactReference, error) {
	selector, err := ParseSelector(input)
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	source, err := domain.NewSourceID(SourceName)
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	return domain.NewArtifactReference(source, selector.Locator())
}

// ParseSelector returns a bounded selector suitable for constructing only the
// documented public GitHub Releases API request.
func ParseSelector(input string) (Selector, error) {
	if input == "" || input != strings.TrimSpace(input) || len(input) > 512 {
		return Selector{}, errors.New("GitHub Release reference is invalid")
	}
	at := strings.IndexByte(input, '@')
	hash := strings.LastIndexByte(input, '#')
	if at <= 0 || hash <= at+1 || hash == len(input)-1 || strings.Count(input, "@") != 1 || strings.Count(input, "#") != 1 {
		return Selector{}, errors.New("GitHub Release reference is invalid")
	}
	repository, tag, asset := input[:at], input[at+1:hash], input[hash+1:]
	if strings.Count(repository, "/") != 1 {
		return Selector{}, errors.New("GitHub Release reference is invalid")
	}
	parts := strings.Split(repository, "/")
	if !validOwner(parts[0]) || !validRepository(parts[1]) || !validTag(tag) || !validAsset(asset) {
		return Selector{}, errors.New("GitHub Release reference is invalid")
	}
	return Selector{owner: strings.ToLower(parts[0]), repo: strings.ToLower(parts[1]), tag: tag, asset: asset}, nil
}

func validOwner(value string) bool {
	if len(value) == 0 || len(value) > 39 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, character := range value {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) && character != '-' {
			return false
		}
	}
	return true
}

func validRepository(value string) bool {
	if len(value) == 0 || len(value) > 100 || value[0] == '.' || value[len(value)-1] == '.' {
		return false
	}
	for _, character := range value {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) && character != '-' && character != '_' && character != '.' {
			return false
		}
	}
	return true
}

func validTag(value string) bool {
	if len(value) == 0 || len(value) > 128 || strings.EqualFold(value, "latest") || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "//") || strings.Contains(value, "..") {
		return false
	}
	for _, character := range value {
		if character <= 0x20 || character == 0x7f || character == '@' || character == '#' || character == '?' || character == '\\' {
			return false
		}
	}
	return true
}

func validAsset(value string) bool {
	if len(value) == 0 || len(value) > 255 || value == "." || value == ".." || strings.Contains(value, "..") {
		return false
	}
	for _, character := range value {
		if character <= 0x20 || character == 0x7f || character == '/' || character == '\\' || character == '@' || character == '#' || character == '?' {
			return false
		}
	}
	return true
}
