package npm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func pnpmInventoryBins(inventory ArchiveInventory) (map[string]string, error) {
	if !inventory.Valid() || strictControlJSON(inventory.manifest) != nil {
		return nil, errPNPMInspectionLayout
	}
	var manifest map[string]json.RawMessage
	if json.Unmarshal(inventory.manifest, &manifest) != nil {
		return nil, errPNPMInspectionLayout
	}
	for _, name := range []string{"bundleDependencies", "bundledDependencies"} {
		if body, present := manifest[name]; present && !bytes.Equal(body, []byte("false")) && !bytes.Equal(body, []byte("null")) {
			var names []string
			if json.Unmarshal(body, &names) != nil || len(names) != 0 {
				return nil, errPNPMInspectionLayout
			}
		}
	}
	if body, present := manifest["directories"]; present && !bytes.Equal(body, []byte("null")) {
		var directories map[string]json.RawMessage
		if json.Unmarshal(body, &directories) != nil {
			return nil, errPNPMInspectionLayout
		}
		if _, present := directories["bin"]; present {
			return nil, errPNPMInspectionLayout
		}
	}
	files := map[string]bool{}
	for _, file := range inventory.entries {
		if file.Path == "node_modules" || strings.HasPrefix(file.Path, "node_modules/") {
			return nil, errPNPMInspectionLayout
		}
		files[file.Path] = !file.Directory
	}
	bins := map[string]string{}
	if body, present := manifest["bin"]; present && !bytes.Equal(body, []byte("null")) {
		var file string
		if json.Unmarshal(body, &file) == nil {
			bins[path.Base(inventory.artifact.Identity().Name())] = file
		} else if json.Unmarshal(body, &bins) != nil || bins == nil {
			return nil, errPNPMInspectionLayout
		}
	}
	if len(bins) > ArchiveEntryLimit {
		return nil, errPNPMInspectionLayout
	}
	for command, file := range bins {
		file = strings.TrimPrefix(file, "./")
		if len(command) > 214 || command == "." || command == ".." || strings.Contains(command, "/") ||
			!safePNPMBinCharacters(command) || !validInventoryPath(file) || !safePNPMBinCharacters(file) || !files[file] {
			return nil, errPNPMInspectionLayout
		}
		bins[command] = file
	}
	return bins, nil
}

// Root scripts, config and personal metadata are not introduced to an Artifact
// session. Its manifest contains only exact frozen root dependency data.
func pnpmInspectionManifest(lock PNPMLock) ([]byte, error) {
	packages := map[string]PNPMLockedPackage{}
	instances := map[string]string{}
	for _, pkg := range lock.packages {
		packages[pkg.Key] = pkg
	}
	for _, instance := range lock.snapshots {
		instances[instance.Key] = instance.Package
	}
	manifest := map[string]any{"private": true}
	groups := map[string]map[string]string{}
	for _, root := range lock.roots {
		pkg, present := packages[instances[root.Snapshot]]
		if !present || pkg.Name != root.Name || root.Group != "dependencies" && root.Group != "devDependencies" {
			return nil, errPNPMInspectionLayout
		}
		if groups[root.Group] == nil {
			groups[root.Group] = map[string]string{}
		}
		if previous, exists := groups[root.Group][root.Name]; exists && previous != pkg.Version {
			return nil, errPNPMInspectionLayout
		}
		groups[root.Group][root.Name] = pkg.Version
	}
	for group, dependencies := range groups {
		manifest[group] = dependencies
	}
	return json.Marshal(manifest)
}

func pnpmInspectionLayoutDigest(input PNPMInspectionInput, layout PNPMLayoutArchive, modules pnpmLayoutModules, manifest []byte) (domain.ContentDigest, error) {
	type subject struct {
		Source, Name, Version, Variant, Digest, Declared, Observed string
		Size                                                       uint64
	}
	var subjects []subject
	for _, artifact := range input.artifacts {
		declared, _ := artifact.DeclaredIntegrity()
		observed, _ := artifact.ObservedIntegrity()
		subjects = append(subjects, subject{artifact.Identity().Source().String(), artifact.Identity().Name(), artifact.Identity().Version(), artifact.Identity().Variant(), artifact.Digest().String(), declared, observed, artifact.SizeBytes()})
	}
	slices.SortFunc(subjects, func(a, b subject) int { return strings.Compare(a.Name+"@"+a.Version, b.Name+"@"+b.Version) })
	modules.PrunedAt = ""
	moduleBytes, err := json.Marshal(modules)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	lockBytes, err := json.Marshal(pnpmSnapshotLockState(input.context.selection.lock))
	if err != nil {
		return domain.ContentDigest{}, err
	}
	entries := layout.Entries()
	for index := range entries {
		var canonical []byte
		switch entries[index].Name {
		case "node_modules/.modules.yaml":
			canonical = moduleBytes
		case "node_modules/.pnpm/lock.yaml":
			canonical = lockBytes
		default:
			continue
		}
		digest := sha256.Sum256(canonical)
		entries[index].Size = int64(len(canonical))
		entries[index].Digest, err = domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
		if err != nil {
			return domain.ContentDigest{}, err
		}
	}
	slices.SortFunc(entries, func(a, b PNPMLayoutEntry) int { return strings.Compare(a.Name, b.Name) })
	// Domain digests deliberately have private fields. Project explicit string
	// values here instead of silently marshaling each digest as an empty object.
	type entryRecord struct {
		Name, Link, Digest string
		Kind               byte
		Mode, Size         int64
		UID, GID           int
	}
	records := make([]entryRecord, 0, len(entries))
	for _, entry := range entries {
		records = append(records, entryRecord{entry.Name, entry.Link, entry.Digest.String(), entry.Kind, entry.Mode, entry.Size, entry.UID, entry.GID})
	}
	document := struct {
		Schema, Graph, Runtime string
		Subjects               []subject
		Entries                []entryRecord
		Manifest               json.RawMessage
	}{"pnpm-inspection-layout/v2", input.context.snapshot.GraphDigest().String(), input.context.runtime, subjects, records, manifest}
	body, err := json.Marshal(document)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	digest := sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
}
