package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Archive inspection bounds are shared by the parser and its optional inventory.
const (
	ArchiveEntryLimit    = 10000
	ArchiveFileLimit     = 20 << 20
	ArchiveDecodedLimit  = 200 << 20
	ArchiveManifestLimit = 1 << 20
)

// ArchiveEntry describes an original package-relative tar entry. Directories
// have zero size and digest. Mode is the original permission bits, not an
// instruction to grant those permissions on the Host.
type ArchiveEntry struct {
	Path      string
	Directory bool
	Size      int64
	Mode      int64
	Digest    domain.ContentDigest
}

// ArchiveInventory is an immutable structural value for one exact acquired
// subject. Its constructor does not read bytes, verify integrity or authorize
// execution. Inspection produces the facts; introduction must recheck bytes.
// Its zero value is invalid.
type ArchiveInventory struct {
	artifact domain.AcquiredArtifact
	entries  []ArchiveEntry
	manifest []byte
}

// NewArchiveInventory validates bounded file facts and their manifest binding.
// Callers must independently measure the archive and every regular file digest.
func NewArchiveInventory(artifact domain.AcquiredArtifact, entries []ArchiveEntry, manifest []byte) (ArchiveInventory, error) {
	invalid := errors.New("npm archive inventory is invalid")
	parts := strings.Split(artifact.ContentHandle(), ":")
	if artifact.Identity().Source().String() != "npm" || artifact.Identity().Variant() != "tarball" || artifact.Digest().String() == "" ||
		artifact.SizeBytes() == 0 || artifact.SizeBytes() > ArchiveDecodedLimit || len(parts) != 3 || parts[0] != "intake" || parts[2] != "tarball" ||
		len(entries) == 0 || len(entries) > ArchiveEntryLimit || len(manifest) == 0 || len(manifest) > ArchiveManifestLimit {
		return ArchiveInventory{}, invalid
	}
	if _, err := domain.ParseRunID(parts[1]); err != nil {
		return ArchiveInventory{}, invalid
	}
	var identity struct{ Name, Version string }
	if json.Unmarshal(manifest, &identity) != nil || identity.Name != artifact.Identity().Name() || identity.Version != artifact.Identity().Version() {
		return ArchiveInventory{}, invalid
	}
	files := append([]ArchiveEntry(nil), entries...)
	slices.SortFunc(files, func(a, b ArchiveEntry) int { return strings.Compare(a.Path, b.Path) })
	seen := make(map[string]bool, len(files))
	var total int64
	manifestFound := false
	for _, item := range files {
		if !validInventoryPath(item.Path) || item.Mode < 0 || item.Mode > 0o777 || item.Size < 0 || item.Size > ArchiveFileLimit {
			return ArchiveInventory{}, invalid
		}
		if _, duplicate := seen[item.Path]; duplicate {
			return ArchiveInventory{}, invalid
		}
		for parent := path.Dir(item.Path); parent != "."; parent = path.Dir(parent) {
			if directory, present := seen[parent]; present && !directory {
				return ArchiveInventory{}, invalid
			}
		}
		seen[item.Path] = item.Directory
		if item.Directory {
			if item.Size != 0 || item.Digest.String() != "" {
				return ArchiveInventory{}, invalid
			}
			continue
		}
		if item.Digest.String() == "" || total > ArchiveDecodedLimit-item.Size {
			return ArchiveInventory{}, invalid
		}
		total += item.Size
		if item.Path == "package.json" {
			sum := sha256.Sum256(manifest)
			if item.Size != int64(len(manifest)) || item.Digest.String() != hex.EncodeToString(sum[:]) {
				return ArchiveInventory{}, invalid
			}
			manifestFound = true
		}
	}
	if !manifestFound {
		return ArchiveInventory{}, invalid
	}
	return ArchiveInventory{artifact, files, append([]byte(nil), manifest...)}, nil
}

func validInventoryPath(value string) bool {
	return value != "" && value != "." && len(value) <= 4096 && utf8.ValidString(value) && path.Clean(value) == value &&
		!strings.HasPrefix(value, "/") && value != ".." && !strings.HasPrefix(value, "../") &&
		!strings.ContainsAny(value, "\\:") && !strings.ContainsFunc(value, unicode.IsControl)
}

// Valid reports structural construction, never a policy decision.
func (i ArchiveInventory) Valid() bool { return len(i.entries) != 0 }

// Matches requires the full acquired subject, including its intake run handle.
func (i ArchiveInventory) Matches(artifact domain.AcquiredArtifact) bool {
	return i.Valid() && i.artifact == artifact
}

func (i ArchiveInventory) Artifact() domain.AcquiredArtifact { return i.artifact }
func (i ArchiveInventory) Entries() []ArchiveEntry           { return append([]ArchiveEntry(nil), i.entries...) }
func (i ArchiveInventory) Manifest() []byte                  { return append([]byte(nil), i.manifest...) }
