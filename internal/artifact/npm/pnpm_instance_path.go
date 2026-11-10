package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// PNPMVirtualStoreDirMaxLength is the reviewed native layout default. It is a
// format bound, not a package-client version or configurable execution policy.
const PNPMVirtualStoreDirMaxLength = 120

// PNPMInstanceDirectory encodes the reviewed registry-only snapshot key subset
// using pnpm's dep_path_to_filename format. Membership, file bytes and collisions
// must be checked against the complete frozen layout by the caller. This path
// encoding is not artifact identity, source verification or execution authority.
func PNPMInstanceDirectory(key string) (string, error) {
	if _, _, _, err := pnpmPackageKey(key); err != nil {
		return "", err
	}
	filename := strings.ReplaceAll(key, "/", "+")
	if strings.Contains(filename, "(") {
		filename = strings.TrimSuffix(filename, ")")
		filename = strings.ReplaceAll(filename, ")(", "_")
		filename = strings.NewReplacer("(", "_", ")", "_").Replace(filename)
	}
	if len(filename) > PNPMVirtualStoreDirMaxLength || filename != strings.ToLower(filename) {
		sum := sha256.Sum256([]byte(filename))
		digest := hex.EncodeToString(sum[:])[:32]
		return filename[:min(len(filename), PNPMVirtualStoreDirMaxLength-33)] + "_" + digest, nil
	}
	return filename, nil
}
