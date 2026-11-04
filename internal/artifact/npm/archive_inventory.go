package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"regexp"
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

// DeclaredSourcePaths collects local source candidates from measured package
// metadata. Conditions and wildcard replacements form a conservative superset,
// not a Node resolution or execution authorization.
func (i ArchiveInventory) DeclaredSourcePaths(metadata map[string][]byte) (map[string]bool, error) {
	invalid := errors.New("npm source declarations are invalid or unsupported")
	if !i.Valid() {
		return nil, invalid
	}
	files := make(map[string]bool, len(i.entries))
	for _, member := range i.entries {
		files[member.Path] = !member.Directory
	}
	selected, binDirectories := map[string]bool{}, map[string]bool{}
	patterns := map[string]bool{}
	expressionBytes, metadataCount := 0, 0
	for _, member := range i.entries {
		if member.Directory || path.Base(member.Path) != "package.json" {
			continue
		}
		metadataCount++
		body, present := metadata[member.Path]
		digest := sha256.Sum256(body)
		if !present || int64(len(body)) != member.Size || hex.EncodeToString(digest[:]) != member.Digest.String() || strictControlJSON(body) != nil {
			return nil, invalid
		}
		var manifest map[string]json.RawMessage
		if json.Unmarshal(body, &manifest) != nil {
			return nil, invalid
		}
		base := path.Dir(member.Path)
		resolve := func(target string) (string, error) {
			if path.IsAbs(target) || strings.ContainsAny(target, "\\\x00\r\n") {
				return "", invalid
			}
			target = path.Join(base, target)
			if target != "." && !validInventoryPath(target) {
				return "", invalid
			}
			return target, nil
		}
		add := func(target string, bin, wildcard bool) error {
			target, err := resolve(target)
			if err != nil {
				return err
			}
			if wildcard && strings.Contains(target, "*") {
				expression := strings.ReplaceAll(regexp.QuoteMeta(target), `\*`, ".*")
				if !patterns[expression] {
					expressionBytes += len(expression)
					if len(patterns) >= ArchiveEntryLimit || expressionBytes > ProjectControlLimit {
						return invalid
					}
					patterns[expression] = true
				}
			} else {
				if bin && !files[target] {
					return invalid
				}
				if files[target] && (bin || path.Ext(target) != ".json") {
					selected[target] = true
				}
			}
			return nil
		}
		for _, field := range []string{"main", "module", "jsnext:main"} {
			raw, present := manifest[field]
			if !present || string(raw) == "null" {
				continue
			}
			var target string
			if json.Unmarshal(raw, &target) != nil {
				return nil, invalid
			}
			if target != "" {
				if err := add(target, false, false); err != nil {
					return nil, err
				}
			}
		}
		raw, hasBin := manifest["bin"]
		hasBin = hasBin && string(raw) != "null"
		if hasBin {
			bins, err := bunPackageBins(raw, i.artifact.Identity().Name())
			if err != nil {
				return nil, invalid
			}
			for _, target := range bins {
				if err := add(target, true, false); err != nil {
					return nil, err
				}
			}
		} else if raw, present := manifest["directories"]; present && string(raw) != "null" {
			var directories map[string]json.RawMessage
			if json.Unmarshal(raw, &directories) != nil {
				return nil, invalid
			}
			if raw, present := directories["bin"]; present {
				var target string
				if json.Unmarshal(raw, &target) != nil || target == "" {
					return nil, invalid
				}
				target, err := resolve(target)
				if err != nil {
					return nil, err
				}
				binDirectories[target] = true
			}
		}
		for _, field := range []string{"exports", "imports", "browser"} {
			raw, present := manifest[field]
			if !present {
				continue
			}
			var value any
			if json.Unmarshal(raw, &value) != nil {
				return nil, invalid
			}
			var visit func(any) error
			visit = func(value any) error {
				switch value := value.(type) {
				case nil:
					return nil
				case string:
					if field != "browser" {
						if !strings.HasPrefix(value, "./") {
							if field == "imports" {
								return nil
							} // External package/builtin, not an archive member.
							return invalid
						}
						parsed, err := url.Parse(value)
						if err != nil || parsed.IsAbs() || parsed.Host != "" {
							return invalid
						}
						value = parsed.Path
					}
					return add(value, false, field != "browser")
				case []any:
					for _, child := range value {
						if err := visit(child); err != nil {
							return err
						}
					}
				case map[string]any:
					for _, child := range value {
						if err := visit(child); err != nil {
							return err
						}
					}
				case bool:
					if field != "browser" || value {
						return invalid
					}
				default:
					return invalid
				}
				return nil
			}
			if err := visit(value); err != nil {
				return nil, err
			}
		}
	}
	if metadataCount != len(metadata) {
		return nil, invalid
	}
	var matcher *regexp.Regexp
	if len(patterns) != 0 {
		expressions := make([]string, 0, len(patterns))
		for expression := range patterns {
			expressions = append(expressions, expression)
		}
		slices.Sort(expressions)
		var err error
		matcher, err = regexp.Compile("^(?:" + strings.Join(expressions, "|") + ")$")
		if err != nil {
			return nil, invalid
		}
	}
	for _, member := range i.entries {
		if member.Directory {
			continue
		}
		bin := binDirectories["."]
		for offset := 0; !bin && offset < len(member.Path); {
			slash := strings.IndexByte(member.Path[offset:], '/')
			if slash < 0 {
				break
			}
			offset += slash
			bin = binDirectories[member.Path[:offset]]
			offset++
		}
		if bin || matcher != nil && path.Ext(member.Path) != ".json" && matcher.MatchString(member.Path) {
			selected[member.Path] = true
		}
	}
	return selected, nil
}
