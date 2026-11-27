package npm

import (
	"bytes"
	"encoding/json"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func parseBunPackage(key string, body []byte) (BunLockedPackage, error) {
	parts, ok := bunKeyParts(key)
	var row []json.RawMessage
	if !ok || json.Unmarshal(body, &row) != nil || len(row) != 4 {
		return BunLockedPackage{}, errBunLock
	}
	identifier, ok := bunJSONString(row[0])
	separator := strings.LastIndexByte(identifier, '@')
	if !ok || separator <= 0 {
		return BunLockedPackage{}, errBunLock
	}
	name, version := identifier[:separator], identifier[separator+1:]
	registry, registryOK := bunJSONString(row[1])
	integrity, integrityOK := bunJSONString(row[3])
	if name != parts[len(parts)-1] || validatePackageName(name) != nil || !strictBunVersion(version) || !registryOK || registry != "" && !validRegistryTarball(registry) || !integrityOK || !validSHA512SRI(integrity) {
		return BunLockedPackage{}, errBunLock
	}
	info, err := bunJSONObject(row[2], "dependencies", "peerDependencies", "bin", "hasInstallScript")
	if err != nil {
		return BunLockedPackage{}, err
	}
	dependencies, err := bunOptionalConstraints(info, "dependencies")
	if err != nil {
		return BunLockedPackage{}, err
	}
	peers, err := bunOptionalConstraints(info, "peerDependencies")
	if err != nil {
		return BunLockedPackage{}, err
	}
	pkg := BunLockedPackage{Key: key, Name: name, Version: version, Registry: registry, Integrity: integrity, Dependencies: dependencies, PeerDependencies: peers, Bins: map[string]string{}}
	if flag, exists := info["hasInstallScript"]; exists {
		flag = bytes.TrimSpace(flag)
		if !bytes.Equal(flag, []byte("true")) && !bytes.Equal(flag, []byte("false")) {
			return BunLockedPackage{}, errBunLock
		}
		pkg.HasInstallScript = bytes.Equal(flag, []byte("true"))
	}
	if value, exists := info["bin"]; exists {
		pkg.Bins, err = bunPackageBins(value, name)
		if err != nil {
			return BunLockedPackage{}, err
		}
	}
	return pkg, nil
}

func bunPackageBins(body []byte, name string) (map[string]string, error) {
	result := map[string]string{}
	if value, ok := bunJSONString(body); ok {
		result[path.Base(name)] = value
	} else {
		fields, err := bunJSONObject(body)
		if err != nil || len(fields) > ArchiveEntryLimit {
			return nil, errBunLock
		}
		for command, raw := range fields {
			value, ok := bunJSONString(raw)
			if !ok {
				return nil, errBunLock
			}
			result[command] = value
		}
	}
	for command, value := range result {
		value = strings.TrimPrefix(value, "./")
		// Reuse the existing npm archive path and bin character boundaries.
		if len(command) > 214 || command == "." || command == ".." || strings.Contains(command, "/") || !safePNPMBinCharacters(command) || !validInventoryPath(value) || !safePNPMBinCharacters(value) {
			return nil, errBunLock
		}
		result[command] = value
	}
	return result, nil
}

func strictBunVersion(value string) bool {
	if len(value) > 128 {
		return false
	}
	parsed, err := semver.StrictNewVersion(value)
	return err == nil && parsed.String() == value
}

func supportedBunConstraint(spec string) bool {
	if spec != strings.TrimSpace(spec) || !publicDependencySpec.MatchString(spec) {
		return false
	}
	_, err := semver.NewConstraint(spec)
	return err == nil
}

func bunConstraintMatches(spec, version string) bool {
	if !strictBunVersion(version) || !supportedBunConstraint(spec) {
		return false
	}
	parsed, _ := semver.StrictNewVersion(version)
	if parsed.Prerelease() != "" {
		// Do not infer Bun/npm prerelease-range equivalence from this library.
		return spec == version
	}
	constraint, err := semver.NewConstraint(spec)
	return err == nil && constraint.Check(parsed)
}

// A scoped package is one ancestry unit, not two filesystem components.
func bunKeyParts(key string) ([]string, bool) {
	if len(key) == 0 || len(key) > 2048 {
		return nil, false
	}
	segments := strings.Split(key, "/")
	var result []string
	for i := 0; i < len(segments); i++ {
		name := segments[i]
		if strings.HasPrefix(name, "@") {
			i++
			if i == len(segments) {
				return nil, false
			}
			name += "/" + segments[i]
		}
		if validatePackageName(name) != nil || len(result) >= 32 {
			return nil, false
		}
		result = append(result, name)
	}
	return result, true
}
