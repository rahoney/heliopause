package cargo

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

const MaxIndexBytes = 8 << 20

// IndexURL retains crate identity but follows Cargo's lowercase index layout.
func IndexURL(name string) (string, error) {
	if !crateNamePattern.MatchString(name) {
		return "", errors.New("cargo index name is invalid")
	}
	name = strings.ToLower(name)
	prefix := ""
	switch len(name) {
	case 1:
		prefix = "1/"
	case 2:
		prefix = "2/"
	case 3:
		prefix = "3/" + name[:1] + "/"
	default:
		prefix = name[:2] + "/" + name[2:4] + "/"
	}
	return "https://index.crates.io/" + prefix + name, nil
}

// ValidateRegistryConfig permits only the official public download endpoint.
// A registry response cannot introduce a new network authority or credential.
func ValidateRegistryConfig(body []byte) error {
	if len(body) > 64<<10 || validateMetadataJSON(body) != nil {
		return errors.New("cargo registry configuration is invalid")
	}
	var config struct {
		Download string `json:"dl"`
		API      string `json:"api"`
		Auth     bool   `json:"auth-required"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&config) != nil || config.Download != crateDownloadHost || (config.API != "" && config.API != "https://crates.io") || config.Auth {
		return errors.New("cargo registry configuration substitutes public source")
	}
	return nil
}

// ParseIndexChecksum reads one exact immutable version from official index
// grammar. Its result remains a declaration until the verifier compares bytes.
func ParseIndexChecksum(body []byte, name, version string) (string, error) {
	if _, err := ParseReference(name + "@" + version); err != nil || len(body) == 0 || len(body) > MaxIndexBytes {
		return "", errors.New("cargo index identity or bytes exceed bounds")
	}
	lines := bytes.Split(body, []byte{'\n'})
	if len(lines) > 16385 {
		return "", errors.New("cargo index version count exceeds bound")
	}
	seen := map[string]bool{}
	checksum := ""
	for index, line := range lines {
		if len(line) == 0 && index == len(lines)-1 {
			continue
		}
		if len(line) == 0 || len(line) > 128<<10 || validateMetadataJSON(line) != nil {
			return "", errors.New("cargo index record is invalid")
		}
		var record struct {
			Name     string `json:"name"`
			Version  string `json:"vers"`
			Checksum string `json:"cksum"`
			Schema   int    `json:"v"`
		}
		if json.Unmarshal(line, &record) != nil || record.Name != name || !crateVersionPattern.MatchString(record.Version) {
			return "", errors.New("cargo index package identity differs")
		}
		// Cargo registry version uniqueness ignores SemVer build metadata.
		base, _, _ := strings.Cut(record.Version, "+")
		if seen[base] {
			return "", errors.New("cargo index contains ambiguous versions")
		}
		seen[base] = true
		if record.Version == version {
			if record.Schema < 0 || record.Schema > 2 || !isSHA256(record.Checksum) || record.Checksum != strings.ToLower(record.Checksum) {
				return "", errors.New("cargo selected index checksum or schema is invalid")
			}
			checksum = record.Checksum
		}
	}
	if checksum == "" {
		return "", errors.New("cargo exact version is absent from public index")
	}
	return checksum, nil
}

func ParseIntegrity(value string) (string, error) {
	checksum, ok := strings.CutPrefix(value, "sha256=")
	if !ok || !isSHA256(checksum) || checksum != strings.ToLower(checksum) {
		return "", errors.New("cargo declared checksum is invalid")
	}
	return checksum, nil
}
