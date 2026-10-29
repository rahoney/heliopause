package pypi

import (
	"cmp"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"slices"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errConsoleExecutionEntry = errors.New("console execution entry requires a measured public wheel and exact supported console declaration")

// ConsoleExecutionInventory is canonical reader DATA, not verification,
// Policy approval, an interpreter proof or runtime admission authority.
// Its fields cannot be reconstructed from externally supplied inspection JSON.
type ConsoleExecutionInventory struct {
	artifact    domain.AcquiredArtifact
	metadata    WheelFile
	entries     []EntryPoint
	fingerprint [sha256.Size]byte
}

// InspectConsoleExecutionWheel reuses the complete existing ZIP, digest,
// RECORD, metadata and target validation. It does not extract or execute code.
func InspectConsoleExecutionWheel(artifact domain.AcquiredArtifact, reader io.ReaderAt, filename string, target WheelTarget, limits WheelLimits) (ConsoleExecutionInventory, error) {
	id := artifact.Identity()
	declared, ok := artifact.DeclaredIntegrity()
	if id.Source() != PublicPyPIProfile().Source() || id.Variant() != "wheel" || artifact.Digest().String() == "" || !ok || declared != "sha256:"+artifact.Digest().String() || limits.MaxCompressed <= 0 || artifact.SizeBytes() == 0 || artifact.SizeBytes() > uint64(limits.MaxCompressed) {
		return ConsoleExecutionInventory{}, errConsoleExecutionEntry
	}
	info, err := InspectWheelForSource(reader, int64(artifact.SizeBytes()), filename, artifact.Digest().String(), target, limits, id.Source())
	if err != nil {
		return ConsoleExecutionInventory{}, err
	}
	if info.Project != id.Name() || info.Version != id.Version() || info.ObservedSHA256 != artifact.Digest().String() || info.DeclaredSHA256 != artifact.Digest().String() {
		return ConsoleExecutionInventory{}, errConsoleExecutionEntry
	}
	metadataPath := wheelDistInfo(info.Project, info.Version) + "/entry_points.txt"
	var metadata WheelFile
	matches := 0
	for _, file := range info.Files {
		if file.Path == metadataPath {
			metadata = file
			matches++
		}
	}
	if matches != 1 || metadata.Size <= 0 || !validSHA256(metadata.SHA256) {
		return ConsoleExecutionInventory{}, errConsoleExecutionEntry
	}
	// Details originate only from entry_points.txt in the existing reader.
	// Legacy METADATA header strings are not executable console declarations.
	entries := make([]EntryPoint, len(info.Surface.EntryPointDetails))
	for i, entry := range info.Surface.EntryPointDetails {
		entry.Extras = append([]string(nil), entry.Extras...)
		entries[i] = entry
	}
	// RECORD uses a map internally. Preserve all measured facts, but do not
	// make its iteration order part of this consumer's whole-source identity.
	slices.SortFunc(info.Files, func(a, b WheelFile) int { return cmp.Compare(a.Path, b.Path) })
	slices.SortFunc(info.Surface.InstalledFiles, func(a, b InstalledFile) int {
		return cmp.Compare(a.ArchivePath, b.ArchivePath)
	})
	slices.SortFunc(info.Surface.SiteHookLines, func(a, b SiteHookLine) int {
		if order := cmp.Compare(a.File, b.File); order != 0 {
			return order
		}
		return cmp.Compare(a.Line, b.Line)
	})
	body, err := json.Marshal(info)
	if err != nil {
		return ConsoleExecutionInventory{}, errConsoleExecutionEntry
	}
	return ConsoleExecutionInventory{artifact: artifact, metadata: metadata, entries: entries, fingerprint: sha256.Sum256(body)}, nil
}

func (i ConsoleExecutionInventory) Valid() bool {
	return i.artifact.Digest().String() != "" && i.metadata.Path != "" && i.metadata.Size > 0 && i.fingerprint != [sha256.Size]byte{}
}
func (i ConsoleExecutionInventory) Artifact() domain.AcquiredArtifact { return i.artifact }

// ConsoleExecutionEntry binds a console declaration and its RECORD metadata
// member to an exact original acquisition and complete static snapshot.
// Selecting it neither imports its module nor invokes its callable.
type ConsoleExecutionEntry struct {
	inventory ConsoleExecutionInventory
	entry     EntryPoint
}

func SelectConsoleExecutionEntry(inventory ConsoleExecutionInventory, command string) (ConsoleExecutionEntry, error) {
	if !inventory.Valid() {
		return ConsoleExecutionEntry{}, errConsoleExecutionEntry
	}
	var candidates []EntryPoint
	for _, entry := range inventory.entries {
		if entry.Group == "console_scripts" {
			candidates = append(candidates, entry)
		}
	}
	if command == "" {
		if len(candidates) != 1 || candidates[0].Name != inventory.Artifact().Identity().Name() {
			return ConsoleExecutionEntry{}, errConsoleExecutionEntry
		}
		command = candidates[0].Name
	}
	if !validToolCommand(command) || domain.ValidateExecutionCommand(command) != nil {
		return ConsoleExecutionEntry{}, errConsoleExecutionEntry
	}
	var chosen EntryPoint
	matches := 0
	for _, entry := range candidates {
		if entry.Name == command {
			chosen = entry
			matches++
		}
	}
	if matches != 1 || chosen.Attr == "" || len(chosen.Extras) != 0 {
		return ConsoleExecutionEntry{}, errConsoleExecutionEntry
	}
	return ConsoleExecutionEntry{inventory: inventory, entry: chosen}, nil
}

func (e ConsoleExecutionEntry) Valid() bool {
	return e.inventory.Valid() && e.entry.Group == "console_scripts" && e.entry.Name != "" && e.entry.Attr != ""
}
func (e ConsoleExecutionEntry) Artifact() domain.AcquiredArtifact { return e.inventory.Artifact() }
func (e ConsoleExecutionEntry) Command() string                   { return e.entry.Name }
func (e ConsoleExecutionEntry) Metadata() WheelFile               { return e.inventory.metadata }
func (e ConsoleExecutionEntry) EntryPoint() EntryPoint {
	entry := e.entry
	entry.Extras = append([]string(nil), entry.Extras...)
	return entry
}

func (e ConsoleExecutionEntry) Matches(inventory ConsoleExecutionInventory) bool {
	if !e.Valid() || !inventory.Valid() || inventory.Artifact() != e.Artifact() || inventory.fingerprint != e.inventory.fingerprint || inventory.metadata != e.inventory.metadata {
		return false
	}
	selected, err := SelectConsoleExecutionEntry(inventory, e.Command())
	return err == nil && selected.entry.key() == e.entry.key()
}
