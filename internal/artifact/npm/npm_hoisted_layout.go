package npm

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errNPMHoistedLayout = errors.New("hoisted tree differs from complete archive, placement or bin facts")

type npmLayoutTarget struct {
	artifact domain.AcquiredArtifact
	instance string
}

type npmLayoutRequirement struct {
	kind       byte
	size, mode int64
	digest     domain.ContentDigest
	link       string
}

// Both native clients validate their own metadata and complete graph before
// sharing this comparison of measured archive members and hoisted tree bytes.
type npmHoistedPlacement struct {
	root      string
	inventory ArchiveInventory
	bins      map[string]string
}

func validateNPMHoistedLayout(ctx context.Context, placements []npmHoistedPlacement, layout PNPMLayoutArchive, uid, gid int) error {
	if ctx == nil || len(placements) == 0 {
		return errNPMHoistedLayout
	}
	wanted := map[string]npmLayoutRequirement{}
	add := func(name string, requirement npmLayoutRequirement) error {
		if !validInventoryPath(name) || name != "node_modules" && !strings.HasPrefix(name, "node_modules/") {
			return errNPMHoistedLayout
		}
		if previous, present := wanted[name]; present && previous != requirement {
			return errNPMHoistedLayout
		}
		wanted[name] = requirement
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if previous, present := wanted[parent]; present && previous.kind != tar.TypeDir {
				return errNPMHoistedLayout
			}
			wanted[parent] = npmLayoutRequirement{kind: tar.TypeDir}
		}
		if len(wanted) > pnpmLayoutEntryLimit {
			return errNPMHoistedLayout
		}
		return nil
	}
	roots := map[string]bool{}
	for _, placement := range placements {
		if err := ctx.Err(); err != nil {
			return errors.Join(errNPMHoistedLayout, err, context.Cause(ctx))
		}
		name, valid := packageNameFromLockPath(placement.root)
		if !valid || !placement.inventory.Valid() || name != placement.inventory.artifact.Identity().Name() || roots[placement.root] {
			return errNPMHoistedLayout
		}
		roots[placement.root] = true
		if err := add(placement.root, npmLayoutRequirement{kind: tar.TypeDir}); err != nil {
			return err
		}
		for _, member := range placement.inventory.entries {
			fact := npmLayoutRequirement{kind: tar.TypeReg, size: member.Size, mode: member.Mode &^ 0o022, digest: member.Digest}
			if member.Directory {
				fact = npmLayoutRequirement{kind: tar.TypeDir}
			}
			if err := add(placement.root+"/"+member.Path, fact); err != nil {
				return err
			}
		}
		modules := strings.TrimSuffix(placement.root, "/"+name)
		for command, file := range placement.bins {
			bin := modules + "/.bin/" + command
			link := pnpmBinRelativePath(path.Dir(bin), placement.root+"/"+file)
			if err := add(bin, npmLayoutRequirement{kind: tar.TypeSymlink, mode: 0o777, link: link}); err != nil {
				return err
			}
		}
	}
	entries := layout.Entries()
	if len(entries) != len(wanted) {
		return errNPMHoistedLayout
	}
	for _, actual := range entries {
		if err := ctx.Err(); err != nil {
			return errors.Join(errNPMHoistedLayout, err, context.Cause(ctx))
		}
		fact, present := wanted[actual.Name]
		if !present || actual.Kind != fact.kind || actual.UID != uid || actual.GID != gid {
			return errNPMHoistedLayout
		}
		switch fact.kind {
		case tar.TypeDir:
			if actual.Mode < 0 || actual.Mode > 0o777 || actual.Mode&0o022 != 0 || actual.Mode&0o700 != 0o700 {
				return errNPMHoistedLayout
			}
		case tar.TypeSymlink:
			if actual.Mode != 0o777 || actual.Link != fact.link {
				return errNPMHoistedLayout
			}
		case tar.TypeReg:
			if actual.Mode != fact.mode || actual.Mode&0o022 != 0 || actual.Mode&0o600 != 0o600 || actual.Size != fact.size || actual.Digest != fact.digest {
				return errNPMHoistedLayout
			}
		default:
			return errNPMHoistedLayout
		}
	}
	return nil
}

// Keep the original Bun encoding/schema unchanged while sharing canonical
// digest rendering with the actual npm project consumer. Domain digests are
// serialized as strings, never as opaque structs with private fields.
func npmInspectionLayoutDigest(schema string, graph domain.ContentDigest, runtime string, artifacts []domain.AcquiredArtifact, entries []PNPMLayoutEntry, manifest []byte) (domain.ContentDigest, error) {
	type subject struct {
		Source, Name, Version, Variant, Digest, Declared, Observed, Handle string
		Size                                                               uint64
	}
	var subjects []subject
	for _, artifact := range artifacts {
		declared, _ := artifact.DeclaredIntegrity()
		observed, _ := artifact.ObservedIntegrity()
		identity := artifact.Identity()
		subjects = append(subjects, subject{identity.Source().String(), identity.Name(), identity.Version(), identity.Variant(), artifact.Digest().String(), declared, observed, artifact.ContentHandle(), artifact.SizeBytes()})
	}
	slices.SortFunc(subjects, func(a, b subject) int { return strings.Compare(a.Name+"@"+a.Version, b.Name+"@"+b.Version) })
	type entry struct {
		Name, Link, Digest string
		Kind               byte
		Mode, Size         int64
		UID, GID           int
	}
	slices.SortFunc(entries, func(a, b PNPMLayoutEntry) int { return strings.Compare(a.Name, b.Name) })
	records := make([]entry, 0, len(entries))
	for _, e := range entries {
		records = append(records, entry{e.Name, e.Link, e.Digest.String(), e.Kind, e.Mode, e.Size, e.UID, e.GID})
	}
	document := struct {
		Schema, Graph, Runtime string
		Subjects               []subject
		Entries                []entry
		Manifest               json.RawMessage
	}{schema, graph.String(), runtime, subjects, records, manifest}
	body, err := json.Marshal(document)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	sum := sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
}
